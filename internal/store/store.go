// Package store owns the .beaver/ layout: it finds the store from a working
// directory and reads, writes, lists, and resolves issue files. The files are
// the sole source of truth; this package is the only one that touches them on disk.
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

// FormatVersion marks the on-disk layout the store was created with, recorded
// in the committed project config.
const FormatVersion = 1

// Sentinel errors callers branch on.
var (
	// ErrNoStore means no .beaver directory was found at or above a working dir.
	ErrNoStore = errors.New("not a Busy Beaver store; run `beaver init`")
	// ErrNotFound means a reference matched no issue in the store.
	ErrNotFound = errors.New("issue not found")
	// ErrNameCollision means a rename would land on a name another file already
	// holds; Rename refuses rather than overwrite it.
	ErrNameCollision = errors.New("the canonical name is already held by another file")
)

// SharedSlugError reports that a reference is a slug several issues share, so it
// names no single issue. It carries the matches (sorted by ID) and Unwraps to
// ErrNotFound, so callers can treat it as a not-found or reach the candidates
// with errors.As.
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

// Warning reports a file a scan skipped because it is not a valid issue:
// Path names the file and Err says what is wrong with it. The store never
// fails a whole read on one bad file — it skips it and reports it here.
type Warning struct {
	Path string // path to the skipped file
	Err  error  // the specific reason it is not a usable issue
}

// Store is a handle to an initialized .beaver directory.
type Store struct {
	root   string        // absolute path to the .beaver directory
	onWarn func(Warning) // optional sink for files a scan skips; nil discards
}

// Root returns the absolute path to the .beaver directory.
func (s *Store) Root() string { return s.root }

// OnWarn registers fn to receive every file the store skips as invalid during
// a scan; with no handler, skips are silent. fn may be called more than once
// for the same path when a command scans repeatedly, so a handler that renders
// warnings should dedupe by path.
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

