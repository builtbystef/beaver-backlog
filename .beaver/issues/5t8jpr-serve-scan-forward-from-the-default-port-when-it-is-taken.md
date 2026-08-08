---
id: 5t8jpr
title: 'serve: scan forward from the default port when it is taken'
state: done
assignee: claude
priority: medium
labels:
    - bug
created: 2026-08-08T03:41:39Z
updated: 2026-08-08T03:43:18Z
---

A second 'beaver serve' in another project fails with 'address already in use' because the default port 2328 is fixed. When the user did not pass --port explicitly, scan forward from 2328 to the next free port so serves for multiple projects coexist without anyone choosing ports. An explicit --port keeps failing outright.

## Notes

**claude** — 2026-08-08T03:43:18Z

Implemented in internal/cli/serve.go: listenLoopback scans forward from 2328 (bounded by portScanLimit=10) only when --port was not given; an explicit --port still fails outright. CLI tests cover both paths; verified live alongside a running serve.
