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
TODO: new method: find all files with a particular name portion
TODO: new method: file contents search (searchfile)
TODO: new method: directory size summary (largedirs)
TODO: new method: directory counts with files modified in the past X amount of hours - maybe even aggregated so I can see the most churn-y of directories

TODO: cleanup: the -version flag is defined but never consulted, so it does nothing. Typing `hecato -version` runs whatever method the config supplies instead, which on a default config means scanning c:/. Version is only reachable via -method=version
TODO: cleanup: the -log flag is likewise never consulted, so file logging cannot actually be turned off
TODO: cleanup: -help renders raw flag.PrintDefaults output, and the descriptions are stale: "OPTOINAL" is misspelled, and -target says REQUIRED when a config can supply it. Everything else the user reads got attention; help did not
TODO: cleanup: heclog resolves the log path from os.Args[0], which is only the executable path when the caller supplies one. Invoked by bare name via PATH on linux it will resolve to the working directory instead. os.Executable() is the reliable call, and internal/config already uses it
TODO: robustness: LogSetup failing is fatal, so the tool refuses to run if it cannot write app.log beside itself, e.g. installed under Program Files without admin
TODO: layout: move main.go out of cmd/ - this will only ever be one CLI, so the cmd/ abstraction is just noise. Not a drive-by: it moves MAIN_PATH in the Makefile and the paths both workflows reference, so it wants its own branch

TODO: output: modfiles prints absolute timestamps. A relative one - "2 minutes ago", "3 days ago" - is what you actually scan for when hunting recent churn, and is the same class of fix as swapping "0.00 MB" for "400 B"
TODO: output: unreadable paths are reported as a bare count. Grouping them by cause - "262 permission denied, 6 path too long" - would say whether they matter, instead of making -verbose and 268 lines the only way to find out
TODO: output: the run exits 0 even when paths could not be read. Defensible, since the scan did succeed, but it means a script cannot tell a clean scan from one that silently skipped a quarter of a tree

*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

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
	noColorFlag *bool
	workersFlag *int
)

var buildContext string = "development"
var logToConsoleVerbose bool = true

// noMethodGiven records that we fell back to examples because nothing was asked
// for, so the run can explain itself rather than silently doing something else.
var noMethodGiven bool

// walkingMethods are the methods that scan the filesystem and therefore need a
// usable target. Everything else runs fine without one.
var walkingMethods = map[string]bool{
	"largefiles": true,
	"modfiles":   true,
}

// knownMethods is every method doWork can dispatch. Validated up front so an
// unknown one fails before the intent summary rather than after it.
var knownMethods = map[string]bool{
	"largefiles": true,
	"modfiles":   true,
	"version":    true,
	"examples":   true,
	"initconfig": true,
}

func main() {
	logFile, err := heclog.LogSetup(buildContext)
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}
	defer logFile.Close() // Ensure the log file is closed when main exits

	cfg := initFlags()

	heclog.Heading(true, "hecato %s", version.Version)
	printFlags(cfg)

	doWork(cfg)
}

