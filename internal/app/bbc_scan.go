package app

// The parse_bbc() scanner loop (Subs.php 1679–2421), translated statement by
// statement. See bbc.go for the codes table.

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// parseBBCCached is parse_bbc($message, $smileys, $cache_id).
func (c *Ctx) parseBBCCached(message string, smileys bool, cacheID string) string {
	a := c.App
	// PHP only caches when cache_enable is on; our in-process cache is always
	// available, so mirror the length gate only.
	if cacheID != "" && len(message) > 1000 {
		key := "parse:" + cacheID + "-" + fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf(
			"%x-%v%s%v%f%s", md5.Sum([]byte(message)), smileys, a.Setting("disabledBBC"),
			c.Browser, c.User.TimeOffset, c.User.TimeFormat))))
		if v, ok := a.cache.Get(key); ok {
			return v.(string)
		}
		out := c.parseBBC(message, smileys)
		a.cache.Put(key, out, 240*time.Second)
		return out
	}
	return c.parseBBC(message, smileys)
}

// parseBBC is parse_bbc($message, $smileys).
func (c *Ctx) parseBBC(message string, smileys bool) string {
	return c.parseBBCMode(message, smileys, false)
}

// parseBBCPrint is parse_bbc($message, 'print').
func (c *Ctx) parseBBCPrint(message string) string {
	return c.parseBBCMode(message, false, true)
}

