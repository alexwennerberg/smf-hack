package app

// Table tests for the parse_bbc port. Expected values are hand-derived from
// Sources/Subs.php (SMF 1.1.21) semantics; they get re-verified against a
// real PHP 5.6 instance when the golden fixture corpus is generated
// (PORTING.md). Browser context: a plain non-IE/non-Gecko client (curl).

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestCtx boots an in-memory app + a guest request context.
func newTestCtx(t *testing.T) *Ctx {
	t.Helper()
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.DBPath = filepath.Join(dir, "test.sqlite")
	cfg.AssetDir = dir
	a, err := Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.DB.Close(); os.Remove(cfg.DBPath) })

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080/index.php", nil)
	c := NewCtx(a, w, r)
	c.loadSession()
	c.loadUserSettings()
	c.loadBoard()
	c.loadTheme()
	c.loadPermissions()
	return c
}

func TestParseBBCBasics(t *testing.T) {
	c := newTestCtx(t)
	smileyImg := func(file, alt string) string {
		return `<img src="` + c.App.Setting("smileys_url") + `/default/` + file + `" alt="` + alt + `" border="0" />`
	}
	tests := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"[b]bold[/b]", "<b>bold</b>"},
		{"[i]it[/i] and [u]un[/u]", `<i>it</i> and <span style="text-decoration: underline;">un</span>`},
		{"[s]gone[/s]", "<del>gone</del>"},
		{"a\nb", "a<br />b"},
		// Unclosed tags are closed at the end.
		{"[b]x", "<b>x</b>"},
		// Mis-nesting: closing a tag closes the ones inside it.
		{"[b][i]x[/b]", "<b><i>x</i></b>"},
		{"[color=red]r[/color]", `<span style="color: red;">r</span>`},
		{"[color=#ff0000]r[/color]", `<span style="color: #ff0000;">r</span>`},
		{"[size=2]big[/size]", `<font size="2" style="line-height: 1.3em;">big</font>`},
		{"[size=10pt]big[/size]", `<span style="font-size: 10pt; line-height: 1.3em;">big</span>`},
		{"[font=Verdana]v[/font]", `<span style="font-family: Verdana;">v</span>`},
		{"[tt]mono[/tt]", "<tt>mono</tt>"},
		{"[sub]s[/sub][sup]p[/sup]", "<sub>s</sub><sup>p</sup>"},
		{"[hr]", "<hr />"},
		{"[br]", "<br />"},
		// Block-level tags.
		{"[center]c[/center]", `<div align="center">c</div>`},
		{"[left]l[/left]", `<div style="text-align: left;">l</div>`},
		{"[right]r[/right]", `<div style="text-align: right;">r</div>`},
		{"[move]m[/move]", "<marquee>m</marquee>"},
		// Quote family (smf240='Quote', smf239='Quote from').
		{"[quote]q[/quote]", `<div class="quoteheader">Quote</div><div class="quote">q</div>`},
		{"[quote=Bob]q[/quote]", `<div class="quoteheader">Quote from: Bob</div><div class="quote">q</div>`},
		{"[quote author=Bob]q[/quote]", `<div class="quoteheader">Quote from: Bob</div><div class="quote">q</div>`},
		// URLs.
		{"[url]example.com[/url]", `<a href="http://example.com" target="_blank">http://example.com</a>`},
		{"[url=http://x.com]text[/url]", `<a href="http://x.com" target="_blank">text</a>`},
		{"[url=x.com]text[/url]", `<a href="http://x.com" target="_blank">text</a>`},
		{"[iurl=#top]up[/iurl]", `<a href="#post_top">up</a>`},
		{"[email]a@b.com[/email]", `<a href="mailto:a@b.com">a@b.com</a>`},
		{"[ftp]ftp.x.com[/ftp]", `<a href="ftp://ftp.x.com" target="_blank">ftp://ftp.x.com</a>`},
		// img.
		{"[img]http://x.com/i.gif[/img]", `<img src="http://x.com/i.gif" alt="" border="0" />`},
		{"[img width=10]x.com/i.gif[/img]", `<img src="http://x.com/i.gif" alt="" width="10" border="0" />`},
		// Lists.
		{"[list][li]a[/li][/list]", `<ul style="margin-top: 0; margin-bottom: 0;"><li>a</li></ul>`},
		// Itemcodes.
		{"[*] item\n", `<ul style="margin-top: 0; margin-bottom: 0;"><li> item</li></ul>`},
		// nobbc.
		{"[nobbc][b]x[/b][/nobbc]", "[b]x[/b]"},
		// code (non-Gecko browser: no <pre> wrapper).
		{"[code]x = 1;[/code]", `<div class="codeheader">Code:</div><div class="code">x = 1;</div>`},
		{"[code=php]x[/code]", `<div class="codeheader">Code: (php)</div><div class="code">x</div>`},
		// Smileys.
		{":)", smileyImg("smiley.gif", "Smiley")},
		{"hi :) there", "hi " + smileyImg("smiley.gif", "Smiley") + " there"},
		// No smiley mid-word.
		{"ab:)cd", "ab:)cd"},
		// Auto-linking.
		{"go to http://example.com now", `go to <a href="http://example.com" target="_blank">http://example.com</a> now`},
		{"go to www.example.com now", `go to <a href="http://www.example.com" target="_blank">www.example.com</a> now`},
		{"mail a@b.com please", `mail <a href="mailto:a@b.com">a@b.com</a> please`},
		// No autolink inside [url].
		{"[url=http://a.com]http://b.com[/url]", `<a href="http://a.com" target="_blank">http://b.com</a>`},
		// Tabs and double spaces.
		{"a\tb", "a&nbsp;&nbsp;&nbsp;b"},
		{"a  b", "a &nbsp;b"},
		// Leading space.
		{" x", "&nbsp;x"},
		// Unknown brackets pass through.
		{"a [not-a-tag] b", "a [not-a-tag] b"},
		// glow/shadow (non-IE).
		{"[glow=red,2]g[/glow]", `<span style="background-color: red;">g</span>`},
		// shadow: non-IE callback's return value is discarded (PHP quirk),
		// so the raw direction is emitted.
		{"[shadow=red,left]s[/shadow]", `<span style="text-shadow: red left">s</span>`},
		// Tables require proper children.
		{"[table][tr][td]x[/td][/tr][/table]",
			`<table style="font: inherit; color: inherit;"><tr><td valign="top" style="font: inherit; color: inherit;">x</td></tr></table>`},
		// Case-insensitive tags.
		{"[B]x[/B]", "<b>x</b>"},
		// abbr.
		{"[abbr=Hypertext]HT[/abbr]", `<abbr title="Hypertext">HT</abbr>`},
		// me.
		{"[me=Bob]waves[/me]", `<div class="meaction">* Bob waves</div>`},
	}
	for _, tt := range tests {
		if got := c.parseBBC(tt.in, true); got != tt.want {
			t.Errorf("parseBBC(%q)\n got: %q\nwant: %q", tt.in, got, tt.want)
		}
	}
}

