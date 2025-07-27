//go:build !custombootcheck
// +build !custombootcheck

// Package bootcheck supports mutually exclusive implementations of
// custom preflight checks for the composition function.
package bootcheck

// CheckEnv performs environment validation checks.
// In the default build configuration, this is a no-op implementation.
func CheckEnv() error {
	// No-op by default. Use build tags for build-time isolation of custom preflight checks.
	return nil
}
