# Actor identity is per-machine; `init` sets up the project, not the person

`beaver init` initializes a *project*: it creates `.beaver/issues/` and a small,
committed project config (a format-version marker, with room for project-wide
settings later). It is run once per repository by whoever adopts Busy Beaver, and it
is shared through version control like the issues themselves.

**Actor identity is separate and per-machine.** It lives in user-level config
(e.g. `~/.config/beaver/`) and is *never* committed. Storing identity in the
committed project config would make every contributor who clones the repo inherit
the initializer's identity — breaking teams and open-source projects. Decoupling
the personal thing (identity, per-machine) from the shared thing (the project,
committed) is what lets Busy Beaver serve solo devs, solo-dev-plus-agents, closed
teams, and unbounded open-source contributors with no per-contributor
registration. Free-form, untyped actors (see CONTEXT.md "Actor") mean an
unbounded contributor needs no roster entry — their identity seeds itself.

Identity is resolved per command, in precedence order:

1. `--as <actor>` flag.
2. `BUSY_BEAVER_ACTOR` environment variable (how agents and CI name themselves).
3. User-level config (the saved identity).
4. Otherwise, *interactively*: detect a VCS and, if present, read its identity
   (e.g. `git config user.name`) and ask the actor to confirm or replace it; with
   no VCS, prompt for a name. The result is saved to user-level config.
5. Otherwise, *non-interactively*: error, asking for `--as` / `BUSY_BEAVER_ACTOR`.

> **Refined by ADR 0010.** The chain above is superseded in detail by ADR 0010,
> which inserts environment-based agent detection ahead of the human-identity
> steps, gates the human/VCS identity (steps 3–4) to interactive sessions only,
> and replaces the step-5 error with a loud generic `agent` fallback.

`beaver init` proactively runs step 4 for the person running it (when they have
no identity yet), so the common solo case is "one command and you're ready."
Contributors who only ever clone reach the same setup lazily, on their first
ownership operation.

A VCS identity is therefore only ever adopted through an **interactive
confirmation**, never as a silent non-interactive fallback. This is deliberate: in
a repo whose Git user is the human owner, an agent that runs `claim` without
setting `BUSY_BEAVER_ACTOR` must *not* silently claim under the human's Git name — it
must error instead.

## Consequences

- The committed project config must never contain actor identity.
- Reading a VCS identity goes through the VCS adapter's identity operation
  (ADR 0007); with no adapter, step 4 has nothing to detect and falls to the
  prompt.
- The exact user-config location and format, and the project-config contents, are
  implementation details, deferred.
