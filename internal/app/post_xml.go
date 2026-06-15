package app

// QuoteFast() and JavaScriptModify() from Sources/Post.php, plus their XML
// sub-templates from Themes/default/Xml.template.php (template_quotefast,
// template_modifyfast, template_modifydone, template_modifytopicdone,
// template_post).

import (
	"strings"
	"time"
)

func init() {
	registerAction("quotefast", (*Ctx).QuoteFast)
	registerAction("jsmodify", (*Ctx).JavaScriptModify)
}

// QuoteFastCtx is the page context for the quotefast/modifyfast templates.
type QuoteFastCtx struct {
	CloseWindow bool
	QuoteXML    string
	// modifyfast:
	MsgID   int
	Body    string
	Subject string
}

// QuoteFast is QuoteFast(): return a post's BBC source for the AJAX
// quote/modify features.
func (c *Ctx) QuoteFast() {
	a := c.App

	c.loadLanguage("Post")

	c.checkSession("get", "", true)

	moderateBoards := c.boardsAllowedTo("moderate_board")

	quoteID := c.REQUEST.Int("quote")
	wantModify := c.REQUEST.Has("modify")

	lockedCond := ""
	if wantModify && !(len(moderateBoards) > 0 && moderateBoards[0] == 0) {
		in := ""
		if len(moderateBoards) > 0 {
			ids := make([]string, len(moderateBoards))
			for i, b := range moderateBoards {
				ids[i] = itoa(b)
			}
			in = " OR b.ID_BOARD IN (" + strings.Join(ids, ", ") + ")"
		}
		lockedCond = `
 			AND (t.locked = 0` + in + `)`
	}

	page := &QuoteFastCtx{}
	c.Page = page

	var posterName, body, subject string
	var posterTime int64
	var topicID, locked int
	err := a.DB.QueryRow(a.Q(`
		SELECT IFNULL(mem.realName, m.posterName) AS posterName, m.posterTime, m.body, m.ID_TOPIC, m.subject, t.locked
		FROM {$db_prefix}messages AS m, {$db_prefix}boards AS b, {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_MSG = `+itoa(quoteID)+`
			AND b.ID_BOARD = m.ID_BOARD
			AND t.ID_TOPIC = m.ID_TOPIC
			AND `+c.User.QuerySeeBoard+lockedCond+`
		LIMIT 1`)).Scan(&posterName, &posterTime, &body, &topicID, &subject, &locked)
	page.CloseWindow = err != nil

	c.SubTemplate = templateXMLQuotefast
	if err == nil {
		// Remove special formatting we don't want anymore.
		body = c.unPreparsecode(body)

		// Censor the message!
		body = c.censorText(body)

		body = brToNlRe.ReplaceAllString(body, "\n")

		// Want to modify a single message by double clicking it?
		if wantModify {
			subject = c.censorText(subject)

			c.SubTemplate = templateXMLModifyfast
			page.MsgID = quoteID
			page.Body = body
			page.Subject = strings.ReplaceAll(subject, `"`, `\"`)
			return
		}

		// Remove any nested quotes.
		if !a.SettingEmpty("removeNestedQuotes") {
			body = nestedQuoteRe.ReplaceAllString(body, "")
			body = strings.TrimPrefix(body, "\n")
			body = strings.ReplaceAll(body, "[/quote]", "")
		}

		// Add a quote string on the front and end.
		quote := "[quote author=" + posterName + " link=topic=" + itoa(topicID) + ".msg" + itoa(quoteID) + "#msg" + itoa(quoteID) + " date=" + i64toa(posterTime) + "]\n" + body + "\n[/quote]"
		page.QuoteXML = strings.NewReplacer("&nbsp;", "&#160;", "<", "&lt;", ">", "&gt;").Replace(quote)
	} else if wantModify {
		// In case our message has been removed in the meantime.
		c.SubTemplate = templateXMLModifyfast
		page.MsgID = 0
		page.Body = ""
		page.Subject = ""
	}
}

// JSModifyCtx is the page context for the modifydone/modifytopicdone
// templates.
type JSModifyCtx struct {
	MsgID          int
	ModifiedTime   string
	ModifiedName   string
	Subject        string
	FirstInTopic   bool
	Body           string
	Errors         []string
	ErrorInSubject bool
	ErrorInBody    bool
}

