package app

// Phase 7: news admin (edit news + settings).

import (
	"net/url"
	"strings"
	"testing"
)

func TestEditNewsSaveAndList(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=news;sa=editnews", admin)
	w := postForm(t, a, "/index.php?action=news;sa=editnews", url.Values{
		"save_items": {"1"},
		"news[]":     {"First announcement", "Second [b]bold[/b] item"},
		"sc":         {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save news: status %d", w.Code)
	}
	if got := a.Setting("news"); !strings.Contains(got, "First announcement") || !strings.Contains(got, "Second") {
		t.Fatalf("news not saved: %q", got)
	}

	_, body := get(t, a, "/index.php?action=news;sa=editnews", admin)
	if !strings.Contains(body, "First announcement") {
		t.Errorf("news item not listed")
	}
	// BBC is parsed in the preview column.
	if !strings.Contains(body, "<b>bold</b>") {
		t.Errorf("news preview not BBC-parsed:\n%.300s", body)
	}
}

func TestNewsSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=news;sa=settings", admin)
	w := postForm(t, a, "/index.php?action=news;sa=settings", url.Values{
		"save_settings":  {"1"},
		"xmlnews_enable": {"on"},
		"xmlnews_maxlen": {"500"},
		"edit_news[2]":   {"on"},
		"sc":             {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save news settings: status %d", w.Code)
	}
	if a.Setting("xmlnews_enable") != "1" || a.Setting("xmlnews_maxlen") != "500" {
		t.Fatalf("news settings not saved")
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 2 AND permission = 'edit_news'`)).Scan(&n)
	if n != 1 {
		t.Errorf("edit_news inline permission not granted to group 2")
	}
}

func TestNewsletterFlow(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	var adminEmail string
	a.DB.QueryRow(a.Q(`SELECT emailAddress FROM {$db_prefix}members WHERE ID_MEMBER = 1`)).Scan(&adminEmail)
	if adminEmail == "" {
		t.Fatal("admin has no email")
	}

	// --- The group picker renders. ---
	w, body := get(t, a, "/index.php?action=news;sa=mailingmembers", admin)
	if w.Code != 200 {
		t.Fatalf("mailingmembers status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{`?action=news;sa=mailingcompose`, `name="who[1]"`, `Administrator`, `id="checkAllGroups"`} {
		if !strings.Contains(body, want) {
			t.Errorf("group picker missing %q", want)
		}
	}

	// --- Compose: select the Administrator group, get the address list. ---
	sc, cookies := mbForm(t, a, "/index.php?action=news;sa=mailingmembers", admin)
	w = postForm(t, a, "/index.php?action=news;sa=mailingcompose", url.Values{
		"who[1]": {"1"},
		"sc":     {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("mailingcompose status %d", w.Code)
	}
	body = w.Body.String()
	if !strings.Contains(body, adminEmail) {
		t.Errorf("compose form missing admin email %q:\n%.500s", adminEmail, body)
	}
	if !strings.Contains(body, `name="emails"`) || !strings.Contains(body, `name="message"`) {
		t.Errorf("compose form incomplete")
	}

	// --- Send: a tiny list finishes in one batch and redirects to admin. ---
	w = postForm(t, a, "/index.php?action=news;sa=mailingsend", url.Values{
		"emails":  {"a@example.com; b@example.com"},
		"subject": {"Hello {$latest_member.name}"},
		"message": {"News for {$member.name} at {$board_url}"},
		"start":   {"0"},
		"sc":      {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("mailingsend status %d body %.400s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "action=admin") {
		t.Fatalf("mailingsend redirect %q, want action=admin", loc)
	}
}
