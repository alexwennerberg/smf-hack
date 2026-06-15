package app

// Phase 6: Modlog viewer — renders log_actions with topic links and the
// localized action description.

import (
	"strings"
	"testing"
)

func TestViewModlog(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	// Skip the admin-session password prompt (adminIndex calls validateSession).
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	topic := makeTopic(t, a, "Locked topic", admin)

	// A 'lock' action old enough to be deletable, tied to the topic.
	oldTime := nowUnix() - 48*3600
	if _, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_actions
		(logTime, ID_MEMBER, ip, action, extra)
		VALUES (?, ?, ?, ?, ?)`), oldTime, 1, "127.0.0.1", "lock", `{"topic":`+itoa(topic)+`}`); err != nil {
		t.Fatal(err)
	}

	_, body := get(t, a, "/index.php?action=modlog", admin)

	// The localized action description ("Locked" from Admin.english.php).
	if !strings.Contains(body, "Locked") {
		t.Errorf("modlog missing action description 'Locked':\n%.600s", body)
	}
	// The topic should be linked by subject.
	if !strings.Contains(body, "?topic="+itoa(topic)+".0") {
		t.Errorf("modlog missing topic link for topic %d", topic)
	}
	if !strings.Contains(body, "Locked topic") {
		t.Errorf("modlog missing linked topic subject")
	}
	// A delete checkbox for the (editable) entry, not disabled.
	if !strings.Contains(body, `name="delete[]"`) {
		t.Errorf("modlog missing delete checkbox")
	}
	if strings.Contains(body, `disabled="disabled"`) {
		t.Errorf("old entry should be deletable, found disabled checkbox")
	}
}
