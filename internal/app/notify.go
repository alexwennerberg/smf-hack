package app

// Port of Sources/Notify.php (Notify, BoardNotify) and the Notify template.

func init() {
	registerAction("notify", (*Ctx).Notify)
	registerAction("notifyboard", (*Ctx).BoardNotify)
}

// NotifyCtx is the page context for the Notify templates.
type NotifyCtx struct {
	NotificationSet bool
	TopicHref       string
	BoardHref       string
	Start           string
}

// Notify is Notify(): turn on/off notification for a particular topic.
func (c *Ctx) Notify() {
	a := c.App
	scripturl := a.ScriptURL

	// Make sure they aren't a guest or something - guests can't really
	// receive notifications!
	c.isNotGuest("")
	c.isAllowedTo("mark_any_notify")

	// Make sure the topic has been specified.
	if c.Topic == 0 {
		c.fatalLangError("472", false)
	}

	// What do we do?  Better ask if they didn't say..
	sa := c.GET.Str("sa")
	if empty(sa) {
		page := &NotifyCtx{}
		c.Page = page

		// Find out if they have notification set for this topic already.
		var dummy int
		err := a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}log_notify
			WHERE ID_MEMBER = ?
				AND ID_TOPIC = ?
			LIMIT 1`), c.User.ID, c.Topic).Scan(&dummy)
		page.NotificationSet = err == nil

		// Set the template variables...
		page.TopicHref = scripturl + "?topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start")
		page.Start = c.REQUEST.Str("start")
		c.PageTitle = c.Txt("418")

		c.SubTemplate = templateNotifyMain
		return
	} else if sa == "on" {
		c.checkSession("get", "", true)

		// Attempt to turn notifications on.
		a.DB.Exec(a.Q(`
			INSERT OR IGNORE INTO {$db_prefix}log_notify
				(ID_MEMBER, ID_TOPIC)
			VALUES (?, ?)`), c.User.ID, c.Topic)
	} else {
		c.checkSession("get", "", true)

		// Just turn notifications off.
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}log_notify
			WHERE ID_MEMBER = ?
				AND ID_TOPIC = ?`), c.User.ID, c.Topic)
	}

	// Send them back to the topic.
	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}

// BoardNotify is BoardNotify(): turn on/off notification for a particular
// board.
func (c *Ctx) BoardNotify() {
	a := c.App
	scripturl := a.ScriptURL

	// Permissions are an important part of anything ;).
	c.isNotGuest("")
	c.isAllowedTo("mark_notify")

	// You have to specify a board to turn notifications on!
	if c.Board == 0 {
		c.fatalLangError("smf232", false)
	}

	// No subaction: find out what to do.
	sa := c.GET.Str("sa")
	if empty(sa) {
		page := &NotifyCtx{}
		c.Page = page

		// Find out if they have notification set for this topic already.
		var dummy int
		err := a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}log_notify
			WHERE ID_MEMBER = ?
				AND ID_BOARD = ?
			LIMIT 1`), c.User.ID, c.Board).Scan(&dummy)
		page.NotificationSet = err == nil

		// Set the template variables...
		page.BoardHref = scripturl + "?board=" + itoa(c.Board) + "." + c.REQUEST.Str("start")
		page.Start = c.REQUEST.Str("start")
		c.PageTitle = c.Txt("418")
		c.SubTemplate = templateNotifyBoard
		return
	} else if sa == "on" {
		c.checkSession("get", "", true)

		// Turn notification on.  (note this just blows smoke if it's
		// already on.)
		a.DB.Exec(a.Q(`
			INSERT OR IGNORE INTO {$db_prefix}log_notify
				(ID_MEMBER, ID_BOARD)
			VALUES (?, ?)`), c.User.ID, c.Board)
	} else {
		c.checkSession("get", "", true)

		// Turn notification off for this board.
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}log_notify
			WHERE ID_MEMBER = ?
				AND ID_BOARD = ?`), c.User.ID, c.Board)
	}

	// Back to the board!
	c.redirectExit("board=" + itoa(c.Board) + "." + c.REQUEST.Str("start"))
}

// templateNotifyMain is template_main() from Notify.template.php.
func templateNotifyMain(c *Ctx) {
	page := c.Page.(*NotifyCtx)
	scripturl := c.App.ScriptURL

	question := c.Txt("126")
	saLink := "on"
	if page.NotificationSet {
		question = c.Txt("212")
		saLink = "off"
	}
	c.O(`
		<table border="0" width="100%" cellspacing="0" cellpadding="3" class="tborder">
			<tr class="titlebg">
				<td>`, c.Txt("125"), `</td>
			</tr>
			<tr class="windowbg">
				<td>
					`, question, `<br />
					<br />
					<b><a href="`, scripturl, `?action=notify;sa=`, saLink, `;topic=`, c.Topic, `.`, page.Start, `;sesc=`, c.Sc, `">`, c.Txt("163"), `</a> - <a href="`, page.TopicHref, `">`, c.Txt("164"), `</a></b>
				</td>
			</tr>
		</table>`)
}

// templateNotifyBoard is template_notify_board() from Notify.template.php.
func templateNotifyBoard(c *Ctx) {
	page := c.Page.(*NotifyCtx)
	scripturl := c.App.ScriptURL

	question := c.Txt("notifyboard_turnon")
	saLink := "on"
	if page.NotificationSet {
		question = c.Txt("notifyboard_turnoff")
		saLink = "off"
	}
	c.O(`
		<table border="0" width="100%" cellspacing="0" cellpadding="3" class="tborder">
			<tr class="titlebg">
				<td>`, c.Txt("125"), `</td>
			</tr>
			<tr class="windowbg">
				<td>
					`, question, `<br />
					<br />
					<b><a href="`, scripturl, `?action=notifyboard;sa=`, saLink, `;board=`, c.Board, `.`, page.Start, `;sesc=`, c.Sc, `">`, c.Txt("163"), `</a> - <a href="`, page.BoardHref, `">`, c.Txt("164"), `</a></b>
				</td>
			</tr>
		</table>`)
}
