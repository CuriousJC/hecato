package files

import (
	"time"
)

// Result is everything a scan produces: the files that matter, the paths that
// could not be read, and enough counts to say afterwards what actually happened.
//
// The counts exist so a run can report "scanned 68,361 files, ignored 412, 3
// unreadable" instead of printing a list and leaving you to guess whether
// anything was quietly dropped.
type Result struct {
	// Files is the answer, already ordered best first and no longer than the
	// requested hits.
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

	// Matched is how many files survived the ignore rules. The difference
	// between this and len(Files) is what -hits discarded.
	//
	// Note this is a count, not a length: the matched files are never all held
	// at once. Only the best `hits` of them are kept.
	Matched int

	// TotalBytes is the size of every matched file, summed as they were seen.
	// It is what makes a single result meaningful as a share rather than an
	// absolute: 14 GB means little until you know it is 3% of what was scanned.
	TotalBytes int64

	Elapsed time.Duration

	// top is the bounded collector the scan accumulated into. Retained so the
	// ordering used to build Files is available if a caller ever needs to merge
	// results; not part of the reported output.
	top *topN
}
