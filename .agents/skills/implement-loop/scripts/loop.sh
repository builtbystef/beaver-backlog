#!/usr/bin/env bash
# Runner for the implement-loop skill: one fresh agent session per queued
# issue, sequentially. The tracker and git are the durable memory between
# iterations; the run directory holds operator state only.
#
# Usage:  loop.sh <run-dir>
#   <run-dir> must contain:
#     queue.txt   one issue ref per line, blockers-first order
#     prompt.md   iteration prompt with {{REF}} left unfilled
#
# Env:
#   LOOP_AGENT_CMD  required. Shell command that runs ONE non-interactive
#                   agent session in the current directory and exits when the
#                   session ends. Receives the prompt on stdin, or via a
#                   {PROMPT_FILE} placeholder substituted with the prompt's
#                   path. Recipes: references/runners.md.
#   LOOP_TIMEOUT    seconds per iteration (default 3600); overrun = failed.
#
# Protocol: each session appends one line to report.log —
#   <ref> closed|needs-review|failed -- <reason>
#   <ref> blocked-by <blocker-ref> -- <reason>
# Only 'closed' counts as progress; every other word (and silence, and a
# timeout) means a human handles that issue. A 'closed' that did not move HEAD
# is reclassified as failed — the run's one check the worker cannot self-report
# around. Real backpressure lives in the repository: pre-commit hooks and the
# checks the entry file names.
#
# Invariants enforced between iterations: the working tree must come back clean
# and HEAD must stay on the launch branch — a violation halts the run rather
# than letting entropy compound. HEAD is logged before every iteration, so
# iteration N's diff is exactly head(N)..head(N+1).
#
# Exit / status: 0 done · 2 halted-stall · 4 halted-dirty / halted-branch

set -u

STALL_LIMIT=3  # consecutive iterations without a close before the run halts

RUN_DIR=${1:?usage: loop.sh <run-dir>}
RUN_DIR=$(cd "$RUN_DIR" && pwd)
QUEUE="$RUN_DIR/queue.txt"
TEMPLATE="$RUN_DIR/prompt.md"
REPORT="$RUN_DIR/report.log"
LOG="$RUN_DIR/run.log"

AGENT_CMD=${LOOP_AGENT_CMD:?set LOOP_AGENT_CMD (see references/runners.md)}
TIMEOUT=${LOOP_TIMEOUT:-3600}

[ -s "$QUEUE" ] || { echo "queue.txt missing or empty in $RUN_DIR" >&2; exit 1; }
[ -f "$TEMPLATE" ] || { echo "prompt.md missing in $RUN_DIR" >&2; exit 1; }
grep -q '{{REF}}' "$TEMPLATE" || { echo "prompt.md has no {{REF}} placeholder" >&2; exit 1; }
command -v timeout >/dev/null || { echo "GNU 'timeout' not found (macOS: brew install coreutils)" >&2; exit 1; }
git rev-parse --show-toplevel >/dev/null 2>&1 || { echo "current directory is not a git repository" >&2; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "working tree not clean — commit or stash before launching" >&2; exit 1; }

BRANCH=$(git rev-parse --abbrev-ref HEAD)

touch "$REPORT"
log() { printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*" >>"$LOG"; }
halt() { log "halting: $2"; echo "$1" >"$RUN_DIR/status"; exit "$3"; }

echo running >"$RUN_DIR/status"
log "run start: $(wc -l <"$QUEUE" | tr -d ' ') queued on branch $BRANCH, timeout=${TIMEOUT}s"

it=0 stall=0

while [ -s "$QUEUE" ]; do
  ref=$(head -n1 "$QUEUE")
  tail -n +2 "$QUEUE" >"$QUEUE.tmp" && mv "$QUEUE.tmp" "$QUEUE"
  [ -n "$ref" ] || continue

  it=$((it + 1))
  head_before=$(git rev-parse HEAD)
  log "iteration $it: $ref (HEAD $(git rev-parse --short HEAD) on $BRANCH)"

  marker=$(wc -l <"$REPORT")
  prompt="$RUN_DIR/iteration-$it-prompt.md"
  template_body=$(cat "$TEMPLATE")
  printf '%s\n' "${template_body//'{{REF}}'/$ref}" >"$prompt"

  if [[ "$AGENT_CMD" == *'{PROMPT_FILE}'* ]]; then
    timeout -k 30 "$TIMEOUT" bash -c "${AGENT_CMD//'{PROMPT_FILE}'/$prompt}" \
      >"$RUN_DIR/iteration-$it.out" 2>&1
  else
    timeout -k 30 "$TIMEOUT" bash -c "$AGENT_CMD" <"$prompt" \
      >"$RUN_DIR/iteration-$it.out" 2>&1
  fi
  rc=$?
  [ "$rc" -eq 124 ] && log "iteration $it: timed out after ${TIMEOUT}s"

  # Invariants before anything else — a violation is systemic, not issue-specific.
  now_branch=$(git rev-parse --abbrev-ref HEAD)
  [ "$now_branch" = "$BRANCH" ] || halt halted-branch "branch drifted $BRANCH -> $now_branch during iteration $it ($ref)" 4
  [ -z "$(git status --porcelain)" ] || halt halted-dirty "dirty working tree after iteration $it ($ref) — see git status" 4

  line=$(tail -n +"$((marker + 1))" "$REPORT" | awk -v r="$ref" '$1 == r { l = $0 } END { print l }')
  status=$(printf '%s' "$line" | awk '{ print $2 }')
  log "iteration $it: reported '${line:-<no report line>}' (exit $rc)"

  if [ "$status" = closed ] && [ "$(git rev-parse HEAD)" = "$head_before" ]; then
    log "iteration $it: reported closed but HEAD did not move — counting as failed"
    status=failed-no-commit
  fi

  if [ "$status" = closed ]; then
    stall=0
  else
    stall=$((stall + 1))
    [ "$stall" -lt "$STALL_LIMIT" ] || halt halted-stall "$stall consecutive iterations without a close" 2
  fi
done

log "run done: queue drained after $it iterations, HEAD $(git rev-parse --short HEAD)"
echo done >"$RUN_DIR/status"
