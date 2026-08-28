# Runner recipes — LOOP_AGENT_CMD

`LOOP_AGENT_CMD` is a shell command that the loop script runs one time for each iteration. The contract:

- The command runs **one full non-interactive agent session** in the current folder, and it exits when the session ends.
- The command receives the iteration prompt **on stdin**. Exception: if the command contains the placeholder `{PROMPT_FILE}`, the script substitutes the path of the prompt file, and it does not pipe.
- The command must never prompt a human. Anything interactive stalls until `LOOP_TIMEOUT` (default 3600 s) kills it, and the iteration counts as failed.

Run these recipes from inside `agentbox`, the machine-level wrapper from the `set-up-sandbox` skill. The loop and every iteration inherit its namespace, so confinement comes from the OS, not from per-repo config or the agents' own permission systems. That is also what makes the permissive flags below safe: inside the wrapper, an auto-approved command still cannot reach masked or read-only paths, and sudo is dead. Outside the wrapper, the same flags give each iteration full, unconfined access to the machine and its credentials — that posture is the user's to accept, never the default.

## Claude Code

```sh
LOOP_AGENT_CMD='claude -p --dangerously-skip-permissions --model <model> --effort <effort>'
```

Pin the model and effort. Claude Code supports `low`, `medium`, `high`, `xhigh`, and `max`, depending on the model. Print mode cannot prompt, so anything short of skip-permissions can fail closed mid-iteration on an unlisted command; inside agentbox the flag is the intended posture — the wrapper, not the agent's config, is the boundary. Run `claude --dangerously-skip-permissions` once interactively first to accept its one-time confirmation, or the first unattended session stalls on it.

## Pi

```sh
LOOP_AGENT_CMD='pi -p --approve --model <provider/id> --thinking <level>'
```

Pin the model and thinking level. Pi supports `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`, depending on the model. `--model` takes `provider/id` — `pi --list-models` lists what the machine is authenticated for. `--approve` trusts the project's local files for the run; print mode shows no trust prompt and no tool approvals. Core Pi has no sandbox or tool gating of its own — inside agentbox that is exactly right: the wrapper is the confinement. Sessions are saved by default, one full transcript for each iteration — read one later with `pi --resume`, or `pi --export <file>` for HTML.

## Any other agent

Any non-interactive agent CLI works — add your own recipe here. The wrapper confines it like any other process, so the recipe only has to hold up its side of the contract: run one full session unattended and exit. For a prompt file, use:

```sh
LOOP_AGENT_CMD='someagent run --auto --prompt-file {PROMPT_FILE}'
```

## The other knob

`LOOP_TIMEOUT` (default `3600`) is the number of seconds one iteration may take. An overrun kills the session, and the iteration counts as failed. A hung session is the one failure the run cannot otherwise recover from; everything else is a report line.

`STALL_LIMIT` is a constant at the top of `loop.sh` (default `3`): after that many iterations in a row without a closure, the run halts, because something systemic is wrong and more iterations only burn money. Edit the script to change it.

## Quality gate

The loop does not run tests. Backpressure belongs in the repository, where every agent meets it: pre-commit hooks for format and lint, and the checks the entry file (`AGENTS.md` / `CLAUDE.md`) tells sessions to run. The iteration prompt forbids `--no-verify`. The loop's own check is narrow and unspoofable — a `closed` report that did not move `HEAD` is recorded as failed.

## Cost ceiling

One session for each queued issue, no retries: the run costs exactly as many sessions as the queue is long. A failed issue is reported, not repeated — re-invoke the skill with it once you know why.

`loop.sh` needs Bash plus GNU `timeout` (macOS: `brew install coreutils`). Start it from inside the project's git repository, with a clean working tree, in an agentbox session.
