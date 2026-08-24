---
name: handoff
description: Writes this session's state into a handoff file. A fresh session can read the file and continue the work.
argument-hint: "optional: the purpose of the next session"
disable-model-invocation: true
---

# Handoff

Write the handoff for a stranger. The next session has no memory of this conversation. The handoff file is the only input that the next session gets.

## Steps

1. **Write the handoff file.** The path is `${TMPDIR:-/tmp}/handoffs/{{repo-dirname}}/YYYY-MM-DD-{{slug}}.md`. Create the folder if it does not exist. Handoffs are disposable. For this reason they go in the OS temp folder, not in the repository. Adjust the content to the purpose that the argument gives. If there is no argument, write for the most probable continuation of this session's work. Use exactly these sections:
   - **Task and state** — the goal, the completed work, the work in progress, and the exact next action.
   - **Decisions and why** — only decisions that are recorded nowhere else. If a decision is in a spec, an ADR, an issue, or a commit, give the path or the reference. Do not copy the content. A copy drifts from its source.
   - **Pointers** — spec issue references, other issue references, the branch, and the key files.
   - **Gotchas** — the dead ends that this session explored, and why they failed. This prevents the next session from an exploration of the same dead ends.
   - **Commands** — exact commands: run the tests, start the application, and each command that this session had to discover.
   - **Suggested skills** — the skills that the next session must invoke, in sequence.

   Close the file with this line: *"This handoff is disposable — the temp folder cleans it up on its own."* Remove the secrets: keys, tokens, and personal data. This step is complete when a stranger can continue from the file alone, and the file has approximately 100 lines or fewer.

2. **Give the handoff to the user.** Show the full path. Show the first instruction for the next session, for example: `Read /tmp/handoffs/{{repo}}/{{file}}, then invoke the implement skill.` Tell the user that the system clears the temp folder at reboot. If a handoff must stay available longer, put it in a tracker note on the applicable issue. This step is complete when the user has the path and the first instruction.