func (c *Ctx) parseBBCMode(message string, smileys bool, printMode bool) string {
	a := c.App

	if a.SettingEmpty("enableBBC") {
		if smileys {
			message = c.parseSmileys(message)
		}
		return message
	}

	disabled := c.bbcDisabled(printMode)

	// Build per-first-letter index like $bbc_codes.
	bbcIndex := map[byte][]bbcCode{}
	for _, code := range c.bbcCodes(disabled) {
		bbcIndex[code.tag[0]] = append(bbcIndex[code.tag[0]], code)
	}
	// So the parser won't skip itemcode letters.
	itemcodesEnabled := !disabled["li"] && !disabled["list"]
	if itemcodesEnabled {
		for ch := range bbcItemcodes {
			if _, ok := bbcIndex[ch]; !ok {
				bbcIndex[ch] = []bbcCode{}
			}
		}
	}

	var openTags []*bbcOpen
	message = strings.ReplaceAll(message, "\n", "<br />")

	pos := -1
	lastPos := -1
	haveLastPos := false
	lastAutoPos := 0

	for {
		if !haveLastPos {
			lastPos = pos
			haveLastPos = true
		} else if pos > lastPos {
			lastPos = pos
		}
		idx := strings.IndexByte(message[min(pos+1, len(message)):], '[')
		if idx >= 0 {
			pos = min(pos+1, len(message)) + idx
		} else {
			pos = -1
		}

		// Failsafe.
		if pos < 0 || lastPos > pos {
			pos = len(message) + 1
		}

		// Can't have a one letter smiley, URL, or email! (sorry.)
		if lastPos < pos-1 {
			// We want to eat one less, and one more, character (for smileys.)
			lastPos = max(lastPos-1, 0)
			data := substr(message, lastPos, pos-lastPos+1)

			// Take care of some HTML!
			if !a.SettingEmpty("enablePostHTML") && strings.Contains(data, "&lt;") {
				data = c.convertPostHTML(data)
			}

			if !a.SettingEmpty("autoLinkUrls") {
				// Are we inside tags that should not be auto linked?
				noAutolinkArea := false
				for _, ot := range openTags {
					if inList(ot.tag, bbcNoAutolinkTags) {
						noAutolinkArea = true
					}
				}

				// Don't go backwards.
				if pos < lastAutoPos {
					noAutolinkArea = true
				}
				lastAutoPos = pos

				if !noAutolinkArea {
					data = c.autoLink(data, disabled)
				}
			}

			data = strings.ReplaceAll(data, "\t", "&nbsp;&nbsp;&nbsp;")

			if fixLong := a.SettingInt("fixLongWords"); fixLong > 5 {
				data = c.fixLongWords(data, fixLong)
			}

			// Do any smileys!
			if smileys {
				data = c.parseSmileys(data)
			}

			// If it wasn't changed, no copying or other boring stuff has to
			// happen!
			if data != substr(message, lastPos, pos-lastPos+1) {
				message = message[:lastPos] + data + substr(message, pos+1, -1)

				// Since we changed it, look again in case we added or removed
				// a tag. But we don't want to skip any.
				oldPos := len(data) + lastPos - 1
				idx := strings.IndexByte(message[min(lastPos, len(message)):], '[')
				if idx >= 0 {
					pos = min(min(lastPos, len(message))+idx, oldPos)
				} else {
					pos = oldPos
				}
			}
		}

		// Are we there yet?  Are we there yet?
		if pos >= len(message)-1 {
			break
		}

		tagChar := lcByte(message[pos+1])

		if tagChar == '/' && len(openTags) > 0 {
			pos2 := strings.IndexByte(message[pos+1:], ']')
			if pos2 < 0 {
				continue
			}
			pos2 += pos + 1
			if pos2 == pos+2 {
				continue
			}
			lookFor := strings.ToLower(substr(message, pos+2, pos2-pos-2))

			var toClose []*bbcOpen
			blockLevel := -1 // -1 = null, 0 = false, 1 = true
			var tag *bbcOpen
			for {
				if len(openTags) == 0 {
					tag = nil
					break
				}
				tag = openTags[len(openTags)-1]
				openTags = openTags[:len(openTags)-1]

				if tag.blockLevel {
					// Only find out if we need to.
					if blockLevel == 0 {
						openTags = append(openTags, tag)
						break
					}

					// The idea is, if we are LOOKING for a block level tag,
					// we can close them on the way.
					if len(lookFor) > 0 {
						if codes, ok := bbcIndex[lookFor[0]]; ok {
							for _, temp := range codes {
								if temp.tag == lookFor {
									if temp.blockLevel {
										blockLevel = 1
									} else {
										blockLevel = 0
									}
									break
								}
							}
						}
					}

					if blockLevel != 1 {
						blockLevel = 0
						openTags = append(openTags, tag)
						break
					}
				}

				toClose = append(toClose, tag)
				if tag.tag == lookFor {
					break
				}
			}

			// Did we just eat through everything and not find it?
			if len(openTags) == 0 && (tag == nil || tag.tag != lookFor) {
				// Restore: $open_tags = $to_close (note: order is reversed
				// from the original stack, exactly as PHP leaves it).
				openTags = append([]*bbcOpen{}, toClose...)
				continue
			} else if len(toClose) > 0 && tag != nil && tag.tag != lookFor {
				if blockLevel == -1 && len(lookFor) > 0 {
					if codes, ok := bbcIndex[lookFor[0]]; ok {
						for _, temp := range codes {
							if temp.tag == lookFor {
								if temp.blockLevel {
									blockLevel = 1
								} else {
									blockLevel = 0
								}
								break
							}
						}
					}
				}

				// We're not looking for a block level tag (or maybe even a
				// tag that exists...)
				if blockLevel != 1 {
					for _, t := range toClose {
						openTags = append(openTags, t)
					}
					continue
				}
			}

			for _, t := range toClose {
				message = message[:pos] + t.after + substr(message, pos2+1, -1)
				pos += len(t.after)
				pos2 = pos - 1

				// See the comment at the end of the big loop - just eating
				// whitespace ;).
				if t.blockLevel && substr(message, pos, 6) == "<br />" {
					message = message[:pos] + substr(message, pos+6, -1)
				}
				if t.trim != "" && t.trim != "inside" {
					if m := trimWhitespaceRe.FindString(substr(message, pos, -1)); m != "" {
						message = message[:pos] + substr(message, pos+len(m), -1)
					}
				}
			}

			if len(toClose) > 0 {
				pos--
			}

			continue
		}

		// No tags for this character, so just keep going (fastest possible
		// course.)
		codes, ok := bbcIndex[tagChar]
		if !ok {
			continue
		}

		var inside *bbcOpen
		if len(openTags) > 0 {
			inside = openTags[len(openTags)-1]
		}
		var tag *bbcOpen
		var tagDef *bbcCode
		pos1 := 0
		for i := range codes {
			possible := &codes[i]

			// Not a match?
			if strings.ToLower(substr(message, pos+1, len(possible.tag))) != possible.tag {
				continue
			}

			nextC := substr(message, pos+1+len(possible.tag), 1)

			// PHP's if/elseif chain: a passing 'test' skips all other
			// next-character checks; parameters need a space; otherwise the
			// type dictates what must follow.
			if possible.test != "" {
				// A test validation?
				re := getCachedRe("^" + possible.test)
				if re == nil || !re.MatchString(substr(message, pos+1+len(possible.tag)+1, -1)) {
					continue
				}
			} else if len(possible.parameters) > 0 {
				// Do we want parameters?
				if nextC != " " {
					continue
				}
			} else if possible.typ != typeParsedContent {
				// Do we need an equal sign?
				switch possible.typ {
				case typeUnparsedEquals, typeUnparsedCommas, typeUnparsedCommasContent,
					typeUnparsedEqualsContent, typeParsedEquals:
					if nextC != "=" {
						continue
					}
				}
				// Maybe we just want a /...
				if possible.typ == typeClosed && nextC != "]" &&
					substr(message, pos+1+len(possible.tag), 2) != "/]" &&
					substr(message, pos+1+len(possible.tag), 3) != " /]" {
					continue
				}
				// An immediate ]?
				if possible.typ == typeUnparsedContent && nextC != "]" {
					continue
				}
			} else if nextC != "]" {
				// No type means 'parsed_content', which demands an immediate
				// ] without parameters!
				continue
			}

			// Check allowed tree?
			if possible.requireParents != nil && (inside == nil || !inList(inside.tag, possible.requireParents)) {
				continue
			} else if inside != nil && inside.requireChildren != nil && !inList(possible.tag, inside.requireChildren) {
				continue
			} else if inside != nil && inside.disallowChildren != nil && inList(possible.tag, inside.disallowChildren) {
				continue
			}

			pos1 = pos + 1 + len(possible.tag) + 1

			inst := &bbcOpen{
				tag: possible.tag, typ: possible.typ,
				content: possible.content, before: possible.before, after: possible.after,
				blockLevel: possible.blockLevel, trim: possible.trim,
				requireChildren: possible.requireChildren, disallowChildren: possible.disallowChildren,
			}

			// This is long, but it makes things much easier and cleaner.
			if len(possible.parameters) > 0 {
				matched, params, matchLen := c.matchBBCParams(possible, substr(message, pos1-1, -1))
				if !matched {
					continue
				}
				// Put the parameters into the string.
				inst.before = strtrMap(inst.before, params)
				inst.after = strtrMap(inst.after, params)
				inst.content = strtrMap(inst.content, params)
				pos1 += matchLen - 1
			}
			tag = inst
			tagDef = possible
			break
		}

		// Item codes are complicated buggers... they are implicit [li]s and
		// can make [list]s! ($smileys !== false — 'print' mode also counts.)
		if (smileys || printMode) && tag == nil && itemcodesEnabled && pos+2 < len(message) {
			if itemType, isItem := bbcItemcodes[message[pos+1]]; isItem && substr(message, pos+2, 1) == "]" {
				if message[pos+1] == '0' && !(pos > 0 && (message[pos-1] == ';' || message[pos-1] == ' ' || message[pos-1] == '\t' || message[pos-1] == '>')) {
					continue
				}

				// First let's set up the tree: it needs to be in a list, or
				// after an li.
				var code string
				if inside == nil || (inside.tag != "list" && inside.tag != "li") {
					var disallow []string
					if inside != nil {
						disallow = inside.disallowChildren
					}
					openTags = append(openTags, &bbcOpen{
						tag: "list", after: "</ul>", blockLevel: true,
						requireChildren: []string{"li"}, disallowChildren: disallow,
					})
					code = `<ul style="margin-top: 0; margin-bottom: 0;">`
				} else if inside.tag == "li" {
					// We're in a list item already: another itemcode? Close
					// it first.
					openTags = openTags[:len(openTags)-1]
					code = "</li>"
				}

				// Now we open a new tag.
				var disallow []string
				if inside != nil {
					disallow = inside.disallowChildren
				}
				openTags = append(openTags, &bbcOpen{
					tag: "li", after: "</li>", trim: "outside", blockLevel: true,
					disallowChildren: disallow,
				})

				// First, open the tag...
				if itemType == "" {
					code += "<li>"
				} else {
					code += `<li type="` + itemType + `">`
				}
				message = message[:pos] + code + substr(message, pos+3, -1)
				pos += len(code) - 1

				// Next, find the next break (if any.)  If there's more
				// itemcode after it, keep it going - otherwise close!
				pos2 := stripos2(message, "<br />", pos)
				pos3 := stripos2(message, "[/", pos)
				if pos2 >= 0 && (pos3 < 0 || pos2 <= pos3) {
					m := itemcodeContinueRe.FindString(substr(message, pos2+6, -1))
					insert := "[/li][/list]"
					if m != "" && strings.HasSuffix(m, "[") {
						insert = "[/li]"
					}
					message = message[:pos2] + insert + substr(message, pos2, -1)

					openTags[len(openTags)-2].after = "</ul>"
				} else {
					// Tell the [list] that it needs to close specially.
					// Move the li over, because we're not sure what we'll hit.
					openTags[len(openTags)-1].after = ""
					openTags[len(openTags)-2].after = "</li></ul>"
				}

				continue
			}
		}

		// Implicitly close lists and tables if something other than what's
		// required is in them. This is needed for itemcode.
		if tag == nil && inside != nil && inside.requireChildren != nil {
			openTags = openTags[:len(openTags)-1]

			message = message[:pos] + inside.after + substr(message, pos, -1)
			pos += len(inside.after) - 1
		}

		// No tag?  Keep looking, then.  Silly people using brackets without
		// actual tags.
		if tag == nil {
			continue
		}

		// Propagate the list to the child (so wrapping the disallowed tag
		// won't work either.)
		if inside != nil && inside.disallowChildren != nil {
			tag.disallowChildren = uniqueStrings(append(append([]string{}, tag.disallowChildren...), inside.disallowChildren...))
		}

		// Is this tag disabled?
		if disabled[tag.tag] {
			if !tagDef.hasDisBefore && !tagDef.hasDisAfter && !tagDef.hasDisContent {
				if tag.blockLevel {
					tag.before, tag.after = "<div>", "</div>"
				} else {
					tag.before, tag.after = "", ""
				}
				if tag.typ == typeClosed {
					tag.content = ""
				} else if tag.blockLevel {
					tag.content = "<div>$1</div>"
				} else {
					tag.content = "$1"
				}
			} else if tagDef.hasDisBefore || tagDef.hasDisAfter {
				if tagDef.hasDisBefore {
					tag.before = tagDef.disabledBefore
				} else if tag.blockLevel {
					tag.before = "<div>"
				} else {
					tag.before = ""
				}
				if tagDef.hasDisAfter {
					tag.after = tagDef.disabledAfter
				} else if tag.blockLevel {
					tag.after = "</div>"
				} else {
					tag.after = ""
				}
			} else {
				tag.content = tagDef.disabledContent
			}
		}

		// The only special case is 'html', which doesn't need to close
		// things.
		if tag.blockLevel && tag.tag != "html" && (inside == nil || !inside.blockLevel) {
			n := len(openTags) - 1
			for n >= 0 && !openTags[n].blockLevel {
				n--
			}

			// Close all the non block level tags so this tag isn't
			// surrounded by them.
			for i := len(openTags) - 1; i > n; i-- {
				message = message[:pos] + openTags[i].after + substr(message, pos, -1)
				pos += len(openTags[i].after)
				pos1 += len(openTags[i].after)

				// Trim or eat trailing stuff...
				if openTags[i].blockLevel && substr(message, pos, 6) == "<br />" {
					message = message[:pos] + substr(message, pos+6, -1)
				}
				if openTags[i].trim != "" && tag.trim != "inside" {
					if m := trimWhitespaceRe.FindString(substr(message, pos, -1)); m != "" {
						message = message[:pos] + substr(message, pos+len(m), -1)
					}
				}

				openTags = openTags[:i]
			}
		}

		switch tag.typ {
		case typeParsedContent:
			// No type means 'parsed_content'.
			openTags = append(openTags, tag)
			message = message[:pos] + tag.before + substr(message, pos1, -1)
			pos += len(tag.before) - 1

		case typeUnparsedContent:
			// Don't parse the content, just skip it.
			pos2 := stripos(message, "[/"+substr(message, pos+1, len(tag.tag))+"]", pos1)
			if pos2 < 0 {
				continue
			}

			data := &bbcData{s: substr(message, pos1, pos2-pos1)}

			if tag.blockLevel && strings.HasPrefix(data.s, "<br />") {
				data.s = data.s[6:]
			}

			if tagDef.validate != nil {
				tagDef.validate(c, tag, data)
			}

			code := strings.ReplaceAll(tag.content, "$1", data.s)
			message = message[:pos] + code + substr(message, pos2+3+len(tag.tag), -1)
			pos += len(code) - 1

		case typeUnparsedEqualsContent:
			// The value may be quoted for some tags - check.
			quoted := false
			if tagDef.quoted != "" {
				quoted = substr(message, pos1, 6) == "&quot;"
				if tagDef.quoted != "optional" && !quoted {
					continue
				}
				if quoted {
					pos1 += 6
				}
			}

			endMark := "]"
			if quoted {
				endMark = "&quot;]"
			}
			pos2 := stripos2(message, endMark, pos1)
			if pos2 < 0 {
				continue
			}
			pos3 := stripos(message, "[/"+substr(message, pos+1, len(tag.tag))+"]", pos2)
			if pos3 < 0 {
				continue
			}

			off := 1
			if quoted {
				off = 7
			}
			data := &bbcData{list: []string{
				substr(message, pos2+off, pos3-(pos2+off)),
				substr(message, pos1, pos2-pos1),
			}}

			if tag.blockLevel && strings.HasPrefix(data.list[0], "<br />") {
				data.list[0] = data.list[0][6:]
			}

			// Validation for my parking, please!
			if tagDef.validate != nil {
				tagDef.validate(c, tag, data)
			}

			code := strings.NewReplacer("$1", data.list[0], "$2", data.list[1]).Replace(tag.content)
			message = message[:pos] + code + substr(message, pos3+3+len(tag.tag), -1)
			pos += len(code) - 1

		case typeClosed:
			// A closed tag, with no content or value.
			pos2 := stripos2(message, "]", pos)
			message = message[:pos] + tag.content + substr(message, pos2+1, -1)
			pos += len(tag.content) - 1

		case typeUnparsedCommasContent:
			// This one is sorta ugly... :/. Unfortunately, it's needed for
			// flash.
			pos2 := stripos2(message, "]", pos1)
			if pos2 < 0 {
				continue
			}
			pos3 := stripos(message, "[/"+substr(message, pos+1, len(tag.tag))+"]", pos2)
			if pos3 < 0 {
				continue
			}

			// We want $1 to be the content, and the rest to be csv.
			data := &bbcData{list: append([]string{""}, strings.Split(substr(message, pos1, pos2-pos1), ",")...)}
			data.list[0] = substr(message, pos2+1, pos3-pos2-1)

			if tagDef.validate != nil {
				tagDef.validate(c, tag, data)
			}

			code := tag.content
			for k, d := range data.list {
				code = strings.ReplaceAll(code, "$"+itoa(k+1), strings.TrimSpace(d))
			}
			message = message[:pos] + code + substr(message, pos3+3+len(tag.tag), -1)
			pos += len(code) - 1

		case typeUnparsedCommas:
			pos2 := stripos2(message, "]", pos1)
			if pos2 < 0 {
				continue
			}

			data := &bbcData{list: strings.Split(substr(message, pos1, pos2-pos1), ",")}

			if tagDef.validate != nil {
				tagDef.validate(c, tag, data)
			}

			// Fix after, for disabled code mainly.
			for k, d := range data.list {
				tag.after = strings.ReplaceAll(tag.after, "$"+itoa(k+1), strings.TrimSpace(d))
			}

			openTags = append(openTags, tag)

			// Replace them out, $1, $2, $3, $4, etc.
			code := tag.before
			for k, d := range data.list {
				code = strings.ReplaceAll(code, "$"+itoa(k+1), strings.TrimSpace(d))
			}
			message = message[:pos] + code + substr(message, pos2+1, -1)
			pos += len(code) - 1

		case typeUnparsedEquals, typeParsedEquals:
			// The value may be quoted for some tags - check.
			quoted := false
			if tagDef.quoted != "" {
				quoted = substr(message, pos1, 6) == "&quot;"
				if tagDef.quoted != "optional" && !quoted {
					continue
				}
				if quoted {
					pos1 += 6
				}
			}

			endMark := "]"
			if quoted {
				endMark = "&quot;]"
			}
			pos2 := stripos2(message, endMark, pos1)
			if pos2 < 0 {
				continue
			}

			data := &bbcData{s: substr(message, pos1, pos2-pos1)}

			// Validation for my parking, please!
			if tagDef.validate != nil {
				tagDef.validate(c, tag, data)
			}

			// For parsed content, we must recurse to avoid security problems.
			if tag.typ != typeUnparsedEquals {
				data.s = c.parseBBC(data.s, true)
			}

			tag.after = strings.ReplaceAll(tag.after, "$1", data.s)

			openTags = append(openTags, tag)

			code := strings.ReplaceAll(tag.before, "$1", data.s)
			off := 1
			if quoted {
				off = 7
			}
			message = message[:pos] + code + substr(message, pos2+off, -1)
			pos += len(code) - 1
		}

		// If this is block level, eat any breaks after it.
		if tag.blockLevel && substr(message, pos+1, 6) == "<br />" {
			message = message[:pos+1] + substr(message, pos+7, -1)
		}

		// Are we trimming outside this tag?
		if tag.trim != "" && tag.trim != "outside" {
			if m := trimWhitespaceRe.FindString(substr(message, pos+1, -1)); m != "" {
				message = message[:pos+1] + substr(message, pos+1+len(m), -1)
			}
		}
	}

	// Close any remaining tags.
	for len(openTags) > 0 {
		tag := openTags[len(openTags)-1]
		openTags = openTags[:len(openTags)-1]
		message += tag.after
	}

	if strings.HasPrefix(message, " ") {
		message = "&nbsp;" + message[1:]
	}

	// Cleanup whitespace.
	message = strings.NewReplacer("  ", " &nbsp;", "\r", "", "\n", "<br />", "<br /> ", "<br />&nbsp;", "&#13;", "\n").Replace(message)

	return message
}

