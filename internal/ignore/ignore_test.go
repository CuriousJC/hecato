package ignore

import (
	"runtime"
	"testing"
)

func TestMatchFile(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"base name glob matches at depth", []string{"*.tmp"}, "a/b/c.tmp", true},
		{"base name glob rejects other extension", []string{"*.tmp"}, "a/b/c.txt", false},
		{"literal base name", []string{"Thumbs.db"}, "some/dir/Thumbs.db", true},
		{"full path pattern matches", []string{"c:/pagefile.sys"}, "c:/pagefile.sys", true},
		{"full path pattern is anchored", []string{"c:/pagefile.sys"}, "d:/other/c:/pagefile.sys", false},
		{"star does not cross a separator", []string{"a/*.tmp"}, "a/b/c.tmp", false},
		{"star matches within one segment", []string{"a/*.tmp"}, "a/c.tmp", true},
		{"leading doublestar is stripped", []string{"**/node_modules"}, "x/y/node_modules", true},
		{"question mark wildcard", []string{"file?.log"}, "dir/file1.log", true},
		{"character class", []string{"file[0-9].log"}, "dir/file7.log", true},
		{"character class rejects outside range", []string{"file[0-9].log"}, "dir/filex.log", false},
		{"no patterns matches nothing", nil, "anything", false},
		{"blank entries are skipped", []string{"", "   "}, "anything", false},
		{"comments are skipped", []string{"# *.tmp"}, "a.tmp", false},
		{"directory pattern does not match files", []string{"build/"}, "build", false},
		{"backslash input is normalised", []string{"*.tmp"}, `a\b\c.tmp`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.patterns).MatchFile(tt.path); got != tt.want {
				t.Errorf("MatchFile(%q) with patterns %q = %v, want %v",
					tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestMatchDir(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"directory pattern matches at any depth", []string{"node_modules/"}, "a/b/node_modules", true},
		{"directory pattern with path matches", []string{"go/pkg/mod/"}, "go/pkg/mod", true},
		{"directory pattern with path is anchored", []string{"go/pkg/mod/"}, "home/go/pkg/mod", false},
		{"file pattern does not prune directories", []string{"*.tmp"}, "a/b.tmp", false},
		{"glob in directory pattern", []string{"tmp*/"}, "a/tmpdir", true},
		{"no patterns prunes nothing", nil, "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.patterns).MatchDir(tt.path); got != tt.want {
				t.Errorf("MatchDir(%q) with patterns %q = %v, want %v",
					tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

// Case sensitivity follows the target platform, so the expectation flips with
// GOOS. A cross-compiled linux binary stays case-sensitive even when it was
// built on Windows.
func TestCaseSensitivityFollowsPlatform(t *testing.T) {
	m := New([]string{"Thumbs.db"})
	got := m.MatchFile("dir/THUMBS.DB")
	want := runtime.GOOS == "windows"

	if got != want {
		t.Errorf("MatchFile on differing case = %v, want %v on %s", got, want, runtime.GOOS)
	}
}

func TestNilMatcherIsSafe(t *testing.T) {
	var m *Matcher

	if m.MatchFile("a/b.tmp") {
		t.Error("nil Matcher matched a file, want no match")
	}
	if m.MatchDir("a/b") {
		t.Error("nil Matcher matched a directory, want no match")
	}
	if !m.Empty() {
		t.Error("nil Matcher reported non-empty")
	}
	if got := m.Patterns(); got != nil {
		t.Errorf("nil Matcher returned patterns %q, want nil", got)
	}
}

func TestEmpty(t *testing.T) {
	if !New(nil).Empty() {
		t.Error("New(nil) reported non-empty")
	}
	if !New([]string{"", "# comment"}).Empty() {
		t.Error("matcher of only blanks and comments reported non-empty")
	}
	if New([]string{"*.tmp"}).Empty() {
		t.Error("matcher with a pattern reported empty")
	}
}

func TestPatternsRoundTrip(t *testing.T) {
	m := New([]string{"*.tmp", "build/"})
	got := m.Patterns()

	// Directory patterns come first and keep their trailing slash.
	want := []string{"build/", "*.tmp"}
	if len(got) != len(want) {
		t.Fatalf("Patterns() = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Patterns()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
