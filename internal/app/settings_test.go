package app

// Phase 7: the settings-form infrastructure via ModSettings (featuresettings)
// and ManageServer (serversettings).

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestFeatureSettingsRender(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Basic tab.
	_, body := get(t, a, "/index.php?action=featuresettings;sa=basic", admin)
	if !strings.Contains(body, `name="enableErrorLogging"`) {
		t.Errorf("basic settings missing a checkbox:\n%.400s", body)
	}
	if !strings.Contains(body, `action=featuresettings2;save;sa=basic`) {
		t.Errorf("basic settings missing post url")
	}
	// The PM-spam-settings split should expose three int inputs.
	if !strings.Contains(body, `name="max_pm_recipients"`) {
		t.Errorf("basic settings missing pm spam inputs")
	}

	// Karma tab renders a select for karmaMode.
	_, body = get(t, a, "/index.php?action=featuresettings;sa=karma", admin)
	if !strings.Contains(body, `name="karmaMode"`) {
		t.Errorf("karma settings missing karmaMode select")
	}
}

func TestFeatureSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Get sc + session from the form.
	w, body := get(t, a, "/index.php?action=featuresettings;sa=karma", admin)
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatalf("no sc in settings form")
	}
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)

	wp := postForm(t, a, "/index.php?action=featuresettings2;save;sa=karma", url.Values{
		"karmaMode":         {"1"},
		"karmaMinPosts":     {"42"},
		"karmaWaitTime":     {"5"},
		"karmaLabel":        {"Karma"},
		"karmaApplaudLabel": {"applaud"},
		"karmaSmiteLabel":   {"smite"},
		"sc":                {sc[1]},
	}, cookies...)
	if wp.Code != 302 {
		t.Fatalf("save karma settings: status %d body %.300s", wp.Code, wp.Body.String())
	}

	if got := a.Setting("karmaMinPosts"); got != "42" {
		t.Errorf("karmaMinPosts = %q, want 42", got)
	}
	if got := a.Setting("karmaMode"); got != "1" {
		t.Errorf("karmaMode = %q, want 1", got)
	}
	if got := a.Setting("karmaLabel"); got != "Karma" {
		t.Errorf("karmaLabel = %q, want Karma", got)
	}
}

func TestServerSettingsTabs(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// serversettings requires checkSession('get') -> needs sesc. Grab it from
	// the admin home (any page exposing the token won't carry session; instead
	// hit serversettings via the maintenance page link is awkward) — use the
	// session cookie + sesc from a form page.
	w, body := get(t, a, "/index.php?action=featuresettings;sa=basic", admin)
	sc := scRe.FindStringSubmatch(body)
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)

	// The 'other' tab is pure DB settings (no file write).
	_, sbody := get(t, a, "/index.php?action=serversettings;sa=other;sesc="+sc[1], cookies...)
	if !strings.Contains(sbody, `name="smtp_host"`) {
		t.Errorf("server 'other' settings missing smtp_host:\n%.400s", sbody)
	}
	if !strings.Contains(sbody, `name="cookieTime"`) {
		t.Errorf("server 'other' settings missing cookieTime")
	}

	// The 'cache' tab.
	_, cbody := get(t, a, "/index.php?action=serversettings;sa=cache;sesc="+sc[1], cookies...)
	if !strings.Contains(cbody, `name="cache_enable"`) {
		t.Errorf("server 'cache' settings missing cache_enable")
	}

	// The 'core' tab renders read-only (save disabled) from smf.conf.
	_, corebody := get(t, a, "/index.php?action=serversettings;sa=core;sesc="+sc[1], cookies...)
	if !strings.Contains(corebody, `name="mbname"`) {
		t.Errorf("server 'core' settings missing mbname")
	}
	if !strings.Contains(corebody, `disabled="disabled"`) {
		t.Errorf("server 'core' settings should be disabled")
	}
}
