---
name: implement
description: "Completes one issue from start to end: claim the issue, build with tests first, verify, record, and close."
argument-hint: "an issue ref; no argument takes the next ready issue"
disable-model-invocation: true
---

# Implement

Complete one issue in one session. Make all checks pass. Tracker operations resolve through the tracker doc, `docs/TRACKER.md`.

Four rules keep the session clean:

- Complete exactly one issue.
- If you discover new work, publish a new issue with a blocking edge. Do not do the work in this session.
- If the issue is blocked on missing work or a missing decision, write a note that tells why. Then release the claim and stop.
- If no issue is ready, tell the user and stop.

If the `docs/` folder or `docs/TRACKER.md` is missing, stop. Ask the user to invoke the `set-up-for-agents` skill.

## Steps

1. **Take the issue.** If the argument is an issue reference, fetch it and claim it. Without an argument, list the ready work and take the highest-priority issue. Skip issues labeled `roadmap`, `session:*`, and `spec` (only a spec's sub-issue slices are buildable). If no issue is ready, tell the user and stop — that is a correct outcome, not a failure.

2. **Load the contract.** Read the issue body and its notes; the parent spec issue, if there is one; `docs/CODING_STANDARDS.md` and `docs/GLOSSARY.md`; `docs/ARCHITECTURE.md` for the modules the issue touches; and the ADRs in that area — an ADR is a decision already made, do not reverse it.

   Escalate before writing code if a criterion is ambiguous, two criteria contradict each other, or a criterion contradicts the spec or an ADR. To escalate: write a note on the issue with the specific question and the options you see, ending with what the user must do — decide, record the decision in a note, and remove the label. Apply `needs-review`, release the claim, and stop. This step is complete when you can state the acceptance criteria and the test seams without invention.

3. **Build with tests first, where TDD is possible.** If the issue has acceptance criteria, read [`../test/SKILL.md`](../test/SKILL.md) and work red → green at the issue's seams, one slice at a time. If the work has no test scope (exploratory work, visual work, glue code), build it, then add tests at the seam the result shows.

   Run the typechecker and the changed test files while you work. At the end, run all four checks once: format, lint, typecheck, and the full test suite (commands in the Checks section of the entry file). This step is complete when the code satisfies each acceptance criterion and all checks pass.

4. **Stay inside the issue.** If you discover work the issue does not cover, publish a new issue for it — with a blocking edge or parent only where one truly exists — and do not do the work in this session. Two types of blockers stop a session:
   - **Missing work.** Other work must occur first. Write a note that names the blocker. Release the claim, report, and stop.
   - **Missing decision.** The work needs a dependency the spec does not name, an interface or module-boundary change goes beyond what the spec decided, or a criterion becomes ambiguous during the build. Escalate as in step 2.

   Escalate on decisions only, never on facts. If the repository, the docs, or the spec can answer a question, read them. This step is complete when the diff serves only this issue's criteria.

5. **Record and close.** Compare each acceptance criterion with the actual diff — the checks find broken code, not the wrong build. If a criterion is not satisfied, go back to step 3. Write a note on the issue: the completed work, the decisions made, and the facts a reviewer needs. Close the issue — unless the body says closure waits for user review; then end the note with what the user must do (close to approve, or note requested changes and remove the label), apply `needs-review`, release the claim, and do not close. Commit with the issue reference in the message; the tracker doc tells whether tracker state travels in the commit. This step is complete when the working tree is clean and the issue is closed or waits for review.
