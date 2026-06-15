package app

// parsesmileys() and the autolink/HTML/long-words passes used inside
// parse_bbc()'s text segments. PHP's patterns use lookbehind/lookahead which
// Go's RE2 lacks, so the core patterns are matched with regexp and the
// context conditions checked manually — non-consuming, like PCRE.

import (
	"regexp"
	"strings"
	"time"
)

type smileyDef struct {
	code, filename, description string
}

// defaultSmileys is the hardcoded list used when smiley_enable is off
// (descriptions resolved per request from $txt).
var defaultSmileyData = []struct {
	code, filename, txtKey string
}{
	{">:D", "evil.gif", ""},
	{":D", "cheesy.gif", "289"},
	{"::)", "rolleyes.gif", "450"},
	{">:(", "angry.gif", "288"},
	{":)", "smiley.gif", "287"},
	{";)", "wink.gif", "292"},
	{";D", "grin.gif", "293"},
	{":(", "sad.gif", "291"},
	{":o", "shocked.gif", "294"},
	{"8)", "cool.gif", "295"},
	{":P", "tongue.gif", "451"},
	{"???", "huh.gif", "296"},
	{":-[", "embarrassed.gif", "526"},
	{":-X", "lipsrsealed.gif", "527"},
	{":-*", "kiss.gif", "529"},
	{":'(", "cry.gif", "530"},
	{":-\\", "undecided.gif", "528"},
	{"^-^", "azn.gif", ""},
	{"O0", "afro.gif", ""},
	{"C:-)", "police.gif", ""},
	{"0:)", "angel.gif", ""},
}

// smileyList returns the (code, replacement-img) pairs in parse order.
func (c *Ctx) smileyList() []smileyDef {
	a := c.App
	var defs []smileyDef
	if a.SettingEmpty("smiley_enable") {
		// Use the default smileys if it is disabled. (better for
		// "portability" of smileys.)
		for _, s := range defaultSmileyData {
			desc := ""
			if s.txtKey != "" {
				desc = c.Txt(s.txtKey)
			}
			defs = append(defs, smileyDef{s.code, s.filename, desc})
		}
		return defs
	}

	if v, ok := a.cache.Get("parsing_smileys"); ok {
		return v.([]smileyDef)
	}
	// PHP relied on the physical row order produced by sortSmileyTable()
	// (ALTER TABLE ... ORDER BY LENGTH(code) DESC) so longer codes match
	// first; SQLite can't reorder rows physically, so we order in the query.
	rows, err := a.DB.Query(a.Q(`SELECT code, filename, description FROM {$db_prefix}smileys ORDER BY LENGTH(code) DESC`))
	if err == nil {
		for rows.Next() {
			var d smileyDef
			rows.Scan(&d.code, &d.filename, &d.description)
			defs = append(defs, d)
		}
		rows.Close()
	}
	a.cache.Put("parsing_smileys", defs, 480*time.Second)
	return defs
}

// smileyPrevOK is the lookbehind class [>:\?\.\s\xA0[\]()*\\;] .
func smileyPrevOK(b byte) bool {
	switch b {
	case '>', ':', '?', '.', ' ', '\t', '\r', '\n', '\v', '\f', 0xA0, '[', ']', '(', ')', '*', '\\', ';':
		return true
	}
	return false
}

// smileyNextOK is the lookahead (?=[^[:alpha:]0-9]|$).
func smileyNextOK(s string, pos int) bool {
	if pos >= len(s) {
		return true
	}
	b := s[pos]
	return !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9')
}

// parseSmileys is parsesmileys(&$message): each smiley pattern is applied
// over the whole message in turn, like preg_replace with pattern arrays.
func (c *Ctx) parseSmileys(message string) string {
	if c.User.SmileySet == "none" {
		return message
	}

	smileysURL := c.App.Setting("smileys_url") + "/" + c.User.SmileySet + "/"

	for _, def := range c.smileyList() {
		if def.code == "" {
			continue
		}
		// Escape a bunch of smiley-related characters in the description so
		// it doesn't get a double dose :P.
		desc := strings.NewReplacer(":", "&#58;", "(", "&#40;", ")", "&#41;", "$", "&#36;", "[", "&#091;").
			Replace(Htmlspecialchars(def.description))
		img := `<img src="` + Htmlspecialchars(smileysURL+def.filename) + `" alt="` + desc + `" border="0" />`

		// Both the raw code and its htmlspecialchars form match.
		alts := []string{def.code}
		if hsc := Htmlspecialchars(def.code); hsc != def.code {
			alts = append(alts, hsc)
		}

		var b strings.Builder
		i := 0
		for i < len(message) {
			matched := ""
			if i == 0 || smileyPrevOK(message[i-1]) {
				for _, alt := range alts {
					if strings.HasPrefix(message[i:], alt) && smileyNextOK(message, i+len(alt)) {
						matched = alt
						break
					}
				}
			}
			if matched != "" {
				b.WriteString(img)
				i += len(matched)
			} else {
				b.WriteByte(message[i])
				i++
			}
		}
		message = b.String()
	}
	return message
}

