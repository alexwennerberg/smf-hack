package app

// Port of Sources/BoardIndex.php.

import (
	"sort"
	"time"
)

func init() {
	registerAction("*boardindex", (*Ctx).BoardIndex)
}

// BICategory is one entry of $context['categories'].
type BICategory struct {
	ID            int
	Name          string
	IsCollapsed   bool
	CanCollapse   bool
	CollapseHref  string
	CollapseImage string
	Href          string
	Link          string
	New           bool
	Boards        []*BIBoard // ordered
}

// BIBoard is one board row on the index.
type BIBoard struct {
	ID             int
	Name           string
	Description    string
	New            bool
	Topics         int
	Posts          int
	Href           string
	Link           string
	Moderators     []Moderator
	LinkModerators []string
	Children       []*BIBoard
	LinkChildren   []string
	ChildrenNew    bool
	LastPost       *BILastPost
}

// BILastPost is the 'last_post' info array.
type BILastPost struct {
	ID         int
	Time       string
	Timestamp  int64
	Subject    string
	Href       string
	Link       string
	Start      string
	Topic      int
	MemberID   int
	MemberName string
	MemberLink string
}

// BIOnlineUser is one entry of $context['users_online'].
type BIOnlineUser struct {
	sortKey  string
	ID       int
	Username string
	Name     string
	Group    int
	Href     string
	Link     string
	IsBuddy  bool
	Hidden   bool
}

type BIOnlineGroup struct {
	ID    int
	Name  string
	Color string
}

// BoardIndexCtx is the page context consumed by tpl_boardindex.go.
type BoardIndexCtx struct {
	Categories      []*BICategory
	UsersOnline     []BIOnlineUser
	ListUsersOnline []string
	OnlineGroups    []BIOnlineGroup
	NumGuests       int
	NumBuddies      int
	NumUsersHidden  int
	NumUsersOnline  int
	ShowBuddies     bool
	LatestPost      *BILastPost
	LatestPosts     []*LatestPost
	ShowStats       bool
	ShowMemberList  bool
	ShowWho         bool
	ShowLoginBar    bool
	ShowCalendar    bool

	// calendarDoIndex() output (board-index info-center block).
	CalendarOnlyToday bool
	CalendarCanEdit   bool
	CalendarHolidays  []string
	CalendarBirthdays []*CalBirthday
	CalendarEvents    []*CalEvent
}

