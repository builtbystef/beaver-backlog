// Package store is the file store layer: it owns the .beaver/ layout, finds the
// store from a working directory, and reads, writes, lists, and resolves issue
// files. The files are the sole source of truth (ADR 0003); this package is the
// only one that touches them on disk.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"beaver/internal/issue"
)

// dirName is the store directory created at a project's root.
const dirName = ".beaver"

// FormatVersion marks the on-disk layout the store was created with, recorded in
// the committed project config so future versions can detect and migrate it.
const FormatVersion = 1

// Sentinel errors callers branch on.
var (
	// ErrNoStore means no .beaver directory was found at or above a working dir.
	ErrNoStore = errors.New("not a Busy Beaver store; run `beaver init`")
	// ErrNotFound means a reference matched no issue in the store.
	ErrNotFound = errors.New("issue not found")
)

// SharedSlugError reports that a reference is a slug several issues share, so it
// names no single issue. It carries the matching issues (sorted by their unique
// ID) so a caller can list them; the remedy is the full ID. It Unwraps to
// ErrNotFound — a shared slug resolves to no single issue — so callers that branch
// on ErrNotFound treat it as a not-found, while one that wants the candidates
// reaches them with errors.As.
type SharedSlugError struct {
	Slug    string
	Matches []issue.Issue
}

func (e *SharedSlugError) Error() string {
	ids := make([]string, len(e.Matches))
	for i, m := range e.Matches {
		ids[i] = m.ID
	}
	return fmt.Sprintf("slug %q is shared by %d issues (%s)", e.Slug, len(e.Matches), strings.Join(ids, ", "))
}

func (e *SharedSlugError) Unwrap() error { return ErrNotFound }

// Warning reports a file the store skipped during a scan because it is not a
// valid issue (ADR 0005): Path names the offending file and Err says what is
// wrong with it (unreadable, malformed frontmatter, or a failed validation).
// The store never fails a whole read on one bad file — it skips it and reports it
// here — so a command can surface it loudly while still serving the valid issues.
// doctor turns the same signal into a full health report (n9b4a7).
type Warning struct {
	Path string // path to the skipped file
	Err  error  // the specific reason it is not a usable issue
}

// Store is a handle to an initialized .beaver directory.
type Store struct {
	root   string        // absolute path to the .beaver directory
	onWarn func(Warning) // optional sink for files a scan skips (ADR 0005); nil discards
}

// Root returns the absolute path to the .beaver directory.
func (s *Store) Root() string { return s.root }

// OnWarn registers fn to receive every file the store skips as invalid during a
// scan (ADR 0005). It is an optional diagnostic channel: with no handler set (the
// default) skipped files are dropped silently, as the store's readers always
// have; the CLI installs a handler that prints each one loudly to stderr, so a
// broken store is never mistaken for a clean one. fn may be called more than once
// for the same path when a single command scans the store repeatedly (create's
// id-collision loop), so a handler that renders warnings dedupes by path.
func (s *Store) OnWarn(fn func(Warning)) { s.onWarn = fn }

// warn reports a skipped file to the registered handler, if any.
func (s *Store) warn(w Warning) {
	if s.onWarn != nil {
		s.onWarn(w)
	}
}

// IssuesDir returns the directory holding issue files.
func (s *Store) IssuesDir() string { return filepath.Join(s.root, "issues") }

// ConfigPath returns the path to the committed project config.
func (s *Store) ConfigPath() string { return filepath.Join(s.root, "config.yml") }

// Init creates (or safely re-creates) a store under workDir: the .beaver/issues
// directory and a committed project config carrying the format version. It is
// idempotent — re-running never clobbers an existing config. created reports
// whether the store directory was newly made. Init never touches a VCS; Busy Beaver
// requires none (ADR 0006, ADR 0008).
func Init(workDir string) (root string, created bool, err error) {
	root = filepath.Join(workDir, dirName)
	created = !dirExists(root)

	if err := os.MkdirAll(filepath.Join(root, "issues"), 0o755); err != nil {
		return "", false, err
	}
	config := filepath.Join(root, "config.yml")
	if !fileExists(config) {
		if err := os.WriteFile(config, defaultConfig(), 0o644); err != nil {
			return "", false, err
		}
	}
	return root, created, nil
}

