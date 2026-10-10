//go:build linux && cgo && !gtk4

package gui

/*
#cgo pkg-config: gtk+-3.0
#include <string.h>
#include <gtk/gtk.h>

// Length of the base name when `name` ends in "-dark" in any letter case
// ("Adwaita-dark", "Mint-Y-Dark", "Arc-Dark"), 0 otherwise.
static size_t wg_dark_theme_base_len(const gchar *name) {
	size_t len = strlen(name);
	size_t suffix = strlen("-dark");
	if (len <= suffix || g_ascii_strcasecmp(name + len - suffix, "-dark") != 0) {
		return 0;
	}
	return len - suffix;
}

// mode: 0 = follow the desktop, 1 = force dark, 2 = force light.
static void wg_apply_theme(int mode) {
	GtkSettings *s = gtk_settings_get_default();
	if (s == NULL) {
		return;
	}
	// Drop any override from a previous call first so the values read
	// below are the desktop's, not our own.
	gtk_settings_reset_property(s, "gtk-application-prefer-dark-theme");
	gtk_settings_reset_property(s, "gtk-theme-name");
	if (mode == 0) {
		return;
	}
	if (mode == 1) {
		g_object_set(s, "gtk-application-prefer-dark-theme", TRUE, NULL);
		return;
	}
	// Forcing light is more than clearing the dark preference: when the
	// desktop is in dark mode GTK3 is handed an explicitly dark theme
	// ("Adwaita-dark"), so select its light sibling by name.
	gchar *name = NULL;
	g_object_get(s, "gtk-theme-name", &name, NULL);
	if (name != NULL) {
		size_t base = wg_dark_theme_base_len(name);
		if (base > 0) {
			name[base] = '\0';
			g_object_set(s, "gtk-theme-name", name, NULL);
		}
		g_free(name);
	}
	g_object_set(s, "gtk-application-prefer-dark-theme", FALSE, NULL);
}
*/
import "C"

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// nativeTheme is the last Settings value handed to applyNativeTheme, kept
// so a desktop light/dark switch can re-resolve "system".
var (
	nativeThemeMu sync.Mutex
	nativeTheme   = "system"
)

// applyNativeTheme makes GTK follow the theme picked in Settings — "dark",
// "light" or "system". That covers two things the web content cannot do
// for itself:
//
//   - GTK's own chrome (client-side title bar, context menus, file
//     dialogs) would otherwise keep the desktop's look, leaving e.g. a
//     light title bar on a window the user set to Dark.
//   - WebKit's prefers-color-scheme comes from the same GtkSettings, and
//     GTK3 does not pick up GNOME's dark style on its own (see
//     desktopPrefersDark). Resolving "system" here is what makes the
//     default theme follow the desktop at all.
func applyNativeTheme(theme string) {
	nativeThemeMu.Lock()
	nativeTheme = theme
	nativeThemeMu.Unlock()
	pushNativeTheme()
	watchDesktopColorScheme(pushNativeTheme)
}

func pushNativeTheme() {
	nativeThemeMu.Lock()
	theme := nativeTheme
	nativeThemeMu.Unlock()

	mode := C.int(0)
	switch theme {
	case "dark":
		mode = 1
	case "light":
		mode = 2
	default:
		if desktopPrefersDark() {
			mode = 1
		}
	}
	// GtkSettings must only be touched from the GTK main thread.
	application.InvokeAsync(func() { C.wg_apply_theme(mode) })
}
