//go:build ignore

// Parity crawler: fetch the golden-crawl URL set from the live PHP SMF
// instance, apply the same normalizers as go/internal/app/golden_test.go, and
// diff against the committed Go goldens. Run: go run crawl.go
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
	"sort"
	"strings"
)

const base = "http://127.0.0.1:8080"
const goldenDir = "/home/alex/dev/smf-port/go/internal/app/testdata/golden"
const outDir = "/tmp/smf-php/out"

var (
	reHexToken   = regexp.MustCompile(`[0-9a-f]{32}`)
	reLoadTime   = regexp.MustCompile(`(?s)<span class="smalltext">[^<]*?seconds[^<]*?</span>`)
	reTodayStamp = regexp.MustCompile(`(?i)(<b>)?(Today|Yesterday)(</b>)? at [\d: ]*(AM|PM)?`)
	reDateTime   = regexp.MustCompile(`(January|February|March|April|May|June|July|August|September|October|November|December) \d{1,2}, \d{4}, \d{1,2}:\d{2}:\d{2} (am|pm)`)
	reSeqnum     = regexp.MustCompile(`name="seqnum" value="\d+"`)
	reDaysAgo    = regexp.MustCompile(`\d+ days ago`)
	reISOTime    = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([Zz]|[+-]\d{2}:?\d{2})`)
	reRFC822     = regexp.MustCompile(`[A-Z][a-z]{2}, \d{2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2}:\d{2}( [A-Z]{2,4}| [+-]\d{4})?`)
)

func normalize(in string) string {
	s := strings.ReplaceAll(in, "172.18.0.1", "192.0.2.1") // docker client IP -> Go httptest RemoteAddr
	s = strings.ReplaceAll(s, "/var/www/html", "{ASSETDIR}")
	s = reHexToken.ReplaceAllString(s, "{TOKEN}")
	s = reLoadTime.ReplaceAllString(s, `<span class="smalltext">{LOADTIME}</span>`)
	s = reTodayStamp.ReplaceAllString(s, "{TODAY}")
	s = reDateTime.ReplaceAllString(s, "{DATETIME}")
	s = reSeqnum.ReplaceAllString(s, `name="seqnum" value="{SEQNUM}"`)
	s = reDaysAgo.ReplaceAllString(s, "{DAYSAGO}")
	s = reISOTime.ReplaceAllString(s, "{ISOTIME}")
	s = reRFC822.ReplaceAllString(s, "{RFC822}")
	return s
}

type tc struct{ name, path, role string }

var cases = []tc{
	{"boardindex_guest", "/index.php", "guest"},
	{"boardindex_member", "/index.php", "member"},
	{"boardindex_mod", "/index.php", "mod"},
	{"messageindex_guest", "/index.php?board=1.0", "guest"},
	{"messageindex_mod", "/index.php?board=1.0", "mod"},
	{"messageindex_page2", "/index.php?board=1.10", "guest"},
	{"messageindex_child", "/index.php?board=2.0", "guest"},
	{"display_guest", "/index.php?topic=1.0", "guest"},
	{"display_member", "/index.php?topic=1.0", "member"},
	{"display_bbc", "/index.php?topic=200.0", "guest"},
	{"display_locked_sticky", "/index.php?topic=201.0", "member"},
	{"display_long_page2", "/index.php?topic=202.10", "guest"},
	{"display_poll", "/index.php?topic=203.0", "member"},
	{"print_topic", "/index.php?action=printpage;topic=200.0", "guest"},
	{"help_guest", "/index.php?action=help", "guest"},
	{"help_post", "/index.php?action=help;page=post", "guest"},
	{"help_profile", "/index.php?action=help;page=profile", "guest"},
	{"help_pm", "/index.php?action=help;page=pm", "guest"},
	{"help_search", "/index.php?action=help;page=searching", "guest"},
	{"profile_summary", "/index.php?action=profile;u=1", "member"},
	{"profile_member", "/index.php?action=profile;u=2", "member"},
	{"profile_statpanel", "/index.php?action=profile;u=1;sa=statPanel", "member"},
	{"profile_showposts", "/index.php?action=profile;u=1;sa=showPosts", "member"},
	{"memberlist_member", "/index.php?action=mlist", "member"},
	{"memberlist_page2", "/index.php?action=mlist;start=10", "member"},
	{"memberlist_sort_posts", "/index.php?action=mlist;sort=posts;desc", "member"},
	{"recent_guest", "/index.php?action=recent", "guest"},
	{"stats_guest", "/index.php?action=stats", "guest"},
	{"search_member", "/index.php?action=search", "member"},
	{"feed_rss", "/index.php?action=.xml;type=rss", "guest"},
	{"feed_atom", "/index.php?action=.xml;type=atom", "guest"},
	{"pm_inbox_member", "/index.php?action=pm", "member"},
	{"post_newtopic_member", "/index.php?action=post;board=1.0", "member"},
	{"post_reply_member", "/index.php?action=post;topic=1.0", "member"},
	{"login_guest", "/index.php?action=login", "guest"},
	{"register_guest", "/index.php?action=register", "guest"},
	{"reminder_guest", "/index.php?action=reminder", "guest"},
	{"err_board_missing", "/index.php?board=99999.0", "member"},
	{"err_topic_missing", "/index.php?topic=99999.0", "member"},
	{"modlog_admin", "/index.php?action=modlog", "admin"},
	{"helpadmin_popup", "/index.php?action=helpadmin;help=ban_members", "admin"},
	{"admin_home", "/index.php?action=admin", "admin"},
	{"admin_postsettings", "/index.php?action=postsettings", "admin"},
	{"admin_featuresettings", "/index.php?action=featuresettings", "admin"},
	{"admin_manageboards", "/index.php?action=manageboards", "admin"},
	{"admin_managemembers", "/index.php?action=viewmembers", "admin"},
	{"admin_members_search", "/index.php?action=viewmembers;sa=search", "admin"},
	{"admin_membergroups", "/index.php?action=membergroups", "admin"},
	{"admin_permissions", "/index.php?action=permissions", "admin"},
	{"admin_ban_add", "/index.php?action=ban;sa=add", "admin"},
	{"admin_ban_list", "/index.php?action=ban;sa=list", "admin"},
	{"admin_reports", "/index.php?action=reports", "admin"},
	{"admin_report_boards", "/index.php?action=reports;rt=boards", "admin"},
	{"admin_report_staff", "/index.php?action=reports;rt=staff", "admin"},
	{"admin_smileys_editsets", "/index.php?action=smileys;sa=editsets", "admin"},
	{"admin_smileys_settings", "/index.php?action=smileys;sa=settings", "admin"},
	{"admin_attachments", "/index.php?action=manageattachments", "admin"},
	{"admin_managesearch", "/index.php?action=managesearch", "admin"},
	{"admin_managecalendar", "/index.php?action=managecalendar", "admin"},
	{"admin_news_editnews", "/index.php?action=news;sa=editnews", "admin"},
	{"admin_news_settings", "/index.php?action=news;sa=settings", "admin"},
	{"admin_serversettings", "/index.php?action=serversettings;sesc={sesc}", "admin"},
	{"admin_errorlog", "/index.php?action=viewErrorLog", "admin"},
	{"admin_maintain", "/index.php?action=maintain", "admin"},
}

func readCookie(role string) string {
	id := map[string]string{"member": "2", "mod": "3", "admin": "1"}[role]
	if id == "" {
		return ""
	}
	b, err := os.ReadFile("/tmp/smf-php/cookie_" + id + ".txt")
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(b))
}

var reSc = regexp.MustCompile(`name="sc" value="([0-9a-f]+)"`)

// adminSession establishes a PHP session for the admin (auth via the login
// cookie) and scrapes the session token (sc), mirroring the Go test's mbForm.
// Some admin GETs (serversettings) require ;sesc=<token> to pass checkSession.
func adminSession() (*http.Client, string) {
	jar, _ := cookiejar.New(nil)
	cl := &http.Client{Jar: jar}
	u, _ := url.Parse(base)
	jar.SetCookies(u, []*http.Cookie{{Name: "SMFCookie11", Value: readCookie("admin")}})
	resp, err := cl.Get(base + "/index.php?action=maintain")
	if err != nil {
		return cl, ""
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	m := reSc.FindStringSubmatch(string(body))
	if m == nil {
		return cl, ""
	}
	return cl, m[1]
}

func main() {
	os.MkdirAll(outDir, 0755)
	// Self-reset volatile state (numViews / most-online / logs) to the
	// goldenApp baseline so each run is idempotent.
	if out, err := exec.Command("bash", "-c", "sudo docker exec -i smf-mysql sh -c 'mysql -uroot -proot smf' < /tmp/smf-php/snapshot.sql && sudo docker exec -i smf-mysql mysql -uroot -proot smf < /tmp/smf-php/reset.sql").CombinedOutput(); err != nil {
		fmt.Println("reset failed:", err, string(out))
		os.Exit(1)
	}
	adminCl, adminSesc := adminSession()
	client := &http.Client{}
	var pass, fail, missing []string
	for _, c := range cases {
		http.Get(base + "/clearonline.php") // mirror Go: clear log_online per request
		// Pages carrying ;sesc=<token> need the matching admin session.
		if strings.Contains(c.path, "{sesc}") {
			req, _ := http.NewRequest("GET", base+strings.ReplaceAll(c.path, "{sesc}", adminSesc), nil)
			resp, err := adminCl.Do(req)
			if err != nil {
				fail = append(fail, c.name+" (req err)")
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			got := normalize(string(body))
			os.WriteFile(filepath.Join(outDir, c.name+".php.html"), []byte(got), 0644)
			if wantB, err := os.ReadFile(filepath.Join(goldenDir, c.name+".html")); err != nil {
				missing = append(missing, c.name)
			} else if string(wantB) == got {
				pass = append(pass, c.name)
			} else {
				fail = append(fail, c.name)
			}
			continue
		}
		req, _ := http.NewRequest("GET", base+c.path, nil)
		// A non-empty Cookie header (any cookie) stops SMF's cookie-less
		// ?PHPSESSID URL rewriting (QueryString.php:445), matching the Go port
		// which dropped that mechanism (always cookie-based).
		cookie := "smfparity=1"
		if ck := readCookie(c.role); ck != "" {
			cookie = "SMFCookie11=" + ck + "; smfparity=1"
		}
		req.Header.Set("Cookie", cookie)
		resp, err := client.Do(req)
		if err != nil {
			fail = append(fail, c.name+" (req err)")
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		got := normalize(string(body))
		os.WriteFile(filepath.Join(outDir, c.name+".php.html"), []byte(got), 0644)

		wantB, err := os.ReadFile(filepath.Join(goldenDir, c.name+".html"))
		if err != nil {
			missing = append(missing, c.name)
			continue
		}
		want := string(wantB)
		if want == got {
			pass = append(pass, c.name)
		} else {
			fail = append(fail, c.name)
		}
	}
	sort.Strings(fail)
	fmt.Printf("\n=== PARITY: %d pass, %d fail, %d missing-golden (of %d) ===\n", len(pass), len(fail), len(missing), len(cases))
	if len(fail) > 0 {
		fmt.Println("\nFAIL:")
		for _, f := range fail {
			fmt.Println("  " + f)
		}
	}
	if len(missing) > 0 {
		fmt.Println("\nMISSING GOLDEN:", strings.Join(missing, ", "))
	}
	fmt.Println("\nPASS:", strings.Join(pass, ", "))
}