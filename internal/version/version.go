// Package version holds build metadata that is injected at link time.
package version

// These values are overridden at build time via -ldflags -X.
var (
	// Version is the semantic version of the build (e.g. v1.2.3).
	Version = "dev"
	// Commit is the git commit the binary was built from.
	Commit = "none"
	// Date is the UTC build timestamp in RFC3339 format.
	Date = "unknown"
)
