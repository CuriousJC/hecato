package files

import (
	"strconv"

	"github.com/curiousjc/hecato/internal/ignore"
)

// worseByMod reports whether a is the weaker candidate for "most recently
// modified", which is to say the older of the two.
func worseByMod(a, b File) bool { return a.ModTime.Before(b.ModTime) }

func GetModFiles(target string, hits string, ig *ignore.Matcher, workers int) (*Result, error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, err
	}

	return scan(target, ig, workers, hitsInt, worseByMod)
}