// Discover walks up from workDir looking for a .beaver directory
// and returns ErrNoStore if none is found.
func Discover(workDir string) (*Store, error) {
	dir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	for {
		candidate := filepath.Join(dir, dirName)
		if dirExists(candidate) {
			return &Store{root: candidate}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached the filesystem root
			return nil, ErrNoStore
		}
		dir = parent
	}
}

// List returns the paths of all issue files, sorted for deterministic ordering.
// A missing issues directory is treated as an empty store rather than an error: a
// half-merged or hand-edited store is a normal, recoverable state (ADR 0005), and
// `doctor` is what reports and repairs it. Other read errors still surface.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.IssuesDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		files = append(files, filepath.Join(s.IssuesDir(), e.Name()))
	}
	sort.Strings(files)
	return files, nil
}

// ReadAll reads and validates every issue in the store, returning them in the
// stable path order List provides. Invalid files are skipped rather than failing
// the whole read (ADR 0005), mirroring Resolve, and reported through the store's
// warning handler so a caller can surface them loudly. A missing issues directory
// yields no issues. Callers that need a specific display order re-sort the result.
func (s *Store) ReadAll() ([]issue.Issue, error) {
	items, err := s.scan()
	if err != nil {
		return nil, err
	}
	issues := make([]issue.Issue, len(items))
	for i, it := range items {
		issues[i] = it.iss
	}
	return issues, nil
}