// ---- autolink ----

var urlCoreRe = regexp.MustCompile(`(?i)((?:http|https|ftp|ftps)://[\w\-_%@:|]+(?:\.[\w\-_%]+)*(?::\d+)?(?:/[\w\-_~%\.@,\?&;=#(){}+:'\\]*)*[/\w\-_~%@\?;=#}\\])`)
var wwwCoreRe = regexp.MustCompile(`(?i)(www(?:\.[\w\-_]+)+(?::\d+)?(?:/[\w\-_~%\.@,\?&;=#(){}+:'\\]*)*[/\w\-_~%@\?;=#}\\])`)
var emailCoreRe = regexp.MustCompile(`([\w\-\.]{1,80}@[\w\-]+\.[\w\-\.]+[\w\-])`)

// Lookahead conditions for the two email patterns, checked manually
// against the text after a match. \xA0 (ISO-8859-1 nbsp) is checked
// separately because Go regexp is UTF-8-oriented.
var emailNext1Re = regexp.MustCompile(`^(?:[?,\t\n\f\r \[\]()*\\]|$|<br />|&nbsp;|&gt;|&lt;|&quot;|&#039;|\.(?:\.|;|&nbsp;|[\t\n\f\r ]|$|<br />))`)
var emailNext2Re = regexp.MustCompile(`^(?:[?\.,;\t\n\f\r \[\]()*\\]|$|<br />|&nbsp;|&gt;|&lt;|&quot;|&#039;)`)

func emailNext1(rest string) bool {
	return emailNext1Re.MatchString(rest) || strings.HasPrefix(rest, "\xA0") ||
		strings.HasPrefix(rest, ".\xA0")
}

func emailNext2(rest string) bool {
	return emailNext2Re.MatchString(rest) || strings.HasPrefix(rest, "\xA0")
}

func classContains(class string, b byte) bool {
	return strings.IndexByte(class, b) >= 0
}

// replaceLookbehind emulates preg_replace with a 1-char lookbehind class:
// matches of core whose preceding byte is in prevClass (or start of string)
// are replaced via repl($1).
func replaceLookbehind(s string, core *regexp.Regexp, prevClass string, nextOK func(rest string) bool, repl func(m string) string) string {
	matches := core.FindAllStringIndex(s, -1)
	if matches == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, loc := range matches {
		start, end := loc[0], loc[1]
		if start < last {
			continue
		}
		prevOK := start == 0 || classContains(prevClass, s[start-1])
		if !prevOK || (nextOK != nil && !nextOK(s[end:])) {
			continue
		}
		b.WriteString(s[last:start])
		b.WriteString(repl(s[start:end]))
		last = end
	}
	b.WriteString(s[last:])
	return b.String()
}

// replaceAfterBR is the second email pattern: lookbehind is exactly '<br />'.
func replaceAfterBR(s string, core *regexp.Regexp, nextOK func(rest string) bool, repl func(m string) string) string {
	matches := core.FindAllStringIndex(s, -1)
	if matches == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, loc := range matches {
		start, end := loc[0], loc[1]
		if start < last {
			continue
		}
		if start < 6 || s[start-6:start] != "<br />" {
			continue
		}
		if nextOK != nil && !nextOK(s[end:]) {
			continue
		}
		b.WriteString(s[last:start])
		b.WriteString(repl(s[start:end]))
		last = end
	}
	b.WriteString(s[last:])
	return b.String()
}

// autoLink applies the URL and email autolinking to a text segment.
func (c *Ctx) autoLink(data string, disabled map[string]bool) string {
	// Parse any URLs.... have to get rid of the @ problems some things
	// cause... stupid email addresses.
	if !disabled["url"] && (strings.Contains(data, "://") || strings.Contains(data, "www.")) {
		// Switch out quotes really quick because they can cause problems.
		data = strings.NewReplacer("&#039;", "'", "&nbsp;", "\xA0", "&quot;", `>">`, `"`, `<"<`, "&lt;", "<lt<").Replace(data)

		data = replaceLookbehind(data, urlCoreRe, " \t\r\n\v\f>.(;'\"", nil,
			func(m string) string { return "[url]" + m + "[/url]" })
		data = replaceLookbehind(data, wwwCoreRe, " \t\r\n\v\f>('<", nil,
			func(m string) string { return "[url=http://" + m + "]" + m + "[/url]" })

		data = strings.NewReplacer("'", "&#039;", "\xA0", "&nbsp;", `>">`, "&quot;", `<"<`, `"`, "<lt<", "&lt;").Replace(data)
	}

	// Next, emails...
	if !disabled["email"] && strings.Contains(data, "@") {
		data = replaceLookbehind(data, emailCoreRe, "? \t\r\n\v\f\xA0[]()*\\;>",
			emailNext1,
			func(m string) string { return "[email]" + m + "[/email]" })
		data = replaceAfterBR(data, emailCoreRe,
			emailNext2,
			func(m string) string { return "[email]" + m + "[/email]" })
	}
	return data
}

