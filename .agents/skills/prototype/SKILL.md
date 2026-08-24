---
name: prototype
description: Answers a design question with disposable code, before a real implementation starts. Use to examine whether a logic or state model is correct, what a UI must look like, or whether something is feasible or fast enough.
argument-hint: "the question, or an issue ref"
---

# Prototype

A prototype is disposable code that answers a question. The question decides the shape. The verdict outlives the code. The skill operates with or without a tracker. If `docs/TRACKER.md` exists, issue intake and closure resolve through the tracker. If not, skip the issue parts and work with docs only.

## Steps

1. **Pin the question, and select the shape.** If the argument is an issue reference, fetch the issue and claim it before you start work. The issue body is the brief. Then select the shape:
   - *"Is this logic or state model correct?"* → one self-contained HTML page, with the CSS and the JavaScript inline. Buttons and controls push the state machine through the cases that are hard to examine on paper.
   - *"What must this look like?"* → several very different variations of the UI on one route. The user switches between them in the browser, without a rebuild.
   - Neither (pure feasibility: "is X fast enough?", "does library Y do Z?") → the smallest script or benchmark that produces the number or the behavior that the verdict needs. The same rules below apply.

   A wrong shape wastes the full prototype. Confirm the question and the shape with the user. This step is complete when the user agrees.

2. **Build it quickly. Mark it as disposable.**
   - Put the prototype near the code that it examines. Give it a name that no reader can mistake for production code. UI routes follow the project's route convention.
   - Make it start with one step. A logic page opens directly in the browser: no build step, no server, no framework. For the other shapes, use the project's task runner. If there is none, use a plain `go run`, `node`, or `python` command, and state the command in your report.
   - Keep state in memory. Persistence is usually the object of the examination, not a dependency. If the question is about a database, use a scratch store with a clear "PROTOTYPE — wipe me" name.
   - Do not polish: no tests, no error handling beyond what keeps it runnable, no abstractions.
   - Show the state: print or render the full applicable state after each action or variant switch. Then the user sees exactly what changed.

   This step is complete when the user can start the prototype with one step and react to it.

3. **Iterate with the user** until the question has a verdict. The verdict is the user's words, not your inference. This step is complete when the user states the answer.

4. **Record, then remove.** Commit the prototype to a `prototype/{{slug}}` branch off the default branch. Then return to your initial branch, with the working tree as you found it. Record the verdict where the question came from:
   - From an issue → write a note with the verdict and the `prototype/{{slug}}` branch pointer. Then close the issue.
   - No issue, but the tracker exists → publish an issue. The title is the question. The label is `research`. The body is the question, the verdict, and the branch pointer. Then close the issue.
   - No tracker → write a short `docs/research/{{slug}}.md` with the same content. If the slug is taken, select a fresh slug. Never overwrite a file.

   The default branch keeps only the validated decision. This step is complete when the verdict is recorded in a location that outlives the code.
