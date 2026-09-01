---
title: Installation
description: Install Beaver Backlog with a Go toolchain, or build it from a clone.
---

## With a Go toolchain

If you already have Go 1.26 or later:

```sh
go install github.com/builtbystef/beaver-backlog/cmd/beaver@latest
```

Or build from a clone:

```sh
git clone https://github.com/builtbystef/beaver-backlog.git
cd beaver-backlog
go build ./cmd/beaver
```

These builds report their version as `dev`, since the version is stamped in at
release time.
