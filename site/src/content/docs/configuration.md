---
title: Configuration
description: What beaver init writes, why project config is committed, and where identity lives.
---

Beaver Backlog keeps two kinds of configuration, on purpose, so cloning a repository never makes a contributor inherit someone else's name.

## What `init` writes

`beaver init` creates a store in the current project. It is idempotent: running it again leaves an existing store in place.

```console
$ beaver init
Initialized empty Beaver Backlog store in /home/you/project/.beaver
```

The store is a `.beaver/` directory with an `issues/` folder and a project config:

```yaml
# Beaver Backlog project configuration.
# Committed and shared through version control, like the issues themselves.
# Safe to read and edit by hand.
format_version: 1
```

That is `.beaver/config.yml`. In an interactive session, `init` may also offer to set a [project name](/command-reference/#init) (written into this file) and a human identity (written elsewhere; see below). A non-interactive `init` prompts for neither.

## Project config is committed and shared

`.beaver/config.yml` travels with the repository, like the issues themselves. Commit it. A setting here is project-wide policy: the on-disk format version, and optionally the name the project is called.

Unknown keys are ignored on read, so a config written by a newer version still loads in an older one. Hand-editing it is safe; follow up with [`beaver doctor`](/doctor/).

The project's name, when unset, falls back to the name of the directory the store sits in, so every project has one without being configured.

## Identity is per-machine

Actor identity lives in per-machine user config and is **never in the repository**. A committed identity would make every cloner inherit the initializer's name. The file sits in the OS user-config directory (for example `~/.config/beaver/config.yml` on Linux), is established the first time an interactive session needs a name, and is reused on later interactive runs.

Agents do not use that file. They set `--as` or `BEAVER_BACKLOG_ACTOR`; see [Working with coding agents](/coding-agents/).

## No version-control system

Beaver Backlog never runs a version-control system itself. The files are the store; committing them, merging them, and pushing them is the operator's job. There is no sync layer, no remote, and no lock server. That is why a [claim is advisory](/coding-agents/#a-claim-is-not-a-lock) and why each concurrent agent needs its own working tree.

## See also

- [Command reference](/command-reference/#init): `init` and `whoami`
- [Working with coding agents](/coding-agents/): actor resolution
- [Architecture decisions](https://github.com/builtbystef/beaver-backlog/tree/main/docs/adr)
- [Contributing](https://github.com/builtbystef/beaver-backlog/blob/main/CONTRIBUTING.md)
