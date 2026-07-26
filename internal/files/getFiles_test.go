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
	found, _, err := getFiles(buildTree(t), nil)
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	got := baseNames(found)
	for _, want := range []string{"keep.txt", "skip.tmp", "dep.txt", "main.go", "gen.tmp"} {
		if !got[want] {
			t.Errorf("%s missing from an unfiltered walk", want)
		}
	}
}

func TestGetFilesFiltersByPattern(t *testing.T) {
	found, _, err := getFiles(buildTree(t), ignore.New([]string{"*.tmp"}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	got := baseNames(found)
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
	found, _, err := getFiles(buildTree(t), ignore.New([]string{"node_modules/"}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}

	got := baseNames(found)
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

	found, _, err := getFiles(root, ignore.New([]string{pattern}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}
	if len(found) == 0 {
		t.Fatalf("a pattern matching the scan root emptied the results")
	}
}

func TestGetFilesTrailingSeparatorRootStillScans(t *testing.T) {
	root := buildTree(t)
	pattern := filepath.Base(root) + "/"

	found, _, err := getFiles(root+string(filepath.Separator), ignore.New([]string{pattern}))
	if err != nil {
		t.Fatalf("getFiles returned error: %v", err)
	}
	if len(found) == 0 {
		t.Error("a trailing separator on the target defeated the root exemption")
	}
}

func TestGetLargeFilesRespectsIgnore(t *testing.T) {
	root := buildTree(t)

	found, _, err := GetLargeFiles(root, "10", ignore.New([]string{"*.tmp"}))
	if err != nil {
		t.Fatalf("GetLargeFiles returned error: %v", err)
	}
	for _, f := range found {
		if strings.HasSuffix(f.Path, ".tmp") {
			t.Errorf("GetLargeFiles returned an ignored file: %s", f.Path)
		}
	}
}

func TestGetModFilesRespectsIgnore(t *testing.T) {
	root := buildTree(t)

	found, _, err := GetModFiles(root, "10", ignore.New([]string{"node_modules/"}))
	if err != nil {
		t.Fatalf("GetModFiles returned error: %v", err)
	}
	for _, f := range found {
		if strings.Contains(filepath.ToSlash(f.Path), "/node_modules/") {
			t.Errorf("GetModFiles descended into a pruned directory: %s", f.Path)
		}
	}
}

func TestHitsTruncatesButIgnoreAppliesFirst(t *testing.T) {
	root := buildTree(t)

	// Five files exist, two are .tmp. Asking for 10 should yield the three
	// survivors, proving the filter runs before the truncation.
	found, _, err := GetLargeFiles(root, "10", ignore.New([]string{"*.tmp"}))
	if err != nil {
		t.Fatalf("GetLargeFiles returned error: %v", err)
	}
	if len(found) != 3 {
		t.Errorf("got %d files, want 3 after ignoring *.tmp", len(found))
	}
}