func doWork(cfg *config.Config) {
	ig := cfg.Matcher()

	// Everything that touches the filesystem needs somewhere to start. Checking
	// here rather than letting the walk fail means an unusable target says so,
	// instead of returning nothing and looking like an empty disk.
	if walkingMethods[*method] && !checkTarget() {
		os.Exit(1)
	}

	switch *method {
	case "version":
		version.Print()
	case "initconfig":
		initConfig()
	case "largefiles":
		heclog.Info(true, "")
		heclog.Info(true, "Scanning %s for the %s largest files...", *target, *hits)

		res, err := files.GetLargeFiles(*target, *hits, ig, *workersFlag)
		if err != nil {
			heclog.Error(true, "Could not list files: %v", err)
			os.Exit(1)
		}

		heclog.Info(true, "")

		// Bars are scaled against the biggest hit rather than the biggest file
		// on disk, so the top row is always full and the rest read as fractions
		// of it. That is the comparison you actually want: how much does the
		// worst offender dominate the others.
		var largest int64
		if len(res.Files) > 0 {
			largest = res.Files[0].Size
		}

		for i, file := range res.Files {
			// The bar is scaled against the largest hit, but the percentage is
			// of everything scanned. Two denominators would be confusing if the
			// percentage were doing the same job as the bar -- it is not. The
			// bar answers "how does this compare to the worst offender", the
			// percentage answers "how much of my disk is this actually worth".
			var share string
			if res.TotalBytes > 0 {
				share = fmt.Sprintf("%.1f%%", float64(file.Size)/float64(res.TotalBytes)*100)
			}

			heclog.Row(true,
				heclog.Seg(heclog.StylePlain, "  %3s. ", strconv.Itoa(i+1)),
				heclog.Seg(heclog.StyleSize, "%10s ", file.HumanSize()),
				heclog.Seg(heclog.StyleDim, "%6s ", share),
				heclog.Seg(heclog.StyleBar, " %s ", bar(file.Size, largest, barWidth)),
				heclog.Seg(heclog.StylePlain, " %s", file.Path),
			)
		}

		reportScan(res)
	case "modfiles":
		heclog.Info(true, "")
		heclog.Info(true, "Scanning %s for the %s most recently modified files...", *target, *hits)

		res, err := files.GetModFiles(*target, *hits, ig, *workersFlag)
		if err != nil {
			heclog.Error(true, "Could not list files: %v", err)
			os.Exit(1)
		}

		heclog.Info(true, "")
		for i, file := range res.Files {
			heclog.Info(true, "  %3s.  %s  %s", strconv.Itoa(i+1), file.ModTime.Format("2006-01-02 15:04:05"), file.Path)
		}

		reportScan(res)

	case "examples":
		if noMethodGiven {
			// Reaching examples because nothing was asked for is different from
			// asking for them. Say which happened, so a first-time run does not
			// look like the tool ignored you.
			heclog.Info(true, "")
			heclog.Warn(true, "No method given, so there is nothing to scan yet.")

			// A loaded config that supplies no method is its own situation, and
			// the unhelpful version of this message is the one that does not
			// mention it. "No method given" is true but sends you looking at
			// your command line when the gap is in the file.
			if cfg.Loaded() {
				heclog.Detail(true, "  Your config sets no defaults.method, so there is no default to fall back on:")
				heclog.Detail(true, "    %s", cfg.Path)
				heclog.Detail(true, "  Add a method under defaults: to make a bare `hecato` run it, for example")
				heclog.Detail(true, "    defaults:")
				heclog.Detail(true, "      method: largefiles")
				heclog.Detail(true, "      target: \"c:/\"")
			} else {
				heclog.Detail(true, "  hecato -method=initconfig   create a config so you can set defaults")
			}

			heclog.Detail(true, "  hecato -help      the full list of flags")
			heclog.Detail(true, "  hecato -examples  worked examples of each method")
		}
		heclog.Info(true, "")
		examples.Print()
	default:
		heclog.Error(true, "Unknown method %q.", *method)
		heclog.Detail(true, "Try hecato -examples for usage, or hecato -help for the full flag list.")
		os.Exit(1)
	}
}

// checkTarget verifies the scan target before the walk, and explains what is
// wrong when it cannot. Without this an unusable target produces an empty
// result set and no complaint, because the walk error lands in Result.Errors
// and those are only listed under -verbose.
func checkTarget() bool {
	if *target == "undefined" || *target == "" {
		heclog.Error(true, "No target given, so there is nothing to scan.")
		heclog.Detail(true, "Pass one with -target, or set defaults.target in your config.")
		heclog.Detail(true, "  hecato -method=%s -target=\"c:/\"", *method)
		return false
	}

	// This has to come before the Stat, because the Stat would succeed. A
	// drive-relative target is a real, readable directory - just not the one the
	// user meant - so every check below it passes and the run reports a confident
	// summary of the wrong volume.
	if isDriveRelative(*target) {
		heclog.Error(true, "Target is drive-relative, not the root of the drive: %s", *target)
		heclog.Detail(true, "Windows reads this as \"the current directory on that drive\", so it would")
		heclog.Detail(true, "scan wherever you last were on %s rather than the whole volume.", (*target)[:2])
		heclog.Detail(true, "  hecato -method=%s -target=\"%s\"", *method, driveRootHint(*target))
		return false
	}

	info, err := os.Stat(*target)
	if err != nil {
		if os.IsNotExist(err) {
			heclog.Error(true, "Target does not exist: %s", *target)
		} else {
			heclog.Error(true, "Cannot read target %s: %v", *target, err)
		}
		return false
	}

	if !info.IsDir() {
		heclog.Error(true, "Target is a file, not a directory: %s", *target)
		return false
	}

	return true
}

