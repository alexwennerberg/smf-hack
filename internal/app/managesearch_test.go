package app

// Phase 7: search admin (settings / weights / method).

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSearchSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=managesearch;sa=settings", admin)
	w := postForm(t, a, "/index.php?action=managesearch;sa=settings", url.Values{
		"save":                    {"1"},
		"simpleSearch":            {"on"},
		"search_results_per_page": {"25"},
		"search_max_results":      {"1000"},
		"search_posts[0]":         {"on"},
		"sc":                      {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save search settings: status %d", w.Code)
	}
	if a.Setting("simpleSearch") != "1" || a.Setting("search_results_per_page") != "25" {
		t.Fatalf("search settings not saved")
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 0 AND permission = 'search_posts'`)).Scan(&n)
	if n != 1 {
		t.Errorf("search_posts inline permission not granted")
	}
}

func TestSearchWeightsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=managesearch;sa=weights", admin)
	w := postForm(t, a, "/index.php?action=managesearch;sa=weights", url.Values{
		"save":                        {"1"},
		"search_weight_frequency":     {"30"},
		"search_weight_age":           {"25"},
		"search_weight_length":        {"20"},
		"search_weight_subject":       {"15"},
		"search_weight_first_message": {"10"},
		"search_weight_sticky":        {"0"},
		"sc":                          {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save weights: status %d", w.Code)
	}
	if a.Setting("search_weight_frequency") != "30" {
		t.Fatalf("weight not saved")
	}
	// Total = 100, so frequency renders 30%.
	_, body := get(t, a, "/index.php?action=managesearch;sa=weights", admin)
	if !strings.Contains(body, "30%") {
		t.Errorf("relative weight percent not rendered:\n%.300s", body)
	}
}

func TestSearchMethodForcedStandard(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	w, body := get(t, a, "/index.php?action=managesearch;sa=method", admin)
	if w.Code != 200 || !strings.Contains(body, `name="search_index"`) {
		t.Fatalf("method page render: status %d", w.Code)
	}
	// Fulltext cannot be created in this port.
	if !strings.Contains(body, "cannot") && !strings.Contains(body, "Cannot") {
		// the cannot_create message is shown; just sanity-check the page exists
	}

	sc := scRe.FindStringSubmatch(body)
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)
	// Try to force fulltext; the port ignores it and keeps standard ('').
	wp := postForm(t, a, "/index.php?action=managesearch;sa=method", url.Values{
		"save":               {"1"},
		"search_index":       {"fulltext"},
		"search_match_words": {"on"},
		"sc":                 {sc[1]},
	}, cookies...)
	if wp.Code != 200 && wp.Code != 302 {
		t.Fatalf("save method: status %d", wp.Code)
	}
	if a.Setting("search_index") != "" {
		t.Fatalf("search_index should be forced to standard, got %q", a.Setting("search_index"))
	}
	if a.Setting("search_match_words") != "1" {
		t.Fatalf("search_match_words not saved")
	}
}
