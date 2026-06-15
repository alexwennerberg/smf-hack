package app

// Login/logout flow tests (Phase 2).

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func postForm(t *testing.T, a *App, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080"+path, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	return w
}

func clearFloodControl(a *App) {
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_floodcontrol`))
}

func cookiesFrom(w *httptest.ResponseRecorder) []*http.Cookie {
	res := http.Response{Header: w.Header()}
	return res.Cookies()
}

func TestLoginWrongPassword(t *testing.T) {
	a := newTestApp(t)
	w := postForm(t, a, "/index.php?action=login2", url.Values{
		"user": {"admin"}, "passwrd": {"wrong-password"},
	})
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Password incorrect") {
		t.Errorf("expected password-incorrect error, got: %.200s", body)
	}
	// An error log entry should exist.
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_errors`)).Scan(&n)
	if n != 1 {
		t.Errorf("log_errors count = %d, want 1", n)
	}
}

func TestLoginLogoutFlow(t *testing.T) {
	a := newTestApp(t)

	// 1. Log in with the plain password.
	w := postForm(t, a, "/index.php?action=login2", url.Values{
		"user": {"admin"}, "passwrd": {"testpass"},
	})
	if w.Code != 302 {
		t.Fatalf("login status = %d, want 302; body: %.200s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "action=login2;sa=check;member=1") {
		t.Fatalf("login redirect = %q", loc)
	}
	cookies := cookiesFrom(w)
	var loginCk, sessCk *http.Cookie
	for _, ck := range cookies {
		switch ck.Name {
		case a.Config.CookieName:
			loginCk = ck
		case sessionCookieName:
			sessCk = ck
		}
	}
	if loginCk == nil || sessCk == nil {
		t.Fatalf("missing cookies after login: %v", cookies)
	}

	// The salt must have been set on first login.
	var salt string
	a.DB.QueryRow(a.Q(`SELECT passwordSalt FROM {$db_prefix}members WHERE ID_MEMBER = 1`)).Scan(&salt)
	if len(salt) != 4 {
		t.Errorf("passwordSalt = %q, want 4 chars", salt)
	}

	// 2. Follow up: the cookie logs us in.
	_, body := get(t, a, "/index.php", loginCk, sessCk)
	if !strings.Contains(body, "Hello <b>admin</b>") {
		t.Fatal("cookie login did not produce member chrome")
	}

	// 3. sa=check redirects home.
	w2, _ := get(t, a, "/index.php?action=login2;sa=check;member=1", loginCk, sessCk)
	if w2.Code != 302 || w2.Header().Get("Location") != a.ScriptURL {
		t.Errorf("check redirect: %d %q", w2.Code, w2.Header().Get("Location"))
	}

	// 4. Logout needs the right sesc; fetch it from the menu first.
	idx := strings.Index(body, "?action=logout;sesc=")
	sesc := body[idx+len("?action=logout;sesc=") : idx+len("?action=logout;sesc=")+32]
	w3, _ := get(t, a, "/index.php?action=logout;sesc="+sesc, loginCk, sessCk)
	if w3.Code != 302 {
		t.Fatalf("logout status = %d", w3.Code)
	}
	// Cookie is cleared (id 0) and salt rotated.
	var salt2 string
	a.DB.QueryRow(a.Q(`SELECT passwordSalt FROM {$db_prefix}members WHERE ID_MEMBER = 1`)).Scan(&salt2)
	if salt2 == salt {
		t.Error("passwordSalt should rotate on logout")
	}
}

func TestLoginViaEmail(t *testing.T) {
	a := newTestApp(t)
	w := postForm(t, a, "/index.php?action=login2", url.Values{
		"user": {"a@b.com"}, "passwrd": {"testpass"},
	})
	if w.Code != 302 {
		t.Fatalf("email login status = %d; body: %.200s", w.Code, w.Body.String())
	}
}

func TestLoginFloodControl(t *testing.T) {
	a := newTestApp(t)
	postForm(t, a, "/index.php?action=login2", url.Values{"user": {"admin"}, "passwrd": {"x"}})
	// Immediately try again - flood control should complain.
	w := postForm(t, a, "/index.php?action=login2", url.Values{"user": {"admin"}, "passwrd": {"x"}})
	if !strings.Contains(w.Body.String(), "few seconds") && !strings.Contains(w.Body.String(), "second") {
		t.Errorf("expected flood-control message, got %.300s", w.Body.String())
	}
	clearFloodControl(a)
	// After clearing, we get the normal wrong-password error again.
	w = postForm(t, a, "/index.php?action=login2", url.Values{"user": {"admin"}, "passwrd": {"x"}})
	if !strings.Contains(w.Body.String(), "Password incorrect") {
		t.Errorf("expected password error after flood clear")
	}
}

func TestGuestLoginPage(t *testing.T) {
	a := newTestApp(t)
	_, body := get(t, a, "/index.php?action=login")
	for _, want := range []string{
		"<title>Login</title>",
		`name="frmLogin"`,
		`name="hash_passwrd"`,
		"sha1.js",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("login page missing %q", want)
		}
	}
}