// JavaScriptModify is JavaScriptModify(): in-place edit of a post (quick
// modify / quick topic-subject edit).
func (c *Ctx) JavaScriptModify() {
	a := c.App

	// We have to have a topic!
	if c.Topic == 0 {
		f := false
		c.obExit(&f, nil, false)
	}

	c.checkSession("get", "", true)

	// Assume the first message if no message ID was given.
	msgExpr := "t.ID_FIRST_MSG"
	if !empty(c.REQUEST.Str("msg")) {
		msgExpr = itoa(c.REQUEST.Int("msg"))
	}
	var locked, numReplies, starterID, firstMsgID, rowMsgID, rowMember, smileysEnabled int
	var posterTime, modifiedTime int64
	var rowSubject, rowBody, modifiedName string
	err := a.DB.QueryRow(a.Q(`
			SELECT
				t.locked, t.numReplies, t.ID_MEMBER_STARTED, t.ID_FIRST_MSG,
				m.ID_MSG, m.ID_MEMBER, m.posterTime, m.subject, m.smileysEnabled, m.body,
				m.modifiedTime, IFNULL(m.modifiedName, '')
			FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
			WHERE m.ID_MSG = `+msgExpr+`
				AND m.ID_TOPIC = ?
				AND t.ID_TOPIC = ?`), c.Topic, c.Topic).Scan(
		&locked, &numReplies, &starterID, &firstMsgID,
		&rowMsgID, &rowMember, &posterTime, &rowSubject, &smileysEnabled, &rowBody,
		&modifiedTime, &modifiedName)
	if err != nil {
		c.fatalLangError("smf232", false)
	}

	// Change either body or subject requires permissions to modify messages.
	moderationAction := false
	if c.POST.Has("message") || c.POST.Has("subject") || c.POST.Has("icon") {
		if locked != 0 {
			c.isAllowedTo("moderate_board")
		}

		if rowMember == c.User.ID && !c.allowedTo("modify_any") {
			if !a.SettingEmpty("edit_disable_time") && posterTime+int64(a.SettingInt("edit_disable_time")+5)*60 < time.Now().Unix() {
				c.fatalLangError("modify_post_time_passed", false)
			} else if starterID == c.User.ID && !c.allowedTo("modify_own") {
				c.isAllowedTo("modify_replies")
			} else {
				c.isAllowedTo("modify_own")
			}
		} else if starterID == c.User.ID && !c.allowedTo("modify_any") {
			// Otherwise, they're locked out; someone who can modify the
			// replies is needed.
			c.isAllowedTo("modify_replies")
		} else {
			c.isAllowedTo("modify_any")
		}

		// Only log this action if it wasn't your message.
		moderationAction = rowMember != c.User.ID
	}

	var postErrors []string
	hasSubject := false
	newSubject := ""
	if c.POST.Has("subject") && Htmltrim(c.POST.Str("subject")) != "" {
		hasSubject = true
		newSubject = strings.NewReplacer("\r", "", "\n", "", "\t", "").Replace(Htmlspecialchars(c.POST.Str("subject")))

		// Maximum number of characters.
		if entityLen(newSubject) > 100 {
			newSubject = entitySubstr(newSubject, 0, 100)
		}
	} else {
		// PHP adds no_subject even when no subject was sent at all.
		postErrors = append(postErrors, "no_subject")
	}

	hasMessage := false
	newMessage := ""
	if c.POST.Has("message") {
		if Htmltrim(c.POST.Str("message")) == "" {
			postErrors = append(postErrors, "no_message")
		} else if !a.SettingEmpty("max_messageLength") && entityLen(c.POST.Str("message")) > a.SettingInt("max_messageLength") {
			postErrors = append(postErrors, "long_message")
		} else {
			hasMessage = true
			newMessage = Htmlspecialchars(c.POST.Str("message"))

			newMessage = c.preparsecode(newMessage, false)

			if Htmltrim(stripTagsKeepImg(c.parseBBC(newMessage, false))) == "" {
				postErrors = append(postErrors, "no_message")
				hasMessage = false
				newMessage = ""
			}
		}
	}

	var lockMode *int
	if c.POST.Has("lock") {
		lockVal := c.POST.Int("lock")
		if !c.allowedTo("lock_any", "lock_own") || (!c.allowedTo("lock_any") && c.User.ID != rowMember) {
			// no change
		} else if !c.allowedTo("lock_any") {
			if locked != 1 {
				v := 0
				if !empty(c.POST.Str("lock")) {
					v = 2
				}
				lockMode = &v
			}
		} else if !(locked != 0 && !empty(c.POST.Str("lock"))) && lockVal != locked {
			v := 0
			if !empty(c.POST.Str("lock")) {
				v = 1
			}
			lockMode = &v
		}
	}

	var stickyMode *int
	if c.POST.Has("sticky") && c.allowedTo("make_sticky") && !a.SettingEmpty("enableStickyTopics") {
		v := c.POST.Int("sticky")
		stickyMode = &v
	}

	msg := &msgOptions{ID: rowMsgID}
	if len(postErrors) == 0 {
		if hasSubject {
			msg.Subject = newSubject
		}
		if hasMessage {
			msg.Body = newMessage
		}
		if c.POST.Has("icon") {
			msg.Icon = iconCleanRe.ReplaceAllString(c.POST.Str("icon"), "")
		}
		topicOpts := &topicOptions{
			ID:         c.Topic,
			Board:      c.Board,
			LockMode:   lockMode,
			StickyMode: stickyMode,
			MarkAsRead: true,
		}

		// Only consider marking as editing if they have edited the subject,
		// message or icon.
		// (PHP compares $_POST['icon'] against an unselected $row['icon'] =
		// null, so any non-empty icon counts as a change — kept bug-for-bug.)
		if (hasSubject && newSubject != rowSubject) || (hasMessage && newMessage != rowBody) ||
			(c.POST.Has("icon") && c.POST.Str("icon") != "") {
			// And even then only if the time has passed...
			if time.Now().Unix()-posterTime > int64(a.SettingInt("edit_wait_time")) || c.User.ID != rowMember {
				msg.ModifyTime = time.Now().Unix()
				msg.ModifyName = c.User.Name
			}
		}

		c.modifyPost(msg, topicOpts, &posterOptions{})

		// If we didn't change anything this time but had before put back the
		// old info.
		if msg.ModifyTime == 0 && modifiedTime != 0 {
			msg.ModifyTime = modifiedTime
			msg.ModifyName = modifiedName
		}

		// Changing the first subject updates other subjects to
		// 'Re: new_subject'.
		if hasSubject && c.REQUEST.Has("change_all_subjects") && firstMsgID == rowMsgID && numReplies > 0 &&
			(c.allowedTo("modify_any") || (starterID == c.User.ID && c.allowedTo("modify_replies"))) {
			prefix := c.Txt("response_prefix")
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}messages
				SET subject = ?
				WHERE ID_TOPIC = ?
					AND ID_MSG != ?`), prefix+newSubject, c.Topic, firstMsgID)
		}

		if moderationAction {
			c.logAction("modify", map[string]any{"topic": c.Topic, "message": rowMsgID, "member": starterID})
		}
	}

	if c.REQUEST.Has("xml") {
		page := &JSModifyCtx{MsgID: rowMsgID}
		c.Page = page
		c.SubTemplate = templateXMLModifydone
		if len(postErrors) == 0 && hasSubject && hasMessage {
			if msg.ModifyTime != 0 {
				page.ModifiedTime = c.timeformat(msg.ModifyTime)
				page.ModifiedName = msg.ModifyName
			}
			page.Subject = c.censorText(msg.Subject)
			page.FirstInTopic = rowMsgID == firstMsgID
			page.Body = strings.ReplaceAll(msg.Body, "]]>", "]]]]><![CDATA[>")
			page.Body = c.censorText(page.Body)
			page.Body = c.parseBBCCached(page.Body, smileysEnabled != 0, itoa(rowMsgID))
		} else if len(postErrors) == 0 && hasSubject {
			// Topic?
			c.SubTemplate = templateXMLModifytopicdone
			if msg.ModifyTime != 0 {
				page.ModifiedTime = c.timeformat(msg.ModifyTime)
				page.ModifiedName = msg.ModifyName
			}
			page.Subject = c.censorText(msg.Subject)
		} else {
			page.ErrorInSubject = inStrings(postErrors, "no_subject")
			page.ErrorInBody = inStrings(postErrors, "no_message") || inStrings(postErrors, "long_message")

			c.loadLanguage("Errors")
			for _, e := range postErrors {
				page.Errors = append(page.Errors, c.Txt("error_"+e))
			}
		}
	} else {
		f := false
		c.obExit(&f, nil, false)
	}
}

// templateXMLQuotefast is template_quotefast().
func templateXMLQuotefast(c *Ctx) {
	page := c.Page.(*QuoteFastCtx)
	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>
	<quote>`, page.QuoteXML, `</quote>
</smf>`)
}

