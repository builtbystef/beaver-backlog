---
name: create-issues
description: Breaks a spec issue into tracer-bullet sub-issues with blocking edges. Each sub-issue has the size of one agent session.
argument-hint: "a spec issue ref; no argument uses the session context"
disable-model-invocation: true
---

# Create Issues

Turn a spec into issues that an agent can take cold and complete — **tracer bullets**. Publication goes through the tracker doc, `docs/TRACKER.md`.

If the `docs/` folder or `docs/TRACKER.md` is missing, stop. Ask the user to invoke the `set-up-for-agents` skill.

## Steps

1. **Gather.** Fetch the spec issue: the reference in the argument, or else the spec issue that this session published. If neither exists, stop, and point the user to the `create-specification` skill. Explore the code that the spec touches, if you did not do this before. This step is complete when you can state each requirement without a second read of the spec.

2. **Slice.** Write the draft of the issue set. Follow these rules:
   - Each slice cuts a narrow but **complete** path through each layer that it needs (schema, logic, API, UI, tests). A slice is vertical, never one layer wide.
   - A completed slice is demoable or verifiable alone. Its size is one fresh agent session.
   - An acceptance criterion that verifies computed behavior includes a worked example: concrete input → expected output. The example is the independent source of truth that the tests assert against.
   - If the spec leaves a question open, and only an investigation or code that runs can settle it, make a `research` issue or a prototype issue. That issue blocks the slices that need its answer. The question is then settled in the tracker, not discovered during implementation.
   - Prefactor slices come first: slices that make the change easy come before slices that make the change.
   - Give each slice its blocking edges. Add only the issues that truly gate it.
   - If the user must approve a slice's result before its dependents unblock, add a body line that says: closure waits for user review. The session that implements the slice then applies `needs-review`, and does not close. (The semantics are in the tracker doc's label list.) The default is no gate.
   - **Wide refactors are the exception** to vertical slices. One mechanical change with a blast radius across the codebase (a column rename, a retype of a shared symbol) gets the **expand–contract** sequence: an *expand* slice adds the new form parallel to the old form; *migrate* slices move the call sites in batches (one batch for each package or folder), each blocked by the expand slice; a *contract* slice deletes the old form, blocked by each migrate slice.

   This step is complete when each spec requirement lands in the acceptance criteria of exactly one slice.

3. **Confirm with the user.** Show the slices as a numbered list: title, blocked-by, and what the slice delivers. Ask these questions: Is the granularity correct? Are the blocking edges real? Must some slices merge or divide? Iterate. This step is complete when the user approves the slice set.

4. **Publish.** Publish the sub-issues in dependency order — blockers first, so that blocking edges point to real references. Each sub-issue has the spec issue as parent, a priority, its blocking edges, and the body shape in [assets/templates/sub-issue.md](assets/templates/sub-issue.md).

   The spec issue is the umbrella. Its `spec` label keeps it out of the build queue. The `implement` skill builds its sub-issues, never the spec issue. Bodies contain behavior and criteria, never file paths or code snippets. Exception: a snippet from a prototype that records a decision more precisely than prose (a state machine, a schema, a type shape), cut to the part that shows the decision. This step is complete when these conditions are true: each approved slice is published with its edges; the slices without blockers appear when you list ready work; and you told the user to invoke the `implement` skill for each ready issue, one issue for each session.
