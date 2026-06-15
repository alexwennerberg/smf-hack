package app

// Phase 7: theme admin settings subset (ThemeAdmin).

import (
	"net/url"
	"strings"
	"testing"
)

func TestThemeAdminRenderAndSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Session token + cookies from a form page (maintenance has forms with sc).
	sc, cookies := mbForm(t, a, "/index.php?action=maintain", admin)

	// The settings page renders (GET requires sesc).
	_, body := get(t, a, "/index.php?action=theme;sesc="+sc+";sa=admin", cookies...)
	if !strings.Contains(body, `name="options[theme_allow]"`) || !strings.Contains(body, `name="theme_reset"`) {
		t.Fatalf("theme admin page missing settings form:\n%.400s", body)
	}

	// Save: enable member theme selection + reset everyone to the default.
	w := postForm(t, a, "/index.php?action=theme;sa=admin", url.Values{
		"submit":                {"Save"},
		"options[theme_allow]":  {"1"},
		"options[theme_guests]": {"1"},
		"theme_reset":           {"0"},
		"sc":                    {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("save theme settings: status %d body %.200s", w.Code, w.Body.String())
	}
	if a.Setting("theme_allow") != "1" || a.Setting("theme_guests") != "1" {
		t.Fatalf("theme settings not saved (allow=%q guests=%q)", a.Setting("theme_allow"), a.Setting("theme_guests"))
	}
	// theme_reset=0 sets every member's ID_THEME to 0.
	var nonZero int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE ID_THEME != 0`)).Scan(&nonZero)
	if nonZero != 0 {
		t.Errorf("theme_reset did not reset all members")
	}
}
