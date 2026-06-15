package app

// Smoke test for getLastPosts via the board-index "recent posts" list
// (number_recent_posts > 1).

import (
	"net/url"
	"strings"
	"testing"
)

func TestBoardIndexLatestPostsList(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Turn on the multi-post recent bar for the default theme.
	if _, err := a.DB.Exec(a.Q(`INSERT OR REPLACE INTO {$db_prefix}themes
		(ID_THEME, ID_MEMBER, variable, value) VALUES (1, 0, 'number_recent_posts', '5')`)); err != nil {
		t.Fatal(err)
	}

	// Post a topic so there's a recent post to list.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic": {"0"}, "subject": {"Front page topic"}, "message": {"Hello."},
		"icon": {"xx"}, "notify": {"0"}, "lock": {"0"}, "sticky": {"0"},
		"move": {"0"}, "additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 status %d", w.Code)
	}

	_, body := get(t, a, "/index.php", admin)
	if !strings.Contains(body, "Front page topic") {
		t.Errorf("board index recent-posts list missing the topic:\n%.600s", body)
	}
	// The list markup (poster + board link in parens) should be present.
	if !strings.Contains(body, "?board=1.0\">General Discussion</a>)") &&
		!strings.Contains(body, "(<a href=") {
		t.Errorf("board index recent-posts list markup missing")
	}
}
