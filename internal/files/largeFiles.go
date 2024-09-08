package files

import (
	"sort"
	"strconv"
)

func GetLargeFiles(target string, hits string) (foundFiles []File, errorFiles []File, err error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, nil, err
	}

	foundFiles, errorFiles, err = getFiles(target)
	if err != nil {
		return nil, nil, err
	}

	sortFilesBySize(foundFiles)

	//handle if there are fewer files than desired hits
	if len(foundFiles) < hitsInt {
		hitsInt = len(foundFiles)
	}

	return foundFiles[:hitsInt], errorFiles, nil
}

// Sort files by size in descending order
func sortFilesBySize(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
}
