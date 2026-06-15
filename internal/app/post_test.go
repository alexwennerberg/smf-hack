package app

// End-to-end posting flow tests (Phase 3): new topic, reply, edit, preview.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

var (
	scRe  = regexp.MustCompile(`name="sc" value="([0-9a-f]+)"`)
	seqRe = regexp.MustCompile(`name="seqnum" value="(\d+)"`)
)

// openPostForm GETs a post form and returns (sc, seqnum, cookies-for-next).
func openPostForm(t *testing.T, a *App, path string, cookies ...*http.Cookie) (string, string, []*http.Cookie) {
	t.Helper()
	w, body := get(t, a, path, cookies...)
	if w.Code != 200 {
		t.Fatalf("GET %s: status %d", path, w.Code)
	}
	if !strings.Contains(body, `id="postmodify"`) {
		t.Fatalf("GET %s: no postmodify form: %.300s", path, body)
	}
	sc := scRe.FindStringSubmatch(body)
	seq := seqRe.FindStringSubmatch(body)
	if sc == nil || seq == nil {
		t.Fatalf("GET %s: missing sc/seqnum in form", path)
	}
	next := append([]*http.Cookie{}, cookies...)
	next = append(next, cookiesFrom(w)...)
	return sc[1], seq[1], next
}

func TestPostNewTopicAndReply(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// --- Start a new topic. ---
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)

	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {"My first Go topic"},
		"message":            {"Hello [b]world[/b] from the Go port."},
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
		t.Fatalf("post2 new topic: status %d body %.500s", w.Code, w.Body.String())
	}
	// Without goback the redirect returns to the board.
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "board=1.0") {
		t.Fatalf("post2 redirect %q, want board=1.0", loc)
	}
	var topicID int
	a.DB.QueryRow(a.Q(`SELECT MAX(ID_TOPIC) FROM {$db_prefix}topics`)).Scan(&topicID)
	if topicID < 2 {
		t.Fatalf("no new topic created")
	}
	topic := itoa(topicID)

	// The topic shows on the message index.
	_, body := get(t, a, "/index.php?board=1.0", admin)
	if !strings.Contains(body, "My first Go topic") {
		t.Error("new topic missing from message index")
	}

	// The message renders with BBC parsed.
	_, body = get(t, a, "/index.php?topic="+topic+".0", admin)
	if !strings.Contains(body, "Hello <b>world</b> from the Go port.") {
		t.Errorf("topic display missing parsed body")
	}

	// Stats updated (the install seeds one topic/message).
	if a.Setting("totalTopics") != "2" || a.Setting("totalMessages") != "2" {
		t.Errorf("totalTopics=%s totalMessages=%s, want 2/2",
			a.Setting("totalTopics"), a.Setting("totalMessages"))
	}

	// --- Reply to it. ---
	clearFloodControl(a)
	sc, seq, cookies = openPostForm(t, a, "/index.php?action=post;topic="+topic+".0", admin)

	w = postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {topic},
		"subject":            {"Re: My first Go topic"},
		"message":            {"A reply with a :) smiley."},
		"icon":               {"xx"},
		"notify":             {"0"},
		"num_replies":        {"0"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 reply: status %d body %.500s", w.Code, w.Body.String())
	}

	_, body = get(t, a, "/index.php?topic="+topic+".0", admin)
	if !strings.Contains(body, "A reply with a") || !strings.Contains(body, "smiley.gif") {
		t.Error("reply missing from topic display")
	}
	var numReplies int
	a.DB.QueryRow(a.Q(`SELECT numReplies FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topic).Scan(&numReplies)
	if numReplies != 1 {
		t.Errorf("numReplies = %d, want 1", numReplies)
	}

	// --- Edit the reply. ---
	clearFloodControl(a)
	var msgID int
	a.DB.QueryRow(a.Q(`SELECT ID_LAST_MSG FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topic).Scan(&msgID)
	// Backdate the post past edit_wait_time so the Last Edit marker is set.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET posterTime = posterTime - 120 WHERE ID_MSG = ?`), msgID)

	// The edit form needs ?sesc= on the GET.
	w2, body := get(t, a, "/index.php?action=post;topic="+topic+".0", admin)
	sc = scRe.FindStringSubmatch(body)[1]
	cookies = append([]*http.Cookie{admin}, cookiesFrom(w2)...)
	editPath := "/index.php?action=post;msg=" + itoa(msgID) + ";topic=" + topic + ".0;sesc=" + sc
	sc, seq, cookies = openPostForm(t, a, editPath, cookies...)

	w = postForm(t, a, "/index.php?action=post2;start=0;msg="+itoa(msgID)+";board=1.0", url.Values{
		"topic":              {topic},
		"subject":            {"Re: My first Go topic"},
		"message":            {"An edited reply."},
		"icon":               {"xx"},
		"notify":             {"0"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 edit: status %d body %.500s", w.Code, w.Body.String())
	}
	_, body = get(t, a, "/index.php?topic="+topic+".0", admin)
	if !strings.Contains(body, "An edited reply.") {
		t.Error("edited body missing from topic display")
	}
	if !strings.Contains(body, "Last Edit:") {
		t.Error("modified marker missing from topic display")
	}
}

func TestPostPreviewAndErrors(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)

	// Preview shows the parsed message and keeps the form.
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {"Preview subject"},
		"message":            {"Previewing [i]italics[/i]."},
		"icon":               {"xx"},
		"preview":            {"Preview"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("preview: status %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Previewing <i>italics</i>.") {
		t.Error("preview body not rendered")
	}
	if !strings.Contains(body, `value="Preview subject"`) {
		t.Error("subject not preserved in form")
	}
	// Nothing was actually posted (only the seeded welcome message exists).
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages`)).Scan(&n)
	if n != 1 {
		t.Errorf("messages count after preview = %d, want 1", n)
	}

	// Submitting with no subject is an error, re-displaying the form.
	clearFloodControl(a)
	sc2, seq2, cookies2 := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w = postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {""},
		"message":            {"Body without subject."},
		"icon":               {"xx"},
		"additional_options": {"0"},
		"sc":                 {sc2},
		"seqnum":             {seq2},
	}, cookies2...)
	if w.Code != 200 {
		t.Fatalf("no-subject post: status %d", w.Code)
	}
	body = w.Body.String()
	if !strings.Contains(body, "No subject was filled in") {
		t.Errorf("expected no_subject error, got: %.300s", body)
	}
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages`)).Scan(&n)
	if n != 1 {
		t.Errorf("messages count after error = %d, want 1", n)
	}
}

