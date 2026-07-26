# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`hecato` is a single-binary filesystem investigation CLI (named for the Hecatoncheires). It walks a target path and reports the top N files by some ordering. One dependency, `gopkg.in/yaml.v3`, for the config file; everything else is standard library.

## Commands

```bash
go build -o hecato.exe cmd/hecato/main.go   # quick local build, no version injection
make all                                     # cross-compile linux + windows, dev logging
make release                                 # same, but BUILD_CONTEXT=release (what CI runs)
make test                                    # go test ./...
make vet
go run cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -verbose=true
```

`internal/ignore`, `internal/config` and `internal/files` have tests; the other packages do not. A single test: `go test ./internal/ignore -run TestMatchDir -v`.

`make all` and `make release` differ only in `BUILD_CONTEXT`, which is injected via ldflags and decides where `heclog` writes `app.log` — see Build metadata below. Use `make release` when you need a binary that behaves like a shipped one.

**`gofmt -l .` lists every file when run locally on Windows.** This is not real formatting drift: `core.autocrlf=true` and the repo has no `.gitattributes`, so blobs are LF but the working tree is CRLF, and gofmt objects to the CRs. CI checks out LF on Linux and passes. Don't "fix" the formatting in response to local gofmt output.

## CI

Two workflows:

- `.github/workflows/ci.yml` — gofmt check, vet, test, and `make all` on pushes to `main` and on PRs.
- `.github/workflows/build_release.yml` — on a `v*.*.*` tag push, builds via `make release` and publishes a GitHub Release with both binaries attached using `gh release create`. `workflow_dispatch` runs the same build and uploads artifacts but publishes nothing, so it is a safe dry run.

The release job needs `fetch-depth: 0` because the Makefile calls `git describe` for the version. Both workflows read the Go version from `go.mod` via `setup-go`'s `go-version-file`, so there is no separate version to keep in sync.

The binary filename is load-bearing: `BINARY_NAME` in the Makefile must stay `hecato`, since the workflow references `hecato` and `hecato.exe` by name for both artifact upload and release assets.

## Architecture

Three layers, and adding a capability touches all of them:

- `cmd/hecato/main.go` — flag parsing plus a `switch *method` dispatch in `doWork()`. This switch is the only routing; there is no command registry.
- `internal/files` — the analysis methods. `getFiles()` (unexported, `getFiles.go`) is the shared engine: one `filepath.Walk` returning `(foundFiles, errorFiles, err)`.
- `internal/config` — the optional YAML config; `internal/ignore` — the pattern matcher it feeds.
- `internal/heclog`, `internal/examples`, `internal/version` — support packages.

**The method pattern.** `GetLargeFiles` and `GetModFiles` are the same function with a different comparator: parse `hits` with `Atoi`, call `getFiles(target, ig)`, sort the whole slice, truncate to `hits`. Everything is walked and sorted in memory before truncation, so `-target=c:/` on a large volume holds every file in the slice at once. A new method means a new file in `internal/files` following that shape, a new `case` in `doWork()`, a line in `internal/examples/examples.go`, and an update to the `-method` flag help text.

**Config and ignore.** A config is always optional — `config.Load` returns a zero `Config` and no error when none is found, and hecato behaves as it did before configs existed. Resolution is `-config`, then `hecato.yaml` beside the executable, then nothing; an explicit `-config` that does not exist is fatal, as is a malformed or misspelled one (the decoder runs with `KnownFields(true)`). The one exception is `-method=initconfig`, which is allowed to be pointed at a path that does not exist yet — that is the whole point of it. `main.go` checks for that method *before* loading, against both `-method` and `-m`, since the method is not resolved until later.

`hecato.example.yaml` at the repo root is a committed copy of what `initconfig` writes. `TestExampleConfigMatchesGenerator` is the only thing keeping the two in sync — if you change `starter.go`, regenerate the example or that test fails with the command to do it. The live `hecato.yaml` is gitignored, since a local build puts the binary at the repo root and would otherwise pick it up and show it dirty.

Flag precedence is **explicit flag > config > built-in default**, implemented with `flag.Visit` in `setFlags()` — `flag.Visit` walks only flags actually typed, which is the only way to tell `-hits=15` from the identical default.

**Ignore patterns prune, they don't filter.** A pattern ending in `/` makes the walk return `filepath.SkipDir`, so an ignored tree is never descended into. That distinction is the whole point on a target like `c:/`. Patterns without a `/` match base names at any depth; patterns with one match the whole path. Matching is case-insensitive only on Windows, keyed off `runtime.GOOS`, which is the *build target* — a cross-compiled Linux binary correctly stays case-sensitive. The scan root itself is exempt from pruning, otherwise a pattern matching your target would silently return nothing.

**Walk errors are values, not failures.** The `filepath.Walk` callback appends unreadable paths to `errorFiles` and returns `nil` rather than aborting — deliberate, so a permission-denied directory doesn't kill a scan of `c:/`. Callers surface `errorFiles` only when verbose. Preserve this when touching the walk.

**Logging is dual-output.** `heclog.LogMessage(logToConsole bool, ...)` always writes to `app.log` and echoes to stdout only when the first arg is true. Call sites pass either a literal `true` (always shown) or the `logToConsoleVerbose` package var in `main.go`, which is assigned from `-verbose` at the end of `initFlags()`. Anything logged before that assignment uses the `true` default.

## Build metadata

Four values are injected at link time by the Makefile's `LDFLAGS`, and all four have working defaults if you build with plain `go build` instead:

- `main.buildContext` — `"development"` makes `heclog.LogSetup` write to the hardcoded path `c:/repos/hecato/app.log`; anything else resolves `app.log` next to the executable. `make release` sets it to `release`. Building without ldflags leaves it `"development"`, so a plain `go build` binary run on another machine will fail at startup, because `LogSetup` errors and `main.go` calls `log.Fatalf`.
- `internal/version.Version` / `.Commit` / `.BuildTime` — from `git describe --tags --always --dirty`, `git rev-parse --short HEAD`, and a UTC timestamp. Without ldflags these stay `dev`/`none`/`unknown`.

## Known rough edges

Present in the code as written — don't treat them as bugs to fix unless asked, but don't be surprised by them either:

- The `-version` and `-log` flags are defined and printed but never consulted. Version is reachable only via `-method=version`; file logging is unconditional.
- The `-method` help string advertises `largedirs`, which does not exist, and omits `modfiles`, which does.
- `hits` is threaded through as a `string` and converted inside each `Get*` function.

## Roadmap

Planned work lives in the header comment block of `cmd/hecato/main.go` (ignore-file support, extension search, `searchfile`, `largedirs`, directory churn counts). Check there before proposing new methods. The README also notes an intent to eventually use concurrency in the walk — it is fully sequential today.
