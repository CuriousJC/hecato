package heclog

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/fatih/color"
)

// captureLog redirects the log package to a buffer, standing in for app.log.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	oldFlags := log.Flags()

	log.SetOutput(&buf)
	log.SetFlags(0)

	t.Cleanup(func() {
		log.SetOutput(nil)
		log.SetFlags(oldFlags)
	})

	return &buf
}

// The whole reason heclog splits console output from file output is so that
// app.log stays free of escape codes. If this fails, the log file has become
// unreadable in an editor and ungreppable from a script.
func TestLogFileNeverGetsColorCodes(t *testing.T) {
	// Force colour on, as if we were attached to a terminal. Without this the
	// test would pass trivially in CI, where stdout is not a TTY.
	oldNoColor := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = oldNoColor })

	buf := captureLog(t)

	Heading(false, "hecato %s", "v1.2.3")
	Info(false, "plain line")
	Detail(false, "dim line")
	Success(false, "scanned %d files", 42)
	Warn(false, "%d paths could not be read", 3)
	Error(false, "target does not exist: %s", "c:/nope")
	Field(false, "Method", "largefiles")

	out := buf.String()

	if strings.Contains(out, "\x1b[") {
		t.Errorf("escape codes reached the log file:\n%q", out)
	}

	// The text itself must still be there -- stripping colour must not mean
	// stripping content.
	for _, want := range []string{
		"hecato v1.2.3",
		"plain line",
		"dim line",
		"scanned 42 files",
		"3 paths could not be read",
		"target does not exist: c:/nope",
		"largefiles",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log file missing %q\ngot:\n%s", want, out)
		}
	}
}

// Every helper writes to the log regardless of the console flag, because the
// log file is the record of what happened and -verbose only governs what you
// were shown at the time.
func TestLogHappensEvenWhenConsoleIsOff(t *testing.T) {
	buf := captureLog(t)

	Detail(false, "written to file only")

	if !strings.Contains(buf.String(), "written to file only") {
		t.Error("a console-suppressed message did not reach the log file")
	}
}

func TestFieldAlignsLabels(t *testing.T) {
	buf := captureLog(t)

	Field(false, "Method", "largefiles")
	Field(false, "Target", "c:/")

	// Split without trimming: TrimSpace would strip the leading indent from the
	// first line only and make the comparison meaningless.
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), buf.String())
	}

	// Both values should start at the same column, so the block reads as a
	// table rather than ragged text.
	if strings.Index(lines[0], "largefiles") != strings.Index(lines[1], "c:/") {
		t.Errorf("field values are not aligned:\n%q\n%q", lines[0], lines[1])
	}
}

func TestDisableColor(t *testing.T) {
	oldNoColor := color.NoColor
	t.Cleanup(func() { color.NoColor = oldNoColor })

	color.NoColor = false
	if !ColorEnabled() {
		t.Error("ColorEnabled() = false when colour is on")
	}

	DisableColor()
	if ColorEnabled() {
		t.Error("ColorEnabled() = true after DisableColor()")
	}
}