// BoardIndex is BoardIndex() from BoardIndex.php.
func (c *Ctx) BoardIndex() {
	a := c.App
	scripturl := a.ScriptURL
	page := &BoardIndexCtx{}
	c.Page = page

	// Remember the most recent topic for optimizing the recent posts feature.
	var mostRecentTimestamp int64
	var mostRecentRef **BILastPost

	// Find all boards and categories, as well as related information.
	query := `
		SELECT
			c.name AS catName, c.ID_CAT, b.ID_BOARD, b.name AS boardName, b.description,
			b.numPosts, b.numTopics, b.ID_PARENT, IFNULL(m.posterTime, 0) AS posterTime,
			IFNULL(mem.memberName, IFNULL(m.posterName, '')) AS posterName, IFNULL(m.subject, ''), IFNULL(m.ID_TOPIC, 0),
			IFNULL(mem.realName, IFNULL(m.posterName, '')) AS realName,` + (func() string {
		if c.User.IsGuest {
			return `
			1 AS isRead, 0 AS new_from, 1 AS canCollapse, 0 AS isCollapsed,`
		}
		return `
			(IFNULL(lb.ID_MSG, 0) >= b.ID_MSG_UPDATED) AS isRead, IFNULL(lb.ID_MSG, -1) + 1 AS new_from,
			c.canCollapse, IFNULL(cc.ID_MEMBER, 0) AS isCollapsed,`
	})() + `
			IFNULL(mem.ID_MEMBER, 0) AS ID_MEMBER, IFNULL(m.ID_MSG, 0),
			IFNULL(mods_mem.ID_MEMBER, 0) AS ID_MODERATOR, IFNULL(mods_mem.realName, '') AS modRealName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_MSG = b.ID_LAST_MSG)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)` + (func() string {
		if c.User.IsGuest {
			return ""
		}
		return `
			LEFT JOIN {$db_prefix}log_boards AS lb ON (lb.ID_BOARD = b.ID_BOARD AND lb.ID_MEMBER = ` + itoa(c.User.ID) + `)
			LEFT JOIN {$db_prefix}collapsed_categories AS cc ON (cc.ID_CAT = c.ID_CAT AND cc.ID_MEMBER = ` + itoa(c.User.ID) + `)`
	})() + `
			LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = b.ID_BOARD)
			LEFT JOIN {$db_prefix}members AS mods_mem ON (mods_mem.ID_MEMBER = mods.ID_MEMBER)
		WHERE ` + c.User.QuerySeeBoard + (func() string {
		if a.SettingEmpty("countChildPosts") {
			return `
			AND b.childLevel <= 1`
		}
		return ""
	})()

	rows, err := a.DB.Query(a.Q(query))
	if err != nil {
		c.fatalDBError(err)
	}

	cats := map[int]*BICategory{}
	boardsByID := map[int]*BIBoard{} // top-level boards in current category context
	catOfBoard := map[int]*BICategory{}

	guestCanCollapse := c.User.IsGuest // guests: canCollapse columns synthetic

	for rows.Next() {
		var catName, boardName, description, posterName, subject, realName, modRealName string
		var idCat, idBoard, numPosts, numTopics, idParent, idTopic, idMember, idMsg, idModerator int
		var posterTime int64
		var isRead, newFrom, canCollapse, isCollapsed int
		if err := rows.Scan(&catName, &idCat, &idBoard, &boardName, &description,
			&numPosts, &numTopics, &idParent, &posterTime,
			&posterName, &subject, &idTopic, &realName,
			&isRead, &newFrom, &canCollapse, &isCollapsed,
			&idMember, &idMsg, &idModerator, &modRealName); err != nil {
			rows.Close()
			c.fatalDBError(err)
		}

		// Haven't set this category yet.
		cat, ok := cats[idCat]
		if !ok {
			cat = &BICategory{
				ID:          idCat,
				Name:        catName,
				IsCollapsed: !c.User.IsGuest && canCollapse == 1 && isCollapsed > 0,
				CanCollapse: !c.User.IsGuest && canCollapse == 1,
				Href:        scripturl + "#" + itoa(idCat),
			}
			if !c.User.IsGuest {
				sa := "collapse"
				img := `collapse.gif" alt="-"`
				if isCollapsed > 0 {
					sa = "expand"
					img = `expand.gif" alt="+"`
				}
				cat.CollapseHref = scripturl + "?action=collapse;c=" + itoa(idCat) + ";sa=" + sa + ";sesc=" + c.Sc + "#" + itoa(idCat)
				cat.CollapseImage = `<img src="` + c.Theme.ImagesURL() + `/` + img + ` border="0" />`
				cat.Link = `<a name="` + itoa(idCat) + `" href="` + cat.CollapseHref + `">` + catName + `</a>`
			} else {
				cat.Link = `<a name="` + itoa(idCat) + `" href="` + cat.Href + `">` + catName + `</a>`
			}
			cats[idCat] = cat
			page.Categories = append(page.Categories, cat)
		}
		_ = guestCanCollapse

		// If this board has new posts in it (and isn't the recycle bin!)
		// then the category is new.
		if a.SettingEmpty("recycle_enable") || a.SettingInt("recycle_board") != idBoard {
			cat.New = cat.New || (isRead == 0 && posterName != "")
		}

		// Collapsed category - don't do any of this.
		if cat.IsCollapsed {
			continue
		}

		isChild := false
		var board *BIBoard

		if idParent == 0 {
			// This is a parent board.
			board, ok = boardsByID[idBoard]
			if !ok {
				board = &BIBoard{
					New:         isRead == 0,
					ID:          idBoard,
					Name:        boardName,
					Description: description,
					Topics:      numTopics,
					Posts:       numPosts,
					Href:        scripturl + "?board=" + itoa(idBoard) + ".0",
					Link:        `<a href="` + scripturl + "?board=" + itoa(idBoard) + `.0">` + boardName + `</a>`,
				}
				boardsByID[idBoard] = board
				catOfBoard[idBoard] = cat
				cat.Boards = append(cat.Boards, board)
			}
			if idModerator != 0 && !boardHasModerator(board, idModerator) {
				board.Moderators = append(board.Moderators, Moderator{
					ID:   idModerator,
					Name: modRealName,
					Href: scripturl + "?action=profile;u=" + itoa(idModerator),
					Link: `<a href="` + scripturl + "?action=profile;u=" + itoa(idModerator) + `" title="` + c.Txt("62") + `">` + modRealName + `</a>`,
				})
				board.LinkModerators = append(board.LinkModerators,
					`<a href="`+scripturl+"?action=profile;u="+itoa(idModerator)+`" title="`+c.Txt("62")+`">`+modRealName+`</a>`)
			}
		} else if parent, ok2 := boardsByID[idParent]; ok2 && !boardHasChild(parent, idBoard) {
			// Found a child board.... make sure we've found its parent and
			// the child hasn't been set already.
			isChild = true
			board = parent // last_post goes on the parent below; child here:

			child := &BIBoard{
				ID:          idBoard,
				Name:        boardName,
				Description: description,
				New:         isRead == 0 && posterName != "",
				Topics:      numTopics,
				Posts:       numPosts,
				Href:        scripturl + "?board=" + itoa(idBoard) + ".0",
				Link:        `<a href="` + scripturl + "?board=" + itoa(idBoard) + `.0">` + boardName + `</a>`,
			}
			parent.Children = append(parent.Children, child)

			// Counting child board posts is... slow :/.
			if !a.SettingEmpty("countChildPosts") {
				parent.Posts += numPosts
				parent.Topics += numTopics
			}

			// Does this board contain new boards?
			parent.ChildrenNew = parent.ChildrenNew || isRead == 0

			// This is easier to use in many cases for the theme....
			parent.LinkChildren = append(parent.LinkChildren, child.Link)
		} else {
			// Child of a child (countChildPosts roll-up omitted: that setting
			// is off by default and the boards query excludes childLevel > 1
			// in that case) — or a child of a child without it - skip.
			continue
		}

		// Prepare the subject, and make sure it's not too long.
		subject = c.censorText(subject)
		shortSubject := shortenSubject(subject, 24)

		lp := &BILastPost{
			ID:        idMsg,
			Timestamp: c.forumTime(true, posterTime),
			Subject:   shortSubject,
			Start:     "msg" + itoa(newFrom),
			Topic:     idTopic,
			MemberID:  idMember,
		}
		if posterTime > 0 {
			lp.Time = c.timeformat(posterTime)
		} else {
			lp.Time = c.Txt("470")
		}
		if posterName != "" {
			lp.MemberName = realName
			if idMember != 0 {
				lp.MemberLink = `<a href="` + scripturl + "?action=profile;u=" + itoa(idMember) + `">` + realName + `</a>`
			} else {
				lp.MemberLink = realName
			}
		} else {
			lp.MemberName = c.Txt("470")
			lp.MemberLink = c.Txt("470")
		}

		// Provide the href and link.
		if subject != "" {
			msgRef := itoa(newFrom)
			if c.User.IsGuest {
				msgRef = a.Setting("maxMsgID")
			}
			seen := ""
			if isRead == 0 {
				seen = ";boardseen"
			}
			lp.Href = scripturl + "?topic=" + itoa(idTopic) + ".msg" + msgRef + seen + "#new"
			lp.Link = `<a href="` + lp.Href + `" title="` + subject + `">` + shortSubject + `</a>`
		} else {
			lp.Href = ""
			lp.Link = c.Txt("470")
		}

		// Set the last post in the parent board.
		target := board
		if isChild {
			child := board.Children[len(board.Children)-1]
			if posterTime != 0 && (board.LastPost == nil || board.LastPost.Timestamp < c.forumTime(true, posterTime)) {
				board.LastPost = lp
			}
			child.LastPost = lp
			// If there are no posts in this board, it really can't be new...
			child.New = child.New && posterName != ""
		} else {
			board.LastPost = lp
			// No last post for this board?  It's not new then, is it..?
			if posterName == "" {
				board.New = false
			}
		}

		// Determine a global most recent topic.
		if posterTime != 0 && c.forumTime(true, posterTime) > mostRecentTimestamp {
			mostRecentTimestamp = c.forumTime(true, posterTime)
			mostRecentRef = &target.LastPost
		}
	}
	rows.Close()

	// Load the users online right now.
	c.loadUsersOnline(page)

	// Track most online statistics?
	if !a.SettingEmpty("trackStats") {
		c.trackMostOnline(page.NumGuests + page.NumUsersOnline)
	}

	// Load the most recent post? (number_recent_posts == 1 or show_sp1_info.)
	if (!c.Theme.Empty("number_recent_posts") && c.Theme.Int("number_recent_posts") == 1) || !c.Theme.Empty("show_sp1_info") {
		if mostRecentRef != nil {
			page.LatestPost = *mostRecentRef
		}
	}
	// Or load several recent posts for the info-center list.
	if !c.Theme.Empty("number_recent_posts") && c.Theme.Int("number_recent_posts") > 1 {
		page.LatestPosts = c.getLastPosts(c.Theme.Int("number_recent_posts"))
	}

	page.ShowStats = c.allowedTo("view_stats") && !a.SettingEmpty("trackStats")
	page.ShowMemberList = c.allowedTo("view_mlist")
	page.ShowWho = c.allowedTo("who_view") && !a.SettingEmpty("who_enabled")

	// Set some permission related settings.
	page.ShowLoginBar = c.User.IsGuest && !a.SettingEmpty("enableVBStyleLogin")
	page.ShowCalendar = c.allowedTo("calendar_view") && !a.SettingEmpty("cal_enabled")
	if page.ShowCalendar {
		page.ShowCalendar = c.calendarDoIndex(page)
	}

	c.PageTitle = c.Txt("18")
	c.SubTemplate = templateBoardIndexMain
}

