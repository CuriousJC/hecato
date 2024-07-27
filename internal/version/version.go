package version

import (
	"fmt"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "none"
)

// Print prints the version information.
func Print() {
	fmt.Printf("Version: %s\nBuild Time: %s\nCommit: %s\n", Version, BuildTime, Commit)
}
