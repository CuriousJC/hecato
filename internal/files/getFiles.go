package files

import (
	"os"
	"path/filepath"
	"time"

	"github.com/curiousjc/hecato/internal/ignore"
)

// Result is everything a scan produces: the files that matter, the paths that
// could not be read, and enough counts to say afterwards what actually happened.
//
// The counts exist so a run can report "scanned 68,361 files, ignored 412, 3
// unreadable" instead of printing a list and leaving you to guess whether
// anything was quietly dropped.
type Result struct {
	// Files is the answer, already sorted and truncated to the requested hits.
	Files []File

	// Errors holds paths the walk could not read. These are collected rather
	// than fatal, so one unreadable directory does not abort a scan of c:/.
	Errors []File

	// Scanned counts every file the walk looked at, before ignore rules.
	Scanned int

	// Ignored counts files excluded by an ignore pattern.
	Ignored int

	// Pruned counts directories the walk refused to descend into. Each one
	// stands for an unknown number of files never looked at, which is why it is
	// reported separately from Ignored rather than added to it.
	Pruned int

	// Matched is how many files survived the ignore rules, before truncation to
	// hits. The difference between this and len(Files) is what -hits discarded.
	Matched int

	// TotalBytes is the size of every matched file, summed before truncation.
	// It is what makes a single result meaningful as a share rather than an
	// absolute: 14 GB means little until you know it is 3% of what was scanned.
	TotalBytes int64

	Elapsed time.Duration
}

// getFiles walks target and returns every file under it, along with the paths
// that could not be read and the counts describing the walk.
//
// A nil or empty matcher means no filtering. When a directory matches, the walk
// returns filepath.SkipDir rather than filtering its contents afterwards, so an
// ignored tree costs nothing instead of being walked and discarded.
func getFiles(target string, ig *ignore.Matcher) (*Result, error) {
	started := time.Now()
	res := &Result{}

	// Keep the root itself out of the pruning check. Ignoring the thing you
	// were asked to scan would silently return nothing.
	root := filepath.Clean(target)

	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//intentionally capture the error but don't panic
			res.Errors = append(res.Errors, File{Path: path})
			return nil
		}

		if info.IsDir() {
			if filepath.Clean(path) != root && ig.MatchDir(path) {
				res.Pruned++
				return filepath.SkipDir
			}
			return nil
		}

		res.Scanned++

		if ig.MatchFile(path) {
			res.Ignored++
			return nil
		}

		res.TotalBytes += info.Size()
		res.Files = append(res.Files, File{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	res.Matched = len(res.Files)
	res.Elapsed = time.Since(started)

	return res, nil
}
