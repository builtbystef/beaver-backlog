---
name: create-specification
description: Turns the planning context of the current session into a spec issue in the tracker.
argument-hint: "optional: a spec title, or a roadmap issue ref"
disable-model-invocation: true
---

# Create Specification

Write a specification from the decisions that this session already made. Do not invent new decisions. An implementer must be able to build from the spec without questions. The spec becomes an issue in the tracker. The issue body is the specification. The implementation slices become its sub-issues through the `create-issues` skill.

If the `docs/` folder or `docs/TRACKER.md` is missing, stop. Ask the user to invoke the `set-up-for-agents` skill.

## Steps

1. **Make sure that the context is sufficient.** Read the spec template ([assets/templates/spec.md](assets/templates/spec.md)). Examine each section. Make sure that you can fill it from this session's context. For a roadmap area, the context also includes the notes of the area's closed sub-issues. If you must invent content for a section, the context is not sufficient. Then read [`../grill-me/SKILL.md`](../grill-me/SKILL.md) and interview the user until the gaps close. Then start this step again. This step is complete when you can fill each section from real decisions.

2. **Connect the spec to the repository.** Explore the code that the spec touches, if you did not do this before. Use the vocabulary from `docs/GLOSSARY.md` in the full spec. Obey the ADRs in the area. Identify the seams where tests will attach to the feature. Prefer seams that exist. Put a new seam at the outermost interface that can observe the behavior. Use as few seams as possible. One seam is best. Propose the seams to the user. This step is complete when the user agrees with the seams.

3. **Write the draft** from the template. Do not include file paths or code snippets. They become incorrect quickly. There are two permitted exceptions:
   - Signatures, type shapes, and API contracts at the seams that the spec adds or changes. A signature is a decision. A written signature lets an implementer build without questions.
   - A snippet from a prototype that records a decision more precisely than prose. Examples: a state machine, a schema. Keep only the part that shows the decision. Give the source of the snippet.

   This step is complete when each section of the template has content.

4. **Review the draft with the user.** Show the draft to the user, section by section. Apply the user's corrections. This step is complete when the user approves the draft.

5. **Publish the spec.** Publish one issue. The title is the spec's title. The label is `spec`. The body is the approved spec. If the spec covers a roadmap area, record a blocking edge from the spec issue to each closed node that the spec covers. These edges record the source of the spec, so that its lineage is walkable. This step is complete when the issue exists and you suggested invoking the `create-issues` skill with `{{spec-ref}}` to the user.
