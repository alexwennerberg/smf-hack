package app

// More of Sources/Subs-Members.php (Phase 4 scope): deleteMembers, plus
// updateMemberData (Load.php) and resetPassword (Subs-Auth.php) and the
// deleteMessages() PM helper from PersonalMessage.php.

import (
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
)

// updateMemberData applies a set of column updates to a member.
// Values may be string/int/int64/float64, or sqlExpr for raw SQL fragments.
type sqlExpr string

func (a *App) updateMemberDataMap(memID int, vars map[string]any) {
	if len(vars) == 0 {
		return
	}
	var sets []string
	var args []any
	for col, v := range vars {
		if expr, ok := v.(sqlExpr); ok {
			sets = append(sets, col+" = "+string(expr))
		} else {
			sets = append(sets, col+" = ?")
			args = append(args, v)
		}
	}
	args = append(args, memID)
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET `+strings.Join(sets, ", ")+` WHERE ID_MEMBER = ?`), args...)
}

// pmDeleteMessages is deleteMessages($personal_messages, $folder, $owner)
// from PersonalMessage.php. pms == nil means all; folder "" means both.
func (c *Ctx) pmDeleteMessages(pms []int, folder string, owner []int) {
	a := c.App

	if owner == nil {
		owner = []int{c.User.ID}
	}
	if len(owner) == 0 {
		return
	}
	ownerIDs := make([]string, len(owner))
	for i, o := range owner {
		ownerIDs[i] = itoa(o)
	}
	ownerIn := strings.Join(ownerIDs, ", ")

	where := ""
	allMode := true
	if pms != nil {
		if len(pms) == 0 {
			return
		}
		ids := make([]string, 0, len(pms))
		for _, p := range uniqueInts(pms) {
			ids = append(ids, itoa(p))
		}
		where = `
			AND ID_PM IN (` + strings.Join(ids, ", ") + `)`
		allMode = false
	}

	if folder == "outbox" || folder == "" {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}personal_messages
			SET deletedBySender = 1
			WHERE ID_MEMBER_FROM IN (` + ownerIn + `)
				AND deletedBySender = 0` + where))
	}
	if folder != "outbox" || folder == "" {
		// Calculate the number of messages each member's gonna lose...
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, COUNT(*) AS numDeletedMessages, IIF(is_read & 1, 1, 0) AS is_read
			FROM {$db_prefix}pm_recipients
			WHERE ID_MEMBER IN (` + ownerIn + `)
				AND deleted = 0` + where + `
			GROUP BY ID_MEMBER, is_read`))
		if err == nil {
			type lossRow struct{ member, num, isRead int }
			var losses []lossRow
			for rows.Next() {
				var l lossRow
				rows.Scan(&l.member, &l.num, &l.isRead)
				losses = append(losses, l)
			}
			rows.Close()
			// ...And update the statistics accordingly - now including
			// unread messages!
			for _, l := range losses {
				vars := map[string]any{}
				if allMode {
					vars["instantMessages"] = 0
				} else {
					vars["instantMessages"] = sqlExpr("MAX(0, instantMessages - " + itoa(l.num) + ")")
				}
				if l.isRead == 0 {
					if allMode {
						vars["unreadMessages"] = 0
					} else {
						vars["unreadMessages"] = sqlExpr("MAX(0, unreadMessages - " + itoa(l.num) + ")")
					}
				}
				a.updateMemberDataMap(l.member, vars)

				// If this is the current member we need to make their
				// message count correct.
				if c.User.ID == l.member {
					c.User.Messages -= l.num
					if l.isRead == 0 {
						c.User.UnreadMessages -= l.num
					}
				}
			}
		}

		// Do the actual deletion.
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}pm_recipients
			SET deleted = 1
			WHERE ID_MEMBER IN (` + ownerIn + `)
				AND deleted = 0` + where))
	}

	// If sender and recipients all have deleted their message, it can be
	// removed.
	pmWhere := strings.ReplaceAll(where, "ID_PM", "pm.ID_PM")
	var removePMs []string
	rows, err := a.DB.Query(a.Q(`
		SELECT pm.ID_PM, MAX(pmr.ID_PM) AS recipient
		FROM {$db_prefix}personal_messages AS pm
			LEFT JOIN {$db_prefix}pm_recipients AS pmr ON (pmr.ID_PM = pm.ID_PM AND deleted = 0)
		WHERE pm.deletedBySender = 1
			` + pmWhere + `
		GROUP BY pm.ID_PM
		HAVING recipient IS NULL`))
	if err == nil {
		for rows.Next() {
			var pm int
			var recipient any
			rows.Scan(&pm, &recipient)
			removePMs = append(removePMs, itoa(pm))
		}
		rows.Close()
	}

	if len(removePMs) > 0 {
		in := strings.Join(removePMs, ", ")
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}personal_messages WHERE ID_PM IN (` + in + `)`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}pm_recipients WHERE ID_PM IN (` + in + `)`))
	}
}

