/*

main executable for hecato
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='d:/' -hide=true
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -hide=false
go run c:/repos/hecato/cmd/hecato/main.go -examples

go build -o hecato.exe c:/repos/hecato/cmd/hecato/main.go
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o hecato.exe cmd/hecato/main.go
.\hecato
c:/repos/hecato/hecato.exe


git tag v1.0.0
git push origin v1.0.0
git push origin main --tags

TODO: new method: last changed files (changedfiles)
TODO: new method: last created files (createdfiles)
TODO: new method: file contents search (searchfile)
TODO: new method: directory size summary (largedirs)
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

var (
	method      *string
	m           *string
	target      *string
	hits        *string
	hide        *bool
	exampleFlag *bool
	eFlag       *bool
	versionFlag *bool
	vFlag       *bool
)

func main() {
	fmt.Println("-----------------Running Hecato-------------")

	initFlags()
	printFlags()
	doWork()

}

func doWork() {

	fmt.Println("---- Doing Work ------")

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

// initFlags defines our inputs and handles edge cases
func initFlags() {
	method = flag.String("method", "undefined", "Specify the method to run. Options: [largefiles, largedirs]")
	m = flag.String("m", "undefined", "Specify the method to run short form")
	target = flag.String("target", "undefined", "target of the given method")
	hits = flag.String("hits", "15", "How many hits for given method")
	hide = flag.Bool("hide", true, "Hide access errors from output")
	exampleFlag = flag.Bool("examples", false, "Show examples of usage")
	eFlag = flag.Bool("e", false, "Show examples of usage")
	versionFlag = flag.Bool("version", false, "Show version")
	vFlag = flag.Bool("v", false, "Show version")

	flag.Parse()

	//clobber our input method if there was example or version flag
	if *versionFlag || *vFlag {
		*method = "version"
	}

	if *exampleFlag || *eFlag {
		*method = "examples"
	}

	// Handling scenarios we can land in with two different method flags
	switch {
	case *method == "undefined" && *m == "undefined":
		fmt.Println("No method provided, defaulting to examples.")
		*method = "examples"
	case *method != "undefined" && *m != "undefined":
		fmt.Println("Short and long form method both provided, please pick a lane, defaulting to examples.")
		*method = "examples"
	case *method == "undefined" && *m != "undefined":
		*method = *m
	}

}

// printFlags prints out what our inputs were after being initiliazed
func printFlags() {
	fmt.Println("---- Inputs Determined ------")
	fmt.Println("Method:", *method)
	fmt.Println("Target:", *target)
	fmt.Println("Hits:", *hits)
	fmt.Println("Hide:", *hide)
	fmt.Println("Examples Flag:", *exampleFlag)
	fmt.Println("Short Examples Flag:", *eFlag)
	fmt.Println("Version Flag:", *versionFlag)
	fmt.Println("Short Version Flag:", *vFlag)

}
