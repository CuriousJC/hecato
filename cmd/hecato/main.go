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
TODO: new method: find all files with a particular extension
TODO: new method: file contents search (searchfile)
TODO: new method: directory size summary (largedirs)
TODO: new method: directory counts with files modified in the past X amount of hours - maybe even aggregated so I can see the most churn-y of directories

TODO: cleanup: the -version flag is defined and printed but never consulted, so it does nothing. Version is only reachable via -method=version
TODO: cleanup: the -log flag is likewise never consulted, so file logging cannot actually be turned off
TODO: cleanup: heclog resolves the log path from os.Args[0], which is only the executable path when the caller supplies one. Invoked by bare name via PATH on linux it will resolve to the working directory instead. os.Executable() is the reliable call
TODO: testing: there are no tests outside internal/config and internal/files. The method dispatch and flag precedence in this file are untested
TODO: performance: getFiles holds every file in memory before sorting and truncating, so a scan of a whole volume scales with the volume rather than with -hits. This collides with the churn-count method above
TODO: robustness: LogSetup failing is fatal, so the tool refuses to run if it cannot write app.log beside itself, e.g. installed under Program Files without admin
TODO: layout: move main.go out of cmd/ - this will only ever be one CLI, so the cmd/ abstraction is just noise. Not a drive-by: it moves MAIN_PATH in the Makefile and the paths both workflows reference, so it wants its own branch

DONE: functionality: generate an "ignore" file that can be updated as needed to throw out results -> -method=initconfig
DONE: functionality: Throw out OS files like page files and things like that on demand -> ignore_os_files in the config
DONE: functionality: ignore whole directories if I want to -> directory patterns prune the walk via filepath.SkipDir
DONE: functionality: ignore the go directories specifically because it changes A LOT of files -> shipped in the generated starter config
DONE: implement optional config file to override defaults -> internal/config

*/

package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/curiousjc/hecato/internal/config"
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
	configFlag  *string
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

	cfg := initFlags()
	printFlags(cfg)

	doWork(cfg)

	heclog.LogMessage(true, "---- Work completed.")

}

func doWork(cfg *config.Config) {

	heclog.LogMessagef(true, "---- Doing Work.  Method: %s ------\n", *method)

	ig := cfg.Matcher()
	if !ig.Empty() {
		heclog.LogMessage(logToConsoleVerbose, "Ignoring:", ig.Patterns())
	}

	switch *method {
	case "version":
		version.Print()
	case "initconfig":
		initConfig()
	case "largefiles":
		foundFiles, errorFiles, err := files.GetLargeFiles(*target, *hits, ig)
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
		foundFiles, errorFiles, err := files.GetModFiles(*target, *hits, ig)
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

// initFlags defines our inputs, folds in the config file, and handles edge cases
func initFlags() *config.Config {
	method = flag.String("method", "undefined", "REQUIRED: Specify the method to run. Options: [largefiles, modfiles, version, examples, initconfig]")
	m = flag.String("m", "undefined", "SHORTHAND for 'method'")
	target = flag.String("target", "undefined", "REQUIRED: Target of the supplied method")
	hits = flag.String("hits", "15", "OPTIONAL: How many hits for given method")
	verbose = flag.Bool("verbose", false, "OPTIONAL: Defaults to False.  Increases visible output.")
	logFlag = flag.Bool("log", true, "OPTIONAL: Enables Log file app.log.  Defaults to True.  Overwrites on execution. Always verbose.")
	exampleFlag = flag.Bool("examples", false, "OPTOINAL: Show examples of usage")
	eFlag = flag.Bool("e", false, "SHORTHAND for 'examples'")
	versionFlag = flag.Bool("version", false, "OPTIONAL: Show version")
	configFlag = flag.String("config", "", "OPTIONAL: Path to a config file. Defaults to hecato.yaml beside the executable.")

	flag.Parse()

	// initconfig exists to create a config, so being pointed at one that is not
	// there yet is the normal case for it rather than a failure. Checked before
	// the load, and against both spellings of the method flag, because the
	// method is not resolved until further down.
	writingConfig := *method == "initconfig" || *m == "initconfig"

	cfg, err := config.Load(*configFlag)
	if err != nil {
		if !writingConfig {
			// Being handed a config we cannot read is worth stopping for.
			// Carrying on with defaults would quietly ignore the rules you
			// asked for.
			heclog.LogMessage(true, "Config error:", err)
			os.Exit(1)
		}
		// Nothing usable to read, but we are about to write one. WriteStarter
		// still refuses to overwrite anything that is already there.
		cfg = &config.Config{}
	}

	applyConfigDefaults(cfg)

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

	return cfg
}

// setFlags reports which flags were actually typed on the command line, as
// opposed to sitting at their default. flag.Visit walks only the ones that were
// set, which is what makes "an explicit flag beats the config" possible.
func setFlags() map[string]bool {
	set := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		set[f.Name] = true
	})
	return set
}

// applyConfigDefaults fills in anything the config specifies that the user did
// not type. Precedence is: explicit flag, then config, then the built-in default.
func applyConfigDefaults(cfg *config.Config) {
	if !cfg.Loaded() {
		return
	}

	set := setFlags()
	d := cfg.Defaults

	// Either spelling of the method flag counts as the user having chosen one.
	if d.Method != "" && !set["method"] && !set["m"] {
		*method = d.Method
	}
	if d.Target != "" && !set["target"] {
		*target = d.Target
	}
	if d.Hits > 0 && !set["hits"] {
		*hits = strconv.Itoa(d.Hits)
	}
	if d.Verbose && !set["verbose"] {
		*verbose = true
	}
}

// initConfig writes a starter config, at -config if given or beside the
// executable otherwise.
func initConfig() {
	path := *configFlag
	if path == "" {
		var err error
		if path, err = config.DefaultPath(); err != nil {
			heclog.LogMessage(true, "Could not work out where to write the config:", err)
			return
		}
	}

	if err := config.WriteStarter(path); err != nil {
		heclog.LogMessage(true, "Could not write config:", err)
		return
	}

	heclog.LogMessagef(true, "Wrote a starter config to %s\n", path)
	heclog.LogMessage(true, "Everything in it is commented out except ignore_os_files, so edit it to taste.")
}

// printFlags prints out what our inputs were after being initiliazed
func printFlags(cfg *config.Config) {
	heclog.LogMessage(logToConsoleVerbose, "---- Inputs Determined ------")
	if cfg.Loaded() {
		heclog.LogMessage(logToConsoleVerbose, "Config:", cfg.Path)
	} else {
		heclog.LogMessage(logToConsoleVerbose, "Config:", "none found, using defaults")
	}
	heclog.LogMessage(logToConsoleVerbose, "Method:", *method)
	heclog.LogMessage(logToConsoleVerbose, "Target:", *target)
	heclog.LogMessage(logToConsoleVerbose, "Hits:", *hits)
	heclog.LogMessage(logToConsoleVerbose, "Log Flag:", *logFlag)
	heclog.LogMessage(logToConsoleVerbose, "Verbose:", *verbose)
	heclog.LogMessage(logToConsoleVerbose, "Examples Flag:", *exampleFlag)
	heclog.LogMessage(logToConsoleVerbose, "Short Examples Flag:", *eFlag)
	heclog.LogMessage(logToConsoleVerbose, "Version Flag:", *versionFlag)

}
