package app

// Port of Sources/SendTopic.php (SendTopic, ReportToModerator,
// ReportToModerator2) and Themes/default/SendTopic.template.php.

import "strings"

func init() {
	registerAction("sendtopic", (*Ctx).SendTopic)
	registerAction("reporttm", (*Ctx).ReportToModerator)
}

// SendTopicCtx is the page context for the SendTopic templates.
type SendTopicCtx struct {
	Start     string
	MessageID int
}

// SendTopic is SendTopic(): send information about a topic to a friend.
func (c *Ctx) SendTopic() {
	a := c.App
	scripturl := a.ScriptURL

	// Check permissions...
	c.isAllowedTo("send_topic")

	// We need at least a topic... go away if you don't have one.
	if c.Topic == 0 {
		c.fatalLangError("472", false)
	}

	// Get the topic's subject.
	var subject string
	err := a.DB.QueryRow(a.Q(`
		SELECT m.subject
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE t.ID_TOPIC = ?
			AND t.ID_FIRST_MSG = m.ID_MSG
		LIMIT 1`), c.Topic).Scan(&subject)
	if err != nil {
		c.fatalLangError("472", false)
	}

	// Censor the subject....
	subject = c.censorText(subject)

	// Sending yet, or just getting prepped?
	if empty(c.POST.Str("send")) {
		page := &SendTopicCtx{Start: c.REQUEST.Str("start")}
		c.Page = page
		c.PageTitle = phpSprintf(c.Txt("sendtopic_title"), subject)
		c.SubTemplate = templateSendTopicMain
		return
	}

	// Actually send the message...
	c.checkSession("post", "", true)
	c.spamProtection("spam")

	// Trim the names..
	yName := strings.TrimSpace(c.POST.Str("y_name"))
	rName := strings.TrimSpace(c.POST.Str("r_name"))
	yEmail := c.POST.Str("y_email")
	rEmail := c.POST.Str("r_email")

	// Make sure they aren't playing "let's use a fake email".
	if yName == "_" || yName == "" {
		c.fatalLangError("75", false)
	}
	if yEmail == "" {
		c.fatalLangError("76", false)
	}
	if !emailRe.MatchString(yEmail) {
		c.fatalLangError("243", false)
	}

	// The receiver should be valid to.
	if rName == "_" || rName == "" {
		c.fatalLangError("75", false)
	}
	if rEmail == "" {
		c.fatalLangError("76", false)
	}
	if !emailRe.MatchString(rEmail) {
		c.fatalLangError("243", false)
	}

	// Emails don't like entities...
	subject = unHtmlspecialchars(subject)

	comment := ""
	if !empty(c.POST.Str("comment")) {
		comment = c.Txt("sendtopic2") + ":\n" + c.POST.Str("comment") + "\n\n"
	}

	// And off we go!
	c.sendmail([]string{rEmail}, c.Txt("118")+": "+subject+" ("+c.Txt("318")+" "+yName+")",
		phpSprintf(c.Txt("sendtopic_dear"), rName)+"\n\n"+
			phpSprintf(c.Txt("sendtopic_this_topic"), subject)+":\n\n"+
			scripturl+"?topic="+itoa(c.Topic)+".0\n\n"+
			comment+
			c.Txt("sendtopic_thanks")+",\n"+
			yName, yEmail)

	// Back to the topic!
	c.redirectExit("topic=" + itoa(c.Topic) + ".0")
}

