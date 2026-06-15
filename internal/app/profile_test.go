package app

// Profile tests (Phase 4).

import (
	"net/url"
	"strings"
	"testing"
)

func TestProfileViews(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Summary renders.
	w, body := get(t, a, "/index.php?action=profile;u=1", admin)
	if w.Code != 200 {
		t.Fatalf("profile summary: status %d", w.Code)
	}
	for _, want := range []string{"Profile Info", "Summary", "Show Stats", "Show Posts", "admin", "Date Registered"} {
		if !strings.Contains(body, want) {
			t.Errorf("summary missing %q", want)
		}
	}

	// Stat panel.
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=statPanel", admin)
	if !strings.Contains(body, "Total Time Spent Online") {
		t.Errorf("statPanel missing stats: %.200s", body)
	}

	// Show posts (the welcome post is a guest post, so none for admin).
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=showPosts", admin)
	if !strings.Contains(body, "Show Posts") {
		t.Error("showPosts missing header")
	}

	// Account settings form.
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=account", admin)
	if !strings.Contains(body, `name="emailAddress"`) || !strings.Contains(body, `name="oldpasswrd"`) {
		t.Errorf("account form incomplete")
	}

	// Forum profile form.
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=forumProfile", admin)
	if !strings.Contains(body, `name="signature"`) {
		t.Error("forumProfile form incomplete")
	}

	// Theme settings.
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=theme", admin)
	if !strings.Contains(body, `name="timeFormat"`) {
		t.Error("theme form incomplete")
	}

	// Notification settings.
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=notification", admin)
	if !strings.Contains(body, `name="notifyTypes"`) {
		t.Error("notification form incomplete")
	}

	// PM prefs.
	_, body = get(t, a, "/index.php?action=profile;u=1;sa=pmprefs", admin)
	if !strings.Contains(body, `name="pm_ignore_list"`) {
		t.Error("pmprefs form incomplete")
	}

	// Guests can view profiles too (profile_view_any is a guest default).
	w, body = get(t, a, "/index.php?action=profile;u=1")
	if w.Code != 200 || !strings.Contains(body, "Date Registered") {
		t.Error("guest profile view failed")
	}
}

func TestProfileSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Get the session token.
	w, body := get(t, a, "/index.php?action=profile;u=1;sa=forumProfile", admin)
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatalf("no sc in forumProfile form")
	}
	allCookies := append(cookiesFrom(w), admin)

	// Save a new signature + location through forumProfile.
	w2 := postForm(t, a, "/index.php?action=profile2", url.Values{
		"sa":           {"forumProfile"},
		"userID":       {"1"},
		"sc":           {sc[1]},
		"personalText": {"I am the admin"},
		"location":     {"Test City"},
		"signature":    {"My [b]signature[/b]"},
		"gender":       {"1"},
		"websiteTitle": {""},
		"websiteUrl":   {""},
		"ICQ":          {""},
		"AIM":          {""},
		"MSN":          {""},
		"YIM":          {""},
		"bday1":        {""}, "bday2": {""}, "bday3": {""},
	}, allCookies...)
	if w2.Code != 302 {
		t.Fatalf("profile2 save: status %d body %.400s", w2.Code, w2.Body.String())
	}

	var location, personalText, signature string
	a.DB.QueryRow(a.Q(`SELECT location, personalText, signature FROM {$db_prefix}members WHERE ID_MEMBER = 1`)).
		Scan(&location, &personalText, &signature)
	if location != "Test City" || personalText != "I am the admin" {
		t.Errorf("saved: location=%q personalText=%q", location, personalText)
	}
	if !strings.Contains(signature, "[b]signature[/b]") {
		t.Errorf("signature = %q", signature)
	}

	// The summary now shows them.
	_, body = get(t, a, "/index.php?action=profile;u=1", admin)
	if !strings.Contains(body, "Test City") || !strings.Contains(body, "I am the admin") {
		t.Error("saved values missing from summary")
	}

	// Password-protected account change with wrong password re-renders with
	// an error.
	w3, body3 := get(t, a, "/index.php?action=profile;u=1;sa=account", admin)
	sc2 := scRe.FindStringSubmatch(body3)
	allCookies = append(cookiesFrom(w3), admin)
	w4 := postForm(t, a, "/index.php?action=profile2", url.Values{
		"sa":           {"account"},
		"userID":       {"1"},
		"sc":           {sc2[1]},
		"emailAddress": {"new@example.com"},
		"oldpasswrd":   {"wrongpass"},
	}, allCookies...)
	if w4.Code != 200 || !strings.Contains(w4.Body.String(), "The password you entered was not correct") {
		t.Errorf("expected bad-password error, status=%d", w4.Code)
	}

	// And with the right password it saves.
	w5, body5 := get(t, a, "/index.php?action=profile;u=1;sa=account", admin)
	sc3 := scRe.FindStringSubmatch(body5)
	allCookies = append(cookiesFrom(w5), admin)
	w6 := postForm(t, a, "/index.php?action=profile2", url.Values{
		"sa":           {"account"},
		"userID":       {"1"},
		"sc":           {sc3[1]},
		"emailAddress": {"new@example.com"},
		"oldpasswrd":   {"testpass"},
	}, allCookies...)
	if w6.Code != 302 {
		t.Fatalf("account save: status %d body %.300s", w6.Code, w6.Body.String())
	}
	var email string
	a.DB.QueryRow(a.Q(`SELECT emailAddress FROM {$db_prefix}members WHERE ID_MEMBER = 1`)).Scan(&email)
	if email != "new@example.com" {
		t.Errorf("email = %q", email)
	}
}
