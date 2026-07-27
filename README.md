# hecato

hecato - Short for [Hecatoncheires](https://en.wikipedia.org/wiki/Hecatoncheires), the Hundred-Handed giants of Greek mythology - is a file system CLI that can be used to investigate various information about the system. Hecato is one of my first Go programs but is my attempt to use Go to solve a number of investigatory things I've found myself needing to do in the past. Eventually I hope to make the "hundred hands" reference meaningful by learning concurrent programming.

## Examples

```bash
hecato -examples
```

A run states what it is about to do, then what it actually did:

```
hecato v0.1.0

  Method     largefiles
  Target     c:/repos
  Hits       15
  Config     C:\repos\hecato\hecato.yaml
  Ignoring   20 patterns, including OS files such as pagefile.sys

Scanning c:/repos for the 15 largest files...

    1.   27.62 MB  10.9%  ████████████████  c:\repos\godanmaku\...\SoukouMincho.go
    2.   11.26 MB   4.5%  ███████           c:\repos\breakout-ebitengine\public\pong.wasm
    3.    9.36 MB   3.7%  █████             c:\repos\godanmaku\...\SoukouMincho.ttf

Scanned 18,708 files, 252.73 MB, in 1.5s
  the largest single file is 10.9% of that
  ignored 32 files and skipped 29 directories via ignore rules
  18,676 files matched, showing the top 15
```

The bar is scaled against the largest file in the results, so the top row is
always full and the rest read as fractions of it. The percentage is of
everything scanned, which is the different and equally useful question: not
"how does this compare to the worst offender" but "how much of my disk is this
actually worth". The percentage is also what carries the tail, where the bar
rounds to nothing.

The scale is linear. A log scale would make the tail more legible but would
flatter it — if one file is four times the next, the bar should be four times
longer.

The bar uses only U+2588 FULL BLOCK, rounded to whole cells. The partial blocks
U+2589–U+258F would give sub-cell resolution, but the Windows console font
ships U+2588 without them, so they render as missing-glyph boxes. A bar that
looks broken is worse than one that is merely coarse.

Output is coloured when attached to a terminal. It turns itself off when piped,
when `NO_COLOR` is set, or with `-no-color`. `app.log` never contains colour
codes regardless — it always gets the plain text.

## Config

Optional. Without one, hecato behaves exactly as it always has.

```bash
hecato -method=initconfig   # writes a starter hecato.yaml beside the executable
```

[`hecato.example.yaml`](hecato.example.yaml) in this repo is a copy of what that
generates, if you'd rather read it than run it.

hecato looks for `hecato.yaml` beside its own executable, or wherever `-config`
points. The file can set default method/target/hits/verbose, skip the OS files
that otherwise dominate a `largefiles` run on Windows, and ignore paths:

```yaml
defaults:
  method: largefiles
  hits: 25

ignore_os_files: true   # pagefile.sys, hiberfil.sys, $Recycle.Bin and friends

ignore:
  - "*.tmp"           # base name, at any depth
  - "node_modules/"   # trailing slash prunes the directory entirely
  - "c:/pagefile.sys" # a pattern containing / matches the whole path
```

A flag you actually type always beats the config. An ignore entry ending in `/`
prunes rather than filters, so the walk never descends into it at all.

`defaults.workers`, or `-workers`, sets how many directories are read at once.
The default of 8 was measured rather than guessed: on a 12-CPU machine a full
`c:/` scan took 8.9s at 4 workers, 6.3s at 8, and 6.0s at 16. Past 8 the work
is waiting on the disk rather than the CPU.

If the config sets a `method`, a bare `hecato` runs it. Comment that line out to
get the examples back as the no-argument behaviour.

## Performance

A full `c:/` scan on the author's machine, 939,000 files:

| | time | peak memory |
|---|---|---|
| `filepath.Walk`, serial | 85.8s | — |
| `filepath.WalkDir` | 27.9s | 557.8 MB |
| + worker pool, bounded results | 20.2s | — |
| + ignore matcher rewrite | **5.6s** | **95.7 MB** |

The memory drop is the bounded collector: hecato no longer holds 872,000 file
records in order to report fifteen of them. Memory now scales with `-hits`
rather than with the size of the volume.

## Credit

Hecato CLI by CuriousJC
![CuriousJC Image](/static/CuriousJC.jpg)
