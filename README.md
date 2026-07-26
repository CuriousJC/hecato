# hecato

hecato - Short for [Hecatoncheires](https://en.wikipedia.org/wiki/Hecatoncheires), the Hundred-Handed giants of Greek mythology - is a file system CLI that can be used to investigate various information about the system. Hecato is one of my first Go programs but is my attempt to use Go to solve a number of investigatory things I've found myself needing to do in the past. Eventually I hope to make the "hundred hands" reference meaningful by learning concurrent programming.

## Examples

```bash
hecato -examples
```

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

## Credit

Hecato CLI by CuriousJC
![CuriousJC Image](/static/CuriousJC.jpg)