// calendarDoIndex is calendarDoIndex() from BoardIndex.php: prepares today's
// (and the next cal_days_for_index days') holidays, birthdays and events for
// the board-index info-center block, returning whether anything should show.
// PHP caches the data PHP-serialized in cal_today_* settings (refreshed by
// updateStats('calendar') once per day); this port recomputes it live each
// request — a perf-only difference with identical output, as elsewhere.
func (c *Ctx) calendarDoIndex(page *BoardIndexCtx) bool {
	a := c.App

	// Make sure at least one of the options is checked.
	if a.SettingEmpty("cal_showeventsonindex") && a.SettingEmpty("cal_showbdaysonindex") && a.SettingEmpty("cal_showholidaysonindex") {
		return false
	}

	// This shouldn't be less than one!
	calDays := a.SettingInt("cal_days_for_index")
	daysCount := calDays
	if daysCount < 1 {
		daysCount = 1
	}
	page.CalendarOnlyToday = calDays == 1

	// Get the current member time/date and the days we span.
	now := c.forumTime(true, 0)
	today := time.Unix(now, 0).UTC().Format("2006-01-02")
	dates := make([]string, daysCount)
	for k := 0; k < daysCount; k++ {
		dates[k] = time.Unix(now+int64(k)*86400, 0).UTC().Format("2006-01-02")
	}
	low, high := dates[0], dates[len(dates)-1]

	// Load today's holidays / birthdays / events (only the enabled ones).
	var holidays map[string][]string
	if !a.SettingEmpty("cal_showholidaysonindex") {
		holidays = c.calendarHolidayArray(low, high)
	}
	var bday map[string][]*CalBirthday
	if !a.SettingEmpty("cal_showbdaysonindex") {
		bday = c.calendarBirthdayArray(low, high)
	}
	var events map[string][]*CalEvent
	if !a.SettingEmpty("cal_showeventsonindex") {
		// Board-visibility filtering replaces PHP's per-event allowed_groups
		// array_intersect check (usePermissions = true).
		events = c.calendarEventArray(low, high, true)
	}

	// No events, birthdays, or holidays... don't show anything. Simple.
	if len(holidays) == 0 && len(bday) == 0 && len(events) == 0 {
		return false
	}

	// This is used to show the "how-do-I-edit" help.
	page.CalendarCanEdit = c.allowedTo("calendar_edit_any")

	// Holidays between now and now + days.
	for _, d := range dates {
		page.CalendarHolidays = append(page.CalendarHolidays, holidays[d]...)
	}

	// Happy Birthday, guys and gals!
	for _, d := range dates {
		for _, b := range bday[d] {
			b.IsToday = d == today
			page.CalendarBirthdays = append(page.CalendarBirthdays, b)
		}
	}

	// Events like community get-togethers (deduped by topic+title).
	duplicates := map[string]bool{}
	for _, d := range dates {
		for _, ev := range events[d] {
			key := itoa(ev.Topic) + ev.Title
			if duplicates[key] {
				continue
			}
			ev.IsToday = d == today
			duplicates[key] = true
			page.CalendarEvents = append(page.CalendarEvents, ev)
		}
	}

	// Recompute is_last across the merged lists.
	for i := range page.CalendarBirthdays {
		page.CalendarBirthdays[i].IsLast = i == len(page.CalendarBirthdays)-1
	}
	for i := range page.CalendarEvents {
		page.CalendarEvents[i].IsLast = i == len(page.CalendarEvents)-1
	}

	// This is used to make sure the header should be displayed.
	return len(page.CalendarHolidays) > 0 || len(page.CalendarBirthdays) > 0 || len(page.CalendarEvents) > 0
}

