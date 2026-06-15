package app

// Phase 7: the admin center home + the admin sidebar chrome.

import (
	"strings"
	"testing"
)

func TestAdminHome(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	// Skip the admin-session password prompt for the test.
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=admin", admin)

	// The sidebar sections (catbg titles) and the home welcome.
	if !strings.Contains(body, "Administration Center") {
		t.Errorf("admin home missing title:\n%.400s", body)
	}
	// The admin should be listed among administrators.
	if !strings.Contains(body, "?action=profile;u=1") {
		t.Errorf("admin home missing administrator link")
	}
	// A sidebar link to maintenance (admin_forum).
	if !strings.Contains(body, "?action=maintain") {
		t.Errorf("admin sidebar missing maintenance link")
	}
	// Quick task: feature settings.
	if !strings.Contains(body, "?action=featuresettings") {
		t.Errorf("admin home missing feature settings quick task")
	}
}

func TestAdminCredits(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=admin;credits", admin)
	if !strings.Contains(body, "Project Managers:") {
		t.Errorf("credits page missing credits text:\n%.300s", body)
	}
	if !strings.Contains(body, "smfSupportVersions.forum") {
		t.Errorf("credits page missing support-version script")
	}
}

func TestAdminRequiresPermission(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// A guest must not reach the admin center.
	w, _ := get(t, a, "/index.php?action=admin")
	if w.Code == 200 && strings.Contains(w.Body.String(), "?action=featuresettings") {
		t.Errorf("guest should not see the admin center")
	}
}
