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

// Store is a handle to an initialized .beaver directory.
type Store struct {
	root string // absolute path to the .beaver directory
}

// Root returns the absolute path to the .beaver directory.
func (s *Store) Root() string { return s.root }

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

// ReadAll reads and parses every issue in the store, returning them in the stable
// path order List provides. Files that fail to parse are skipped rather than
// failing the whole read (ADR 0005), mirroring Resolve; surfacing them with a loud
// warning is doctor's job (b8q3). A missing issues directory yields no issues.
// Callers that need a specific display order re-sort the result.
func (s *Store) ReadAll() ([]issue.Issue, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}
	issues := make([]issue.Issue, 0, len(files))
	for _, f := range files {
		iss, err := readIssue(f)
		if err != nil {
			continue // skip invalid files; doctor reports them (b8q3)
		}
		issues = append(issues, iss)
	}
	return issues, nil
}

// IDTaken reports whether an issue with the given ID already exists, so create
// can regenerate on the rare collision. It consults the authoritative frontmatter
// ID — the same identity Resolve uses — so the two can never disagree.
func (s *Store) IDTaken(id string) (bool, error) {
	_, _, ok, err := s.findByID(id)
	return ok, err
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

// Resolve turns a reference into a single issue by its authoritative frontmatter
// ID (ADR 0002, ADR 0005) — never by filename, which is only a human convenience
// and may have drifted via a hand-edit or merge. This slice resolves by exact ID;
// the shared resolver gains prefix and slug support in a later slice (r7p2), which
// every issue-addressing command will route through. Returns ErrNotFound when
// nothing matches.
func (s *Store) Resolve(ref string) (issue.Issue, string, error) {
	iss, path, ok, err := s.findByID(ref)
	if err != nil {
		return issue.Issue{}, "", err
	}
	if !ok {
		return issue.Issue{}, "", ErrNotFound
	}
	return iss, path, nil
}

// findByID scans the store's issue files and returns the first whose authoritative
// frontmatter ID equals id, along with its path. Filenames are never consulted:
// the ID inside the file is the sole identity (ADR 0002, ADR 0005). Files that
// fail to parse are skipped here; `doctor` reports them (b8q3). ok is false when
// nothing matches.
func (s *Store) findByID(id string) (iss issue.Issue, path string, ok bool, err error) {
	files, err := s.List()
	if err != nil {
		return issue.Issue{}, "", false, err
	}
	for _, f := range files {
		i, err := readIssue(f)
		if err != nil {
			continue
		}
		if i.ID == id {
			return i, f, true, nil
		}
	}
	return issue.Issue{}, "", false, nil
}

func readIssue(path string) (issue.Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return issue.Issue{}, err
	}
	iss, err := issue.Unmarshal(data)
	if err != nil {
		return issue.Issue{}, fmt.Errorf("%s: %w", filepath.Base(path), err)
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
