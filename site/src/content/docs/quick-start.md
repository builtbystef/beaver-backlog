---
title: Quick start
description: Initialize a store, create an issue, and walk it through to done.
---

A first session with Beaver Backlog: initialize a store, create an [issue](/issue-file/), list it, claim it, note what you found, update a field, and close it. Install the binary first if you have not already; see [Installation](/installation/).

The output below is what the CLI prints on a terminal. When stdout is not a terminal, the same commands emit JSON instead; see [Command reference](/command-reference/#output-format).

## Initialize a store

From the root of your project:

```console
$ beaver init
Initialized empty Beaver Backlog store in /home/you/project/.beaver
```

This creates `.beaver/` in the current project. Commit it with your code; the files are the store.

## Create an issue

```console
$ beaver create "Login form rejects valid passwords" --label bug --priority high
Created ix2guj  Login form rejects valid passwords
  .beaver/issues/ix2guj-login-form-rejects-valid-passwords.md
```

The issue starts in `todo`. The file name is `<id>-<slug>.md`; `ix2guj` is the ID, and the slug mirrors the title.

## List issues

```console
$ beaver list
ID      PRIORITY  STATE  ASSIGNEE  LABELS  TITLE
ix2guj  high      todo   -         bug     Login form rejects valid passwords
```

`beaver list --ready` selects what is actionable now: issues in `todo` whose every dependency is `done`. This one has no dependencies, so it is ready.

## Claim it and start work

```console
$ beaver start ix2guj
Started ix2guj (claimed for stefan)
```

`start` moves the issue to `in-progress` and [claims](/command-reference/#start) it for the current actor if it has no assignee. A claim is a coordination signal: it records who is working, and it does not change who *may* work. The actor here is `stefan`; identity comes from `--as`, then `BEAVER_BACKLOG_ACTOR`, then per-machine config.

## Add a note

```console
$ beaver note ix2guj "root cause: form strips ! before hashing"
Added note to ix2guj as stefan
```

A [note](/issue-file/#notes) is an attributed, timestamped entry on the issue, appended to the file. Use it as a coordination journal: what you tried, what you found, what you are handing back.

## Update a field

```console
$ beaver update ix2guj --priority urgent --label regression
Updated ix2guj
```

`update` changes any field that is not state: title, description, assignee, priority, labels, relationships. It takes as many flags as you like in one invocation, and writes nothing if they net out to no change.

## Close it

```console
$ beaver done ix2guj
Marked ix2guj done
```

State changes are verbs of their own (`start`, `done`, `cancel`, `reopen`). Every other field changes through `update`.

## What to read next

- [Command reference](/command-reference/): every command, its flags, refs, output format, and exit codes
- [The issue file](/issue-file/): the on-disk shape, and the rules for a safe hand edit
