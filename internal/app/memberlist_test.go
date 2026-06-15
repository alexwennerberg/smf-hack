package app

// Memberlist tests (Phase 4).

import (
	"strings"
	"testing"
)

func TestMemberlist(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	w, body := get(t, a, "/index.php?action=mlist", admin)
	if w.Code != 200 {
		t.Fatalf("mlist: status %d", w.Code)
	}
	if !strings.Contains(body, "admin") || !strings.Contains(body, "Viewing Members") {
		t.Errorf("memberlist missing admin row or title: %.300s", body)
	}
	// Letter links present.
	if !strings.Contains(body, ";sa=all;start=a#lettera") {
		t.Error("letter links missing")
	}

	// Search form renders.
	_, body = get(t, a, "/index.php?action=mlist;sa=search", admin)
	if !strings.Contains(body, `name="fields[]"`) {
		t.Errorf("search form missing fields: %.200s", body)
	}

	// Search finds the admin by name.
	_, body = get(t, a, "/index.php?action=mlist;sa=search;search=adm;fields=name", admin)
	if !strings.Contains(body, "admin") {
		t.Error("search did not find admin")
	}

	// Guests without view_mlist... guests have view_mlist by default in SMF;
	// check the page renders for guests too.
	w, _ = get(t, a, "/index.php?action=mlist")
	if w.Code != 200 {
		t.Errorf("guest mlist: status %d", w.Code)
	}
}
