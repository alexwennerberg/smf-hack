package app

// Smoke tests for the PersonalMessage templates: render the folder, send,
// prune, labels and ask-delete pages as a logged-in member and assert they
// come back 200 without panicking, with a few stable markers present.

import (
	"strings"
	"testing"
)

func pmGet(t *testing.T, a *App, path string) string {
	t.Helper()
	w, body := get(t, a, path, adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("GET %s: status %d\n%s", path, w.Code, body)
	}
	return body
}

func TestPMFolder(t *testing.T) {
	a := newTestApp(t)
	body := pmGet(t, a, "/index.php?action=pm")

	// The sidebar (pm_above layer) and the empty-inbox notice.
	if !strings.Contains(body, `name="pmFolder"`) {
		t.Errorf("folder form missing:\n%s", body)
	}
	for _, want := range []string{"Messages", "Inbox"} {
		if !strings.Contains(body, want) {
			t.Errorf("folder page missing %q", want)
		}
	}
	// No messages yet: the "no messages" row should appear and there should
	// be no message anchors.
	if strings.Contains(body, `<a name="msg`) {
		t.Errorf("unexpected message anchor in empty inbox")
	}
}

func TestPMSendForm(t *testing.T) {
	a := newTestApp(t)
	body := pmGet(t, a, "/index.php?action=pm;sa=send")

	for _, want := range []string{
		`name="postmodify"`,
		`name="to"`,
		`name="bcc"`,
		`name="subject"`,
		`?action=pm;sa=send2`,
		`new autocompleter("to")`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("send form missing %q", want)
		}
	}
}

func TestPMPruneAndLabels(t *testing.T) {
	a := newTestApp(t)

	prune := pmGet(t, a, "/index.php?action=pm;sa=prune")
	if !strings.Contains(prune, `?action=pm;sa=prune`) || !strings.Contains(prune, `name="age"`) {
		t.Errorf("prune page missing markers:\n%s", prune)
	}

	labels := pmGet(t, a, "/index.php?action=pm;sa=manlabels")
	if !strings.Contains(labels, `?action=pm;sa=manlabels`) || !strings.Contains(labels, `name="label"`) {
		t.Errorf("labels page missing markers:\n%s", labels)
	}
}

func TestPMAskDelete(t *testing.T) {
	a := newTestApp(t)
	body := pmGet(t, a, "/index.php?action=pm;sa=removeall")
	if !strings.Contains(body, `?action=pm;sa=removeall2`) {
		t.Errorf("ask-delete page missing confirm link:\n%s", body)
	}
}

func TestPMSendByID(t *testing.T) {
	a := newTestApp(t)

	// Add a second member to address the PM to.
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
		(memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
		VALUES ('bob', 'bob', '', 'bob@b.com', 0, 0, 0, 0, 1, 4)`)); err != nil {
		t.Fatal(err)
	}

	// The ?u= path prefills the "To" field with the member's realName.
	body := pmGet(t, a, "/index.php?action=pm;sa=send;u=2")
	if !strings.Contains(body, `name="to"`) {
		t.Fatalf("send-by-id form did not render:\n%s", body)
	}
	if !strings.Contains(body, `value="&quot;bob&quot;"`) {
		t.Errorf("send-by-id did not prefill recipient:\n%s", body)
	}
}
