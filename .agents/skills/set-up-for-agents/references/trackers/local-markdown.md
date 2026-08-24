# Tracker: local Markdown

How this project translates the tracker verbs. When a skill says the verb on the left, do what follows the arrow. The tracker is plain Markdown files in `scratch/` at the repository root. There is no CLI, no service, and nothing to install. The files travel with the repository, and each operation is a file read or a file edit. `scratch/` holds working state with a lifecycle.

## Issue file format

One file for each issue: `scratch/NNN-{{slug}}.md`. Number `NNN` up from the highest number that exists across `scratch/` and `scratch/done/`. The issue's ref is its number.

```markdown
# {{Title}}

- State: todo | in-progress | done
- Assignee: {{name, or blank}}
- Priority: urgent | high | medium | low
- Labels: {{comma-separated}}
- Parent: {{ref, or omit the line}}
- Blocked by: {{refs, comma-separated, or omit the line}}

{{body}}

## Notes

- YYYY-MM-DD {{actor}}: {{text}}
```

## Verbs

- **Publish an issue** → create the file from the format above. Fill the title, the body, the labels, the priority, the parent, and the `Blocked by` refs. Create the blockers first, so that the refs are real.
- **Fetch an issue** → read the file. Open issues live in `scratch/`. Closed issues live in `scratch/done/`.
- **Claim an issue** → set `State: in-progress`, and set `Assignee:` to yourself.
- **Note an issue** → append a dated line under `## Notes`.
- **Close an issue** → set `State: done`, and move the file to `scratch/done/`, with the same filename.
- **Release an issue** → write the reason in a note, clear the `Assignee:` line, and set `State: todo`.
- **List ready work** → scan `scratch/` (not `scratch/done/`). An issue is ready when: its state is todo; it has no assignee; its Labels line does not contain `needs-review`; and each ref on its `Blocked by` line points to an issue in `scratch/done/`. Order by the Priority line, urgent first.
- **Record a blocking edge** → add the blocker's ref to the blocked issue's `Blocked by` line. Create the line if it is missing.

## Closed issues

Closed issues live in `scratch/done/`: the ready-work scan never reads them, their refs stay resolvable, and their numbers are never used again. Git history is the archive. During a tracker-health pass, delete old files from `done/` when no open issue points to them on its `Blocked by` or `Parent` line. Before you delete, move each decision whose only record is such a file into `docs/` — a closed roadmap node's answer has no other home when its file is gone.

## Roadmap operations

- **Create a roadmap** → publish an issue with the title `{{goal}} — roadmap`, the label `roadmap`, and the overview as the body.
- **Add a sub-issue under a roadmap** → publish an issue with `Parent:` set to the root issue's ref, the labels `roadmap:{{roadmap-ref}}` and `session:{{type}}`, and a `Blocked by` ref for each edge.
- **List a roadmap's ready work** → the ready-work scan, limited to the files whose Labels line contains `roadmap:{{roadmap-ref}}`.

## Labels

The `Labels:` line is free-form. Use the canonical names without changes: `bug`, `spec`, `maintenance`, `review`, `research`, `needs-review`, `roadmap`, `roadmap:{{ref}}`, `session:research` / `session:prototype` / `session:grill` / `session:task` / `session:spec`. A spec issue carries `spec` — build its sub-issues, never the spec issue.

## Capabilities

Everything is a convention over files, so fidelity is exactly as good as the discipline. "Ready" is a manual scan. A claim does not lock anything against a parallel session. There is no integrity check beyond what a reader notices. In exchange: zero dependencies, full offline operation, and tracker state that versions with the code. Commit the issue-file changes together with the work that they track.
