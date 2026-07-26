package files

import (
	"sort"
	"strconv"

	"github.com/curiousjc/hecato/internal/ignore"
)

func GetModFiles(target string, hits string, ig *ignore.Matcher) (*Result, error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, err
	}

	res, err := getFiles(target, ig)
	if err != nil {
		return nil, err
	}

	sortFilesByMod(res.Files)
	res.Files = truncate(res.Files, hitsInt)

	return res, nil
}

// Sort files by modification time, most recent first
func sortFilesByMod(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})
}
