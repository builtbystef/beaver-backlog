---
name: create-plan
description: "Interviews the user until a goal is fully planned and ready for a specification. If the goal is too large for one session, creates a roadmap: a DAG of grill, research, prototype, and task sessions. The advance-plan skill then works the roadmap, one node for each session."
argument-hint: "a goal to plan"
disable-model-invocation: true
---

# Create Plan

The user has a goal, but is not sure how to move forward. Interview the user, and develop a plan together. Find facts yourself when they are needed. The user makes the decisions. Explore the full decision space. Identify the choices that matter. Then decide honestly: can the remaining planning work occur in this session, or must it be divided? A planning session decides. It never implements.

If the `docs/` folder or `docs/TRACKER.md` is missing, stop. Ask the user to invoke the `set-up-for-agents` skill.

## Steps

1. **Name the goal.** Write one or two sentences: what "done" looks like, from the user's point of view. This step is complete when the user agrees with your statement of the goal. Each later question relates back to this goal.

2. **Scout before you ask.** Find facts yourself. The user makes the decisions. Never ask the user a question that the repository can answer. Send read-only sub-agents to map the applicable code: what exists today in the goal's area, its seams, and its tests. Read `docs/GLOSSARY.md`, `docs/ARCHITECTURE.md`, and the ADRs in the area yourself. They are short, and they feed the interview directly. Delegation of the code walk matters here more than anywhere. The plan lives in this session's conversation, so keep the context for decisions and findings, not for file contents. If the host has no sub-agents, walk the code yourself, with the same economy: carry forward findings, not file contents. This step is complete when you know what exists today in the goal's area.

3. **List the open decisions.** Cover the full decision space widely. Do not go deep on one thread. Include: scope edges, data shapes, interfaces and seams, new dependencies, the call paths that the change alters at module boundaries, failure modes, migration, and UX. Then tell the user which case applies, and why:
   - **Plannable here** — the full set of decisions, plus the code that they touch, fits in this session (approximately 200k tokens). No decision waits on outside knowledge, a prototype, or work that must occur first. → Continue to step 4.
   - **Too large** — the goal becomes a **roadmap**: a DAG of nodes in the tracker. Each node has the size of one question and one fresh session. Read [`../advance-plan/SKILL.md`](../advance-plan/SKILL.md), and follow its "Create the roadmap" section. The sharp decisions become nodes. The remainder goes in the Frontier. Then stop.

   This step is complete when you told the user which case applies, and why.

4. **Interview.** Read [`../grill-me/SKILL.md`](../grill-me/SKILL.md), and follow it. Limit the interview to the goal and to the open decisions from step 3. This step is complete when no decision remains that would make an implementer stop and ask.

5. **Confirm, then close.** Before you declare the interview complete, show the user the decisions made, one line for each decision. Ask what is missing. Apply the answers. If a real gap appears, do step 4 again. Only then tell the user that the interview is complete. The plan lives in this session's conversation, so do not repeat the full plan. Suggest invoking the `create-specification` skill to record the plan while the context is fresh. This step is complete when the user confirmed that nothing is missing, and knows the next action.
