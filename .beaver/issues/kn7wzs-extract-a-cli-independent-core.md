---
id: kn7wzs
title: Extract a CLI-independent core
state: done
labels:
    - spec
depends_on:
    - tlz52g
created: 2026-07-25T08:42:28Z
updated: 2026-07-30T03:40:11Z
---

# Extract a CLI-independent core

## Problem Statement

The CLI package *is* the application: transition legality, claim rules, label algebra, the query engine, ID minting, the `updated`-bump policy, and the entire doctor engine live inside command handlers, tangled with flag parsing, stderr writes, and exit codes. A web UI, an SDK, or any other interface would have to shell out to the binary or duplicate the rules.

## Solution

A new core package becomes the application: a service over the store that every consumer — today's CLI, tomorrow's web UI or SDK — calls. The CLI shrinks to one interface: parse arguments, call the core, render the result, map typed errors to exit codes. User-visible behavior does not change at all in this spec.

## User Stories

1. As a maintainer, I want the business rules in one CLI-free package, so that a future web UI or SDK reuses them instead of shelling out or reimplementing.
2. As a CLI user, I want every command to behave exactly as before, so that this refactor is invisible to me.
3. As a future interface author, I want typed errors and warnings-as-data instead of exit codes and stderr text, so that I can present failures my own way.
4. As a test author, I want the rules testable at the core seam, so that behavior coverage doesn't require driving a CLI.

## Implementation Decisions

- New package `internal/core` — the application facade. Designed as if public (no CLI types in its signatures) but kept under `internal/` until an external consumer exists; promotion later is a mechanical move.
- The core API is shaped for the **final** command surface (defined in the consolidation spec): a single multi-field `Update` from day one. The existing commands become thin wrappers over it — `assign`/`release`/`priority`/`label` each map to an `Update` with one field set. `claim`'s `--force` steal guard stays in its CLI wrapper for the interim (read the issue, compare assignees, refuse without force) and is deleted along with the command in the consolidation spec.
- Moves into core: the transition legality table and no-op classification; `start`'s compound behavior (transition + claim + unmet-dependency detection); label set algebra (removal wins); priority parsing; the list query/filter/sort engine; ref resolution; ID minting with collision retry; the `updated`-bump policy (centralized in one place: truncate to seconds, UTC); the entire doctor engine including fixes.
- Stays interface-side: flag parsing, output rendering, exit-code mapping, TTY detection, identity resolution (the environment-variable → user-config → prompt chain; core methods take the actor as a plain string parameter), editor invocation (until the consolidation spec removes it), and per-machine user config.
- The service is constructed with an injectable clock and ID generator (production defaults); these replace the corresponding `Env` fields as the seam for time and identity of new issues. The CLI `Env` keeps only what the interface layer itself needs.
- Skip-and-warn on invalid files (existing graceful-degradation ADR) becomes data: core results carry warnings; the CLI prints them to stderr as today.

### Core API contract (the new seam)

```go
package core

func Open(dir string, opts ...Option) (*Service, error) // walks up from dir to find the store
// Options: WithClock(clock.Clock), WithIDSource(func() string)

// Reads
func (s *Service) Get(ref string) (Detail, error)   // issue + derived relationships/readiness
func (s *Service) List(q Query) (Listing, error)

// Writes
func (s *Service) Create(d Draft) (issue.Issue, error)
func (s *Service) Transition(ref string, to issue.State) (Outcome, error) // done/cancelled/todo; legality enforced
func (s *Service) Start(ref, actor string, force bool) (StartOutcome, error)
func (s *Service) Update(ref string, c Changes) (Outcome, error)
func (s *Service) Note(ref, actor, text string) (issue.Issue, error)
func (s *Service) Delete(ref string) error
func (s *Service) Doctor(fix bool) (Report, error)

type Draft struct {
    Title, Body string
    Labels      []string
    Priority    issue.Priority
    DependsOn   []string
    Parent      string
}

type Changes struct {
    Title, Body                   *string // nil = leave untouched
    Assignee                      *string // non-nil empty string clears
    Priority                      *issue.Priority // includes the none value to clear
    AddLabels, RemoveLabels       []string
    AddDependsOn, RemoveDependsOn []string
    Parent                        *string // non-nil empty string clears
}

type Outcome struct {
    Issue   issue.Issue
    Changed bool // false = net no-op: nothing was written, updated untouched
}
type StartOutcome struct {
    Outcome
    UnmetDependencies []issue.Issue
}

type Query struct {
    States     []issue.State
    Ready, Blocked bool
    Labels     []string
    Priorities []issue.Priority
    Assignee   *string
}
type Listing struct {
    Issues   []issue.Issue
    Warnings []Warning // skipped invalid files
}
```

Typed errors (sentinel or small structs — e.g. not-found, ambiguous ref, no store, illegal transition with from/to states, validation failure). The CLI maps them to today's exit codes and diagnostic messages; both are preserved exactly, enforced by the existing end-to-end suite.

