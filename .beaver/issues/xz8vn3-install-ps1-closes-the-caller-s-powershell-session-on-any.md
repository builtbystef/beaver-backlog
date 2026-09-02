---
id: xz8vn3
title: install.ps1 closes the caller's PowerShell session on any failure
state: done
priority: high
labels:
    - bug
created: 2026-09-02T08:22:08Z
updated: 2026-09-02T08:22:08Z
---

## Problem

The catch block ended with `exit 1`. Piped through `iex` as the documented
one-liner does, the script runs inside the caller's own session, so `exit`
closed the user's PowerShell window before the error could be read. A user on
Windows saw exactly that: the window vanished, nothing was installed, and the
underlying failure was never visible.

The same mechanism let `$ErrorActionPreference = 'Stop'`, `Set-StrictMode`, and
`$ProgressPreference` leak into the interactive session after a successful
install.

## Fix

The body runs in its own `& { }` scope, so preferences and strict mode do not
outlive the install, and the catch block rethrows instead of exiting. An
interactive shell stays open with the message on screen, and
`powershell -File install.ps1` still exits non-zero on failure.

The README and the Installation page note that a saved copy of the script is
subject to the default execution policy and how to run one.
