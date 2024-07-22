package listfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func GetLargeFiles(root string) ([]File, error) {
	var files []File

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
		return nil, err
	}

	sortFilesBySize(files)

	// Limit the result to the first 10 files
	topN := 10
	if len(files) < topN {
		topN = len(files)
	}

	return files[:topN], nil
}

// Sort files by size in descending order
func sortFilesBySize(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
}
