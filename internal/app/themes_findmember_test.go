package app

// Smoke tests for the remaining Phase 4 actions: the findmember /
// requestmembers popups (Subs-Auth.php) and the theme picker / jsoption
// option setter (Themes.php).

import (
	"net/http"
	"strings"
	"testing"
)

// sescFromSendForm renders the PM send form (which carries a hidden sc) and
// returns the session token plus the cookies that established the session.
func sescFromSendForm(t *testing.T, a *App) (string, []*http.Cookie) {
	t.Helper()
	admin := adminCookie(t, a)
	w, body := get(t, a, "/index.php?action=pm;sa=send", admin)
	if w.Code != 200 {
		t.Fatalf("send form status %d", w.Code)
	}
	m := scRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no sc in send form:\n%s", body)
	}
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)
	return m[1], cookies
}

func TestFindMemberPopup(t *testing.T) {
	a := newTestApp(t)
	sc, cookies := sescFromSendForm(t, a)

	// No search yet: the popup shell renders and focuses the search box.
	w, body := get(t, a, "/index.php?action=findmember;input=to;quote=1;sesc="+sc, cookies...)
	if w.Code != 200 {
		t.Fatalf("findmember status %d:\n%s", w.Code, body)
	}
	for _, want := range []string{
		"<title>Find Members</title>",
		`name="search"`,
		`name="input" value="to"`,
		`document.getElementById("search").focus();`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("findmember popup missing %q", want)
		}
	}

	// Now search for the admin and confirm a result row with an addMember link.
	// JSMembers (unlike requestmembers) doesn't auto-append a wildcard, so the
	// caller must supply one, exactly as the real popup requires.
	_, body = get(t, a, "/index.php?action=findmember;input=to;quote=1;search=adm*;sesc="+sc, cookies...)
	if !strings.Contains(body, `onclick="addMember(this.title)`) {
		t.Errorf("findmember search produced no result row:\n%s", body)
	}
	if !strings.Contains(body, ">admin</a>") {
		t.Errorf("findmember search did not list admin:\n%s", body)
	}
}

func TestRequestMembers(t *testing.T) {
	a := newTestApp(t)
	sc, cookies := sescFromSendForm(t, a)

	w, body := get(t, a, "/index.php?action=requestmembers;search=adm;sesc="+sc, cookies...)
	if w.Code != 200 {
		t.Fatalf("requestmembers status %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("requestmembers Content-Type = %q, want text/plain", ct)
	}
	if strings.TrimSpace(body) != "admin" {
		t.Errorf("requestmembers body = %q, want \"admin\\n\"", body)
	}
}

func TestThemePickRenders(t *testing.T) {
	a := newTestApp(t)
	sc, cookies := sescFromSendForm(t, a)

	w, body := get(t, a, "/index.php?action=theme;sa=pick;sesc="+sc, cookies...)
	if w.Code != 200 {
		t.Fatalf("theme pick status %d:\n%s", w.Code, body)
	}
	if !strings.Contains(body, "use this theme") || !strings.Contains(body, "Forum or Board Default") {
		t.Errorf("theme picker missing expected entries:\n%s", body)
	}
}

func TestJsOptionStoresAndGuards(t *testing.T) {
	a := newTestApp(t)
	sc, cookies := sescFromSendForm(t, a)

	// A normal option round-trips into the themes table for this member.
	w, _ := get(t, a, "/index.php?action=jsoption;var=collapse_header;val=1;sesc="+sc, cookies...)
	if w.Code != 302 {
		t.Fatalf("jsoption status %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); !strings.HasSuffix(loc, "/blank.gif") {
		t.Errorf("jsoption redirect %q, want .../blank.gif", loc)
	}
	var val string
	err := a.DB.QueryRow(a.Q(`SELECT value FROM {$db_prefix}themes
		WHERE ID_MEMBER = 1 AND variable = 'collapse_header'`)).Scan(&val)
	if err != nil || val != "1" {
		t.Errorf("jsoption did not store option: err=%v val=%q", err, val)
	}

	// Reserved vars are refused (no row written).
	get(t, a, "/index.php?action=jsoption;var=theme_url;val=evil;sesc="+sc, cookies...)
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}themes
		WHERE ID_MEMBER = 1 AND variable = 'theme_url'`)).Scan(&n)
	if n != 0 {
		t.Errorf("jsoption wrote a reserved var (%d rows)", n)
	}
}
