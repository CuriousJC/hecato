/*

main executable for hecato
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='d:/' -hide=true
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -hide=false
go run c:/repos/hecato/cmd/hecato/main.go -examples

go build -o hecato.exe c:/repos/hecato/cmd/hecato/main.go
c:/repos/hecato/hecato.exe

git tag v1.0.0
git push origin v1.0.0
git push origin main --tags

TODO: setup makefile for local creation
TODO: flag: handle no flags at all and maybe show examples and help
TODO: new method: last changed files
TODO: new method: last created files
TODO: new method: file contents search
TODO: new method: directory size summary
TODO: flag: verbose flag
TODO: flag: log file output
TODO: functionality: Throw out OS files like page files and things like that on demand, or maybe an input file to ignore certain things


*/

package main

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/curiousjc/hecato/internal/examples"
	"github.com/curiousjc/hecato/internal/listfiles"
	"github.com/curiousjc/hecato/internal/version"
)

func main() {
	fmt.Println("-----------------Running Hecato-------------")

	method := flag.String("method", "empty", "Specify the method to run")
	m := flag.String("m", "empty", "Specify the method to run short form")
	target := flag.String("target", "undefined", "target of the given method")
	hits := flag.String("hits", "15", "How many hits for given method")
	hide := flag.Bool("hide", true, "Hide access errors from output")

	exampleFlag := flag.Bool("examples", false, "Show examples of usage")
	eFlag := flag.Bool("e", false, "Show examples of usage")

	versionFlag := flag.Bool("version", false, "Show version")
	vFlag := flag.Bool("v", false, "Show version")

	flag.Parse()

	if *exampleFlag || *eFlag {
		*method = "examples"
	}

	if *versionFlag || *vFlag {
		*method = "version"
	}

	// Determine the method based on the provided flags
	switch {
	case *method == "empty" && *m == "empty":
		fmt.Println("No method provided, defaulting to examples.")
		*method = "examples"
	case *method != "empty" && *m != "empty":
		fmt.Println("Short and long form method both provided, please pick a lane, defaulting to examples.")
		*method = "examples"
	case *method == "empty" && *m != "empty":
		*method = *m
	}

	switch *method {
	case "version":
		version.Print()
	case "largefiles":
		foundFiles, errorFiles, err := listfiles.GetLargeFiles(*target, *hits)
		if err != nil {
			fmt.Println("Error listing files: ", err)
		}

		for i, file := range foundFiles {
			fmt.Printf(" %s. Size: %s bytes | Path: %s \n", strconv.Itoa(i), file.SizeInMB(), file.Path)
		}
		if !*hide {
			for _, file := range errorFiles {
				fmt.Printf(" Following access errors encountered: %s \n", file.Path)
			}
		}
	case "examples":
		examples.Print()
	default:
		fmt.Println("Unknown method.  Perhaps you should check out our examples (hecato -examples) or our help (hecato -help)")
	}
}
