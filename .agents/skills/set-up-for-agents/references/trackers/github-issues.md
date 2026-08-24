# Tracker: GitHub Issues

How this project translates the tracker verbs. When a skill says the verb on the left, do what follows the arrow. Issues live in the repository's GitHub Issues, worked through the `gh` CLI. GitHub has no native priorities and no native blocking edges. Both are conventions that this file defines. Follow them exactly, or ready-work queries stop being reliable.

## Conventions

- **Priority** → a `priority:{{urgent|high|medium|low}}` label.
- **Blocking edge** → a `Blocked by #{{n}}` line in the blocked issue's body, one line for each blocker.
- **Parent** → a `Parent: #{{n}}` line in the child's body. The parent keeps a task list (`- [ ] #{{n}}`) of its children, so that progress shows on the parent.

## Verbs

- **Publish an issue** → `gh issue create --title "{{title}}" --body "{{body}}"`, with `--label {{name}}` for each label (priority labels and canonical labels). The parent and the blocking edges go in the body, as the convention lines above. Create the blockers first, so that the `#{{n}}` refs are real.
- **Fetch an issue** → `gh issue view {{n}} --comments`.
- **Claim an issue** → `gh issue edit {{n}} --add-assignee @me`.
- **Note an issue** → `gh issue comment {{n}} --body "{{text}}"`.
- **Close an issue** → `gh issue close {{n}}`.
- **Release an issue** → write the reason in a comment. Then `gh issue edit {{n}} --remove-assignee @me`.
- **List ready work** → `gh issue list --state open --search "no:assignee -label:needs-review"`. Then fetch each candidate's body, and drop each issue whose `Blocked by #{{n}}` lines name an issue that is still open. Order by the priority label, urgent first.
- **Record a blocking edge** → read the blocked issue's body (`gh issue view {{n}} --json body -q .body`), append the `Blocked by #{{n}}` line, and write the body back with `gh issue edit {{n}} --body-file -`.

## Roadmap operations

- **Create a roadmap** → publish an issue with the title `{{goal}} — roadmap`, the label `roadmap`, and the overview as the body.
- **Add a sub-issue under a roadmap** → publish an issue with the labels `roadmap:{{roadmap-number}}` and `session:{{type}}`, a `Parent: #{{roadmap-number}}` body line, and a `Blocked by` line for each edge. Add the issue to the root issue's task list.
- **List a roadmap's ready work** → `gh issue list --state open --label "roadmap:{{roadmap-number}}"`. Then do the same Blocked-by scan as for ready work.

## Labels

The canonical names map to repository labels with the same names: `bug`, `spec`, `maintenance`, `review`, `research`, `needs-review`, `roadmap`, plus `roadmap:{{n}}` and `session:research` / `session:prototype` / `session:grill` / `session:task` / `session:spec`, created when needed (`gh label create "{{name}}"` — a label must exist before an issue can carry it). A spec issue carries `spec` — build its sub-issues, never the spec issue.

## Capabilities

Degradations to know: priority is a label, not a sortable field. Blocking edges and parents are body-line conventions, invisible to GitHub's own UI relations. "Ready" is a client-side scan, not a server query — it is only as accurate as the convention lines are complete. Assignment, comments, and state are native and fully reliable.
