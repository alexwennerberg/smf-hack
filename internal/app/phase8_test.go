package app

// Phase 8 — parity sweep: the small cross-cutting behaviors (easter egg,
// robots noindex, gzip output) that the golden crawl doesn't cover.

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// getUA is get() with a chosen User-Agent (and optional cookies).
func getUA(t *testing.T, a *App, path, ua string, cookies ...*http.Cookie) (*httptest.ResponseRecorder, string) {
	t.Helper()
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil)
	if i := strings.IndexByte(path, '?'); i >= 0 {
		u, _ := url.Parse("http://127.0.0.1:8080" + path[:i])
		u.RawQuery = path[i+1:]
		r.URL = u
	}
	if ua != "" {
		r.Header.Set("User-Agent", ua)
	}
	for _, ck := range cookies {
		r.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	return w, w.Body.String()
}

const geckoUA = "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0"

func TestEasterEggBookOfUnknown(t *testing.T) {
	a := newTestApp(t)

	// about:unknown renders the Book of Unknown; verse 2:18 is selectable.
	_, body := getUA(t, a, "/index.php?action=about:unknown;verse=2:18", geckoUA)
	if !strings.Contains(body, "The Book of Unknown, 2:18") {
		t.Fatalf("easter egg page not served:\n%.300s", body)
	}

	// about:mozilla is the browser joke: Gecko -> about:mozilla, else firefox.
	w, _ := getUA(t, a, "/index.php?action=about:mozilla", geckoUA)
	if w.Code != 302 || w.Header().Get("Location") != "about:mozilla" {
		t.Fatalf("Gecko should bounce to about:mozilla: code=%d loc=%q", w.Code, w.Header().Get("Location"))
	}
	w, _ = getUA(t, a, "/index.php?action=about:mozilla", "SomeBot/1.0")
	if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "getfirefox.com") {
		t.Fatalf("non-Gecko not redirected to firefox: code=%d loc=%q", w.Code, w.Header().Get("Location"))
	}
}

func TestRobotsNoindex(t *testing.T) {
	a := newTestApp(t)

	// Seed a topic so display has something to render.
	res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}topics (ID_BOARD, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, numReplies) VALUES (1, 1, 1, 0)`))
	tid, _ := res.LastInsertId()
	mres, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterTime, subject, body) VALUES (?, 1, 1, 'admin', ?, 's', 'b')`), tid, nowUnix())
	mid, _ := mres.LastInsertId()
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_FIRST_MSG = ?, ID_LAST_MSG = ? WHERE ID_TOPIC = ?`), mid, mid, tid)

	// A normal topic view is indexable.
	_, body := get(t, a, "/index.php?topic="+itoa(int(tid))+".0")
	if strings.Contains(body, `name="robots" content="noindex"`) {
		t.Errorf("normal topic view should be indexable")
	}

	// A "jump to message" view (topic=N.msgM) is a duplicate link -> noindex.
	_, body = get(t, a, "/index.php?topic="+itoa(int(tid))+".msg"+itoa(int(mid)))
	if !strings.Contains(body, `name="robots" content="noindex"`) {
		t.Errorf("topic .msg view should carry the noindex meta:\n%.300s", body[:min(len(body), 600)])
	}
}

func TestGzipOutput(t *testing.T) {
	a := newTestApp(t)
	// enableCompressedOutput defaults on; be explicit.
	a.UpdateSettings(map[string]string{"enableCompressedOutput": "1"})

	// Without Accept-Encoding: plain HTML.
	w, body := get(t, a, "/index.php")
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatalf("must not gzip when the client did not ask")
	}
	if !strings.Contains(body, "<html") {
		t.Fatalf("uncompressed body should be HTML")
	}

	// With Accept-Encoding: gzip -> compressed, and it inflates back to HTML.
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080/index.php", nil)
	r.Header.Set("Accept-Encoding", "gzip, deflate")
	rw := httptest.NewRecorder()
	a.Handler().ServeHTTP(rw, r)
	if rw.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip Content-Encoding, got %q", rw.Header().Get("Content-Encoding"))
	}
	gz, err := gzip.NewReader(rw.Body)
	if err != nil {
		t.Fatalf("response is not valid gzip: %v", err)
	}
	dec, _ := io.ReadAll(gz)
	if !strings.Contains(string(dec), "<html") || !strings.Contains(string(dec), "My Community") {
		t.Fatalf("inflated body is not the expected page:\n%.200s", dec)
	}

	// When disabled, gzip is never applied even if the client asks.
	a.UpdateSettings(map[string]string{"enableCompressedOutput": "0"})
	r2 := httptest.NewRequest("GET", "http://127.0.0.1:8080/index.php", nil)
	r2.Header.Set("Accept-Encoding", "gzip")
	rw2 := httptest.NewRecorder()
	a.Handler().ServeHTTP(rw2, r2)
	if rw2.Header().Get("Content-Encoding") == "gzip" {
		t.Fatalf("must not gzip when enableCompressedOutput is off")
	}
}
