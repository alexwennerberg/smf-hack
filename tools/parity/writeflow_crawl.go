//go:build ignore

// Write-flow parity vs the live PHP SMF reference. For each scenario: restore
// the seed snapshot, establish a session, perform the mutation (POST), render
// the result page, normalize, and diff against the Go writeflow goldens.
// Run: go run writeflow_crawl.go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const base = "http://127.0.0.1:8080"
const goldenDir = "/home/alex/dev/smf-port/go/internal/app/testdata/golden_writeflow"
const outDir = "/tmp/smf-php/out_wf"

var (
	reHexToken   = regexp.MustCompile(`[0-9a-f]{32}`)
	reLoadTime   = regexp.MustCompile(`(?s)<span class="smalltext">[^<]*?seconds[^<]*?</span>`)
	reTodayStamp = regexp.MustCompile(`(?i)(<b>)?(Today|Yesterday)(</b>)? at [\d: ]*(am|pm)?`)
	reDateTime   = regexp.MustCompile(`(January|February|March|April|May|June|July|August|September|October|November|December) \d{1,2}, \d{4}, \d{1,2}:\d{2}:\d{2} (am|pm)`)
	reSeqnum     = regexp.MustCompile(`name="seqnum" value="\d+"`)
	reDaysAgo    = regexp.MustCompile(`\d+ days ago`)
	reSc         = regexp.MustCompile(`name="sc" value="([0-9a-f]+)"`)
	reSeqVal     = regexp.MustCompile(`name="seqnum" value="(\d+)"`)
)

func normalize(in string) string {
	s := strings.ReplaceAll(in, "172.18.0.1", "192.0.2.1")
	s = strings.ReplaceAll(s, "/var/www/html", "{ASSETDIR}")
	s = reHexToken.ReplaceAllString(s, "{TOKEN}")
	s = reLoadTime.ReplaceAllString(s, `<span class="smalltext">{LOADTIME}</span>`)
	s = reTodayStamp.ReplaceAllString(s, "{TODAY}")
	s = reDateTime.ReplaceAllString(s, "{DATETIME}")
	s = reSeqnum.ReplaceAllString(s, `name="seqnum" value="{SEQNUM}"`)
	s = reDaysAgo.ReplaceAllString(s, "{DAYSAGO}")
	return s
}

func restoreSnapshot() {
	cmd := "sudo docker exec -i smf-mysql sh -c 'mysql -uroot -proot smf' < /tmp/smf-php/snapshot.sql && " +
		"sudo docker exec smf-mysql mysql -uroot -proot smf -e \"UPDATE smf_members SET dateRegistered=1262304000,lastLogin=1262304000 WHERE ID_MEMBER=1; UPDATE smf_messages SET posterTime=1262304000 WHERE ID_MSG=1;\""
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	if err != nil {
		fmt.Println("restore failed:", err, string(out))
		os.Exit(1)
	}
}

func clearOnline() { http.Get(base + "/clearonline.php") }

func cookieVal(id string) string {
	b, _ := os.ReadFile("/tmp/smf-php/cookie_" + id + ".txt")
	return strings.TrimSpace(string(b))
}

