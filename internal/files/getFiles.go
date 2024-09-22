package files

import (
	"os"
	"path/filepath"
)

func getFiles(target string) (foundFiles []File, errorFiles []File, err error) {

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//intentionally capture the error but don't panic
			errorFiles = append(errorFiles, File{Path: path})
			return nil
		}
		if !info.IsDir() {
			foundFiles = append(foundFiles, File{
				Path:    path,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return foundFiles, errorFiles, nil
}
