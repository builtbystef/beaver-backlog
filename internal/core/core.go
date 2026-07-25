// Package core is Beaver Backlog's application layer: the operations every
// interface — today's CLI, tomorrow's web UI or SDK — performs on a store. It
// owns the rules (what a reference resolves to, which issues a query selects,
// how they are ordered) and knows nothing about flags, terminals, or exit
// codes: failures come back as typed errors and skipped files as data, for the
// caller to present its own way.
package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/builtbystef/beaver-backlog/internal/clock"
	"github.com/builtbystef/beaver-backlog/internal/issue"
	"github.com/builtbystef/beaver-backlog/internal/store"
)

// Sentinel errors callers branch on, each mapped to the caller's own
// diagnostic.
var (
	// ErrNoStore means no store was found at or above the directory Open was
	// given.
	ErrNoStore = errors.New("not a Beaver Backlog store")
	// ErrNotFound means a reference matched no issue.
	ErrNotFound = errors.New("issue not found")
)

// AmbiguousRefError reports that a reference names several issues rather than
// one — the case of a slug two issues share. It carries the candidates (sorted
// by ID) so a caller can list them, and unwraps to ErrNotFound because the
// reference still named no single issue.
type AmbiguousRefError struct {
	Ref     string        // the reference as given
	Matches []issue.Issue // the issues it names, sorted by ID
}

func (e *AmbiguousRefError) Error() string {
	ids := make([]string, len(e.Matches))
	for i, m := range e.Matches {
		ids[i] = m.ID
	}
	return fmt.Sprintf("%q names %d issues (%s)", e.Ref, len(e.Matches), strings.Join(ids, ", "))
}

func (e *AmbiguousRefError) Unwrap() error { return ErrNotFound }

// Warning reports a file a read skipped because it is not a usable issue: Path
// names the file and Err says what is wrong with it. Warnings are data, not
// output — no core operation prints or fails on one, so every interface decides
// for itself how to surface a broken file (ADR 0003).
type Warning struct {
	Path string // path to the skipped file
	Err  error  // the specific reason it is not a usable issue
}

// Service is the application over one store, together with the seams its
// operations depend on: the clock that stamps a write and the source new issues
// draw their IDs from. Construct it with Open.
type Service struct {
	store *store.Store
	clock clock.Clock
	newID func() string
}

// Option customizes a Service at construction; the defaults are the real clock
// and the real ID generator.
type Option func(*Service)

// WithClock replaces the source of the timestamps writes record.
func WithClock(c clock.Clock) Option { return func(s *Service) { s.clock = c } }

// WithIDSource replaces the generator new issues draw their IDs from.
func WithIDSource(gen func() string) Option { return func(s *Service) { s.newID = gen } }

// Open finds the store by walking up from dir and returns the service over it.
// It returns ErrNoStore when neither dir nor any of its ancestors holds one.
func Open(dir string, opts ...Option) (*Service, error) {
	st, err := store.Discover(dir)
	if err != nil {
		if errors.Is(err, store.ErrNoStore) {
			return nil, ErrNoStore
		}
		return nil, err
	}
	s := &Service{store: st, clock: clock.System(), newID: issue.NewID}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Detail is one issue together with the relationship facts derived from the
// rest of the store — what it waits on, whether it is ready, blocked, or stuck,
// and the inverse edges that are never stored.
type Detail struct {
	Issue        issue.Issue
	Relationship issue.Relationship
	Warnings     []Warning
}

// Get resolves ref — a full ID, a slug, or an "<id>-<slug>" file name, stale or
// canonical — to one issue and derives its relationships over the same scan. It
// returns ErrNotFound for an unknown reference and an *AmbiguousRefError for one
// that names several issues. Warnings are reported whatever the outcome, so a
// failed lookup still tells the caller which files were skipped.
func (s *Service) Get(ref string) (Detail, error) {
	snap, warnings, err := s.scan()
	if err != nil {
		return Detail{Warnings: warnings}, err
	}
	iss, err := resolve(snap, ref)
	if err != nil {
		return Detail{Warnings: warnings}, err
	}
	rel := issue.NewRelations(snap.Issues()).For(iss)
	return Detail{Issue: iss, Relationship: rel, Warnings: warnings}, nil
}

// scan takes one point-in-time view of the store, collecting the files it skips
// as warnings instead of reporting them itself. Every read goes through it, so
// a broken file surfaces the same way whichever operation ran.
func (s *Service) scan() (*store.Snapshot, []Warning, error) {
	var warnings []Warning
	s.store.OnWarn(func(w store.Warning) {
		warnings = append(warnings, Warning{Path: w.Path, Err: w.Err})
	})
	snap, err := s.store.Snapshot()
	return snap, warnings, err
}

// resolve turns a user reference into one issue of the scanned set,
// translating the store's resolution failures into the core's typed errors.
func resolve(snap *store.Snapshot, ref string) (issue.Issue, error) {
	iss, _, err := snap.Resolve(ref)

	// SharedSlugError unwraps to the store's not-found, so it must be matched
	// before the generic not-found branch swallows it.
	var shared *store.SharedSlugError
	switch {
	case err == nil:
		return iss, nil
	case errors.As(err, &shared):
		return issue.Issue{}, &AmbiguousRefError{Ref: shared.Slug, Matches: shared.Matches}
	case errors.Is(err, store.ErrNotFound):
		return issue.Issue{}, fmt.Errorf("%w: %s", ErrNotFound, ref)
	default:
		return issue.Issue{}, err
	}
}
