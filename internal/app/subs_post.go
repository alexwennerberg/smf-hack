package app

// Port of Sources/Subs-Post.php (Phase 3 scope: preparsecode,
// un_preparsecode, fixTags/fixTag, createPost, modifyPost,
// updateLastMessages, trackStats counters; createAttachment ports with the
// attachment pipeline, sendNotifications below).

import (
	"regexp"
	"strings"
	"time"
)

var codeSplitRe = regexp.MustCompile(`(?i)(\[/code\]|\[code(?:=[^\]]+)?\])`)

// splitDelimCapture is preg_split with PREG_SPLIT_DELIM_CAPTURE: parts
// alternate text, delim, text, delim, ... like PHP's 0=outside 1=open
// 2=inside 3=close cycle.
func splitDelimCapture(re *regexp.Regexp, s string) []string {
	var parts []string
	last := 0
	for _, loc := range re.FindAllStringIndex(s, -1) {
		parts = append(parts, s[last:loc[0]], s[loc[0]:loc[1]])
		last = loc[1]
	}
	parts = append(parts, s[last:])
	return parts
}

var nobbcRe = regexp.MustCompile(`(?is)\[nobbc\](.+?)\[/nobbc\]`)
var periodsRe = regexp.MustCompile(`\.{100,}`)
var codeOpenRe = regexp.MustCompile(`(?is)(\[code(?:=[^\]]+)?\])`)
var codeCloseRe = regexp.MustCompile(`(?is)(\[/code\])`)
var meLineRe = regexp.MustCompile(`(?i)(\A|\n)/me(?: |&nbsp;)([^\n]*)`)
var htmlTagRe = regexp.MustCompile(`(?is)\[html\](.+?)\[/html\]`)
var htmlStripRe = regexp.MustCompile(`(?i)\[/?html\]`)
var timeTagRe = regexp.MustCompile(`(?is)\[time(=(absolute))?\](.+?)\[/time\]`)
var lowerTagRe = regexp.MustCompile(`(?i)\[(/?)(list|li|table|tr|td)((\s[^\]]+)*)\]`)

// preparsecode is preparsecode(&$message, $previewing): normalize a
// just-posted message body for storage.
func (c *Ctx) preparsecode(message string, previewing bool) string {
	// Clean up after nobbc ;).
	message = nobbcRe.ReplaceAllStringFunc(message, func(m string) string {
		inner := nobbcRe.FindStringSubmatch(m)[1]
		return "[nobbc]" + strings.NewReplacer("[", "&#91;", "]", "&#93;", ":", "&#58;", "@", "&#64;").Replace(inner) + "[/nobbc]"
	})

	// Remove \r's... they're evil!
	message = strings.ReplaceAll(message, "\r", "")

	// You won't believe this - but too many periods upsets apache it seems!
	message = periodsRe.ReplaceAllString(message, "...")

	// Trim off trailing quotes - these often happen by accident.
	for strings.HasSuffix(message, "[quote]") {
		message = message[:len(message)-7]
	}
	for strings.HasPrefix(message, "[/quote]") {
		message = message[8:]
	}

	// Check if all code tags are closed.
	codeopen := len(codeOpenRe.FindAllString(message, -1))
	codeclose := len(codeCloseRe.FindAllString(message, -1))
	if codeopen > codeclose {
		message += strings.Repeat("[/code]", codeopen-codeclose)
	} else if codeclose > codeopen {
		message = strings.Repeat("[code]", codeclose-codeopen) + message
	}

	// Only mess with stuff outside [code] tags.
	parts := splitDelimCapture(codeSplitRe, message)
	for i := range parts {
		if i%4 != 0 {
			continue
		}
		parts[i] = c.fixTags(parts[i])

		// Replace /me.+?\n with [me=name]...[/me].
		name := c.User.Name
		meName := name
		if strings.ContainsAny(name, `[]'"`) {
			meName = "&quot;" + name + "&quot;"
		}
		parts[i] = meLineRe.ReplaceAllString(parts[i], "${1}[me="+meName+"]$2[/me]")

		if !previewing && strings.Contains(parts[i], "[html]") {
			if c.allowedTo("admin_forum") {
				parts[i] = htmlTagRe.ReplaceAllStringFunc(parts[i], func(m string) string {
					inner := htmlTagRe.FindStringSubmatch(m)[1]
					return "[html]" + strings.NewReplacer("\n", "&#13;", "  ", " &#32;").Replace(unHtmlspecialchars(inner)) + "[/html]"
				})
			} else {
				// We should edit them out...
				for strings.Contains(parts[i], "[html]") {
					parts[i] = htmlStripRe.ReplaceAllString(parts[i], "")
				}
			}
		}

		// Let's look at the time tags...
		parts[i] = timeTagRe.ReplaceAllStringFunc(parts[i], func(m string) string {
			sub := timeTagRe.FindStringSubmatch(m)
			value := sub[3]
			if isAllDigits(strings.TrimSpace(value)) {
				return "[time]" + value + "[/time]"
			}
			// PHP strtotime on a free-form string: keep the literal if we
			// can't parse it (strtotime==0 path).
			if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
				ts := t.Unix()
				if sub[2] != "absolute" {
					ts -= int64((float64(c.App.SettingInt("time_offset")) + c.User.TimeOffset) * 3600)
				}
				return "[time]" + itoa(int(ts)) + "[/time]"
			}
			return "[time]" + value + "[/time]"
		})

		listOpen := strings.Count(parts[i], "[list]") + strings.Count(parts[i], "[list ")
		listClose := strings.Count(parts[i], "[/list]")
		if listClose-listOpen > 0 {
			parts[i] = strings.Repeat("[list]", listClose-listOpen) + parts[i]
		}
		if listOpen-listClose > 0 {
			parts[i] = parts[i] + strings.Repeat("[/list]", listOpen-listClose)
		}

		// Make sure all tags are lowercase.
		parts[i] = lowerTagRe.ReplaceAllStringFunc(parts[i], func(m string) string {
			sub := lowerTagRe.FindStringSubmatch(m)
			return "[" + sub[1] + strings.ToLower(sub[2]) + sub[3] + "]"
		})

		// Fix up some use of tables without [tr]s, etc.
		for j := 0; j < 3; j++ {
			parts[i] = applyMistakeFixes(parts[i])
		}
	}

	message = strings.Join(parts, "")
	if !previewing {
		message = strings.NewReplacer("  ", "&nbsp; ", "\n", "<br />", "\xA0", "&nbsp;").Replace(message)
	} else {
		message = strings.NewReplacer("  ", "&nbsp; ", "\xA0", "&nbsp;").Replace(message)
	}

	// Now let's quickly clean up things that will slow our parser.
	message = strings.NewReplacer("[]", "&#91;]", "[&#039;", "&#91;&#039;").Replace(message)
	return message
}

