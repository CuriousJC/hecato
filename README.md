# hecato

hecato - Short for [Hecatoncheires](https://en.wikipedia.org/wiki/Hecatoncheires), the Hundred-Handed giants of Greek mythology - is a file system CLI that can be used to investigate various information about the system. Hecato is one of my first Go programs but is my attempt to use Go to solve a number of investigatory things I've found myself needing to do in the past. Eventually I hope to make the "hundred hands" reference meaningful by learning concurrent programming.

## Examples

Display example usage in the program

`go run c:/repos/hecato/cmd/hecato/main.go -method examples`

Searching my steam library on my D:/ drive for the largest 25 files

`go run c:/repos/hecato/cmd/hecato/main.go -method largefiles -hits 25 -target d:/SteamLibrary`

Searching the windows directory and displaying access errors. By default hecato masks access errors

`go run c:/repos/hecato/cmd/hecato/main.go -method largefiles -hits 10 -target 'c:/windows' -hide=false`
