package app

// Port of Sources/Profile.php part 2: the view subactions — summary,
// showPosts, editBuddies, statPanel, trackUser, TrackIP, showPermissions.

import (
	"math"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ProfileSummaryCtx is the extra data for the summary template.
type ProfileSummaryCtx struct {
	AllowHideEmail   bool
	CanSendPM        bool
	CanHaveBuddy     bool
	PostsPerDay      string
	Age              string
	TodayIsBirthday  bool
	Hostname         string
	CanSeeIP         bool
	ActivateType     int
	ActivateLinkText string
	ActivateMessage  string
	Bans             []ProfileBan
	CanEditBan       bool
}

// ProfileBan is one active ban on the summary page.
type ProfileBan struct {
	Reason      string
	Explanation string
}

// profileSummary is summary($memID).
func (c *Ctx) profileSummary(page *ProfileCtx, memID int) {
	a := c.App
	scripturl := a.ScriptURL

	// Attempt to load the member's profile data.
	if !c.loadMemberContext(memID) {
		c.fatalError(c.Txt("453")+" - "+itoa(memID), false)
	}
	member := c.memberCtx[memID]
	page.MemberFull = member

	sub := &ProfileSummaryCtx{
		AllowHideEmail: !a.SettingEmpty("allow_hideEmail"),
		CanSendPM:      c.allowedTo("pm_send"),
		CanHaveBuddy:   c.allowedTo("profile_identity_own") && !a.SettingEmpty("enable_buddylist"),
	}
	page.Sub = sub
	c.PageTitle = c.Txt("92") + " " + member.Name

	profile := c.memberProfiles[memID]

	// They haven't even been registered for a full day!?
	daysRegistered := int((time.Now().Unix() - profile.DateRegistered) / (3600 * 24))
	if profile.DateRegistered == 0 || daysRegistered < 1 {
		sub.PostsPerDay = c.Txt("470")
	} else {
		sub.PostsPerDay = c.commaFormatFloat(float64(member.RealPosts)/float64(daysRegistered), 3)
	}

	// Set the age...
	if member.BirthDate == "" {
		sub.Age = c.Txt("470")
	} else {
		var birthYear, birthMonth, birthDay int
		parts := strings.SplitN(member.BirthDate, "-", 3)
		if len(parts) == 3 {
			birthYear, birthMonth, birthDay = atoi(parts[0]), atoi(parts[1]), atoi(parts[2])
		}
		now := time.Unix(c.forumTime(false, 0), 0)
		if birthYear <= 4 {
			sub.Age = c.Txt("470")
		} else {
			age := now.Year() - birthYear
			if !(int(now.Month()) > birthMonth || (int(now.Month()) == birthMonth && now.Day() >= birthDay)) {
				age--
			}
			sub.Age = itoa(age)
		}
		sub.TodayIsBirthday = int(now.Month()) == birthMonth && now.Day() == birthDay
	}

	ipRe := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	if c.allowedTo("moderate_forum") {
		// Make sure it's a valid ip address; otherwise, don't bother...
		if ipRe.MatchString(member.IP) && a.SettingEmpty("disableHostnameLookup") {
			if names, err := net.LookupAddr(member.IP); err == nil && len(names) > 0 {
				sub.Hostname = strings.TrimSuffix(names[0], ".")
			}
		}
		sub.CanSeeIP = true
	}

	// (who_enabled action display ports with Phase 5/Who.php.)

	// If the user is awaiting activation, and the viewer has permission -
	// setup some activation context messages.
	if member.IsActivated%10 != 1 && c.allowedTo("moderate_forum") {
		sub.ActivateType = member.IsActivated
		// What should the link text be?
		switch member.IsActivated {
		case 3, 4, 5, 13, 14, 15:
			sub.ActivateLinkText = c.Txt("account_approve")
		default:
			sub.ActivateLinkText = c.Txt("account_activate")
		}

		// Should we show a custom message?
		key := "account_activate_method_" + itoa(member.IsActivated%10)
		if msg := c.Txt(key); msg != key {
			sub.ActivateMessage = msg
		} else {
			sub.ActivateMessage = c.Txt("account_not_activated")
		}
	}

	// How about, are they banned?
	if c.allowedTo("moderate_forum") {
		// Can they edit the ban?
		sub.CanEditBan = c.allowedTo("manage_bans")

		banQuery := []string{"ID_MEMBER = " + itoa(member.ID)}

		// Valid IP?
		if m := regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`).FindStringSubmatch(member.IP); m != nil {
			banQuery = append(banQuery, "(("+m[1]+" BETWEEN bi.ip_low1 AND bi.ip_high1)"+`
						AND (`+m[2]+` BETWEEN bi.ip_low2 AND bi.ip_high2)
						AND (`+m[3]+` BETWEEN bi.ip_low3 AND bi.ip_high3)
						AND (`+m[4]+` BETWEEN bi.ip_low4 AND bi.ip_high4))`)

			// Do we have a hostname already?
			if sub.Hostname != "" {
				banQuery = append(banQuery, "('"+strings.ReplaceAll(sub.Hostname, "'", "''")+"' LIKE hostname)")
			}
		} else if member.IP == "unknown" {
			banQuery = append(banQuery, `(bi.ip_low1 = 255 AND bi.ip_high1 = 255
						AND bi.ip_low2 = 255 AND bi.ip_high2 = 255
						AND bi.ip_low3 = 255 AND bi.ip_high3 = 255
						AND bi.ip_low4 = 255 AND bi.ip_high4 = 255)`)
		}

		// Check their email as well...
		if member.Email != "" {
			banQuery = append(banQuery, "('"+strings.ReplaceAll(member.Email, "'", "''")+"' LIKE bi.email_address)")
		}

		// So... are they banned?  Dying to know!
		rows, err := a.DB.Query(a.Q(`
			SELECT bg.ID_BAN_GROUP, bg.name, bg.cannot_access, bg.cannot_post, bg.cannot_register,
				bg.cannot_login, bg.reason
			FROM {$db_prefix}ban_items AS bi, {$db_prefix}ban_groups AS bg
			WHERE bg.ID_BAN_GROUP = bi.ID_BAN_GROUP
				AND (bg.expire_time IS NULL OR bg.expire_time > ` + i64toa(time.Now().Unix()) + `)
				AND (` + strings.Join(banQuery, " OR ") + `)
			GROUP BY bg.ID_BAN_GROUP`))
		if err == nil {
			for rows.Next() {
				var banGroup, cannotAccess, cannotPost, cannotRegister, cannotLogin int
				var name, reason string
				rows.Scan(&banGroup, &name, &cannotAccess, &cannotPost, &cannotRegister, &cannotLogin, &reason)

				// Work out what restrictions we actually have.
				var restrictions []string
				for _, t := range []struct {
					on  int
					key string
				}{{cannotAccess, "access"}, {cannotRegister, "register"}, {cannotLogin, "login"}, {cannotPost, "post"}} {
					if t.on != 0 {
						restrictions = append(restrictions, c.Txt("ban_type_"+t.key))
					}
				}

				// No actual ban in place?
				if len(restrictions) == 0 {
					continue
				}

				// Prepare the link for context.
				explanation := phpSprintf(c.Txt("user_cannot_due_to"), strings.Join(restrictions, ", "),
					`<a href="`+scripturl+`?action=ban;sa=edit;bg=`+itoa(banGroup)+`">`+name+`</a>`)

				banReason := ""
				if reason != "" {
					banReason = `<br /><br /><b>` + c.Txt("ban_reason") + `:</b> ` + reason
				}
				sub.Bans = append(sub.Bans, ProfileBan{Reason: banReason, Explanation: explanation})
			}
			rows.Close()
		}
	}

	c.SubTemplate = templateProfileSummary
}

// ProfilePost is one post on the showPosts page.
type ProfilePost struct {
	Body          string
	Counter       int
	CategoryName  string
	CategoryID    int
	BoardName     string
	BoardID       int
	Topic         int
	Subject       string
	Start         string
	Time          string
	ID            int
	CanReply      bool
	CanMarkNotify bool
	CanDelete     bool
}

// ProfileShowPostsCtx is the data for the showPosts template.
type ProfileShowPostsCtx struct {
	PageIndex string
	Start     int
	Posts     []*ProfilePost
}

// profileShowPosts is showPosts($memID).
func (c *Ctx) profileShowPosts(page *ProfileCtx, memID int) {
	a := c.App
	scripturl := a.ScriptURL

	// If just deleting a message, do it and then redirect back.
	if c.GET.Has("delete") {
		c.checkSession("get", "", true)

		// We can be lazy, since removeMessage() will check the permissions
		// for us.
		c.removeMessage(c.GET.Int("delete"), true)

		// Back to... where we are now ;).
		c.redirectExit("action=profile;u=" + itoa(memID) + ";sa=showPosts;start=" + c.GET.Str("start"))
	}

	sub := &ProfileShowPostsCtx{}
	page.Sub = sub

	profile := c.memberProfiles[memID]

	var msgCount int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}messages AS m, {$db_prefix}boards AS b
		WHERE m.ID_MEMBER = ?
			AND b.ID_BOARD = m.ID_BOARD
			AND `+c.User.QuerySeeBoard), memID).Scan(&msgCount)

	maxIndex := a.SettingInt("defaultMaxMessages")

	// Make sure the starting place makes sense and construct our friend the
	// page index.
	sub.Start = c.Start
	sub.PageIndex, _ = c.constructPageIndex(scripturl+"?action=profile;u="+itoa(memID)+";sa=showPosts",
		sub.Start, msgCount, maxIndex, false)

	// Reverse the query if we're past 50% of the pages for better
	// performance.
	start := sub.Start
	reverse := sub.Start > msgCount/2
	if reverse {
		if msgCount < sub.Start+a.SettingInt("defaultMaxMessages")+1 && msgCount > sub.Start {
			maxIndex = msgCount - sub.Start
		}
		if msgCount < sub.Start+a.SettingInt("defaultMaxMessages")+1 || msgCount < sub.Start+a.SettingInt("defaultMaxMessages") {
			start = 0
		} else {
			start = msgCount - sub.Start - a.SettingInt("defaultMaxMessages")
		}
	}

	c.PageTitle = c.Txt("458") + " " + profile.RealName

	order := "DESC"
	if reverse {
		order = "ASC"
	}

	// Find this user's posts.
	type postRow struct {
		boardID                      int
		bname                        string
		catID                        int
		cname                        string
		topicID, msgID               int
		starterID, firstMsg, lastMsg int
		body, subject                string
		smileysEnabled               int
		posterTime                   int64
	}
	var rowsData []postRow
	rows, err := a.DB.Query(a.Q(`
		SELECT
			b.ID_BOARD, b.name AS bname, IFNULL(c.ID_CAT, 0), IFNULL(c.name, '') AS cname, m.ID_TOPIC, m.ID_MSG,
			t.ID_MEMBER_STARTED, t.ID_FIRST_MSG, t.ID_LAST_MSG, m.body, m.smileysEnabled,
			m.subject, m.posterTime
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t, {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE m.ID_MEMBER = ` + itoa(memID) + `
			AND m.ID_TOPIC = t.ID_TOPIC
			AND t.ID_BOARD = b.ID_BOARD
			AND ` + c.User.QuerySeeBoard + `
		ORDER BY m.ID_MSG ` + order + `
		LIMIT ` + itoa(maxIndex) + ` OFFSET ` + itoa(start)))
	if err == nil {
		for rows.Next() {
			var r postRow
			rows.Scan(&r.boardID, &r.bname, &r.catID, &r.cname, &r.topicID, &r.msgID,
				&r.starterID, &r.firstMsg, &r.lastMsg, &r.body, &r.smileysEnabled,
				&r.subject, &r.posterTime)
			rowsData = append(rowsData, r)
		}
		rows.Close()
	}

	// Start counting at the number of the first message displayed.
	counter := sub.Start
	if reverse {
		counter = sub.Start + maxIndex + 1
	}
	deletePossibleMap := map[*ProfilePost]bool{}
	boardIDs := map[string]map[int][]*ProfilePost{"own": {}, "any": {}}
	for _, r := range rowsData {
		// Censor....
		r.body = c.censorText(r.body)
		r.subject = c.censorText(r.subject)

		// Do the code.
		r.body = c.parseBBCCached(r.body, r.smileysEnabled != 0, itoa(r.msgID))

		if reverse {
			counter--
		} else {
			counter++
		}

		deletePossible := (r.firstMsg != r.msgID || r.lastMsg == r.msgID) &&
			(a.SettingEmpty("edit_disable_time") || r.posterTime+int64(a.SettingInt("edit_disable_time"))*60 >= time.Now().Unix())

		post := &ProfilePost{
			Body:         r.body,
			Counter:      counter,
			CategoryName: r.cname,
			CategoryID:   r.catID,
			BoardName:    r.bname,
			BoardID:      r.boardID,
			Topic:        r.topicID,
			Subject:      r.subject,
			Start:        "msg" + itoa(r.msgID),
			Time:         c.timeformat(r.posterTime),
			ID:           r.msgID,
		}
		sub.Posts = append(sub.Posts, post)

		if c.User.ID == r.starterID {
			boardIDs["own"][r.boardID] = append(boardIDs["own"][r.boardID], post)
		}
		boardIDs["any"][r.boardID] = append(boardIDs["any"][r.boardID], post)

		deletePossibleMap[post] = deletePossible
	}

	// All posts were retrieved in reverse order, get them right again.
	if reverse {
		for i, j := 0, len(sub.Posts)-1; i < j; i, j = i+1, j-1 {
			sub.Posts[i], sub.Posts[j] = sub.Posts[j], sub.Posts[i]
		}
	}

	// These are all the permissions that are different from board to board..
	permissions := map[string]map[string]string{
		"own": {
			"post_reply_own": "can_reply",
			"delete_own":     "can_delete",
		},
		"any": {
			"post_reply_any":  "can_reply",
			"mark_any_notify": "can_mark_notify",
			"delete_any":      "can_delete",
		},
	}

	for typ, list := range permissions {
		for permission, allowed := range list {
			boards := c.boardsAllowedTo(permission)

			// Hmm, they can do it on all boards, can they?
			if len(boards) > 0 && boards[0] == 0 {
				boards = nil
				for b := range boardIDs[typ] {
					boards = append(boards, b)
				}
			}

			for _, boardID := range boards {
				for _, post := range boardIDs[typ][boardID] {
					switch allowed {
					case "can_reply":
						post.CanReply = true
					case "can_mark_notify":
						post.CanMarkNotify = true
					case "can_delete":
						post.CanDelete = true
					}
				}
			}
		}
	}

	// Clean up after posts that cannot be deleted.
	for _, post := range sub.Posts {
		if !deletePossibleMap[post] {
			post.CanDelete = false
		}
	}

	c.SubTemplate = templateProfileShowPosts
}

