package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

func TestLoadFull(t *testing.T) {
	path := writeTemp(t, `
defaults:
  method: modfiles
  target: "d:/"
  hits: 25
  verbose: true
ignore_os_files: true
ignore:
  - "*.tmp"
  - "node_modules/"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Defaults.Method != "modfiles" {
		t.Errorf("Method = %q, want modfiles", cfg.Defaults.Method)
	}
	if cfg.Defaults.Target != "d:/" {
		t.Errorf("Target = %q, want d:/", cfg.Defaults.Target)
	}
	if cfg.Defaults.Hits != 25 {
		t.Errorf("Hits = %d, want 25", cfg.Defaults.Hits)
	}
	if !cfg.Defaults.Verbose {
		t.Error("Verbose = false, want true")
	}
	if !cfg.IgnoreOSFiles {
		t.Error("IgnoreOSFiles = false, want true")
	}
	if len(cfg.Ignore) != 2 {
		t.Errorf("Ignore = %q, want 2 entries", cfg.Ignore)
	}
	if !cfg.Loaded() {
		t.Error("Loaded() = false after a successful load")
	}
}

func TestLoadEmptyFileIsValid(t *testing.T) {
	cfg, err := Load(writeTemp(t, ""))
	if err != nil {
		t.Fatalf("empty config returned error: %v", err)
	}
	if len(cfg.Ignore) != 0 || cfg.IgnoreOSFiles {
		t.Errorf("empty config produced %+v, want zero values", cfg)
	}
}

func TestLoadMissingExplicitPathIsAnError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("Load of a missing explicit path returned no error, want one")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	_, err := Load(writeTemp(t, "ignore_os_fils: true\n"))
	if err == nil {
		t.Fatal("a misspelled field was accepted, want an error")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %v, want it to mention parsing", err)
	}
}

func TestLoadRejectsMalformedYAML(t *testing.T) {
	if _, err := Load(writeTemp(t, "ignore: [unclosed\n")); err == nil {
		t.Fatal("malformed YAML was accepted, want an error")
	}
}

func TestNoConfigIsNotAnError(t *testing.T) {
	// With no explicit path and (almost certainly) no hecato.yaml beside the
	// test binary, Load should succeed with an unloaded config.
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(\"\") returned a nil config")
	}
}

func TestMatcherFoldsInOSFiles(t *testing.T) {
	withOS := &Config{Ignore: []string{"*.tmp"}, IgnoreOSFiles: true}
	if !withOS.Matcher().MatchFile("c:/pagefile.sys") {
		t.Error("ignore_os_files did not match pagefile.sys")
	}
	if !withOS.Matcher().MatchFile("a/b.tmp") {
		t.Error("explicit pattern stopped matching once OS files were added")
	}

	withoutOS := &Config{Ignore: []string{"*.tmp"}}
	if withoutOS.Matcher().MatchFile("c:/pagefile.sys") {
		t.Error("pagefile.sys matched with ignore_os_files off")
	}
}

func TestNilConfigMatcherIsSafe(t *testing.T) {
	var cfg *Config
	if m := cfg.Matcher(); m == nil || !m.Empty() {
		t.Error("nil Config produced a non-empty matcher")
	}
	if cfg.Loaded() {
		t.Error("nil Config reported Loaded")
	}
}

func TestStarterIsValidConfig(t *testing.T) {
	// The generated file must round-trip through the strict loader, otherwise
	// initconfig hands the user something that fails on the next run.
	cfg, err := Load(writeTemp(t, string(Starter())))
	if err != nil {
		t.Fatalf("generated starter config failed to load: %v", err)
	}
	if !cfg.IgnoreOSFiles {
		t.Error("starter config did not enable ignore_os_files")
	}
	if len(cfg.Ignore) != 0 {
		t.Errorf("starter config had active ignore entries %q, want all commented out", cfg.Ignore)
	}
}

// The defaults ship active. A generated config that sets no target leaves hecato
// walking the literal path "undefined", which reports nothing and, because walk
// errors are only shown under -verbose, does not say why.
func TestStarterShipsUsableDefaults(t *testing.T) {
	cfg, err := Load(writeTemp(t, string(Starter())))
	if err != nil {
		t.Fatalf("generated starter config failed to load: %v", err)
	}

	if cfg.Defaults.Target == "" {
		t.Error("starter config left target unset, so a run would walk \"undefined\"")
	}
	if cfg.Defaults.Hits <= 0 {
		t.Errorf("starter config Hits = %d, want a positive count", cfg.Defaults.Hits)
	}
	if cfg.Defaults.Method == "" {
		t.Error("starter config left method unset")
	}
}

// The committed hecato.example.yaml is a reference copy of what initconfig
// generates. Nothing stops the two drifting apart except this test, so it fails
// loudly and tells you how to regenerate rather than just reporting a mismatch.
func TestExampleConfigMatchesGenerator(t *testing.T) {
	const examplePath = "../../hecato.example.yaml"

	onDisk, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("reading %s: %v", examplePath, err)
	}

	// The repo checks out with CRLF on Windows, so compare on content rather
	// than line endings.
	got := strings.ReplaceAll(string(onDisk), "\r\n", "\n")
	want := strings.ReplaceAll(string(Starter()), "\r\n", "\n")

	if got != want {
		t.Errorf("%s is out of date with config.Starter().\n"+
			"Regenerate it with:\n"+
			"  rm hecato.example.yaml && ./hecato -method=initconfig -config=hecato.example.yaml",
			examplePath)
	}
}

func TestWriteStarterRefusesToClobber(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	if err := WriteStarter(path); err != nil {
		t.Fatalf("first WriteStarter failed: %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written config: %v", err)
	}

	if err := WriteStarter(path); err == nil {
		t.Fatal("second WriteStarter overwrote an existing config, want an error")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-reading config: %v", err)
	}
	if string(before) != string(after) {
		t.Error("the refused write still modified the file")
	}
}

func TestDefaultPathSitsBesideTheExecutable(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath returned error: %v", err)
	}
	if filepath.Base(path) != FileName {
		t.Errorf("DefaultPath base = %q, want %q", filepath.Base(path), FileName)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultPath = %q, want an absolute path", path)
	}
}