// applyMistakeFixes ports the $mistake_fixes regex pipeline. RE2 lacks
// lookaheads, so the "not followed by" patterns are emulated by capturing
// the optional follower.
func applyMistakeFixes(s string) string {
	// Go regexps require valid UTF-8, so widen the ISO-8859-1 bytes to runes
	// for the pipeline's duration (the patterns only insert ASCII).
	s = latin1ToRunes(s)
	sp := `[\s\x{00A0}]*`

	// Find [table]s not followed by [tr].
	s = replaceNotFollowedBy(s, `\[table\]`, sp+`\[tr\]`, "[table][tr]")
	// Find [tr]s not followed by [td].
	s = replaceNotFollowedBy(s, `\[tr\]`, sp+`\[td\]`, "[tr][td]")
	// Find [/td]s not followed by something valid.
	s = replaceNotFollowedBy(s, `\[/td\]`, sp+`(?:\[td\]|\[/tr\]|\[/table\])`, "[/td][/tr]")
	// Find [/tr]s not followed by something valid.
	s = replaceNotFollowedBy(s, `\[/tr\]`, sp+`(?:\[tr\]|\[/table\])`, "[/tr][/table]")
	// Find [/td]s incorrectly followed by [/table].
	s = regexp.MustCompile(`(?s)\[/td\]`+sp+`\[/table\]`).ReplaceAllStringFunc(s, func(m string) string {
		mid := m[len("[/td]") : len(m)-len("[/table]")]
		return "[/td][/tr]" + mid + "[/table]"
	})
	// Find [table]s, [tr]s, and [/td]s (possibly correctly) followed by [td].
	s = regexp.MustCompile(`(?s)\[(table|tr|/td)\](`+sp+`)\[td\]`).ReplaceAllString(s, "[$1]$2[_td_]")
	// Now, any [td]s left should have a [tr] before them.
	s = strings.ReplaceAll(s, "[td]", "[tr][td]")
	// Look for [tr]s which are correctly placed.
	s = regexp.MustCompile(`(?s)\[(table|/tr)\](`+sp+`)\[tr\]`).ReplaceAllString(s, "[$1]$2[_tr_]")
	// Any remaining [tr]s should have a [table] before them.
	s = strings.ReplaceAll(s, "[tr]", "[table][tr]")
	// Look for [/td]s followed by [/tr].
	s = regexp.MustCompile(`(?s)\[/td\](`+sp+`)\[/tr\]`).ReplaceAllString(s, "[/td]$1[_/tr_]")
	// Any remaining [/tr]s should have a [/td].
	s = strings.ReplaceAll(s, "[/tr]", "[/td][/tr]")
	// Look for properly opened [li]s which aren't closed.
	s = regexp.MustCompile(`(?s)\[li\]([^\[\]]+?)\[li\]`).ReplaceAllString(s, "[li]$1[_/li_][_li_]")
	s = regexp.MustCompile(`(?s)\[li\]([^\[\]]+?)$`).ReplaceAllString(s, "[li]$1[/li]")
	// Lists - find correctly closed items/lists.
	s = regexp.MustCompile(`(?s)\[/li\](`+sp+`)\[/list\]`).ReplaceAllString(s, "[_/li_]$1[/list]")
	// Find list items closed and then opened.
	s = regexp.MustCompile(`(?s)\[/li\](`+sp+`)\[li\]`).ReplaceAllString(s, "[_/li_]$1[_li_]")
	// Now, find any [list]s or [/li]s followed by [li].
	s = regexp.MustCompile(`(?s)\[(list(?: [^\]]*?)?|/li)\](`+sp+`)\[li\]`).ReplaceAllString(s, "[$1]$2[_li_]")
	// Any remaining [li]s weren't inside a [list].
	s = strings.ReplaceAll(s, "[li]", "[list][li]")
	// Any remaining [/li]s weren't before a [/list].
	s = strings.ReplaceAll(s, "[/li]", "[/li][/list]")
	// Put the correct ones back how we found them.
	s = regexp.MustCompile(`\[_(li|/li|td|tr|/tr)_\]`).ReplaceAllString(s, "[$1]")
	return runesToLatin1(s)
}

