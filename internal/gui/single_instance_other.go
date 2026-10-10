//go:build !linux

package gui

// claimSingleInstance is a no-op off Linux: macOS LaunchServices already
// enforces one instance, and Windows keeps its existing behaviour.
func claimSingleInstance(func()) bool { return false }
