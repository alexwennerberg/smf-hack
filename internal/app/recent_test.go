package app

// Smoke test for the recent-posts page (?action=recent).

import (
	"net/url"
	"strings"
	"testing"
)

func TestRecentPosts(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Post a topic so there's something recent to show.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {"A recent topic"},
		"message":            {"Body of the [b]recent[/b] post."},
		"icon":               {"xx"},
		"notify":             {"0"},
		"lock":               {"0"},
		"sticky":             {"0"},
		"move":               {"0"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 status %d", w.Code)
	}

	w, body := get(t, a, "/index.php?action=recent", admin)
	if w.Code != 200 {
		t.Fatalf("recent status %d:\n%.400s", w.Code, body)
	}
	if !strings.Contains(body, "A recent topic") {
		t.Errorf("recent page missing the topic subject:\n%.600s", body)
	}
	if !strings.Contains(body, "Body of the <b>recent</b> post.") {
		t.Errorf("recent page missing the parsed message body")
	}
	// The admin can reply/delete, so a button strip should render.
	if !strings.Contains(body, "?action=post;topic=") {
		t.Errorf("recent page missing reply button")
	}
}
