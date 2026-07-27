package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curiousjc/hecato/internal/ignore"
)

// buildTree lays out a small fixture:
//
//	root/keep.txt
//	root/skip.tmp
//	root/node_modules/dep.txt
//	root/src/main.go
//	root/src/gen.tmp
func buildTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{"node_modules", "src"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("creating %s: %v", d, err)
		}
	}

	files := []string{
		"keep.txt",
		"skip.tmp",
		filepath.Join("node_modules", "dep.txt"),
		filepath.Join("src", "main.go"),
		filepath.Join("src", "gen.tmp"),
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0644); err != nil {
			t.Fatalf("creating %s: %v", f, err)
		}
	}
	return root
}

// baseNames reduces results to base names so assertions do not depend on the
// temp directory path.
func baseNames(found []File) map[string]bool {
	out := make(map[string]bool, len(found))
	for _, f := range found {
		out[filepath.Base(f.Path)] = true
	}
	return out
}

func TestGetFilesNoMatcherReturnsEverything(t *testing.T) {
	res, err := getFiles(buildTree(t), nil)
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	got := baseNames(res.Files)
	for _, want := range []string{"keep.txt", "skip.tmp", "dep.txt", "main.go", "gen.tmp"} {
		if !got[want] {
			t.Errorf("%s missing from an unfiltered walk", want)
		}
	}
}

func TestGetFilesFiltersByPattern(t *testing.T) {
	res, err := getFiles(buildTree(t), ignore.New([]string{"*.tmp"}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	got := baseNames(res.Files)
	if got["skip.tmp"] || got["gen.tmp"] {
		t.Error("*.tmp files survived the filter")
	}
	if !got["keep.txt"] || !got["main.go"] {
		t.Error("the filter removed files it should have kept")
	}
}

// Note: from outside the package, a pruned directory and a filtered-out
// directory look identical. This asserts the observable result; the pruning
// itself is the filepath.SkipDir return in getFiles.
func TestGetFilesPrunesDirectories(t *testing.T) {
	res, err := getFiles(buildTree(t), ignore.New([]string{"node_modules/"}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	got := baseNames(res.Files)
	if got["dep.txt"] {
		t.Error("a file under an ignored directory was returned")
	}
	if !got["main.go"] || !got["keep.txt"] {
		t.Error("pruning one directory removed files elsewhere")
	}
}

// Ignoring the directory you were told to scan would silently return nothing,
// which is worse than useless. The root is exempt from pruning.
func TestGetFilesNeverPrunesTheRoot(t *testing.T) {
	root := buildTree(t)
	pattern := filepath.Base(root) + "/"

	res, err := getFiles(root, ignore.New([]string{pattern}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}
	if len(res.Files) == 0 {
		t.Fatalf("a pattern matching the scan root emptied the results")
	}
}

func TestGetFilesTrailingSeparatorRootStillScans(t *testing.T) {
	root := buildTree(t)
	pattern := filepath.Base(root) + "/"

	res, err := getFiles(root+string(filepath.Separator), ignore.New([]string{pattern}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}
	if len(res.Files) == 0 {
		t.Error("a trailing separator on the target defeated the root exemption")
	}
}

func TestGetLargeFilesRespectsIgnore(t *testing.T) {
	root := buildTree(t)

	res, err := GetLargeFiles(root, "10", ignore.New([]string{"*.tmp"}))
	if err != nil {
		t.Fatalf("GetLargeFiles returned error: %v", err)
	}
	for _, f := range res.Files {
		if strings.HasSuffix(f.Path, ".tmp") {
			t.Errorf("GetLargeFiles returned an ignored file: %s", f.Path)
		}
	}
}

func TestGetModFilesRespectsIgnore(t *testing.T) {
	root := buildTree(t)

	res, err := GetModFiles(root, "10", ignore.New([]string{"node_modules/"}))
	if err != nil {
		t.Fatalf("GetModFiles returned error: %v", err)
	}
	for _, f := range res.Files {
		if strings.Contains(filepath.ToSlash(f.Path), "/node_modules/") {
			t.Errorf("GetModFiles descended into a pruned directory: %s", f.Path)
		}
	}
}

func TestHitsTruncatesButIgnoreAppliesFirst(t *testing.T) {
	root := buildTree(t)

	// Five files exist, two are .tmp. Asking for 10 should yield the three
	// survivors, proving the filter runs before the truncation.
	res, err := GetLargeFiles(root, "10", ignore.New([]string{"*.tmp"}))
	if err != nil {
		t.Fatalf("GetLargeFiles returned error: %v", err)
	}
	if len(res.Files) != 3 {
		t.Errorf("got %d files, want 3 after ignoring *.tmp", len(res.Files))
	}
}

func TestResultCountsScannedIgnoredAndPruned(t *testing.T) {
	// The fixture is 5 files: keep.txt, skip.tmp, node_modules/dep.txt,
	// src/main.go, src/gen.tmp.
	root := buildTree(t)

	res, err := getFiles(root, ignore.New([]string{"*.tmp", "node_modules/"}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	// dep.txt is never scanned, because node_modules is pruned before descent.
	// That is the whole point of pruning, and it is why Scanned is 4 not 5.
	if res.Scanned != 4 {
		t.Errorf("Scanned = %d, want 4 (5 files less the one under a pruned dir)", res.Scanned)
	}
	if res.Ignored != 2 {
		t.Errorf("Ignored = %d, want 2 (skip.tmp and gen.tmp)", res.Ignored)
	}
	if res.Pruned != 1 {
		t.Errorf("Pruned = %d, want 1 (node_modules)", res.Pruned)
	}
	if res.Matched != 2 {
		t.Errorf("Matched = %d, want 2 (keep.txt and main.go)", res.Matched)
	}
	// Only that it is non-negative. Asserting a positive duration would be
	// flaky: on Windows the monotonic clock is coarser than a five-file
	// WalkDir, so time.Since legitimately returns exactly zero. This assertion
	// used to pass only because filepath.Walk's per-file lstat calls were slow
	// enough to tick the clock.
	if res.Elapsed < 0 {
		t.Errorf("Elapsed = %v, want a non-negative duration", res.Elapsed)
	}
}

func TestResultMatchedExceedsShownWhenHitsTruncates(t *testing.T) {
	root := buildTree(t)

	res, err := GetLargeFiles(root, "2", nil)
	if err != nil {
		t.Fatalf("GetLargeFiles returned error: %v", err)
	}

	if len(res.Files) != 2 {
		t.Errorf("returned %d files, want 2", len(res.Files))
	}
	// Matched records the pre-truncation count, so a run can say "5 matched,
	// showing the top 2" rather than implying only 2 existed.
	if res.Matched != 5 {
		t.Errorf("Matched = %d, want 5", res.Matched)
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{999, "999 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{5 * 1024 * 1024 * 1024, "5.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tt := range tests {
		if got := HumanBytes(tt.n); got != tt.want {
			t.Errorf("HumanBytes(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// The old SizeInMB reported "0.00 MB" for a small file, which is technically
// true and practically useless. This is the behaviour HumanSize replaces.
func TestHumanSizeBeatsSizeInMBForSmallFiles(t *testing.T) {
	f := File{Size: 400}

	if got := f.SizeInMB(); got != "0.00 MB" {
		t.Errorf("SizeInMB() = %q, want 0.00 MB (documenting the old behaviour)", got)
	}
	if got := f.HumanSize(); got != "400 B" {
		t.Errorf("HumanSize() = %q, want 400 B", got)
	}
}
