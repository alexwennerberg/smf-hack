package app

// Write-flow parity: perform a mutation (post, reply, vote, quickmod, PM) and
// capture the resulting read-page as a golden. The PHP parity harness
// (tools/parity) performs the SAME mutation against the live SMF reference and
// diffs the rendered result against these goldens — so we verify not just that
// the write path doesn't error, but that its DB side effects render identically
// to real SMF. Goldens live under testdata/golden_writeflow/.
//
// Regenerate: GOLDEN_UPDATE=1 go test ./internal/app/ -run TestWriteFlow

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func wfWriteGolden(t *testing.T, name, body string) {
	t.Helper()
	dir := filepath.Join("testdata", "golden_writeflow")
	if os.Getenv("GOLDEN_UPDATE") != "" {
		os.MkdirAll(dir, 0755)
		if err := os.WriteFile(filepath.Join(dir, name+".html"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(filepath.Join(dir, name+".html"))
	if err != nil {
		t.Fatalf("missing writeflow golden %s (run GOLDEN_UPDATE=1): %v", name, err)
	}
	if string(want) != body {
		t.Errorf("writeflow golden mismatch %s:\n%s", name, firstDiff(string(want), body))
	}
}

// render fetches a read page as a role and returns the normalized body.
func wfRender(t *testing.T, a *App, path, role string) string {
	t.Helper()
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online`))
	var cookies []*http.Cookie
	switch role {
	case "member":
		cookies = []*http.Cookie{goldenLoginCookie(t, a, "member")}
	case "admin":
		cookies = []*http.Cookie{adminCookie(t, a)}
	}
	_, body := get(t, a, path, cookies...)
	return normalizeGolden(strings.ReplaceAll(body, a.Config.AssetDir, "{ASSETDIR}"))
}

func TestWriteFlowReply(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	clearFloodControl(a)
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;topic=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;topic=1.0;start=0;board=1.0", url.Values{
		"topic": {"1"}, "subject": {"Re: Welcome to SMF!"},
		"message": {"A parity reply body."}, "icon": {"xx"},
		"notify": {"0"}, "lock": {"0"}, "sticky": {"0"}, "move": {"0"},
		"additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("reply post2: status %d body %.300s", w.Code, w.Body.String())
	}
	wfWriteGolden(t, "reply_topic1", wfRender(t, a, "/index.php?topic=1.0", "guest"))
}

func TestWriteFlowNewTopic(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	clearFloodControl(a)
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic": {"0"}, "subject": {"Parity new topic"},
		"message": {"Parity new topic body."}, "icon": {"xx"},
		"notify": {"0"}, "lock": {"0"}, "sticky": {"0"}, "move": {"0"},
		"additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("new topic post2: status %d body %.300s", w.Code, w.Body.String())
	}
	// The new topic is the highest ID; render the board index (page 1) where it
	// appears at the top (newest), and the topic itself.
	var tid int
	a.DB.QueryRow(a.Q(`SELECT MAX(ID_TOPIC) FROM {$db_prefix}topics`)).Scan(&tid)
	wfWriteGolden(t, "newtopic_board1", wfRender(t, a, "/index.php?board=1.0", "guest"))
	wfWriteGolden(t, "newtopic_display", wfRender(t, a, "/index.php?topic="+itoa(tid)+".0", "guest"))
}

func TestWriteFlowPollVote(t *testing.T) {
	a := goldenApp(t)
	member := goldenLoginCookie(t, a, "member")
	// Member votes for choice 1 (Green). vote posts to action=vote;topic=203.
	sc, _, cookies := openVoteForm(t, a, member)
	w := postForm(t, a, "/index.php?action=vote;topic=203", url.Values{
		"options[]": {"1"}, "sc": {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("poll vote: status %d body %.300s", w.Code, w.Body.String())
	}
	wfWriteGolden(t, "pollvote_topic203", wfRender(t, a, "/index.php?topic=203.0", "member"))
}

// openVoteForm grabs an sc + session for the member by rendering the poll topic.
func openVoteForm(t *testing.T, a *App, member *http.Cookie) (string, string, []*http.Cookie) {
	t.Helper()
	w, body := get(t, a, "/index.php?topic=203.0", member)
	if w.Code != 200 {
		t.Fatalf("GET poll topic: %d", w.Code)
	}
	m := scRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no sc in poll topic")
	}
	return m[1], "", append([]*http.Cookie{member}, cookiesFrom(w)...)
}

func TestWriteFlowQuickmodLock(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	sc, _, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=quickmod;board=1.0", url.Values{
		"topics[]": {"101"}, "qaction": {"lock"}, "sc": {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("quickmod lock: status %d body %.300s", w.Code, w.Body.String())
	}
	wfWriteGolden(t, "quickmod_lock_board1", wfRender(t, a, "/index.php?board=1.0", "guest"))
}

func TestWriteFlowPMSend(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	sc, _, cookies := pmSendForm(t, a, admin)
	w := postForm(t, a, "/index.php?action=pm;sa=send2", url.Values{
		"to": {"Member One"}, "subject": {"Parity PM"},
		"message": {"Parity PM body."}, "sc": {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("pm send2: status %d body %.300s", w.Code, w.Body.String())
	}
	// Render the recipient's inbox.
	wfWriteGolden(t, "pm_inbox_after_send", wfRender(t, a, "/index.php?action=pm", "member"))
}

func pmSendForm(t *testing.T, a *App, admin *http.Cookie) (string, string, []*http.Cookie) {
	t.Helper()
	w, body := get(t, a, "/index.php?action=pm;sa=send", admin)
	if w.Code != 200 {
		t.Fatalf("GET pm send form: %d", w.Code)
	}
	m := scRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no sc in pm form")
	}
	return m[1], "", append([]*http.Cookie{admin}, cookiesFrom(w)...)
}

func TestWriteFlowEditPost(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	clearFloodControl(a)
	// The edit form requires a valid sesc; establish a session + sc first.
	w0, b0 := get(t, a, "/index.php?topic=101.0", admin)
	sc0 := scRe.FindStringSubmatch(b0)
	if w0.Code != 200 || sc0 == nil {
		t.Fatalf("GET topic 101 for sc: %d", w0.Code)
	}
	sess := append([]*http.Cookie{admin}, cookiesFrom(w0)...)
	// Edit topic 101's first message (msg 2): body changes, "Last Edit" appears.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;msg=2;topic=101.0;sesc="+sc0[1], sess...)
	w := postForm(t, a, "/index.php?action=post2;start=0;msg=2;board=1.0", url.Values{
		"topic": {"101"}, "subject": {"Topic number 01"}, "message": {"Edited body."},
		"icon": {"xx"}, "notify": {"0"}, "additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("edit post2: status %d body %.300s", w.Code, w.Body.String())
	}
	wfWriteGolden(t, "edit_topic101", wfRender(t, a, "/index.php?topic=101.0", "guest"))
}

func TestWriteFlowDeleteReply(t *testing.T) {
	a := goldenApp(t)
	member := goldenLoginCookie(t, a, "member")
	admin := adminCookie(t, a)
	// Scrape an sc via the topic display, then delete reply msg 17 of topic 202.
	w0, body := get(t, a, "/index.php?topic=202.0", admin)
	if w0.Code != 200 {
		t.Fatalf("GET topic 202: %d", w0.Code)
	}
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatal("no sc on topic 202")
	}
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w0)...)
	w := mustGet(t, a, "/index.php?action=deletemsg;topic=202.0;msg=17;sesc="+sc[1], cookies...)
	if w.Code != 302 {
		t.Fatalf("deletemsg: status %d body %.300s", w.Code, w.Body.String())
	}
	_ = member
	wfWriteGolden(t, "delete_reply_topic202", wfRender(t, a, "/index.php?topic=202.0", "guest"))
}

func TestWriteFlowMoveTopic(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	wForm, body := get(t, a, "/index.php?action=movetopic;topic=101.0", admin)
	if wForm.Code != 200 {
		t.Fatalf("GET movetopic form: %d", wForm.Code)
	}
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatal("no sc on movetopic form")
	}
	cookies := append([]*http.Cookie{admin}, cookiesFrom(wForm)...)
	w := postForm(t, a, "/index.php?action=movetopic2;topic=101.0", url.Values{
		"toboard": {"3"}, "sc": {sc[1]},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("movetopic2: status %d body %.300s", w.Code, w.Body.String())
	}
	wfWriteGolden(t, "move_board1_after", wfRender(t, a, "/index.php?board=1.0", "guest"))
	wfWriteGolden(t, "move_board3_after", wfRender(t, a, "/index.php?board=3.0", "guest"))
}

func TestWriteFlowCensorSave(t *testing.T) {
	a := goldenApp(t)
	admin := adminCookie(t, a)
	wForm, body := get(t, a, "/index.php?action=postsettings;sa=censor", admin)
	if wForm.Code != 200 {
		t.Fatalf("GET censor form: %d", wForm.Code)
	}
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatal("no sc on censor form")
	}
	cookies := append([]*http.Cookie{admin}, cookiesFrom(wForm)...)
	w := postForm(t, a, "/index.php?action=postsettings;sa=censor", url.Values{
		"save_censor": {"1"}, "censortext": {"badword=goodword"}, "sc": {sc[1]},
	}, cookies...)
	if w.Code != 200 { // SetCensor persists then re-renders (no redirect)
		t.Fatalf("censor save: status %d body %.300s", w.Code, w.Body.String())
	}
	wfWriteGolden(t, "censor_after_save", wfRender(t, a, "/index.php?action=postsettings;sa=censor", "admin"))
}

func mustGet(t *testing.T, a *App, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	w, _ := get(t, a, path, cookies...)
	return w
}
