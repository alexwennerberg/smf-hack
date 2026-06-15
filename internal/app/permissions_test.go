package app

// Phase 7: the permission center + inline-permission helpers.

import (
	"net/url"
	"strings"
	"testing"
)

func TestPermissionIndex(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=permissions", admin)
	// Synthetic groups + the membergroups.
	if !strings.Contains(body, "Administrator") || !strings.Contains(body, "Global Moderator") {
		t.Errorf("permission index missing groups:\n%.500s", body)
	}
	// A modify link for a modifiable group.
	if !strings.Contains(body, "?action=permissions;sa=modify;group=") {
		t.Errorf("permission index missing modify link")
	}
	// The predefined-level quick action.
	if !strings.Contains(body, `name="predefined"`) {
		t.Errorf("permission index missing quick form")
	}
}

func TestModifyGroupSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Render the matrix for Regular Members (group 0).
	sc, cookies := mbForm(t, a, "/index.php?action=permissions;sa=modify;group=0", admin)
	// (No assertion on form body beyond sc; render is enough.)

	w := postForm(t, a, "/index.php?action=permissions;sa=modify2;group=0;boardid=0", url.Values{
		"perm[membergroup][view_stats]": {"on"},
		"perm[membergroup][view_mlist]": {"on"},
		"perm[board][post_new]":         {"on"},
		"sc":                            {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("modify2: status %d body %.300s", w.Code, w.Body.String())
	}

	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 0 AND permission = 'view_stats' AND addDeny = 1`)).Scan(&n)
	if n != 1 {
		t.Fatalf("view_stats not granted to group 0")
	}
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}board_permissions WHERE ID_GROUP = 0 AND ID_BOARD = 0 AND permission = 'post_new'`)).Scan(&n)
	if n != 1 {
		t.Fatalf("post_new board permission not granted to group 0")
	}
}

func TestSetQuickPredefined(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=permissions", admin)
	w := postForm(t, a, "/index.php?action=permissions;sa=quick", url.Values{
		"group[]":    {"0"},
		"predefined": {"standard"},
		"sc":         {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("quick predefined: status %d", w.Code)
	}
	// "standard" grants view_mlist + pm_read globally.
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 0 AND permission = 'pm_read'`)).Scan(&n)
	if n != 1 {
		t.Fatalf("standard level did not grant pm_read")
	}
	// ...and poll_vote at board level.
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}board_permissions WHERE ID_GROUP = 0 AND ID_BOARD = 0 AND permission = 'poll_vote'`)).Scan(&n)
	if n != 1 {
		t.Fatalf("standard level did not grant board poll_vote")
	}
}

func TestGeneralPermSettingsInline(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// The settings page renders the inline manage_permissions box.
	sc, cookies := mbForm(t, a, "/index.php?action=permissions;sa=settings", admin)
	_, body := get(t, a, "/index.php?action=permissions;sa=settings", admin)
	if !strings.Contains(body, `name="manage_permissions[`) {
		t.Errorf("settings page missing inline manage_permissions box:\n%.400s", body)
	}

	// Grant manage_permissions to group 2 via the inline box + enable deny.
	w := postForm(t, a, "/index.php?action=permissions;sa=settings", url.Values{
		"save_settings":          {"1"},
		"permission_enable_deny": {"on"},
		"manage_permissions[2]":  {"on"},
		"sc":                     {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save perm settings: status %d", w.Code)
	}
	if a.Setting("permission_enable_deny") != "1" {
		t.Fatalf("permission_enable_deny not saved")
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 2 AND permission = 'manage_permissions' AND addDeny = 1`)).Scan(&n)
	if n != 1 {
		t.Fatalf("manage_permissions not granted to group 2 via inline box")
	}
}
