package app

// Phase 8 — golden-crawl harness.
//
// A deterministic in-process crawl: seed a fixed scenario, request a curated
// set of pages as each role, normalize the volatile bytes (session tokens,
// page-generation time, "Today at ..." stamps), and diff against committed
// golden files under testdata/golden/. This is verification strategy #4 from
// the plan — committed golden HTML vs an httptest server on seeded SQLite, no
// PHP needed after capture. It locks current output so future changes surface
// as reviewable diffs.
//
// Regenerate the goldens after an intentional output change with:
//   GOLDEN_UPDATE=1 go test ./internal/app/ -run TestGoldenCrawl

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"smf/internal/phpser"
)

// A fixed instant well in the past so timeformat never hits "Today/Yesterday".
const goldenEpoch = 1262304000 // 2010-01-01 00:00:00 UTC

func goldenApp(t *testing.T) *App {
	t.Helper()
	// Deterministic time zone for strftime output.
	time.Local = time.UTC

	a := newTestApp(t)
	// Pin the "now"-seeded timestamps to the fixed epoch so committed goldens
	// don't drift with the wall clock: admin's registration/last-login (the
	// memberlist shows a date-only registration column the datetime normalizer
	// doesn't cover) and the welcome message's posterTime (otherwise it renders
	// "Today at ..." only on the day it was generated).
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET dateRegistered = ?, lastLogin = ? WHERE ID_MEMBER = 1`), goldenEpoch, goldenEpoch)
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET posterTime = ? WHERE ID_MSG = 1`), goldenEpoch)
	a.UpdateSettings(map[string]string{
		"securityDisable": "1",
		// Freeze the "most online" info-center stats.
		"mostOnline":       "5",
		"mostOnlineToday":  "5",
		"mostDate":         itoa(goldenEpoch),
		"settings_updated": itoa(goldenEpoch),
		// Pin pagination so it doesn't ride on default changes.
		"defaultMaxTopics":   "10",
		"defaultMaxMessages": "10",
		"defaultMaxMembers":  "10",
	})

	ts := goldenEpoch
	next := func() int { ts += 3600; return ts }

	// A regular member ("member") and a board moderator ("mod"), plus a pile of
	// ordinary members so the memberlist paginates and sorts.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
		(memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
		VALUES ('member', 'Member One', ?, 'm@b.com', 0, 1, ?, ?, 1, 4)`),
		fmt.Sprintf("%x", sha1.Sum([]byte("member"+"memberpass"))), goldenEpoch, goldenEpoch)
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
		(memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
		VALUES ('mod', 'Mod One', ?, 'mod@b.com', 0, 2, ?, ?, 1, 4)`),
		fmt.Sprintf("%x", sha1.Sum([]byte("mod"+"modpass"))), goldenEpoch, goldenEpoch)
	var modID int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'mod'`)).Scan(&modID)
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}moderators (ID_BOARD, ID_MEMBER) VALUES (1, ?)`), modID)
	for i := 1; i <= 14; i++ {
		nm := fmt.Sprintf("user%02d", i)
		a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
			(memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
			VALUES (?, ?, 'x', ?, 0, ?, ?, ?, 1, 4)`),
			nm, "User "+itoa(i), nm+"@b.com", i*3, goldenEpoch, goldenEpoch)
	}

	// A child board under board 1, and a second top-level board in the cat.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}boards (ID_BOARD, ID_CAT, ID_PARENT, childLevel, name, description, memberGroups, boardOrder) VALUES (2, 1, 1, 1, 'Child Board', 'A nested board.', '-1,0,2', 1)`))
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}boards (ID_BOARD, ID_CAT, name, description, memberGroups, boardOrder) VALUES (3, 1, 'Second Board', 'Another board.', '-1,0,2', 2)`))

	// (Topic 1 "Welcome to SMF!" is already seeded by the schema; leave it.)

	// addTopic inserts the first message, then the topic pointing at it (so the
	// UNIQUE (ID_FIRST_MSG, ID_BOARD) index is satisfied). Returns (tid, mid).
	addTopic := func(tid, board, memberID, views, locked, sticky, idPoll int, poster, email, subject, body string) (int, int) {
		res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'xx')`),
			tid, board, memberID, poster, email, next(), subject, body)
		mid, _ := res.LastInsertId()
		a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
			tid, board, mid, mid, memberID, memberID, idPoll, views, locked, sticky)
		return tid, int(mid)
	}

	// 12 plain topics on board 1 so the message index paginates.
	for i := 1; i <= 12; i++ {
		addTopic(100+i, 1, 1, i, 0, 0, 0, "admin", "a@b.com", fmt.Sprintf("Topic number %02d", i), fmt.Sprintf("Body of topic %02d.", i))
	}

	// A BBC-heavy topic (ID 200): quote/code/list/table/url/color.
	bbcBody := "[quote author=admin]nested [b]quote[/b][/quote]\n[code]if (x) { return; }[/code]\n[list][li]one[/li][li]two[/li][/list]\n[url=http://example.com]link[/url] [color=red]red[/color]\n[table][tr][td]a[/td][td]b[/td][/tr][/table]"
	addTopic(200, 1, 1, 7, 0, 0, 0, "admin", "a@b.com", "BBC showcase", bbcBody)

	// A locked + sticky topic (ID 201).
	addTopic(201, 1, 1, 2, 1, 1, 0, "admin", "a@b.com", "Announcement", "Pinned and locked.")

	// A long topic (ID 202) with 12 replies so display paginates.
	_, lastMid := addTopic(202, 1, 1, 9, 0, 0, 0, "admin", "a@b.com", "Long thread", "First post.")
	for i := 1; i <= 12; i++ {
		res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202, 1, 1, 'member', 'm@b.com', ?, 'Re: Long thread', ?, 'xx')`),
			next(), fmt.Sprintf("Reply %02d.", i))
		m, _ := res.LastInsertId()
		lastMid = int(m)
	}
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_LAST_MSG = ?, ID_MEMBER_UPDATED = 0, numReplies = 12 WHERE ID_TOPIC = 202`), lastMid)

	// A poll topic (ID 203).
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}polls (ID_POLL, question, votingLocked, maxVotes, expireTime, hideResults, changeVote, ID_MEMBER, posterName) VALUES (1, 'Favorite color?', 0, 1, 0, 0, 0, 1, 'admin')`))
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}poll_choices (ID_POLL, ID_CHOICE, label, votes) VALUES (1, 0, 'Red', 2), (1, 1, 'Green', 5), (1, 2, 'Blue', 3)`))
	addTopic(203, 1, 1, 4, 0, 0, 1, "admin", "a@b.com", "Vote here", "Cast your vote.")

	// A PM from admin to the member.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}personal_messages (ID_PM, ID_MEMBER_FROM, deletedBySender, fromName, msgtime, subject, body) VALUES (1, 1, 0, 'admin', ?, 'Hello there', 'A private message body.')`), next())
	var memberID int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'member'`)).Scan(&memberID)
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}pm_recipients (ID_PM, ID_MEMBER, labels, bcc, is_read, deleted) VALUES (1, ?, '-1', 0, 0, 0)`), memberID)

	// A moderation-log entry.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_actions (logTime, ID_MEMBER, ip, action, extra) VALUES (?, 1, '127.0.0.1', 'remove', ?)`),
		goldenEpoch, `{"topic":101,"subject":"Topic number 01","member":"admin","board":1,"board_name":"General Discussion"}`)

	// Make topic first/last/reply counts and board totals consistent. (Done in
	// Go with grouped SELECT + per-row UPDATE; SQLite correlated UPDATE
	// subqueries don't update reliably here.)
	type tFix struct{ tid, first, last, replies int }
	var tfix []tFix
	if rows, err := a.DB.Query(a.Q(`SELECT ID_TOPIC, MIN(ID_MSG), MAX(ID_MSG), COUNT(*) - 1 FROM {$db_prefix}messages GROUP BY ID_TOPIC`)); err == nil {
		for rows.Next() {
			var x tFix
			rows.Scan(&x.tid, &x.first, &x.last, &x.replies)
			tfix = append(tfix, x)
		}
		rows.Close()
	}
	for _, x := range tfix {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_FIRST_MSG = ?, ID_LAST_MSG = ?, numReplies = ? WHERE ID_TOPIC = ?`), x.first, x.last, x.replies, x.tid)
	}
	type bFix struct{ bid, posts, last int }
	var bfix []bFix
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, COUNT(*), MAX(ID_MSG) FROM {$db_prefix}messages GROUP BY ID_BOARD`)); err == nil {
		for rows.Next() {
			var x bFix
			rows.Scan(&x.bid, &x.posts, &x.last)
			bfix = append(bfix, x)
		}
		rows.Close()
	}
	topicCounts := map[int]int{}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, COUNT(*) FROM {$db_prefix}topics GROUP BY ID_BOARD`)); err == nil {
		for rows.Next() {
			var b, n int
			rows.Scan(&b, &n)
			topicCounts[b] = n
		}
		rows.Close()
	}
	for _, x := range bfix {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numTopics = ?, numPosts = ?, ID_LAST_MSG = ?, ID_MSG_UPDATED = ? WHERE ID_BOARD = ?`),
			topicCounts[x.bid], x.posts, x.last, x.last, x.bid)
	}

	return a
}

