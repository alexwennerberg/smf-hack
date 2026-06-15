package app

// Screen goldens: byte-lock the rendered output of templates that the main
// TestGoldenCrawl doesn't reach because they need a dynamic URL (a message/topic
// id computed from the seed) or a mid-flow sub-action. These are read renders
// (no mutation), so they go through goldenApp like the crawl does.
//
// The motivating gap: tpl_splittopics.go was the only template with no byte
// snapshot — its split/merge *selection* UI never rendered in a test, so a
// template refactor there would not be caught. These cases close that.
//
// Regenerate: GOLDEN_UPDATE=1 go test ./internal/app/ -run TestScreenGoldens

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func screenGolden(t *testing.T, name, body string) {
	t.Helper()
	dir := filepath.Join("testdata", "golden_screens")
	if os.Getenv("GOLDEN_UPDATE") != "" {
		os.MkdirAll(dir, 0755)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(filepath.Join(dir, name+".html"))
	if err != nil {
		t.Fatalf("missing screen golden %s (run GOLDEN_UPDATE=1): %v", name, err)
	}
	if string(want) != body {
		t.Errorf("screen golden mismatch %s:\n%s", name, firstDiff(string(want), body))
	}
}

// renderScreen GETs a page with the given cookies and returns the normalized
// body, failing if the status isn't 200.
func renderScreen(t *testing.T, a *App, path string, cookies ...*http.Cookie) string {
	t.Helper()
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online`))
	w, body := get(t, a, path, cookies...)
	if w.Code != 200 {
		t.Fatalf("GET %s: status %d body %.300s", path, w.Code, w.Body.String())
	}
	return normalizeGolden(strings.ReplaceAll(body, a.Config.AssetDir, "{ASSETDIR}"))
}

// topic202Messages returns the message IDs of topic 202 (the long thread seeded
// by goldenApp) ordered by ID_MSG: [first post, reply 1, ..., reply 12].
func topic202Messages(t *testing.T, a *App) []int {
	t.Helper()
	rows, err := a.DB.Query(a.Q(`SELECT ID_MSG FROM {$db_prefix}messages WHERE ID_TOPIC = 202 ORDER BY ID_MSG`))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var m int
		rows.Scan(&m)
		ids = append(ids, m)
	}
	if len(ids) < 3 {
		t.Fatalf("topic 202 has %d messages, want >=3", len(ids))
	}
	return ids
}

// TestScreenGoldens locks the split/merge selection screens — the
// tpl_splittopics.go surface that no other test renders.
func TestScreenGoldens(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	msgs := topic202Messages(t, a)
	atSecond := itoa(msgs[1]) // a non-first message → the "how do you want to split?" ask screen

	// 1) Split "ask" screen: how to split, anchored at a reply.
	screenGolden(t, "split_ask",
		renderScreen(t, a, "/index.php?action=splittopics;topic=202.0;at="+atSecond, admin))

	// 2) Split selective message-picker (the two-column selected/not-selected UI).
	screenGolden(t, "split_select",
		renderScreen(t, a, "/index.php?action=splittopics;sa=selectTopics;topic=202.0;subname=Split+off+topic", admin))

	// 3) Merge index: choose a topic on board 1 to merge topic 202 into.
	screenGolden(t, "merge_index",
		renderScreen(t, a, "/index.php?action=mergetopics;board=1.0;from=202", admin))

	// 4) Split selective picker, XML (AJAX) variant → templateSplitXML.
	screenGolden(t, "split_select_xml",
		renderScreen(t, a, "/index.php?action=splittopics;sa=selectTopics;topic=202.0;xml;subname=Split+off+topic", admin))

	// 5) Merge "extra options" form (pick the kept subject) → templateMergeExtraOptions.
	//    MergeExecute checks the session, so carry a valid sesc + session cookie.
	sesc, adminCookies := mbForm(t, a, "/index.php?action=maintain", admin)
	screenGolden(t, "merge_options",
		renderScreen(t, a, "/index.php?action=mergetopics;sa=options;board=1.0;from=202;to=106;sesc="+sesc, adminCookies...))

	// 6) Merge "done" confirmation → templateMergeDone.
	screenGolden(t, "merge_done",
		renderScreen(t, a, "/index.php?action=mergetopics;sa=done;targetboard=1;to=202", admin))
}

// TestRegisterScreenGoldens locks the registration sub-screens that the plain
// register_guest crawl case doesn't reach: COPPA approval/contact-form pages and
// the activation retry/resend pages (all 0% covered before this).
func TestRegisterScreenGoldens(t *testing.T) {
	time.Local = time.UTC
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{
		"registration_method": "1", // member activation → enables resend page
		"coppaAge":            "13",
		"coppaType":           "1",
		"coppaPost":           "PO Box 7, Anytown",
		"coppaFax":            "555-0100",
		"coppaPhone":          "555-0199",
	})
	// A COPPA-pending account (is_activated = 5) for the approval screens.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
		(memberName, realName, passwd, emailAddress, ID_GROUP, dateRegistered, is_activated, ID_POST_GROUP)
		VALUES ('kiddo', 'Kid Doe', 'x', 'kid@b.com', 0, ?, 5, 4)`), goldenEpoch)
	var coppaID int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'kiddo'`)).Scan(&coppaID)

	screenGolden(t, "coppa_notice",
		renderScreen(t, a, "/index.php?action=coppa;member="+itoa(coppaID)))
	screenGolden(t, "coppa_form",
		renderScreen(t, a, "/index.php?action=coppa;member="+itoa(coppaID)+";form"))
	screenGolden(t, "activate_retry",
		renderScreen(t, a, "/index.php?action=activate;u=999999"))
	screenGolden(t, "activate_resend",
		renderScreen(t, a, "/index.php?action=activate"))
}
