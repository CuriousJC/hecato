package files

import (
	"sort"
	"strconv"

	"github.com/curiousjc/hecato/internal/ignore"
)

func GetModFiles(target string, hits string, ig *ignore.Matcher) (foundFiles []File, errorFiles []File, err error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, nil, err
	}

	foundFiles, errorFiles, err = getFiles(target, ig)
	if err != nil {
		return nil, nil, err
	}

	sortFilesByMod(foundFiles)

	//handle if there are fewer files than desired hits
	if len(foundFiles) < hitsInt {
		hitsInt = len(foundFiles)
	}

	return foundFiles[:hitsInt], errorFiles, nil
}

// Sort files by size in descending order
func sortFilesByMod(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})
}
