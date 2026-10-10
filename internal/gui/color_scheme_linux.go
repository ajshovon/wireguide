//go:build linux

package gui

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// The XDG settings portal is the desktop-neutral source for the light/dark
// preference (GNOME's "Style" switch, KDE's colour scheme, …).
const (
	portalBusName      = "org.freedesktop.portal.Desktop"
	portalObjectPath   = "/org/freedesktop/portal/desktop"
	portalSettings     = "org.freedesktop.portal.Settings"
	appearanceNS       = "org.freedesktop.appearance"
	colorSchemeKey     = "color-scheme"
	colorSchemeDark    = 1 // 0 = no preference, 1 = prefer dark, 2 = prefer light
	settingChangedName = portalSettings + ".SettingChanged"
	portalTimeout      = time.Second
)

// desktopPrefersDark asks the settings portal whether the desktop is in
// dark mode.
//
// This cannot be left to GTK. GNOME's dark style is the color-scheme
// setting, which only libadwaita/GTK4 apps act on; a GTK3 process still
// sees theme "Adwaita" with no dark preference, and WebKitGTK derives
// prefers-color-scheme from that. Left alone, the "System" theme renders
// light on a dark GNOME desktop.
//
// Any failure (no session bus, no portal, no answer in time) reports false,
// which leaves GTK on whatever the theme itself says. The deadline covers
// both attempts: gui.Run asks this on the main goroutine before the window
// exists, and a portal that is slow to activate must not hold startup up.
func desktopPrefersDark() bool {
	conn, err := dbus.SessionBus()
	if err != nil {
		return false
	}
	obj := conn.Object(portalBusName, portalObjectPath)
	ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
	defer cancel()

	var v dbus.Variant
	// ReadOne (portal settings v2) returns the value directly; the older
	// Read wraps it in a second variant.
	if err := obj.CallWithContext(ctx, portalSettings+".ReadOne", 0, appearanceNS, colorSchemeKey).Store(&v); err != nil {
		if err := obj.CallWithContext(ctx, portalSettings+".Read", 0, appearanceNS, colorSchemeKey).Store(&v); err != nil {
			slog.Debug("color scheme: portal read failed", "error", err)
			return false
		}
	}
	return colorSchemeFromVariant(v) == colorSchemeDark
}

// colorSchemeFromVariant unwraps the portal's uint32, tolerating the
// variant-in-variant shape of the legacy Read call.
func colorSchemeFromVariant(v dbus.Variant) uint32 {
	val := v.Value()
	if inner, ok := val.(dbus.Variant); ok {
		val = inner.Value()
	}
	n, _ := val.(uint32)
	return n
}

var colorSchemeWatchOnce sync.Once

// watchDesktopColorScheme calls onChange whenever the desktop's light/dark
// preference changes. It uses a private bus connection so the signal
// channel does not receive traffic meant for other users of the shared
// one. Runs for the life of the process; safe to call more than once.
func watchDesktopColorScheme(onChange func()) {
	colorSchemeWatchOnce.Do(func() {
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			slog.Debug("color scheme: cannot watch for changes", "error", err)
			return
		}
		if err := conn.AddMatchSignal(
			dbus.WithMatchObjectPath(portalObjectPath),
			dbus.WithMatchInterface(portalSettings),
			dbus.WithMatchMember("SettingChanged"),
		); err != nil {
			slog.Debug("color scheme: cannot subscribe to portal", "error", err)
			conn.Close()
			return
		}
		signals := make(chan *dbus.Signal, 8)
		conn.Signal(signals)
		go func() {
			for sig := range signals {
				if sig.Name != settingChangedName || len(sig.Body) < 2 {
					continue
				}
				ns, _ := sig.Body[0].(string)
				key, _ := sig.Body[1].(string)
				if ns == appearanceNS && key == colorSchemeKey {
					onChange()
				}
			}
		}()
	})
}
