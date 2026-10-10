//go:build linux

package gui

import (
	"context"
	"log/slog"
	"time"

	"github.com/godbus/dbus/v5"
)

// A running WireGuide GUI owns instanceBusName on the session bus for as
// long as it lives and answers org.freedesktop.Application.Activate there —
// the freedesktop contract for "raise the instance that is already
// running". Only Activate is implemented; nothing here is D-Bus activatable.
const (
	instanceBusName     = "io.github.korjwl1.wireguide"
	instanceObjectPath  = "/io/github/korjwl1/wireguide"
	instanceInterface   = "org.freedesktop.Application"
	instanceCallTimeout = 3 * time.Second
)

// instanceActivator adapts a callback to the exported D-Bus object.
type instanceActivator func()

// Activate implements org.freedesktop.Application.Activate. The platform
// data (startup id, activation token) is accepted and ignored. The callback
// runs on its own goroutine so the caller — a second launch that exits as
// soon as we reply — is not held up by window work.
func (f instanceActivator) Activate(map[string]dbus.Variant) *dbus.Error {
	go f()
	return nil
}

// claimSingleInstance makes this process the session's one WireGuide GUI.
// It returns true when another instance already holds that role; that
// instance has then been asked to raise its window (its own onActivate) and
// the caller should exit without doing anything else.
//
// gui.Run calls this before the helper phase on purpose. A second launch
// that got as far as ensureHelper could put up its own pkexec prompt while
// the first is still waiting on one, or — when it is a newer build than the
// running GUI — shut down the helper that GUI is using, tunnel included,
// before finding out it was never going to stay.
//
// Without a session bus there is nothing to coordinate through, so the app
// starts unguarded, as it always did, rather than refusing to run.
func claimSingleInstance(onActivate func()) bool {
	// Private connection, deliberately left open on the owning path:
	// holding the name is the lock, and the bus releases it when we exit.
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		slog.Warn("single instance: no session bus, not enforcing", "error", err)
		return false
	}
	// Export before requesting the name, so a racing second launch never
	// finds the name owned with nothing answering behind it.
	if err := conn.Export(instanceActivator(onActivate), instanceObjectPath, instanceInterface); err != nil {
		slog.Warn("single instance: export failed, not enforcing", "error", err)
		conn.Close()
		return false
	}
	reply, err := conn.RequestName(instanceBusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		slog.Warn("single instance: cannot request bus name, not enforcing", "error", err)
		conn.Close()
		return false
	}
	if reply != dbus.RequestNameReplyExists {
		return false
	}

	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), instanceCallTimeout)
	defer cancel()
	err = conn.Object(instanceBusName, instanceObjectPath).
		CallWithContext(ctx, instanceInterface+".Activate", 0, map[string]dbus.Variant{}).Err
	if err != nil {
		// Still report "already running": a second GUI fighting the first
		// over the helper is worse than a launch that appears to do nothing.
		slog.Warn("single instance: running instance did not answer", "error", err)
	}
	return true
}
