//go:build !custombootcheck
// +build !custombootcheck

/*
Package bootcheck provides helpers for checks at boot time.
*/
package bootcheck

// CheckEnv is a no-op by default. Use build tags for build-time isolation of
// custom preflight checks. Ensure to update the build tags on L1-L2 so that
// they are mutually exclusive across implementations.
func CheckEnv() error {
	return nil
}