- Call-path change: every command handler becomes parse → resolve format → core call → render + exit-code map. No handler touches the store, the clock, or issue files directly anymore. The end-to-end harness keeps driving the CLI engine unchanged.

## Dependencies

None.

## Testing Decisions

Two seams:

- **The core API (new — becomes the primary behavior suite).** As each rule moves, it gains core-seam tests: transition legality and no-op classification, label algebra (removal wins over add), priority parsing, ready/blocked/stuck queries, the update net-change rule, ID collision retry, doctor findings and fixes.
- **The existing end-to-end CLI harness (unchanged — the characterization net).** The full suite must pass throughout the extraction; it pins exit codes, JSON shapes, and stderr text. It thins only in the consolidation spec.

Worked examples for the core seam:
- `Update` with `AddLabels ["a"]`, `RemoveLabels ["a"]` on an unlabelled issue → `Changed: false`, nothing written.
- `Transition` to `done` on an already-`done` issue → `Changed: false` (idempotent no-op, matching today's redundant-`done` behavior).
- `Start` on an issue with an unmet dependency → succeeds, `UnmetDependencies` non-empty (warning, not error — matches today).

Prior art: the existing unit suites in the store and issue packages show the internal-test style.

## Out of Scope

- Any user-visible change to commands, output, or exit codes.
- Command consolidation (next spec) — ideally that spec needs zero core changes; if it does need one, the API missed something and the miss is the finding.
- Promoting the core out of `internal/`, a web UI, or an SDK.

## Further Notes

Second of three sequenced specs. Depends on VCS removal having landed (smaller `Env`, no commit paths to migrate). The coding-standards test-policy rewrite happens in the consolidation spec, when the end-to-end suite actually thins.

## Notes

**claude** — 2026-07-25T10:15:11Z

StartOutcome.UnmetDependencies landed as []issue.Blocker, not []issue.Issue as this spec sketches: a dangling depends_on target has no issue to return, and start's warning names it — gone99 (missing). StartOutcome also carries Previous (the issue as it stood before) and Relationship, which the CLI needs to render start's confirmation line and its JSON readiness view.

**claude** — 2026-07-26T19:11:41Z

Two findings from the update slice (u09zmf).

Outcome gained a Previous field — the issue as it stood before the call — and StartOutcome's own Previous moved up into it. A caller cannot describe what an update did from the result alone: release reports "Released <id> (was <actor>)", and the assignee it names is exactly what the write cleared.

Cycle detection has no create-side counterpart to match. This spec and the consolidation spec both say update's cycle rule works "as at create", but create runs none — a newly minted issue can only depend on issues that already exist, so it can never close a loop. Update introduces the rule, and *CycleError with it, refusing only a cycle the change itself introduces: one that arrived by merge or hand-edit stays doctor's (ADR 0005), and refusing every edit to an issue caught in one would leave no way to edit it back out. So the update-command slice (u3krpx) must pick that refusal's wording and exit code itself rather than copy create's; nothing maps *CycleError in the CLI yet.

One more shape decision u3krpx will document: a depends-on removal takes its reference as written when nothing resolves it, so "-gone99" drops a dangling edge. Additions stay strict.

**claude** — 2026-07-26T20:24:42Z

Four findings from the doctor and delete slice (3agw9c), which completes the contraction.

Doctor's findings crossed the seam as facts rather than as prose. Every message doctor prints embeds a file path rendered relative to where the command ran, so a core that built the sentences would own a display concern. core.Finding instead carries the anchors (the files at fault, the ids) and the facts of its class — the canonical name a drifted file should hold, the suspect key and the field it resembles, the unrecognized priority value, the timestamp fields an issue lacks, the field and target of a dangling edge, the cancelled dependencies a stuck issue waits on — and the CLI words them. Report answers Problems, Advisories, Fixed, and Fixable, since the exit code and the ok flag turn on the count of what still stands. Doctor also no longer bypasses store discovery to avoid double-reporting: unusable files come back as findings and the scan raises no warnings at all.

Delete returns a Deleted (issue, path, warnings), not the bare error the spec sketches. The human confirmation names the id, the title, and the file it removed, and none of that survives an error-only signature — the same reason Create returns a Created.

Two operations the API list does not mention had to exist for "no handler touches the store" to be true. core.Init wraps store creation, which init called directly; and Editable/Reread are the hand-editing seam, which edit needed because a hand-edit happens in the file rather than through an operation the core could apply. Editable is the only core read that hands out a path, and both it and Reread die with the editor machinery in 0b8jtl.

Env shed Clock and NewID for CoreOptions []core.Option, so the harness's fakes now reach the application rather than the interface. With that and the last of the pre-core plumbing deleted (discover, storeError, resolveRef and its resolver interface, warnInvalid), internal/cli no longer imports internal/store at all — a checkable statement of the contraction, and the invariant the next slice inherits.
