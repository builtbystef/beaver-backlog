---
name: implement-loop
description: Drives a set of tracker issues to done through a loop. Each issue gets one fresh agent session. The tracker and git are the only memory between iterations. Use to implement explicit issue refs or a spec issue's sub-issues unattended, or to triage ready work into an ordered run first.
argument-hint: "issue refs, or a spec issue ref; no argument starts triage"
disable-model-invocation: true
---

# Implement Loop

One run, many issues. A fresh session completes each issue. This session is the **operator**: it assembles the queue, starts [scripts/loop.sh](scripts/loop.sh), watches the run, and reports. Each iteration is a disposable session that follows the project's `implement` skill on exactly one issue. The loop stays thin, because `implement` already carries the discipline: claim, build with tests first, escalate on decisions, close or release. The tracker and the git history are the only memory between iterations. The run folder is operator state, not a record.

The loop does not run tests. Backpressure belongs in the repository — pre-commit hooks, and the checks the entry file tells every session to run. The loop enforces only what a worker cannot self-report around: a `closed` moved HEAD, the tree came back clean, and the branch did not drift.

If the `docs/` folder or `docs/TRACKER.md` is missing, stop. Ask the user to invoke the `set-up-for-agents` skill. The `implement` skill must also be installed in this project. If it is not, stop and say so.

The recommended place for a loop is inside `agentbox`, the machine-level OS sandbox from the `set-up-sandbox` skill. The wrapper covers the whole process tree, so the script and every iteration inherit it automatically: private paths and credentials stay masked, the system stays untouchable, and the permissive flags the recipes use become safe to run unattended. Confirm the session is inside the wrapper before starting — a masked path (such as `~/Main` on this skill's home machine) lists as empty. If the machine has no agentbox, suggest running the `set-up-sandbox` skill first. A run outside the wrapper is the user's decision. Do not block on it. But say plainly that each iteration then has full, unconfined access to the machine and its credentials. The loop's runner recipes cover Claude Code and Pi; for any other agent, the developer adds their own recipe in [references/runners.md](references/runners.md), following the note there.

While the loop runs, the loop owns the working tree. Do not edit files. Do not claim issues. Do not commit. A dirty tree leaks into the next iteration's commit.

## Steps

1. **Assemble the queue.** Three intake forms:
   - **Issue refs** — fetch each one. Drop the closed issues. Flag the issues that are claimed or that have the label `needs-review`, and ask whether to include them.
   - **A spec issue** (label `spec`) — its open sub-issues are the queue. The spec itself never enters the queue.
   - **Nothing** — triage: list the ready work; show it numbered, with priority and blocking edges; recommend the issues for the run, and their order; iterate with the user. Gaps in the backlog are not yours to fill here. Point to the `create-issues` or `maintain-codebase` skill instead.

   Order the queue blockers first, then by priority. An issue must come after everything that blocks it — the run never reorders itself, so this ordering is the only one it gets. Show the final ordered queue in one list. This step is complete when the user approves the queue.

2. **Preflight.** Four things:
   - The working tree must be clean. A dirty tree means stop. The commit or the stash is the user's move. The branch also: the run puts commits on the current branch, so suggest a branch or a worktree now, if the user wants isolation.
   - Find the project's `implement` skill file (for example `.claude/skills/implement/SKILL.md` or `.agents/skills/implement/SKILL.md`). Iterations read it by path.
   - Settle the runner command with the user. Read [references/runners.md](references/runners.md), and propose the default unattended recipe. The cost, and (outside agentbox) the permission posture, are the user's to accept. For this reason, say plainly what the recipe lets sessions do.
   - Create the run folder `.implement-loop/{{UTC-timestamp}}/`. Make sure that `.implement-loop/` is in `.gitignore` (append it if it is missing — that one edit is permitted before the start). Write `queue.txt`, one ref for each line, in the approved order. Copy [assets/templates/prompt.md](assets/templates/prompt.md) into the run folder as `prompt.md`, and fill `{{REPORT_FILE}}` (the absolute path of the run's `report.log`) and `{{IMPLEMENT_SKILL_PATH}}` (the absolute path found above). Do not touch `{{REF}}`. The script fills it for each iteration.

   This step is complete when the run folder holds `queue.txt` and `prompt.md`.

3. **Start.** Start the script detached, so that the run survives this session:

   ```sh
   LOOP_AGENT_CMD='<runner command>' nohup bash <skill-dir>/scripts/loop.sh <run-dir> >> <run-dir>/nohup.out 2>&1 &
   ```

   `LOOP_TIMEOUT` (seconds for each iteration, default 3600 — a stuck session counts as failed) is the only other knob. This step is complete when the script runs, and `run.log` shows the first iteration.

4. **Operate.** Watch `run.log` and `report.log`. Tell the user what completes, when it completes. The protocol that the script runs: one session for each ref, in queue order, no retries. Each session ends with one outcome line in `report.log` — `closed`, `needs-review`, `blocked-by <ref>`, or `failed`, with a reason. Only `closed` counts as progress, and a `closed` that did not move HEAD is recorded as failed. Everything else — including silence and a timeout — leaves that issue for a human and moves on. After three iterations in a row without a closure, the full run stops: at that point something systemic is wrong, and more iterations burn money. The script also enforces two invariants between iterations, and it stops on both: the working tree must come back clean, and HEAD must stay on the start branch. Work past a violation is how entropy compounds. If the user leaves during the run, the run continues. A later session finds everything in the newest `.implement-loop/` folder. This step is complete when the script has exited — `status` in the run folder reads `done`, `halted-stall`, `halted-dirty`, or `halted-branch`.

5. **Report.** Trust, then verify. Before you compose the report, compare the report lines with the tracker: each `closed` issue is closed, and each `needs-review` issue has the label. A mismatch is a finding to show, not a line to repeat. Then compose the end-of-run report from `report.log` and the remainder of `queue.txt`: the closed issues; the issues that wait for the user (`needs-review` — say what each one needs; the note on the issue has it); the issues blocked, failed, or never run, and the blockers they named. `run.log` records HEAD before each iteration. For this reason, one iteration's exact diff is `head(N)..head(N+1)`. Use it when the user wants to inspect or reverse one issue's work. Then the next moves:
   - Suggest invoking the `review-code` skill over the run's full diff.
   - The queue came from a spec issue, and each sub-issue closed → write the run's outcome in a note on the spec issue. Tell the user that the spec is ready for review and closure. The loop never closes the spec itself.
   - Some issues stay open → to continue, invoke the `implement-loop` skill again with the remaining refs, and put any named blockers ahead of the issues that named them. The tracker's claims and notes make re-entry safe.

   This step is complete when the user knows the state of each queued issue, and the next move.
