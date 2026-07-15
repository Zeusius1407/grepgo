package cmd

import (
	"runtime/debug"
	"strings"
)

// buildVersion derives the version string from the module's build metadata,
// which the Go toolchain embeds automatically from the Git repository — no
// -ldflags needed. When the binary is installed from a tagged release (e.g.
// `go install .../mygrep@v0.1.0`) it reports that tag; a local build reports
// the short Git commit it was built from, with "-dirty" if the working tree
// had uncommitted changes.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	var revision, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if len(revision) > 12 {
		revision = revision[:12] // short commit hash
	}
	dirty := modified == "true"

	// A clean release tag is the nicest thing to report. For untagged VCS
	// builds Go synthesizes a "v0.0.0-<time>-<commit>" pseudo-version, and for
	// plain local builds it uses "(devel)"; in both cases fall back to the raw
	// commit so we don't print a redundant, noisy string.
	//
	// Go also appends its own "+dirty" build-metadata suffix to Main.Version
	// when the tree is modified; strip it so we don't double up with the
	// "-dirty" marker we add from vcs.modified below.
	tag := strings.TrimSuffix(info.Main.Version, "+dirty")
	isReleaseTag := tag != "" && tag != "(devel)" && !strings.HasPrefix(tag, "v0.0.0-")

	var version string
	switch {
	case isReleaseTag:
		version = tag
	case revision != "":
		version = revision
	default:
		return "dev"
	}
	if dirty {
		version += "-dirty"
	}
	return version
}
