package app

// Smoke tests for the search form (?action=search) and the deferred
// search2 placeholder.

import (
	"net/url"
	"strings"
	"testing"
)

func TestSearchFormAdvanced(t *testing.T) {
	a := newTestApp(t)

	w, body := get(t, a, "/index.php?action=search;advanced", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("search status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{
		`name="searchform"`,
		`?action=search2`,             // posts to the engine
		`<input type="hidden" name="advanced" value="1" />`,
		`name="brd[1]"`,               // the board picker lists board 1
		`name="searchtype"`, // advanced-only field
		`name="sort"`,       // sort select
	} {
		if !strings.Contains(body, want) {
			t.Errorf("advanced search form missing %q", want)
		}
	}
}

func TestSearchFormSimple(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"simpleSearch": "1"})

	w, body := get(t, a, "/index.php?action=search", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("simple search status %d", w.Code)
	}
	if !strings.Contains(body, `<input type="hidden" name="advanced" value="0" />`) {
		t.Errorf("simple search form not rendered:\n%.400s", body)
	}
	// Simple form does not show the searchtype selector.
	if strings.Contains(body, `name="searchtype"`) {
		t.Errorf("simple form should not show advanced fields")
	}
}

func TestSearch2NoQuery(t *testing.T) {
	a := newTestApp(t)
	// No search string -> invalid_search_string -> back to the form (200).
	w, body := get(t, a, "/index.php?action=search2", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("search2 status %d", w.Code)
	}
	if !strings.Contains(body, `name="searchform"`) {
		t.Errorf("empty search2 should re-show the search form")
	}
}

func TestSearch2Results(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Post a topic whose body contains a distinctive word.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic": {"0"}, "subject": {"A searchable subject"},
		"message": {"The quick brown fox jumps over the lazy dog."},
		"icon": {"xx"}, "notify": {"0"}, "lock": {"0"}, "sticky": {"0"},
		"move": {"0"}, "additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("seed post status %d", w.Code)
	}

	// Search the body for "fox".
	w, body := get(t, a, "/index.php?action=search2;search=fox", admin)
	if w.Code != 200 {
		t.Fatalf("search2 status %d:\n%.400s", w.Code, body)
	}
	if !strings.Contains(body, "A searchable subject") {
		t.Errorf("search results missing the matched topic:\n%.1000s", body)
	}
	if !strings.Contains(body, `<b class="highlight">fox</b>`) &&
		!strings.Contains(body, `<b class="highlight">`) {
		t.Errorf("search results did not highlight the keyword")
	}

	// A query that matches nothing shows the "no results" + adjust-query box.
	_, none := get(t, a, "/index.php?action=search2;search=zzzznotpresent", admin)
	if !strings.Contains(none, "No results found") {
		t.Errorf("no-results search missing the no-results notice")
	}
	if !strings.Contains(none, "Adjust Search Parameters") {
		t.Errorf("no-results search missing the adjust-query box")
	}
	if strings.Contains(none, "A searchable subject") {
		t.Errorf("no-results search should not list any topic")
	}
}