// ---- enablePostHTML conversion ----

var htmlARe = regexp.MustCompile(`(?i)&lt;a\s+href=((?:&quot;)?)((?:https?://|ftps?://|mailto:)\S+?)(?:&quot;)?&gt;`)
var htmlCloseARe = regexp.MustCompile(`(?i)&lt;/a&gt;`)
var htmlImgRe = regexp.MustCompile(`(?i)&lt;img\s+src=((?:&quot;)?)((?:https?://|ftps?://)\S+?)(?:&quot;)?(?:\s+alt=(&quot;.*?&quot;|\S*?))?(?:\s?/)?&gt;`)

// convertPostHTML is the enablePostHTML block in parse_bbc's segment loop.
func (c *Ctx) convertPostHTML(data string) string {
	data = htmlARe.ReplaceAllString(data, "[url=$2]")
	data = htmlCloseARe.ReplaceAllString(data, "[/url]")

	// <br /> should be empty.
	for _, tag := range []string{"br", "hr"} {
		data = strings.NewReplacer(
			"&lt;"+tag+"&gt;", "["+tag+" /]",
			"&lt;"+tag+"/&gt;", "["+tag+" /]",
			"&lt;"+tag+" /&gt;", "["+tag+" /]").Replace(data)
	}

	// b, u, i, s, pre... basic tags.
	for _, tag := range []string{"b", "u", "i", "s", "em", "ins", "del", "pre", "blockquote"} {
		diff := strings.Count(data, "&lt;"+tag+"&gt;") - strings.Count(data, "&lt;/"+tag+"&gt;")
		data = strings.NewReplacer("&lt;"+tag+"&gt;", "<"+tag+">", "&lt;/"+tag+"&gt;", "</"+tag+">").Replace(data)
		if diff > 0 {
			data += strings.Repeat("</"+tag+">", diff)
		}
	}

	// Do <img ... /> - with security... action= -> action-.
	if matches := htmlImgRe.FindAllStringSubmatch(data, -1); matches != nil {
		replaces := map[string]string{}
		for _, m := range matches {
			full, imgtag, altRaw := m[0], m[2], m[3]
			alt := ""
			if altRaw != "" {
				alt = " alt=" + strings.TrimSuffix(strings.TrimPrefix(altRaw, "&quot;"), "&quot;")
			}

			// Remove action= from the URL - no funny business, now.
			// (?!dlattach) negative lookahead, done manually for RE2.)
			imgtag = stripActionParam(imgtag)

			// (max_image_width/height resizing needs url_image_size's remote
			// fetch — deferred; see PORTING.md. Without the limits set, PHP
			// takes this same plain path.)
			replaces[full] = "[img" + alt + "]" + imgtag + "[/img]"
		}
		for from, to := range replaces {
			data = strings.ReplaceAll(data, from, to)
		}
	}
	return data
}

// stripActionParam replaces action= / action%3d with action- unless followed
// by 'dlattach' (PHP's (?!dlattach) lookahead).
func stripActionParam(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	i := 0
	for i < len(s) {
		matched := 0
		if strings.HasPrefix(lower[i:], "action=") {
			matched = len("action=")
		} else if strings.HasPrefix(lower[i:], "action%3d") {
			matched = len("action%3d")
		}
		if matched > 0 && !strings.HasPrefix(lower[i+matched:], "dlattach") {
			b.WriteString("action-")
			i += matched
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// ---- fixLongWords ----

// fixLongWords ports the long-word breaker (browser-dependent <span>).
func (c *Ctx) fixLongWords(data string, fixLong int) string {
	var breaker string
	switch {
	case c.Browser.IsGecko || c.Browser.IsKonqueror:
		breaker = `<span style="margin: 0 -0.5ex 0 0;"> </span>`
	case c.Browser.IsOpera:
		breaker = `<span style="margin: 0 -0.65ex 0 -1px;"> </span>`
	default:
		breaker = `<span style="width: 0; margin: 0 -0.6ex 0 -1px;"> </span>`
	}

	if fixLong > 65535 {
		fixLong = 65535
	}
	if len(data) <= fixLong {
		return data
	}

	// This is done in a roundabout way because $breaker has "long words" :P.
	data = strings.NewReplacer(breaker, "< >", "&nbsp;", "\xA0").Replace(data)

	wordRe := getCachedRe(`[\w\.]{` + itoa(fixLong) + `,}`)
	data = replaceLookbehind(data, wordRe, ">;:!? \xA0])(", nil, func(word string) string {
		var b strings.Builder
		for i := 0; i < len(word); i += fixLong - 1 {
			end := min(i+fixLong-1, len(word))
			b.WriteString(word[i:end])
			if end < len(word) || end == i+fixLong-1 {
				b.WriteString("< >")
			}
		}
		return b.String()
	})

	return strings.NewReplacer("< >", breaker, "\xA0", "&nbsp;").Replace(data)
}