// replaceNotFollowedBy replaces every match of token NOT followed by
// follower with repl (lookahead emulation: matches token+optional follower,
// and only rewrites when the follower is absent).
func replaceNotFollowedBy(s, token, follower, repl string) string {
	re := regexp.MustCompile(`(?s)(` + token + `)(` + follower + `)?`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if sub[2] != "" {
			return m // follower present: untouched
		}
		return repl
	})
}

// unPreparsecode is un_preparsecode($message).
func (c *Ctx) unPreparsecode(message string) string {
	parts := splitDelimCapture(codeSplitRe, message)
	for i := range parts {
		if i%4 != 0 {
			continue
		}
		parts[i] = htmlTagRe.ReplaceAllStringFunc(parts[i], func(m string) string {
			inner := htmlTagRe.FindStringSubmatch(m)[1]
			return "[html]" + strings.NewReplacer("&amp;#13;", "<br />", "&amp;#32;", " ").Replace(Htmlspecialchars(inner)) + "[/html]"
		})
		// Attempt to un-parse the time to something less awful.
		parts[i] = regexp.MustCompile(`(?i)\[time\](\d{0,10})\[/time\]`).ReplaceAllStringFunc(parts[i], func(m string) string {
			ts := regexp.MustCompile(`\d{0,10}`).FindString(m[6:])
			return "[time]" + c.timeformatFmt(int64(atoi(ts)), false, "%c") + "[/time]"
		})
	}
	out := strings.ReplaceAll(strings.Join(parts, ""), "&nbsp;", " ")
	return regexp.MustCompile(`<br( /)?>`).ReplaceAllString(out, "\n")
}

// fixTags is fixTags(&$message).
func (c *Ctx) fixTags(message string) string {
	type fixParam struct {
		tag          string
		protocols    []string
		embeddedURL  bool
		hasEqualSign bool
		hasExtra     bool
	}
	fixArray := []fixParam{
		{"img", []string{"http", "https"}, false, false, true},
		{"url", []string{"http", "https"}, true, false, false},
		{"url", []string{"http", "https"}, true, true, false},
		{"iurl", []string{"http", "https"}, true, false, false},
		{"iurl", []string{"http", "https"}, true, true, false},
		{"ftp", []string{"ftp", "ftps"}, true, false, false},
		{"ftp", []string{"ftp", "ftps"}, true, true, false},
		{"flash", []string{"http", "https"}, false, false, true},
	}
	for _, p := range fixArray {
		message = c.fixTag(message, p.tag, p.protocols, p.embeddedURL, p.hasEqualSign, p.hasExtra)
	}

	// Now fix possible security problems with images loading links
	// automatically...
	imgRe := regexp.MustCompile(`(?is)(\[img.*?\])(.+?)\[/img\]`)
	message = imgRe.ReplaceAllStringFunc(message, func(m string) string {
		sub := imgRe.FindStringSubmatch(m)
		return sub[1] + stripActionParam(sub[2]) + "[/img]"
	})

	// Limit the size of images posted? Only when an admin set a cap; with
	// max_image_width/max_image_height at their 0 defaults PHP skips this.
	maxW := c.App.SettingInt("max_image_width")
	maxH := c.App.SettingInt("max_image_height")
	if maxW != 0 || maxH != 0 {
		message = c.resizePostedImages(message, maxW, maxH)
	}

	return message
}

