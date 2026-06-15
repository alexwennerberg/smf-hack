package app

// Port of Sources/Karma.php: ModifyKarma and BookOfUnknown (the
// about:mozilla / about:unknown easter eggs).

import "time"

func init() {
	registerAction("modifykarma", (*Ctx).ModifyKarma)
	registerAction("about:mozilla", (*Ctx).BookOfUnknown)
	registerAction("about:unknown", (*Ctx).BookOfUnknown)
}

// ModifyKarma is ModifyKarma(): gives or takes karma from a user.
func (c *Ctx) ModifyKarma() {
	a := c.App

	// If the mod is disabled, show an error.
	if a.SettingEmpty("karmaMode") {
		c.fatalLangError("smf63", true)
	}

	// If you're a guest or can't do this, blow you off...
	c.isNotGuest("")
	c.isAllowedTo("karma_edit")

	c.checkSession("get", "", true)

	// If you don't have enough posts, tough luck.
	if c.User.Posts < a.SettingInt("karmaMinPosts") {
		c.fatalError(c.Txt("smf60")+a.Setting("karmaMinPosts")+".", true)
	}

	// And you can't modify your own, punk! (use the profile if you need
	// to.)
	uid := c.REQUEST.Int("uid")
	if empty(c.REQUEST.Str("uid")) || uid == c.User.ID {
		c.fatalLangError("smf61", false)
	}

	// Applauding or smiting?
	dir := -1
	if c.REQUEST.Str("sa") == "applaud" {
		dir = 1
	}

	// Delete any older items from the log. (karmaWaitTime is by hour.)
	a.DB.Exec(a.Q(`
		DELETE FROM {$db_prefix}log_karma
		WHERE ? - logTime > ?`), time.Now().Unix(), a.SettingInt("karmaWaitTime")*3600)

	// Start off with no change in karma.
	action := 0

	// Not an administrator... or one who is restricted as well.
	if !a.SettingEmpty("karmaTimeRestrictAdmins") || !c.allowedTo("moderate_forum") {
		// Find out if this user has done this recently...
		a.DB.QueryRow(a.Q(`
			SELECT action
			FROM {$db_prefix}log_karma
			WHERE ID_TARGET = ?
				AND ID_EXECUTOR = ?
			LIMIT 1`), uid, c.User.ID).Scan(&action)
	}

	// They haven't, not before now, anyhow.
	if action == 0 || a.SettingEmpty("karmaWaitTime") {
		// Put it in the log.
		a.DB.Exec(a.Q(`
			REPLACE INTO {$db_prefix}log_karma
				(action, ID_TARGET, ID_EXECUTOR, logTime)
			VALUES (?, ?, ?, ?)`), dir, uid, c.User.ID, time.Now().Unix())

		// Change by one.
		col := "karmaBad"
		if dir == 1 {
			col = "karmaGood"
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET `+col+` = `+col+` + 1 WHERE ID_MEMBER = ?`), uid)
	} else {
		// If you are gonna try to repeat.... don't allow it.
		if action == dir {
			c.fatalError(c.Txt("smf62")+" "+a.Setting("karmaWaitTime")+" "+c.Txt("578")+".", false)
		}

		// You decided to go back on your previous choice?
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}log_karma
			SET action = ?, logTime = ?
			WHERE ID_TARGET = ?
				AND ID_EXECUTOR = ?`), dir, time.Now().Unix(), uid, c.User.ID)

		// It was recently changed the OTHER way... so... reverse it!
		if dir == 1 {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET karmaGood = karmaGood + 1, karmaBad = MAX(0, karmaBad - 1) WHERE ID_MEMBER = ?`), uid)
		} else {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET karmaBad = karmaBad + 1, karmaGood = MAX(0, karmaGood - 1) WHERE ID_MEMBER = ?`), uid)
		}
	}

	// Figure out where to go back to.... the topic?
	if c.Topic != 0 {
		c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start") + "#msg" + c.REQUEST.Str("m"))
	} else if c.REQUEST.Has("f") {
		// Hrm... maybe a personal message?
		extra := ""
		if c.REQUEST.Has("l") {
			extra += ";l=" + c.REQUEST.Str("l")
		}
		if c.REQUEST.Has("pm") {
			extra += "#" + c.REQUEST.Str("pm")
		}
		c.redirectExit("action=pm;f=" + c.REQUEST.Str("f") + ";start=" + c.REQUEST.Str("start") + extra)
	} else {
		// JavaScript as a last resort.
		c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
	<head>
		<title>...</title>
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			history.go(-1);
		// ]]></script>
	</head>
	<body>&laquo;</body>
</html>`)

		f := false
		c.obExit(&f, nil, false)
	}
}

// BookOfUnknown is BookOfUnknown(): what's this?  I dunno, what are you
// talking about?  Never seen this before, nope.  No siree.
func (c *Ctx) BookOfUnknown() {
	action := c.GET.Str("action")

	if strContains(action, "mozilla") && !c.Browser.IsGecko {
		c.redirectExit("http://www.getfirefox.com/")
	} else if strContains(action, "mozilla") {
		c.redirectExit("about:mozilla")
	}

	verse := c.GET.Str("verse")
	title := "4:16"
	if verse == "2:18" {
		title = "2:18"
	}
	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
	<head>
		<title>The Book of Unknown, `, title, `</title>
		<style type="text/css">
			em
			{
				font-size: 1.3em;
				line-height: 0;
			}
		</style>
	</head>
	<body style="background-color: #444455; color: white; font-style: italic; font-family: serif;">
		<div style="margin-top: 12%; font-size: 1.1em; line-height: 1.4; text-align: center;">`)
	if verse == "2:18" {
		c.O(`
			Woe, it was that his name wasn't <em>known</em>, that he came in mystery, and was recognized by none.&nbsp;And it became to be in those days <em>something</em>.&nbsp; Something not yet <em id="unknown" name="[Unknown]">unknown</em> to mankind.&nbsp; And thus what was to be known the <em>secret project</em> began into its existence.&nbsp; Henceforth the opposition was only <em>weary</em> and <em>fearful</em>, for now their match was at arms against them.`)
	} else {
		c.O(`
			And it came to pass that the <em>unbelievers</em> dwindled in number and saw rise of many <em>proselytizers</em>, and the opposition found fear in the face of the <em>x</em> and the <em>j</em> while those who stood with the <em>something</em> grew stronger and came together.&nbsp; Still, this was only the <em>beginning</em>, and what lay in the future was <em id="unknown" name="[Unknown]">unknown</em> to all, even those on the right side.`)
	}
	c.O(`
		</div>
		<div style="margin-top: 2ex; font-size: 2em; text-align: right;">`)
	if verse == "2:18" {
		c.O(`
			from <span style="font-family: Georgia, serif;"><strong><a href="http://www.unknownbrackets.com/about:unknown" style="color: white; text-decoration: none; cursor: text;">The Book of Unknown</a></strong>, 2:18</span>`)
	} else {
		c.O(`
			from <span style="font-family: Georgia, serif;"><strong><a href="http://www.unknownbrackets.com/about:unknown" style="color: white; text-decoration: none; cursor: text;">The Book of Unknown</a></strong>, 4:16</span>`)
	}
	c.O(`
		</div>
	</body>
</html>`)

	f := false
	c.obExit(&f, nil, false)
}

func strContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
