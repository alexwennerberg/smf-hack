package app

// Registration + activation flow tests (Phase 2).

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// sessionValue reads a key out of the stored session row.
func sessionValue(t *testing.T, a *App, sessID, key string) string {
	t.Helper()
	var data string
	if err := a.DB.QueryRow(a.Q(`SELECT data FROM {$db_prefix}sessions WHERE session_id = ?`), sessID).Scan(&data); err != nil {
		t.Fatalf("session %s not found: %v", sessID, err)
	}
	var m map[string]any
	json.Unmarshal([]byte(data), &m)
	v, _ := m[key].(string)
	return v
}

func TestRegistrationImmediate(t *testing.T) {
	a := newTestApp(t)

	// 1. GET the registration form - sets the session + verification code.
	w, body := get(t, a, "/index.php?action=register")
	if !strings.Contains(body, `name="creator"`) || !strings.Contains(body, `name="regagree"`) {
		t.Fatalf("registration form incomplete")
	}
	var sessCk *http.Cookie
	for _, ck := range cookiesFrom(w) {
		if ck.Name == sessionCookieName {
			sessCk = ck
		}
	}
	if sessCk == nil {
		t.Fatal("no session cookie")
	}
	code := sessionValue(t, a, sessCk.Value, "visual_verification_code")
	if len(code) != 5 {
		t.Fatalf("verification code = %q", code)
	}

	// 2. POST the registration.
	clearFloodControl(a)
	w2 := postForm(t, a, "/index.php?action=register2", url.Values{
		"regagree":                 {"yes"},
		"user":                     {"newuser"},
		"email":                    {"new@example.com"},
		"passwrd1":                 {"secret123"},
		"passwrd2":                 {"secret123"},
		"visual_verification_code": {code},
	}, sessCk)

	// registration_method=0 -> immediate login redirect.
	if w2.Code != 302 || !strings.Contains(w2.Header().Get("Location"), "action=login2;sa=check;member=2") {
		t.Fatalf("register2: %d %q body=%.300s", w2.Code, w2.Header().Get("Location"), w2.Body.String())
	}

	// Member exists, activated, with the right password scheme.
	var passwd string
	var activated int
	a.DB.QueryRow(a.Q(`SELECT passwd, is_activated FROM {$db_prefix}members WHERE memberName = 'newuser'`)).Scan(&passwd, &activated)
	if passwd != sha1hex("newuser"+"secret123") || activated != 1 {
		t.Errorf("member row wrong: passwd match=%v activated=%d", passwd == sha1hex("newusersecret123"), activated)
	}

	// Stats updated.
	if a.Setting("latestRealName") != "newuser" || a.SettingInt("totalMembers") != 2 {
		t.Errorf("stats: latest=%q total=%s", a.Setting("latestRealName"), a.Setting("totalMembers"))
	}
}

