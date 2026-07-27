package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/curiousjc/hecato/internal/ignore"
)

// buildWideTree makes a fixture big enough that the workers genuinely contend:
// several directories, each with files, plus nesting. A five-file tree is
// finished by one worker before the others wake up and proves nothing.
func buildWideTree(t *testing.T, dirs, filesPerDir int) string {
	t.Helper()
	root := t.TempDir()

	for d := 0; d < dirs; d++ {
		dir := filepath.Join(root, fmt.Sprintf("dir%02d", d), "nested")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
		for f := 0; f < filesPerDir; f++ {
			// Sizes are distinct and ascending so ordering assertions have
			// something unambiguous to check.
			name := filepath.Join(dir, fmt.Sprintf("f%03d.dat", f))
			size := d*filesPerDir + f + 1
			if err := os.WriteFile(name, make([]byte, size), 0644); err != nil {
				t.Fatalf("creating %s: %v", name, err)
			}
		}
	}
	return root
}

// serialWalk is a deliberately simple reference implementation. The concurrent
// scan must agree with it exactly, which is the only assertion that really
// matters after a rewrite this invasive.
func serialWalk(t *testing.T, root string, ig *ignore.Matcher) (files []File, scanned, ignored, matched int, total int64) {
	t.Helper()

	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() {
				if !ig.MatchDir(path) {
					walk(path)
				}
				continue
			}
			scanned++
			if ig.MatchFile(path) {
				ignored++
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			matched++
			total += info.Size()
			files = append(files, File{Path: path, Size: info.Size(), ModTime: info.ModTime()})
		}
	}
	walk(filepath.Clean(root))

	sort.Slice(files, func(i, j int) bool { return files[i].Size > files[j].Size })
	return files, scanned, ignored, matched, total
}

// Run with -race to make this meaningful. Worker counts vary so the scan is
// exercised single-threaded and heavily contended by the same assertions.
func TestScanMatchesSerialWalk(t *testing.T) {
	root := buildWideTree(t, 12, 20)
	ig := ignore.New([]string{"f00[0-4].dat", "nested/"})

	// Deliberately no ignore rules that prune, since the reference walk and the
	// scan must see the same tree; "nested/" above prunes in both.
	wantFiles, wantScanned, wantIgnored, wantMatched, wantTotal := serialWalk(t, root, ig)

	for _, workers := range []int{1, 2, 8, 32} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			res, err := scan(root, ig, workers, 1000, worseBySize)
			if err != nil {
				t.Fatalf("scan returned error: %v", err)
			}

			if res.Scanned != wantScanned {
				t.Errorf("Scanned = %d, want %d", res.Scanned, wantScanned)
			}
			if res.Ignored != wantIgnored {
				t.Errorf("Ignored = %d, want %d", res.Ignored, wantIgnored)
			}
			if res.Matched != wantMatched {
				t.Errorf("Matched = %d, want %d", res.Matched, wantMatched)
			}
			if res.TotalBytes != wantTotal {
				t.Errorf("TotalBytes = %d, want %d", res.TotalBytes, wantTotal)
			}
			if len(res.Files) != len(wantFiles) {
				t.Fatalf("got %d files, want %d", len(res.Files), len(wantFiles))
			}
			for i := range wantFiles {
				if res.Files[i].Size != wantFiles[i].Size {
					t.Errorf("file %d size = %d, want %d", i, res.Files[i].Size, wantFiles[i].Size)
				}
			}
		})
	}
}

// The whole point of the bounded collector: the answer must not depend on how
// many workers happened to see which files.
func TestScanTopNIsIndependentOfWorkerCount(t *testing.T) {
	root := buildWideTree(t, 12, 20)

	var reference []int64
	for _, workers := range []int{1, 3, 8, 16} {
		res, err := scan(root, nil, workers, 7, worseBySize)
		if err != nil {
			t.Fatalf("scan returned error: %v", err)
		}
		if len(res.Files) != 7 {
			t.Fatalf("workers=%d returned %d files, want 7", workers, len(res.Files))
		}

		sizes := make([]int64, len(res.Files))
		for i, f := range res.Files {
			sizes[i] = f.Size
		}

		if reference == nil {
			reference = sizes
			continue
		}
		for i := range reference {
			if sizes[i] != reference[i] {
				t.Errorf("workers=%d gave sizes %v, want %v", workers, sizes, reference)
				break
			}
		}
	}
}

func TestScanRootIsNeverPruned(t *testing.T) {
	root := buildWideTree(t, 2, 3)
	pattern := filepath.Base(root) + "/"

	res, err := scan(root, ignore.New([]string{pattern}), 4, 100, worseBySize)
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}
	if len(res.Files) == 0 {
		t.Error("a pattern matching the scan root emptied the results")
	}
}

// A directory that cannot be read must be recorded and skipped, not fatal, and
// must not stall the queue: pending has to be decremented on the error path too
// or the walk would never terminate.
func TestScanUnreadableDirectoryDoesNotStall(t *testing.T) {
	root := buildWideTree(t, 3, 4)

	// Point at a directory that does not exist. The walk should end promptly
	// with the failure recorded rather than blocking forever.
	missing := filepath.Join(root, "nope")
	res, err := scan(missing, nil, 4, 10, worseBySize)
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}
	if len(res.Errors) == 0 {
		t.Error("an unreadable root produced no recorded error")
	}
	if len(res.Files) != 0 {
		t.Errorf("an unreadable root produced %d files", len(res.Files))
	}
}

func TestScanEmptyDirectory(t *testing.T) {
	res, err := scan(t.TempDir(), nil, 4, 10, worseBySize)
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}
	if len(res.Files) != 0 || res.Scanned != 0 {
		t.Errorf("empty directory produced %d files, %d scanned", len(res.Files), res.Scanned)
	}
}

// Zero workers means "use the default" rather than "do nothing", which would
// otherwise hang forever with no goroutines to drain the queue.
func TestScanZeroWorkersUsesDefault(t *testing.T) {
	root := buildWideTree(t, 2, 3)

	res, err := scan(root, nil, 0, 10, worseBySize)
	if err != nil {
		t.Fatalf("scan returned error: %v", err)
	}
	if len(res.Files) == 0 {
		t.Error("scan with workers=0 returned nothing")
	}
}
