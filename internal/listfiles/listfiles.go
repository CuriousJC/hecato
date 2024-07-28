package listfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

func GetLargeFiles(target string, hits string) (foundFiles []File, errorFiles []File, err error) {

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid value for hits: %v", err)
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				errorFiles = append(errorFiles, File{
					Path: path,
				})
				return nil
			}
			return err
		}
		if !info.IsDir() {
			foundFiles = append(foundFiles, File{
				Path: path,
				Size: info.Size(),
			})
		}
		return nil
	})
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
