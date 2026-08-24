# Tracker verbs — the contract

This file defines the operations that a skill can ask of a project's tracker. Skills speak these verbs and nothing else: no tracker names, no commands. The project's `docs/TRACKER.md`, written from a seed template in [trackers/](trackers/), translates each verb into the selected tracker's form. Each template translates each verb, at the fidelity that its tracker permits, and it records where fidelity degrades. Skills trust the doc, and they never doubt it.

## Core verbs

- **Publish an issue** — create an issue from a title and a body; optionally with labels, a priority, a parent issue, and blocking edges.
- **Fetch an issue** — read one issue in full: body, state, labels, notes.
- **Claim an issue** — mark the issue as in progress and assigned to the current actor, so that parallel sessions do not take it.
- **Note an issue** — append to its coordination log: decisions made, blockers found, anything that the next reader needs.
- **Close an issue** — mark the issue as done.
- **Release an issue** — clear the claim, and return the issue to the pool. Keep its state for the next taker.
- **List ready work** — the issues that can start now: open, unclaimed, no unmet blocking edges, and not labelled `needs-review`. The computation of "ready" varies by tracker. Each template says how.
- **Record a blocking edge** — declare that one issue cannot start until another issue is done.

## Roadmap operations

These richer verbs serve only the roadmap workflow in planning. Each template has a section named **Roadmap operations** that translates them:

- **Create a roadmap** — publish a root issue, labelled `roadmap`, that holds the planning overview.
- **Add a sub-issue under a roadmap** — publish an issue with the roadmap's root issue as parent, the label `roadmap:{{roadmap-id}}` plus a session-type label, and blocking edges to the sub-issues that must resolve first.
- **List a roadmap's ready work** — list ready work, limited to one roadmap's sub-issues. As sub-issues close, blocked sub-issues become ready.

## Canonical labels

Skills use these names only. Each template maps them onto its tracker's label mechanism.

- `bug` — intake: a reported defect that waits for diagnosis.
- `spec` — a spec issue: its body is the specification, and its sub-issues are the implementation slices. Build the sub-issues, never the spec issue.
- `maintenance` — follow-up work, filed by audits, reviews, and post-fix reflection.
- `research` — a recorded finding: a question and its answer, from a research or prototype session that ran without an issue that existed before. It is published, noted with the report, and closed in one step.
- `review` — a code-review finding that the user accepted.
- `needs-review` — the issue waits for the user. Issues with this label never appear in ready work. Two situations produce it. The note that applies the label must say which situation, and what the user must do:
  - **A deliverable waits for approval.** This was requested at creation, by a body line that says that closure waits for user review. The session that finishes writes the deliverable in a note, applies the label, and releases the claim. It does not close. The user closes to approve, or notes the requested changes and removes the label. Then the issue returns to ready work.
  - **A decision blocks the issue**, and only the user can make it. The blocked session notes the specific question and the options that it examined, applies the label, and releases the claim. The user notes the decision and removes the label. Then the issue returns to ready work.
- `roadmap`, `roadmap:{{id}}`, `session:research` / `session:prototype` / `session:grill` / `session:task` / `session:spec` — planning across sessions (see above).
