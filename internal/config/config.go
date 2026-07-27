// Package config loads hecato's optional configuration file.
//
// A config is never required. When none is found, Load returns a zero Config
// and no error, and hecato behaves exactly as it did before configs existed.
//
// Resolution order:
//
//  1. the path given to -config, which must exist or Load fails
//  2. hecato.yaml beside the executable
//  3. no config
//
// Resolving relative to the executable rather than the working directory means
// the same binary behaves the same way wherever it is invoked from, which is
// what you want for a tool that gets put on PATH.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/curiousjc/hecato/internal/ignore"
)

// FileName is the config file looked for beside the executable.
const FileName = "hecato.yaml"

// Config is the whole config file. Every field is optional.
type Config struct {
	Defaults Defaults `yaml:"defaults"`

	// Ignore holds glob patterns. See the ignore package for the full syntax;
	// briefly, a trailing "/" prunes a directory and a pattern without any "/"
	// matches a base name at any depth.
	Ignore []string `yaml:"ignore"`

	// IgnoreOSFiles adds the built-in list of operating-system artefacts, most
	// importantly pagefile.sys, which otherwise tops every largefiles run.
	IgnoreOSFiles bool `yaml:"ignore_os_files"`

	// Path records where this config came from, for reporting. It is not read
	// from the file itself.
	Path string `yaml:"-"`
}

// Defaults override hecato's built-in flag defaults. An explicitly supplied
// flag still wins; see the flag precedence handling in cmd/hecato.
type Defaults struct {
	Method  string `yaml:"method"`
	Target  string `yaml:"target"`
	Hits    int    `yaml:"hits"`
	Verbose bool   `yaml:"verbose"`

	// Workers is how many directories are read concurrently. Zero means use
	// the built-in default; see files.DefaultWorkers for why it is 8.
	Workers int `yaml:"workers"`
}

// Loaded reports whether a config file was actually found and read.
func (c *Config) Loaded() bool {
	return c != nil && c.Path != ""
}

// Matcher builds the ignore matcher this config describes, folding in the
// built-in OS file list when ignore_os_files is set.
func (c *Config) Matcher() *ignore.Matcher {
	if c == nil {
		return ignore.New(nil)
	}

	patterns := make([]string, 0, len(c.Ignore)+len(ignore.OSFiles))
	patterns = append(patterns, c.Ignore...)
	if c.IgnoreOSFiles {
		patterns = append(patterns, ignore.OSFiles...)
	}
	return ignore.New(patterns)
}

// DefaultPath returns the config path beside the executable.
func DefaultPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating the executable: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), FileName), nil
}

// Load reads the config. An explicit path is required to exist, on the grounds
// that being told to use a specific file and silently not using it is worse
// than failing. The default path is allowed to be absent.
func Load(explicitPath string) (*Config, error) {
	if explicitPath != "" {
		cfg, err := readFile(explicitPath)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	defaultPath, err := DefaultPath()
	if err != nil {
		// Not being able to locate our own executable is not a reason to refuse
		// to run; it just means there is no config to find.
		return &Config{}, nil
	}

	cfg, err := readFile(defaultPath)
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func readFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config %s: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)

	// Reject unknown fields. A typo in a config file that silently does nothing
	// is a bad afternoon; better to say so at startup.
	dec.KnownFields(true)

	// An empty file decodes to io.EOF. That is a valid, if pointless, config.
	if err := dec.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	cfg.Path = path
	return &cfg, nil
}