var imgSizeRe = regexp.MustCompile(`(?is)\[img(\s+width=\d+)?(\s+height=\d+)?(\s+width=\d+)?\](.+?)\[/img\]`)

// resizePostedImages ports the preparsecode max_image_width/height block:
// for each [img] (with or without width/height), clamp the dimensions to the
// admin caps, fetching the real size via urlImageSize when one or both are
// omitted, then rewrite the tag as [img width=.. height=..].
func (c *Ctx) resizePostedImages(message string, maxW, maxH int) string {
	replaces := map[string]string{}
	for _, m := range imgSizeRe.FindAllStringSubmatch(message, -1) {
		full, w1, w3, hStr, src := m[0], m[1], m[3], m[2], m[4]
		// If the width was after the height, handle it.
		if w3 != "" {
			w1 = w3
		}
		// substr(trim, 6) past "width=", substr(trim, 7) past "height=".
		desiredW := 0
		if w1 != "" {
			desiredW = atoi(strings.TrimSpace(w1)[6:])
		}
		desiredH := 0
		if hStr != "" {
			desiredH = atoi(strings.TrimSpace(hStr)[7:])
		}

		// One was omitted, or both. We'll have to find its real size...
		if desiredW == 0 || desiredH == 0 {
			width, height, _ := urlImageSize(unHtmlspecialchars(src))
			switch {
			case desiredW == 0 && desiredH == 0:
				desiredW, desiredH = width, height
			case desiredW == 0 && height != 0:
				desiredW = desiredH * width / height
			case width != 0:
				desiredH = desiredW * height / width
			}
		}

		// If the width and height are fine, just continue along...
		if desiredW <= maxW && desiredH <= maxH {
			continue
		}
		// Too bad, it's too wide. Make it as wide as the maximum.
		if desiredW > maxW && maxW != 0 {
			desiredH = maxW * desiredH / desiredW
			desiredW = maxW
		}
		// Now check the height, as well. Might have to scale twice, even...
		if desiredH > maxH && maxH != 0 {
			desiredW = maxH * desiredW / desiredH
			desiredH = maxH
		}

		repl := "[img"
		if desiredW != 0 {
			repl += " width=" + itoa(desiredW)
		}
		if desiredH != 0 {
			repl += " height=" + itoa(desiredH)
		}
		repl += "]" + src + "[/img]"
		replaces[full] = repl
	}

	if len(replaces) > 0 {
		pairs := make([]string, 0, len(replaces)*2)
		for k, v := range replaces {
			pairs = append(pairs, k, v)
		}
		message = strings.NewReplacer(pairs...).Replace(message)
	}
	return message
}

