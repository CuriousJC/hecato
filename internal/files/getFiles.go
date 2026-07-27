package files

import (
	"io/fs"
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
// returns fs.SkipDir rather than filtering its contents afterwards, so an
// ignored tree costs nothing instead of being walked and discarded.
//
// This uses filepath.WalkDir rather than filepath.Walk. Walk calls os.Lstat on
// every entry it visits; WalkDir hands back the fs.DirEntry that the directory
// read already produced, and only pays for metadata when Info() is called. On
// Windows that metadata arrives with the FindNextFile enumeration, so Info() is
// effectively free and the saving is the entire lstat per file -- measured at
// 6.6x on C:/Program Files. On linux getdents does not return size or mtime, so
// Info() still costs a stat there and the gain is smaller. Either way it is
// never worse, because Walk was making exactly that call anyway.
func getFiles(target string, ig *ignore.Matcher) (*Result, error) {
	started := time.Now()
	res := &Result{}

	// Keep the root itself out of the pruning check. Ignoring the thing you
	// were asked to scan would silently return nothing.
	root := filepath.Clean(target)

	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			//intentionally capture the error but don't panic
			res.Errors = append(res.Errors, File{Path: path})
			return nil
		}

		if d.IsDir() {
			if filepath.Clean(path) != root && ig.MatchDir(path) {
				res.Pruned++
				return fs.SkipDir
			}
			return nil
		}

		res.Scanned++

		if ig.MatchFile(path) {
			res.Ignored++
			return nil
		}

		// Deferred until after the ignore check, so an ignored file costs
		// nothing beyond the pattern match. Info can still fail -- a file
		// deleted between the directory read and this call, or a broken
		// symlink -- which is the same class of problem as an unreadable path
		// and is recorded the same way.
		info, err := d.Info()
		if err != nil {
			res.Errors = append(res.Errors, File{Path: path})
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