// templateXMLModifyfast is template_modifyfast().
func templateXMLModifyfast(c *Ctx) {
	page := c.Page.(*QuoteFastCtx)
	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>
	<subject><![CDATA[`, page.Subject, `]]></subject>
	<message id="msg_`, page.MsgID, `"><![CDATA[`, page.Body, `]]></message>
</smf>`)
}

func xmlModifiedBlock(c *Ctx, page *JSModifyCtx) string {
	if page.ModifiedTime == "" {
		return ""
	}
	return `&#171; <i>` + c.Txt("211") + `: ` + page.ModifiedTime + ` ` + c.Txt("525") + ` ` + page.ModifiedName + `</i> &#187;`
}

// templateXMLModifydone is template_modifydone().
func templateXMLModifydone(c *Ctx) {
	page := c.Page.(*JSModifyCtx)
	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>
	<message id="msg_`, page.MsgID, `">`)
	if len(page.Errors) == 0 {
		first := "0"
		if page.FirstInTopic {
			first = "1"
		}
		c.O(`
		<modified><![CDATA[`, xmlModifiedBlock(c, page), `]]></modified>
		<subject is_first="`, first, `"><![CDATA[`, page.Subject, `]]></subject>
		<body><![CDATA[`, page.Body, `]]></body>`)
	} else {
		c.O(`
		<error in_subject="`, b2s(page.ErrorInSubject), `" in_body="`, b2s(page.ErrorInBody), `"><![CDATA[`, strings.Join(page.Errors, "<br />"), `]]></error>`)
	}
	c.O(`
	</message>
</smf>`)
}

// templateXMLModifytopicdone is template_modifytopicdone().
func templateXMLModifytopicdone(c *Ctx) {
	page := c.Page.(*JSModifyCtx)
	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>
	<message id="msg_`, page.MsgID, `">`)
	if len(page.Errors) == 0 {
		c.O(`
		<modified><![CDATA[`, xmlModifiedBlock(c, page), `]]></modified>
		<subject><![CDATA[`, page.Subject, `]]></subject>`)
	} else {
		c.O(`
		<error in_subject="`, b2s(page.ErrorInSubject), `"><![CDATA[`, strings.Join(page.Errors, "<br />"), `]]></error>`)
	}
	c.O(`
	</message>
</smf>`)
}