func TestRegistrationWithActivation(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"registration_method": "1"})

	w, _ := get(t, a, "/index.php?action=register")
	var sessCk *http.Cookie
	for _, ck := range cookiesFrom(w) {
		if ck.Name == sessionCookieName {
			sessCk = ck
		}
	}
	code := sessionValue(t, a, sessCk.Value, "visual_verification_code")

	clearFloodControl(a)
	w2 := postForm(t, a, "/index.php?action=register2", url.Values{
		"regagree":                 {"yes"},
		"user":                     {"pending"},
		"email":                    {"pending@example.com"},
		"passwrd1":                 {"secret123"},
		"passwrd2":                 {"secret123"},
		"visual_verification_code": {code},
	}, sessCk)
	if w2.Code != 200 || !strings.Contains(w2.Body.String(), "activate") {
		t.Fatalf("activation register: %d %.300s", w2.Code, w2.Body.String())
	}

	// Unactivated with a validation code.
	var activated int
	var vcode string
	a.DB.QueryRow(a.Q(`SELECT is_activated, validation_code FROM {$db_prefix}members WHERE memberName = 'pending'`)).Scan(&activated, &vcode)
	if activated != 0 || len(vcode) != 10 {
		t.Fatalf("pending member: activated=%d code=%q", activated, vcode)
	}

	// Login before activation fails with the activation message.
	clearFloodControl(a)
	w3 := postForm(t, a, "/index.php?action=login2", url.Values{
		"user": {"pending"}, "passwrd": {"secret123"},
	})
	if !strings.Contains(w3.Body.String(), "activat") {
		t.Errorf("expected activation error on login, got %.200s", w3.Body.String())
	}

	// Activate with the code.
	_, body := get(t, a, "/index.php?action=activate;u=2;code="+vcode)
	if !strings.Contains(body, "Your account has now been activated") &&
		!strings.Contains(body, "activated") {
		t.Errorf("activation response: %.300s", body)
	}
	a.DB.QueryRow(a.Q(`SELECT is_activated FROM {$db_prefix}members WHERE memberName = 'pending'`)).Scan(&activated)
	if activated != 1 {
		t.Errorf("after activation: is_activated=%d", activated)
	}

	// Now login works.
	clearFloodControl(a)
	w4 := postForm(t, a, "/index.php?action=login2", url.Values{
		"user": {"pending"}, "passwrd": {"secret123"},
	})
	if w4.Code != 302 {
		t.Errorf("post-activation login: %d %.200s", w4.Code, w4.Body.String())
	}
}

func TestRegistrationBadVerification(t *testing.T) {
	a := newTestApp(t)
	w, _ := get(t, a, "/index.php?action=register")
	var sessCk *http.Cookie
	for _, ck := range cookiesFrom(w) {
		if ck.Name == sessionCookieName {
			sessCk = ck
		}
	}
	clearFloodControl(a)
	w2 := postForm(t, a, "/index.php?action=register2", url.Values{
		"regagree":                 {"yes"},
		"user":                     {"botuser"},
		"email":                    {"bot@example.com"},
		"passwrd1":                 {"secret123"},
		"passwrd2":                 {"secret123"},
		"visual_verification_code": {"WRONG"},
	}, sessCk)
	if !strings.Contains(w2.Body.String(), "letters you typed") {
		t.Errorf("expected verification failure, got %.300s", w2.Body.String())
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE memberName = 'botuser'`)).Scan(&n)
	if n != 0 {
		t.Error("bot user should not be registered")
	}
}

func TestReservedNameRejected(t *testing.T) {
	a := newTestApp(t)
	w, _ := get(t, a, "/index.php?action=register")
	var sessCk *http.Cookie
	for _, ck := range cookiesFrom(w) {
		if ck.Name == sessionCookieName {
			sessCk = ck
		}
	}
	code := sessionValue(t, a, sessCk.Value, "visual_verification_code")
	clearFloodControl(a)
	// 'Admin' is in the default reserveNames list.
	w2 := postForm(t, a, "/index.php?action=register2", url.Values{
		"regagree":                 {"yes"},
		"user":                     {"Admin"},
		"email":                    {"admin2@example.com"},
		"passwrd1":                 {"secret123"},
		"passwrd2":                 {"secret123"},
		"visual_verification_code": {code},
	}, sessCk)
	if !strings.Contains(w2.Body.String(), "name was reserved") &&
		!strings.Contains(w2.Body.String(), "reserved") {
		t.Errorf("expected reserved-name error, got %.300s", w2.Body.String())
	}
}

func TestVerificationLetterImage(t *testing.T) {
	a := newTestApp(t)
	w, _ := get(t, a, "/index.php?action=register")
	var sessCk *http.Cookie
	for _, ck := range cookiesFrom(w) {
		if ck.Name == sessionCookieName {
			sessCk = ck
		}
	}
	// The test assetdir has no fonts -> 400; with the real assetdir it would
	// serve a gif. Either way the handler must not 500.
	w2, _ := get(t, a, "/index.php?action=verificationcode;rand=abc;letter=1", sessCk)
	if w2.Code != 400 && w2.Code != 200 {
		t.Errorf("letter image status = %d", w2.Code)
	}
}