func TestGuestCannotPost(t *testing.T) {
	a := newTestApp(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080/index.php", nil)
	u, _ := url.Parse("http://127.0.0.1:8080/index.php")
	u.RawQuery = "action=post;board=1.0"
	r.URL = u
	a.Handler().ServeHTTP(w, r)
	// Guests can't start topics on the default board: login redirect/kick.
	body := w.Body.String()
	if strings.Contains(body, `id="postmodify"`) {
		t.Error("guest got the post form")
	}
}

func TestQuoteFastAndJSModify(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Grab a session (sc) from any page with the form.
	sc, _, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)

	// QuoteFast the seeded welcome post (msg 1).
	w, body := get(t, a, "/index.php?action=quotefast;quote=1;sesc="+sc+";xml", cookies...)
	if w.Code != 200 {
		t.Fatalf("quotefast: status %d", w.Code)
	}
	if !strings.Contains(body, "<quote>[quote author=") || !strings.Contains(body, "Welcome to Simple Machines Forum!") {
		t.Errorf("quotefast body wrong: %.300s", body)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/xml") {
		t.Errorf("quotefast content-type = %q", ct)
	}

	// Quick-modify fetch (modifyfast).
	_, body = get(t, a, "/index.php?action=quotefast;quote=1;modify;sesc="+sc+";xml", cookies...)
	if !strings.Contains(body, `<message id="msg_1">`) || !strings.Contains(body, "<subject><![CDATA[Welcome to SMF!]]></subject>") {
		t.Errorf("modifyfast body wrong: %.300s", body)
	}

	// jsmodify: change the subject in place.
	w2 := postForm(t, a, "/index.php?action=jsmodify;topic=1;sesc="+sc+";xml", url.Values{
		"subject": {"Welcome retitled"},
		"message": {"Quick-modified body."},
	}, cookies...)
	if w2.Code != 200 {
		t.Fatalf("jsmodify: status %d", w2.Code)
	}
	xbody := w2.Body.String()
	if !strings.Contains(xbody, "<![CDATA[Welcome retitled]]>") || !strings.Contains(xbody, "Quick-modified body.") {
		t.Errorf("jsmodify response wrong: %.400s", xbody)
	}
	var subj string
	a.DB.QueryRow(a.Q(`SELECT subject FROM {$db_prefix}messages WHERE ID_MSG = 1`)).Scan(&subj)
	if subj != "Welcome retitled" {
		t.Errorf("subject in DB = %q", subj)
	}

	// jsmodify with empty message reports an error and changes nothing.
	w3 := postForm(t, a, "/index.php?action=jsmodify;topic=1;sesc="+sc+";xml", url.Values{
		"subject": {"Welcome retitled"},
		"message": {""},
	}, cookies...)
	if !strings.Contains(w3.Body.String(), `in_body="1"`) {
		t.Errorf("jsmodify empty message: %.300s", w3.Body.String())
	}
}

