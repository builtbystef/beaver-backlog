---
name: maintain-codebase
description: "Audits the codebase for maintainability drift: test gaps, shallow modules, tight coupling, dead weight, doc rot, and code smells. Files the findings as issues."
argument-hint: "optional: the area to audit"
disable-model-invocation: true
---

# Maintain Codebase

This session audits the codebase and files issues. The fixes then flow through the `implement` skill, like other work. An audit that also attempts its own refactors does both tasks badly.

If the `docs/` folder or `docs/TRACKER.md` is missing, stop. Ask the user to invoke the `set-up-for-agents` skill.

Audit vocabulary: a **module** is anything with an interface and an implementation — a function, a class, or a package. A module is **deep** when a small interface hides much behavior. A module is **shallow** when the interface is almost as complex as what it hides. The **deletion test** shows shallowness. Imagine that you delete the module. If the complexity disappears, the module was a pass-through. If the complexity appears again across the callers, the module was useful.

## Steps

1. **Set the scope before you look.** If the argument gives an area, use that area. If not, give weight to recently changed areas:
   - Find the time when the last audit filed its findings: the newest issue with the label `maintenance` in the tracker.
   - Run `git log --since={{that time}} --oneline`. If no such issue exists, use a recent slice of history: `git log --oneline -200`.
   - Give the attention to the files and areas that occur again and again. Depth pays off where change occurs. Code that nobody touches can stay shallow.

   This step is complete when you told the user the scope.

2. **Sweep with parallel sub-agents.** Start one read-only sub-agent for each of the codebase dimensions below, except tracker health. Start all of them in one message. Each brief contains: the scope from step 1, the full audit vocabulary above (a sub-agent can read the repository, but not this skill), and its dimension — for the code smells dimension, the full smell baseline. Each sub-agent returns findings with evidence: file:line, and the observed symptom. The dimensions' file reads stay out of this session's context. This session judges the findings:
   - **Test gaps** — seams where critical or complex behavior crosses, with no test on them.
   - **Shallow modules** — apply the deletion test to thin modules. Pass-throughs and one-call wrappers are candidates: make them inline, or make them deep.
   - **Tight coupling** — one-concept changes that spread across modules; knowledge that leaks across seams; modules that you cannot test without their neighbors.
   - **Dead weight** — unreachable code, unused dependencies, old feature flags, and commented-out blocks.
   - **Doc rot** — statements in `docs/GLOSSARY.md`, `docs/ARCHITECTURE.md`, or `docs/CODING_STANDARDS.md` that the current code contradicts.
   - **Code smells** — occurrences of the smell baseline below. A documented repository standard overrides the baseline. Skip what a linter or formatter already enforces. Smells are judgement calls, never hard violations.
   - **Guardrail drift** — suppressions that accumulate (`eslint-disable`, `# type: ignore`, skipped tests); check commands in the entry file that do not run, or that do not pass; strictness that the project outgrew — lax type settings with an unresolved ratchet issue, a repository that gained collaborators or a remote but still has no CI, layer rules that live only in `docs/ARCHITECTURE.md` while violations occur again and again. An outgrown guardrail becomes an issue that names the specific gap, and the tool or check that closes it.
   - **Tracker health** — for each open issue, ask: can a fresh session take this issue cold, and complete it against the current code? If not, rewrite the issue, or close it with the note "closed in triage — refile if still wanted". When you are not sure, close it. A real need comes back and gets a new issue with current context. Before you close the only record of a decision, move the decision to `docs/`. Release old claims: an issue that is still claimed, with no note or commit since before the last audit, is the remainder of a dead session. Release it, and write a note that tells why. Also run the health check that the tracker doc describes, if there is one. Do this dimension yourself while the sub-agents sweep. It needs the tracker, not the codebase walk.

   Smell baseline (Fowler) — *what it is → the repair*:
   - **Mysterious Name** — a name that hides what the item does or contains → rename it; if no honest name exists, the design itself is unclear.
   - **Duplicated Code** — the same logic shape in more than one place → extract it, and call it from both places.
   - **Feature Envy** — a function that touches another module's data more than its own data → move the function to the data.
   - **Data Clumps** — the same few fields that travel together → make them one type.
   - **Primitive Obsession** — a primitive that represents a domain concept → give the concept its own small type.
   - **Repeated Switches** — the same conditional cascade on the same type, in many places → polymorphism, or one shared map.
   - **Shotgun Surgery** — one logical change that forces edits across many files → collect what changes together.
   - **Divergent Change** — one module that changes for several unrelated reasons → divide it, so that each part changes for one reason.
   - **Speculative Generality** — abstraction or hooks for needs that do not exist → delete them, until a real need shows.
   - **Message Chains** — long navigation, such as `a.b().c().d()` → hide the walk behind one method.
   - **Middle Man** — a layer that mostly delegates onward → remove it, and call the target directly.
   - **Refused Bequest** — an implementer that ignores most of what it inherits → remove the inheritance, and compose instead.

   If the host has no sub-agents, do the same sweep yourself, one dimension at a time. The parallel sweep is an economy, not part of the audit. Then remove duplicate findings. Two dimensions often find the same module. This step is complete when each dimension swept the full scope — not only the first finding in each dimension.

3. **Repair or file.** For mechanical findings with zero decisions (dead code, doc corrections): repair now. The entry file's checks (format, lint, typecheck, test) must be green afterward. For each finding with a design choice: publish an issue, with the label `maintenance` and a priority. The body follows the shape in [assets/templates/finding-issue.md](assets/templates/finding-issue.md). This step is complete when each finding from step 2 is repaired or filed.

4. **Report.** List the findings by dimension, the repairs made in this session, the issues published, and the one issue that you would work first, with the reason. This step is complete when the user can select what to schedule.
