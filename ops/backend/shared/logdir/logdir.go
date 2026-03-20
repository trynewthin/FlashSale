// Package logdir manages runtime log directories for ops command entrypoints.
//
// Constraints:
// - `.memory/` is only for local planning files and must not be used for runtime logs.
// - Runtime logs are written under the repository `log/` directory for easier inspection.
package logdir

import "path/filepath"

// Root returns the repository-level runtime log root directory.
func Root(repoRoot string) string {
	return filepath.Join(repoRoot, "log")
}

// ServicesDir returns the service process log directory.
func ServicesDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "services")
}

// FrontendsDir returns the frontend dev server log directory.
func FrontendsDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "frontends")
}

// OpsJobsDir returns the ops-control job archive directory.
func OpsJobsDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "ops-jobs")
}

// DataDir returns the data tool output directory.
func DataDir(repoRoot string) string {
	return filepath.Join(Root(repoRoot), "data")
}
