# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`hecato` is a single-binary filesystem investigation CLI (named for the Hecatoncheires). It walks a target path and reports the top N files by some ordering. Standard library only — `go.mod` has no requires.

## Commands

```bash
go build -o hecato.exe cmd/hecato/main.go   # local build
make all                                     # cross-compile linux + windows (what CI runs)
make test                                    # go test ./...
go vet ./...
go run cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -verbose=true
```

There are currently **no test files anywhere in the repo**, so `make test` passes vacuously. When adding the first test, `go test ./internal/files -run TestName -v` targets a single test.

Releases fire from `.github/workflows/build_release.yml` on a `v*.*.*` tag push — `git tag v1.0.0 && git push origin v1.0.0` cuts a public GitHub release. There is no CI on ordinary pushes or PRs.

## Architecture

Three layers, and adding a capability touches all of them:

- `cmd/hecato/main.go` — flag parsing plus a `switch *method` dispatch in `doWork()`. This switch is the only routing; there is no command registry.
- `internal/files` — the analysis methods. `getFiles()` (unexported, `getFiles.go`) is the shared engine: one `filepath.Walk` returning `(foundFiles, errorFiles, err)`.
- `internal/heclog`, `internal/examples`, `internal/version` — support packages.

**The method pattern.** `GetLargeFiles` and `GetModFiles` are the same function with a different comparator: parse `hits` with `Atoi`, call `getFiles(target)`, sort the whole slice, truncate to `hits`. Everything is walked and sorted in memory before truncation, so `-target=c:/` on a large volume holds every file in the slice at once. A new method means a new file in `internal/files` following that shape, a new `case` in `doWork()`, a line in `internal/examples/examples.go`, and an update to the `-method` flag help text.

**Walk errors are values, not failures.** The `filepath.Walk` callback appends unreadable paths to `errorFiles` and returns `nil` rather than aborting — deliberate, so a permission-denied directory doesn't kill a scan of `c:/`. Callers surface `errorFiles` only when verbose. Preserve this when touching the walk.

**Logging is dual-output.** `heclog.LogMessage(logToConsole bool, ...)` always writes to `app.log` and echoes to stdout only when the first arg is true. Call sites pass either a literal `true` (always shown) or the `logToConsoleVerbose` package var in `main.go`, which is assigned from `-verbose` at the end of `initFlags()`. Anything logged before that assignment uses the `true` default.

## Known rough edges

Present in the code as written — don't treat them as bugs to fix unless asked, but don't be surprised by them either:

- `buildContext` is a plain `var` in `main.go` hardcoded to `"development"`, which makes `heclog.LogSetup` write to the literal path `c:/repos/hecato/app.log`. Nothing in the Makefile or CI overrides it, so released binaries also log there rather than beside the executable.
- `internal/version` exposes ldflags-injectable `Version`/`BuildTime`/`Commit`, but no build path injects them — `-method=version` always prints `dev`/`unknown`/`none`.
- The `-version` and `-log` flags are defined and printed but never consulted. Version is reachable only via `-method=version`; file logging is unconditional.
- The `-method` help string advertises `largedirs`, which does not exist, and omits `modfiles`, which does.
- `hits` is threaded through as a `string` and converted inside each `Get*` function.

## Roadmap

Planned work lives in the header comment block of `cmd/hecato/main.go` (ignore-file support, extension search, `searchfile`, `largedirs`, directory churn counts). Check there before proposing new methods. The README also notes an intent to eventually use concurrency in the walk — it is fully sequential today.
