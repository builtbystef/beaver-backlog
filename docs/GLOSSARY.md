# Glossary

The project's ubiquitous language. Rules: one term per concept — rejected synonyms go under _Avoid_; a definition is one or two sentences saying what the term IS, not what it does; only terms specific to this project belong (general programming concepts don't); no implementation details; group under subheadings when clusters emerge.

## Language

**Issue**:
The unit of work Beaver Backlog tracks — a bug, feature, chore, or any discrete piece of
work. Each issue is a single Markdown file living in the repository.
_Avoid_: Task, Ticket

**Issue ID**:
The canonical identifier of an issue — a short, randomly generated,
collision-resistant handle (deliberately not a sequential number). Stable for
the life of the issue, unaffected by any later edits to its content.
_Avoid_: Number, Index

**Slug**:
A human-readable label derived from an issue's title, used to recognize an issue
at a glance and as a secondary way to refer to it. Not the issue's identity — it
may go stale if the title changes.
_Avoid_: Name, Handle

**State**:
An issue's position in its lifecycle, stored as a field in its frontmatter. One
of four fixed values: `todo` (not started), `in-progress` (actively being
worked), `done` (completed), `cancelled` (deliberately abandoned — terminal but
not completed). The set is fixed.
_Avoid_: Status, Stage

**Open / Closed**:
Umbrella query categories, not stored values. _Open_ is any non-terminal issue
(`todo` or `in-progress`); _Closed_ is any terminal issue (`done` or
`cancelled`). Never use "open" to mean specifically "not started" — that sense is
`todo`.

**Actor**:
A participant Beaver Backlog can attribute work to — a human or a coding agent, treated
identically. An actor is just a free-form name (e.g. `stefan`, `claude`); Beaver Backlog
does not formally distinguish humans from agents.
_Avoid_: User, Person, Member, Contributor

**Assignee**:
The single actor currently responsible for an issue, recorded in the optional
`assignee` frontmatter field. An issue has at most one assignee — the primitive
that stops two actors claiming the same work.
_Avoid_: Owner

**Label**:
A free-form, multi-valued tag on an issue, used for any classification the team
wants — including its "type" (`bug`, `feature`, `chore`). Beaver Backlog has no separate
type or category concept; those are simply labels.
_Avoid_: Tag, Type, Category

**Priority**:
An optional ordinal ranking of an issue's urgency, stored in the `priority`
frontmatter field. One of four fixed levels — `urgent`, `high`, `medium`, `low`
— or absent for no priority. Used to sort and triage what to work on first.
_Avoid_: Severity, Importance

**Claim**:
The act of an actor making themselves an issue's assignee to signal "this is
mine." Advisory only — a coordination signal, never a lock; Beaver Backlog cannot stop
two actors on different branches from claiming the same issue, and the VCS merge
surfaces the clash. Claiming reserves an issue without changing its state.
_Avoid_: Lock

**Dependency**:
A relationship recording that one issue waits on another: "A depends on B" means A
should not proceed until B is done. Stored one-sided as `depends_on` on the
dependent issue; the inverse (what an issue blocks) is derived, never stored.
_Avoid_: Blocker

**Blocked / Ready**:
Derived conditions, not stored states. A dependency is satisfied only when its
target is `done`. An issue is _blocked_ when any `depends_on` target is not
`done`, and _ready_ when it is `todo` with every dependency `done`. A dependency
on a `cancelled` issue is never satisfied, leaving the dependent _stuck_ — a
condition `doctor` flags for a human to resolve, not one Beaver Backlog clears
automatically. Like Open/Closed, these are query views over the data — never
written to a file.

**Sub-issue**:
An issue that names another as its `parent`, used to decompose larger work. An
issue that has sub-issues is informally an "epic," but Beaver Backlog has no distinct epic
type — it is just an issue with sub-issues.
_Avoid_: Subtask, Child

**Board**:
The web UI's primary view: one column per state, each issue a draggable card.
A presentation of the store, not stored data — column membership is just the
issue's state, and card order is the same fixed ordering every list uses.
_Avoid_: Kanban view, Dashboard

**Quick view**:
A compact, in-place summary of an issue — its key fields and conditions —
shown without leaving the current view, with a way through to the full issue
page. Used on the graph view to inspect a node where navigating away would
lose the reader's place.
_Avoid_: Preview, Popup

**Note**:
A flat, append-only, attributed and timestamped entry appended to an issue's body
— a coordination journal for humans and agents ("tried X, see commit abc; handing
back"). Not threaded: no replies, and no editing another actor's entries.
_Avoid_: Comment, Discussion

**Project name**:
What the project a store belongs to is called — by default the name of the
directory the store sits in, so every project has one without being configured.
_Avoid_: Store name, Workspace, Repository name
