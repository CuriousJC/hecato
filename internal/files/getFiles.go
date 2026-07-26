package files

import (
	"os"
	"path/filepath"

	"github.com/curiousjc/hecato/internal/ignore"
)

// getFiles walks target and returns every file under it, along with the paths
// that could not be read.
//
// A nil or empty matcher means no filtering. When a directory matches, the walk
// returns filepath.SkipDir rather than filtering its contents afterwards, so an
// ignored tree costs nothing instead of being walked and discarded.
func getFiles(target string, ig *ignore.Matcher) (foundFiles []File, errorFiles []File, err error) {
	// Keep the root itself out of the pruning check. Ignoring the thing you
	// were asked to scan would silently return nothing.
	root := filepath.Clean(target)

	err = filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			//intentionally capture the error but don't panic
			errorFiles = append(errorFiles, File{Path: path})
			return nil
		}

		if info.IsDir() {
			if filepath.Clean(path) != root && ig.MatchDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if ig.MatchFile(path) {
			return nil
		}

		foundFiles = append(foundFiles, File{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return foundFiles, errorFiles, nil
}
