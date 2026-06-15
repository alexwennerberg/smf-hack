package app

// Phase 7: member administration (list/search/approve/delete).

import (
	"net/url"
	"strings"
	"testing"
)

func TestViewMembersListAndDelete(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members (memberName, realName, emailAddress, ID_GROUP, ID_POST_GROUP, is_activated) VALUES ('deluser', 'Del User', 'del@x.com', 0, 4, 1)`))
	var uid int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'deluser'`)).Scan(&uid)

	_, body := get(t, a, "/index.php?action=viewmembers", admin)
	if !strings.Contains(body, "deluser") || !strings.Contains(body, "?action=viewmembers") {
		t.Fatalf("member list missing user:\n%.400s", body)
	}
	if !strings.Contains(body, `name="delete[]"`) {
		t.Errorf("member list missing delete checkboxes")
	}

	sc, cookies := mbForm(t, a, "/index.php?action=viewmembers", admin)
	w := postForm(t, a, "/index.php?action=viewmembers", url.Values{
		"delete_members": {"1"},
		"delete[]":       {itoa(uid)},
		"sort":           {"memberName"},
		"start":          {"0"},
		"sc":             {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("delete member: status %d", w.Code)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE ID_MEMBER = ?`), uid).Scan(&n)
	if n != 0 {
		t.Fatalf("member not deleted")
	}
}

func TestViewMembersByGroupAndSearch(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Filter by group 1 (administrators) -> the seed admin shows.
	_, body := get(t, a, "/index.php?action=viewmembers;sa=query;group=1", admin)
	if !strings.Contains(body, "admin") {
		t.Errorf("group filter did not list the admin:\n%.300s", body)
	}

	// Search form renders.
	_, body = get(t, a, "/index.php?action=viewmembers;sa=search", admin)
	if !strings.Contains(body, `name="membername"`) || !strings.Contains(body, `name="membergroups[1][]"`) {
		t.Errorf("search form missing fields")
	}
}

func TestMemberApproval(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// A member awaiting approval (is_activated = 3).
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members (memberName, realName, emailAddress, ID_GROUP, ID_POST_GROUP, is_activated, dateRegistered) VALUES ('pending', 'Pending One', 'pend@x.com', 0, 4, 3, ?)`), nowUnix())
	var uid int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'pending'`)).Scan(&uid)

	// The approval queue lists them.
	_, body := get(t, a, "/index.php?action=viewmembers;sa=browse;type=approve", admin)
	if !strings.Contains(body, "pending") {
		t.Fatalf("approval queue missing pending member:\n%.400s", body)
	}

	// Approve them.
	sc, cookies := mbForm(t, a, "/index.php?action=viewmembers;sa=browse;type=approve", admin)
	w := postForm(t, a, "/index.php?action=viewmembers;sa=approve", url.Values{
		"todo":         {"ok"},
		"todoAction[]": {itoa(uid)},
		"type":         {"approve"},
		"sort":         {"dateRegistered"},
		"start":        {"0"},
		"orig_filter":  {"3"},
		"sc":           {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("approve: status %d body %.300s", w.Code, w.Body.String())
	}
	var activated int
	a.DB.QueryRow(a.Q(`SELECT is_activated FROM {$db_prefix}members WHERE ID_MEMBER = ?`), uid).Scan(&activated)
	if activated != 1 {
		t.Fatalf("member not approved (is_activated=%d)", activated)
	}
}