// fixTag is fixTag(&$message, ...): normalize the URL in one tag type.
func (c *Ctx) fixTag(message, myTag string, protocols []string, embeddedURL, hasEqualSign, hasExtra bool) string {
	domainURL := c.App.Config.BoardURL + "/"
	if m := regexp.MustCompile(`^([^:]+://[^/]+)`).FindStringSubmatch(c.App.Config.BoardURL); m != nil {
		domainURL = m[1]
	}
	scripturl := c.App.ScriptURL

	var re *regexp.Regexp
	if hasEqualSign {
		re = regexp.MustCompile(`(?is)\[(` + myTag + `)=([^\]]*?)\](?:(.+?)\[/(` + myTag + `)\])?`)
	} else {
		extra := ""
		if hasExtra {
			extra = `(?:[^\]]*?)`
		}
		re = regexp.MustCompile(`(?is)\[(` + myTag + extra + `)\](.+?)\[/(` + myTag + `)\]`)
	}

	replaces := map[string]string{}
	for _, sub := range re.FindAllStringSubmatch(message, -1) {
		replace := strings.TrimSpace(sub[2])
		thisTag := sub[1]
		var thisClose string
		if hasEqualSign {
			if len(sub) > 4 && sub[4] != "" {
				thisClose = sub[4]
			}
		} else {
			thisClose = sub[3]
		}

		found := false
		for _, protocol := range protocols {
			if len(replace) >= len(protocol)+3 && strings.EqualFold(replace[:len(protocol)+3], protocol+"://") {
				found = true
				break
			}
		}

		if !found && protocols[0] == "http" {
			if strings.HasPrefix(replace, "/") {
				replace = domainURL + replace
			} else if strings.HasPrefix(replace, "?") {
				replace = scripturl + replace
			} else if strings.HasPrefix(replace, "#") && embeddedURL {
				replace = "#" + regexp.MustCompile(`[^A-Za-z0-9_\-#]`).ReplaceAllString(replace[1:], "")
				thisTag = "iurl"
				thisClose = "iurl"
			} else {
				replace = protocols[0] + "://" + replace
			}
		} else if !found {
			replace = protocols[0] + "://" + replace
		}

		if hasEqualSign && embeddedURL {
			repl := "[" + thisTag + "=" + replace + "]"
			if len(sub) > 4 && sub[4] != "" {
				repl += sub[3] + "[/" + thisClose + "]"
			}
			replaces[sub[0]] = repl
		} else if hasEqualSign {
			replaces["["+sub[1]+"="+sub[2]+"]"] = "[" + thisTag + "=" + replace + "]"
		} else if embeddedURL {
			replaces["["+sub[1]+"]"+sub[2]+"[/"+sub[3]+"]"] = "[" + thisTag + "=" + replace + "]" + sub[2] + "[/" + thisClose + "]"
		} else {
			replaces["["+sub[1]+"]"+sub[2]+"[/"+sub[3]+"]"] = "[" + thisTag + "]" + replace + "[/" + thisClose + "]"
		}
	}

	for k, v := range replaces {
		if k == v {
			delete(replaces, k)
		}
	}
	if len(replaces) > 0 {
		pairs := make([]string, 0, len(replaces)*2)
		for k, v := range replaces {
			pairs = append(pairs, k, v)
		}
		message = strings.NewReplacer(pairs...).Replace(message)
	}
	return message
}

// msgOptions / topicOptions / posterOptions for createPost/modifyPost.
type msgOptions struct {
	ID             int
	Subject        string
	Body           string
	Icon           string
	SmileysEnabled bool
	Attachments    []int
	ModifyTime     int64
	ModifyName     string
}

type topicOptions struct {
	ID         int
	Board      int
	Poll       int // 0 = none
	HasPoll    bool
	LockMode   *int
	StickyMode *int
	MarkAsRead bool
}

type posterOptions struct {
	ID              int
	Name            string
	Email           string
	IP              string
	UpdatePostCount bool
}

