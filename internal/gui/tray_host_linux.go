//go:build linux

package gui

import (
	"context"
	"log/slog"
	"time"

	"github.com/godbus/dbus/v5"
)

// statusNotifierWatcher is the well-known bus name a desktop's tray
// registry owns. KDE, Cinnamon, XFCE and most panels provide it natively.
// Stock GNOME Shell does not: it has no tray unless the user installs the
// AppIndicator extension (Fedora Workstation ships without it).
const (
	statusNotifierWatcher     = "org.kde.StatusNotifierWatcher"
	statusNotifierWatcherPath = "/StatusNotifierWatcher"
	trayHostCheckTimeout      = time.Second
)

// trayHostAvailable reports whether something on the session bus will
// actually display our tray icon. Close-to-tray is only safe when it does;
// otherwise hiding the window leaves the app running with no way back.
//
// The watcher owning its name is not enough. It only brokers registrations,
// and can outlive or exist without anything that draws them (a watcher
// daemon left running with no panel applet), so the watcher is also asked
// whether a host has registered with it.
//
// Queried on every call rather than cached so enabling or disabling the
// GNOME extension mid-session takes effect on the next window close. Any
// D-Bus failure or timeout is treated as "no tray": minimising a window
// that could have been hidden is harmless, hiding one that can't be
// restored is not.
func trayHostAvailable() bool {
	conn, err := dbus.SessionBus()
	if err != nil {
		slog.Debug("tray host check: no session bus", "error", err)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), trayHostCheckTimeout)
	defer cancel()

	var owned bool
	err = conn.BusObject().
		CallWithContext(ctx, "org.freedesktop.DBus.NameHasOwner", 0, statusNotifierWatcher).
		Store(&owned)
	if err != nil {
		slog.Debug("tray host check: NameHasOwner failed", "error", err)
		return false
	}
	if !owned {
		return false
	}

	// NoAutoStart: only ask a watcher that is already there, never launch one.
	var hostRegistered dbus.Variant
	err = conn.Object(statusNotifierWatcher, statusNotifierWatcherPath).
		CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", dbus.FlagNoAutoStart,
			statusNotifierWatcher, "IsStatusNotifierHostRegistered").
		Store(&hostRegistered)
	if err != nil {
		slog.Debug("tray host check: cannot read IsStatusNotifierHostRegistered", "error", err)
		return false
	}
	registered, _ := hostRegistered.Value().(bool)
	return registered
}