// isDriveRelative reports whether target is a Windows drive-relative path, the
// "c:" and "c:temp" forms. Windows resolves those against a current directory
// it tracks per drive, so "c:" means "wherever you last were on C:" and not the
// root of the volume. Someone scanning a disk essentially never means that, and
// it is the worst kind of wrong: it stats fine, walks fine, and returns a
// plausible handful of files from the working directory with no error at all.
//
// The GOOS check is the *build* target, matching internal/ignore: on linux "c:"
// is an ordinary relative filename and a cross-compiled binary must keep
// treating it as one. driveRelativeSyntax holds the parsing so it can be tested
// on any platform rather than only on Windows.
func isDriveRelative(target string) bool {
	return runtime.GOOS == "windows" && driveRelativeSyntax(target)
}

// driveRelativeSyntax is isDriveRelative without the platform gate.
func driveRelativeSyntax(target string) bool {
	if len(target) < 2 || target[1] != ':' {
		return false
	}

	c := target[0]
	if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
		return false
	}

	// "c:/" and "c:\" name the volume root and are exactly right. It is the
	// absence of that separator that makes the path relative.
	return len(target) == 2 || (target[2] != '/' && target[2] != '\\')
}

// driveRootHint rewrites a drive-relative target into the absolute form the
// user probably wanted, so the error can suggest something they can paste back.
func driveRootHint(target string) string {
	return target[:2] + "/" + target[2:]
}

// reportScan is the closing summary. It exists so a run says what it actually
// did rather than leaving you to infer it from the results, and in particular so
// that unreadable paths and ignored files are visible without -verbose.
func reportScan(res *files.Result) {
	heclog.Info(true, "")

	// Note that the byte total belongs to the matched files, not the scanned
	// ones: TotalBytes accumulates after the ignore check. Pairing it with
	// Scanned would put two different populations in one sentence, so the size
	// is reported on the "matched" line below instead.
	heclog.Success(true, "Scanned %s in %s",
		plural(res.Scanned, "file", "files"), res.Elapsed.Round(time.Millisecond))

	if res.Ignored > 0 || res.Pruned > 0 {
		heclog.Detail(true, "  ignored %s and skipped %s via ignore rules",
			plural(res.Ignored, "file", "files"),
			plural(res.Pruned, "directory", "directories"))
	}

	// The size total goes here because it is the size of what matched, which is
	// also the denominator behind the percentage column above.
	if res.Matched > len(res.Files) {
		heclog.Detail(true, "  %s matched, %s in total, showing the top %d",
			plural(res.Matched, "file", "files"), files.HumanBytes(res.TotalBytes), len(res.Files))
	} else if res.Matched > 0 {
		heclog.Detail(true, "  %s matched, %s in total",
			plural(res.Matched, "file", "files"), files.HumanBytes(res.TotalBytes))
	}

	// A single large file only means something against the whole, so this sits
	// immediately after the total it refers to. Move one and move the other.
	//
	// Only for largefiles: under modfiles the first row is the most recently
	// modified file, which says nothing about size.
	if *method == "largefiles" && len(res.Files) > 0 && res.TotalBytes > 0 {
		if share := float64(res.Files[0].Size) / float64(res.TotalBytes) * 100; share >= 0.1 {
			heclog.Detail(true, "  the largest single file is %.1f%% of that total", share)
		}
	}

	if n := len(res.Errors); n > 0 {
		heclog.Warn(true, "  %s could not be read", plural(n, "path", "paths"))
		if logToConsoleVerbose {
			for _, file := range res.Errors {
				heclog.Detail(true, "    %s", file.Path)
			}
		} else {
			heclog.Detail(true, "    run again with -verbose=true to list them")
		}
	}
}

// barWidth is how many cells a full bar occupies. Wide enough to show a
// meaningful fraction, narrow enough to leave room for long Windows paths on an
// 80-column terminal.
const barWidth = 16

// barCell is the only glyph the bar uses: U+2588 FULL BLOCK.
//
// Deliberately not the partial blocks U+2589-U+258F. They would give sub-cell
// resolution, but the Windows console font ships U+2588 without them, so they
// render as missing-glyph boxes -- a bar that looks broken is worse than one
// that is merely coarse. The percentage column carries the precision the
// partial blocks would have added.
const barCell = "█"