// goldenLoginCookie builds a login cookie for any seeded member by name.
func goldenLoginCookie(t *testing.T, a *App, memberName string) *http.Cookie {
	t.Helper()
	var idMember int
	var passwd, salt string
	if err := a.DB.QueryRow(a.Q(`SELECT ID_MEMBER, passwd, passwordSalt FROM {$db_prefix}members WHERE memberName = ?`), memberName).Scan(&idMember, &passwd, &salt); err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(passwd+salt)))
	val, err := phpser.Serialize([]any{idMember, hash, int(time.Now().Unix()) + 3600, 2})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: a.Config.CookieName, Value: url.QueryEscape(val)}
}

// goldenCase is one crawl entry.
type goldenCase struct {
	name string
	path string
	role string // "guest" | "member" | "admin"
}

func TestGoldenCrawl(t *testing.T) {
	a := goldenApp(t)
	// Admin requests reuse one session (login + session cookie) so its sesc
	// token is stable; some admin settings pages require ;sesc=<session> in the
	// URL (SMF's admin links carry it) which we substitute for {sesc}.
	adminSesc, adminCookies := mbForm(t, a, "/index.php?action=maintain", adminCookie(t, a))
	roleCookies := map[string][]*http.Cookie{
		"admin":  adminCookies,
		"member": {goldenLoginCookie(t, a, "member")},
		"mod":    {goldenLoginCookie(t, a, "mod")},
	}
	cookieFor := func(role string) []*http.Cookie { return roleCookies[role] }

	cases := []goldenCase{
		// Read-only forum, across roles.
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

		// Members & discovery.
		{"profile_summary", "/index.php?action=profile;u=1", "member"},
		{"profile_member", "/index.php?action=profile;u=2", "member"},
		{"profile_statpanel", "/index.php?action=profile;u=1;sa=statPanel", "member"},
		{"profile_showposts", "/index.php?action=profile;u=1;sa=showPosts", "member"},
		// (Guests have no view_mlist perm in a default SMF install, so crawl the
		// memberlist as a member to exercise the real listing.)
		{"memberlist_member", "/index.php?action=mlist", "member"},
		{"memberlist_page2", "/index.php?action=mlist;start=10", "member"},
		{"memberlist_sort_posts", "/index.php?action=mlist;sort=posts;desc", "member"},
		{"recent_guest", "/index.php?action=recent", "guest"},
		{"stats_guest", "/index.php?action=stats", "guest"},
		{"search_member", "/index.php?action=search", "member"},

		// Feeds.
		{"feed_rss", "/index.php?action=.xml;type=rss", "guest"},
		{"feed_atom", "/index.php?action=.xml;type=atom", "guest"},

		// Member-only.
		{"pm_inbox_member", "/index.php?action=pm", "member"},
		{"post_newtopic_member", "/index.php?action=post;board=1.0", "member"},
		{"post_reply_member", "/index.php?action=post;topic=1.0", "member"},

		// Guest entry forms.
		{"login_guest", "/index.php?action=login", "guest"},
		{"register_guest", "/index.php?action=register", "guest"},
		{"reminder_guest", "/index.php?action=reminder", "guest"},

		// Error pages (as a member, so we get the fatal-error page rather than
		// the guest login kick).
		{"err_board_missing", "/index.php?board=99999.0", "member"},
		{"err_topic_missing", "/index.php?topic=99999.0", "member"},

		// Moderation.
		{"modlog_admin", "/index.php?action=modlog", "admin"},
		{"helpadmin_popup", "/index.php?action=helpadmin;help=ban_members", "admin"},

		// Admin panel.
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

	update := os.Getenv("GOLDEN_UPDATE") != ""
	dir := filepath.Join("testdata", "golden")
	if update {
		os.MkdirAll(dir, 0755)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each request logs its user online; clear first so the "users
			// online" info-center count reflects only this requester.
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online`))
			reqPath := strings.ReplaceAll(tc.path, "{sesc}", adminSesc)
			w, body := get(t, a, reqPath, cookieFor(tc.role)...)
			if w.Code != 200 {
				t.Fatalf("%s: status %d", tc.path, w.Code)
			}
			got := normalizeGolden(strings.ReplaceAll(body, a.Config.AssetDir, "{ASSETDIR}"))
			path := filepath.Join(dir, tc.name+".html")
			if update {
				if err := os.WriteFile(path, []byte(got), 0644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden %s (run with GOLDEN_UPDATE=1): %v", path, err)
			}
			if string(want) != got {
				t.Errorf("golden mismatch for %s; diff:\n%s", tc.name, firstDiff(string(want), got))
			}
		})
	}
}

var (
	reHexToken   = regexp.MustCompile(`[0-9a-f]{32}`)
	reLoadTime   = regexp.MustCompile(`(?s)<span class="smalltext">[^<]*?seconds[^<]*?</span>`)
	reTodayStamp = regexp.MustCompile(`(?i)(<b>)?(Today|Yesterday)(</b>)? at [\d: ]*(AM|PM)?`)
	// Full strftime datetimes ("June 10, 2026, 08:21:14 PM") — the header's
	// live clock is per-second volatile; the seeded ones are fixed but get
	// normalized too (harmless — coverage is structural, not value-level here).
	reDateTime = regexp.MustCompile(`(January|February|March|April|May|June|July|August|September|October|November|December) \d{1,2}, \d{4}, \d{1,2}:\d{2}:\d{2} (am|pm)`)
	// checkSubmitOnce()'s anti-double-post token (time/random derived).
	reSeqnum = regexp.MustCompile(`name="seqnum" value="\d+"`)
	// jeffsdatediff's relative last-login stamp ("6004 days ago") is measured
	// from the real current date, so it drifts a day at a time between runs.
	reDaysAgo = regexp.MustCompile(`\d+ days ago`)
	// Feed timestamps: ISO-8601 (atom) and RFC-822 (rss). The seeded item dates
	// are fixed but the channel <modified>/<lastBuildDate> are "now".
	reISOTime = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}([Zz]|[+-]\d{2}:?\d{2})`)
	reRFC822  = regexp.MustCompile(`[A-Z][a-z]{2}, \d{2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2}:\d{2}( [A-Z]{2,4}| [+-]\d{4})?`)
)

// normalizeGolden blanks out the bytes that legitimately vary between runs.
func normalizeGolden(s string) string {
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

// firstDiff returns a short window around the first divergence for readability.
func firstDiff(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	start := i - 60
	if start < 0 {
		start = 0
	}
	end := i + 120
	aEnd, bEnd := end, end
	if aEnd > len(a) {
		aEnd = len(a)
	}
	if bEnd > len(b) {
		bEnd = len(b)
	}
	return "...want: " + a[start:aEnd] + "\n...got:  " + b[start:bEnd]
}
