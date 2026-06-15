package app

// Phase 7: post/topic/bbc/censor settings.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestPostSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=postsettings;sa=posts", admin)
	w := postForm(t, a, "/index.php?action=postsettings;sa=posts", url.Values{
		"save_settings":      {"1"},
		"removeNestedQuotes": {"on"},
		"max_messageLength":  {"20000"},
		"spamWaitTime":       {"7"},
		"edit_disable_time":  {"15"},
		"sc":                 {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save post settings: status %d", w.Code)
	}
	if a.Setting("spamWaitTime") != "7" || a.Setting("max_messageLength") != "20000" || a.Setting("removeNestedQuotes") != "1" {
		t.Fatalf("post settings not saved: spamWaitTime=%q max=%q nested=%q",
			a.Setting("spamWaitTime"), a.Setting("max_messageLength"), a.Setting("removeNestedQuotes"))
	}
}

func TestTopicSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=postsettings;sa=topics", admin)
	w := postForm(t, a, "/index.php?action=postsettings;sa=topics", url.Values{
		"save_settings":      {"1"},
		"enableStickyTopics": {"on"},
		"hotTopicPosts":      {"25"},
		"defaultMaxTopics":   {"30"},
		"sc":                 {sc},
	}, cookies...)
	if w.Code != 200 && w.Code != 302 {
		t.Fatalf("save topic settings: status %d", w.Code)
	}
	if a.Setting("hotTopicPosts") != "25" || a.Setting("enableStickyTopics") != "1" {
		t.Fatalf("topic settings not saved")
	}
}

func TestCensorWords(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=postsettings;sa=censor", admin)
	w := postForm(t, a, "/index.php?action=postsettings;sa=censor", url.Values{
		"save_censor":     {"1"},
		"censor_vulgar[]": {"badword"},
		"censor_proper[]": {"niceword"},
		"sc":              {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("save censor: status %d", w.Code)
	}
	if a.Setting("censor_vulgar") != "badword" || a.Setting("censor_proper") != "niceword" {
		t.Fatalf("censor not saved: vulgar=%q proper=%q", a.Setting("censor_vulgar"), a.Setting("censor_proper"))
	}
	// The saved word renders back in the form.
	_, body := get(t, a, "/index.php?action=postsettings;sa=censor", admin)
	if !strings.Contains(body, `value="badword"`) {
		t.Errorf("censor word not shown in form")
	}
}

func TestBBCSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	w, body := get(t, a, "/index.php?action=postsettings;sa=bbc", admin)
	if w.Code != 200 || !strings.Contains(body, `name="enabledTags[]"`) {
		t.Fatalf("bbc settings render: status %d", w.Code)
	}
	sc := scRe.FindStringSubmatch(body)
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)

	// Enable only [b] and [i] -> everything else becomes disabled.
	wp := postForm(t, a, "/index.php?action=postsettings;sa=bbc", url.Values{
		"save_settings": {"1"},
		"enableBBC":     {"on"},
		"enabledTags[]": {"b", "i"},
		"sc":            {sc[1]},
	}, cookies...)
	if wp.Code != 200 && wp.Code != 302 {
		t.Fatalf("save bbc: status %d", wp.Code)
	}
	disabled := a.Setting("disabledBBC")
	if disabled == "" || strings.Contains(","+disabled+",", ",b,") {
		t.Fatalf("disabledBBC = %q; expected b not disabled and others disabled", disabled)
	}
	if !strings.Contains(","+disabled+",", ",url,") {
		t.Errorf("expected url to be disabled, got %q", disabled)
	}
}
