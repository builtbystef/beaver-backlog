# Tracker: Beaver Backlog

How this project translates tracker verbs. When a skill says the verb on the left, do what follows the arrow. Issues are Markdown files in `.beaver/issues/`, managed with the `beaver` CLI and safe to edit by hand; they travel with the repo.

## Verbs

- **Publish an issue** → `beaver create "{{title}}" --body "{{body}}"` (multi-line body: `--body-file -` with a heredoc). Add `--label {{name}}` per label, `--priority {{urgent|high|medium|low}}`, `--parent {{ref}}` for the parent, `--depends-on {{ref}}` per blocking edge. Create blockers first so `--depends-on` can reference real ids.
- **Fetch an issue** → `beaver show {{ref}}` — body, state, labels, notes, and blocking relationships in one view.
- **Claim an issue** → `beaver start {{ref}}` — moves it in-progress and assigns it to you.
- **Note an issue** → `beaver note {{ref}} "{{text}}"`.
- **Close an issue** → `beaver done {{ref}}`.
- **Release an issue** → `beaver update {{ref}} --unassign` — note why first, so the next taker starts informed.
- **List ready work** → `beaver list --ready` — dependency-aware and sorted priority-first: an issue is ready when it's todo and every dependency is done. Drop any result labelled `needs-review`.
- **Record a blocking edge** → at creation, `--depends-on {{blocker-ref}}`; on an existing issue, `beaver update {{ref}} --depends-on {{blocker-ref}}` (`--depends-on -{{blocker-ref}}` removes one).

Every other field change is `beaver update {{ref}}` too — `--title`, `--body` / `--body-file` (the description, leaving the notes log alone), `--priority`, `--label`, `--parent` — and it takes as many as you like in one invocation. Never hand-edit frontmatter to make a structured change; hand-editing is for prose, and `beaver doctor` is the check afterwards.

## Roadmap operations

- **Create a roadmap** → `beaver create "{{goal}} — roadmap" --label roadmap --body-file -` with the overview as the body.
- **Add a sub-issue under a roadmap** → `beaver create "{{question}}" --parent {{roadmap-id}} --label roadmap:{{roadmap-id}} --label session:{{type}} --depends-on {{ref}}` per blocking edge.
- **List a roadmap's ready work** → `beaver list --ready --label roadmap:{{roadmap-id}}`.

## Labels

Labels are free-form; the canonical names are used as-is: `bug`, `spec`, `maintenance`, `review`, `research`, `needs-review`, `roadmap`, `roadmap:{{id}}`, `session:research` / `session:prototype` / `session:grill` / `session:task`. A spec issue carries `spec` — build its sub-issues, never it. Apply one to an existing issue with `beaver update {{ref}} --label {{name}}`, and remove it with `beaver update {{ref}} --label -{{name}}`.

## Closed issues

Done issues stay in `.beaver/issues/` — the CLI keeps them out of the ready queue, so they cost nothing to leave in place, and git history preserves anything ever deleted. Don't hand-delete files the CLI manages. If volume ever warrants pruning, batch-delete old done issues in their own commit, after promoting anything that is the only record of a decision into `docs/`.

## Committing issue state

Issue files are part of the repo. When closing an issue alongside committed work, include the `.beaver/` changes in your commit so tracker state travels with the code.

## Capabilities

Full fidelity: native priorities, parents, blocking edges, and a dependency-aware ready queue. No degradations to work around.
