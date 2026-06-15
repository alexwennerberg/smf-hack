package app

// Port of Sources/Post.php: Post2 + the notification senders +
// checkSubmitOnce/logAction + the Post2 attachment block. (Post() lives in
// post_form.go; QuoteFast/JavaScriptModify in post_xml.go; AnnounceTopic in
// announce.go.)

import (
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
)

func init() {
	registerAction("post2", (*Ctx).Post2)
}

// checkSubmitOnce is checkSubmitOnce($action): double-post protection.
func (c *Ctx) checkSubmitOnce(action string) bool {
	forms, _ := c.Session.Get("forms").([]any)

	switch action {
	case "register":
		num := 0
		for num == 0 {
			num = rand.Intn(16000000) + 1
			for _, f := range forms {
				if int(toFloat(f)) == num {
					num = 0
					break
				}
			}
		}
		c.FormSequenceNumber = num
		return true
	case "check":
		if !c.REQUEST.Has("seqnum") {
			return true
		}
		seq := c.REQUEST.Int("seqnum")
		for _, f := range forms {
			if int(toFloat(f)) == seq {
				c.fatalLangError("error_form_already_submitted", false)
				return false
			}
		}
		c.Session.Set("forms", append(forms, seq))
		return true
	case "free":
		if c.REQUEST.Has("seqnum") {
			seq := c.REQUEST.Int("seqnum")
			var out []any
			for _, f := range forms {
				if int(toFloat(f)) != seq {
					out = append(out, f)
				}
			}
			c.Session.Set("forms", out)
		}
	}
	return true
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

// logAction is logAction($action, $extra): the moderation log. extra is
// stored as JSON (documented deviation from PHP serialize()).
func (c *Ctx) logAction(action string, extra map[string]any) int {
	a := c.App
	if a.SettingEmpty("modlog_enabled") {
		return 0
	}
	data, _ := json.Marshal(extra)
	ip := c.User.IP
	if len(ip) > 16 {
		ip = ip[:16]
	}
	if len(action) > 30 {
		action = action[:30]
	}
	res, err := a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}log_actions
			(logTime, ID_MEMBER, ip, action, extra)
		VALUES (?, ?, ?, ?, SUBSTR(?, 1, 65534))`),
		time.Now().Unix(), c.User.ID, ip, action, string(data))
	if err != nil {
		return 0
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// notificationStrip converts a body for notification emails.
func (c *Ctx) notificationStrip(body string, cacheID string) string {
	parsed := c.parseBBCCached(body, false, cacheID)
	parsed = strings.NewReplacer("<br />", "\n", "</div>", "\n", "</li>", "\n", "&#91;", "[", "&#93;", "]").Replace(parsed)
	return strings.TrimSpace(unHtmlspecialchars(stripTags(parsed)))
}

// sendNotifications is sendNotifications($ID_TOPIC, $type).
func (c *Ctx) sendNotifications(idTopic int, notifyType string) {
	a := c.App
	types := map[string][2]string{
		"reply":  {"notification_reply_subject", "notification_reply"},
		"sticky": {"notification_sticky_subject", "notification_sticky"},
		"lock":   {"notification_lock_subject", "notification_lock"},
		"unlock": {"notification_unlock_subject", "notification_unlock"},
		"remove": {"notification_remove_subject", "notification_remove"},
		"move":   {"notification_move_subject", "notification_move"},
		"merge":  {"notification_merge_subject", "notification_merge"},
		"split":  {"notification_split_subject", "notification_split"},
	}
	current, ok := types[notifyType]
	if !ok || idTopic == 0 {
		return
	}

	c.loadLanguage("Post")

	// Get the subject and body...
	var subject, body string
	var lastID int
	err := a.DB.QueryRow(a.Q(`
		SELECT mf.subject, ml.body, t.ID_LAST_MSG
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS mf, {$db_prefix}messages AS ml
		WHERE t.ID_TOPIC = ?
			AND mf.ID_MSG = t.ID_FIRST_MSG
			AND ml.ID_MSG = t.ID_LAST_MSG
		LIMIT 1`), idTopic).Scan(&subject, &body, &lastID)
	if err != nil {
		return
	}

	subject = unHtmlspecialchars(c.censorText(subject))
	body = c.notificationStrip(c.censorText(body), itoa(lastID))

	maxType := "3"
	if notifyType == "reply" {
		maxType = "4"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT
			mem.ID_MEMBER, mem.emailAddress, mem.notifyOnce, mem.notifyTypes, mem.notifySendBody, mem.lngfile,
			ln.sent, mem.ID_GROUP, mem.additionalGroups, b.memberGroups, mem.ID_POST_GROUP, t.ID_MEMBER_STARTED
		FROM {$db_prefix}log_notify AS ln, {$db_prefix}members AS mem, {$db_prefix}topics AS t, {$db_prefix}boards AS b
		WHERE ln.ID_TOPIC = ?
			AND t.ID_TOPIC = ?
			AND b.ID_BOARD = t.ID_BOARD
			AND mem.ID_MEMBER != ?
			AND mem.is_activated = 1
			AND mem.notifyTypes < `+maxType+`
			AND ln.ID_MEMBER = mem.ID_MEMBER
		GROUP BY mem.ID_MEMBER
		ORDER BY mem.lngfile`), idTopic, idTopic, c.User.ID)
	if err != nil {
		return
	}
	defer rows.Close()

	sent := 0
	for rows.Next() {
		var idMember, notifyOnce, notifyTypes, notifySendBody, lnSent, idGroup, idPostGroup, idStarted int
		var email, lngfile, additionalGroups, memberGroups string
		rows.Scan(&idMember, &email, &notifyOnce, &notifyTypes, &notifySendBody, &lngfile,
			&lnSent, &idGroup, &additionalGroups, &memberGroups, &idPostGroup, &idStarted)

		// If they aren't the topic poster do they really want to know?
		if notifyType != "reply" && notifyTypes == 2 && idMember != idStarted {
			continue
		}

		if idGroup != 1 {
			if !groupsIntersect(memberGroups, additionalGroups, idGroup, idPostGroup) {
				continue
			}
		}

		message := phpSprintf(c.Txt(current[1]), unHtmlspecialchars(c.User.Name))
		if notifyType != "remove" {
			message += a.ScriptURL + "?topic=" + itoa(idTopic) + ".new;topicseen#new\n\n" +
				c.Txt("notifyUnsubscribe") + ": " + a.ScriptURL + "?action=notify;topic=" + itoa(idTopic) + ".0"
		}
		if notifySendBody != 0 && notifyType == "reply" && a.SettingEmpty("disallow_sendBody") {
			message += "\n\n" + c.Txt("notification_reply_body") + "\n\n" + body
		}
		if notifyOnce != 0 && notifyType == "reply" {
			message += "\n\n" + c.Txt("notifyXOnce2")
		}

		// Send only if once is off or it's on and it hasn't been sent.
		if notifyType != "reply" || notifyOnce == 0 || lnSent == 0 {
			c.sendmail([]string{email}, phpSprintf(c.Txt(current[0]), subject),
				message+"\n\n"+c.Txt("130"), "")
			sent++
		}
	}

	if sent > 0 && notifyType == "reply" {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_notify SET sent = 1 WHERE ID_TOPIC = ? AND ID_MEMBER != ?`),
			idTopic, c.User.ID)
	}
}

func groupsIntersect(allowedCSV, additionalCSV string, idGroup, idPostGroup int) bool {
	allowed := map[string]bool{}
	for _, g := range strings.Split(allowedCSV, ",") {
		allowed[g] = true
	}
	if allowed[itoa(idGroup)] || allowed[itoa(idPostGroup)] {
		return true
	}
	for _, g := range strings.Split(additionalCSV, ",") {
		if g != "" && allowed[g] {
			return true
		}
	}
	return false
}

// notifyMembersBoard is notifyMembersBoard() from Post.php.
func (c *Ctx) notifyMembersBoard(subject, message string) {
	a := c.App
	if c.Board == 0 {
		return
	}

	c.loadLanguage("Post")

	subject = unHtmlspecialchars(c.censorText(subject))
	message = c.notificationStrip(c.censorText(message), "")

	rows, err := a.DB.Query(a.Q(`
		SELECT
			mem.ID_MEMBER, mem.emailAddress, mem.notifyOnce, mem.notifySendBody, mem.lngfile,
			ln.sent, mem.ID_GROUP, mem.additionalGroups, b.memberGroups, mem.ID_POST_GROUP
		FROM {$db_prefix}log_notify AS ln, {$db_prefix}members AS mem, {$db_prefix}boards AS b
		WHERE ln.ID_BOARD = ?
			AND b.ID_BOARD = ?
			AND mem.ID_MEMBER != ?
			AND mem.is_activated = 1
			AND mem.notifyTypes != 4
			AND ln.ID_MEMBER = mem.ID_MEMBER
		GROUP BY mem.ID_MEMBER
		ORDER BY mem.lngfile`), c.Board, c.Board, c.User.ID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var idMember, notifyOnce, notifySendBody, lnSent, idGroup, idPostGroup int
		var email, lngfile, additionalGroups, memberGroups string
		rows.Scan(&idMember, &email, &notifyOnce, &notifySendBody, &lngfile,
			&lnSent, &idGroup, &additionalGroups, &memberGroups, &idPostGroup)

		if idGroup != 1 && !groupsIntersect(memberGroups, additionalGroups, idGroup, idPostGroup) {
			continue
		}

		bodyText := ""
		if a.SettingEmpty("disallow_sendBody") {
			bodyText = c.Txt("notification_new_topic_body") + "\n\n" + message + "\n\n"
		}
		sendSubject := phpSprintf(c.Txt("notify_boards_subject"), subject)
		base := phpSprintf(c.Txt("notify_boards"), subject, a.ScriptURL+"?topic="+itoa(c.Topic)+".new#new", unHtmlspecialchars(c.User.Name))
		unsub := c.Txt("notify_boardsUnsubscribe") + ": " + a.ScriptURL + "?action=notifyboard;board=" + itoa(c.Board) + ".0\n\n" + c.Txt("130")
		sendBody := ""
		if notifySendBody != 0 {
			sendBody = bodyText
		}

		if notifyOnce != 0 && lnSent == 0 {
			c.sendmail([]string{email}, sendSubject, base+c.Txt("notify_boards_once")+"\n\n"+sendBody+unsub, "")
		} else if notifyOnce == 0 {
			c.sendmail([]string{email}, sendSubject, base+sendBody+unsub, "")
		}
	}

	a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_notify SET sent = 1 WHERE ID_BOARD = ? AND ID_MEMBER != ?`),
		c.Board, c.User.ID)
}

