/*

main executable for hecato
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='d:/' -verbose=false
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -verbose=true
go run c:/repos/hecato/cmd/hecato/main.go -method=modfiles -hits=10 -target='d:/' -verbose=false
go run c:/repos/hecato/cmd/hecato/main.go -method=modfiles -hits=10000 -target='c:/' -verbose=false
go run c:/repos/hecato/cmd/hecato/main.go -examples


// For creating an executable:::
go build -o hecato.exe c:/repos/hecato/cmd/hecato/main.go
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o hecato.exe cmd/hecato/main.go
.\hecato
c:/repos/hecato/hecato.exe

//For creating a release:::
git tag v1.0.0
git push origin v1.0.0
git push origin main --tags

//Stuff todo:::
TODO: functionality: Throw out OS files like page files and things like that on demand, or maybe an input file to ignore certain things
TODO: functionality: ignore whole directories if I want to
TODO: new method: file contents search (searchfile)
TODO: new method: directory size summary (largedirs)

*/

package main

import (
	"flag"
	"log"
	"strconv"

	"github.com/curiousjc/hecato/internal/examples"
	"github.com/curiousjc/hecato/internal/files"
	"github.com/curiousjc/hecato/internal/heclog"
	"github.com/curiousjc/hecato/internal/version"
)

var (
	method      *string
	m           *string
	target      *string
	hits        *string
	verbose     *bool
	exampleFlag *bool
	eFlag       *bool
	versionFlag *bool
	logFlag     *bool
)

var buildContext string = "development"
var logToConsoleVerbose bool = true

func main() {
	heclog.LogMessage(true, "-----------------Running Hecato-------------")

	logFile, err := heclog.LogSetup(buildContext)
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}
	defer logFile.Close() // Ensure the log file is closed when main exits

	initFlags()
	printFlags()

	doWork()

	heclog.LogMessage(true, "---- Work completed.")

}

func doWork() {

	heclog.LogMessagef(true, "---- Doing Work.  Method: %s ------\n", *method)

	switch *method {
	case "version":
		version.Print()
	case "largefiles":
		foundFiles, errorFiles, err := files.GetLargeFiles(*target, *hits)
		if err != nil {
			heclog.LogMessage(true, "Error listing files: ", err)
		}

		for i, file := range foundFiles {
			heclog.LogMessagef(true, " %s. Size: %s bytes | Path: %s \n", strconv.Itoa(i+1), file.SizeInMB(), file.Path)
		}

		if logToConsoleVerbose {
			for _, file := range errorFiles {
				heclog.LogMessagef(logToConsoleVerbose, " Following access errors encountered: %s \n", file.Path)
			}
		}
	case "modfiles":
		foundFiles, errorFiles, err := files.GetModFiles(*target, *hits)
		if err != nil {
			heclog.LogMessage(true, "Error listing files: ", err)
		}

		for i, file := range foundFiles {
			heclog.LogMessagef(true, " %s. ModTime: %s | Path: %s \n", strconv.Itoa(i+1), file.ModTime, file.Path)
		}

		if logToConsoleVerbose {
			for _, file := range errorFiles {
				heclog.LogMessagef(logToConsoleVerbose, " Following access errors encountered: %s \n", file.Path)
			}
		}

	case "examples":
		examples.Print()
	default:
		heclog.LogMessage(true, "Unknown method.  Perhaps you should check out our examples (hecato -examples) or our help (hecato -help)")
	}
}

// initFlags defines our inputs and handles edge cases
func initFlags() {
	method = flag.String("method", "undefined", "REQUIRED: Specify the method to run. Options: [largefiles, largedirs]")
	m = flag.String("m", "undefined", "SHORTHAND for 'method'")
	target = flag.String("target", "undefined", "REQUIRED: Target of the supplied method")
	hits = flag.String("hits", "15", "OPTIONAL: How many hits for given method")
	verbose = flag.Bool("verbose", false, "OPTIONAL: Defaults to False.  Increases visible output.")
	logFlag = flag.Bool("log", true, "OPTIONAL: Enables Log file app.log.  Defaults to True.  Overwrites on execution. Always verbose.")
	exampleFlag = flag.Bool("examples", false, "OPTOINAL: Show examples of usage")
	eFlag = flag.Bool("e", false, "SHORTHAND for 'examples'")
	versionFlag = flag.Bool("version", false, "OPTIONAL: Show version")

	flag.Parse()

	//clobber our input method if examples were requested
	if *exampleFlag || *eFlag {
		*method = "examples"
	}

	// Handling scenarios we can land in with two different method flags
	switch {
	case *method == "undefined" && *m == "undefined":
		heclog.LogMessage(true, "No method provided, defaulting to examples.")
		*method = "examples"
	case *method != "undefined" && *m != "undefined":
		heclog.LogMessage(true, "Short and long form method both provided, please pick a lane, defaulting to examples.")
		*method = "examples"
	case *method == "undefined" && *m != "undefined":
		*method = *m
	}

	//defining app level verbosity now that the user has spoken
	logToConsoleVerbose = *verbose

}

// printFlags prints out what our inputs were after being initiliazed
func printFlags() {
	heclog.LogMessage(logToConsoleVerbose, "---- Inputs Determined ------")
	heclog.LogMessage(logToConsoleVerbose, "Method:", *method)
	heclog.LogMessage(logToConsoleVerbose, "Target:", *target)
	heclog.LogMessage(logToConsoleVerbose, "Hits:", *hits)
	heclog.LogMessage(logToConsoleVerbose, "Log Flag:", *logFlag)
	heclog.LogMessage(logToConsoleVerbose, "Verbose:", *verbose)
	heclog.LogMessage(logToConsoleVerbose, "Examples Flag:", *exampleFlag)
	heclog.LogMessage(logToConsoleVerbose, "Short Examples Flag:", *eFlag)
	heclog.LogMessage(logToConsoleVerbose, "Version Flag:", *versionFlag)

}
