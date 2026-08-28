---
id: kajzu1
title: The project's name in the committed config, offered at init
state: done
assignee: claude
priority: medium
depends_on:
    - carvk9
created: 2026-08-28T05:46:32Z
updated: 2026-08-28T06:29:25Z
---

## What to build

A project can say what it is called. `.beaver/config.yml` — committed and
shared like the issues, already carrying the store's format version — gains an
optional name, and the project's name reads that where it is set, falling back
to the store directory's name where it is not.

The name is committed on purpose: it is the project's name for everyone who
clones it, not a personal alias. Personal settings stay in the per-machine user
config, as ADR 0004 has it.

`beaver init` offers to set it, the way it already offers to establish an
identity: only when there is someone there to answer, with the directory's name
as the default, and accepting nothing leaves the key unset rather than writing
the default out. An agent or a CI run is never prompted and never has a name
written for it.

## Acceptance criteria

- [x] The project config carries an optional name, and the project's name is that name where set and the store directory's name otherwise.
- [x] A config without the key, and a store with no config file at all, both still yield the directory's name — no store needs migrating.
- [x] An interactive `beaver init` offers the directory's name as the default and writes what is accepted into the committed config.
- [x] Accepting nothing at that prompt leaves the store named after its directory and the key absent from the config.
- [x] A non-interactive `init` never prompts and never writes a name.
- [x] Re-running `init` on a store whose config already names it leaves that name alone.
- [x] `init --format json` reports the project's name.
- [x] A name set in the config is the name the web UI's sidebar and page titles show.
- [x] The README's Configuration section no longer says the config records the format version and nothing else.

## Notes for the builder

`seedIdentity` in `internal/cli/init.go` is the prompt to model: it runs only on
a TTY, only when nothing is saved, and never fails `init` — a declined or
unreadable prompt warns and leaves the store initialized. Two prompts on a first
run is acceptable; the project is asked about before the person is.

`store.Config` already ignores unknown keys so a config written by a newer
version loads in an older one, and `store.Init` already refuses to clobber an
existing config file. Both properties are load-bearing here and both deserve to
stay pinned.

The configuration guide in f08wtt covers documenting this for users; this issue
owes only the README's existing Configuration paragraph, which becomes wrong.

## Notes

**claude** — 2026-08-28T06:23:36Z

Built. `.beaver/config.yml` gains an optional `name:`, and the project's name is
that where set, the store directory's name otherwise.

Seams tested (the issue named none, so the outermost that can observe each
criterion): the store, for the derivation and the config write; the core API,
for naming a project and reading back what the config says; the CLI surface
through beavertest, for the prompt, the JSON report, and the non-interactive
silence; the web handlers over HTTP, for the sidebar and the tab.

Decisions:

- `Store.ProjectName()` stays a plain string. A config that cannot be read
  names nothing and the directory's name answers instead — a caller that wants
  a heading should not fail over a broken config, and `doctor` is where that is
  reported.
- `Store.SetProjectName` rewrites the file through the YAML node tree rather
  than re-serializing the struct, so the comments that make config.yml worth
  hand-editing survive, and so do keys a newer version wrote that this one does
  not know. The name is written as a tagged string, so a project called `2026`
  reads back as itself. An empty name is refused: no name means no key.
- Core gained `ConfiguredProjectName()` beside `ProjectName()` — the question
  init asks before offering, so an offer is only ever made to a project that has
  not answered it. The CLI still reaches nothing past the core API.
- The prompts now share one `*bufio.Reader`, passed into
  `establishHumanIdentity`. A buffered reader takes in more than the line it
  hands back, so two readers over the same stdin lost the second answer; with
  two prompts on a first run that was a real bug rather than a tidy-up. Verified
  over a real pty, not only in the harness.
- `init --format json` reports the name under `project`.

Facts for a reviewer: `TestInitSeedsIdentityInteractively` now answers two
prompts — an empty line for the project, then the identity — because the
project is asked about first. The web needed no production change; carvk9
already routed every mention through `ProjectName`, so the new web test pins a
criterion rather than driving new code. README's Configuration paragraph,
ARCHITECTURE's store line, and the glossary's Project name entry are current.
The user-facing configuration guide is still f08wtt's.

**claude** — 2026-08-28T06:29:25Z

Follow-up: the README's Configuration section was dropped entirely rather than reworded, so the config is documented for users only by f08wtt's guide. Comments across the change were shortened and their em dashes removed.