// ReportToModerator is ReportToModerator(): ask for a comment to report
// abuse to the moderator(s).
func (c *Ctx) ReportToModerator() {
	a := c.App

	// You can't use this if it's off or you are not allowed to do it.
	c.isAllowedTo("report_any")

	// If they're posting, it should be processed by ReportToModerator2.
	if c.POST.Has("sc") || c.POST.Has("submit") {
		c.ReportToModerator2()
		return
	}

	// We need a message ID to check!
	if empty(c.GET.Str("msg")) && empty(c.GET.Str("mid")) {
		c.fatalLangError("1", false)
	}

	// For compatibility, accept mid, but we should be using msg. (not the
	// flavor kind!)
	msgID := c.GET.Int("msg")
	if msgID == 0 {
		msgID = c.GET.Int("mid")
	}

	// Check the message's ID - don't want anyone reporting a post they
	// can't even see!
	var member, starter int
	err := a.DB.QueryRow(a.Q(`
		SELECT m.ID_MSG, m.ID_MEMBER, t.ID_MEMBER_STARTED
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE m.ID_MSG = ?
			AND m.ID_TOPIC = ?
			AND t.ID_TOPIC = ?
		LIMIT 1`), msgID, c.Topic, c.Topic).Scan(&msgID, &member, &starter)
	if err != nil {
		c.fatalLangError("smf232", true)
	}

	// If they can't modify their post, then they should be able to report
	// it... otherwise it is illogical.
	if member == c.User.ID && (c.allowedTo("modify_own", "modify_any") ||
		(c.User.ID == starter && c.allowedTo("modify_replies"))) {
		c.fatalLangError("rtm_not_own", false)
	}

	// Show the inputs for the comment, etc.
	c.loadLanguage("Post")

	// This is here so that the user could, in theory, be redirected back to
	// the topic.
	page := &SendTopicCtx{Start: c.REQUEST.Str("start"), MessageID: msgID}
	c.Page = page

	c.PageTitle = c.Txt("rtm1")
	c.SubTemplate = templateSendTopicReport
}

