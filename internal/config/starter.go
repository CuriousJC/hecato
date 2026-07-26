package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/curiousjc/hecato/internal/ignore"
)

// starterTemplate is the config written by -method=initconfig.
//
// The defaults block and ignore_os_files ship active, because a config whose
// every line is commented out looks like it is configuring something and is not.
// The ignore list stays commented: those are suggestions that depend on what is
// actually on the machine, whereas a target and a hit count are needed by every
// run. Without an active target, hecato walks the literal path "undefined",
// finds nothing, and says nothing unless you pass -verbose.
const starterTemplate = `# hecato configuration
#
# Looked for beside the hecato executable, or passed explicitly with -config.
# Delete this file to go back to built-in defaults.

# Override the built-in flag defaults. A flag you actually type still wins.
#
# Note that setting method here means a bare "hecato" with no arguments runs it,
# rather than printing the examples. Comment the method line out if you would
# rather keep the examples as the no-argument behaviour.
defaults:
  method: largefiles
  target: "c:/"
  hits: 15
  verbose: false

# Skip the operating-system files that otherwise dominate a largefiles run,
# pagefile.sys above all. The built-in list also covers hiberfil.sys,
# swapfile.sys, $Recycle.Bin and friends.
ignore_os_files: true

# Ignore patterns.
#
#   *.tmp           matches that base name at any depth
#   node_modules/   trailing slash prunes the directory, so the walk never
#                   descends into it -- much faster than filtering it out
#   c:/pagefile.sys a pattern containing / is matched against the whole path
#
# Wildcards are *, ? and [...]; * does not cross a path separator.
# Matching is case-insensitive on Windows and case-sensitive elsewhere.
ignore:
%s
`

// Starter returns the contents of a new config file. The ignore entries are
// emitted commented out, so a generated config is immediately usable without
// silently excluding anything the user did not ask to exclude.
func Starter() []byte {
	var b strings.Builder

	for _, entry := range ignore.StarterIgnore {
		if strings.HasPrefix(entry, "#") {
			// Section headings pass through as comments, with a blank line
			// before them so the generated file is readable.
			fmt.Fprintf(&b, "\n  %s\n", entry)
			continue
		}
		// Suggestions ship commented out; the user opts in by uncommenting.
		fmt.Fprintf(&b, "  #- %q\n", entry)
	}

	return []byte(fmt.Sprintf(starterTemplate, strings.TrimRight(b.String(), "\n")))
}

// WriteStarter writes a new config file, refusing to clobber an existing one.
func WriteStarter(path string) error {
	// O_EXCL makes the "does it already exist" check atomic rather than a
	// racy stat-then-write.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("config already exists at %s, not overwriting it", path)
		}
		return fmt.Errorf("creating config %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(Starter()); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}
