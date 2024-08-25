/*

main executable for hecato
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='d:/' -verbose=false
go run c:/repos/hecato/cmd/hecato/main.go -method=largefiles -hits=10 -target='c:/windows' -verbose=true
go run c:/repos/hecato/cmd/hecato/main.go -examples

go build -o hecato.exe c:/repos/hecato/cmd/hecato/main.go
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o hecato.exe cmd/hecato/main.go
.\hecato
c:/repos/hecato/hecato.exe


git tag v1.0.0
git push origin v1.0.0
git push origin main --tags

TODO: polish: move the log functions out to their own library from main so things like the examples method can use it as well
TODO: functionality: Throw out OS files like page files and things like that on demand, or maybe an input file to ignore certain things
TODO: new method: last changed files (changedfiles)
TODO: new method: last created files (createdfiles)
TODO: new method: file contents search (searchfile)
TODO: new method: directory size summary (largedirs)

*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	verbose     *bool
	exampleFlag *bool
	eFlag       *bool
	versionFlag *bool
	logFlag     *bool
)

var buildContext string = "development"
var logToConsoleVerbose bool = true

func main() {
	logMessage(true, "-----------------Running Hecato-------------")

	logFile, err := logSetup()
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}
	defer logFile.Close() // Ensure the log file is closed when main exits

	initFlags()
	printFlags()

	doWork()

	logMessage(true, "---- Work completed.")

}

func doWork() {

	logMessagef(true, "---- Doing Work.  Method: %s ------\n", *method)

	switch *method {
	case "version":
		version.Print()
	case "largefiles":
		foundFiles, errorFiles, err := listfiles.GetLargeFiles(*target, *hits)
		if err != nil {
			logMessage(true, "Error listing files: ", err)
		}

		for i, file := range foundFiles {
			logMessagef(true, " %s. Size: %s bytes | Path: %s \n", strconv.Itoa(i+1), file.SizeInMB(), file.Path)
		}
		for _, file := range errorFiles {
			logMessagef(logToConsoleVerbose, " Following access errors encountered: %s \n", file.Path)
		}
	case "examples":
		examples.Print()
	default:
		logMessage(true, "Unknown method.  Perhaps you should check out our examples (hecato -examples) or our help (hecato -help)")
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
		logMessage(true, "No method provided, defaulting to examples.")
		*method = "examples"
	case *method != "undefined" && *m != "undefined":
		logMessage(true, "Short and long form method both provided, please pick a lane, defaulting to examples.")
		*method = "examples"
	case *method == "undefined" && *m != "undefined":
		*method = *m
	}

	//defining app level verbosity now that the user has spoken
	logToConsoleVerbose = *verbose

}

// printFlags prints out what our inputs were after being initiliazed
func printFlags() {
	logMessage(logToConsoleVerbose, "---- Inputs Determined ------")
	logMessage(logToConsoleVerbose, "Method:", *method)
	logMessage(logToConsoleVerbose, "Target:", *target)
	logMessage(logToConsoleVerbose, "Hits:", *hits)
	logMessage(logToConsoleVerbose, "Log Flag:", *logFlag)
	logMessage(logToConsoleVerbose, "Verbose:", *verbose)
	logMessage(logToConsoleVerbose, "Examples Flag:", *exampleFlag)
	logMessage(logToConsoleVerbose, "Short Examples Flag:", *eFlag)
	logMessage(logToConsoleVerbose, "Version Flag:", *versionFlag)

}

func logSetup() (*os.File, error) {
	logFilePath := ""
	if buildContext == "development" {
		logFilePath = filepath.Join("c:/repos/hecato", "app.log")
	} else {
		// Get the path to the executable directory
		execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalf("failed to get executable directory: %v", err)
		}
		// Create or open the log file in the executable directory
		logFilePath = filepath.Join(execDir, "app.log")
	}

	println("Log Path ", logFilePath)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Configure the log package to use the log file
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Customize log format as needed
	log.Println("Logger starting...")

	return logFile, nil
}

// logMessage logs a message line and optionally prints to the console
func logMessage(logToConsole bool, v ...interface{}) {
	log.Println(v...)
	if logToConsole {
		fmt.Println(v...)
	}
}

// logMessagef logs a formatted message and optionally prints to the console
func logMessagef(logToConsole bool, format string, args ...interface{}) {
	log.Printf(format, args...)
	if logToConsole {
		fmt.Printf(format, args...)
	}
}