func TestPollLifecycle(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Create a topic with a poll.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0;poll", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;poll;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {"Poll topic"},
		"message":            {"Vote here."},
		"icon":               {"xx"},
		"question":           {"Favourite colour?"},
		"options[0]":         {"Red"},
		"options[1]":         {"Blue"},
		"poll_max_votes":     {"1"},
		"poll_hide":          {"0"},
		"poll_expire":        {""},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 poll: status %d body %.500s", w.Code, w.Body.String())
	}
	var topicID, pollID int
	a.DB.QueryRow(a.Q(`SELECT MAX(ID_TOPIC) FROM {$db_prefix}topics`)).Scan(&topicID)
	a.DB.QueryRow(a.Q(`SELECT ID_POLL FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topicID).Scan(&pollID)
	if pollID == 0 {
		t.Fatal("no poll created")
	}
	topic := itoa(topicID)

	// The poll shows on the topic page.
	_, body := get(t, a, "/index.php?topic="+topic+".0", admin)
	if !strings.Contains(body, "Favourite colour?") || !strings.Contains(body, "Red") {
		t.Error("poll not displayed on topic")
	}

	// Vote.
	w = postForm(t, a, "/index.php?action=vote;topic="+topic+".0;sesc="+sc, url.Values{
		"options[]": {"0"},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("vote: status %d body %.300s", w.Code, w.Body.String())
	}
	var votes int
	a.DB.QueryRow(a.Q(`SELECT votes FROM {$db_prefix}poll_choices WHERE ID_POLL = ? AND ID_CHOICE = 0`), pollID).Scan(&votes)
	if votes != 1 {
		t.Errorf("votes = %d, want 1", votes)
	}

	// Edit poll form renders.
	_, body = get(t, a, "/index.php?action=editpoll;topic="+topic+".0", admin)
	if !strings.Contains(body, `value="Favourite colour?"`) {
		t.Errorf("editpoll form missing question: %.300s", body)
	}
	seqm := seqRe.FindStringSubmatch(body)
	if seqm == nil {
		t.Fatal("editpoll: no seqnum")
	}

	// Change the question.
	w = postForm(t, a, "/index.php?action=editpoll2;topic="+topic+".0", url.Values{
		"question":       {"Best colour?"},
		"options[0]":     {"Red"},
		"options[1]":     {"Blue"},
		"poll_max_votes": {"2"},
		"poll_hide":      {"0"},
		"poll_expire":    {""},
		"sc":             {sc},
		"seqnum":         {seqm[1]},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("editpoll2: status %d body %.300s", w.Code, w.Body.String())
	}
	var q string
	a.DB.QueryRow(a.Q(`SELECT question FROM {$db_prefix}polls WHERE ID_POLL = ?`), pollID).Scan(&q)
	if q != "Best colour?" {
		t.Errorf("question = %q", q)
	}

	// Lock voting.
	w, _ = get(t, a, "/index.php?action=lockVoting;topic="+topic+".0;sesc="+sc, cookies...)
	if w.Code != 302 {
		t.Fatalf("lockVoting: status %d", w.Code)
	}
	var lockedV int
	a.DB.QueryRow(a.Q(`SELECT votingLocked FROM {$db_prefix}polls WHERE ID_POLL = ?`), pollID).Scan(&lockedV)
	if lockedV != 2 { // admin/moderator lock
		t.Errorf("votingLocked = %d, want 2", lockedV)
	}

	// Remove the poll.
	w, _ = get(t, a, "/index.php?action=removepoll;topic="+topic+".0;sesc="+sc, cookies...)
	if w.Code != 302 {
		t.Fatalf("removepoll: status %d", w.Code)
	}
	var left int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}polls WHERE ID_POLL = ?`), pollID).Scan(&left)
	var topicPoll int
	a.DB.QueryRow(a.Q(`SELECT ID_POLL FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topicID).Scan(&topicPoll)
	if left != 0 || topicPoll != 0 {
		t.Errorf("poll remnants: polls=%d topic.ID_POLL=%d", left, topicPoll)
	}
}

func TestDeleteAndRemove(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)

	// Make a topic with a reply.
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic": {"0"}, "subject": {"Doomed topic"}, "message": {"First post."},
		"icon": {"xx"}, "additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
	}, cookies...)
	var topicID int
	a.DB.QueryRow(a.Q(`SELECT MAX(ID_TOPIC) FROM {$db_prefix}topics`)).Scan(&topicID)
	topic := itoa(topicID)

	clearFloodControl(a)
	sc2, seq2, cookies2 := openPostForm(t, a, "/index.php?action=post;topic="+topic+".0", admin)
	postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic": {topic}, "subject": {"Re: Doomed topic"}, "message": {"Doomed reply."},
		"icon": {"xx"}, "num_replies": {"0"}, "additional_options": {"0"}, "sc": {sc2}, "seqnum": {seq2},
	}, cookies2...)
	var lastMsg int
	a.DB.QueryRow(a.Q(`SELECT ID_LAST_MSG FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topicID).Scan(&lastMsg)

	// Delete the reply.
	w, _ := get(t, a, "/index.php?action=deletemsg;msg="+itoa(lastMsg)+";topic="+topic+".0;sesc="+sc2, cookies2...)
	if w.Code != 302 {
		t.Fatalf("deletemsg: status %d", w.Code)
	}
	var n, numReplies int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_MSG = ?`), lastMsg).Scan(&n)
	a.DB.QueryRow(a.Q(`SELECT numReplies FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topicID).Scan(&numReplies)
	if n != 0 || numReplies != 0 {
		t.Errorf("after delete: msg rows=%d numReplies=%d", n, numReplies)
	}

	// Remove the whole topic.
	w, _ = get(t, a, "/index.php?action=removetopic2;topic="+topic+".0;sesc="+sc2, cookies2...)
	if w.Code != 302 {
		t.Fatalf("removetopic2: status %d", w.Code)
	}
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topicID).Scan(&n)
	if n != 0 {
		t.Errorf("topic still present")
	}
	if a.Setting("totalTopics") != "1" || a.Setting("totalMessages") != "1" {
		t.Errorf("totals after removal: topics=%s messages=%s", a.Setting("totalTopics"), a.Setting("totalMessages"))
	}

	// Lock + sticky toggles on the seeded topic.
	w, _ = get(t, a, "/index.php?action=lock;topic=1.0;sesc="+sc2, cookies2...)
	if w.Code != 302 {
		t.Fatalf("lock: status %d", w.Code)
	}
	var locked int
	a.DB.QueryRow(a.Q(`SELECT locked FROM {$db_prefix}topics WHERE ID_TOPIC = 1`)).Scan(&locked)
	if locked != 1 {
		t.Errorf("locked = %d, want 1", locked)
	}
	w, _ = get(t, a, "/index.php?action=sticky;topic=1.0;sesc="+sc2, cookies2...)
	if w.Code != 302 {
		t.Fatalf("sticky: status %d", w.Code)
	}
	var isSticky int
	a.DB.QueryRow(a.Q(`SELECT isSticky FROM {$db_prefix}topics WHERE ID_TOPIC = 1`)).Scan(&isSticky)
	if isSticky != 1 {
		t.Errorf("isSticky = %d, want 1", isSticky)
	}
}