// deleteMembers is deleteMembers($users) from Subs-Members.php.
func (c *Ctx) deleteMembers(users []int) {
	a := c.App

	// Make sure there's no void user in here.
	var clean []int
	for _, u := range uniqueInts(users) {
		if u != 0 {
			clean = append(clean, u)
		}
	}
	users = clean

	// How many are they deleting?
	if len(users) == 0 {
		return
	} else if len(users) == 1 {
		if users[0] == c.User.ID {
			c.isAllowedTo("profile_remove_own")
		} else {
			c.isAllowedTo("profile_remove_any")
		}
	} else {
		// Deleting more than one?  You can't have more than one account...
		c.isAllowedTo("profile_remove_any")
	}

	ids := func(us []int) string {
		ss := make([]string, len(us))
		for i, u := range us {
			ss[i] = itoa(u)
		}
		return strings.Join(ss, ", ")
	}

	// Make sure they aren't trying to delete administrators if they aren't
	// one.  But don't bother checking if it's just themself.
	if !c.allowedTo("admin_forum") && (len(users) != 1 || users[0] != c.User.ID) {
		admins := map[int]bool{}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}members
			WHERE ID_MEMBER IN (` + ids(users) + `)
				AND (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups) != 0)`))
		if err == nil {
			for rows.Next() {
				var m int
				rows.Scan(&m)
				admins[m] = true
			}
			rows.Close()
		}
		if len(admins) > 0 {
			var left []int
			for _, u := range users {
				if !admins[u] {
					left = append(left, u)
				}
			}
			users = left
		}
	}

	if len(users) == 0 {
		return
	}

	condition := "IN (" + ids(users) + ")"

	// Log the action - regardless of who is deleting it.
	for _, user := range users {
		c.logAction("delete_member", map[string]any{"member": user})
	}

	// Make these peoples' posts guest posts.
	emailClear := ""
	if !a.SettingEmpty("allow_hideEmail") {
		emailClear = ", posterEmail = ''"
	}
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_MEMBER = 0` + emailClear + ` WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}polls SET ID_MEMBER = 0 WHERE ID_MEMBER ` + condition))

	// Make these peoples' posts guest first posts and last posts.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_MEMBER_STARTED = 0 WHERE ID_MEMBER_STARTED ` + condition))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_MEMBER_UPDATED = 0 WHERE ID_MEMBER_UPDATED ` + condition))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_actions SET ID_MEMBER = 0 WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_banned SET ID_MEMBER = 0 WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_errors SET ID_MEMBER = 0 WHERE ID_MEMBER ` + condition))

	// Delete the member.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}members WHERE ID_MEMBER ` + condition))

	// Delete the logs...
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_boards WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_karma WHERE ID_TARGET ` + condition + ` OR ID_EXECUTOR ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_mark_read WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_notify WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_polls WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_MEMBER ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}collapsed_categories WHERE ID_MEMBER ` + condition))

	// Delete personal messages.
	c.pmDeleteMessages(nil, "", users)

	a.DB.Exec(a.Q(`UPDATE {$db_prefix}personal_messages SET ID_MEMBER_FROM = 0 WHERE ID_MEMBER_FROM ` + condition))

	// Delete avatar.
	a.removeAttachments("a.ID_MEMBER "+condition, "", false, true)

	// It's over, no more moderation for you.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}moderators WHERE ID_MEMBER ` + condition))

	// If you don't exist we can't ban you.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}ban_items WHERE ID_MEMBER ` + condition))

	// Remove individual theme settings.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}themes WHERE ID_MEMBER ` + condition))

	// These users are nobody's buddy nomore.
	var ignoreFinds, buddyFinds []string
	for _, u := range users {
		ignoreFinds = append(ignoreFinds, "FIND_IN_SET("+itoa(u)+", pm_ignore_list)")
		buddyFinds = append(buddyFinds, "FIND_IN_SET("+itoa(u)+", buddy_list)")
	}
	type fixRow struct {
		member             int
		ignoreList, buddys string
	}
	var fixes []fixRow
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, pm_ignore_list, buddy_list
		FROM {$db_prefix}members
		WHERE ` + strings.Join(ignoreFinds, " OR ") + ` OR ` + strings.Join(buddyFinds, " OR ")))
	if err == nil {
		for rows.Next() {
			var f fixRow
			rows.Scan(&f.member, &f.ignoreList, &f.buddys)
			fixes = append(fixes, f)
		}
		rows.Close()
	}
	removed := map[string]bool{}
	for _, u := range users {
		removed[itoa(u)] = true
	}
	filterList := func(list string) string {
		var keep []string
		for _, item := range strings.Split(list, ",") {
			if item != "" && !removed[item] {
				keep = append(keep, item)
			}
		}
		return strings.Join(keep, ",")
	}
	for _, f := range fixes {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}members
			SET
				pm_ignore_list = ?,
				buddy_list = ?
			WHERE ID_MEMBER = ?`), filterList(f.ignoreList), filterList(f.buddys), f.member)
	}

	// Make sure no member's birthday is still sticking in the calendar...
	a.updateStatsCalendar()
	c.updateStatsMemberRecount()
}

