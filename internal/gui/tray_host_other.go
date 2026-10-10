//go:build !linux

package gui

// trayHostAvailable is always true off Linux: the macOS menu bar and the
// Windows notification area are part of the OS and cannot be absent.
func trayHostAvailable() bool { return true }