// ReportToModerator2 is ReportToModerator2(): send the report emails to all
// the moderators.
func (c *Ctx) ReportToModerator2() {
	a := c.App
	scripturl := a.ScriptURL

	// Check their session... don't want them redirected here without their
	// knowledge.
	c.checkSession("post", "", true)
	c.spamProtection("spam")

	// You must have the proper permissions!
	c.isAllowedTo("report_any")

	// Get the basic topic information, and make sure they can see it.
	msgID := c.POST.Int("msg")

	var subject, posterName, realName string
	var member int
	err := a.DB.QueryRow(a.Q(`
		SELECT m.subject, m.ID_MEMBER, m.posterName, IFNULL(mem.realName, '')
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (m.ID_MEMBER = mem.ID_MEMBER)
		WHERE m.ID_MSG = ?
			AND m.ID_TOPIC = ?
		LIMIT 1`), msgID, c.Topic).Scan(&subject, &member, &posterName, &realName)
	if err != nil {
		c.fatalLangError("smf232", true)
	}

	if member == c.User.ID {
		c.fatalLangError("rtm_not_own", false)
	}

	displayPoster := unHtmlspecialchars(realName)
	if realName != posterName {
		displayPoster += " (" + posterName + ")"
	}
	reporterName := unHtmlspecialchars(c.User.Name)
	if c.User.Name != c.User.Username && c.User.Username != "" {
		reporterName += " (" + c.User.Username + ")"
	}
	subject = unHtmlspecialchars(subject)

	// Get a list of members with the moderate_board permission.
	moderators := c.membersAllowedTo("moderate_board", c.Board)

	ids := make([]string, len(moderators))
	for i, m := range moderators {
		ids[i] = itoa(m)
	}
	type modRow struct{ email string }
	var mods []modRow
	if len(moderators) > 0 {
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, emailAddress, lngfile
			FROM {$db_prefix}members
			WHERE ID_MEMBER IN (` + strings.Join(ids, ", ") + `)
				AND notifyTypes != 4
			ORDER BY lngfile`))
		if err == nil {
			for rows.Next() {
				var id int
				var email, lngfile string
				rows.Scan(&id, &email, &lngfile)
				mods = append(mods, modRow{email})
			}
			rows.Close()
		}
	}

	// Check that moderators do exist!
	if len(mods) == 0 {
		c.fatalLangError("rtm11", false)
	}

	c.loadLanguage("Post")

	reporter := reporterName
	if c.User.ID == 0 {
		reporter = c.Txt("28") + " (" + c.User.IP + ")"
	}

	// Send every moderator an email.
	for _, mod := range mods {
		// Send it to the moderator.
		c.sendmail([]string{mod.email}, c.Txt("rtm3")+": "+subject+" "+c.Txt("rtm4")+" "+displayPoster,
			phpSprintf(c.Txt("rtm_email1"), subject)+" "+displayPoster+" "+c.Txt("rtm_email2")+" "+reporter+" "+c.Txt("rtm_email3")+":\n\n"+
				scripturl+"?topic="+itoa(c.Topic)+".msg"+itoa(msgID)+"#msg"+itoa(msgID)+"\n\n"+
				c.Txt("rtm_email_comment")+":\n"+
				c.POST.Str("comment")+"\n\n"+
				c.Txt("130"), c.User.Email)
	}

	// Back to the board! (you probably don't want to see the post
	// anymore..)
	c.redirectExit("board=" + itoa(c.Board) + ".0")
}

// templateSendTopicMain is template_main() from SendTopic.template.php.
func templateSendTopicMain(c *Ctx) {
	page := c.Page.(*SendTopicCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=sendtopic;topic=`, c.Topic, `.`, page.Start, `" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="400" cellpadding="3" cellspacing="0" border="0" class="tborder" align="center">
				<tr class="titlebg">
					<td align="left" colspan="2">
						<img src="`, c.Theme.ImagesURL(), `/email_sm.gif" alt="" />
						`, c.PageTitle, `
					</td>
				</tr>`)

	// Just show all the input boxes, in a line...
	c.O(`
				<tr class="windowbg">
					<td align="right"><b>`, c.Txt("sendtopic_sender_name"), `:</b></td>
					<td align="left"><input type="text" name="y_name" size="24" maxlength="40" value="`, c.User.Name, `" /></td>
				</tr>
				<tr class="windowbg">
					<td align="right"><b>`, c.Txt("sendtopic_sender_email"), `:</b></td>
					<td align="left"><input type="text" name="y_email" size="24" maxlength="50" value="`, c.User.Email, `" /></td>
				</tr>
				<tr class="windowbg">
					<td align="right"><b>`, c.Txt("sendtopic_comment"), `:</b></td>
					<td align="left"><input type="text" name="comment" size="24" maxlength="100" /></td>
				</tr>
				<tr class="windowbg">
					<td align="center" colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
				</tr>
				<tr class="windowbg">
					<td align="right"><b>`, c.Txt("sendtopic_receiver_name"), `:</b></td>
					<td align="left"><input type="text" name="r_name" size="24" maxlength="40" /></td>
				</tr>
				<tr class="windowbg">
					<td align="right"><b>`, c.Txt("sendtopic_receiver_email"), `:</b></td>
					<td align="left"><input type="text" name="r_email" size="24" maxlength="50" /></td>
				</tr>
				<tr class="windowbg">
					<td align="center" colspan="2"><br /><input type="submit" name="send" value="`, c.Txt("sendtopic_send"), `" /></td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateSendTopicReport is template_report() from SendTopic.template.php.
func templateSendTopicReport(c *Ctx) {
	page := c.Page.(*SendTopicCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=reporttm;topic=`, c.Topic, `.`, page.Start, `" method="post" accept-charset="`, c.CharacterSet, `">
		<input type="hidden" name="msg" value="`+itoa(page.MessageID)+`" />
		<table border="0" width="80%" cellspacing="0" class="tborder" align="center" cellpadding="4">
			<tr class="titlebg">
				<td>`, c.Txt("rtm1"), `</td>
			</tr><tr class="windowbg">
				<td style="padding-bottom: 3ex;" align="center">
					<div style="margin-top: 1ex; margin-bottom: 3ex;" align="left">`, c.Txt("smf315"), `</div>
					`, c.Txt("rtm2"), `: <input type="text" name="comment" size="50" />
					<input type="submit" name="submit" value="`, c.Txt("rtm10"), `" style="margin-left: 1ex;" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