// session builds an http.Client whose jar carries the login cookie for member
// id, and that does NOT auto-follow redirects (so a post2's 302 doesn't render
// the result page an extra time before we do).
func session(memberID string) *http.Client {
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(base)
	jar.SetCookies(u, []*http.Cookie{{Name: "SMFCookie11", Value: cookieVal(memberID)}})
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func getBody(cl *http.Client, path string) (string, int) {
	resp, err := cl.Get(base + path)
	if err != nil {
		return "", 0
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b), resp.StatusCode
}

func post(cl *http.Client, path string, form url.Values) (string, int) {
	resp, err := cl.Post(base+path, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b), resp.StatusCode
}

func scOf(body string) string {
	if m := reSc.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}
func seqOf(body string) string {
	if m := reSeqVal.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

var pass, fail []string

// check diffs a rendered+normalized page against its Go writeflow golden.
func check(name, body string) {
	os.MkdirAll(outDir, 0755)
	got := normalize(body)
	os.WriteFile(filepath.Join(outDir, name+".php.html"), []byte(got), 0644)
	want, err := os.ReadFile(filepath.Join(goldenDir, name+".html"))
	if err != nil {
		fail = append(fail, name+" (no golden)")
		return
	}
	if string(want) == got {
		pass = append(pass, name)
	} else {
		fail = append(fail, name)
	}
}

func main() {
	// reply_topic1: admin replies to topic 1, then render topic 1 as guest.
	restoreSnapshot()
	{
		cl := session("1")
		form, _ := getBody(cl, "/index.php?action=post;topic=1.0")
		sc, seq := scOf(form), seqOf(form)
		body, code := post(cl, "/index.php?action=post2;topic=1.0;start=0;board=1.0", url.Values{
			"topic": {"1"}, "subject": {"Re: Welcome to SMF!"}, "message": {"A parity reply body."},
			"icon": {"xx"}, "notify": {"0"}, "lock": {"0"}, "sticky": {"0"}, "move": {"0"},
			"additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
		})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("reply_topic1 (post2 code %d: %.200s)", code, body))
		} else {
			guest := &http.Client{}
			clearOnline()
			b, _ := getBody(guest, "/index.php?topic=1.0")
			check("reply_topic1", b)
		}
	}

	// newtopic: admin posts a new topic on board 1; render board 1 + the topic.
	restoreSnapshot()
	{
		cl := session("1")
		form, _ := getBody(cl, "/index.php?action=post;board=1.0")
		sc, seq := scOf(form), seqOf(form)
		_, code := post(cl, "/index.php?action=post2;start=0;board=1.0", url.Values{
			"topic": {"0"}, "subject": {"Parity new topic"}, "message": {"Parity new topic body."},
			"icon": {"xx"}, "notify": {"0"}, "lock": {"0"}, "sticky": {"0"}, "move": {"0"},
			"additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
		})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("newtopic (post2 code %d)", code))
		} else {
			guest := &http.Client{}
			clearOnline()
			b1, _ := getBody(guest, "/index.php?board=1.0")
			check("newtopic_board1", b1)
			clearOnline()
			b2, _ := getBody(&http.Client{}, "/index.php?topic=204.0")
			check("newtopic_display", b2)
		}
	}

	// pollvote: member votes choice 1 in poll topic 203; render as member.
	restoreSnapshot()
	{
		cl := session("2")
		form, _ := getBody(cl, "/index.php?topic=203.0")
		sc := scOf(form)
		_, code := post(cl, "/index.php?action=vote;topic=203", url.Values{"options[]": {"1"}, "sc": {sc}})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("pollvote_topic203 (vote code %d)", code))
		} else {
			clearOnline()
			b, _ := getBody(session("2"), "/index.php?topic=203.0")
			check("pollvote_topic203", b)
		}
	}

	// quickmod lock topic 101; render board 1 as guest.
	restoreSnapshot()
	{
		cl := session("1")
		form, _ := getBody(cl, "/index.php?action=post;board=1.0")
		sc := scOf(form)
		_, code := post(cl, "/index.php?action=quickmod;board=1.0", url.Values{
			"topics[]": {"101"}, "qaction": {"lock"}, "sc": {sc},
		})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("quickmod_lock_board1 (code %d)", code))
		} else {
			clearOnline()
			b, _ := getBody(&http.Client{}, "/index.php?board=1.0")
			check("quickmod_lock_board1", b)
		}
	}

	// PM: admin sends to Member One; render member's inbox as member.
	restoreSnapshot()
	{
		cl := session("1")
		form, _ := getBody(cl, "/index.php?action=pm;sa=send")
		sc := scOf(form)
		_, code := post(cl, "/index.php?action=pm;sa=send2", url.Values{
			"to": {"Member One"}, "subject": {"Parity PM"}, "message": {"Parity PM body."}, "sc": {sc},
		})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("pm_inbox_after_send (send2 code %d)", code))
		} else {
			clearOnline()
			b, _ := getBody(session("2"), "/index.php?action=pm")
			check("pm_inbox_after_send", b)
		}
	}

	// edit topic 101's first message (msg 2): needs a session + sesc.
	restoreSnapshot()
	{
		cl := session("1")
		disp, _ := getBody(cl, "/index.php?topic=101.0")
		sc0 := scOf(disp)
		form, _ := getBody(cl, "/index.php?action=post;msg=2;topic=101.0;sesc="+sc0)
		sc, seq := scOf(form), seqOf(form)
		_, code := post(cl, "/index.php?action=post2;start=0;msg=2;board=1.0", url.Values{
			"topic": {"101"}, "subject": {"Topic number 01"}, "message": {"Edited body."},
			"icon": {"xx"}, "notify": {"0"}, "additional_options": {"0"}, "sc": {sc}, "seqnum": {seq},
		})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("edit_topic101 (post2 code %d)", code))
		} else {
			clearOnline()
			b, _ := getBody(&http.Client{}, "/index.php?topic=101.0")
			check("edit_topic101", b)
		}
	}

	// delete reply msg 17 of topic 202 (deletemsg is a GET with sesc).
	restoreSnapshot()
	{
		cl := session("1")
		disp, _ := getBody(cl, "/index.php?topic=202.0")
		sc0 := scOf(disp)
		_, code := getBody(cl, "/index.php?action=deletemsg;topic=202.0;msg=17;sesc="+sc0)
		if code != 302 {
			fail = append(fail, fmt.Sprintf("delete_reply_topic202 (deletemsg code %d)", code))
		} else {
			clearOnline()
			b, _ := getBody(&http.Client{}, "/index.php?topic=202.0")
			check("delete_reply_topic202", b)
		}
	}

	// move topic 101 from board 1 to board 3.
	restoreSnapshot()
	{
		cl := session("1")
		form, _ := getBody(cl, "/index.php?action=movetopic;topic=101.0")
		sc := scOf(form)
		_, code := post(cl, "/index.php?action=movetopic2;topic=101.0", url.Values{"toboard": {"3"}, "sc": {sc}})
		if code != 302 {
			fail = append(fail, fmt.Sprintf("move (movetopic2 code %d)", code))
		} else {
			clearOnline()
			b1, _ := getBody(&http.Client{}, "/index.php?board=1.0")
			check("move_board1_after", b1)
			clearOnline()
			b3, _ := getBody(&http.Client{}, "/index.php?board=3.0")
			check("move_board3_after", b3)
		}
	}

	// admin save: censor word list (SetCensor persists then re-renders, 200).
	restoreSnapshot()
	{
		cl := session("1")
		form, _ := getBody(cl, "/index.php?action=postsettings;sa=censor")
		sc := scOf(form)
		_, code := post(cl, "/index.php?action=postsettings;sa=censor", url.Values{
			"save_censor": {"1"}, "censortext": {"badword=goodword"}, "sc": {sc},
		})
		if code != 200 {
			fail = append(fail, fmt.Sprintf("censor_after_save (save code %d)", code))
		} else {
			clearOnline()
			b, _ := getBody(session("1"), "/index.php?action=postsettings;sa=censor")
			check("censor_after_save", b)
		}
	}

	fmt.Printf("\n=== WRITE-FLOW PARITY: %d pass, %d fail ===\n", len(pass), len(fail))
	if len(fail) > 0 {
		fmt.Println("FAIL:")
		for _, f := range fail {
			fmt.Println("  " + f)
		}
	}
	fmt.Println("PASS:", strings.Join(pass, ", "))
}