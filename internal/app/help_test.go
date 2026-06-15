package app

// Smoke test for the ?action=helpadmin popup.

import (
	"strings"
	"testing"
)

func TestHelpAdminPopup(t *testing.T) {
	a := newTestApp(t)

	w, body := get(t, a, "/index.php?action=helpadmin;help=manage_boards", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("helpadmin status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{
		"<div class=\"popuptext\">",
		"Edit Boards",  // from the manage_boards help string
		"Close window", // txt[1006]
		"self.close();",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("help popup missing %q", want)
		}
	}
	// No chrome: it's a standalone popup document.
	if strings.Contains(body, "main_above") {
		t.Errorf("help popup should not render the main chrome")
	}
}

func TestShowHelpManual(t *testing.T) {
	a := newTestApp(t)

	// Default page (no ?page=) is the intro.
	w, body := get(t, a, "/index.php?action=help")
	if w.Code != 200 {
		t.Fatalf("help status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{
		`<div id="helpmenu" class="titlebg"`,  // manual_above layer
		`<div id="helpmenu2" class="titlebg"`, // manual_below layer
		`SMF User Help: Introduction`,         // page title in chrome
		`/help.css`,                           // the injected stylesheet
		`<span class="error" style="font-weight: bold;">Introduction</span>`,         // current page bold in menu
		`<a href="` + a.ScriptURL + `?action=help;page=registering">Registering</a>`, // other page linked
		`function collapseExpandCategory()`,                                          // intro JS block
	} {
		if !strings.Contains(body, want) {
			t.Errorf("intro help page missing %q", want)
		}
	}

	// Each known page resolves to its own sub-template with a distinctive marker.
	cases := map[string]string{
		"registering": `<h2 id="how-to">`,
		"loginout":    `<h2 id="login">`,
		"profile":     `<h2 id="all">`,
		"post":        `<h2 id="basics">`,
		"pm":          `<h2 id="pm">`,
		"searching":   `<h2 id="starting">`,
	}
	for page, marker := range cases {
		w, body := get(t, a, "/index.php?action=help;page="+page)
		if w.Code != 200 {
			t.Fatalf("help page=%s status %d", page, w.Code)
		}
		if !strings.Contains(body, marker) {
			t.Errorf("help page=%s missing marker %q", page, marker)
		}
		// The current page should be the bold (non-link) menu entry.
		if strings.Contains(body, `?action=help;page=`+page+`">`) {
			t.Errorf("help page=%s should render its own menu entry as bold, not a link", page)
		}
	}

	// An unknown page falls back to the intro.
	w, body = get(t, a, "/index.php?action=help;page=bogus")
	if w.Code != 200 || !strings.Contains(body, `SMF User Help: Introduction`) {
		t.Errorf("unknown help page should fall back to the intro")
	}
}

func TestHelpAdminUnknownKey(t *testing.T) {
	a := newTestApp(t)
	w, body := get(t, a, "/index.php?action=helpadmin;help=no_such_help_key", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	// Falls back to echoing the raw key.
	if !strings.Contains(body, "no_such_help_key") {
		t.Errorf("unknown help key should fall back to the key itself")
	}
}
