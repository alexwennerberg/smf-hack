package app

// Phase 8 — fuzzing the text/query parsers. These are Go native fuzz targets;
// their seed corpora also run as ordinary cases under `go test`, so the tricky
// inputs are exercised in CI. Run an actual fuzzing campaign with e.g.:
//   go test ./internal/app/ -run x -fuzz FuzzParseBBC -fuzztime 30s

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// newFuzzCtx is newTestCtx for a testing.TB (so *testing.F works too).
func newFuzzCtx(tb testing.TB) *Ctx {
	tb.Helper()
	dir := tb.TempDir()
	cfg := DefaultConfig()
	cfg.DBPath = filepath.Join(dir, "test.sqlite")
	cfg.AssetDir = dir
	a, err := Open(cfg)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { a.DB.Close() })

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

// bbcSeeds are tricky / malformed BBC inputs that have historically tripped up
// scanners: unbalanced tags, deep nesting, naked brackets, partial tags,
// attribute edge cases, auto-link punctuation, code/php blocks, smiley
// boundaries, entity-ish sequences.
var bbcSeeds = []string{
	"",
	"plain text",
	"[b]bold[/b]",
	"[b]unclosed bold",
	"closed without open[/b]",
	"[b][i][u]deep[/u][/i][/b]",
	"[b][i]mismatched[/b][/i]",
	"[quote author=Bob link=topic=1.0 date=123]q[/quote]",
	"[url=http://example.com]link[/url]",
	"[url]http://example.com[/url]",
	"naked http://example.com/a?b=c&d=e end",
	"www.example.com, trailing punctuation.",
	"[img]http://x/y.png[/img]",
	"[code]<b>not parsed</b> [b]nor this[/b][/code]",
	"[code=php]echo 1;[/code]",
	"[php]<?php echo 'x'; ?>[/php]",
	"[list][li]a[li]b[/list]",
	"[color=red]r[/color] [color=#abcdef]hex[/color]",
	"[size=24pt]big[/size]",
	"[[[[[[nested brackets]]]]]]",
	"[/b][/i][/u] only closers",
	"unterminated [url=",
	"[b",
	"]b[",
	"[table][tr][td]cell[/td][/tr][/table]",
	":) :( ;) >:( :'( mixed smileys",
	"text:)nospace",
	"[b]a[/b][b]b[/b][b]c[/b] many",
	"&amp; &lt; &#1234; entities",
	"\x00\x01 control bytes \x1f",
	"multi\nline\r\ntext\rwith breaks",
	"[ftp]ftp://h/p[/ftp] [email]a@b.com[/email]",
	"[quote][quote][quote]deep quotes[/quote][/quote][/quote]",
	"[abbr=Hyper Text]HTML[/abbr]",
	"a]b[c]d[/e]f[g",
}

func FuzzParseBBC(f *testing.F) {
	for _, s := range bbcSeeds {
		f.Add(s, true)
		f.Add(s, false)
	}
	c := newFuzzCtx(f)
	f.Fuzz(func(t *testing.T, in string, smileys bool) {
		// Must not panic, and must be deterministic for the same input.
		out1 := c.parseBBCMode(in, smileys, false)
		out2 := c.parseBBCMode(in, smileys, false)
		if out1 != out2 {
			t.Fatalf("parseBBC not deterministic for %q", in)
		}
	})
}

func FuzzPreparseCode(f *testing.F) {
	for _, s := range bbcSeeds {
		f.Add(s)
	}
	c := newFuzzCtx(f)
	f.Fuzz(func(t *testing.T, in string) {
		// preparsecode then parse — the post pipeline; must not panic.
		pre := c.preparsecode(in, true)
		_ = c.parseBBC(c.unPreparsecode(pre), true)
	})
}

// querySeeds are malformed / adversarial query strings.
var querySeeds = []string{
	"",
	"a=1",
	"a=1;b=2;c=3",
	"a=1&b=2&c=3",
	"action=post;board=1.0;sa=x",
	"a[]=1;a[]=2",
	"a[b][c]=1;a[b][d]=2",
	"a[=broken",
	"a]=broken",
	"=noname",
	"key%20space=val%20ue",
	"k=%ZZ", // bad percent-encoding
	"k=" + "verylong",
	"topic=1.msg5;start=new",
	"a=1;a=2;a=3", // repeated scalar
	"GLOBALS=x",
	"a[GLOBALS]=x",
	";;;;",
	"&&&&",
	"a=1;;b=2",
	"foo=%41%42%43",
	"x[y][]=1;x[y][]=2;x[z]=3",
}

func FuzzQueryString(f *testing.F) {
	for _, s := range querySeeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		// Must not panic, and must round-trip deterministically.
		p1 := ParseQueryString(raw)
		p2 := ParseQueryString(raw)
		if p1.Len() != p2.Len() {
			t.Fatalf("ParseQueryString not deterministic for %q (%d vs %d keys)", raw, p1.Len(), p2.Len())
		}
		// Every key must be retrievable without panicking.
		for _, k := range p1.Keys() {
			_ = p1.Str(k)
		}
	})
}
