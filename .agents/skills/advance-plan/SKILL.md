---
name: advance-plan
description: "Advances a roadmap by one node: resolves the next ready session in its DAG, then extends the DAG with what the answer made sharp."
argument-hint: "roadmap issue ref, or a specific node's ref"
disable-model-invocation: true
---

# Advance Plan

A roadmap is a goal too large for one planning session. The tracker holds it as a **DAG**. Each **node** is a sub-issue that asks one question, with the size of one fresh session. Each **edge** is a blocking relation: a node cannot start until each node that it depends on is closed. A node is **ready** when it is open, unclaimed, and no blocker of it is open. The DAG is the record. A closed node's note holds its answer, and the reason. Nothing duplicates the note elsewhere. Each invocation of the `advance-plan` skill resolves exactly one node. Then it extends the DAG with what the answer made sharp.

Each operation below resolves through the **Roadmap operations** section of the tracker doc (`docs/TRACKER.md`). If the doc is missing, stop, and ask the user to invoke the `set-up-for-agents` skill.

## The DAG

- **The root issue** — the title is `{{goal}} — roadmap`. The label is `roadmap`. The body comes from [assets/templates/roadmap-root.md](assets/templates/roadmap-root.md): the goal, the Frontier, and Out of scope.
- **Nodes** — the test for a node is whether the _question_ is sharp, not whether you can answer it. Each node: asks one question, with the size of one fresh session, with the question expanded in the body, plus pointers to what that session needs; sits under the root issue as its parent, with the group label `roadmap:{{roadmap-id}}` and a node-type label (below); has a blocking edge to each node that must resolve first. For an AFK node whose answer the user must approve before dependents unblock: add a body line that says that closure waits for user review (the `needs-review` semantics live in the tracker doc's label list). The default is no gate. HITL work never needs one.
- **The Frontier** — in-scope questions that you can see, but cannot phrase sharply yet. They stay as prose in the root issue's body, never as nodes. One vague question can become several nodes, or none, when the work comes closer to it.
- **Out of scope** — items excluded on purpose. The list only grows. One line for each item, with the node's reference if it was a node. An item never moves back in.

## Node types

Each type runs **HITL** — human in the loop, live with the user — or **AFK** — the agent alone. A HITL node resolves only through that live exchange. Never speak the user's side of it. A grill session that answers its own questions has broken this rule.

Where a type points to a skill, read the skill and follow it. The node's body is the brief.

- `session:research` (AFK) — a fact outside this session gates a decision → [`../research/SKILL.md`](../research/SKILL.md).
- `session:prototype` (HITL) — "how must it look or behave" gates a decision → [`../prototype/SKILL.md`](../prototype/SKILL.md).
- `session:grill` (HITL) — a decision that needs the user → [`../grill-me/SKILL.md`](../grill-me/SKILL.md), limited to the node's question.
- `session:task` (HITL or AFK) — work that must occur before a decision is possible. There is nothing to decide, research, or prototype, but the decision waits on the work. Examples: registration for a service, so that its API can be judged; access provision; a data move, so that the data's shape is visible. Do the work yourself where you can. Where you cannot, give the user a precise checklist. At closure, the note records the completed work, plus the facts that later nodes depend on: credential locations, new URLs, counts.
- `session:spec` (HITL) — the decisions of an **area** are all closed; write their spec. An area is a set of closed nodes whose answers an implementer needs together. A small roadmap is one area. A large roadmap divides along seams, where parts can be implemented independently of what is still open. Read the notes and linked artifacts of the area's closed nodes (only that area's, not the full roadmap's). Interview to close the remaining gaps ([`../grill-me/SKILL.md`](../grill-me/SKILL.md)). Confirm with the user. Then invoke the `create-specification` skill. The spec issue is published with a blocking edge back to each node that it covers, so that its lineage is walkable.

The user can also work a node directly in a fresh session by invoking the `research`, `prototype`, or `grill-me` skill with `{{node-ref}}`. Those sessions claim the node, write its note, and close it themselves.

## Create the roadmap

Do this one time, from the session that found the goal too large (create-plan step 3):

1. Create the root issue.
2. Add the nodes that you can state sharply now, in dependency order — blockers first, so that edges point to real references.
3. Put everything that is in scope, but not sharp yet, in the Frontier.
4. Tell the user to invoke the `advance-plan` skill with `{{root-ref}}` in each later session. Then stop. The creation of the roadmap is that session's full job.

## Advance

1. **Fetch the root** — the goal, the Frontier, and Out of scope; not each node. If an area settled without its `session:spec` node — a direct session closed its last blocker — add that node now, blocked on the area's nodes. This step is complete when you know the goal, and what is ready.

2. **Select one node**, in this order: the node that the user named; a ready `session:spec` node — specs come before other work, so that settled decisions are recorded before more planning stacks on top; the first other ready node. If no node is ready, the Frontier is this session's job: interview the user until the sharpest entry becomes nodes, connect them as in "Create the roadmap" step 2, and stop. Claim the node before any work. This step is complete when one node is claimed.

3. **Resolve the node** by its node type. This step is complete when the node's question has its answer — or its deliverable exists.

4. **Record the result.** Write a note on the node, with the answer and the reason. Close the node. If the body gates closure on user review: write the deliverable in the note, apply `needs-review`, and release the claim. Do not close. This step is complete when the node's note would let a stranger use its answer.

5. **Extend the DAG.** Frontier entries that the answer made sharp become nodes, connected as in "Create the roadmap" step 2. What the answer excluded gets a line under Out of scope. If the excluded item was a node, close that node, with a note that says why. If this closure settled an area, add its `session:spec` node, blocked on the area's nodes. When each node is closed and the Frontier is empty, close the root. The roadmap is then done. One node for each session — stop after the extension.
