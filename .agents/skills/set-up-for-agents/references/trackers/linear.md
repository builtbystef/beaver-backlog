# Tracker: Linear

How this project translates the tracker verbs. When a skill says the verb on the left, do what follows the arrow. Issues live in Linear, team `{{TEAM}}`, worked through the Linear MCP tools (or Linear's GraphQL API, where no MCP connection exists). Linear is rich natively: priorities, parents, and blocking relations are first-class. Use them. Never use body-line conventions.

## Verbs

- **Publish an issue** → create an issue in team `{{TEAM}}`, with the title and the description. Set the labels, the priority (Urgent/High/Medium/Low map directly), the parent issue, and a "blocked by" relation for each blocking edge.
- **Fetch an issue** → get the issue by identifier (for example `{{TEAM}}-123`), with the description, state, labels, relations, and comments.
- **Claim an issue** → assign the issue to yourself, and move it to the team's In Progress state.
- **Note an issue** → add a comment.
- **Close an issue** → move the issue to the team's Done state.
- **Release an issue** → write the reason in a comment, unassign the issue, and move it back to the team's Todo state.
- **List ready work** → the issues in the Todo state, unassigned, with no open "blocked by" relation, and without the `needs-review` label. Linear evaluates the blocked status natively, so this list knows dependencies. Order by priority, urgent first.
- **Record a blocking edge** → create a "blocked by" relation from the blocked issue to its blocker.

## Roadmap operations

- **Create a roadmap** → publish an issue with the title `{{goal}} — roadmap`, the label `roadmap`, and the overview as the description.
- **Add a sub-issue under a roadmap** → publish an issue with the roadmap's root issue as its native parent, the labels `roadmap:{{roadmap-identifier}}` and `session:{{type}}`, and a "blocked by" relation for each edge.
- **List a roadmap's ready work** → ready work, filtered to the label `roadmap:{{roadmap-identifier}}`.

## Labels

The canonical names map to team labels with the same names: `bug`, `spec`, `maintenance`, `review`, `research`, `needs-review`, `roadmap`, plus `roadmap:{{identifier}}` and `session:research` / `session:prototype` / `session:grill` / `session:task` / `session:spec`, created when needed — a Linear label must exist in the team before an issue can carry it. Priority is Linear's native field, never a label. A spec issue carries `spec` — build its sub-issues, never the spec issue.

## Capabilities

Full fidelity: native priorities, parents, blocking relations, and a ready query that knows dependencies. One seam to know: state names vary from team to team. Resolve the team's actual Todo / In Progress / Done states one time, and use those. Do not assume the default names.
