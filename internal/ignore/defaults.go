package ignore

// OSFiles are the operating-system artefacts that are almost never what you are
// looking for when investigating disk usage, but which reliably dominate a
// "largest files" result on Windows. pagefile.sys alone is routinely the biggest
// file on the volume.
//
// This list is applied only when ignore_os_files is turned on in the config, so
// running hecato without a config keeps its original behaviour.
var OSFiles = []string{
	// Windows memory and hibernation files, always at the volume root.
	"pagefile.sys",
	"swapfile.sys",
	"hiberfil.sys",
	"DumpStack.log",
	"DumpStack.log.tmp",

	// Volume-level directories that are either inaccessible or uninteresting.
	"$Recycle.Bin/",
	"System Volume Information/",
	"$WinREAgent/",
	"Recovery/",

	// Per-directory metadata droppings.
	"Thumbs.db",
	"desktop.ini",
	".DS_Store",
}

// StarterIgnore is what -method=initconfig writes into a new config. These are
// suggestions rather than defaults: they are commented in the generated file so
// that a new config changes nothing until it is deliberately edited.
//
// The Go entries are here because a Go toolchain rewrites an enormous number of
// files under the module cache and build cache, which swamps a modfiles run.
var StarterIgnore = []string{
	"# Go rewrites huge numbers of files under these, which swamps a modfiles run",
	"go/pkg/mod/",
	".cache/go-build/",

	"# Dependency and build trees",
	"node_modules/",
	"vendor/",
	".git/",

	"# Build output",
	"*.exe",
	"*.dll",
	"*.so",
}
