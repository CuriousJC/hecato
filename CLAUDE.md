# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`hecato` is a single-binary filesystem investigation CLI (named for the Hecatoncheires). It walks a target path and reports the top N files by some ordering. Two direct dependencies: `gopkg.in/yaml.v3` for the config file and `github.com/fatih/color` for console output.

## Commands

```bash
go build -o hecato.exe cmd/hecato/main.go   # quick local build, no version injection
make all                                     # cross-compile linux + windows, dev logging
make release                                 # same, but BUILD_CONTEXT=release (what CI runs)
make test                                    # go test ./...
make vet
go run cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -verbose=true
```

`cmd/hecato`, `internal/heclog`, `internal/config`, `internal/files` and `internal/ignore` have tests; `internal/examples` and `internal/version` do not. A single test: `go test ./internal/ignore -run TestMatchDir -v`.

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
- `internal/files` — the analysis methods. `scan()` (unexported, `scan.go`) is the shared engine: a worker pool over directories returning a `*Result`. `topn.go` is the bounded collector it accumulates into; `getFiles.go` now holds only the `Result` type.
- `internal/config` — the optional YAML config; `internal/ignore` — the pattern matcher it feeds.
- `internal/heclog`, `internal/examples`, `internal/version` — support packages.

**The method pattern.** `GetLargeFiles` and `GetModFiles` are now nothing but a `worseFunc` and an `Atoi`: they parse `hits` and hand `scan` a comparator saying which of two files is the weaker candidate. Adding a method that ranks files differently means writing one two-line function. Everything is walked and sorted in memory before truncation, so `-target=c:/` on a large volume holds every file in the slice at once. A new method means a new file in `internal/files` following that shape, a new `case` in `doWork()`, a line in `internal/examples/examples.go`, and an update to the `-method` flag help text.

**Config and ignore.** A config is always optional — `config.Load` returns a zero `Config` and no error when none is found, and hecato behaves as it did before configs existed. Resolution is `-config`, then `hecato.yaml` beside the executable, then nothing; an explicit `-config` that does not exist is fatal, as is a malformed or misspelled one (the decoder runs with `KnownFields(true)`). The one exception is `-method=initconfig`, which is allowed to be pointed at a path that does not exist yet — that is the whole point of it. `main.go` checks for that method *before* loading, against both `-method` and `-m`, since the method is not resolved until later.

`hecato.example.yaml` at the repo root is a committed copy of what `initconfig` writes. `TestExampleConfigMatchesGenerator` is the only thing keeping the two in sync — if you change `starter.go`, regenerate the example or that test fails with the command to do it. The live `hecato.yaml` is gitignored, since a local build puts the binary at the repo root and would otherwise pick it up and show it dirty.

Flag precedence is **explicit flag > config > built-in default**, implemented with `flag.Visit` in `setFlags()` — `flag.Visit` walks only flags actually typed, which is the only way to tell `-hits=15` from the identical default.

**Ignore patterns prune, they don't filter.** A pattern ending in `/` makes the walk return `filepath.SkipDir`, so an ignored tree is never descended into. That distinction is the whole point on a target like `c:/`. Patterns without a `/` match base names at any depth; patterns with one match the whole path. Matching is case-insensitive only on Windows, keyed off `runtime.GOOS`, which is the *build target* — a cross-compiled Linux binary correctly stays case-sensitive. The scan root itself is exempt from pruning, otherwise a pattern matching your target would silently return nothing.

**Walk errors are values, not failures.** `readDir` appends unreadable paths to the worker's shard and carries on rather than aborting — deliberate, so a permission-denied directory doesn't kill a scan of `c:/`. Preserve this when touching the walk.

That convention has one sharp edge, which is why `checkTarget()` exists: a target that cannot be walked at all produces an empty result and no error, because the failure lands in `Result.Errors` like any other unreadable path. `main.go` therefore validates the target *before* the walk for any method in `walkingMethods`. Without that, a typo'd `-target` looks exactly like an empty disk.

`checkTarget` also rejects **drive-relative targets** — the `c:` and `c:temp` forms, as opposed to `c:/`. Windows resolves those against a current directory it tracks per drive, so `c:` means "wherever you last were on C:" rather than the volume root. This is the nastiest target bug the tool can have, and the only one that has to be caught *before* the `os.Stat`: the path is a real readable directory, so every check below it passes and the run reports a confident summary of the wrong tree. Found in testing v0.2.0, where `-target="d:"` scanned 56,054 files and `-target="c:"` scanned 2 — the two files in the working directory. `isDriveRelative` gates on `runtime.GOOS` for the same build-target reason the ignore matcher does; the parsing lives in `driveRelativeSyntax` so it is still tested on Linux CI.

**`Result` carries counts, not just files.** `Scanned`, `Ignored`, `Pruned` and `Matched` exist so a run can say what it did. Note `Scanned` does not include files under a pruned directory — they are never looked at, which is the entire point of pruning, and why `Pruned` counts directories separately rather than being folded into `Ignored`. `Matched` is the pre-truncation count, so the summary can say "18 files matched, showing the top 5".

