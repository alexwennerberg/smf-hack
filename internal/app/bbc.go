package app

// Faithful port of parse_bbc() / parsesmileys() from Sources/Subs.php
// (lines 958–2483). The $bbc_codes definition table is reproduced as a
// struct slice with the same fields, and the strpos-driven scanner loop is
// translated statement by statement — output must be byte-identical.
//
// Browser-dependent markup (code/flash/glow/shadow) is resolved per request,
// exactly as PHP rebuilt its static table per request.

import (
	"strings"
	"time"
)

func nowUnix() int64 { return time.Now().Unix() }

type bbcType int

const (
	typeParsedContent bbcType = iota // (missing): [tag]parsed content[/tag]
	typeUnparsedEquals
	typeParsedEquals
	typeUnparsedContent
	typeClosed
	typeUnparsedCommas
	typeUnparsedCommasContent
	typeUnparsedEqualsContent
)

type bbcParam struct {
	name     string
	match    string // regex for the value; default (.+?)
	quoted   bool
	optional bool
	value    string                        // replacement with $1
	validate func(c *Ctx, v string) string // 'validate' callback (parse_bbc / timeformat)
}

type bbcCode struct {
	tag              string
	typ              bbcType
	parameters       []bbcParam
	test             string // regex tested right after '=', ' ' or ']'
	content          string
	before           string
	after            string
	disabledContent  string
	disabledBefore   string
	disabledAfter    string
	hasDisContent    bool
	hasDisBefore     bool
	hasDisAfter      bool
	blockLevel       bool
	trim             string // "", "inside", "outside", "both"
	validate         func(c *Ctx, tag *bbcOpen, data *bbcData)
	quoted           string // "", "optional", "required"
	requireParents   []string
	requireChildren  []string
	disallowChildren []string
}

// bbcOpen is an entry on the open-tags stack: a mutable copy of a code.
type bbcOpen struct {
	tag              string
	typ              bbcType
	content          string
	before           string
	after            string
	blockLevel       bool
	trim             string
	requireChildren  []string
	disallowChildren []string
}

// bbcData carries the validate-callback data: scalar for most types, list
// for commas/equals_content types.
type bbcData struct {
	s    string
	list []string
}

func inList(s string, list []string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// stripos is PHP stripos (ASCII case-insensitive, fine for tag names).
func stripos(haystack, needle string, offset int) int {
	if offset > len(haystack) {
		return -1
	}
	i := strings.Index(strings.ToLower(haystack[offset:]), strings.ToLower(needle))
	if i < 0 {
		return -1
	}
	return offset + i
}

// substr is PHP substr for our use (start always >= 0 here).
func substr(s string, start, length int) string {
	if start >= len(s) {
		return ""
	}
	end := start + length
	if length < 0 || end > len(s) {
		end = len(s)
	}
	if end < start {
		return ""
	}
	return s[start:end]
}

// bbcDisabled computes the $disabled set for this request.
func (c *Ctx) bbcDisabled(printMode bool) map[string]bool {
	disabled := map[string]bool{}
	if !c.App.SettingEmpty("disabledBBC") {
		for _, tag := range strings.Split(strings.ToLower(c.App.Setting("disabledBBC")), ",") {
			disabled[strings.TrimSpace(tag)] = true
		}
	}
	if c.App.SettingEmpty("enableEmbeddedFlash") {
		disabled["flash"] = true
	}
	if printMode {
		for _, t := range []string{"glow", "shadow", "move", "color", "black", "blue", "white",
			"red", "green", "me", "php", "ftp", "url", "iurl", "email", "flash"} {
			disabled[t] = true
		}
		if !c.GET.Has("images") {
			disabled["img"] = true
		}
	}
	return disabled
}

// stripBR removes <br /> from data, as several validate callbacks do.
func stripBR(s string) string { return strings.ReplaceAll(s, "<br />", "") }

func ensureHTTP(s string) string {
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return "http://" + s
	}
	return s
}

func ensureFTP(s string) string {
	if !strings.HasPrefix(s, "ftp://") && !strings.HasPrefix(s, "ftps://") {
		return "ftp://" + s
	}
	return s
}