// createPost is createPost(&$msgOptions, &$topicOptions, &$posterOptions).
func (c *Ctx) createPost(msg *msgOptions, topic *topicOptions, poster *posterOptions) bool {
	a := c.App

	if msg.Icon == "" {
		msg.Icon = "xx"
	}
	if poster.IP == "" {
		poster.IP = c.User.IP2
	}

	// If nothing was filled in as name/e-mail address, try the member table.
	if poster.Name == "" || (poster.Email == "" && poster.ID != 0) {
		if poster.ID == 0 {
			poster.Name = c.Txt("28")
			poster.Email = ""
		} else if poster.ID != c.User.ID {
			err := a.DB.QueryRow(a.Q(`SELECT memberName, emailAddress FROM {$db_prefix}members WHERE ID_MEMBER = ? LIMIT 1`),
				poster.ID).Scan(&poster.Name, &poster.Email)
			if err != nil {
				poster.ID = 0
				poster.Name = c.Txt("28")
				poster.Email = ""
			}
		} else {
			poster.Name = c.User.Name
			poster.Email = c.User.Email
		}
	}

	newTopic := topic.ID == 0

	// Insert the post.
	res, err := a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}messages
			(ID_BOARD, ID_TOPIC, ID_MEMBER, subject, body, posterName, posterEmail, posterTime,
			posterIP, smileysEnabled, modifiedName, icon)
		VALUES (?, ?, ?, SUBSTR(?, 1, 255), SUBSTR(?, 1, 65534), SUBSTR(?, 1, 255), SUBSTR(?, 1, 255), ?,
			SUBSTR(?, 1, 255), ?, '', SUBSTR(?, 1, 16))`),
		topic.Board, topic.ID, poster.ID, msg.Subject, msg.Body, poster.Name, poster.Email, time.Now().Unix(),
		poster.IP, boolToInt(msg.SmileysEnabled), msg.Icon)
	if err != nil {
		return false
	}
	id64, _ := res.LastInsertId()
	msg.ID = int(id64)
	if msg.ID == 0 {
		return false
	}

	// Fix the attachments.
	if len(msg.Attachments) > 0 {
		ids := make([]string, len(msg.Attachments))
		for i, v := range msg.Attachments {
			ids[i] = itoa(v)
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}attachments SET ID_MSG = ` + itoa(msg.ID) + ` WHERE ID_ATTACH IN (` + strings.Join(ids, ", ") + `)`))
	}

	// Insert a new topic (if the topicID was left empty.)
	if newTopic {
		lockMode, stickyMode := 0, 0
		if topic.LockMode != nil {
			lockMode = *topic.LockMode
		}
		if topic.StickyMode != nil {
			stickyMode = *topic.StickyMode
		}
		res, err := a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}topics
				(ID_BOARD, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_FIRST_MSG, ID_LAST_MSG, locked, isSticky, numViews, ID_POLL)
			VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)`),
			topic.Board, poster.ID, poster.ID, msg.ID, msg.ID, lockMode, stickyMode, topic.Poll)
		if err != nil {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}messages WHERE ID_MSG = ?`), msg.ID)
			return false
		}
		tid, _ := res.LastInsertId()
		topic.ID = int(tid)
		if topic.ID == 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}messages WHERE ID_MSG = ?`), msg.ID)
			return false
		}

		// Fix the message with the topic.
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_TOPIC = ? WHERE ID_MSG = ?`), topic.ID, msg.ID)

		// There's been a new topic AND a new post today.
		c.trackStatsBump("topics", "posts")
		a.UpdateSettings(map[string]string{"totalTopics": itoa(a.SettingInt("totalTopics") + 1)})
		c.updateStatsSubject(topic.ID, msg.Subject)
	} else {
		// Update the number of replies and the lock/sticky status.
		extra := ""
		if topic.LockMode != nil {
			extra += ", locked = " + itoa(*topic.LockMode)
		}
		if topic.StickyMode != nil {
			extra += ", isSticky = " + itoa(*topic.StickyMode)
		}
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET
				ID_MEMBER_UPDATED = ?, ID_LAST_MSG = ?,
				numReplies = numReplies + 1`+extra+`
			WHERE ID_TOPIC = ?`), poster.ID, msg.ID, topic.ID)

		// One new post has been added today.
		c.trackStatsBump("posts")
	}

	// Creating is modifying...in a way.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_MSG_MODIFIED = ? WHERE ID_MSG = ?`), msg.ID, msg.ID)

	// Increase the number of posts and topics on the board.
	if newTopic {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numPosts = numPosts + 1, numTopics = numTopics + 1 WHERE ID_BOARD = ?`), topic.Board)
	} else {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numPosts = numPosts + 1 WHERE ID_BOARD = ?`), topic.Board)
	}

	// Mark inserted topic as read (only for the user calling this function).
	if topic.MarkAsRead && !c.User.IsGuest {
		flag := false
		if !newTopic {
			res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_topics SET ID_MSG = ? WHERE ID_MEMBER = ? AND ID_TOPIC = ?`),
				msg.ID+1, c.User.ID, topic.ID)
			if res != nil {
				n, _ := res.RowsAffected()
				flag = n != 0
			}
		}
		if !flag {
			a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_topics (ID_TOPIC, ID_MEMBER, ID_MSG) VALUES (?, ?, ?)`),
				topic.ID, c.User.ID, msg.ID+1)
		}
	}

	// Increase the post counter for the user that created the post.
	if poster.UpdatePostCount && poster.ID != 0 {
		if c.User.ID == poster.ID {
			c.User.Posts++
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET posts = posts + 1 WHERE ID_MEMBER = ?`), poster.ID)
		c.updateStatsPostgroups(poster.ID)
	}

	// They've posted, so they can make the view count go up one if they
	// really want.
	c.Session.Set("last_read_topic", 0)

	// Update all the stats so everyone knows about this new topic and message.
	a.UpdateSettings(map[string]string{
		"totalMessages": itoa(a.SettingInt("totalMessages") + 1),
		"maxMsgID":      itoa(msg.ID),
	})
	c.updateLastMessages([]int{topic.Board}, msg.ID)

	return true
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// modifyPost is modifyPost(...): edit an existing post.
func (c *Ctx) modifyPost(msg *msgOptions, topic *topicOptions, poster *posterOptions) bool {
	a := c.App

	var sets []string
	var args []any
	if poster.Name != "" {
		sets = append(sets, "posterName = ?")
		args = append(args, poster.Name)
	}
	if poster.Email != "" {
		sets = append(sets, "posterEmail = ?")
		args = append(args, poster.Email)
	}
	if msg.Icon != "" {
		sets = append(sets, "icon = ?")
		args = append(args, msg.Icon)
	}
	if msg.Subject != "" {
		sets = append(sets, "subject = ?")
		args = append(args, msg.Subject)
	}
	if msg.Body != "" {
		sets = append(sets, "body = ?")
		args = append(args, msg.Body)
	}
	if msg.ModifyTime != 0 {
		sets = append(sets, "modifiedTime = ?", "modifiedName = ?", "ID_MSG_MODIFIED = ?")
		args = append(args, msg.ModifyTime, msg.ModifyName, a.SettingInt("maxMsgID"))
	}
	sets = append(sets, "smileysEnabled = ?")
	args = append(args, boolToInt(msg.SmileysEnabled))

	args = append(args, msg.ID)
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET `+strings.Join(sets, ", ")+` WHERE ID_MSG = ?`), args...)

	// Lock and or sticky the post.
	if topic.StickyMode != nil || topic.LockMode != nil || topic.HasPoll {
		sticky, lock, poll := "isSticky", "locked", "ID_POLL"
		if topic.StickyMode != nil {
			sticky = itoa(*topic.StickyMode)
		}
		if topic.LockMode != nil {
			lock = itoa(*topic.LockMode)
		}
		if topic.HasPoll {
			poll = itoa(topic.Poll)
		}
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET
				isSticky = `+sticky+`,
				locked = `+lock+`,
				ID_POLL = `+poll+`
			WHERE ID_TOPIC = ?`), topic.ID)
	}

	// Mark inserted topic as read.
	if topic.MarkAsRead && !c.User.IsGuest {
		a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_topics (ID_TOPIC, ID_MEMBER, ID_MSG) VALUES (?, ?, ?)`),
			topic.ID, c.User.ID, a.SettingInt("maxMsgID"))
	}

	// If the subject changes, updateStats('subject') for the first post.
	if msg.Subject != "" {
		var firstMsg int
		a.DB.QueryRow(a.Q(`SELECT ID_FIRST_MSG FROM {$db_prefix}topics WHERE ID_TOPIC = ?`), topic.ID).Scan(&firstMsg)
		if firstMsg == msg.ID {
			c.updateStatsSubject(topic.ID, msg.Subject)
		}
	}

	return true
}