// resetPassword is resetPassword($memID, $username) from Subs-Auth.php.
func (c *Ctx) resetPassword(memID int, username string) {
	a := c.App
	scripturl := a.ScriptURL

	// Get some important details.
	var user, email string
	a.DB.QueryRow(a.Q(`SELECT memberName, emailAddress FROM {$db_prefix}members WHERE ID_MEMBER = ?`),
		memID).Scan(&user, &email)

	if username != "" {
		user = strings.TrimSpace(latin1NameClean(username))
	}

	// Generate a random password.
	newPassword := generateValidationCode()
	newPasswordSha1 := fmt.Sprintf("%x", sha1.Sum([]byte(strings.ToLower(user)+newPassword)))

	// Do some checks on the username if needed.
	if username != "" {
		// No name?!  How can you register with no name?
		if user == "" {
			c.fatalLangError("37", false)
		}

		// Only these characters are permitted.
		if user == "_" || user == "|" || regexp.MustCompile(`[<>&"'=\\]`).MatchString(user) ||
			strings.Contains(user, "[code") || strings.Contains(user, "[/code") {
			c.fatalLangError("240", false)
		}

		if strings.Contains(strings.ToLower(user), strings.ToLower(c.Txt("28"))) {
			c.fatalLangError("244", true, c.Txt("28"))
		}

		if c.isReservedName(user, memID, false, true) {
			c.fatalError("("+Htmlspecialchars(user)+") "+c.Txt("473"), false)
		}

		// Update the database...
		a.updateMemberDataMap(memID, map[string]any{"memberName": user, "passwd": newPasswordSha1})
	} else {
		a.updateMemberDataMap(memID, map[string]any{"passwd": newPasswordSha1})
	}

	// Send them the email informing them of the change - then we're done!
	c.sendmail([]string{email}, c.Txt("change_password"),
		c.Txt("hello_member")+" "+user+"!\n\n"+
			c.Txt("change_password_1")+" "+a.Config.MbName+" "+c.Txt("change_password_2")+"\n\n"+
			c.Txt("719")+user+", "+c.Txt("492")+" "+newPassword+"\n\n"+
			c.Txt("701")+"\n"+
			scripturl+"?action=profile\n\n"+
			c.Txt("130"), "")
}

// latin1NameClean strips the control/space junk PHP removes from names
// (the ISO-8859-1 branch: [\t\n\r \x0B\0\x00-\x08\x0B\x0C\x0E-\x19\xA0]).
func latin1NameClean(s string) string {
	var b strings.Builder
	prevSpace := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		junk := ch == '\t' || ch == '\n' || ch == '\r' || ch == ' ' || ch == 0x0B || ch == 0 ||
			ch <= 0x08 || ch == 0x0C || (ch >= 0x0E && ch <= 0x19) || ch == 0xA0
		if junk {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteByte(ch)
			prevSpace = false
		}
	}
	return b.String()
}

func init() {
	registerAction("buddy", (*Ctx).BuddyListToggle)
}

// BuddyListToggle is BuddyListToggle() (?action=buddy): flip a member in
// and out of the buddy list.
func (c *Ctx) BuddyListToggle() {
	a := c.App

	c.checkSession("get", "", true)

	c.isAllowedTo("profile_identity_own")
	c.isNotGuest("")

	if empty(c.REQUEST.Str("u")) {
		c.fatalLangError("1", false)
	}
	target := c.REQUEST.Int("u")

	// Remove if it's already there... or add if it's not.
	found := false
	var buddies []int
	for _, b := range c.User.Buddies {
		if b == target {
			found = true
			continue
		}
		buddies = append(buddies, b)
	}
	if !found {
		buddies = append(buddies, target)
	}
	c.User.Buddies = buddies

	// Update the settings.
	ids := make([]string, len(buddies))
	for i, b := range buddies {
		ids[i] = itoa(b)
	}
	a.updateMemberDataMap(c.User.ID, map[string]any{"buddy_list": strings.Join(ids, ",")})

	// Redirect back to the profile
	c.redirectExit("action=profile;u=" + itoa(target))
}