var trimWhitespaceRe = regexp.MustCompile(`^(<br />|&nbsp;|\s)*`)
var itemcodeContinueRe = regexp.MustCompile(`^(<br />|&nbsp;|\s|\[)+`)

// stripos2 is strpos with offset (case sensitive), -1 when not found.
func stripos2(haystack, needle string, offset int) int {
	if offset > len(haystack) {
		return -1
	}
	i := strings.Index(haystack[offset:], needle)
	if i < 0 {
		return -1
	}
	return offset + i
}

func lcByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// strtrMap is strtr($s, $params) for the {param} replacements.
func strtrMap(s string, params map[string]string) string {
	if len(params) == 0 || s == "" {
		return s
	}
	pairs := make([]string, 0, len(params)*2)
	for k, v := range params {
		pairs = append(pairs, k, v)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// matchBBCParams implements the parameter-permutation matching: tries every
// order of the parameter regexes against the text starting at the byte
// before ']' would close an unparameterized tag. Returns the {param} map and
// the length of the consumed match.
func (c *Ctx) matchBBCParams(possible *bbcCode, text string) (bool, map[string]string, int) {
	n := len(possible.parameters)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}

	var try func(order []int) (bool, map[string]string, int)
	try = func(order []int) (bool, map[string]string, int) {
		pattern := "(?i)^"
		for _, pi := range order {
			p := possible.parameters[pi]
			match := p.match
			if match == "" {
				match = "(.+?)"
			}
			quote := ""
			if p.quoted {
				quote = "&quot;"
			}
			pattern += `(\s+` + p.name + `=` + quote + match + quote + `)`
			if p.optional {
				pattern += "?"
			}
		}
		pattern += `\]`
		re := getCachedRe(pattern)
		if re == nil {
			return false, nil, 0
		}
		m := re.FindStringSubmatch(text)
		if m == nil {
			return false, nil, 0
		}
		params := map[string]string{}
		// Submatches come in (outer, inner) pairs per parameter in order.
		for i := 0; i < len(order); i++ {
			outer := m[1+i*2]
			inner := m[2+i*2]
			if outer == "" {
				continue
			}
			key := strings.SplitN(strings.TrimLeft(outer, " \t\n\r"), "=", 2)[0]
			var p *bbcParam
			for j := range possible.parameters {
				if possible.parameters[j].name == key {
					p = &possible.parameters[j]
					break
				}
			}
			if p == nil {
				continue
			}
			var val string
			switch {
			case p.value != "":
				val = strings.ReplaceAll(p.value, "$1", inner)
			case p.validate != nil:
				val = p.validate(c, inner)
			default:
				val = inner
			}
			// Just to make sure: replace any $ or { so they can't
			// interpolate wrongly.
			val = strings.NewReplacer("$", "&#036;", "{", "&#123;").Replace(val)
			params["{"+key+"}"] = val
		}
		for _, p := range possible.parameters {
			if _, ok := params["{"+p.name+"}"]; !ok {
				params["{"+p.name+"}"] = ""
			}
		}
		return true, params, len(m[0])
	}

	// Permute orders, first the identity (most common case first, like
	// PHP's permute() which returns the original order first).
	ok, params, length := try(idx)
	if ok {
		return true, params, length
	}
	perms := permuteInts(idx)
	for _, order := range perms {
		ok, params, length := try(order)
		if ok {
			return true, params, length
		}
	}
	return false, nil, 0
}

// permuteInts returns all permutations of v (n! of them; n <= 3 here).
func permuteInts(v []int) [][]int {
	if len(v) <= 1 {
		return [][]int{append([]int{}, v...)}
	}
	var out [][]int
	for i := range v {
		rest := make([]int, 0, len(v)-1)
		rest = append(rest, v[:i]...)
		rest = append(rest, v[i+1:]...)
		for _, p := range permuteInts(rest) {
			out = append(out, append([]int{v[i]}, p...))
		}
	}
	return out
}

// reCache caches compiled regexes for tests/parameters.
var reCacheMap = map[string]*regexp.Regexp{}
var reCacheMu chMutex

type chMutex struct{ ch chan struct{} }

func (m *chMutex) Lock() {
	if m.ch == nil {
		m.ch = make(chan struct{}, 1)
	}
	m.ch <- struct{}{}
}
func (m *chMutex) Unlock() { <-m.ch }

func getCachedRe(pattern string) *regexp.Regexp {
	reCacheMu.Lock()
	defer reCacheMu.Unlock()
	if re, ok := reCacheMap[pattern]; ok {
		return re
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		reCacheMap[pattern] = nil
		return nil
	}
	reCacheMap[pattern] = re
	return re
}