// ProfileBuddiesCtx is the data for the editBuddies template.
type ProfileBuddiesCtx struct {
	BuddyCount int
	Buddies    []*MemberCtx
}

// profileEditBuddies is editBuddies($memID).
func (c *Ctx) profileEditBuddies(page *ProfileCtx, memID int) {
	a := c.App

	// Do a quick check to ensure people aren't getting here illegally!
	if !page.IsOwner || a.SettingEmpty("enable_buddylist") {
		c.fatalLangError("1", false)
	}

	profile := c.memberProfiles[memID]

	// For making changes!
	var buddiesArray []int
	for _, b := range strings.Split(profile.BuddyList, ",") {
		if b != "" {
			buddiesArray = append(buddiesArray, atoi(b))
		}
	}

	// Removing a buddy?
	if c.GET.Has("remove") {
		removeID := c.GET.Int("remove")
		var kept []int
		for _, b := range buddiesArray {
			if b != removeID {
				kept = append(kept, b)
			}
		}
		// Make the changes.
		ids := make([]string, len(kept))
		for i, b := range kept {
			ids[i] = itoa(b)
		}
		a.updateMemberDataMap(memID, map[string]any{"buddy_list": strings.Join(ids, ",")})

		// Redirect off the page because we don't like all this ugly query
		// stuff to stick in the history.
		c.redirectExit("action=profile;u=" + itoa(memID) + ";sa=editBuddies")
	} else if c.POST.Has("new_buddy") {
		// Prepare the string for extraction...
		newBuddy := strings.ReplaceAll(Htmlspecialchars(c.POST.Str("new_buddy")), "&quot;", `"`)
		quoted := regexp.MustCompile(`"([^"]+)"`)
		var newBuddies []string
		seen := map[string]bool{}
		for _, m := range quoted.FindAllStringSubmatch(newBuddy, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				newBuddies = append(newBuddies, m[1])
			}
		}
		for _, b := range strings.Split(quoted.ReplaceAllString(newBuddy, ""), ",") {
			b = strings.TrimSpace(b)
			if b != "" && !seen[b] {
				seen[b] = true
				newBuddies = append(newBuddies, b)
			}
		}

		if len(newBuddies) > 0 {
			// Now find out the ID_MEMBER of the buddy.
			quotedNames := make([]string, len(newBuddies))
			for i, b := range newBuddies {
				quotedNames[i] = "'" + strings.ReplaceAll(b, "'", "''") + "'"
			}
			in := strings.Join(quotedNames, ",")
			rows, err := a.DB.Query(a.Q(`
				SELECT ID_MEMBER
				FROM {$db_prefix}members
				WHERE memberName IN (` + in + `) OR realName IN (` + in + `)`))
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					buddiesArray = append(buddiesArray, id)
				}
				rows.Close()
			}

			// Now update the current users buddy list.
			ids := make([]string, len(buddiesArray))
			for i, b := range buddiesArray {
				ids[i] = itoa(b)
			}
			a.updateMemberDataMap(memID, map[string]any{"buddy_list": strings.Join(ids, ",")})
		}

		// Back to the buddy list!
		c.redirectExit("action=profile;u=" + itoa(memID) + ";sa=editBuddies")
	}

	// Get all the users "buddies"...
	var buddies []int
	if len(buddiesArray) > 0 {
		ids := make([]string, len(buddiesArray))
		for i, b := range buddiesArray {
			ids[i] = itoa(b)
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}members
			WHERE ID_MEMBER IN (` + strings.Join(ids, ", ") + `)
			ORDER BY realName`))
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				buddies = append(buddies, id)
			}
			rows.Close()
		}
	}

	sub := &ProfileBuddiesCtx{BuddyCount: len(buddies)}
	page.Sub = sub

	// Load all the members up.
	c.loadMemberDataSet(buddies, "", "profile")

	// Setup the context for each buddy.
	for _, buddy := range buddies {
		if c.loadMemberContext(buddy) {
			sub.Buddies = append(sub.Buddies, c.memberCtx[buddy])
		}
	}

	c.SubTemplate = templateProfileEditBuddies
}

// ProfileStatsBoard is one board entry on the stats panel.
type ProfileStatsBoard struct {
	ID           int
	Posts        int
	Href         string
	Link         string
	PostsPercent string
	TotalPosts   int
	Percent      float64
}

// ProfileStatsHour is one hour bar.
type ProfileStatsHour struct {
	Hour         int
	PostsPercent int
}

// ProfileStatPanelCtx is the data for the statPanel template.
type ProfileStatPanelCtx struct {
	TimeLoggedIn  string
	NumPosts      string
	NumTopics     string
	NumPolls      string
	NumVotes      string
	PopularBoards []*ProfileStatsBoard
	BoardActivity []*ProfileStatsBoard
	PostsByTime   []ProfileStatsHour
}

// profileStatPanel is statPanel($memID).
func (c *Ctx) profileStatPanel(page *ProfileCtx, memID int) {
	a := c.App
	scripturl := a.ScriptURL

	profile := c.memberProfiles[memID]
	sub := &ProfileStatPanelCtx{}
	page.Sub = sub

	c.PageTitle = c.Txt("statPanel_showStats") + " " + profile.RealName

	// General user statistics.
	timeDays := profile.TotalTimeLoggedIn / 86400
	timeHours := (profile.TotalTimeLoggedIn % 86400) / 3600
	if timeDays > 0 {
		sub.TimeLoggedIn += i64toa(timeDays) + c.Txt("totalTimeLogged2")
	}
	if timeHours > 0 {
		sub.TimeLoggedIn += i64toa(timeHours) + c.Txt("totalTimeLogged3")
	}
	sub.TimeLoggedIn += i64toa((profile.TotalTimeLoggedIn%3600)/60) + c.Txt("totalTimeLogged4")
	sub.NumPosts = c.commaFormat(profile.Posts)

	recycleCond := ""
	if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
		recycleCond = `
			AND ID_BOARD != ` + itoa(a.SettingInt("recycle_board"))
	}

	// Number of topics started.
	var numTopics, numPolls, numVotes int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}topics
		WHERE ID_MEMBER_STARTED = ?`+recycleCond), memID).Scan(&numTopics)

	// Number polls started.
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}topics
		WHERE ID_MEMBER_STARTED = ?`+recycleCond+`
			AND ID_POLL != 0`), memID).Scan(&numPolls)

	// Number polls voted in.
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(DISTINCT ID_POLL)
		FROM {$db_prefix}log_polls
		WHERE ID_MEMBER = ?`), memID).Scan(&numVotes)

	// Format the numbers...
	sub.NumTopics = c.commaFormat(numTopics)
	sub.NumPolls = c.commaFormat(numPolls)
	sub.NumVotes = c.commaFormat(numVotes)

	// Grab the board this member posted in most often.
	rows, err := a.DB.Query(a.Q(`
		SELECT
			b.ID_BOARD, b.name, b.numPosts, COUNT(*) AS messageCount
		FROM {$db_prefix}messages AS m, {$db_prefix}boards AS b
		WHERE m.ID_MEMBER = ` + itoa(memID) + `
			AND b.ID_BOARD = m.ID_BOARD
			AND ` + c.User.QuerySeeBoard + `
		GROUP BY b.ID_BOARD
		ORDER BY messageCount DESC
		LIMIT 10`))
	if err == nil {
		for rows.Next() {
			var id, numPosts, messageCount int
			var name string
			rows.Scan(&id, &name, &numPosts, &messageCount)
			sub.PopularBoards = append(sub.PopularBoards, &ProfileStatsBoard{
				ID:           id,
				Posts:        messageCount,
				Href:         scripturl + "?board=" + itoa(id) + ".0",
				Link:         `<a href="` + scripturl + `?board=` + itoa(id) + `.0">` + name + `</a>`,
				PostsPercent: "0",
				TotalPosts:   numPosts,
			})
		}
		rows.Close()
	}

	// Now that we know the total, calculate the percentage.
	for _, b := range sub.PopularBoards {
		if b.TotalPosts == 0 {
			b.PostsPercent = "0"
		} else {
			b.PostsPercent = c.commaFormatFloat(float64(b.Posts*100)/float64(b.TotalPosts), 2)
		}
	}

	// Now get the 10 boards this user has most often participated in.
	rows, err = a.DB.Query(a.Q(`
		SELECT
			b.ID_BOARD, b.name, IIF(COUNT(*) > b.numPosts, 1, CAST(COUNT(*) AS REAL) / b.numPosts) * 100 AS percentage
		FROM {$db_prefix}messages AS m, {$db_prefix}boards AS b
		WHERE m.ID_MEMBER = ` + itoa(memID) + `
			AND b.ID_BOARD = m.ID_BOARD
			AND ` + c.User.QuerySeeBoard + `
		GROUP BY b.ID_BOARD
		ORDER BY percentage DESC
		LIMIT 10`))
	if err == nil {
		for rows.Next() {
			var id int
			var name string
			var percentage float64
			rows.Scan(&id, &name, &percentage)
			sub.BoardActivity = append(sub.BoardActivity, &ProfileStatsBoard{
				ID:      id,
				Href:    scripturl + "?board=" + itoa(id) + ".0",
				Link:    `<a href="` + scripturl + `?board=` + itoa(id) + `.0">` + name + `</a>`,
				Percent: percentage,
			})
		}
		rows.Close()
	}

	// Posting activity by time.
	offset := int64((c.User.TimeOffset + float64(a.SettingInt("time_offset"))) * 3600)
	hourly := map[int]int{}
	rows, err = a.DB.Query(a.Q(`
		SELECT
			CAST(strftime('%H', posterTime + ` + i64toa(offset) + `, 'unixepoch') AS INTEGER) AS hour,
			COUNT(*) AS postCount
		FROM {$db_prefix}messages
		WHERE ID_MEMBER = ` + itoa(memID) + `
		GROUP BY hour`))
	maxPosts := 0
	if err == nil {
		for rows.Next() {
			var hour, count int
			rows.Scan(&hour, &count)
			hourly[hour] = count
			if count > maxPosts {
				maxPosts = count
			}
		}
		rows.Close()
	}

	if maxPosts > 0 {
		var hours []int
		for h := range hourly {
			hours = append(hours, h)
		}
		for hour := 0; hour < 24; hour++ {
			if _, ok := hourly[hour]; !ok {
				sub.PostsByTime = append(sub.PostsByTime, ProfileStatsHour{Hour: hour})
			} else {
				sub.PostsByTime = append(sub.PostsByTime, ProfileStatsHour{
					Hour:         hour,
					PostsPercent: int(math.Round(float64(hourly[hour]*100) / float64(maxPosts))),
				})
			}
		}
	} else {
		for hour := range hourly {
			sub.PostsByTime = append(sub.PostsByTime, ProfileStatsHour{Hour: hour, PostsPercent: hourly[hour]})
		}
		sort.Slice(sub.PostsByTime, func(i, j int) bool { return sub.PostsByTime[i].Hour < sub.PostsByTime[j].Hour })
	}

	c.SubTemplate = templateProfileStatPanel
}

