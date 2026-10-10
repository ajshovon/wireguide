//go:build !linux || !cgo || gtk4

package gui

// applyNativeTheme is only needed for GTK3 chrome on Linux. macOS and
// Windows draw the title bar themselves and expose no per-app override
// we use.
func applyNativeTheme(string) {}
