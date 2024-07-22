package listfiles

import (
	"fmt"
)

type File struct {
	Path string
	Size int64
}

// SizeInMB returns the size of the file in megabytes.
func (df File) SizeInMB() string {
	sizeInMB := float64(df.Size) / (1024 * 1024)
	return fmt.Sprintf("%.2f MB", sizeInMB)
}
