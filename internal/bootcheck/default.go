//go:build !custombootcheck
// +build !custombootcheck

package bootcheck

// CheckEnv performs environment validation checks.
// In the default build configuration, this is a no-op implementation.
func CheckEnv() error {
	// No-op by default. Use build tags for build-time isolation of custom preflight checks.
	return nil
}