// bar renders size as a proportion of largest, rounded to whole cells.
//
// The scale is linear on purpose. A log scale would make the tail more legible,
// but it would also flatter it: when one file is four times the next, the honest
// picture is that the bar is four times longer. Anything under half a cell is
// left blank rather than rounded up to one, so a bar of any length always means
// "at least this much" rather than "something, possibly nothing".
func bar(size, largest int64, width int) string {
	if largest <= 0 || size <= 0 {
		return strings.Repeat(" ", width)
	}

	cells := int(float64(size)/float64(largest)*float64(width) + 0.5)
	if cells > width {
		cells = width
	}

	return strings.Repeat(barCell, cells) + strings.Repeat(" ", width-cells)
}

// plural pairs a grouped count with the right form of its noun, so a summary
// reads "1 file" rather than "1 files".
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return comma(n) + " " + many
}

// comma groups digits so six-figure file counts stay readable.
func comma(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return "-" + comma(-n)
	}
	if len(s) <= 3 {
		return s
	}

	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
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
	noColorFlag = flag.Bool("no-color", false, "OPTIONAL: Disable coloured output. Colour is off automatically when piped or when NO_COLOR is set.")
	workersFlag = flag.Int("workers", files.DefaultWorkers, "OPTIONAL: How many directories to read concurrently.")

	flag.Parse()

	if *noColorFlag {
		heclog.DisableColor()
	}

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
			heclog.Error(true, "Config error: %v", err)
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
		// Recorded rather than announced here, because the explanation belongs
		// next to the examples once we know a config did not supply a method.
		noMethodGiven = true
		*method = "examples"
	case *method != "undefined" && *m != "undefined":
		heclog.Warn(true, "Both -method and -m were given; pick one. Showing examples instead.")
		*method = "examples"
	case *method == "undefined" && *m != "undefined":
		*method = *m
	}

	//defining app level verbosity now that the user has spoken
	logToConsoleVerbose = *verbose

	// Rejected here rather than in doWork so the failure comes before the
	// intent summary. Describing a run that is not going to happen is noise.
	if !knownMethods[*method] {
		heclog.Error(true, "Unknown method %q.", *method)
		heclog.Detail(true, "Known methods: largefiles, modfiles, version, examples, initconfig")
		heclog.Detail(true, "Try hecato -examples for usage, or hecato -help for the full flag list.")
		os.Exit(1)
	}

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
	if d.Workers > 0 && !set["workers"] {
		*workersFlag = d.Workers
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

// printFlags states what this run is about to do, before it does it.
//
// This is deliberately not a dump of every flag. It answers the questions you
// would otherwise have to reverse-engineer from the output: which method, over
// what, with which rules in effect, and where those rules came from. The old
// version was verbose-only, which meant the default run explained nothing.
func printFlags(cfg *config.Config) {
	// Nothing to describe for the methods that do not act on the filesystem.
	if *method == "examples" || *method == "version" {
		return
	}

	heclog.Info(true, "")

	heclog.Field(true, "Method", *method)

	if walkingMethods[*method] {
		heclog.Field(true, "Target", *target)
		heclog.Field(true, "Hits", *hits)
		if *workersFlag != files.DefaultWorkers {
			heclog.Field(true, "Workers", strconv.Itoa(*workersFlag))
		}
	}

	if cfg.Loaded() {
		heclog.Field(true, "Config", cfg.Path)
	} else {
		heclog.Field(true, "Config", "none found, using built-in defaults")
	}

	describeIgnores(cfg)
}

// describeIgnores summarises the ignore rules in effect. The count is always
// shown because silently dropping results is exactly the thing a user needs to
// know about; the patterns themselves are only worth the space under -verbose.
func describeIgnores(cfg *config.Config) {
	if !walkingMethods[*method] {
		return
	}

	ig := cfg.Matcher()
	if ig.Empty() {
		heclog.Field(true, "Ignoring", "nothing")
		return
	}

	patterns := ig.Patterns()

	summary := fmt.Sprintf("%d patterns", len(patterns))
	if cfg.IgnoreOSFiles {
		summary += ", including OS files such as pagefile.sys"
	}
	heclog.Field(true, "Ignoring", summary)

	if logToConsoleVerbose {
		for _, p := range patterns {
			heclog.Detail(true, "             %s", p)
		}
	}
}