// ProfileTrackError is one error-log row.
type ProfileTrackError struct {
	IP         string
	MemberID   int
	MemberName string
	MemberLink string
	Message    string
	URL        string
	Time       string
}

// ProfileTrackMessage is one message row on trackIP.
type ProfileTrackMessage struct {
	IP         string
	MemberID   int
	MemberLink string
	BoardID    int
	BoardHref  string
	Topic      int
	ID         int
	Subject    string
	Time       string
}

// ProfileTrackCtx is the data for trackUser/trackIP templates.
type ProfileTrackCtx struct {
	LastIP           string
	MemberName       string
	IPs              []string
	ErrorIPs         []string
	PageIndex        string
	Start            int
	ErrorMessages    []ProfileTrackError
	MembersInRange   []string
	IP               string
	SingleIP         bool
	MessageStart     int
	ErrorStart       int
	MessagePageIndex string
	ErrorPageIndex   string
	IPMembers        [][2]string // ip, links
	Messages         []ProfileTrackMessage
	WhoisServers     []ProfileWhois
	AutoWhois        *ProfileWhois
}

// ProfileWhois is one whois server link.
type ProfileWhois struct {
	Name string
	URL  string
}

// profileTrackUser is trackUser($memID).
func (c *Ctx) profileTrackUser(page *ProfileCtx, memID int) {
	a := c.App
	scripturl := a.ScriptURL

	// Verify if the user has sufficient permissions.
	c.isAllowedTo("moderate_forum")

	profile := c.memberProfiles[memID]
	sub := &ProfileTrackCtx{}
	page.Sub = sub

	c.PageTitle = c.Txt("trackUser") + " - " + profile.RealName

	sub.LastIP = profile.MemberIP
	sub.MemberName = profile.RealName

	// Default to at least the ones we know about.
	ips := []string{profile.MemberIP, profile.MemberIP2}

	// Get all IP addresses this user has used for his messages.
	rows, err := a.DB.Query(a.Q(`
		SELECT posterIP
		FROM {$db_prefix}messages
		WHERE ID_MEMBER = ?
		GROUP BY posterIP`), memID)
	if err == nil {
		for rows.Next() {
			var ip string
			rows.Scan(&ip)
			sub.IPs = append(sub.IPs, `<a href="`+scripturl+`?action=trackip;searchip=`+ip+`">`+ip+`</a>`)
			ips = append(ips, ip)
		}
		rows.Close()
	}

	// Now also get the IP addresses from the error messages.
	totalErrors := 0
	rows, err = a.DB.Query(a.Q(`
		SELECT COUNT(*) AS errorCount, ip
		FROM {$db_prefix}log_errors
		WHERE ID_MEMBER = ?
		GROUP BY ip`), memID)
	if err == nil {
		for rows.Next() {
			var count int
			var ip string
			rows.Scan(&count, &ip)
			sub.ErrorIPs = append(sub.ErrorIPs, `<a href="`+scripturl+`?action=trackip;searchip=`+ip+`">`+ip+`</a>`)
			ips = append(ips, ip)
			totalErrors += count
		}
		rows.Close()
	}

	// Create the page indexes.
	sub.PageIndex, _ = c.constructPageIndex(scripturl+"?action=profile;u="+itoa(memID)+";sa=trackUser",
		c.Start, totalErrors, 20, false)
	sub.Start = c.Start

	// Get a list of error messages from this ip (range).
	rows, err = a.DB.Query(a.Q(`
		SELECT
			le.logTime, le.ip, le.url, le.message, IFNULL(mem.ID_MEMBER, 0) AS ID_MEMBER,
			IFNULL(mem.realName, '`+c.Txt("28")+`') AS display_name, IFNULL(mem.memberName, '')
		FROM {$db_prefix}log_errors AS le
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = le.ID_MEMBER)
		WHERE le.ID_MEMBER = ?
		ORDER BY le.ID_ERROR DESC
		LIMIT 20 OFFSET `+itoa(sub.Start)), memID)
	if err == nil {
		for rows.Next() {
			var logTime int64
			var ip, url, message, displayName, memberName string
			var idMember int
			rows.Scan(&logTime, &ip, &url, &message, &idMember, &displayName, &memberName)
			sub.ErrorMessages = append(sub.ErrorMessages, ProfileTrackError{
				IP:      ip,
				Message: strings.NewReplacer("&lt;span class=&quot;remove&quot;&gt;", "", "&lt;/span&gt;", "").Replace(message),
				URL:     url,
				Time:    c.timeformat(logTime),
			})
		}
		rows.Close()
	}

	// Find other users that might use the same IP.
	seenIPs := map[string]bool{}
	var uniqueIPs []string
	for _, ip := range ips {
		if ip != "" && !seenIPs[ip] {
			seenIPs[ip] = true
			uniqueIPs = append(uniqueIPs, "'"+strings.ReplaceAll(ip, "'", "''")+"'")
		}
	}
	members := map[int]string{}
	if len(uniqueIPs) > 0 {
		in := strings.Join(uniqueIPs, ", ")
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, realName
			FROM {$db_prefix}members
			WHERE ID_MEMBER != ` + itoa(memID) + `
				AND memberIP IN (` + in + `)`))
		if err == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				members[id] = `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + name + `</a>`
			}
			rows.Close()
		}

		rows, err = a.DB.Query(a.Q(`
			SELECT mem.ID_MEMBER, mem.realName
			FROM {$db_prefix}messages AS m, {$db_prefix}members AS mem
			WHERE mem.ID_MEMBER = m.ID_MEMBER
				AND mem.ID_MEMBER != ` + itoa(memID) + `
				AND m.posterIP IN (` + in + `)`))
		if err == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				members[id] = `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + name + `</a>`
			}
			rows.Close()
		}
	}
	var memberIDs []int
	for id := range members {
		memberIDs = append(memberIDs, id)
	}
	sort.Ints(memberIDs)
	for _, id := range memberIDs {
		sub.MembersInRange = append(sub.MembersInRange, members[id])
	}

	c.SubTemplate = templateProfileTrackUser
}

// TrackIP is TrackIP($memID): standalone ?action=trackip and the profile
// trackIP subaction.
func (c *Ctx) TrackIP(memID int) {
	a := c.App
	scripturl := a.ScriptURL

	// Can the user do this?
	c.isAllowedTo("moderate_forum")

	var page *ProfileCtx
	if p, ok := c.Page.(*ProfileCtx); ok {
		page = p
	} else {
		page = &ProfileCtx{ModifyError: map[string]bool{}}
		c.Page = page
	}
	sub := &ProfileTrackCtx{}
	page.Sub = sub

	if memID == 0 {
		sub.IP = strings.TrimSpace(c.REQUEST.Str("searchip"))
		if sub.IP == "" {
			sub.IP = c.User.IP
		}
		c.loadLanguage("Profile")
		c.PageTitle = c.Txt("79")
	} else {
		sub.IP = c.memberProfiles[memID].MemberIP
	}

	if !regexp.MustCompile(`^\d{1,3}\.(\d{1,3}|\*)\.(\d{1,3}|\*)\.(\d{1,3}|\*)$`).MatchString(sub.IP) {
		c.fatalError(c.Txt("invalid_ip"), false)
	}

	dbipVal := strings.ReplaceAll(sub.IP, "*", "%")
	dbip := "= '" + dbipVal + "'"
	if strings.Contains(dbipVal, "%") {
		dbip = "LIKE '" + dbipVal + "'"
	}

	c.PageTitle = c.Txt("trackIP") + " - " + sub.IP

	// Get some totals for pagination.
	var totalMessages, totalErrors int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}messages AS m
			INNER JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = m.ID_BOARD)
		WHERE ` + c.User.QuerySeeBoard + `
			AND m.posterIP ` + dbip)).Scan(&totalMessages)
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}log_errors
		WHERE ip ` + dbip)).Scan(&totalErrors)

	sub.MessageStart = c.GET.Int("mesStart")
	sub.ErrorStart = c.GET.Int("errStart")
	baseAction := "profile;u=" + itoa(memID) + ";sa=trackIP"
	if memID == 0 {
		baseAction = "trackip;searchip=" + sub.IP
	}
	sub.MessagePageIndex, _ = c.constructPageIndex(scripturl+"?action="+baseAction+";mesStart=%d;errStart="+itoa(sub.ErrorStart),
		sub.MessageStart, totalMessages, 20, true)
	sub.ErrorPageIndex, _ = c.constructPageIndex(scripturl+"?action="+baseAction+";mesStart="+itoa(sub.MessageStart)+";errStart=%d",
		sub.ErrorStart, totalErrors, 20, true)

	ipLinks := map[string][]string{}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, realName AS display_name, memberIP
		FROM {$db_prefix}members
		WHERE memberIP ` + dbip))
	if err == nil {
		for rows.Next() {
			var id int
			var name, ip string
			rows.Scan(&id, &name, &ip)
			ipLinks[ip] = append(ipLinks[ip], `<a href="`+scripturl+`?action=profile;u=`+itoa(id)+`">`+name+`</a>`)
		}
		rows.Close()
	}
	var sortedIPs []string
	for ip := range ipLinks {
		sortedIPs = append(sortedIPs, ip)
	}
	sort.Strings(sortedIPs)
	for _, ip := range sortedIPs {
		sub.IPMembers = append(sub.IPMembers, [2]string{ip, strings.Join(ipLinks[ip], ", ")})
	}

	rows, err = a.DB.Query(a.Q(`
		SELECT
			m.ID_MSG, m.posterIP, IFNULL(mem.realName, m.posterName) AS display_name, IFNULL(mem.ID_MEMBER, 0),
			m.subject, m.posterTime, m.ID_TOPIC, m.ID_BOARD
		FROM {$db_prefix}messages AS m
			INNER JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = m.ID_BOARD)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE ` + c.User.QuerySeeBoard + `
			AND m.posterIP ` + dbip + `
		ORDER BY m.ID_MSG DESC
		LIMIT 20 OFFSET ` + itoa(sub.MessageStart)))
	if err == nil {
		for rows.Next() {
			var msgID, idMember, topicID, boardID int
			var ip, name, subject string
			var posterTime int64
			rows.Scan(&msgID, &ip, &name, &idMember, &subject, &posterTime, &topicID, &boardID)
			link := name
			if idMember != 0 {
				link = `<a href="` + scripturl + `?action=profile;u=` + itoa(idMember) + `">` + name + `</a>`
			}
			sub.Messages = append(sub.Messages, ProfileTrackMessage{
				IP:         ip,
				MemberID:   idMember,
				MemberLink: link,
				BoardID:    boardID,
				BoardHref:  scripturl + "?board=" + itoa(boardID),
				Topic:      topicID,
				ID:         msgID,
				Subject:    subject,
				Time:       c.timeformat(posterTime),
			})
		}
		rows.Close()
	}

	rows, err = a.DB.Query(a.Q(`
		SELECT
			le.logTime, le.ip, le.url, le.message, IFNULL(mem.ID_MEMBER, 0) AS ID_MEMBER,
			IFNULL(mem.realName, '` + c.Txt("28") + `') AS display_name, IFNULL(mem.memberName, '')
		FROM {$db_prefix}log_errors AS le
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = le.ID_MEMBER)
		WHERE le.ip ` + dbip + `
		ORDER BY le.ID_ERROR DESC
		LIMIT 20 OFFSET ` + itoa(sub.ErrorStart)))
	if err == nil {
		for rows.Next() {
			var logTime int64
			var ip, url, message, displayName, memberName string
			var idMember int
			rows.Scan(&logTime, &ip, &url, &message, &idMember, &displayName, &memberName)
			link := displayName
			if idMember > 0 {
				link = `<a href="` + scripturl + `?action=profile;u=` + itoa(idMember) + `">` + displayName + `</a>`
			}
			sub.ErrorMessages = append(sub.ErrorMessages, ProfileTrackError{
				IP:         ip,
				MemberID:   idMember,
				MemberName: displayName,
				MemberLink: link,
				Message:    strings.NewReplacer("&lt;span class=&quot;remove&quot;&gt;", "", "&lt;/span&gt;", "").Replace(message),
				URL:        url,
				Time:       c.timeformat(logTime),
			})
		}
		rows.Close()
	}

	sub.SingleIP = !strings.Contains(sub.IP, "*")
	if sub.SingleIP {
		sub.WhoisServers = []ProfileWhois{
			{Name: c.Txt("whois_afrinic"), URL: "http://www.afrinic.net/cgi-bin/whois?searchtext=" + sub.IP},
			{Name: c.Txt("whois_apnic"), URL: "http://wq.apnic.net/apnic-bin/whois.pl?searchtext=" + sub.IP},
			{Name: c.Txt("whois_arin"), URL: "http://ws.arin.net/whois/?queryinput=" + sub.IP},
			{Name: c.Txt("whois_lacnic"), URL: "http://lacnic.net/cgi-bin/lacnic/whois?query=" + sub.IP},
			{Name: c.Txt("whois_ripe"), URL: "http://www.db.ripe.net/whois?searchtext=" + sub.IP},
		}
		ranges := [][]int{
			{},
			{58, 59, 60, 61, 124, 125, 126, 202, 203, 210, 211, 218, 219, 220, 221, 222},
			{63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 199, 204, 205, 206, 207, 208, 209, 216},
			{200, 201},
			{62, 80, 81, 82, 83, 84, 85, 86, 87, 88, 193, 194, 195, 212, 213, 217},
		}
		firstOctet := atoi(sub.IP)
		for i, r := range ranges {
			for _, v := range r {
				if v == firstOctet {
					w := sub.WhoisServers[i]
					sub.AutoWhois = &w
				}
			}
		}
	}

	c.SubTemplate = templateProfileTrackIP
}