func boardHasModerator(b *BIBoard, id int) bool {
	for _, m := range b.Moderators {
		if m.ID == id {
			return true
		}
	}
	return false
}

func boardHasChild(b *BIBoard, id int) bool {
	for _, ch := range b.Children {
		if ch.ID == id {
			return true
		}
	}
	return false
}

// loadUsersOnline fills the users-online block (shared by template's info
// center).
func (c *Ctx) loadUsersOnline(page *BoardIndexCtx) {
	a := c.App
	rows, err := a.DB.Query(a.Q(`
		SELECT
			lo.ID_MEMBER, lo.logTime, IFNULL(mem.realName, '') AS realName, IFNULL(mem.memberName, '') AS memberName,
			IFNULL(mem.showOnline, 0), IFNULL(mg.onlineColor, ''), IFNULL(mg.ID_GROUP, 0), IFNULL(mg.groupName, '')
		FROM {$db_prefix}log_online AS lo
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = lo.ID_MEMBER)
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = CASE WHEN mem.ID_GROUP = 0 THEN mem.ID_POST_GROUP ELSE mem.ID_GROUP END)`))
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()

	page.ShowBuddies = len(c.User.Buddies) > 0
	groupSeen := map[int]bool{}

	for rows.Next() {
		var idMember, showOnline, idGroup int
		var logTime int64
		var realName, memberName, onlineColor, groupName string
		rows.Scan(&idMember, &logTime, &realName, &memberName, &showOnline, &onlineColor, &idGroup, &groupName)

		if realName == "" {
			page.NumGuests++
			continue
		} else if showOnline == 0 && !c.allowedTo("moderate_forum") {
			page.NumUsersHidden++
			continue
		}

		// Some basic color coding...
		var link string
		if onlineColor != "" {
			link = `<a href="` + a.ScriptURL + "?action=profile;u=" + itoa(idMember) + `" style="color: ` + onlineColor + `;">` + realName + `</a>`
		} else {
			link = `<a href="` + a.ScriptURL + "?action=profile;u=" + itoa(idMember) + `">` + realName + `</a>`
		}

		isBuddy := false
		for _, b := range c.User.Buddies {
			if b == idMember {
				isBuddy = true
				break
			}
		}
		if isBuddy {
			page.NumBuddies++
			link = "<b>" + link + "</b>"
		}

		listLink := link
		if showOnline == 0 {
			listLink = "<i>" + link + "</i>"
		}

		page.UsersOnline = append(page.UsersOnline, BIOnlineUser{
			sortKey:  itoa(int(logTime)) + memberName,
			ID:       idMember,
			Username: memberName,
			Name:     realName,
			Group:    idGroup,
			Href:     a.ScriptURL + "?action=profile;u=" + itoa(idMember),
			Link:     link,
			IsBuddy:  isBuddy,
			Hidden:   showOnline == 0,
		})
		_ = listLink

		if !groupSeen[idGroup] {
			groupSeen[idGroup] = true
			page.OnlineGroups = append(page.OnlineGroups, BIOnlineGroup{ID: idGroup, Name: groupName, Color: onlineColor})
		}
	}

	// krsort: descending by logTime+memberName key.
	sort.Slice(page.UsersOnline, func(i, j int) bool {
		return page.UsersOnline[i].sortKey > page.UsersOnline[j].sortKey
	})
	for _, u := range page.UsersOnline {
		l := u.Link
		if u.Hidden {
			l = "<i>" + l + "</i>"
		}
		page.ListUsersOnline = append(page.ListUsersOnline, l)
	}
	// ksort: ascending by group id.
	sort.Slice(page.OnlineGroups, func(i, j int) bool {
		return page.OnlineGroups[i].ID < page.OnlineGroups[j].ID
	})

	page.NumUsersOnline = len(page.UsersOnline) + page.NumUsersHidden
}

