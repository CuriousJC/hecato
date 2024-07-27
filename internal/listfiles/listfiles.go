package listfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

func GetLargeFiles(target string, hits string) ([]File, error) {
	var files []File

	// Convert hits from string to int
	hitsInt, err := strconv.Atoi(hits)
	if err != nil {
		return nil, fmt.Errorf("invalid value for hits: %v", err)
	}

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error encountered but continuing...\n", err)
		}
		if !info.IsDir() {
			files = append(files, File{
				Path: path,
				Size: info.Size(),
			})
		}
		return nil
	})
	if err != nil {
		//TODO: check for an Access is denied and throw that into a "couldn't read" files slice, otherwise...
		return nil, err
	}

	sortFilesBySize(files)

	//handle if there are fewer files than desired hits
	if len(files) < hitsInt {
		hitsInt = len(files)
	}

	return files[:hitsInt], nil
}

// Sort files by size in descending order
func sortFilesBySize(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
}
