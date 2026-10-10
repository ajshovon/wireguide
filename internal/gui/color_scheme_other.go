//go:build !linux

package gui

// desktopPrefersDark is only consulted on Linux; elsewhere the webview's
// own prefers-color-scheme is already correct.
func desktopPrefersDark() bool { return false }