// bbcCodes builds the $codes table for this request (browser-dependent
// entries included). Order matters: same as the PHP literal.
func (c *Ctx) bbcCodes(disabled map[string]bool) []bbcCode {
	isIE := c.Browser.IsIE && !c.Browser.IsMacIE
	isGecko := c.Browser.IsGecko

	codeContent := `<div class="codeheader">` + c.Txt("smf238") + `:</div><div class="code">$1</div>`
	codeEqContent := `<div class="codeheader">` + c.Txt("smf238") + `: ($2)</div><div class="code">$1</div>`
	if isGecko {
		codeContent = `<div class="codeheader">` + c.Txt("smf238") + `:</div><div class="code"><pre style="margin-top: 0; display: inline;">$1</pre></div>`
		codeEqContent = `<div class="codeheader">` + c.Txt("smf238") + `: ($2)</div><div class="code"><pre style="margin-top: 0; display: inline;">$1</pre></div>`
	}

	var codeValidate func(c *Ctx, tag *bbcOpen, data *bbcData)
	if !disabled["code"] {
		codeValidate = func(c *Ctx, tag *bbcOpen, data *bbcData) {
			target := &data.s
			if data.list != nil {
				target = &data.list[0]
			}
			*target = c.highlightPHPSections(*target)
		}
	}

	flashContent := `<embed type="application/x-shockwave-flash" src="$1" width="$2" height="$3" play="true" loop="true" quality="high" AllowScriptAccess="never" /><noembed><a href="$1" target="_blank">$1</a></noembed>`
	if c.Browser.IsIE && !c.Browser.IsMacIE {
		flashContent = `<object classid="clsid:D27CDB6E-AE6D-11cf-96B8-444553540000" width="$2" height="$3"><param name="movie" value="$1" /><param name="play" value="true" /><param name="loop" value="true" /><param name="quality" value="high" /><param name="AllowScriptAccess" value="never" /><embed src="$1" width="$2" height="$3" play="true" loop="true" quality="high" AllowScriptAccess="never" /><noembed><a href="$1" target="_blank">$1</a></noembed></object>`
	}

	glowBefore, glowAfter := `<span style="background-color: $1;">`, `</span>`
	if c.Browser.IsIE {
		glowBefore = `<table border="0" cellpadding="0" cellspacing="0" style="display: inline; vertical-align: middle; font: inherit;"><tr><td style="filter: Glow(color=$1, strength=$2); font: inherit;">`
		glowAfter = `</td></tr></table> `
	}

	shadowBefore := `<span style="text-shadow: $1 $2">`
	if c.Browser.IsIE {
		shadowBefore = `<span style="filter: Shadow(color=$1, direction=$2); height: 1.2em;\">`
	}
	var shadowValidate func(c *Ctx, tag *bbcOpen, data *bbcData)
	if c.Browser.IsIE {
		shadowValidate = func(c *Ctx, tag *bbcOpen, data *bbcData) {
			switch data.list[1] {
			case "left":
				data.list[1] = "270"
			case "right":
				data.list[1] = "90"
			case "top":
				data.list[1] = "0"
			case "bottom":
				data.list[1] = "180"
			default:
				data.list[1] = itoa(atoi(data.list[1]))
			}
		}
	} else {
		// PHP's non-IE callback only *returns* a value, which the caller
		// discards — so the data passes through unchanged. Bug-for-bug.
		shadowValidate = func(c *Ctx, tag *bbcOpen, data *bbcData) {}
	}

	codes := []bbcCode{
		{tag: "abbr", typ: typeUnparsedEquals, before: `<abbr title="$1">`, after: `</abbr>`,
			quoted: "optional", disabledAfter: ` ($1)`, hasDisAfter: true},
		{tag: "acronym", typ: typeUnparsedEquals, before: `<acronym title="$1">`, after: `</acronym>`,
			quoted: "optional", disabledAfter: ` ($1)`, hasDisAfter: true},
		{tag: "anchor", typ: typeUnparsedEquals, test: `[#]?([A-Za-z][A-Za-z0-9_\-]*)\]`,
			before: `<span id="post_$1" />`, after: ``},
		{tag: "b", before: `<b>`, after: `</b>`},
		{tag: "black", before: `<span style="color: black;">`, after: `</span>`},
		{tag: "blue", before: `<span style="color: blue;">`, after: `</span>`},
		{tag: "br", typ: typeClosed, content: `<br />`},
		{tag: "code", typ: typeUnparsedContent, content: codeContent,
			validate: codeValidate, blockLevel: true},
		{tag: "code", typ: typeUnparsedEqualsContent, content: codeEqContent,
			validate: codeValidate, blockLevel: true},
		{tag: "center", before: `<div align="center">`, after: `</div>`, blockLevel: true},
		{tag: "color", typ: typeUnparsedEquals, test: `(#[\da-fA-F]{3}|#[\da-fA-F]{6}|[A-Za-z]{1,12})\]`,
			before: `<span style="color: $1;">`, after: `</span>`},
		{tag: "email", typ: typeUnparsedContent, content: `<a href="mailto:$1">$1</a>`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) { data.s = stripBR(data.s) }},
		{tag: "email", typ: typeUnparsedEquals, before: `<a href="mailto:$1">`, after: `</a>`,
			disallowChildren: []string{"email", "ftp", "url", "iurl"},
			disabledAfter:    ` ($1)`, hasDisAfter: true},
		{tag: "ftp", typ: typeUnparsedContent, content: `<a href="$1" target="_blank">$1</a>`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) { data.s = ensureFTP(stripBR(data.s)) }},
		{tag: "ftp", typ: typeUnparsedEquals, before: `<a href="$1" target="_blank">`, after: `</a>`,
			validate:         func(c *Ctx, tag *bbcOpen, data *bbcData) { data.s = ensureFTP(data.s) },
			disallowChildren: []string{"email", "ftp", "url", "iurl"},
			disabledAfter:    ` ($1)`, hasDisAfter: true},
		{tag: "font", typ: typeUnparsedEquals, test: `[A-Za-z0-9_,\-\s]+?\]`,
			before: `<span style="font-family: $1;">`, after: `</span>`},
		{tag: "flash", typ: typeUnparsedCommasContent, test: `\d+,\d+\]`, content: flashContent,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				if disabled["url"] {
					tag.content = `$1`
				} else if !strings.HasPrefix(data.list[0], "http://") && !strings.HasPrefix(data.list[0], "https://") {
					data.list[0] = "http://" + data.list[0]
				}
			},
			disabledContent: `<a href="$1" target="_blank">$1</a>`, hasDisContent: true},
		{tag: "green", before: `<span style="color: green;">`, after: `</span>`},
		{tag: "glow", typ: typeUnparsedCommas, test: `[#0-9a-zA-Z\-]{3,12},([012]\d{1,2}|\d{1,2})(,[^]]+)?\]`,
			before: glowBefore, after: glowAfter},
		{tag: "hr", typ: typeClosed, content: `<hr />`, blockLevel: true},
		{tag: "html", typ: typeUnparsedContent, content: `$1`, blockLevel: true,
			disabledContent: `$1`, hasDisContent: true},
		{tag: "img", typ: typeUnparsedContent,
			parameters: []bbcParam{
				{name: "alt", optional: true},
				{name: "width", optional: true, value: ` width="$1"`, match: `(\d+)`},
				{name: "height", optional: true, value: ` height="$1"`, match: `(\d+)`},
			},
			content: `<img src="$1" alt="{alt}"{width}{height} border="0" />`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				data.s = ensureHTTP(stripBR(data.s))
			},
			disabledContent: `($1)`, hasDisContent: true},
		{tag: "img", typ: typeUnparsedContent, content: `<img src="$1" alt="" border="0" />`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				data.s = ensureHTTP(stripBR(data.s))
			},
			disabledContent: `($1)`, hasDisContent: true},
		{tag: "i", before: `<i>`, after: `</i>`},
		{tag: "iurl", typ: typeUnparsedContent, content: `<a href="$1">$1</a>`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				data.s = ensureHTTP(stripBR(data.s))
			}},
		{tag: "iurl", typ: typeUnparsedEquals, before: `<a href="$1">`, after: `</a>`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				if strings.HasPrefix(data.s, "#") {
					data.s = "#post_" + data.s[1:]
				} else {
					data.s = ensureHTTP(data.s)
				}
			},
			disallowChildren: []string{"email", "ftp", "url", "iurl"},
			disabledAfter:    ` ($1)`, hasDisAfter: true},
		{tag: "li", before: `<li>`, after: `</li>`, trim: "outside",
			requireParents: []string{"list"}, blockLevel: true,
			disabledBefore: ``, hasDisBefore: true,
			disabledAfter: `<br />`, hasDisAfter: true},
		{tag: "list", before: `<ul style="margin-top: 0; margin-bottom: 0;">`, after: `</ul>`,
			trim: "inside", requireChildren: []string{"li"}, blockLevel: true},
		{tag: "list",
			parameters: []bbcParam{
				{name: "type", match: `(none|disc|circle|square|decimal|decimal-leading-zero|lower-roman|upper-roman|lower-alpha|upper-alpha|lower-greek|lower-latin|upper-latin|hebrew|armenian|georgian|cjk-ideographic|hiragana|katakana|hiragana-iroha|katakana-iroha)`},
			},
			before: `<ul style="margin-top: 0; margin-bottom: 0; list-style-type: {type};">`, after: `</ul>`,
			trim: "inside", requireChildren: []string{"li"}, blockLevel: true},
		{tag: "left", before: `<div style="text-align: left;">`, after: `</div>`, blockLevel: true},
		{tag: "ltr", before: `<div dir="ltr">`, after: `</div>`, blockLevel: true},
		{tag: "me", typ: typeUnparsedEquals, before: `<div class="meaction">* $1 `, after: `</div>`,
			quoted: "optional", blockLevel: true,
			disabledBefore: `/me `, hasDisBefore: true,
			disabledAfter: `<br />`, hasDisAfter: true},
		{tag: "move", before: `<marquee>`, after: `</marquee>`, blockLevel: true},
		{tag: "nobbc", typ: typeUnparsedContent, content: `$1`},
		{tag: "pre", before: `<pre>`, after: `</pre>`},
		{tag: "php", typ: typeUnparsedContent, content: `<div class="phpcode">$1</div>`,
			validate: func() func(c *Ctx, tag *bbcOpen, data *bbcData) {
				if disabled["php"] {
					return nil
				}
				return func(c *Ctx, tag *bbcOpen, data *bbcData) {
					data.s = c.highlightPHPBlock(data.s)
				}
			}(),
			blockLevel:      true,
			disabledContent: `$1`, hasDisContent: true},
		{tag: "quote", before: `<div class="quoteheader">` + c.Txt("smf240") + `</div><div class="quote">`,
			after: `</div>`, blockLevel: true},
		{tag: "quote",
			parameters: []bbcParam{
				{name: "author", match: `(.{1,192}?)`, quoted: true,
					validate: func(c *Ctx, v string) string { return c.parseBBC(v, true) }},
			},
			before: `<div class="quoteheader">` + c.Txt("smf239") + `: {author}</div><div class="quote">`,
			after:  `</div>`, blockLevel: true},
		{tag: "quote", typ: typeParsedEquals,
			before: `<div class="quoteheader">` + c.Txt("smf239") + `: $1</div><div class="quote">`,
			after:  `</div>`, quoted: "optional", blockLevel: true},
		{tag: "quote",
			parameters: []bbcParam{
				{name: "author", match: `([^<>]{1,192}?)`},
				{name: "link", match: `(?:board=\d+;)?((?:topic|threadid)=[\dmsg#\./]{1,40}(?:;start=[\dmsg#\./]{1,40})?|action=profile;u=\d+)`},
				{name: "date", match: `(\d+)`,
					validate: func(c *Ctx, v string) string { return c.timeformat(int64(atoi(v))) }},
			},
			before: `<div class="quoteheader"><a href="` + c.App.ScriptURL + `?{link}">` + c.Txt("smf239") + `: {author} ` + c.Txt("176") + ` {date}</a></div><div class="quote">`,
			after:  `</div>`, blockLevel: true},
		{tag: "quote",
			parameters: []bbcParam{
				{name: "author", match: `(.{1,192}?)`,
					validate: func(c *Ctx, v string) string { return c.parseBBC(v, true) }},
			},
			before: `<div class="quoteheader">` + c.Txt("smf239") + `: {author}</div><div class="quote">`,
			after:  `</div>`, blockLevel: true},
		{tag: "right", before: `<div style="text-align: right;">`, after: `</div>`, blockLevel: true},
		{tag: "red", before: `<span style="color: red;">`, after: `</span>`},
		{tag: "rtl", before: `<div dir="rtl">`, after: `</div>`, blockLevel: true},
		{tag: "s", before: `<del>`, after: `</del>`},
		{tag: "size", typ: typeUnparsedEquals, test: `([1-9][\d]?p[xt]|(?:x-)?small(?:er)?|(?:x-)?large[r]?)\]`,
			before: `<span style="font-size: $1; line-height: 1.3em;">`, after: `</span>`},
		{tag: "size", typ: typeUnparsedEquals, test: `[1-9]\]`,
			before: `<font size="$1" style="line-height: 1.3em;">`, after: `</font>`},
		{tag: "sub", before: `<sub>`, after: `</sub>`},
		{tag: "sup", before: `<sup>`, after: `</sup>`},
		{tag: "shadow", typ: typeUnparsedCommas, test: `[#0-9a-zA-Z\-]{3,12},(left|right|top|bottom|[0123]\d{0,2})\]`,
			before: shadowBefore, after: `</span>`, validate: shadowValidate},
		{tag: "time", typ: typeUnparsedContent, content: `$1`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				if isAllDigits(strings.TrimSpace(data.s)) && data.s != "" {
					data.s = c.timeformat(int64(atoi(data.s)))
				} else {
					tag.content = `[time]$1[/time]`
				}
			}},
		{tag: "tt", before: `<tt>`, after: `</tt>`},
		{tag: "table", before: `<table style="font: inherit; color: inherit;">`, after: `</table>`,
			trim: "inside", requireChildren: []string{"tr"}, blockLevel: true},
		{tag: "tr", before: `<tr>`, after: `</tr>`, requireParents: []string{"table"},
			requireChildren: []string{"td"}, trim: "both", blockLevel: true,
			disabledBefore: ``, hasDisBefore: true, disabledAfter: ``, hasDisAfter: true},
		{tag: "td", before: `<td valign="top" style="font: inherit; color: inherit;">`, after: `</td>`,
			requireParents: []string{"tr"}, trim: "outside", blockLevel: true,
			disabledBefore: ``, hasDisBefore: true, disabledAfter: ``, hasDisAfter: true},
		{tag: "url", typ: typeUnparsedContent, content: `<a href="$1" target="_blank">$1</a>`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				data.s = ensureHTTP(stripBR(data.s))
			}},
		{tag: "url", typ: typeUnparsedEquals, before: `<a href="$1" target="_blank">`, after: `</a>`,
			validate: func(c *Ctx, tag *bbcOpen, data *bbcData) {
				data.s = ensureHTTP(data.s)
			},
			disallowChildren: []string{"email", "ftp", "url", "iurl"},
			disabledAfter:    ` ($1)`, hasDisAfter: true},
		{tag: "u", before: `<span style="text-decoration: underline;">`, after: `</span>`},
		{tag: "white", before: `<span style="color: white;">`, after: `</span>`},
	}

	_ = isIE

	// Shhhh!
	if !disabled["color"] {
		codes = append(codes,
			bbcCode{tag: "chrissy", before: `<span style="color: #CC0099;">`, after: ` :-*</span>`},
			bbcCode{tag: "kissy", before: `<span style="color: #CC0099;">`, after: ` :-*</span>`},
		)
	}
	return codes
}

var bbcItemcodes = map[byte]string{
	'*': "", '@': "disc", '+': "square", 'x': "square", '#': "square",
	'o': "circle", 'O': "circle", '0': "circle",
}

var bbcNoAutolinkTags = []string{"url", "iurl", "ftp", "email"}
