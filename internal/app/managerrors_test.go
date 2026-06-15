package app

// Phase 7: the error-log viewer and deletion.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestViewErrorLog(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	now := nowUnix()
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_errors (logTime, ID_MEMBER, ip, url, message, session) VALUES (?, ?, ?, ?, ?, ?)`),
		now, 1, "127.0.0.1", "?action=admin", "First logged error", "abc123")
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_errors (logTime, ID_MEMBER, ip, url, message, session) VALUES (?, ?, ?, ?, ?, ?)`),
		now, 0, "10.0.0.1", "?action=who", "Guest error here", "")

	w, body := get(t, a, "/index.php?action=viewErrorLog", admin)
	if w.Code != 200 {
		t.Fatalf("viewErrorLog: status %d", w.Code)
	}
	if !strings.Contains(body, "First logged error") || !strings.Contains(body, "Guest error here") {
		t.Errorf("error log missing messages:\n%.500s", body)
	}
	// Member link for admin (id 1), guest label for id 0.
	if !strings.Contains(body, "?action=profile;u=1") {
		t.Errorf("error log missing member link")
	}
	// Delete checkboxes present.
	if !strings.Contains(body, `name="delete[]"`) {
		t.Errorf("error log missing delete checkboxes")
	}
}

func TestDeleteErrorSelected(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	now := nowUnix()
	res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_errors (logTime, ID_MEMBER, ip, url, message, session) VALUES (?, 1, '1.1.1.1', '?a=b', 'doomed error', 'x')`), now)
	id64, _ := res.LastInsertId()
	id := int(id64)

	// Grab sc + session cookie from the log page.
	w, body := get(t, a, "/index.php?action=viewErrorLog", admin)
	sc := scRe.FindStringSubmatch(body)
	if sc == nil {
		t.Fatalf("no sc in error log form")
	}
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)

	wd := postForm(t, a, "/index.php?action=viewErrorLog", url.Values{
		"delete[]": {itoa(id)},
		"sc":       {sc[1]},
	}, cookies...)
	if wd.Code != 302 {
		t.Fatalf("delete error: status %d body %.300s", wd.Code, wd.Body.String())
	}

	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_errors WHERE ID_ERROR = ?`), id).Scan(&n)
	if n != 0 {
		t.Fatalf("error %d not deleted", id)
	}
}
