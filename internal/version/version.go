package version

var (
	// Version holds the complete version number. Filled in at linking time via -ldflags.
	Version = "dev"

	// GitCommit holds the git revision. Filled in at linking time via -ldflags.
	GitCommit = "unknown"
)
