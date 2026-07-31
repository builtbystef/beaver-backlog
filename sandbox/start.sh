#!/usr/bin/env bash
# Launches the beaver-backlog agent sandbox. THIS SCRIPT RUNS ON THE HOST.
# A sandboxed agent can edit this file, and edits execute on your host the
# next time you run it: review `git diff -- sandbox/` before every launch.
#
# Security flags (--cap-drop, --security-opt, the mount list) are the sandbox
# boundary — do not weaken them. The resource limits below them are host
# protection only; edit those freely.
set -euo pipefail

cd "$(dirname "$0")/.."
REPO="$(pwd)"
NAME="sandbox-beaver-backlog"

tty_flags="-i"
[ -t 0 ] && tty_flags="-it"

exec podman run \
  --rm $tty_flags \
  --name "$NAME" \
  --cap-drop=all \
  --security-opt=no-new-privileges \
  --dns 1.1.1.1 \
  --pids-limit=2048 \
  --memory=4g \
  --cpus=2 \
  --volume "$REPO:/workspace" \
  --volume "$NAME-home:/root" \
  --env "GIT_AUTHOR_NAME=$(git config user.name)" \
  --env "GIT_AUTHOR_EMAIL=$(git config user.email)" \
  --env "GIT_COMMITTER_NAME=$(git config user.name)" \
  --env "GIT_COMMITTER_EMAIL=$(git config user.email)" \
  --workdir /workspace \
  "$NAME" \
  "${@:-bash}"
