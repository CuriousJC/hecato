package files

import (
	"sort"
	"strconv"

	"github.com/curiousjc/hecato/internal/ignore"
)

func GetLargeFiles(target string, hits string, ig *ignore.Matcher) (*Result, error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, err
	}

	res, err := getFiles(target, ig)
	if err != nil {
		return nil, err
	}

	sortFilesBySize(res.Files)
	res.Files = truncate(res.Files, hitsInt)

	return res, nil
}

// Sort files by size in descending order
func sortFilesBySize(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
}

// truncate caps the results at the requested hits, handling the case where
// fewer files were found than asked for.
func truncate(files []File, hits int) []File {
	if hits < 0 {
		hits = 0
	}
	if len(files) < hits {
		return files
	}
	return files[:hits]
}
