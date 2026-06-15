package app

// Smoke test for the Who's Online page (?action=who).

import (
	"strings"
	"testing"
)

func TestWhoOnline(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// First request establishes the session and logs the admin online.
	_, _ = get(t, a, "/index.php", admin)

	w, body := get(t, a, "/index.php?action=who", admin)
	if w.Code != 200 {
		t.Fatalf("who status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{
		"?action=who;start=0;sort=user", // sortable header
		">admin</a>",                    // the online member, linked
	} {
		if !strings.Contains(body, want) {
			t.Errorf("who page missing %q", want)
		}
	}
}

func TestWhoDisabled(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"who_enabled": ""})

	w, _ := get(t, a, "/index.php?action=who", adminCookie(t, a))
	// who_off is a fatal_lang_error: the page still renders (200) with the
	// error, not a redirect.
	if w.Code != 200 {
		t.Fatalf("who disabled status %d", w.Code)
	}
}
