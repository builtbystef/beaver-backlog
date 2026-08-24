---
name: set-up-for-agents
description: Sets this repository up for coding agents. Creates the docs/ folder that the agents read, the tracker doc that they file work through, the entry files that they start from (CLAUDE.md and AGENTS.md), and the checks that they verify their work with.
argument-hint: "optional: what the project is"
disable-model-invocation: true
---

# Set Up For Agents

Prepare this repository for coding agents. The agents need the `docs/` folder — the project's shared knowledge. This is a conversation, not a script: explore, interview, confirm the decisions, then write the files.

## The setup, and the reasons

- **`docs/GLOSSARY.md`** — the project's shared language: one term for each domain concept, with the rejected synonyms under _Avoid_. Planning sessions test unclear words against it, and they update it as terms settle. Code, tests, specs, and issues all use its vocabulary. It is a glossary and nothing else: no implementation details, no spec content.
- **`docs/CODING_STANDARDS.md`** — the conventions that linters and formatters cannot enforce: error handling, names, module shape, test style.
- **`docs/ARCHITECTURE.md`** — the modules of the system, and the seams between them, present or intended.
- **`docs/adr/`** — the folder for architecture decision records. One ADR for each decision that is hard to reverse. Then future sessions do not debate it again, and they do not "repair" it by accident.
- **`docs/TRACKER.md`** — how to use this project's issue tracker. It maps the verbs that the skills speak — publish, fetch, claim, note, close, release, list ready work, record a blocking edge, defined in [the verb contract](references/tracker-verbs.md) — onto the concrete commands that operate here. Each skill that touches an issue reads this file.
- **The checks** — the project's format, lint, typecheck, and test commands, recorded in the entry file, so that each session runs them before it finishes. Where a tool is missing and the stack is known, set up the stack's standard tool with a minimal configuration.

## Steps

1. **Explore, then interview.** Read what exists: the argument, the README, the code, an existing `CLAUDE.md` or `AGENTS.md`, and an existing `docs/` (do not recreate what is there). Detect tracker candidates: existing tracker state (`.beaver/`, `scratch/`, issues on the git remote), `git remote -v`, installed CLIs (`beaver`, `gh`), and connected tracker tools (for example Linear). Detect the toolchain: formatter, linter, typechecker, and test-runner configurations; CI workflows; and whether `.gitignore` covers env files and secret files. Then ask the user for what the code and the docs cannot tell, for example the project's purpose, the tech stack, how tests run, the domain terms in use, and the standards that the team holds. This step is complete when you can state the project in two sentences, and the user approves them.

2. **Select the tracker.** Show what you detected. Give a recommendation first: the tracker that the project already shows evidence of, or else the one that its tools make cheapest. The options: Beaver Backlog, GitHub Issues, Linear, local Markdown, or freeform — the user describes their own workflow in a paragraph, and that paragraph becomes the tracker doc. This step is complete when the user selects one.

3. **Write the files.** With the decisions from steps 1–2 settled, create the `docs/` files — each from its template, as the **Seed templates** section below tells, `docs/TRACKER.md` included. Initialize what the selected tracker needs. This step is complete when the files exist, and the tracker doc's verbs operate here.

4. **Establish the checks.** Four checks: format, lint, typecheck, test. For each check:
   - A tool exists → record its command.
   - The tool is missing, and the stack is known → propose the stack's standard tool, confirm with the user, and add a minimal configuration.
   - No code or stack exists yet → publish one tracker issue — establish the checks when the stack lands — and continue.

   Start the type checks at the strictest settings that the current code passes. On a fresh project, use full strict, because this is the cheapest moment there will ever be. On an existing codebase, adopt what passes today, and publish a tracker issue for a later increase. Make sure that each recorded command runs green on the current tree (a test runner that fails on an empty suite gets one smoke test). Add env files and secret files to `.gitignore`, if they are missing. Where the project is runnable (dev server, CLI entry), also record the run command — how to start it locally. If the repository has a CI-capable remote and no workflow, offer a minimal workflow that runs these same commands on push. Existing CI stays as it is, but note which of the four checks it skips. This step is complete when each check has a command that passes, or a tracker issue.

5. **Connect the entry points.** Update each agent entry file that the project already has: `CLAUDE.md` for Claude Code, `AGENTS.md` for Codex and other agents. If none exists, ask the user which to create. This step is complete when a fresh agent that reads only its own entry file finds the docs, the tracker doc, the conventions, and the check commands.

## Entry-file block

Append [assets/templates/entry-file-block.md](assets/templates/entry-file-block.md) to each entry file. Resolve `{{Tracker name}}` to the selected tracker, and the `{{… command}}` placeholders to the commands from step 4. Remove each line whose doc or check the project does not have yet.

## Seed templates

Each file below starts from a template in [assets/templates/](assets/templates/). Copy the template. Then fill it with real content from steps 1–2 — however small the content. Where nothing is known yet, keep the template's header, which describes itself, and add nothing invented.

| Destination                | Template                                                    | Seed content                                                                 |
| -------------------------- | ----------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `docs/GLOSSARY.md`         | [GLOSSARY.md](assets/templates/GLOSSARY.md)                 | the terms the user used to describe the project, under the Language section  |
| `docs/CODING_STANDARDS.md` | [CODING_STANDARDS.md](assets/templates/CODING_STANDARDS.md) | the standards the user stated, grouped by area                               |
| `docs/ARCHITECTURE.md`     | [ARCHITECTURE.md](assets/templates/ARCHITECTURE.md)         | the modules named or intended so far                                         |
| `docs/adr/README.md`       | [adr-README.md](assets/templates/adr-README.md)             | — (copy without changes)                                                     |

`docs/TRACKER.md` — the selected file from [references/trackers/](references/trackers/), with the placeholders resolved (team, repository). For freeform: the user's workflow paragraph, under a `# Tracker: {{name}}` heading.

A second run of this skill on a grown project is safe and expected. It fills only what is missing, checks that the stack now supports, CI when a remote exists,  and it never recreates what is there.

When the setup is done, suggest the `set-up-sandbox` skill as the follow-up: it gives the repository OS-enforced sandbox config, so that agent sessions here can run autonomously — writes confined to the repository, the user's protected directories and credentials unreadable.
