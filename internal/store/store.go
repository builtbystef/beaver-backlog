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
	ErrNoStore = errors.New("not a Beaver store; run `beaver init`")
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
// whether the store directory was newly made. Init never touches a VCS; Beaver
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
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.IssuesDir())
	if err != nil {
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

// IDTaken reports whether an issue with the given ID already exists, so create
// can regenerate on the rare collision. It checks the filename's ID portion,
// which Beaver always keeps in sync with the frontmatter on write.
func (s *Store) IDTaken(id string) (bool, error) {
	files, err := s.List()
	if err != nil {
		return false, err
	}
	for _, f := range files {
		if issue.IDFromFileName(filepath.Base(f)) == id {
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

// Resolve turns a reference into a single issue. This slice resolves by exact ID
// (the authoritative identity); the shared resolver gains prefix and slug support
// in a later slice (r7p2), which every issue-addressing command will route
// through. Returns ErrNotFound when nothing matches.
func (s *Store) Resolve(ref string) (issue.Issue, string, error) {
	files, err := s.List()
	if err != nil {
		return issue.Issue{}, "", err
	}

	// Fast path: a file whose name mirrors the ID — the normal case, since Beaver
	// always names files <id>-<slug>.md. A parse failure here is reported with the
	// file name rather than masked as "not found".
	for _, f := range files {
		if issue.IDFromFileName(filepath.Base(f)) == ref {
			iss, err := readIssue(f)
			if err != nil {
				return issue.Issue{}, "", err
			}
			return iss, f, nil
		}
	}

	// Fallback: the authoritative ID in the frontmatter, in case a filename has
	// drifted from its ID via a hand-edit or merge (ADR 0002, ADR 0005). Files
	// that fail to parse are skipped here; `doctor` reports them (b8q3).
	for _, f := range files {
		iss, err := readIssue(f)
		if err != nil {
			continue
		}
		if iss.ID == ref {
			return iss, f, nil
		}
	}

	return issue.Issue{}, "", ErrNotFound
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
		`# Beaver project configuration.
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