var iconCleanRe = regexp.MustCompile(`[\./\\*':"<>]`)

// Post2 is Post2(): handle a post submission.
func (c *Ctx) Post2() {
	a := c.App

	// Previewing? Go back to start.
	if c.REQUEST.Has("preview") {
		c.checkSession("post", "", true)
		c.Post()
		return
	}

	// Prevent double submission of this form.
	c.checkSubmitOnce("check")

	// No errors as yet.
	var postErrors []string

	// If the session has timed out, let the user re-submit their form.
	if c.checkSession("post", "", false) != "" {
		postErrors = append(postErrors, "session_timeout")
	}

	c.loadLanguage("Post")

	var posterIsGuest bool
	var editRow struct {
		idMember, firstMsg, locked, isSticky, posterStarter int
		posterName, posterEmail                             string
		posterTime                                          int64
		loaded                                              bool
	}
	moderationAction := false

	lockPost := -1   // -1 = unset
	stickyPost := -1 // -1 = unset
	if c.POST.Has("lock") {
		lockPost = c.POST.Int("lock")
	}
	if c.POST.Has("sticky") {
		stickyPost = c.POST.Int("sticky")
	}

	msgID := c.REQUEST.Int("msg")
	hasMsg := c.REQUEST.Has("msg") && msgID != 0
	hasPoll := c.REQUEST.Has("poll")

	if c.Topic != 0 && !hasMsg {
		// Replying to a topic?
		var tmpLocked, tmpStickied, pollID, numReplies, posterID int
		err := a.DB.QueryRow(a.Q(`
			SELECT t.locked, t.isSticky, t.ID_POLL, t.numReplies, m.ID_MEMBER
			FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m
			WHERE t.ID_TOPIC = ?
				AND m.ID_MSG = t.ID_FIRST_MSG
			LIMIT 1`), c.Topic).Scan(&tmpLocked, &tmpStickied, &pollID, &numReplies, &posterID)
		if err != nil {
			c.fatalLangError("472", false)
		}

		// Don't allow a post if it's locked.
		if tmpLocked != 0 && !c.allowedTo("moderate_board") {
			c.fatalLangError("90", false)
		}

		// Sorry, multiple polls aren't allowed... yet.
		if hasPoll && pollID > 0 {
			hasPoll = false
		}

		if posterID != c.User.ID {
			c.isAllowedTo("post_reply_any")
		} else if !c.allowedTo("post_reply_any") {
			c.isAllowedTo("post_reply_own")
		}

		if lockPost >= 0 {
			if (tmpLocked == 0 && lockPost == 0) || (lockPost != 0 && tmpLocked != 0) {
				lockPost = -1
			} else if !c.allowedTo("lock_any", "lock_own") || (!c.allowedTo("lock_any") && c.User.ID != posterID) {
				lockPost = -1
			} else if !c.allowedTo("lock_any") {
				if tmpLocked == 1 {
					lockPost = -1
				} else if lockPost != 0 {
					lockPost = 2
				} else {
					lockPost = 0
				}
			} else if lockPost != 0 {
				lockPost = 1
			} else {
				lockPost = 0
			}
		}

		// So you wanna (un)sticky this...let's see.
		if stickyPost >= 0 && (a.SettingEmpty("enableStickyTopics") || stickyPost == tmpStickied || !c.allowedTo("make_sticky")) {
			stickyPost = -1
		}

		// If the number of replies has changed, go back to Post().
		if empty(c.Options["no_new_reply_warning"]) && c.POST.Has("num_replies") && numReplies > c.POST.Int("num_replies") {
			c.REQUEST.Set("preview", "1")
			c.Post()
			return
		}

		posterIsGuest = c.User.IsGuest
	} else if c.Topic == 0 {
		// Posting a new topic.
		hasMsg = false

		if !hasPoll || a.Setting("pollMode") != "1" {
			c.isAllowedTo("post_new")
		}

		if lockPost >= 0 {
			if lockPost == 0 {
				lockPost = -1
			} else if !c.allowedTo("lock_any", "lock_own") {
				lockPost = -1
			} else if c.allowedTo("lock_any") {
				lockPost = 1
			} else {
				lockPost = 2
			}
		}

		if stickyPost >= 0 && (a.SettingEmpty("enableStickyTopics") || stickyPost == 0 || !c.allowedTo("make_sticky")) {
			stickyPost = -1
		}

		posterIsGuest = c.User.IsGuest
	} else {
		// Modifying an existing message?
		err := a.DB.QueryRow(a.Q(`
			SELECT
				m.ID_MEMBER, m.posterName, m.posterEmail, m.posterTime,
				t.ID_FIRST_MSG, t.locked, t.isSticky, t.ID_MEMBER_STARTED
			FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
			WHERE m.ID_MSG = ?
				AND t.ID_TOPIC = ?
			LIMIT 1`), msgID, c.Topic).Scan(&editRow.idMember, &editRow.posterName, &editRow.posterEmail,
			&editRow.posterTime, &editRow.firstMsg, &editRow.locked, &editRow.isSticky, &editRow.posterStarter)
		if err != nil {
			c.fatalLangError("smf272", false)
		}
		editRow.loaded = true

		if editRow.locked != 0 && !c.allowedTo("moderate_board") {
			c.fatalLangError("90", false)
		}

		if lockPost >= 0 {
			if (lockPost == 0 && editRow.locked == 0) || (lockPost != 0 && editRow.locked != 0) {
				lockPost = -1
			} else if !c.allowedTo("lock_any", "lock_own") || (!c.allowedTo("lock_any") && c.User.ID != editRow.posterStarter) {
				lockPost = -1
			} else if !c.allowedTo("lock_any") {
				if editRow.locked == 1 {
					lockPost = -1
				} else if lockPost != 0 {
					lockPost = 2
				} else {
					lockPost = 0
				}
			} else if lockPost != 0 {
				lockPost = 1
			} else {
				lockPost = 0
			}
		}

		// Change the sticky status of this topic?
		if stickyPost >= 0 && (!c.allowedTo("make_sticky") || stickyPost == editRow.isSticky) {
			stickyPost = -1
		}

		if editRow.idMember == c.User.ID && !c.allowedTo("modify_any") {
			if !a.SettingEmpty("edit_disable_time") && editRow.posterTime+int64(a.SettingInt("edit_disable_time")+5)*60 < time.Now().Unix() {
				c.fatalLangError("modify_post_time_passed", false)
			} else if editRow.posterStarter == c.User.ID && !c.allowedTo("modify_own") {
				c.isAllowedTo("modify_replies")
			} else {
				c.isAllowedTo("modify_own")
			}
		} else if editRow.posterStarter == c.User.ID && !c.allowedTo("modify_any") {
			c.isAllowedTo("modify_replies")
			moderationAction = true
		} else {
			c.isAllowedTo("modify_any")
			if editRow.idMember != c.User.ID {
				moderationAction = true
			}
		}

		posterIsGuest = editRow.idMember == 0

		if !c.allowedTo("moderate_forum") || !posterIsGuest {
			c.POST.Set("guestname", editRow.posterName)
			c.POST.Set("email", editRow.posterEmail)
		}
	}

	// If the poster is a guest evaluate the legality of name and email.
	if posterIsGuest {
		guestname := strings.TrimSpace(c.POST.Str("guestname"))
		email := strings.TrimSpace(c.POST.Str("email"))
		c.POST.Set("guestname", guestname)
		c.POST.Set("email", email)

		if guestname == "" || guestname == "_" {
			postErrors = append(postErrors, "no_name")
		}
		if entityLen(guestname) > 25 {
			postErrors = append(postErrors, "long_name")
		}

		if a.SettingEmpty("guest_post_no_email") {
			// Only check if they changed it!
			if !editRow.loaded || editRow.posterEmail != email {
				if !c.allowedTo("moderate_forum") && email == "" {
					postErrors = append(postErrors, "no_email")
				}
				if !c.allowedTo("moderate_forum") && !emailRe.MatchString(email) {
					postErrors = append(postErrors, "bad_email")
				}
			}

			// Now make sure this email address is not banned from posting.
			c.isBannedEmail(email, "cannot_post", phpSprintf(c.Txt("you_are_post_banned"), c.Txt("28")))
		}
	}

	// Check the subject and message.
	if Htmltrim(c.POST.Str("subject")) == "" {
		postErrors = append(postErrors, "no_subject")
	}
	message := c.POST.Str("message")
	if Htmltrim(message) == "" {
		postErrors = append(postErrors, "no_message")
	} else if !a.SettingEmpty("max_messageLength") && entityLen(message) > a.SettingInt("max_messageLength") {
		postErrors = append(postErrors, "long_message")
	} else {
		// Prepare the message a bit for some additional testing.
		message = Htmlspecialchars(message)

		// Preparse code. (Zef)
		if c.User.IsGuest {
			c.User.Name = c.POST.Str("guestname")
		}
		message = c.preparsecode(message, false)
		c.POST.Set("message", message)

		// Let's see if there's still some content left without the tags.
		stripped := stripTagsKeepImg(c.parseBBC(message, false))
		if Htmltrim(stripped) == "" {
			postErrors = append(postErrors, "no_message")
		}
	}

	// You are not!
	if strings.ToLower(c.POST.Str("message")) == "i am the administrator." && !c.User.IsAdmin {
		c.fatalError("Knave! Masquerader! Charlatan!", false)
	}

	// Validate the poll...
	var pollOptions []string
	if hasPoll && a.Setting("pollMode") == "1" {
		if c.Topic != 0 && !hasMsg {
			c.fatalLangError("1", false)
		}

		// This is a new topic... so it's a new poll.
		if c.Topic == 0 {
			c.isAllowedTo("poll_post")
		} else if c.User.ID == editRow.posterStarter && !c.allowedTo("poll_add_any") {
			c.isAllowedTo("poll_add_own")
		} else {
			c.isAllowedTo("poll_add_any")
		}

		if strings.TrimSpace(c.POST.Str("question")) == "" {
			postErrors = append(postErrors, "no_question")
		}

		if opts := c.POST.Arr("options"); opts != nil {
			opts.Values(func(k string, v any) {
				if s, ok := v.(string); ok {
					s = Htmltrim(s)
					if s != "" {
						pollOptions = append(pollOptions, s)
					}
				}
			})
		}

		// What are you going to vote between with one choice?!?
		if len(pollOptions) < 2 {
			postErrors = append(postErrors, "poll_few")
		}
	}

	if posterIsGuest {
		// If user is a guest, make sure the chosen name isn't taken.
		if c.isReservedName(c.POST.Str("guestname"), 0, true, false) &&
			(!editRow.loaded || c.POST.Str("guestname") != editRow.posterName) {
			postErrors = append(postErrors, "bad_name")
		}
	} else if !hasMsg {
		c.POST.Set("guestname", c.User.Username)
		c.POST.Set("email", c.User.Email)
	}

	// Any mistakes?
	if len(postErrors) > 0 {
		c.loadLanguage("Errors")
		c.REQUEST.Set("preview", "1")
		c.PostErrors = postErrors
		c.Post()
		return
	}

	// Make sure the user isn't spamming the board.
	if !hasMsg {
		c.spamProtection("spam")
	}

	// Add special html entities to the subject, name, and email.
	subject := strings.NewReplacer("\r", "", "\n", "", "\t", "").Replace(Htmlspecialchars(c.POST.Str("subject")))
	guestname := Htmlspecialchars(c.POST.Str("guestname"))
	email := Htmlspecialchars(c.POST.Str("email"))

	// At this point, we want to make sure the subject isn't too long.
	if entityLen(subject) > 100 {
		subject = entitySubstr(subject, 0, 100)
	}

	// Check if they are trying to delete any current attachments....
	if hasMsg && c.POST.Arr("attach_del") != nil && c.allowedTo("post_attachment") {
		del := c.POST.Arr("attach_del")
		var delTemp []string
		for _, k := range del.Keys() {
			delTemp = append(delTemp, itoa(atoi(del.Str(k))))
		}
		a.removeAttachments("a.attachmentType = 0 AND a.ID_MSG = "+itoa(msgID)+
			" AND a.ID_ATTACH NOT IN ("+strings.Join(delTemp, ", ")+")", "", false, true)
	}

	// ...or attach a new file...
	attachIDs := c.handlePost2Attachments(hasMsg, msgID)

	// Make the poll...
	idPoll := 0
	if hasPoll {
		maxVotes := c.POST.Int("poll_max_votes")
		if maxVotes <= 0 {
			maxVotes = 1
		} else if maxVotes > len(pollOptions) {
			maxVotes = len(pollOptions)
		}
		pollHide := c.POST.Int("poll_hide")
		pollChangeVote := boolToInt(c.POST.Has("poll_change_vote"))

		pollExpire := c.POST.Int("poll_expire")
		if c.POST.Has("poll_expire") && !empty(c.POST.Str("poll_expire")) && pollExpire < 1 {
			c.fatalLangError("poll_range_error", false)
		} else if pollExpire == 0 && pollHide == 2 {
			pollHide = 1
		}

		question := Htmlspecialchars(c.POST.Str("question"))
		expireTime := 0
		if pollExpire > 0 {
			expireTime = int(time.Now().Unix()) + pollExpire*3600*24
		}

		res, err := a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}polls
				(question, hideResults, maxVotes, expireTime, ID_MEMBER, posterName, changeVote)
			VALUES (SUBSTR(?, 1, 255), ?, ?, ?, ?, SUBSTR(?, 1, 255), ?)`),
			question, pollHide, maxVotes, expireTime, c.User.ID, guestname, pollChangeVote)
		if err == nil {
			pid, _ := res.LastInsertId()
			idPoll = int(pid)
			for i, option := range pollOptions {
				a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}poll_choices (ID_POLL, ID_CHOICE, label) VALUES (?, ?, SUBSTR(?, 1, 255))`),
					idPoll, i, Htmlspecialchars(option))
			}
		}
	}

	// Creating a new topic?
	newTopic := !hasMsg && c.Topic == 0

	// Collect all parameters for the creation or modification of a post.
	msg := &msgOptions{
		ID:             msgID,
		Subject:        subject,
		Body:           c.POST.Str("message"),
		Icon:           iconCleanRe.ReplaceAllString(c.POST.Str("icon"), ""),
		SmileysEnabled: !c.POST.Has("ns"),
		Attachments:    attachIDs,
	}
	topicOpts := &topicOptions{
		ID:         c.Topic,
		Board:      c.Board,
		Poll:       idPoll,
		HasPoll:    hasPoll,
		MarkAsRead: true,
	}
	if lockPost >= 0 {
		topicOpts.LockMode = &lockPost
	}
	if stickyPost >= 0 {
		topicOpts.StickyMode = &stickyPost
	}
	poster := &posterOptions{
		ID:              c.User.ID,
		Name:            guestname,
		Email:           email,
		UpdatePostCount: !c.User.IsGuest && !hasMsg && c.BoardInfo.PostsCount,
	}

	if hasMsg {
		// This is an already existing message. Edit it.
		if time.Now().Unix()-editRow.posterTime > int64(a.SettingInt("edit_wait_time")) || c.User.ID != editRow.idMember {
			msg.ModifyTime = time.Now().Unix()
			msg.ModifyName = c.User.Name
		}
		c.modifyPost(msg, topicOpts, poster)
	} else {
		// This is a new topic or an already existing one. Save it.
		c.createPost(msg, topicOpts, poster)
		c.Topic = topicOpts.ID
	}

	// Marking read should be done even for editing messages....
	if !c.User.IsGuest && c.BoardInfo != nil && len(c.BoardInfo.ParentBoards) > 0 {
		// (Parent boards in our BoardInfo carry URLs, not IDs; re-derive.)
		for id := range c.getBoardParentsWithIDs(c.Board) {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_boards SET ID_MSG = ? WHERE ID_MEMBER = ? AND ID_BOARD = ?`),
				a.SettingInt("maxMsgID"), c.User.ID, id)
		}
	}

	// Turn notification on or off.
	if !empty(c.POST.Str("notify")) {
		if c.allowedTo("mark_any_notify") {
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}log_notify (ID_MEMBER, ID_TOPIC, ID_BOARD) VALUES (?, ?, 0)`),
				c.User.ID, c.Topic)
		}
	} else if !newTopic {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_notify WHERE ID_MEMBER = ? AND ID_TOPIC = ?`),
			c.User.ID, c.Topic)
	}

	// Log an act of moderation - modifying.
	if moderationAction {
		c.logAction("modify", map[string]any{"topic": c.Topic, "message": msgID, "member": editRow.idMember})
	}

	if lockPost >= 0 && lockPost != 2 {
		c.logAction("lock", map[string]any{"topic": topicOpts.ID})
	}
	if stickyPost >= 0 && !a.SettingEmpty("enableStickyTopics") {
		c.logAction("sticky", map[string]any{"topic": topicOpts.ID})
	}

	// Notify any members who have notification turned on for this topic.
	if newTopic {
		c.notifyMembersBoard(subject, c.POST.Str("message"))
	} else if !hasMsg {
		c.sendNotifications(c.Topic, "reply")
	}

	// Returning to the topic?
	if !empty(c.REQUEST.Str("goback")) {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_boards SET ID_MSG = ? WHERE ID_MEMBER = ? AND ID_BOARD = ?`),
			a.SettingInt("maxMsgID"), c.User.ID, c.Board)
	}

	if !empty(c.POST.Str("move")) && c.allowedTo("move_any") {
		goback := ""
		if !empty(c.REQUEST.Str("goback")) {
			goback = ";goback"
		}
		c.redirectExit("action=movetopic;topic=" + itoa(c.Topic) + ".0" + goback)
	}

	// Return to post if the mod is on.
	if hasMsg && !empty(c.REQUEST.Str("goback")) {
		c.redirectExit("topic=" + itoa(c.Topic) + ".msg" + itoa(msgID) + "#msg" + itoa(msgID))
	} else if !empty(c.REQUEST.Str("goback")) {
		c.redirectExit("topic=" + itoa(c.Topic) + ".new#new")
	} else {
		c.redirectExit("board=" + itoa(c.Board) + ".0")
	}
}