func b2s(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// templateXMLPost is template_post(): the AJAX preview response.
func templateXMLPost(c *Ctx) {
	page := c.Page.(*PostCtx)

	serious := "0"
	if page.ErrorType == "serious" {
		serious = "1"
	}
	capColor := func(on bool) string {
		if on {
			return "red"
		}
		return ""
	}
	postErrTag := ""
	if page.PostError["no_message"] || page.PostError["long_message"] {
		postErrTag = `
		<post_error />`
	}
	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>
	<preview>
		<subject><![CDATA[`, page.PreviewSubject, `]]></subject>
		<body><![CDATA[`, page.PreviewMessage, `]]></body>
	</preview>
	<errors serious="`, serious, `" topic_locked="`, b2s(page.Locked), `">`)
	for _, message := range page.ErrorMessages {
		c.O(`
		<error><![CDATA[`, message, `]]></error>`)
	}
	c.O(`
		<caption name="guestname" color="`, capColor(page.PostError["long_name"] || page.PostError["no_name"] || page.PostError["bad_name"]), `" />
		<caption name="email" color="`, capColor(page.PostError["no_email"] || page.PostError["bad_email"]), `" />
		<caption name="subject" color="`, capColor(page.PostError["no_subject"]), `" />
		<caption name="question" color="`, capColor(page.PostError["no_question"]), `" />`, postErrTag, `
	</errors>
	<num_replies>`, page.NumReplies, `</num_replies>`)

	if len(page.PreviousPosts) > 0 {
		c.O(`
	<new_posts>`)
		for _, post := range page.PreviousPosts {
			c.O(`
		<post id="`, post.ID, `">
			<time><![CDATA[`, post.Time, `]]></time>
			<poster><![CDATA[`, post.Poster, `]]></poster>
			<message><![CDATA[`, post.Message, `]]></message>
		</post>`)
		}
		c.O(`
	</new_posts>`)
	}

	c.O(`
</smf>`)
}
