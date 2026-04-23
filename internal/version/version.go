package version

var (
	Placeholder = "dev"

	// Version holds the complete version number. Filled in at linking time via -ldflags.
	Version = Placeholder

	// GitCommit holds the git revision. Filled in at linking time via -ldflags.
	GitCommit = "unknown"
)
