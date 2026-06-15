package app

// Phase 7: admin report generator.

import (
	"strings"
	"testing"
)

func TestReportsTypeChooser(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=reports", admin)
	if !strings.Contains(body, `name="rt"`) || !strings.Contains(body, `value="boards"`) || !strings.Contains(body, `value="staff"`) {
		t.Fatalf("report chooser missing types:\n%.400s", body)
	}
}

func TestReportsMemberGroups(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=reports;rt=member_groups", admin)
	// The single member-groups table: the "Color" settings label + guests row.
	if !strings.Contains(body, "Color") {
		t.Errorf("member groups report missing settings labels:\n%.400s", body)
	}
	if !strings.Contains(body, "Guests") {
		t.Errorf("member groups report missing the guests row")
	}
}

func TestReportsGroupPermissions(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Seed a permission so the report has at least one row.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}permissions (ID_GROUP, permission, addDeny) VALUES (0, 'post_new', 1)`))

	_, body := get(t, a, "/index.php?action=reports;rt=group_perms", admin)
	if !strings.Contains(body, "Group Permissions") || !strings.Contains(body, "Allow") {
		t.Fatalf("group perms report did not render:\n%.400s", body)
	}
}

func TestReportsStaff(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// The admin (group 1) is staff via admin_forum.
	_, body := get(t, a, "/index.php?action=reports;rt=staff", admin)
	if !strings.Contains(body, "admin") || !strings.Contains(body, "Position") {
		t.Fatalf("staff report missing the admin:\n%.500s", body)
	}
}

func TestReportsBoardsAndPrint(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// The schema seeds a "General Discussion" board.
	w, body := get(t, a, "/index.php?action=reports;rt=boards", admin)
	if w.Code != 200 {
		t.Fatalf("boards report: status %d", w.Code)
	}
	if !strings.Contains(body, "General Discussion") || !strings.Contains(body, "Category") {
		t.Fatalf("boards report missing the board:\n%.500s", body)
	}

	// The print view renders a standalone HTML doc.
	_, body = get(t, a, "/index.php?action=reports;rt=boards;st=print", admin)
	if !strings.Contains(body, "<html") || !strings.Contains(body, "General Discussion") {
		t.Fatalf("boards print view malformed:\n%.300s", body)
	}
}