// updateLastMessages is updateLastMessages($setboards, $ID_MSG).
func (c *Ctx) updateLastMessages(setboards []int, idMsg int) {
	a := c.App
	if len(setboards) == 0 {
		return
	}

	lastMsg := map[int]int{}
	if idMsg == 0 {
		ids := make([]string, len(setboards))
		for i, b := range setboards {
			ids[i] = itoa(b)
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_BOARD, IFNULL(MAX(ID_LAST_MSG), 0) AS ID_MSG
			FROM {$db_prefix}topics
			WHERE ID_BOARD IN (` + strings.Join(ids, ", ") + `)
			GROUP BY ID_BOARD`))
		if err == nil {
			for rows.Next() {
				var b, m int
				rows.Scan(&b, &m)
				lastMsg[b] = m
			}
			rows.Close()
		}
	} else {
		for _, b := range setboards {
			lastMsg[b] = idMsg
		}
	}

	parentBoards := map[int]int{}
	for _, idBoard := range setboards {
		if _, ok := lastMsg[idBoard]; !ok {
			lastMsg[idBoard] = 0
		}
		// Get the board's parents (with their IDs).
		parents := c.getBoardParentsWithIDs(idBoard)
		for id, level := range parents {
			if level == 0 {
				continue
			}
			if v, ok := lastMsg[id]; ok && lastMsg[idBoard] > v {
				lastMsg[id] = lastMsg[idBoard]
			} else if !ok {
				if pv, pok := parentBoards[id]; !pok || pv < lastMsg[idBoard] {
					parentBoards[id] = lastMsg[idBoard]
				}
			}
		}
	}

	for id, msg := range parentBoards {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_MSG_UPDATED = ? WHERE ID_BOARD = ? AND ID_MSG_UPDATED < ?`),
			msg, id, msg)
	}
	for id, msg := range lastMsg {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_LAST_MSG = ?, ID_MSG_UPDATED = ? WHERE ID_BOARD = ?`),
			msg, msg, id)
	}
}

// getBoardParentsWithIDs returns parent board ID -> childLevel.
func (c *Ctx) getBoardParentsWithIDs(idBoard int) map[int]int {
	a := c.App
	out := map[int]int{}
	var idParent int
	a.DB.QueryRow(a.Q(`SELECT ID_PARENT FROM {$db_prefix}boards WHERE ID_BOARD = ?`), idBoard).Scan(&idParent)
	for idParent != 0 {
		var next, level int
		if err := a.DB.QueryRow(a.Q(`SELECT ID_PARENT, childLevel FROM {$db_prefix}boards WHERE ID_BOARD = ?`), idParent).Scan(&next, &level); err != nil {
			break
		}
		out[idParent] = level
		idParent = next
	}
	return out
}

// trackStatsBump increments today's log_activity counters (trackStats '+').
func (c *Ctx) trackStatsBump(fields ...string) {
	a := c.App
	if a.SettingEmpty("trackStats") {
		return
	}
	date := time.Unix(c.forumTime(false, 0), 0).Format("2006-01-02")
	for _, f := range fields {
		res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_activity SET ` + f + ` = ` + f + ` + 1 WHERE date = '` + date + `'`))
		if res != nil {
			if n, _ := res.RowsAffected(); n == 0 {
				a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}log_activity (date, ` + f + `) VALUES ('` + date + `', 1)`))
			}
		}
	}
}

// updateStatsSubject is updateStats('subject', $topic, $subject).
func (c *Ctx) updateStatsSubject(topicID int, subject string) {
	a := c.App
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_subjects WHERE ID_TOPIC = ?`), topicID)
	if subject != "" {
		for _, word := range text2words(subject, 20) {
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}log_search_subjects (word, ID_TOPIC) VALUES (?, ?)`),
				word, topicID)
		}
	}
}

// text2words is text2words($text, $max_chars) (non-encrypt mode).
func text2words(text string, maxChars int) []string {
	cleaned := runesToLatin1(regexp.MustCompile(`([\x0B\x00\x{00A0}\t\r\s\n(){}\[\]<>!@$%^*.,:+=`+"`"+`~\?/\\]|&(amp|lt|gt|quot);)+`).
		ReplaceAllString(latin1ToRunes(strings.ReplaceAll(text, "<br />", " ")), " "))
	cleaned = unHtmlspecialchars(strings.ToLower(cleaned))

	var out []string
	seen := map[string]bool{}
	for _, word := range strings.Split(cleaned, " ") {
		word = strings.Trim(word, "-_'")
		if word == "" {
			continue
		}
		if len(word) > maxChars {
			word = word[:maxChars]
		}
		if !seen[word] {
			seen[word] = true
			out = append(out, word)
		}
	}
	return out
}

// updateStatsPostgroups is updateStats('postgroups', condition).
func (c *Ctx) updateStatsPostgroups(memberID int) {
	a := c.App
	rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, minPosts FROM {$db_prefix}membergroups WHERE minPosts != -1 ORDER BY minPosts DESC`))
	if err != nil {
		return
	}
	type pg struct{ id, minPosts int }
	var groups []pg
	for rows.Next() {
		var g pg
		rows.Scan(&g.id, &g.minPosts)
		groups = append(groups, g)
	}
	rows.Close()
	if len(groups) == 0 {
		return
	}

	conditions := ""
	lastMin := -1
	for _, g := range groups {
		cond := "WHEN posts >= " + itoa(g.minPosts)
		if lastMin >= 0 {
			cond += " AND posts <= " + itoa(lastMin)
		}
		conditions += "\n" + cond + " THEN " + itoa(g.id)
		lastMin = g.minPosts
	}

	where := ""
	if memberID != 0 {
		where = " WHERE ID_MEMBER = " + itoa(memberID)
	}
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_POST_GROUP = CASE` + conditions + ` ELSE 0 END` + where))
}

// latin1ToRunes widens each ISO-8859-1 byte to a rune so Go regexps (which
// require valid UTF-8 patterns and input) can process the text;
// runesToLatin1 reverses it.
func latin1ToRunes(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		b.WriteRune(rune(s[i]))
	}
	return b.String()
}

func runesToLatin1(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteByte(byte(r))
	}
	return b.String()
}
