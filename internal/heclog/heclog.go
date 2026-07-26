// Package heclog is hecato's logging. Every message goes to app.log; whether it
// also reaches the console is the caller's choice, passed as the leading bool.
//
// The console and the log file are deliberately not the same stream. The console
// gets colour; app.log gets the identical text with no escape codes in it, so
// the file stays greppable and readable in an editor. Anything added here must
// preserve that split.
package heclog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

func LogSetup(buildContext string) (*os.File, error) {
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

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	// Configure the log package to use the log file.
	//
	// No Lshortfile: every message funnels through write() below, so the file
	// and line would be the same constant on every line. It was already close
	// to useless when it pointed at LogMessage; as a fixed value it is pure
	// noise in front of the text you actually want to read.
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("Logger starting...")

	return logFile, nil
}

// DisableColor turns off colour unconditionally, for -no-color.
//
// It does not need to be called for a pipe or for NO_COLOR: the color package
// already disables itself when stdout is not a terminal and when NO_COLOR is
// set in the environment.
func DisableColor() {
	color.NoColor = true
}

// ColorEnabled reports whether console output will actually be coloured, after
// TTY detection, NO_COLOR and -no-color have all had their say.
func ColorEnabled() bool {
	return !color.NoColor
}

// logMessage logs a message line and optionally prints to the console
func LogMessage(logToConsole bool, v ...interface{}) {
	log.Println(v...)
	if logToConsole {
		fmt.Println(v...)
	}
}

// logMessagef logs a formatted message and optionally prints to the console
func LogMessagef(logToConsole bool, format string, args ...interface{}) {
	log.Printf(format, args...)
	if logToConsole {
		fmt.Printf(format, args...)
	}
}

// write is the single path through which every coloured message goes. The log
// file always receives the plain string; only the console sees the escape codes.
func write(c *color.Color, logToConsole bool, format string, args ...interface{}) {
	plain := fmt.Sprintf(format, args...)

	log.Print(plain)

	if !logToConsole {
		return
	}

	// color.Output is the wrapped stdout that makes escape codes work on older
	// Windows consoles. Writing to it rather than os.Stdout is what keeps this
	// working outside Windows Terminal.
	if c == nil {
		fmt.Fprintln(color.Output, plain)
		return
	}
	c.Fprintln(color.Output, plain)
}

var (
	headingColor = color.New(color.Bold)
	labelColor   = color.New(color.FgCyan)
	successColor = color.New(color.FgGreen)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed, color.Bold)
	detailColor  = color.New(color.Faint)
)

// Heading is a section title.
func Heading(logToConsole bool, format string, args ...interface{}) {
	write(headingColor, logToConsole, format, args...)
}

// Info is ordinary output with no colour of its own.
func Info(logToConsole bool, format string, args ...interface{}) {
	write(nil, logToConsole, format, args...)
}

// Detail is secondary text: dimmed, for things worth showing but not reading.
func Detail(logToConsole bool, format string, args ...interface{}) {
	write(detailColor, logToConsole, format, args...)
}

// Success marks a completed run.
func Success(logToConsole bool, format string, args ...interface{}) {
	write(successColor, logToConsole, format, args...)
}

// Warn is for things that did not stop the run but that you should know about,
// such as unreadable paths.
func Warn(logToConsole bool, format string, args ...interface{}) {
	write(warnColor, logToConsole, format, args...)
}

// Error is for the run failing. Callers decide whether to exit.
func Error(logToConsole bool, format string, args ...interface{}) {
	write(errorColor, logToConsole, format, args...)
}

// Style names the colours a Segment can take. Named rather than exposing
// *color.Color so the rest of the program does not have to import the colour
// library to print a line.
type Style int

const (
	StylePlain Style = iota
	StyleDim
	StyleSize
	StyleBar
	StyleWarn
)

func (s Style) color() *color.Color {
	switch s {
	case StyleDim:
		return detailColor
	case StyleSize:
		return sizeColor
	case StyleBar:
		return barColor
	case StyleWarn:
		return warnColor
	default:
		return nil
	}
}

var (
	sizeColor = color.New(color.FgHiYellow)
	barColor  = color.New(color.FgBlue)
)

// Segment is one styled run of text within a Row.
type Segment struct {
	Text  string
	Style Style
}

// Seg builds a Segment, formatting like Printf.
func Seg(style Style, format string, args ...interface{}) Segment {
	return Segment{Text: fmt.Sprintf(format, args...), Style: style}
}

// Row prints segments as a single line, each in its own colour, while the log
// file receives the plain concatenation.
//
// This exists because Info and friends colour a whole line, which cannot
// distinguish a size from the path next to it. The invariant to preserve: what
// reaches the log is exactly the concatenated Text fields, no escape codes.
func Row(logToConsole bool, segs ...Segment) {
	var plain strings.Builder
	for _, s := range segs {
		plain.WriteString(s.Text)
	}

	log.Print(plain.String())

	if !logToConsole {
		return
	}

	for _, s := range segs {
		if c := s.Style.color(); c != nil {
			c.Fprint(color.Output, s.Text)
			continue
		}
		fmt.Fprint(color.Output, s.Text)
	}
	fmt.Fprintln(color.Output)
}

// Field prints an aligned "label  value" pair, used by the intent summary. The
// label is coloured and padded; the value is left plain so paths stay easy to
// copy out of a terminal.
func Field(logToConsole bool, label, value string) {
	plain := fmt.Sprintf("  %-10s %s", label, value)

	log.Print(plain)

	if !logToConsole {
		return
	}
	fmt.Fprintf(color.Output, "  %s %s\n", labelColor.Sprintf("%-10s", label), value)
}
