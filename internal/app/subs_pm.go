package app

// sendpm() from Subs-Post.php, findMembers() from Subs-Auth.php and
// markMessages() from PersonalMessage.php. (pmDeleteMessages lives in
// subs_members2.go.)

import (
	"regexp"
	"strings"
	"time"
)

// pmFrom mirrors sendpm's $from.
type pmFrom struct {
	ID       int
	Name     string
	Username string
}

// pmLog is sendpm's return value.
type pmLog struct {
	Sent   []string
	Failed []string
}

// pmRecipients holds the to/bcc member IDs.
type pmRecipients struct {
	To  []int
	Bcc []int
}

// sendpm is sendpm($recipients, $subject, $message, $store_outbox, $from).
func (c *Ctx) sendpm(recipients *pmRecipients, subject, message string, storeOutbox bool, from *pmFrom) *pmLog {
	a := c.App
	scripturl := a.ScriptURL

	log := &pmLog{}

	if from == nil {
		from = &pmFrom{ID: c.User.ID, Name: c.User.Name, Username: c.User.Username}
	}

	// This is the one that will go in their inbox.
	htmlmessage := Htmlspecialchars(message)
	htmlsubject := Htmlspecialchars(subject)
	htmlmessage = c.preparsecode(htmlmessage, false)

	// Make sure there are no duplicate 'to' members.
	recipients.To = uniqueInts(recipients.To)

	// Only 'bcc' members that aren't already in 'to'.
	inTo := map[int]bool{}
	for _, t := range recipients.To {
		inTo[t] = true
	}
	var bcc []int
	for _, b := range uniqueInts(recipients.Bcc) {
		if !inTo[b] {
			bcc = append(bcc, b)
		}
	}
	recipients.Bcc = bcc
	isBcc := map[int]bool{}
	for _, b := range bcc {
		isBcc[b] = true
	}

	// Combine 'to' and 'bcc' recipients.
	allTo := append(append([]int{}, recipients.To...), recipients.Bcc...)
	if len(allTo) == 0 {
		return log
	}

	ids := make([]string, len(allTo))
	for i, t := range allTo {
		ids[i] = itoa(t)
	}

	ignoredExpr := "(mem.pm_ignore_list = '*' OR FIND_IN_SET(" + itoa(from.ID) + ", mem.pm_ignore_list))"
	if c.allowedTo("moderate_forum") {
		ignoredExpr = "0"
	}

	type recRow struct {
		memberName, realName, email, lngfile      string
		id, maxMessages, pmEmailNotify, im        int
		ignored, isBuddy, isActivated, isAdminInt int
	}
	var recRows []recRow
	rows, err := a.DB.Query(a.Q(`
		SELECT
			mem.memberName, mem.realName, mem.ID_MEMBER, mem.emailAddress, mem.lngfile, IFNULL(mg.maxMessages, 0),
			mem.pm_email_notify, mem.instantMessages, ` + ignoredExpr + ` AS ignored,
			FIND_IN_SET(` + itoa(from.ID) + `, mem.buddy_list) AS is_buddy, mem.is_activated,
			(mem.ID_GROUP = 1 OR FIND_IN_SET(1, mem.additionalGroups)) AS is_admin
		FROM {$db_prefix}members AS mem
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))
		WHERE mem.ID_MEMBER IN (` + strings.Join(ids, ", ") + `)
		ORDER BY mem.lngfile`))
	if err == nil {
		for rows.Next() {
			var r recRow
			rows.Scan(&r.memberName, &r.realName, &r.id, &r.email, &r.lngfile,
				&r.maxMessages, &r.pmEmailNotify, &r.im, &r.ignored, &r.isBuddy, &r.isActivated, &r.isAdminInt)
			recRows = append(recRows, r)
		}
		rows.Close()
	}

	removeFromAll := func(id int) {
		var left []int
		for _, t := range allTo {
			if t != id {
				left = append(left, t)
			}
		}
		allTo = left
	}

	var notifications []string
	for _, r := range recRows {
		// Has the receiver gone over their message limit?
		if r.maxMessages != 0 && r.maxMessages <= r.im && !c.allowedTo("moderate_forum") && r.isAdminInt == 0 {
			log.Failed = append(log.Failed, phpSprintf(c.Txt("pm_error_data_limit_reached"), r.realName))
			removeFromAll(r.id)
			continue
		}

		if r.ignored != 0 {
			log.Failed = append(log.Failed, phpSprintf(c.Txt("pm_error_ignored_by_user"), r.realName))
			removeFromAll(r.id)
			continue
		}

		// Send a notification, if enabled - taking into account buddy list!
		if r.email != "" && (r.pmEmailNotify == 1 || (r.pmEmailNotify > 1 && (r.isBuddy != 0 || !a.SettingEmpty("enable_buddylist")))) && r.isActivated == 1 {
			notifications = append(notifications, r.email)
		}

		log.Sent = append(log.Sent, phpSprintf(c.Txt("pm_successfully_sent"), r.realName))
	}

	// Only 'send' the message if there are any recipients left.
	if len(allTo) == 0 {
		return log
	}

	// Insert the message itself and then grab the last insert id.
	deletedBySender := 1
	if storeOutbox {
		deletedBySender = 0
	}
	res, err := a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}personal_messages
			(ID_MEMBER_FROM, deletedBySender, fromName, msgtime, subject, body)
		VALUES (?, ?, SUBSTR(?, 1, 255), ?, SUBSTR(?, 1, 255), SUBSTR(?, 1, 65534))`),
		from.ID, deletedBySender, from.Username, time.Now().Unix(), htmlsubject, htmlmessage)
	if err != nil {
		return log
	}
	idPM64, _ := res.LastInsertId()
	idPM := int(idPM64)

	// Add the recipients.
	if idPM != 0 {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}pm_recipients WHERE ID_PM = ?`), idPM)

		var values []string
		for _, to := range allTo {
			values = append(values, "("+itoa(idPM)+", "+itoa(to)+", "+itoa(boolToInt(isBcc[to]))+")")
		}
		a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}pm_recipients
				(ID_PM, ID_MEMBER, bcc)
			VALUES ` + strings.Join(values, `,
				`)))
	}

	message = c.censorText(message)
	subject = c.censorText(subject)
	message = strings.TrimSpace(unHtmlspecialchars(stripTags(strings.NewReplacer(
		"<br />", "\n", "</div>", "\n", "</li>", "\n", "&#91;", "[", "&#93;", "]",
	).Replace(c.parseBBC(Htmlspecialchars(message), false)))))

	if len(notifications) > 0 {
		// Replace the right things in the message strings.
		mailsubject := strings.NewReplacer("SUBJECT", subject, "SENDER", unHtmlspecialchars(from.Name)).Replace(c.Txt("561"))
		mailmessage := strings.NewReplacer("SUBJECT", subject, "MESSAGE", message, "SENDER", unHtmlspecialchars(from.Name)).Replace(c.Txt("562"))
		mailmessage += "\n\n" + c.Txt("instant_reply") + " " + scripturl + "?action=pm;sa=send;f=inbox;pmsg=" + itoa(idPM) + ";quote;u=" + itoa(from.ID)

		// Off the notification email goes!
		c.sendmail(notifications, mailsubject, mailmessage, "")
	}

	// Add one to their unread and read message counts.
	for _, to := range allTo {
		a.updateMemberDataMap(to, map[string]any{
			"instantMessages": sqlExpr("instantMessages + 1"),
			"unreadMessages":  sqlExpr("unreadMessages + 1"),
		})
	}

	return log
}

// foundMember is one findMembers result.
type foundMember struct {
	ID       int
	Name     string
	Username string
	Email    string
	Href     string
	Link     string
}

// findMembers is findMembers($names, $use_wildcards, $buddies_only, $max).
func (c *Ctx) findMembers(names []string, useWildcards, buddiesOnly bool, max int) map[int]*foundMember {
	a := c.App
	scripturl := a.ScriptURL

	maybeEmail := false
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		n := strings.TrimSpace(strings.ToLower(name))
		if strings.Contains(n, "@") {
			maybeEmail = true
		}
		if useWildcards {
			n = strings.NewReplacer("%", `\%`, "_", `\_`, "*", "%", "?", "_", `\'`, "&#039;").Replace(n)
		} else {
			n = strings.ReplaceAll(n, `\'`, "&#039;")
		}
		cleaned = append(cleaned, strings.ReplaceAll(n, "'", "''"))
	}
	if len(cleaned) == 0 {
		return nil
	}

	comparison := "="
	escapeSuffix := ""
	if useWildcards {
		comparison = "LIKE"
		escapeSuffix = ` ESCAPE '\'`
	}

	// This ensures you can't search someones email address if you can't see
	// it.
	emailGuard := ""
	if !c.User.IsAdmin && !a.SettingEmpty("allow_hideEmail") {
		emailGuard = "hideEmail = 0 AND "
	}

	cond := func(col string) string {
		var parts []string
		for _, n := range cleaned {
			parts = append(parts, col+" "+comparison+" '"+n+"'"+escapeSuffix)
		}
		return strings.Join(parts, " OR ")
	}

	emailCondition := ""
	if useWildcards || maybeEmail {
		var parts []string
		for _, n := range cleaned {
			parts = append(parts, "("+emailGuard+"emailAddress "+comparison+" '"+n+"'"+escapeSuffix+")")
		}
		emailCondition = `
			OR ` + strings.Join(parts, " OR ")
	}

	buddiesCond := ""
	if buddiesOnly {
		ids := make([]string, len(c.User.Buddies))
		for i, b := range c.User.Buddies {
			ids[i] = itoa(b)
		}
		buddiesCond = "AND ID_MEMBER IN (" + strings.Join(ids, ", ") + ")"
	}
	limit := ""
	if max > 0 {
		limit = `
		LIMIT ` + itoa(max)
	}

	results := map[int]*foundMember{}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, memberName, realName, emailAddress, hideEmail
		FROM {$db_prefix}members
		WHERE (` + cond("memberName") + `
			OR ` + cond("realName") + emailCondition + `)
			` + buddiesCond + `
			AND is_activated IN (1, 11)` + limit))
	if err == nil {
		for rows.Next() {
			var id, hideEmail int
			var memberName, realName, email string
			rows.Scan(&id, &memberName, &realName, &email, &hideEmail)

			showEmail := email
			if hideEmail != 0 && !a.SettingEmpty("allow_hideEmail") && !c.User.IsAdmin {
				showEmail = ""
			}
			results[id] = &foundMember{
				ID:       id,
				Name:     realName,
				Username: memberName,
				Email:    showEmail,
				Href:     scripturl + "?action=profile;u=" + itoa(id),
				Link:     `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + realName + `</a>`,
			}
		}
		rows.Close()
	}
	return results
}

