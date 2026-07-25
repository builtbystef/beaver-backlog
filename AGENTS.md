# Beaver Backlog — agent guide

Beaver Backlog is a local-first markdown issue tracker. This repo dogfoods its own
tracker — project work is tracked in the `.beaver/` store right here.

## Checks

Run whichever check the change touches as you go, and all of them before ending a session that changed code — every one must pass:

- Format: `gofmt -l .` (must print nothing; fix with `gofmt -w .`)
- Lint: `go vet ./...`
- Typecheck: `go build ./...`
- Test: `go test ./...`

Run the app locally: `go run ./cmd/beaver <command>` (or `go build ./cmd/beaver` for a `./beaver` binary)

## Project docs & tracker

### Domain glossary

`docs/GLOSSARY.md` — the project's terms. Use its vocabulary in code, tests, specs, and issues; format rules at the top of the file.

### Coding standards

`docs/CODING_STANDARDS.md` — conventions beyond the linter. Reviews check diffs against it.

### Architecture & decisions

`docs/ARCHITECTURE.md` — the modules and seams. `docs/adr/` — decisions already made (format in `docs/adr/README.md`); don't re-litigate them.

### Issue tracker

`docs/TRACKER.md` — how to use Beaver Backlog.
