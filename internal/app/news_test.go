package app

// Smoke tests for the XML feeds (?action=.xml) across formats and sub-actions.

import (
	"net/url"
	"strings"
	"testing"
)

func seedTopic(t *testing.T, a *App, subject, message string) {
	t.Helper()
	admin := adminCookie(t, a)
	sc, seq, cookies := openPostForm(t, a, "/index.php?action=post;board=1.0", admin)
	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {subject},
		"message":            {message},
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
		t.Fatalf("seed post status %d", w.Code)
	}
}

func TestNewsFeedRSS2(t *testing.T) {
	a := newTestApp(t)
	seedTopic(t, a, "Feed topic one", "The [b]body[/b] of the feed post.")

	w, body := get(t, a, "/index.php?action=.xml;type=rss2;sa=recent", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("rss2 status %d:\n%.400s", w.Code, body)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/rss+xml") {
		t.Errorf("rss2 Content-Type = %q", ct)
	}
	for _, want := range []string{
		`<rss version="2.0"`,
		"<channel>",
		"<item>",
		"<title><![CDATA[Feed topic one]]></title>",
		"<guid>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rss2 feed missing %q\n%s", want, body)
		}
	}
	// The BBC tag was parsed and the entities pulled out of CDATA.
	if !strings.Contains(body, "<b>body</b>") {
		t.Errorf("rss2 feed body not BBC-parsed:\n%s", body)
	}
}

func TestNewsFeedAtomAndSMF(t *testing.T) {
	a := newTestApp(t)
	seedTopic(t, a, "Atom topic", "Atom body text.")

	_, atom := get(t, a, "/index.php?action=.xml;type=atom;sa=recent", adminCookie(t, a))
	for _, want := range []string{`<feed version="0.3"`, "<entry>", "<summary type=\"html\">", "<author>"} {
		if !strings.Contains(atom, want) {
			t.Errorf("atom feed missing %q", want)
		}
	}

	_, smf := get(t, a, "/index.php?action=.xml;sa=recent", adminCookie(t, a))
	for _, want := range []string{"<smf:xml-feed", "<recent-post>", "<starter>", "<board>"} {
		if !strings.Contains(smf, want) {
			t.Errorf("smf feed missing %q", want)
		}
	}
}

func TestNewsFeedMembersAndRDF(t *testing.T) {
	a := newTestApp(t)

	_, members := get(t, a, "/index.php?action=.xml;type=rss2;sa=members", adminCookie(t, a))
	if !strings.Contains(members, "<title><![CDATA[admin]]></title>") {
		t.Errorf("members feed missing the admin:\n%s", members)
	}

	seedTopic(t, a, "RDF topic", "Rdf body.")
	_, rdf := get(t, a, "/index.php?action=.xml;type=rdf;sa=recent", adminCookie(t, a))
	for _, want := range []string{"<rdf:RDF", "<rdf:Seq>", "rdf:about=", "<dc:format>text/html</dc:format>"} {
		if !strings.Contains(rdf, want) {
			t.Errorf("rdf feed missing %q", want)
		}
	}
}

func TestNewsFeedDisabled(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"xmlnews_enable": ""})
	w, body := get(t, a, "/index.php?action=.xml;sa=recent", adminCookie(t, a))
	if w.Code != 200 || strings.Contains(body, "<smf:xml-feed") {
		t.Errorf("disabled feed should produce no feed body (status %d)", w.Code)
	}
}
