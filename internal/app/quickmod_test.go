package app

// Phase 6: QuickModeration — batch lock from the message index, and in-topic
// quickmod2 message deletion.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// makeTopic posts a new topic in board 1 and returns its ID.
func makeTopic(t *testing.T, a *App, subject string, admin *http.Cookie) int {
	t.Helper()
	clearFloodControl(a)
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {subject},
		"message":            {"Body of " + subject},
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
		t.Fatalf("post2 new topic %q: status %d", subject, w.Code)
	}
	var id int
	a.DB.QueryRow(a.Q(`SELECT MAX(ID_TOPIC) FROM {$db_prefix}topics`)).Scan(&id)
	return id
}

func TestQuickModerationLock(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	t1 := makeTopic(t, a, "First topic", admin)
	t2 := makeTopic(t, a, "Second topic", admin)

	// Grab an sc + session cookie via a form GET.
	sc, _, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)

	w := postForm(t, a, "/index.php?action=quickmod;board=1.0", url.Values{
		"topics[]": {itoa(t1), itoa(t2)},
		"qaction":  {"lock"},
		"sc":       {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("quickmod lock: status %d body %.400s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "board=1.0") {
		t.Fatalf("quickmod redirect %q, want board=1.0", loc)
	}

	for _, id := range []int{t1, t2} {
		var locked int
		a.DB.QueryRow(a.Q(`SELECT locked FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), id).Scan(&locked)
		if locked != 1 {
			t.Fatalf("topic %d locked = %d, want 1", id, locked)
		}
	}

	// Toggle again -> unlocked.
	w2 := postForm(t, a, "/index.php?action=quickmod;board=1.0", url.Values{
		"topics[]": {itoa(t1)},
		"qaction":  {"lock"},
		"sc":       {sc},
	}, cookies...)
	if w2.Code != 302 {
		t.Fatalf("quickmod unlock: status %d", w2.Code)
	}
	var locked int
	a.DB.QueryRow(a.Q(`SELECT locked FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), t1).Scan(&locked)
	if locked != 0 {
		t.Fatalf("topic %d after second toggle locked = %d, want 0", t1, locked)
	}
}

func TestQuickModeration2DeleteMessage(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	topic := makeTopic(t, a, "Topic with replies", admin)

	// Post a reply so there's a non-first message to delete.
	clearFloodControl(a)
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;topic="+itoa(topic)+".0", admin)
	w := postForm(t, a, "/index.php?action=post2;topic="+itoa(topic)+".0;start=0;board=1.0", url.Values{
		"topic":              {itoa(topic)},
		"subject":            {"Re: reply"},
		"message":            {"A reply to delete."},
		"icon":               {"xx"},
		"notify":             {"0"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post reply: status %d body %.400s", w.Code, w.Body.String())
	}

	var firstMsg, lastMsg int
	a.DB.QueryRow(a.Q(`SELECT ID_FIRST_MSG, ID_LAST_MSG FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topic).Scan(&firstMsg, &lastMsg)
	if firstMsg == lastMsg {
		t.Fatalf("reply did not create a second message (first=%d last=%d)", firstMsg, lastMsg)
	}

	sc2, _, cookies2 := openPostForm(t, a, "/index.php?action=post;topic="+itoa(topic)+".0", admin)
	wd := postForm(t, a, "/index.php?action=quickmod2;topic="+itoa(topic)+".0;board=1.0", url.Values{
		"msgs[]": {itoa(lastMsg)},
		"sc":     {sc2},
	}, cookies2...)
	if wd.Code != 302 {
		t.Fatalf("quickmod2 delete: status %d body %.400s", wd.Code, wd.Body.String())
	}

	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_MSG = ?`), lastMsg).Scan(&n)
	if n != 0 {
		t.Fatalf("message %d still present after quickmod2 delete", lastMsg)
	}
	// First message must survive.
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_MSG = ?`), firstMsg).Scan(&n)
	if n != 1 {
		t.Fatalf("first message %d was wrongly deleted", firstMsg)
	}
}
