//go:build ignore

// Generate BBC fixtures from the live SMF parse_bbc(): POST a corpus to
// bbc_harness.php and write base64(input)\tbase64(php_output) records to the
// committed fixture file. The Go test (bbc_fixtures_test.go) then checks that
// parseBBC reproduces each PHP output. Run: go run bbc_gen.go
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const harness = "http://127.0.0.1:8080/bbc_harness.php"
const fixtureFile = "/home/alex/dev/smf-port/go/internal/app/testdata/bbc_fixtures.txt"

// corpus — exercise every BBC tag, nesting/mis-nesting, auto-linking,
// smileys, whitespace, and edge cases. Outputs must be deterministic (no
// "today" timestamps): the only dated input uses a fixed unix time.
var corpus = []string{
	// Basics & inline.
	"plain text", "[b]bold[/b]", "[i]it[/i] and [u]un[/u]", "[s]gone[/s]",
	"[b][i][u]nested[/u][/i][/b]", "a\nb", "a\n\nb", "[b]x", "[b][i]x[/b]",
	"[b]a[/b][b]b[/b]", "[i]un[b]bold[/b]closed",
	"[tt]mono[/tt]", "[pre]pre text[/pre]", "[sub]s[/sub][sup]p[/sup]",
	"[hr]", "[br]", "text[hr]more",
	// Color / size / font.
	"[color=red]r[/color]", "[color=#ff0000]r[/color]", "[color=#FF0000]r[/color]",
	"[red]r[/red]", "[green]g[/green]", "[blue]b[/blue]", "[white]w[/white]", "[black]k[/black]",
	"[size=2]big[/size]", "[size=10pt]big[/size]", "[size=24pt]huge[/size]",
	"[font=Verdana]v[/font]", "[font=Times New Roman]t[/font]",
	// Alignment / block.
	"[center]c[/center]", "[left]l[/left]", "[right]r[/right]", "[move]m[/move]",
	"[glow=red,2]g[/glow]", "[shadow=red,left]s[/shadow]",
	// Quotes.
	"[quote]q[/quote]", "[quote=Bob]q[/quote]", "[quote author=Bob]q[/quote]",
	"[quote][quote]inner[/quote]outer[/quote]",
	"[quote author=Bob link=topic=1.msg1#msg1 date=1000000000]q[/quote]",
	"[quote author=A][quote author=B]b[/quote]a[/quote]",
	// Code / php / nobbc / html.
	"[code]x = 1;[/code]", "[code=php]x[/code]", "[code][b]not bold[/b][/code]",
	"[nobbc][b]x[/b][/nobbc]", "[php]echo 'hi';[/php]",
	"[html]<b>raw</b>[/html]",
	// URLs / email / ftp.
	"[url]example.com[/url]", "[url=http://x.com]text[/url]", "[url=x.com]text[/url]",
	"[url=http://a.com]http://b.com[/url]", "[iurl=#top]up[/iurl]", "[iurl=http://x.com]i[/iurl]",
	"[email]a@b.com[/email]", "[email=a@b.com]mail me[/email]",
	"[ftp]ftp.x.com[/ftp]", "[ftp=ftp://x.com]f[/ftp]",
	// Images.
	"[img]http://x.com/i.gif[/img]", "[img width=10]x.com/i.gif[/img]",
	"[img width=10 height=20]http://x.com/i.gif[/img]", "[img height=20]http://x.com/i.gif[/img]",
	// Lists / itemcodes.
	"[list][li]a[/li][/list]", "[list][li]a[/li][li]b[/li][/list]",
	"[list type=decimal][li]a[/li][/list]", "[list][li]a[list][li]nested[/li][/list][/li][/list]",
	"[*] item\n", "[*] one\n[*] two\n",
	// Tables.
	"[table][tr][td]x[/td][/tr][/table]",
	"[table][tr][td]a[/td][td]b[/td][/tr][tr][td]c[/td][td]d[/td][/tr][/table]",
	// abbr / acronym / me / time / anchor.
	"[abbr=Hypertext]HT[/abbr]", "[acronym=As Soon As Possible]ASAP[/acronym]",
	"[me=Bob]waves[/me]", "[time]1000000000[/time]", "[anchor=top]a[/anchor]",
	"[flash=100,100]x.com/f.swf[/flash]",
	// Smileys.
	":)", ":(", ";)", ":D", ">:(", ":P", ":-[", ":-X", ":-\\", ":-*", ":'(",
	"8)", ":o", "::)", ":-/", ":-\\ huh", "hi :) there", "ab:)cd", ":):):)",
	// Auto-linking.
	"go to http://example.com now", "go to www.example.com now", "mail a@b.com please",
	"https://secure.example.com/path?q=1&r=2", "see (http://example.com) ok",
	"end http://example.com.", "http://example.com, and more",
	"ftp://files.example.com/x", "visit www.example.com/path/to/page.html",
	// Whitespace / escaping.
	"a\tb", "a  b", "a   b", " x", "\tindented", "trailing \n",
	"line1\nline2\nline3",
	// Entities & special chars (already htmlspecialchars'd at the door — feed raw text).
	"a < b > c", "tom & jerry", "say \"hi\"", "it's fine",
	// Unknown / passthrough.
	"a [not-a-tag] b", "[b]x[/notb]", "[unknown]y[/unknown]", "[[b]x[/b]]",
	// Case-insensitivity.
	"[B]x[/B]", "[QUOTE]q[/QUOTE]", "[Url=http://x.com]t[/Url]",
	// Mixed / realistic.
	"[b]Bold[/b] and [url=http://x.com]a link[/url] and :) a smiley.",
	"[quote author=admin]nested [b]quote[/b][/quote]\n[code]if (x) { return; }[/code]",
	"Check [list][li]one[/li][li]two[/li][/list] and [color=red]red[/color]",
}

func main() {
	payload, _ := json.Marshal(b64all(corpus))
	resp, err := http.Post(harness, "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Println("harness error:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var outs []string
	if err := json.Unmarshal(body, &outs); err != nil {
		fmt.Println("decode error:", err, string(body[:min(200, len(body))]))
		os.Exit(1)
	}
	if len(outs) != len(corpus) {
		fmt.Printf("length mismatch: %d inputs, %d outputs\n", len(corpus), len(outs))
		os.Exit(1)
	}
	var b strings.Builder
	for i, in := range corpus {
		b.WriteString(base64.StdEncoding.EncodeToString([]byte(in)))
		b.WriteByte('\t')
		b.WriteString(outs[i]) // already base64 from PHP
		b.WriteByte('\n')
	}
	if err := os.WriteFile(fixtureFile, []byte(b.String()), 0644); err != nil {
		fmt.Println("write error:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d BBC fixtures to %s\n", len(corpus), fixtureFile)
}

func b64all(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = base64.StdEncoding.EncodeToString([]byte(s))
	}
	return out
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
