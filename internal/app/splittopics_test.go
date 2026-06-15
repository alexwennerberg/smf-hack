package app

// Phase 6: SplitTopics + MergeTopics end-to-end.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// postReply adds a reply to a topic and returns nothing; flood control is
// cleared beforehand.
func postReply(t *testing.T, a *App, topic int, message string, admin *http.Cookie) {
	t.Helper()
	clearFloodControl(a)
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;topic="+itoa(topic)+".0", admin)
	w := postForm(t, a, "/index.php?action=post2;topic="+itoa(topic)+".0;start=0;board=1.0", url.Values{
		"topic":              {itoa(topic)},
		"subject":            {"Re: reply"},
		"message":            {message},
		"icon":               {"xx"},
		"notify":             {"0"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post reply: status %d body %.300s", w.Code, w.Body.String())
	}
}

func TestSplitTopicAfterThis(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	topic := makeTopic(t, a, "Topic to split", admin)
	postReply(t, a, topic, "Second message.", admin)
	postReply(t, a, topic, "Third message.", admin)

	// The three message IDs in order.
	var msgs []int
	rows, _ := a.DB.Query(a.Q(`SELECT ID_MSG FROM {$db_prefix}messages WHERE ID_TOPIC = ? ORDER BY ID_MSG`), topic)
	for rows.Next() {
		var m int
		rows.Scan(&m)
		msgs = append(msgs, m)
	}
	rows.Close()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	// Split everything at/after the second message into a new topic.
	sc, _, cookies := openPostForm(t, a, "/index.php?action=post;topic="+itoa(topic)+".0", admin)
	w := postForm(t, a, "/index.php?action=splittopics;sa=execute;topic="+itoa(topic)+".0", url.Values{
		"subname": {"Split off topic"},
		"step2":   {"afterthis"},
		"at":      {itoa(msgs[1])},
		"sc":      {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("split execute: status %d body %.400s", w.Code, w.Body.String())
	}

	// A new topic now holds messages 2 and 3.
	var newTopic int
	a.DB.QueryRow(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}messages WHERE ID_MSG = ?`), msgs[2]).Scan(&newTopic)
	if newTopic == topic || newTopic == 0 {
		t.Fatalf("third message still on old topic (newTopic=%d, old=%d)", newTopic, topic)
	}
	var t2 int
	a.DB.QueryRow(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}messages WHERE ID_MSG = ?`), msgs[1]).Scan(&t2)
	if t2 != newTopic {
		t.Fatalf("second message went to topic %d, want %d", t2, newTopic)
	}
	// The first message stays on the original topic.
	var t0 int
	a.DB.QueryRow(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}messages WHERE ID_MSG = ?`), msgs[0]).Scan(&t0)
	if t0 != topic {
		t.Fatalf("first message left the old topic (now %d)", t0)
	}

	// Old topic: 0 replies, new topic: 1 reply.
	var oldReplies, newReplies int
	a.DB.QueryRow(a.Q(`SELECT numReplies FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topic).Scan(&oldReplies)
	a.DB.QueryRow(a.Q(`SELECT numReplies FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), newTopic).Scan(&newReplies)
	if oldReplies != 0 || newReplies != 1 {
		t.Fatalf("replies after split: old=%d (want 0), new=%d (want 1)", oldReplies, newReplies)
	}

	if !strings.Contains(w.Body.String(), "?topic="+itoa(newTopic)+".0") {
		t.Errorf("split-done page missing new topic link")
	}
}

func TestMergeTopicsExecute(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	t1 := makeTopic(t, a, "Alpha topic", admin)
	t2 := makeTopic(t, a, "Beta topic", admin)

	// Counts before.
	var msgCountBefore int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_TOPIC IN (?, ?)`), t1, t2).Scan(&msgCountBefore)

	sc, _, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=mergetopics;sa=execute;board=1.0", url.Values{
		"topics[]": {itoa(t1), itoa(t2)},
		"board":    {"1"},
		"subject":  {itoa(t1)}, // keep Alpha's subject
		"sc":       {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("merge execute: status %d body %.400s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "action=mergetopics;sa=done") {
		t.Fatalf("merge redirect %q, want sa=done", loc)
	}

	// The merged topic is the lower ID; the other topic is gone.
	merged := t1
	if t2 < t1 {
		merged = t2
	}
	gone := t1 + t2 - merged
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), gone).Scan(&n)
	if n != 0 {
		t.Fatalf("topic %d should have been deleted by merge", gone)
	}
	// All messages now live on the merged topic.
	var onMerged int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_TOPIC = ?`), merged).Scan(&onMerged)
	if onMerged != msgCountBefore {
		t.Fatalf("merged topic has %d messages, want %d", onMerged, msgCountBefore)
	}
	// numReplies reflects the combined post count.
	var replies int
	a.DB.QueryRow(a.Q(`SELECT numReplies FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), merged).Scan(&replies)
	if replies != msgCountBefore-1 {
		t.Fatalf("merged numReplies = %d, want %d", replies, msgCountBefore-1)
	}
}
