//go:build !darwin

package permissions

// EnsurePermissions is a no-op on non-macOS platforms.
func EnsurePermissions() error {
	return nil
}