var imgKeepRe = regexp.MustCompile(`(?i)<img[^>]*>`)

// stripTagsKeepImg is strip_tags($s, '<img>').
func stripTagsKeepImg(s string) string {
	placeholder := "\x01IMG\x01"
	var imgs []string
	s = imgKeepRe.ReplaceAllStringFunc(s, func(m string) string {
		imgs = append(imgs, m)
		return placeholder
	})
	s = stripTags(s)
	for _, img := range imgs {
		s = strings.Replace(s, placeholder, img, 1)
	}
	return s
}

// handlePost2Attachments is the attachment block of Post2() (Post.php lines
// 1474-1567): turn session temp attachments and direct uploads into real
// attachment rows via createAttachment.
func (c *Ctx) handlePost2Attachments(hasMsg bool, msgID int) []int {
	a := c.App

	type pending struct {
		tmpName string // path or post_tmp_* name
		name    string
		size    int
	}
	var files []pending

	// upload_tmp_* files staged from a direct multipart upload this request. A
	// successful createAttachment renames the temp into place (so removing it
	// here is a no-op), but a rejected attachment (bad extension, too large,
	// directory full, ...) aborts via fatalError before any cleanup — without
	// this the temp would leak on disk. The defer runs on the smfExit unwind too.
	var stagedTemps []string
	defer func() {
		for _, p := range stagedTemps {
			os.Remove(p)
		}
	}()

	temp := c.tempAttachments()
	haveDirect := c.R.MultipartForm != nil && len(c.R.MultipartForm.File["attachment[]"]) > 0

	if !haveDirect && len(temp) == 0 {
		return nil
	}

	c.isAllowedTo("post_attachment")

	// If this isn't a new post, check the current attachments.
	quantity := 0
	totalSize := 0
	if hasMsg {
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*), IFNULL(SUM(size), 0)
			FROM {$db_prefix}attachments
			WHERE ID_MSG = ?
				AND attachmentType = 0`), msgID).Scan(&quantity, &totalSize)
	}

	attachDel := map[string]bool{}
	attachDelSent := false
	if del := c.POST.Arr("attach_del"); del != nil {
		attachDelSent = del.Len() > 0
		for _, k := range del.Keys() {
			attachDel[del.Str(k)] = true
		}
	}

	myTmpRe := regexp.MustCompile(`^post_tmp_` + itoa(c.User.ID) + `_\d+$`)
	for _, t := range temp {
		if !myTmpRe.MatchString(t.ID) {
			continue
		}

		if attachDelSent && !attachDel[t.ID] {
			os.Remove(a.Setting("attachmentUploadDir") + "/" + t.ID)
			continue
		}

		size := 0
		if info, err := os.Stat(a.Setting("attachmentUploadDir") + "/" + t.ID); err == nil {
			size = int(info.Size())
		}
		files = append(files, pending{tmpName: t.ID, name: t.Name, size: size})
	}
	c.setTempAttachments(nil)

	if haveDirect {
		for _, fh := range c.R.MultipartForm.File["attachment[]"] {
			if fh.Filename == "" {
				continue
			}
			// Spool the upload to a temp file so createAttachment can
			// rename it into place.
			src, err := fh.Open()
			if err != nil {
				c.fatalLangError("smf124", true)
			}
			tmp, err := os.CreateTemp(a.Setting("attachmentUploadDir"), "upload_tmp_*")
			if err != nil {
				src.Close()
				c.fatalLangError("attachments_no_write", true)
			}
			stagedTemps = append(stagedTemps, tmp.Name())
			if _, err := io.Copy(tmp, src); err != nil {
				tmp.Close()
				src.Close()
				c.fatalLangError("smf124", true)
			}
			tmp.Close()
			src.Close()
			files = append(files, pending{tmpName: tmp.Name(), name: filepathBase(fh.Filename), size: int(fh.Size)})
		}
	}

	var attachIDs []int
	for _, f := range files {
		if f.name == "" {
			continue
		}

		// Have we reached the maximum number of files we are allowed?
		quantity++
		if !a.SettingEmpty("attachmentNumPerPostLimit") && quantity > a.SettingInt("attachmentNumPerPostLimit") {
			c.fatalLangError("attachments_limit_per_post", false, a.Setting("attachmentNumPerPostLimit"))
		}

		// Check the total upload size for this post...
		totalSize += f.size
		if !a.SettingEmpty("attachmentPostLimit") && totalSize > a.SettingInt("attachmentPostLimit")*1024 {
			c.fatalLangError("smf122", false, a.Setting("attachmentPostLimit"))
		}

		opt := &attachmentOptions{
			Post:    msgID,
			Poster:  c.User.ID,
			Name:    f.name,
			TmpName: f.tmpName,
			Size:    f.size,
		}

		if c.createAttachment(opt) {
			attachIDs = append(attachIDs, opt.ID)
			if opt.Thumb != 0 {
				attachIDs = append(attachIDs, opt.Thumb)
			}
		} else {
			has := func(e string) bool {
				for _, x := range opt.Errors {
					if x == e {
						return true
					}
				}
				return false
			}
			if has("could_not_upload") {
				c.fatalLangError("smf124", true)
			}
			if has("too_large") {
				c.fatalLangError("smf122", false, a.Setting("attachmentSizeLimit"))
			}
			if has("bad_extension") {
				c.fatalError(opt.Name+".<br />"+c.Txt("smf123")+" "+a.Setting("attachmentExtensions")+".", false)
			}
			if has("directory_full") {
				c.fatalLangError("smf126", true)
			}
			if has("bad_filename") {
				c.fatalError(filepathBase(opt.Name)+".<br />"+c.Txt("smf130b")+".", true)
			}
			if has("taken_filename") {
				c.fatalLangError("smf125", true)
			}
		}
	}
	return attachIDs
}
