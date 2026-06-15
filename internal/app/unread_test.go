package app

// Smoke tests for ?action=unread and ?action=unreadreplies.

import (
	"strings"
	"testing"
	"time"
)

// seedUnreadTopic inserts a topic/message posted by someone other than the
// admin, with no read markers, so it shows up as unread for the admin.
func seedUnreadTopic(t *testing.T, a *App) {
	t.Helper()
	now := time.Now().Unix()
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages
		(ID_MSG, ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterTime, subject, body, icon, smileysEnabled)
		VALUES (100, 50, 1, 2, 'bob', ?, 'An unread topic', 'Some body text.', 'xx', 1)`), now); err != nil {
		t.Fatal(err)
	}
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}topics
		(ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, numReplies, numViews)
		VALUES (50, 1, 100, 100, 2, 2, 0, 0)`)); err != nil {
		t.Fatal(err)
	}
	a.UpdateSettings(map[string]string{"maxMsgID": "100", "totalMessages": "1", "totalTopics": "1"})
}

func TestUnreadTopics(t *testing.T) {
	a := newTestApp(t)
	seedUnreadTopic(t, a)

	w, body := get(t, a, "/index.php?action=unread", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("unread status %d:\n%.400s", w.Code, body)
	}
	if !strings.Contains(body, "An unread topic") {
		t.Errorf("unread listing missing the topic:\n%.800s", body)
	}
	// Sort headers and the mark-read button should be present.
	if !strings.Contains(body, "?action=unread;all") {
		t.Errorf("unread listing missing the 'show all' link")
	}
	if !strings.Contains(body, "action=markasread;sa=all") {
		t.Errorf("unread listing missing the mark-read button")
	}
}

func TestUnreadTopicsAll(t *testing.T) {
	a := newTestApp(t)
	seedUnreadTopic(t, a)

	w, body := get(t, a, "/index.php?action=unread;all", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("unread;all status %d", w.Code)
	}
	if !strings.Contains(body, "An unread topic") {
		t.Errorf("unread;all listing missing the topic")
	}
}

func TestUnreadRepliesPositive(t *testing.T) {
	a := newTestApp(t)
	now := time.Now().Unix()
	// Admin (member 1) starts the topic; bob (member 2) has the last post.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages
		(ID_MSG, ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterTime, subject, body, icon, smileysEnabled)
		VALUES (100, 60, 1, 1, 'admin', ?, 'A topic I posted in', 'My post.', 'xx', 1),
		       (101, 60, 1, 2, 'bob', ?, 'Re: A topic I posted in', 'A reply.', 'xx', 1)`), now, now+1)
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}topics
		(ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, numReplies, numViews)
		VALUES (60, 1, 100, 101, 1, 2, 1, 0)`))
	a.UpdateSettings(map[string]string{"maxMsgID": "101", "totalMessages": "2", "totalTopics": "1"})

	w, body := get(t, a, "/index.php?action=unreadreplies", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("unreadreplies status %d:\n%.400s", w.Code, body)
	}
	if !strings.Contains(body, "A topic I posted in") {
		t.Errorf("unreadreplies should list the topic the admin posted in:\n%.800s", body)
	}
	if !strings.Contains(body, "action=markasread;sa=unreadreplies") {
		t.Errorf("unreadreplies missing its mark-read button")
	}
}

func TestUnreadRepliesEmpty(t *testing.T) {
	a := newTestApp(t)
	seedUnreadTopic(t, a)

	// The admin hasn't posted in any topic, so unreadreplies is empty but
	// should still render its (no-topics) page.
	w, body := get(t, a, "/index.php?action=unreadreplies", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("unreadreplies status %d:\n%.400s", w.Code, body)
	}
	if strings.Contains(body, "An unread topic") {
		t.Errorf("unreadreplies should not list a topic the admin never posted in")
	}
}
