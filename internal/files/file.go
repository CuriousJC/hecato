package files

import (
	"fmt"
	"time"
)

type File struct {
	Path       string
	Size       int64
	CreateTime time.Time
	ModTime    time.Time
}

// SizeInMB returns the size of the file in megabytes.
//
// Kept because it is an honest answer to a specific question, but note that it
// reports "0.00 MB" for anything under about 5 KB. HumanSize is what you want
// for display.
func (df File) SizeInMB() string {
	sizeInMB := float64(df.Size) / (1024 * 1024)
	return fmt.Sprintf("%.2f MB", sizeInMB)
}

// HumanSize picks a unit that suits the number, so a 400-byte file reads as
// "400 B" rather than "0.00 MB".
func (df File) HumanSize() string {
	return HumanBytes(df.Size)
}

// HumanBytes formats a byte count with a unit that keeps it readable. Binary
// units, so 1 KB is 1024 bytes, matching what Windows reports.
func HumanBytes(n int64) string {
	const unit = 1024

	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for x := n / unit; x >= unit && exp < 4; x /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %cB", float64(n)/float64(div), "KMGTP"[exp])
}