// IDTaken reports whether an issue with the given ID already exists, so create
// can regenerate on the rare collision. It consults the authoritative frontmatter
// ID — the same identity Resolve matches on — so the two can never disagree.
func (s *Store) IDTaken(id string) (bool, error) {
	items, err := s.scan()
	if err != nil {
		return false, err
	}
	for _, it := range items {
		if it.iss.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// Write serializes an issue to its canonical file (<id>-<slug>.md), replacing any
// existing file for that name atomically.
func (s *Store) Write(iss issue.Issue) (string, error) {
	data, err := issue.Marshal(iss)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.IssuesDir(), issue.FileName(iss.ID, issue.Slug(iss.Title)))
	if err := atomicWrite(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Update writes back an edited issue that was located at oldPath (as returned by
// Resolve), and returns the file's canonical path. It is the write half of a
// read-modify-write: unlike Write, which only knows the canonical name, Update
// also removes the old file when its name has drifted from the canonical
// <id>-<slug> — a hand-edit or merge may leave a stale name (ADR 0005). This
// keeps filenames correct on every write and, crucially, never leaves a second
// file with the same id behind, which a bare Write to the canonical name would.
// When oldPath is already canonical (the common case) it is simply overwritten in
// place.
func (s *Store) Update(oldPath string, iss issue.Issue) (string, error) {
	newPath, err := s.Write(iss)
	if err != nil {
		return "", err
	}
	if oldPath != "" && oldPath != newPath {
		if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return newPath, nil
}

// Resolve turns a user-supplied reference into a single issue, matched on the
// authoritative frontmatter identity (ADR 0002) and never on the filename, which
// only mirrors it and may have drifted via a hand-edit or merge (ADR 0005). It is
// the one resolver every issue-addressing command routes through, so they all
// accept the same references. Matching is exact — there is no prefix or fuzzy
// matching — and a reference may take any of three forms:
//
//  1. a full ID — the authoritative, unique identity;
//  2. the full "<id>-<slug>" name — the canonical file name (minus .md), also
//     unique, so a user can paste what they see on disk;
//  3. a full slug — the title's canonical slug (not the possibly-stale filename
//     slug), a human-friendly handle.
//
// The first two forms carry the unique ID, so they never collide. A slug is
// derived from the mutable title and so is not unique; only a slug that names a
// single issue resolves. A slug several issues share names no single issue and
// yields a *SharedSlugError (which Unwraps to ErrNotFound but carries the
// candidates); an unknown reference yields ErrNotFound. There is no separate
// "ambiguous" outcome — a shared slug simply does not resolve, and the caller
// falls back to the ID. Invalid files carry no identity Resolve can match and are
// skipped (ADR 0005), each reported through the store's warning handler.
func (s *Store) Resolve(ref string) (issue.Issue, string, error) {
	items, err := s.scan()
	if err != nil {
		return issue.Issue{}, "", err
	}

	// The two unique forms first — the ID itself and the canonical "<id>-<slug>"
	// name. Both carry the unique ID, so a match is decisive.
	for _, it := range items {
		slug := issue.Slug(it.iss.Title)
		if ref == it.iss.ID || (slug != "" && ref == it.iss.ID+"-"+slug) {
			return it.iss, it.path, nil
		}
	}
	// Then the slug alone. It is not unique: a single match resolves, several are a
	// SharedSlugError naming the candidates, and none (or an empty ref) is a plain
	// not-found.
	switch m := matchSlug(items, ref); len(m) {
	case 1:
		return m[0].iss, m[0].path, nil
	case 0:
		return issue.Issue{}, "", ErrNotFound
	default:
		return issue.Issue{}, "", sharedSlug(ref, m)
	}
}

// resolvable pairs a parsed issue with the path it was read from — the two things
// Resolve returns together.
type resolvable struct {
	iss  issue.Issue
	path string
}

// scan reads and validates every issue in the store, pairing each with its path
// and skipping any file that is not a usable issue (ADR 0005) — the same tolerant
// read ReadAll performs. Each skipped file is reported through the store's warning
// handler, naming it and the specific problem, so a command can surface it loudly
// while still operating on the valid issues; with no handler the skip is silent.
// scan backs Resolve, ReadAll, and IDTaken, so they share one notion of which
// issues exist and one warning for each that does not.
func (s *Store) scan() ([]resolvable, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}
	items := make([]resolvable, 0, len(files))
	for _, f := range files {
		iss, err := readIssue(f)
		if err != nil {
			s.warn(Warning{Path: f, Err: err}) // skip it loudly (ADR 0005)
			continue
		}
		items = append(items, resolvable{iss: iss, path: f})
	}
	return items, nil
}

// matchSlug returns the issues whose canonical (title-derived) slug equals ref. An
// empty slug — a title with no alphanumerics — is not a usable reference and never
// matches, so an empty ref matches nothing here.
func matchSlug(items []resolvable, ref string) []resolvable {
	var out []resolvable
	for _, it := range items {
		if slug := issue.Slug(it.iss.Title); slug != "" && slug == ref {
			out = append(out, it)
		}
	}
	return out
}

// sharedSlug builds a *SharedSlugError from the matches, sorted by their unique ID
// so the candidate listing is deterministic regardless of file iteration order.
func sharedSlug(slug string, matches []resolvable) error {
	issues := make([]issue.Issue, len(matches))
	for i, m := range matches {
		issues[i] = m.iss
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].ID < issues[j].ID })
	return &SharedSlugError{Slug: slug, Matches: issues}
}

// readIssue reads, parses, and validates one issue file, returning the reason it
// is not a usable issue — an unreadable file, malformed frontmatter (Unmarshal),
// or a failed validation (missing/malformed id, illegal state) — with no filename
// prefix, since the caller pairs the reason with the path. A nil error means a
// valid issue. It draws the validation line of ADR 0005: only these hard failures
// make a file unusable; a valid issue's untidiness (filename drift, unknown keys)
// is lint, left intact for doctor (n9b4a7).
func readIssue(path string) (issue.Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return issue.Issue{}, err
	}
	iss, err := issue.Unmarshal(data)
	if err != nil {
		return issue.Issue{}, err
	}
	if err := issue.Validate(iss); err != nil {
		return issue.Issue{}, err
	}
	return iss, nil
}

func defaultConfig() []byte {
	return fmt.Appendf(nil,
		`# Busy Beaver project configuration.
# Committed and shared through version control, like the issues themselves.
# Safe to read and edit by hand.
format_version: %d
`, FormatVersion)
}

// atomicWrite writes data to path via a temp file and rename, so a reader never
// observes a half-written issue.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".beaver-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
