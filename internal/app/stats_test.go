package app

// Smoke tests for the Stats page (?action=stats) and its expand XML.

import (
	"strings"
	"testing"
)

func TestStatsPage(t *testing.T) {
	a := newTestApp(t)

	// Seed a couple of log_activity rows so the history table renders.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_activity (date, hits, topics, posts, registers, mostOn)
		VALUES ('2026-01-05', 10, 2, 5, 1, 3), ('2026-01-09', 7, 1, 3, 0, 4)`))

	w, body := get(t, a, "/index.php?action=stats", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("stats status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{
		"Statistics Center",
		"Top 10 Posters",
		`id="stats"`,   // history table
		"January 2026", // month row from the seeded activity
	} {
		if !strings.Contains(body, want) {
			t.Errorf("stats page missing %q", want)
		}
	}
}

func TestStatsExpandXML(t *testing.T) {
	a := newTestApp(t)
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_activity (date, hits, topics, posts, registers, mostOn)
		VALUES ('2026-01-05', 10, 2, 5, 1, 3)`))

	sc, cookies := sescFromSendForm(t, a)

	w, body := get(t, a, "/index.php?action=stats;expand=202601;xml;sesc="+sc, cookies...)
	if w.Code != 200 {
		t.Fatalf("stats xml status %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/xml") {
		t.Errorf("stats xml Content-Type = %q", ct)
	}
	if !strings.Contains(body, `<month id="202601">`) {
		t.Errorf("stats xml missing month element:\n%s", body)
	}
	if !strings.Contains(body, `<day date="2026-01-05"`) {
		t.Errorf("stats xml missing day element:\n%s", body)
	}
}
