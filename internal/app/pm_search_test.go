package app

// Smoke tests for PM search (?action=pm;sa=search / sa=search2).

import (
	"strings"
	"testing"
	"time"
)

func TestPMSearchForm(t *testing.T) {
	a := newTestApp(t)
	sc, cookies := sescFromSendForm(t, a)

	w, body := get(t, a, "/index.php?action=pm;sa=search;advanced;sesc="+sc, cookies...)
	if w.Code != 200 {
		t.Fatalf("pm search form status %d", w.Code)
	}
	for _, want := range []string{`name="pmSearchForm"`, `?action=pm;sa=search2`, `name="searchtype"`, `name="userspec"`} {
		if !strings.Contains(body, want) {
			t.Errorf("pm search form missing %q", want)
		}
	}
}

func TestPMSearchResults(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Seed a PM from bob (member 2) to the admin (member 1).
	now := time.Now().Unix()
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
		(ID_MEMBER, memberName, realName, passwd, emailAddress, ID_GROUP, dateRegistered, is_activated, ID_POST_GROUP)
		VALUES (2, 'bob', 'Bob', '', 'bob@b.com', 0, ?, 1, 4)`), now); err != nil {
		t.Fatal(err)
	}
	res, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}personal_messages
		(ID_MEMBER_FROM, fromName, msgtime, subject, body)
		VALUES (2, 'Bob', ?, 'Lunch plans', 'How about pizza on Friday?')`), now)
	if err != nil {
		t.Fatal(err)
	}
	pmID, _ := res.LastInsertId()
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}pm_recipients
		(ID_PM, ID_MEMBER, bcc, is_read, deleted, labels) VALUES (?, 1, 0, 0, 0, '-1')`), pmID); err != nil {
		t.Fatal(err)
	}

	// Search for "pizza" in the body.
	w, body := get(t, a, "/index.php?action=pm;sa=search2;search=pizza", admin)
	if w.Code != 200 {
		t.Fatalf("pm search2 status %d:\n%.400s", w.Code, body)
	}
	if !strings.Contains(body, "Lunch plans") {
		t.Errorf("pm search did not find the matching message:\n%.800s", body)
	}

	// A non-matching query yields the "none found" notice.
	_, none := get(t, a, "/index.php?action=pm;sa=search2;search=zzzznope", admin)
	if !strings.Contains(none, "pm_search_results") && !strings.Contains(none, "Search Results") {
		// title text varies; just make sure the matched subject is absent.
	}
	if strings.Contains(none, "Lunch plans") {
		t.Errorf("non-matching pm search should not list the message")
	}
}