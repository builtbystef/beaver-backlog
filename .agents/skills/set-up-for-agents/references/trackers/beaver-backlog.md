# Tracker: Beaver Backlog

How this project translates the tracker verbs. When a skill says the verb on the left, do what follows the arrow. Issues are Markdown files in `.beaver/issues/`. The `beaver` CLI manages them, and manual edits are safe. The files travel with the repository.

## Verbs

- **Publish an issue** → `beaver create "{{title}}" --body "{{body}}"` (for a multi-line body: `--body-file -` with a heredoc). Add `--label {{name}}` for each label, `--priority {{urgent|high|medium|low}}`, `--parent {{ref}}` for the parent, and `--depends-on {{ref}}` for each blocking edge. Create the blockers first, so that `--depends-on` can point to real ids.
- **Fetch an issue** → `beaver show {{ref}}` — the body, state, labels, notes, and blocking relations, in one view.
- **Claim an issue** → `beaver start {{ref}}` — this moves the issue to in-progress and assigns it to you.
- **Note an issue** → `beaver note {{ref}} "{{text}}"`.
- **Close an issue** → `beaver done {{ref}}`.
- **Release an issue** → `beaver update {{ref}} --unassign` — write the reason in a note first, so that the next taker starts informed.
- **List ready work** → `beaver list --ready` — the list knows dependencies, and it sorts by priority first: an issue is ready when its state is todo and each dependency is done. Drop each result with the label `needs-review`.
- **Record a blocking edge** → at creation, `--depends-on {{blocker-ref}}`; on an existing issue, `beaver update {{ref}} --depends-on {{blocker-ref}}` (`--depends-on -{{blocker-ref}}` removes one).

Each other field change is also `beaver update {{ref}}` — `--title`, `--body` / `--body-file` (the description; the notes log stays), `--priority`, `--label`, `--parent` — and one command takes as many flags as you want. Never edit frontmatter by hand to make a structured change. Manual edits are for prose, and `beaver doctor` is the check afterward.

## Roadmap operations

- **Create a roadmap** → `beaver create "{{goal}} — roadmap" --label roadmap --body-file -`, with the overview as the body.
- **Add a sub-issue under a roadmap** → `beaver create "{{question}}" --parent {{roadmap-id}} --label roadmap:{{roadmap-id}} --label session:{{type}}`, plus `--depends-on {{ref}}` for each blocking edge.
- **List a roadmap's ready work** → `beaver list --ready --label roadmap:{{roadmap-id}}`.

## Labels

Labels are free-form. Use the canonical names without changes: `bug`, `spec`, `maintenance`, `review`, `research`, `needs-review`, `roadmap`, `roadmap:{{id}}`, `session:research` / `session:prototype` / `session:grill` / `session:task` / `session:spec`. A spec issue carries `spec` — build its sub-issues, never the spec issue. Apply a label to an existing issue with `beaver update {{ref}} --label {{name}}`. Remove one with `beaver update {{ref}} --label -{{name}}`.

## Closed issues

Done issues stay in `.beaver/issues/`. The CLI keeps them out of the ready queue, so they cost nothing in place, and git history keeps everything that was ever deleted. Do not delete files that the CLI manages, by hand. If the volume ever makes removal necessary, delete old done issues in a batch, in their own commit. Before that, move each decision whose only record is such an issue into `docs/`.

## Committing issue state

Issue files are part of the repository. When you close an issue adjacent to committed work, include the `.beaver/` changes in your commit, so that tracker state travels with the code.

## Capabilities

Full fidelity: native priorities, parents, blocking edges, and a ready queue that knows dependencies. There are no degradations to work around.