// ProfilePermission is one permission row on showPermissions.
type ProfilePermission struct {
	ID       string
	Name     string
	Allowed  []string
	Denied   []string
	IsDenied bool
	IsGlobal bool
}

// ProfilePermBoard is one board on the permission board selector.
type ProfilePermBoard struct {
	ID       int
	Name     string
	Selected bool
	IsLast   bool
}

// ProfileShowPermsCtx is the data for the showPermissions template.
type ProfileShowPermsCtx struct {
	MemberName        string
	Board             int
	Boards            []ProfilePermBoard
	NoAccessBoards    []ProfilePermBoard
	HasAllPermissions bool
	General           []*ProfilePermission
	BoardPerms        []*ProfilePermission
}

// profileShowPermissions is showPermissions($memID).
func (c *Ctx) profileShowPermissions(page *ProfileCtx, memID int) {
	a := c.App

	// Verify if the user has sufficient permissions.
	c.isAllowedTo("manage_permissions")

	c.loadLanguage("ManagePermissions")
	c.loadLanguage("Admin")

	profile := c.memberProfiles[memID]
	sub := &ProfileShowPermsCtx{MemberName: profile.RealName}
	page.Sub = sub

	c.PageTitle = c.Txt("showPermissions")
	board := c.Board
	sub.Board = board

	// Determine which groups this user is in.
	var curGroups []int
	if profile.AdditionalGroups != "" {
		for _, g := range strings.Split(profile.AdditionalGroups, ",") {
			curGroups = append(curGroups, atoi(g))
		}
	}
	curGroups = append(curGroups, profile.IDGroup, profile.IDPostGroup)
	curIDs := make([]string, len(curGroups))
	inCur := map[int]bool{}
	for i, g := range curGroups {
		curIDs[i] = itoa(g)
		inCur[g] = true
	}

	// Load a list of boards for the jump box.
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name, b.permission_mode, b.memberGroups, (b.permission_mode != 0 OR mods.ID_MEMBER IS NOT NULL) AS show_board
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = b.ID_BOARD AND mods.ID_MEMBER = ` + itoa(memID) + `)
		WHERE ` + c.User.QuerySeeBoard))
	if err == nil {
		for rows.Next() {
			var id, permissionMode, showBoard int
			var name, memberGroups string
			rows.Scan(&id, &name, &permissionMode, &memberGroups, &showBoard)

			hasAccess := false
			for _, g := range strings.Split(memberGroups, ",") {
				if inCur[atoi(g)] {
					hasAccess = true
				}
			}
			if !hasAccess {
				sub.NoAccessBoards = append(sub.NoAccessBoards, ProfilePermBoard{ID: id, Name: name})
			} else if showBoard != 0 {
				sub.Boards = append(sub.Boards, ProfilePermBoard{ID: id, Name: name, Selected: board == id})
			}
		}
		rows.Close()
	}
	if len(sub.NoAccessBoards) > 0 {
		sub.NoAccessBoards[len(sub.NoAccessBoards)-1].IsLast = true
	}

	// If you're an admin we know you can do everything, we might as well
	// leave.
	sub.HasAllPermissions = inCur[1]
	if sub.HasAllPermissions {
		c.SubTemplate = templateProfileShowPermissions
		return
	}

	permName := func(perm string) (string, bool) {
		full := c.Txt("permissionname_" + perm)
		if full == "permissionname_"+perm {
			return "", false
		}
		if len(perm) > 4 && (strings.HasSuffix(perm, "_own") || strings.HasSuffix(perm, "_any")) {
			base := c.Txt("permissionname_" + perm[:len(perm)-4])
			if base != "permissionname_"+perm[:len(perm)-4] {
				return base + " - " + full, true
			}
		}
		return full, true
	}

	// Get all general permissions.
	general := map[string]*ProfilePermission{}
	rows, err = a.DB.Query(a.Q(`
		SELECT p.permission, p.addDeny, IFNULL(mg.groupName, ''), p.ID_GROUP
		FROM {$db_prefix}permissions AS p
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = p.ID_GROUP)
		WHERE p.ID_GROUP IN (` + strings.Join(curIDs, ", ") + `)
		ORDER BY p.addDeny DESC, p.permission, mg.minPosts, IIF(mg.ID_GROUP < 4, mg.ID_GROUP, 4), mg.groupName`))
	if err == nil {
		for rows.Next() {
			var permission, groupName string
			var addDeny, idGroup int
			rows.Scan(&permission, &addDeny, &groupName, &idGroup)

			name, known := permName(permission)
			if !known {
				continue
			}

			p, ok := general[permission]
			if !ok {
				p = &ProfilePermission{ID: permission, Name: name, IsGlobal: true}
				general[permission] = p
				sub.General = append(sub.General, p)
			}
			displayName := groupName
			if idGroup == 0 {
				displayName = c.Txt("membergroups_members")
			}
			if addDeny == 0 {
				p.Denied = append(p.Denied, displayName)
				p.IsDenied = true
			} else {
				p.Allowed = append(p.Allowed, displayName)
			}
		}
		rows.Close()
	}

	// Board permissions.
	boardCond := "= 0"
	fromExtra := ""
	whereExtra := ""
	extraCols := ""
	if board != 0 {
		fromExtra = `, {$db_prefix}boards AS b`
		extraCols = `,
			b.permission_mode, IIF(mods.ID_MEMBER IS NULL, 0, 1) AS is_moderator`
		whereExtra = `
			AND b.ID_BOARD = ` + itoa(board) + `
			AND (mods.ID_MEMBER IS NOT NULL OR bp.ID_GROUP != 3)`
		if !a.SettingEmpty("permission_enable_by_board") {
			boardCond = "IIF(b.permission_mode = 1, b.ID_BOARD, 0)"
		}
	}
	groupCond := "IN (" + strings.Join(curIDs, ", ") + ")"
	joinMods := ""
	if board != 0 {
		groupCond = "IN (" + strings.Join(curIDs, ", ") + ", 3)"
		joinMods = `
			LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = b.ID_BOARD AND mods.ID_MEMBER = ` + itoa(memID) + `)`
	}

	boardPerms := map[string]*ProfilePermission{}
	rows, err = a.DB.Query(a.Q(`
		SELECT
			bp.addDeny, bp.permission, bp.ID_GROUP, IFNULL(mg.groupName, '')` + extraCols + `
		FROM {$db_prefix}board_permissions AS bp` + fromExtra + joinMods + `
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = bp.ID_GROUP)
		WHERE bp.ID_BOARD = ` + boardCond + `
			AND bp.ID_GROUP ` + groupCond + whereExtra))
	if err == nil {
		for rows.Next() {
			var addDeny, idGroup int
			var permission, groupName string
			permissionMode := 0
			isModerator := 0
			dest := []any{&addDeny, &permission, &idGroup, &groupName}
			if board != 0 {
				dest = append(dest, &permissionMode, &isModerator)
			}
			rows.Scan(dest...)

			name, known := permName(permission)
			if !known {
				continue
			}

			// Filter these special cases of board permissions out.
			if a.SettingEmpty("permission_enable_by_board") && board != 0 && idGroup != 3 {
				if (permission == "post_reply_own" || permission == "post_reply_any") && permissionMode == 4 {
					continue
				} else if permission == "post_new" && permissionMode >= 3 {
					continue
				} else if permission == "poll_post" && permissionMode >= 2 {
					continue
				}
			}

			p, ok := boardPerms[permission]
			if !ok {
				p = &ProfilePermission{ID: permission, Name: name, IsGlobal: board == 0}
				boardPerms[permission] = p
				sub.BoardPerms = append(sub.BoardPerms, p)
			}
			displayName := groupName
			if idGroup == 0 {
				displayName = c.Txt("membergroups_members")
			}
			if addDeny == 0 {
				p.Denied = append(p.Denied, displayName)
				p.IsDenied = true
			} else {
				p.Allowed = append(p.Allowed, displayName)
			}
		}
		rows.Close()
	}

	c.SubTemplate = templateProfileShowPermissions
}