// Init creates a store under workDir: the .beaver/issues directory and a
// project config carrying the format version. It is idempotent — re-running
// never clobbers an existing config. created reports whether the store
// directory was newly made.
func Init(workDir string) (root string, created bool, err error) {
	root = filepath.Join(workDir, dirName)
	// Mkdir both creates and reports prior existence, so created never races a
	// separate existence check.
	switch err := os.Mkdir(root, 0o755); {
	case err == nil:
		created = true
	case errors.Is(err, os.ErrExist):
		created = false
	default:
		return "", false, err
	}

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

// Discover walks up from workDir looking for a .beaver directory and returns
// ErrNoStore if none is found.
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
// A missing issues directory is treated as an empty store, not an error — a
// half-merged or hand-edited store is a normal, recoverable state. Other read
// errors still surface.
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

// ReadAll reads and validates every issue in the store, in stable path order.
// Invalid files are skipped rather than failing the whole read, and reported
// through the store's warning handler. A missing issues directory yields no issues.
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

// IDTaken reports whether an issue with the given ID already exists, matching
// on the authoritative frontmatter ID — the same identity Resolve matches on.
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

// Update writes back an edited issue located at oldPath (as returned by
// Resolve) and returns the canonical path. Unlike Write, it also removes the
// old file when its name has drifted from the canonical <id>-<slug>, so a
// read-modify-write never leaves a second file with the same id. An
// already-canonical oldPath is overwritten in place.
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

// Read parses and validates the single issue file at path, applying the same
// usable-issue contract a store scan applies: an unreadable file, malformed
// frontmatter, or a failed validation is an error.
func (s *Store) Read(path string) (issue.Issue, error) {
	return readIssue(path)
}

// Rename moves the issue file at oldPath to its canonical <id>-<slug> name
// without rewriting its contents. When the canonical name is already held by a
// different file (the sign of a duplicate id) it returns ErrNameCollision
// rather than overwrite it. A file already at its canonical name is a no-op.
func (s *Store) Rename(oldPath string, iss issue.Issue) (string, error) {
	newPath := filepath.Join(s.IssuesDir(), issue.FileName(iss.ID, issue.Slug(iss.Title)))
	if newPath == oldPath {
		return newPath, nil
	}
	if fileExists(newPath) {
		return "", ErrNameCollision
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", err
	}
	return newPath, nil
}

// StashDraft moves the file at path into .beaver/drafts, preserving its base
// name, and returns the destination. Stashing keeps a human's typed-but-invalid
// authoring out of the scanned issue set without discarding their words.
func (s *Store) StashDraft(path string) (string, error) {
	dir := filepath.Join(s.root, "drafts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, filepath.Base(path))
	if err := os.Rename(path, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// Delete removes the issue file at path. The store keeps no other copy; a
// missing file surfaces as the underlying os error.
func (s *Store) Delete(path string) error {
	return os.Remove(path)
}

// Resolve turns a user-supplied reference into a single issue, matched on the
// authoritative frontmatter identity, never on the filename (which may have
// drifted). Matching is exact, in four forms tried in order: a full ID, the
// canonical "<id>-<slug>" name, the title's canonical slug, and — strictly last,
// so it never shadows the others — the id part of a stale "<id>-<oldslug>" file
// name (".md" suffix tolerated).
//
// A slug shared by several issues names no single issue and yields a
// *SharedSlugError (Unwraps to ErrNotFound, carries the candidates); an unknown
// reference yields ErrNotFound. Invalid files carry no identity to match and
// are skipped, each reported through the store's warning handler.
func (s *Store) Resolve(ref string) (issue.Issue, string, error) {
	items, err := s.scan()
	if err != nil {
		return issue.Issue{}, "", err
	}
	return resolveIn(items, ref)
}

// resolveIn is Resolve's matching logic over an already-scanned set, shared with
// Snapshot so a command that resolves several references pays for one scan.
func resolveIn(items []resolvable, ref string) (issue.Issue, string, error) {
	// The two unique forms first: both carry the ID, so a match is decisive.
	for _, it := range items {
		slug := issue.Slug(it.iss.Title)
		if ref == it.iss.ID || (slug != "" && ref == it.iss.ID+"-"+slug) {
			return it.iss, it.path, nil
		}
	}
	// The slug is not unique: several matches are a SharedSlugError, none falls
	// through to the stale-name form.
	switch m := matchSlug(items, ref); {
	case len(m) == 1:
		return m[0].iss, m[0].path, nil
	case len(m) > 1:
		return issue.Issue{}, "", sharedSlug(ref, m)
	}
	// Stale on-disk name, matched only when the id part differs from the ref
	// itself (a bare id was already tried above).
	if id := issue.IDFromFileName(ref); id != ref {
		for _, it := range items {
			if it.iss.ID == id {
				return it.iss, it.path, nil
			}
		}
	}
	return issue.Issue{}, "", ErrNotFound
}

// resolvable pairs a parsed issue with the path it was read from.
type resolvable struct {
	iss  issue.Issue
	path string
}

// Snapshot is one scan of the store held for several lookups, so a command that
// asks repeatedly pays for one scan instead of one per question. It is an
// explicit point-in-time view, not a hidden cache: the caller chooses its
// lifetime and takes it before mutating anything.
type Snapshot struct {
	items []resolvable
}

// Snapshot scans the store once and returns the point-in-time view. Invalid
// files are skipped and reported through the store's warning handler, like
// every other scan.
func (s *Store) Snapshot() (*Snapshot, error) {
	items, err := s.scan()
	if err != nil {
		return nil, err
	}
	return &Snapshot{items: items}, nil
}

// Resolve matches ref against the snapshot with Store.Resolve's exact contract,
// without re-scanning.
func (sn *Snapshot) Resolve(ref string) (issue.Issue, string, error) {
	return resolveIn(sn.items, ref)
}

// Issues returns every valid issue in the snapshot, in the stable path order
// ReadAll uses.
func (sn *Snapshot) Issues() []issue.Issue {
	issues := make([]issue.Issue, len(sn.items))
	for i, it := range sn.items {
		issues[i] = it.iss
	}
	return issues
}

// IDTaken reports whether an issue with the given id exists in the snapshot,
// matching the authoritative frontmatter id like Store.IDTaken.
func (sn *Snapshot) IDTaken(id string) bool {
	for _, it := range sn.items {
		if it.iss.ID == id {
			return true
		}
	}
	return false
}

// scan reads and validates every issue, pairing each with its path and skipping
// unusable files, each reported through the warning handler. It backs Resolve,
// ReadAll, and IDTaken, so they share one notion of which issues exist.
func (s *Store) scan() ([]resolvable, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}
	items := make([]resolvable, 0, len(files))
	for _, f := range files {
		iss, err := readIssue(f)
		if err != nil {
			s.warn(Warning{Path: f, Err: err})
			continue
		}
		items = append(items, resolvable{iss: iss, path: f})
	}
	return items, nil
}

// matchSlug returns the issues whose canonical (title-derived) slug equals ref.
// An empty slug (a title with no alphanumerics) never matches, so an empty ref
// matches nothing.
func matchSlug(items []resolvable, ref string) []resolvable {
	var out []resolvable
	for _, it := range items {
		if slug := issue.Slug(it.iss.Title); slug != "" && slug == ref {
			out = append(out, it)
		}
	}
	return out
}

// sharedSlug builds a *SharedSlugError from the matches, sorted by ID so the
// candidate listing is deterministic.
func sharedSlug(slug string, matches []resolvable) error {
	issues := make([]issue.Issue, len(matches))
	for i, m := range matches {
		issues[i] = m.iss
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].ID < issues[j].ID })
	return &SharedSlugError{Slug: slug, Matches: issues}
}

// readIssue reads, parses, and validates one issue file, returning the reason
// it is not usable (unreadable, malformed frontmatter, or failed validation)
// with no filename prefix, since the caller pairs the reason with the path.
// Only these hard failures make a file unusable; untidiness like filename
// drift or unknown keys is lint, left for doctor.
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
