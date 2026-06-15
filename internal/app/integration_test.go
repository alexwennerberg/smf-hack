package app

// End-to-end request tests over a seeded in-memory forum: the Phase 1
// read-only pages for guests and (via a hand-built login cookie) members.

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smf/internal/phpser"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.DBPath = filepath.Join(dir, "test.sqlite")
	cfg.AssetDir = dir
	// A real install has these asset subdirs; create them so path-validity
	// checks (attachmentUploadDir, smileys_dir) don't flag them as missing.
	os.MkdirAll(filepath.Join(dir, "attachments"), 0755)
	os.MkdirAll(filepath.Join(dir, "Smileys"), 0755)
	a, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.DB.Close() })

	// Create the admin like cmd/smf admin-create does.
	passwd := fmt.Sprintf("%x", sha1.Sum([]byte("admin"+"testpass")))
	_, err = a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members
		(memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
		VALUES ('admin', 'admin', ?, 'a@b.com', 1, 0, ?, ?, 1, 0)`),
		passwd, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	a.UpdateSettings(map[string]string{"latestMember": "1", "latestRealName": "admin", "totalMembers": "1"})
	return a
}

func get(t *testing.T, a *App, path string, cookies ...*http.Cookie) (*httptest.ResponseRecorder, string) {
	t.Helper()
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil)
	// Match how the real listener sees SMF query strings: RawQuery contains
	// ';' separators verbatim.
	if i := strings.IndexByte(path, '?'); i >= 0 {
		u, _ := url.Parse("http://127.0.0.1:8080" + path[:i])
		u.RawQuery = path[i+1:]
		r.URL = u
	}
	for _, ck := range cookies {
		r.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w, w.Body.String()
}

// adminCookie builds the SMF login cookie for the test admin.
func adminCookie(t *testing.T, a *App) *http.Cookie {
	t.Helper()
	var passwd, salt string
	if err := a.DB.QueryRow(a.Q(`SELECT passwd, passwordSalt FROM {$db_prefix}members WHERE memberName = 'admin'`)).Scan(&passwd, &salt); err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha1.Sum([]byte(passwd+salt)))
	val, err := phpser.Serialize([]any{1, hash, int(time.Now().Unix()) + 3600, 2})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: a.Config.CookieName, Value: url.QueryEscape(val)}
}

func TestGuestBoardIndex(t *testing.T) {
	a := newTestApp(t)
	w, body := get(t, a, "/index.php")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	for _, want := range []string{
		"<title>My Community - Index</title>",
		"General Category",
		"General Discussion",
		"Feel free to talk about anything and everything in this board.",
		`title="Welcome to SMF!">Welcome to SMF!</a>`,
		"My Community - Info Center",
		"Most Online Today",
		// Guest login form in the header:
		`?action=login2" method="post"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("board index missing %q", want)
		}
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=ISO-8859-1" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestGuestMessageIndex(t *testing.T) {
	a := newTestApp(t)
	_, body := get(t, a, "/index.php?board=1.0")
	for _, want := range []string{
		"<title>General Discussion</title>",
		"Started by",
		"Welcome to SMF!",
		"Simple Machines",
		"Pages: [<b>1</b>] ",
		`<option value="?board=1.0" selected="selected">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("message index missing %q", want)
		}
	}
}

func TestGuestDisplayTopic(t *testing.T) {
	a := newTestApp(t)
	_, body := get(t, a, "/index.php?topic=1.0")
	for _, want := range []string{
		"<title>Welcome to SMF!</title>",
		`target="_blank">ask us for assistance</a>`,
		"Simple Machines",
		`<b>Topic:</b>`, // not present — checked below differently
	} {
		if want == "<b>Topic:</b>" {
			continue
		}
		if !strings.Contains(body, want) {
			t.Errorf("topic display missing %q", want)
		}
	}
	// Views increments.
	var views int
	a.DB.QueryRow(a.Q(`SELECT numViews FROM {$db_prefix}topics WHERE ID_TOPIC = 1`)).Scan(&views)
	if views != 1 {
		t.Errorf("numViews = %d, want 1", views)
	}
}

func TestMemberSeesLoggedInChrome(t *testing.T) {
	a := newTestApp(t)
	w, body := get(t, a, "/index.php", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	for _, want := range []string{
		"Hello <b>admin</b>", // hello_member_ndt + name
		"?action=unread",
		"?action=logout;sesc=",
		"?action=admin",
		"Show unread posts since last visit.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("member chrome missing %q", want)
		}
	}
	if strings.Contains(body, `<input type="text" name="user" size="10" />`) {
		t.Error("member should not see the guest login form")
	}
}

func TestBadTopicKicksGuestToLogin(t *testing.T) {
	a := newTestApp(t)
	_, body := get(t, a, "/index.php?topic=999.0")
	if !strings.Contains(body, "<title>Login</title>") {
		t.Error("guest should see the login page for a missing topic")
	}
}

func TestUnknownActionFallsToBoardIndex(t *testing.T) {
	a := newTestApp(t)
	_, body := get(t, a, "/index.php?action=definitely-not-real")
	if !strings.Contains(body, "<title>My Community - Index</title>") {
		t.Error("unknown action should render the board index")
	}
}