// trackMostOnline is the most-online statistics block of BoardIndex().
func (c *Ctx) trackMostOnline(totalUsers int) {
	a := c.App

	// More members on now than ever were?  Update it!
	if a.Setting("mostOnline") == "" || totalUsers >= a.SettingInt("mostOnline") {
		a.UpdateSettings(map[string]string{
			"mostOnline": itoa(totalUsers),
			"mostDate":   itoa(int(time.Now().Unix())),
		})
	}

	date := time.Unix(c.forumTime(false, 0), 0).Format("2006-01-02")

	if a.Setting("mostOnlineUpdated") != date {
		var mostOn int
		err := a.DB.QueryRow(a.Q(`SELECT mostOn FROM {$db_prefix}log_activity WHERE date = ? LIMIT 1`), date).Scan(&mostOn)
		if err != nil {
			// The log_activity hasn't got an entry for today?
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}log_activity (date, mostOn) VALUES (?, ?)`), date, totalUsers)
		} else {
			// There's an entry in log_activity on today...
			if totalUsers > mostOn {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_activity SET mostOn = ? WHERE date = ?`), totalUsers, date)
			}
			if mostOn > totalUsers {
				totalUsers = mostOn
			}
		}
		a.UpdateSettings(map[string]string{"mostOnlineUpdated": date, "mostOnlineToday": itoa(totalUsers)})
	} else if totalUsers > a.SettingInt("mostOnlineToday") {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_activity SET mostOn = ? WHERE date = ?`), totalUsers, date)
		a.UpdateSettings(map[string]string{"mostOnlineUpdated": date, "mostOnlineToday": itoa(totalUsers)})
	}
}
