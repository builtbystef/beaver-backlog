# Busy Beaver

Busy Beaver is a local-first issue tracker for software projects. It stores issues as
Markdown files inside the project so that humans and coding agents can coordinate
work through the files themselves — with no external service and no hard
dependency on any version-control system. Git is the first-class companion, but
Busy Beaver stays correct with another VCS, or none.

## Language

**Issue**:
The unit of work Busy Beaver tracks — a bug, feature, chore, or any discrete piece of
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
Umbrella query categories, not stored values. *Open* is any non-terminal issue
(`todo` or `in-progress`); *Closed* is any terminal issue (`done` or
`cancelled`). Never use "open" to mean specifically "not started" — that sense is
`todo`.

**Actor**:
A participant Busy Beaver can attribute work to — a human or a coding agent, treated
identically. An actor is just a free-form name (e.g. `stefan`, `claude`); Busy Beaver
does not formally distinguish humans from agents.
_Avoid_: User, Person, Member, Contributor

**Assignee**:
The single actor currently responsible for an issue, recorded in the optional
`assignee` frontmatter field. An issue has at most one assignee — the primitive
that stops two actors claiming the same work.
_Avoid_: Owner

**Label**:
A free-form, multi-valued tag on an issue, used for any classification the team
wants — including its "type" (`bug`, `feature`, `chore`). Busy Beaver has no separate
type or category concept; those are simply labels.
_Avoid_: Tag, Type, Category

**Priority**:
An optional ordinal ranking of an issue's urgency, stored in the `priority`
frontmatter field. One of four fixed levels — `urgent`, `high`, `medium`, `low`
— or absent for no priority. Used to sort and triage what to work on first.
_Avoid_: Severity, Importance

**Claim**:
The act of an actor making themselves an issue's assignee to signal "this is
mine." Advisory only — a coordination signal, never a lock; Busy Beaver cannot stop
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
target is `done`. An issue is *blocked* when any `depends_on` target is not
`done`, and *ready* when it is `todo` with every dependency `done`. A dependency
on a `cancelled` issue is never satisfied, leaving the dependent *stuck* — a
condition `doctor` flags for a human to resolve, not one Busy Beaver clears
automatically. Like Open/Closed, these are query views over the data — never
written to a file.

**Sub-issue**:
An issue that names another as its `parent`, used to decompose larger work. An
issue that has sub-issues is informally an "epic," but Busy Beaver has no distinct epic
type — it is just an issue with sub-issues.
_Avoid_: Subtask, Child

**Note**:
A flat, append-only, attributed and timestamped entry appended to an issue's body
— a coordination journal for humans and agents ("tried X, see commit abc; handing
back"). Not threaded: no replies, and no editing another actor's entries.
_Avoid_: Comment, Discussion
