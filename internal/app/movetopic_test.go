package app

// Phase 6: MoveTopic UI — the destination form lists boards and movetopic2
// relocates the topic.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestMoveTopic(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Create a topic in board 1.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {"Topic to be moved"},
		"message":            {"Move me please."},
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
		t.Fatalf("post2 new topic: status %d", w.Code)
	}
	var topicID int
	a.DB.QueryRow(a.Q(`SELECT MAX(ID_TOPIC) FROM {$db_prefix}topics`)).Scan(&topicID)

	// Add a second board to move into.
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}boards
		(ID_BOARD, ID_CAT, boardOrder, name, description, memberGroups)
		VALUES (2, 1, 2, 'Second Board', 'Another board.', '-1,0')`)); err != nil {
		t.Fatal(err)
	}

	// GET the move form: it should list the destination board.
	wForm, body := get(t, a, "/index.php?action=movetopic;topic="+itoa(topicID)+".0", admin)
	if wForm.Code != 200 {
		t.Fatalf("movetopic form: status %d body %.300s", wForm.Code, body)
	}
	if !strings.Contains(body, `<option value="2"`) || !strings.Contains(body, "Second Board") {
		t.Fatalf("movetopic form missing destination board: %.500s", body)
	}
	if strings.Contains(body, `<option value="1"`) {
		t.Fatalf("movetopic form should exclude the current board")
	}
	msc := scRe.FindStringSubmatch(body)
	mseq := seqRe.FindStringSubmatch(body)
	if msc == nil || mseq == nil {
		t.Fatalf("movetopic form missing sc/seqnum")
	}
	// Carry the session cookie so checkSession/seqnum match.
	moveCookies := append([]*http.Cookie{admin}, cookiesFrom(wForm)...)

	// Execute the move (no redirect post, no rename).
	wMove := postForm(t, a, "/index.php?action=movetopic2;topic="+itoa(topicID)+".0", url.Values{
		"toboard": {"2"},
		"sc":      {msc[1]},
		"seqnum":  {mseq[1]},
	}, moveCookies...)
	if wMove.Code != 302 {
		t.Fatalf("movetopic2: status %d body %.500s", wMove.Code, wMove.Body.String())
	}
	if loc := wMove.Header().Get("Location"); !strings.Contains(loc, "board=1.0") {
		t.Fatalf("movetopic2 redirect %q, want board=1.0", loc)
	}

	var board int
	a.DB.QueryRow(a.Q(`SELECT ID_BOARD FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topicID).Scan(&board)
	if board != 2 {
		t.Fatalf("topic board after move = %d, want 2", board)
	}
}