**Logging is dual-output, and the two outputs are deliberately different.** Every `heclog` helper takes a leading bool controlling whether the message also reaches the console; the log file always gets it either way, because `app.log` is the record of what happened and `-verbose` only governs what you were shown at the time.

The console gets colour; `app.log` gets the identical text with no escape codes. Everything funnels through the unexported `write()` in `heclog.go`, which formats once, sends the plain string to `log`, and only then applies colour on the way to `color.Output`. `TestLogFileNeverGetsColorCodes` guards this — if it fails, the log file has become ungreppable. Anything added to heclog must preserve that split.

Use the semantic helpers (`Heading`, `Info`, `Detail`, `Success`, `Warn`, `Error`, `Field`) rather than `LogMessage`/`LogMessagef`, which are the uncoloured originals kept for compatibility.

Colour turns itself off when stdout is not a terminal and when `NO_COLOR` is set — both handled by `fatih/color`, not by us. `-no-color` calls `heclog.DisableColor()` for the explicit case.

`log.Lshortfile` is deliberately *not* set: since every message goes through `write()`, it would print the same constant on every line.

## Build metadata

Four values are injected at link time by the Makefile's `LDFLAGS`, and all four have working defaults if you build with plain `go build` instead:

- `main.buildContext` — `"development"` makes `heclog.LogSetup` write to the hardcoded path `c:/repos/hecato/app.log`; anything else resolves `app.log` next to the executable. `make release` sets it to `release`. Building without ldflags leaves it `"development"`, so a plain `go build` binary run on another machine will fail at startup, because `LogSetup` errors and `main.go` calls `log.Fatalf`.
- `internal/version.Version` / `.Commit` / `.BuildTime` — from `git describe --tags --always --dirty`, `git rev-parse --short HEAD`, and a UTC timestamp. Without ldflags these stay `dev`/`none`/`unknown`.

## Known rough edges

Present in the code as written — don't treat them as bugs to fix unless asked, but don't be surprised by them either:

- The `-version` and `-log` flags are defined but never consulted. Version is reachable only via `-method=version`; file logging is unconditional.
- `hits` is threaded through as a `string` and converted inside each `Get*` function.
- `File.SizeInMB()` survives alongside `HumanSize()`. It reports `0.00 MB` for anything under about 5 KB, which is why display uses `HumanSize`.

## Roadmap

Planned work lives in the header comment block of `cmd/hecato/main.go`. Check there before proposing new methods or performance work.

### Concurrency

`scan` is a worker pool over directories. Three things about it are load-bearing:

**The queue is an unbounded slice under a mutex, not a channel.** A buffered channel deadlocks here: workers are *also* producers, so once the buffer fills, every worker blocks trying to enqueue a subdirectory and nobody is left to drain it. The alternative fix — spawning a goroutine per blocked send — trades a deadlock for unbounded goroutines.

**Termination is `pending`, not an empty queue.** An empty queue means nothing while a worker is still running and might enqueue children. `finish()` decrements `pending` and only closes when it hits zero, and callers must push a directory's children *before* calling it or the walk ends early.

**Each worker owns a shard.** Counters and a private `topN`, merged once at the end, so the scanning path takes no locks at all. `topN.merge` is tested to give the same answer as a single collector — otherwise the result would depend on which worker happened to see which file.

The `Matcher` is read-only once built, which is what makes sharing one across workers safe.

**The race detector needs cgo**, so it cannot run on a stock Windows dev box. `ci.yml` runs `go test -race ./...` on Linux; that is the only place it reliably executes.

### A note on benchmark numbers

`filepath.Walk` was replaced by `filepath.WalkDir`. Worth recording what that taught us, because the remaining performance TODO carries a number from the same source:

A full `c:/` scan, 939,000 files, measured at each step:

| | time | peak memory |
|---|---|---|
| `filepath.Walk`, serial | 85.8s | — |
| `filepath.WalkDir` | 27.9s | 557.8 MB |
| + worker pool and top-N heap | 20.2s | — |
| + ignore matcher rewrite | **5.6s** | **95.7 MB** |

Two lessons worth keeping:

**Synthetic benchmarks overstated everything.** `WalkDir` measured 6.6x on a warm-cache `C:/Program Files` and delivered 3.07x on a real volume. The parallel walk measured 5.3x and delivered 4.8x only *after* the matcher was fixed. Always re-measure on a real target.

**The bottleneck moved.** After the worker pool, the walk was 5.6s and pattern matching was 14.6s — 72% of runtime, invisible until measured by rerunning with zero patterns. That is why `ruleSet` splits literal patterns from globs.

The `WalkDir` win is also largely Windows-specific: `FindNextFile` returns size and timestamps with the directory enumeration, so `DirEntry.Info()` is nearly free. On Linux `getdents` does not, so `Info()` still costs an `lstat` and the released Linux binary will see less.