// pmNameClean strips [<>&"'=\] like sendpm does for non-numeric names.
var pmNameCleanRe = regexp.MustCompile(`[<>&"'=\\]`)

// markMessages is markMessages($personal_messages, $label, $owner). Pass
// pms == nil for all, label == "" for all labels.
func (c *Ctx) markMessages(pms []int, label string, owner int, labels map[int]*PMLabel) {
	a := c.App

	if owner == 0 {
		owner = c.User.ID
	}

	labelCond := ""
	if label != "" {
		labelCond = `
			AND FIND_IN_SET(` + label + `, labels)`
	}
	pmCond := ""
	if pms != nil {
		ids := make([]string, len(pms))
		for i, p := range pms {
			ids[i] = itoa(p)
		}
		pmCond = `
			AND ID_PM IN (` + strings.Join(ids, ", ") + `)`
	}

	res, err := a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}pm_recipients
		SET is_read = is_read | 1
		WHERE ID_MEMBER = ` + itoa(owner) + `
			AND NOT (is_read & 1)` + labelCond + pmCond))

	if owner == c.User.ID && labels != nil {
		for _, l := range labels {
			l.UnreadMessages = 0
		}
	}

	// If something wasn't marked as read, get the number of unread messages
	// remaining.
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			totalUnread := 0
			rows, err := a.DB.Query(a.Q(`
				SELECT labels, COUNT(*) AS num
				FROM {$db_prefix}pm_recipients
				WHERE ID_MEMBER = ?
					AND NOT (is_read & 1)
				GROUP BY labels`), owner)
			if err == nil {
				for rows.Next() {
					var rowLabels string
					var num int
					rows.Scan(&rowLabels, &num)
					totalUnread += num

					if owner != c.User.ID || labels == nil {
						continue
					}
					for _, thisLabel := range strings.Split(rowLabels, ",") {
						if l, ok := labels[atoi(thisLabel)]; ok {
							l.UnreadMessages += num
						}
					}
				}
				rows.Close()
			}

			a.updateMemberDataMap(owner, map[string]any{"unreadMessages": totalUnread})

			// If it was for the current member, reflect this in $user_info
			// too.
			if owner == c.User.ID {
				c.User.UnreadMessages = totalUnread
			}
		}
	}
}
