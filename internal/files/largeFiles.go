package files

import (
	"strconv"

	"github.com/curiousjc/hecato/internal/ignore"
)

// worseBySize reports whether a is the weaker candidate for "largest file",
// which is to say the smaller of the two.
func worseBySize(a, b File) bool { return a.Size < b.Size }

func GetLargeFiles(target string, hits string, ig *ignore.Matcher, workers int) (*Result, error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, err
	}

	return scan(target, ig, workers, hitsInt, worseBySize)
}
