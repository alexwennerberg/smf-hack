package app

// Robustness sweep: hit a broad matrix of actions across every role and assert
// nothing panics or 500s. This is the "doesn't error" complement to the golden
// crawl (which only proves read-parity for a curated set). Edge inputs included
// (missing/invalid ids, write actions reached by GET, bad sub-actions) — SMF
// handles these gracefully (form, redirect, or fatal-error PAGE), so a panic or
// a propagated server error here is a real bug.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// sweepRequest issues one request and reports (status, panicValue). A panic in
// the handler (anything other than the smfExit control-flow signal, which the
// router already absorbs) propagates here and is recovered so the sweep can
// flag it instead of crashing the test binary.
func sweepRequest(a *App, path string, cookies []*http.Cookie) (status int, body string, panicked any) {
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080"+path, nil)
	if i := strings.IndexByte(path, '?'); i >= 0 {
		u, _ := url.Parse("http://127.0.0.1:8080" + path[:i])
		u.RawQuery = path[i+1:]
		r.URL = u
	}
	for _, ck := range cookies {
		r.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	defer func() {
		if rec := recover(); rec != nil {
			panicked = rec
		}
	}()
	a.Handler().ServeHTTP(w, r)
	return w.Code, w.Body.String(), nil
}

func TestActionSweep(t *testing.T) {
	a := goldenApp(t)
	roleCookies := map[string][]*http.Cookie{
		"guest":  nil,
		"member": {goldenLoginCookie(t, a, "member")},
		"mod":    {goldenLoginCookie(t, a, "mod")},
		"admin":  {adminCookie(t, a)},
	}

	// Every registered action, reached with plausible ids plus a few edge
	// inputs. Write actions are intentionally hit by GET (no sesc): SMF answers
	// with a form/redirect/guest-resubmit page, never a crash.
	paths := []string{
		// Defaults & core read.
		"/index.php", "/index.php?board=1.0", "/index.php?board=2.0", "/index.php?topic=1.0",
		"/index.php?topic=203.0", "/index.php?board=1.10", "/index.php?topic=202.10",
		// Bad/edge ids.
		"/index.php?board=0.0", "/index.php?board=99999.0", "/index.php?topic=0.0",
		"/index.php?topic=99999.0", "/index.php?board=abc", "/index.php?topic=1.abc",
		"/index.php?action=doesnotexist", "/index.php?action=", "/index.php?action=display;topic=1.msg9999",
		// Feeds.
		"/index.php?action=.xml;type=rss", "/index.php?action=.xml;type=atom",
		"/index.php?action=.xml;type=rss2;sa=news", "/index.php?action=.xml;type=webslice",
		// Posting & topic ops (GET — show form or graceful error).
		"/index.php?action=post;board=1.0", "/index.php?action=post;topic=1.0",
		"/index.php?action=post;topic=99999.0", "/index.php?action=post2;board=1.0",
		"/index.php?action=editpoll;topic=203.0", "/index.php?action=editpoll;topic=1.0",
		"/index.php?action=vote;topic=203", "/index.php?action=removepoll;topic=203.0",
		"/index.php?action=lockVoting;topic=203.0", "/index.php?action=deletemsg;topic=1.0;msg=1",
		"/index.php?action=quotefast;quote=1", "/index.php?action=jsmodify;topic=1.0;msg=1",
		// Moderation.
		"/index.php?action=lock;topic=1.0", "/index.php?action=sticky;topic=1.0",
		"/index.php?action=quickmod;board=1.0", "/index.php?action=quickmod2;board=1.0",
		"/index.php?action=movetopic;topic=1.0", "/index.php?action=movetopic2;topic=1.0",
		"/index.php?action=mergetopics;board=1.0", "/index.php?action=splittopics;topic=202.0",
		"/index.php?action=removetopic2;topic=1.0", "/index.php?action=reporttm;topic=1.0;msg=1",
		"/index.php?action=modlog", "/index.php?action=trackip;searchip=127.0.0.1",
		// Notify / karma / markread.
		"/index.php?action=notify;topic=1.0", "/index.php?action=notifyboard;board=1.0",
		"/index.php?action=markasread;sa=all", "/index.php?action=markasread;sa=board;board=1.0",
		"/index.php?action=modifykarma;type=applaud;uid=2", "/index.php?action=collapse;c=1",
		// Members & profile.
		"/index.php?action=mlist", "/index.php?action=mlist;sa=search",
		"/index.php?action=profile;u=1", "/index.php?action=profile;u=2",
		"/index.php?action=profile;u=99999", "/index.php?action=profile;u=1;sa=statPanel",
		"/index.php?action=profile;u=1;sa=showPosts", "/index.php?action=profile;u=2;sa=summary",
		"/index.php?action=viewprofile;u=1", "/index.php?action=buddy;u=2",
		"/index.php?action=findmember", "/index.php?action=requestmembers",
		// PM.
		"/index.php?action=pm", "/index.php?action=pm;sa=send", "/index.php?action=pm;sa=settings",
		"/index.php?action=pm;f=inbox;sort=date", "/index.php?action=pm;sa=search",
		// Discovery & misc.
		"/index.php?action=recent", "/index.php?action=unread", "/index.php?action=unreadreplies",
		"/index.php?action=search", "/index.php?action=search2", "/index.php?action=stats",
		"/index.php?action=calendar", "/index.php?action=who", "/index.php?action=printpage;topic=1.0",
		"/index.php?action=sendtopic;topic=1.0", "/index.php?action=help", "/index.php?action=help;page=post",
		"/index.php?action=helpadmin;help=ban_members",
		// Auth/registration.
		"/index.php?action=login", "/index.php?action=logout", "/index.php?action=register",
		"/index.php?action=reminder", "/index.php?action=activate", "/index.php?action=coppa",
		"/index.php?action=verificationcode",
		// Admin home + per-area.
		"/index.php?action=admin", "/index.php?action=featuresettings", "/index.php?action=postsettings",
		"/index.php?action=serversettings", "/index.php?action=manageboards", "/index.php?action=viewmembers",
		"/index.php?action=viewmembers;sa=search", "/index.php?action=membergroups",
		"/index.php?action=permissions", "/index.php?action=ban;sa=add", "/index.php?action=ban;sa=list",
		"/index.php?action=reports", "/index.php?action=reports;rt=boards", "/index.php?action=reports;rt=staff",
		"/index.php?action=smileys;sa=editsets", "/index.php?action=smileys;sa=settings",
		"/index.php?action=manageattachments", "/index.php?action=managesearch", "/index.php?action=managecalendar",
		"/index.php?action=news;sa=editnews", "/index.php?action=news;sa=settings", "/index.php?action=viewErrorLog",
		"/index.php?action=maintain", "/index.php?action=boardrecount", "/index.php?action=repairboards",
		"/index.php?action=regcenter", "/index.php?action=theme;sa=settings", "/index.php?action=theme;sa=list",
	}

	// Endpoints that legitimately emit a tiny/binary/redirect body (not HTML).
	tinyOK := func(p string) bool {
		for _, s := range []string{"verificationcode", "dlattach", "collapse", "jsoption", "=logout", "quotefast", "jsmodify", "findmember", "buddy", "markasread", "notify", "modifykarma", ".xml", "trackip"} {
			if strings.Contains(p, s) {
				return true
			}
		}
		return false
	}
	type fail struct {
		role, path, reason string
	}
	var fails []fail
	hits := 0
	for _, p := range paths {
		for role, cookies := range roleCookies {
			// Reset users-online noise each hit (cheap; keeps who/info-center sane).
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online`))
			status, body, panicked := sweepRequest(a, p, cookies)
			hits++
			switch {
			case panicked != nil:
				fails = append(fails, fail{role, p, fmt.Sprintf("PANIC: %v", panicked)})
			case status >= 500:
				fails = append(fails, fail{role, p, fmt.Sprintf("STATUS %d", status)})
			case status == 200 && len(body) < 50 && !tinyOK(p):
				fails = append(fails, fail{role, p, fmt.Sprintf("EMPTY 200 (%d bytes)", len(body))})
			case strings.Contains(body, "{unknown"), strings.Contains(body, "%!"):
				// Unresolved {marker} / Go fmt verb leaked into output.
				fails = append(fails, fail{role, p, "unresolved template marker / fmt verb"})
			}
		}
	}
	t.Logf("action sweep: %d requests across %d paths × %d roles", hits, len(paths), len(roleCookies))
	for _, f := range fails {
		t.Errorf("[%s] %s — %s", f.role, f.path, f.reason)
	}
}