func TestParseBBCDisabled(t *testing.T) {
	c := newTestCtx(t)
	// Flash is disabled by default (enableEmbeddedFlash=0): shows the link.
	got := c.parseBBC("[flash=100,100]x.com/f.swf[/flash]", true)
	want := `<a href="http://x.com/f.swf" target="_blank">http://x.com/f.swf</a>`
	if got != want {
		t.Errorf("flash disabled:\n got: %q\nwant: %q", got, want)
	}

	// enableBBC off: tags pass through, smileys still parse.
	if err := c.App.UpdateSettings(map[string]string{"enableBBC": "0"}); err != nil {
		t.Fatal(err)
	}
	got = c.parseBBC("[b]x[/b]", false)
	if got != "[b]x[/b]" {
		t.Errorf("enableBBC=0: got %q", got)
	}
	c.App.UpdateSettings(map[string]string{"enableBBC": "1"})
}

func TestParseBBCQuoteWithLink(t *testing.T) {
	c := newTestCtx(t)
	got := c.parseBBC(`[quote author=Bob link=topic=1.msg1#msg1 date=1000000000]q[/quote]`, true)
	// date=1000000000 formats via timeformat (Sep 09, 2001 01:46:40 UTC + offsets).
	if !strings.Contains(got, `<div class="quoteheader"><a href="`+c.App.ScriptURL+`?topic=1.msg1#msg1">Quote from: Bob on `) ||
		!strings.HasSuffix(got, `</a></div><div class="quote">q</div>`) {
		t.Errorf("quote with link: %q", got)
	}
}

func TestParseBBCNestedQuote(t *testing.T) {
	c := newTestCtx(t)
	got := c.parseBBC("[quote][quote]inner[/quote]outer[/quote]", true)
	want := `<div class="quoteheader">Quote</div><div class="quote"><div class="quoteheader">Quote</div><div class="quote">inner</div>outer</div>`
	if got != want {
		t.Errorf("nested quote:\n got: %q\nwant: %q", got, want)
	}
}
