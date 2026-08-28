---
id: kajzu1
title: The project's name in the committed config, offered at init
state: todo
priority: medium
depends_on:
    - carvk9
created: 2026-08-28T05:46:32Z
updated: 2026-08-28T05:46:32Z
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

- [ ] The project config carries an optional name, and the project's name is that name where set and the store directory's name otherwise.
- [ ] A config without the key, and a store with no config file at all, both still yield the directory's name — no store needs migrating.
- [ ] An interactive `beaver init` offers the directory's name as the default and writes what is accepted into the committed config.
- [ ] Accepting nothing at that prompt leaves the store named after its directory and the key absent from the config.
- [ ] A non-interactive `init` never prompts and never writes a name.
- [ ] Re-running `init` on a store whose config already names it leaves that name alone.
- [ ] `init --format json` reports the project's name.
- [ ] A name set in the config is the name the web UI's sidebar and page titles show.
- [ ] The README's Configuration section no longer says the config records the format version and nothing else.

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
