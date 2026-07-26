// Package ignore implements the pattern matching behind hecato's ignore rules.
//
// Pattern rules, in full:
//
//   - Paths are normalised to forward slashes before matching, and lowercased
//     on Windows so that patterns are case-insensitive there and case-sensitive
//     everywhere else. That matches how the underlying filesystems behave.
//
//   - A pattern ending in "/" is a directory pattern. When a directory matches,
//     the walk does not descend into it at all. This is the difference between
//     ignoring a large tree and merely hiding it: pruning costs nothing, while
//     filtering still pays to walk every file underneath.
//
//   - A pattern containing no "/" matches against the base name only, at any
//     depth. "*.tmp" matches a/b/c.tmp.
//
//   - A pattern containing "/" matches against the whole path. "c:/pagefile.sys"
//     matches only that file.
//
//   - A leading "**/" is stripped, because matching at any depth is already the
//     default for base-name patterns. It is accepted so that "**/node_modules"
//     reads the way people expect.
//
// Wildcards are those of path/filepath.Match: "*", "?" and "[...]". Note that
// "*" does not cross a path separator. Anchoring a pattern to the root of the
// scan, the way a leading "/" does in a gitignore, is not supported.
package ignore

import (
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// Matcher holds the compiled ignore rules for a single run. The zero value is
// not useful; build one with New. A nil *Matcher matches nothing, so callers
// that have no rules can pass nil rather than branching.
type Matcher struct {
	filePatterns []string
	dirPatterns  []string
}

// New compiles patterns into a Matcher. Blank entries and "#" comments are
// skipped so that a pattern list can be read straight out of a config file or a
// line-oriented ignore file without pre-cleaning.
func New(patterns []string) *Matcher {
	m := &Matcher{}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}

		p = normalise(p)
		p = strings.TrimPrefix(p, "**/")
		if p == "" {
			continue
		}

		if strings.HasSuffix(p, "/") {
			m.dirPatterns = append(m.dirPatterns, strings.TrimSuffix(p, "/"))
			continue
		}
		m.filePatterns = append(m.filePatterns, p)
	}
	return m
}

// Empty reports whether the Matcher has no rules, so callers can skip work
// entirely rather than testing every path against nothing.
func (m *Matcher) Empty() bool {
	return m == nil || (len(m.filePatterns) == 0 && len(m.dirPatterns) == 0)
}

// MatchDir reports whether a directory should be pruned from the walk.
func (m *Matcher) MatchDir(dirPath string) bool {
	if m == nil {
		return false
	}
	return matchAny(m.dirPatterns, dirPath)
}

// MatchFile reports whether a file should be left out of the results.
func (m *Matcher) MatchFile(filePath string) bool {
	if m == nil {
		return false
	}
	return matchAny(m.filePatterns, filePath)
}

// Patterns returns the compiled patterns, directory rules first with their
// trailing slash restored. Used for verbose reporting of what is in effect.
func (m *Matcher) Patterns() []string {
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m.dirPatterns)+len(m.filePatterns))
	for _, d := range m.dirPatterns {
		out = append(out, d+"/")
	}
	out = append(out, m.filePatterns...)
	return out
}

func matchAny(patterns []string, target string) bool {
	if len(patterns) == 0 {
		return false
	}

	full := normalise(target)
	base := path.Base(full)

	for _, p := range patterns {
		// A pattern with a separator addresses the whole path; one without it
		// addresses the base name, which is what makes it match at any depth.
		subject := base
		if strings.Contains(p, "/") {
			subject = full
		}

		if ok, err := path.Match(p, subject); err == nil && ok {
			return true
		}
	}
	return false
}

// normalise makes a path or pattern comparable: forward slashes throughout, and
// lowercased on Windows where the filesystem is case-insensitive. Note that
// runtime.GOOS is the *target* of the build, so a cross-compiled linux binary
// correctly stays case-sensitive.
func normalise(s string) string {
	s = filepath.ToSlash(s)
	if runtime.GOOS == "windows" {
		s = strings.ToLower(s)
	}
	return s
}
