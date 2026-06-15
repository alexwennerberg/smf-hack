package app

// Port of Sources/Calendar.php: the month-grid view (CalendarMain), the
// birthday/event/holiday array builders, and event posting (CalendarPost,
// calendarInsertEvent, calendarCanLink). Event posting is gated behind the
// calendar_post permission (off by default). The topic-linked event path runs
// through Post()/Post2() (the make_event blocks in post_form.go / post.go).

import (
	"strings"
	"time"
)

func init() {
	registerAction("calendar", (*Ctx).CalendarMain)
}

// CalBirthday is one birthday on a calendar day.
type CalBirthday struct {
	ID      int
	Name    string
	Age     int
	HasAge  bool
	IsLast  bool
	IsToday bool // set by calendarDoIndex (board-index block)
}

// CalEvent is one posted event on a calendar day.
type CalEvent struct {
	ID         int
	Title      string
	Topic      int  // for calendarDoIndex's topic+title dedup
	CanEdit    bool
	ModifyHref string
	Href       string
	Link       string
	StartDate  string
	EndDate    string
	IsLast     bool
	IsToday    bool // set by calendarDoIndex (board-index block)
}

// CalDay is one cell in the month grid.
type CalDay struct {
	Day        int
	Date       string
	IsToday    bool
	IsFirstDay bool
	Holidays   []string
	Events     []*CalEvent
	Birthdays  []*CalBirthday
}

// CalWeek is one row in the month grid.
type CalWeek struct {
	Days   []*CalDay
	Number int
}

// CalendarCtx is the page context for the Calendar template.
type CalendarCtx struct {
	WeekDays     []int
	Weeks        []*CalWeek
	CanPost      bool
	LastDay      int
	CurrentMonth int
	CurrentYear  int

	HasPrev   bool
	PrevHref  string
	PrevMonth int
	PrevYear  int
	HasNext   bool
	NextHref  string
	NextMonth int
	NextYear  int
}

func mktimeUTC(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// CalendarMain is CalendarMain(): the month grid.
func (c *Ctx) CalendarMain() {
	a := c.App
	scripturl := a.ScriptURL

	// If we are posting a new event defect to the posting function.
	if c.GET.Str("sa") == "post" {
		c.CalendarPost()
		return
	}

	c.isAllowedTo("calendar_view")
	if a.SettingEmpty("cal_enabled") {
		c.fatalLangError("calendar_off", false)
	}

	c.PageTitle = a.Config.MbName + ": " + c.Txt("calendar24")

	// Today (in the user's local time).
	todayT := time.Unix(c.forumTime(true, 0), 0).UTC()
	today := todayT.Format("2006-01-02")

	curMonth := int(todayT.Month())
	curYear := todayT.Year()
	if c.REQUEST.Has("month") {
		curMonth = c.REQUEST.Int("month")
	}
	if c.REQUEST.Has("year") {
		curYear = c.REQUEST.Int("year")
	}

	if curMonth < 1 || curMonth > 12 {
		c.fatalLangError("calendar1", false)
	}
	if curYear < a.SettingInt("cal_minyear") || curYear > a.SettingInt("cal_maxyear") {
		c.fatalLangError("calendar2", false)
	}

	page := &CalendarCtx{CurrentMonth: curMonth, CurrentYear: curYear}
	c.Page = page

	firstOfMonth := mktimeUTC(curYear, curMonth, 1)
	firstDayOfWeek := int(firstOfMonth.Weekday())
	weekNum := (firstOfMonth.YearDay() + 6 - firstDayOfWeek) / 7

	// Last day of the month.
	nextMonth := curMonth + 1
	nextMonthYear := curYear
	if curMonth == 12 {
		nextMonth = 1
		nextMonthYear++
	}
	nLastDay := mktimeUTC(nextMonthYear, nextMonth, 1).AddDate(0, 0, -1).Day()

	// The start-of-week shift.
	nShift := firstDayOfWeek
	nStartDay := atoi(c.Options["calendar_start_day"])
	if nStartDay != 0 {
		nShift -= nStartDay
		if nShift < 0 {
			nShift = 7 + nShift
		}
	}

	nRows := (nLastDay + nShift) / 7
	if (nLastDay+nShift)%7 != 0 {
		nRows++
	}

	low := itoa(curYear) + "-" + sprintf02(curMonth) + "-01"
	high := itoa(curYear) + "-" + sprintf02(curMonth) + "-" + sprintf02(nLastDay)

	var bday map[string][]*CalBirthday
	var events map[string][]*CalEvent
	var holidays map[string][]string
	if !a.SettingEmpty("cal_showbdaysoncalendar") {
		bday = c.calendarBirthdayArray(low, high)
	}
	if !a.SettingEmpty("cal_showeventsoncalendar") {
		events = c.calendarEventArray(low, high, true)
	}
	if !a.SettingEmpty("cal_showholidaysoncalendar") {
		holidays = c.calendarHolidayArray(low, high)
	}

	// Days of the week, honoring the configured start day.
	count := nStartDay
	for i := 0; i < 7; i++ {
		page.WeekDays = append(page.WeekDays, count)
		count++
		if count == 7 {
			count = 0
		}
	}

	// Week-number adjustment.
	nWeekAdjust := 0
	if !a.SettingEmpty("cal_showweeknum") {
		foy := int(mktimeUTC(curYear, 1, 1).Weekday())
		if nStartDay == 0 {
			if foy != 0 {
				nWeekAdjust = 1
			}
		} else if nStartDay > foy && foy != 0 {
			nWeekAdjust = 2
		} else {
			nWeekAdjust = 1
		}
		if firstDayOfWeek < nStartDay {
			nWeekAdjust--
		}
	}

	page.CanPost = c.allowedTo("calendar_post")
	page.LastDay = nLastDay

	c.LinkTree = append(c.LinkTree, Link{
		URL:  scripturl + "?action=calendar;year=" + itoa(curYear) + ";month=" + itoa(curMonth),
		Name: c.TxtListItem("months", curMonth) + " " + itoa(curYear),
	})

	// Build the weeks.
	for nRow := 0; nRow < nRows; nRow++ {
		week := &CalWeek{Number: weekNum + nRow + nWeekAdjust}
		if week.Number == 53 && nShift != 4 {
			week.Number = 1
		}
		for nCol := 0; nCol < 7; nCol++ {
			nDay := (nRow * 7) + nCol - nShift + 1
			if nDay < 1 || nDay > nLastDay {
				nDay = 0
			}
			date := itoa(curYear) + "-" + sprintf02(curMonth) + "-" + sprintf02(nDay)
			week.Days = append(week.Days, &CalDay{
				Day:        nDay,
				Date:       date,
				IsToday:    date == today,
				IsFirstDay: !a.SettingEmpty("cal_showweeknum") && (firstDayOfWeek+nDay-1)%7 == nStartDay,
				Holidays:   holidays[date],
				Events:     events[date],
				Birthdays:  bday[date],
			})
		}
		page.Weeks = append(page.Weeks, week)
	}

	// Previous month.
	if curMonth > 1 || (curMonth == 1 && curYear > a.SettingInt("cal_minyear")) {
		py, pm := curYear, curMonth-1
		if curMonth == 1 {
			py, pm = curYear-1, 12
		}
		page.HasPrev = true
		page.PrevYear, page.PrevMonth = py, pm
		page.PrevHref = scripturl + "?action=calendar;year=" + itoa(py) + ";month=" + itoa(pm)
	}
	// Next month.
	if curMonth < 12 || (curMonth == 12 && curYear < a.SettingInt("cal_maxyear")) {
		ny, nm := curYear, curMonth+1
		if curMonth == 12 {
			ny, nm = curYear+1, 1
		}
		page.HasNext = true
		page.NextYear, page.NextMonth = ny, nm
		page.NextHref = scripturl + "?action=calendar;year=" + itoa(ny) + ";month=" + itoa(nm)
	}

	c.SubTemplate = templateCalendarMain
}

// calendarBirthdayArray is calendarBirthdayArray($low, $high).
func (c *Ctx) calendarBirthdayArray(low, high string) map[string][]*CalBirthday {
	a := c.App

	lowMD := low[4:]   // "-MM-DD"
	highMD := high[4:] // "-MM-DD"
	yearLow := atoi(low[0:4])
	yearHigh := atoi(high[0:4])

	var allyear string
	if low[0:4] != high[0:4] {
		allyear = "birthdate BETWEEN '0004" + lowMD + "' AND '0004-12-31' OR birthdate BETWEEN '0004-01-01' AND '0004" + highMD + "'"
	} else {
		allyear = "birthdate BETWEEN '0004" + lowMD + "' AND '0004" + highMD + "'"
	}

	cond := "('" + itoa(yearLow) + "' || SUBSTR(birthdate, 5)) BETWEEN '" + low + "' AND '" + high + "'"
	if yearLow != yearHigh {
		cond += "\n\t\t\t\tOR ('" + itoa(yearHigh) + "' || SUBSTR(birthdate, 5)) BETWEEN '" + low + "' AND '" + high + "'"
	}

	result := map[string][]*CalBirthday{}
	var order []string
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, realName, YEAR(birthdate) AS birthYear, birthdate
		FROM {$db_prefix}members
		WHERE YEAR(birthdate) != 1
			AND (` + allyear + `
				OR ` + cond + `)
			AND is_activated = 1`))
	if err == nil {
		for rows.Next() {
			var id, birthYear int
			var realName, birthdate string
			rows.Scan(&id, &realName, &birthYear, &birthdate)

			ageYear := yearLow
			if yearLow != yearHigh && birthdate[5:] < high[5:] {
				ageYear = yearHigh
			}
			b := &CalBirthday{ID: id, Name: realName}
			if birthYear > 4 && birthYear <= ageYear {
				b.Age = ageYear - birthYear
				b.HasAge = true
			}
			key := itoa(ageYear) + birthdate[4:]
			if _, ok := result[key]; !ok {
				order = append(order, key)
			}
			result[key] = append(result[key], b)
		}
		rows.Close()
	}
	for _, k := range order {
		result[k][len(result[k])-1].IsLast = true
	}
	return result
}

// calendarHolidayArray is calendarHolidayArray($low, $high).
func (c *Ctx) calendarHolidayArray(low, high string) map[string][]string {
	a := c.App

	lowMD := low[4:]
	highMD := high[4:]
	var allyear string
	if low[0:4] != high[0:4] {
		allyear = "eventDate BETWEEN '0004" + lowMD + "' AND '0004-12-31' OR eventDate BETWEEN '0004-01-01' AND '0004" + highMD + "'"
	} else {
		allyear = "eventDate BETWEEN '0004" + lowMD + "' AND '0004" + highMD + "'"
	}

	result := map[string][]string{}
	rows, err := a.DB.Query(a.Q(`
		SELECT eventDate, YEAR(eventDate) AS year, title
		FROM {$db_prefix}calendar_holidays
		WHERE eventDate BETWEEN '` + low + `' AND '` + high + `'
			OR ` + allyear))
	if err == nil {
		for rows.Next() {
			var eventDate, title string
			var year int
			rows.Scan(&eventDate, &year, &title)
			eventYear := low[0:4]
			if low[0:4] != high[0:4] && eventDate[5:] < high[5:] {
				eventYear = high[0:4]
			}
			key := eventYear + eventDate[4:]
			result[key] = append(result[key], title)
		}
		rows.Close()
	}
	return result
}

// calendarEventArray is calendarEventArray($low, $high, true) — the
// permission-aware (calendar page) variant.
func (c *Ctx) calendarEventArray(low, high string, usePermissions bool) map[string][]*CalEvent {
	a := c.App
	scripturl := a.ScriptURL

	lowTime, _ := time.Parse("2006-01-02", low)
	highTime, _ := time.Parse("2006-01-02", high)

	seeBoard := ""
	if usePermissions {
		seeBoard = "\n\t\t\tAND (cal.ID_BOARD = 0 OR " + c.User.QuerySeeBoard + ")"
	}

	result := map[string][]*CalEvent{}
	var order []string
	rows, err := a.DB.Query(a.Q(`
		SELECT
			cal.ID_EVENT, cal.startDate, cal.endDate, cal.title, cal.ID_MEMBER, cal.ID_TOPIC,
			cal.ID_BOARD, IFNULL(b.memberGroups, '') AS memberGroups, IFNULL(t.ID_FIRST_MSG, 0) AS ID_FIRST_MSG
		FROM {$db_prefix}calendar AS cal
			LEFT JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = cal.ID_BOARD)
			LEFT JOIN {$db_prefix}topics AS t ON (t.ID_TOPIC = cal.ID_TOPIC)
		WHERE cal.startDate <= '` + high + `'
			AND cal.endDate >= '` + low + `'` + seeBoard))
	if err != nil {
		return result
	}
	for rows.Next() {
		var idEvent, idMember, idTopic, idBoard, idFirstMsg int
		var startDate, endDate, title, memberGroups string
		rows.Scan(&idEvent, &startDate, &endDate, &title, &idMember, &idTopic, &idBoard, &memberGroups, &idFirstMsg)

		title = c.censorText(title)
		sd, _ := time.Parse("2006-01-02", startDate)
		ed, _ := time.Parse("2006-01-02", endDate)
		if sd.Before(lowTime) {
			sd = lowTime
		}
		if ed.After(highTime) {
			ed = highTime
		}

		for d := sd; !d.After(ed); d = d.AddDate(0, 0, 1) {
			key := d.Format("2006-01-02")
			ev := &CalEvent{
				ID:        idEvent,
				Title:     title,
				Topic:     idTopic,
				StartDate: startDate,
				EndDate:   endDate,
			}
			if idBoard == 0 {
				ev.Link = title
			} else {
				ev.Href = scripturl + "?topic=" + itoa(idTopic) + ".0"
				ev.Link = `<a href="` + ev.Href + `">` + title + `</a>`
			}
			ev.CanEdit = c.allowedTo("calendar_edit_any") || (idMember == c.User.ID && c.allowedTo("calendar_edit_own"))
			if idBoard == 0 {
				ev.ModifyHref = scripturl + "?action=calendar;sa=post;eventid=" + itoa(idEvent) + ";sesc=" + c.Sc
			} else {
				ev.ModifyHref = scripturl + "?action=post;msg=" + itoa(idFirstMsg) + ";topic=" + itoa(idTopic) + ".0;calendar;eventid=" + itoa(idEvent) + ";sesc=" + c.Sc
			}
			if _, ok := result[key]; !ok {
				order = append(order, key)
			}
			result[key] = append(result[key], ev)
		}
	}
	rows.Close()

	for _, k := range order {
		result[k][len(result[k])-1].IsLast = true
	}
	return result
}

// EventBoard is one selectable board in the calendar event-post form's
// "link to board" dropdown ($context['event']['boards'][]).
type EventBoard struct {
	ID         int
	Name       string
	ChildLevel int
	Prefix     string
	CatName    string
}

// EventContext is $context['event'] — shared by the standalone event-post
// form (template_event_post) and the make_event blocks in the Post() form.
type EventContext struct {
	Title   string
	ID      int
	New     bool
	Month   int
	Day     int
	Year    int
	Span    int
	LastDay int
	Board   int
	Boards  []EventBoard
}

// EventPostCtx is the page context for the standalone template_event_post.
type EventPostCtx struct {
	Event         *EventContext
	PostError     map[string]bool
	ErrorMessages []string
	ErrorType     string
}

// lastDayOfMonth returns the last day-of-month number for the given month.
func lastDayOfMonth(year, month int) int {
	ny, nm := year, month+1
	if month == 12 {
		ny, nm = year+1, 1
	}
	return mktimeUTC(ny, nm, 1).AddDate(0, 0, -1).Day()
}

// calendarInsertEvent is calendarInsertEvent(): inserts a new calendar event.
func (c *Ctx) calendarInsertEvent(idBoard, idTopic int, title string, idMember, month, day, year int, span string) {
	a := c.App

	// Add special chars to the title.
	title = Htmlspecialchars(title)

	// Add some sanity checking to the span.
	nSpan := 0
	if !empty(span) && strings.TrimSpace(span) != "" {
		nSpan = atoi(span) - 1
		if mx := a.SettingInt("cal_maxspan"); nSpan > mx {
			nSpan = mx
		}
	}

	start := mktimeUTC(year, month, day)
	end := start.AddDate(0, 0, nSpan)
	if len(title) > 48 {
		title = title[:48]
	}

	a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}calendar
			(ID_BOARD, ID_TOPIC, title, ID_MEMBER, startDate, endDate)
		VALUES (?, ?, ?, ?, ?, ?)`),
		idBoard, idTopic, title, idMember, start.Format("2006-01-02"), end.Format("2006-01-02"))

	a.updateStatsCalendar()
}

// calendarCanLink is calendarCanLink(): may this user link $topic in $board?
func (c *Ctx) calendarCanLink() bool {
	a := c.App

	// If you can't post, you can't link.
	c.isAllowedTo("calendar_post")

	// No board?  No topic?!?
	if c.Board == 0 {
		c.fatalLangError("calendar38", false)
	}
	if c.Topic == 0 {
		c.fatalLangError("calendar39", false)
	}

	// Administrator, Moderator, or owner.  Period.
	if !c.allowedTo("admin_forum") && !c.allowedTo("moderate_board") {
		var starter int
		err := a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER_STARTED
			FROM {$db_prefix}topics
			WHERE ID_TOPIC = ?
			LIMIT 1`), c.Topic).Scan(&starter)
		if err == nil {
			// Not the owner of the topic.
			if starter != c.User.ID {
				c.fatalLangError("calendar41", true)
			}
		} else {
			// Topic/Board doesn't exist.....
			c.fatalLangError("calendar40", true)
		}
	}

	return true
}

// calendarValidatePost is calendarValidatePost() from Subs-Post.php.
func (c *Ctx) calendarValidatePost() {
	a := c.App

	if !c.POST.Has("deleteevent") {
		// No month?  No year?
		if !c.POST.Has("month") {
			c.fatalLangError("calendar7", false)
		}
		if !c.POST.Has("year") {
			c.fatalLangError("calendar8", false)
		}

		// Check the month and year...
		if c.POST.Int("month") < 1 || c.POST.Int("month") > 12 {
			c.fatalLangError("calendar1", false)
		}
		if c.POST.Int("year") < a.SettingInt("cal_minyear") || c.POST.Int("year") > a.SettingInt("cal_maxyear") {
			c.fatalLangError("calendar2", false)
		}
	}

	// Make sure they're allowed to post...
	c.isAllowedTo("calendar_post")

	if c.POST.Has("span") {
		// Make sure it's turned on and not some fool trying to trick it.
		if a.SettingEmpty("cal_allowspan") {
			c.fatalLangError("calendar55", false)
		}
		if c.POST.Int("span") < 1 || c.POST.Int("span") > a.SettingInt("cal_maxspan") {
			c.fatalLangError("calendar56", false)
		}
	}

	// There is no need to validate the following values if we are just deleting the event.
	if !c.POST.Has("deleteevent") {
		// No day?
		if !c.POST.Has("day") {
			c.fatalLangError("calendar14", false)
		}
		if !c.POST.Has("evtitle") && !c.POST.Has("subject") {
			c.fatalLangError("calendar15", false)
		} else if !c.POST.Has("evtitle") {
			c.POST.Set("evtitle", c.POST.Str("subject"))
		}

		// Bad day?
		if !checkdate(c.POST.Int("month"), c.POST.Int("day"), c.POST.Int("year")) {
			c.fatalLangError("calendar16", false)
		}

		// No title?
		if Htmltrim(c.POST.Str("evtitle")) == "" {
			c.fatalLangError("calendar17", false)
		}
		if entityLen(c.POST.Str("evtitle")) > 30 {
			c.POST.Set("evtitle", entitySubstr(c.POST.Str("evtitle"), 0, 30))
		}
	}
}

// CalendarPost is CalendarPost(): posting/editing/deleting a calendar event.
func (c *Ctx) CalendarPost() {
	a := c.App

	// Well - can they?
	c.isAllowedTo("calendar_post")

	hasEventID := c.REQUEST.Has("eventid")
	eventID := c.REQUEST.Int("eventid")

	// Submitting?
	if c.POST.Has("sc") && hasEventID {
		c.checkSession("post", "", true)

		// Validate the post...
		if !c.POST.Has("link_to_board") {
			c.calendarValidatePost()
		}

		// If you're not allowed to edit any events, you have to be the poster.
		if eventID > 0 && !c.allowedTo("calendar_edit_any") {
			// Get the event's poster.
			var poster int
			a.DB.QueryRow(a.Q(`
				SELECT ID_MEMBER
				FROM {$db_prefix}calendar
				WHERE ID_EVENT = ?
				LIMIT 1`), eventID).Scan(&poster)

			// Finally, test if they can either edit ANY, or just their own...
			if poster == c.User.ID {
				c.isAllowedTo("calendar_edit_own")
			} else {
				c.isAllowedTo("calendar_edit_any")
			}
		}

		switch {
		case eventID == -1 && c.POST.Has("link_to_board"):
			// New - and directing?
			c.REQUEST.Set("calendar", "1")
			c.Post()
			return
		case eventID == -1:
			// New...
			c.calendarInsertEvent(0, 0, c.POST.Str("evtitle"), c.User.ID, c.POST.Int("month"), c.POST.Int("day"), c.POST.Int("year"), c.POST.Str("span"))
		case c.REQUEST.Has("deleteevent"):
			// Deleting...
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}calendar WHERE ID_EVENT = ?`), eventID)
		default:
			// ... or just update it?
			nSpan := 0
			if !a.SettingEmpty("cal_allowspan") && !empty(c.POST.Str("span")) && c.POST.Str("span") != "1" &&
				!a.SettingEmpty("cal_maxspan") && c.POST.Int("span") <= a.SettingInt("cal_maxspan") {
				nSpan = c.POST.Int("span") - 1
				if mx := a.SettingInt("cal_maxspan"); nSpan > mx {
					nSpan = mx
				}
			}
			start := mktimeUTC(c.REQUEST.Int("year"), c.REQUEST.Int("month"), c.REQUEST.Int("day"))
			end := start.AddDate(0, 0, nSpan)

			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}calendar
				SET startDate = ?, endDate = ?, title = ?
				WHERE ID_EVENT = ?`),
				start.Format("2006-01-02"), end.Format("2006-01-02"), Htmlspecialchars(c.REQUEST.Str("evtitle")), eventID)
		}

		a.updateStatsCalendar()

		// No point hanging around here now...
		c.redirectExit("action=calendar;month=" + c.POST.Str("month") + ";year=" + c.POST.Str("year"))
	}

	// If we are not enabled... we are not enabled.
	if a.SettingEmpty("cal_allow_unlinked") && empty(c.REQUEST.Str("eventid")) {
		c.REQUEST.Set("calendar", "1")
		c.Post()
		return
	}

	event := &EventContext{}
	page := &EventPostCtx{Event: event, PostError: map[string]bool{}}
	c.Page = page

	// New?
	if !hasEventID {
		today := time.Unix(c.forumTime(true, 0), 0).UTC()

		event.New = true
		event.ID = -1
		event.Year = today.Year()
		event.Month = int(today.Month())
		event.Day = today.Day()
		event.Span = 1
		event.Board = a.SettingInt("cal_defaultboard")
		if c.REQUEST.Has("year") {
			event.Year = c.REQUEST.Int("year")
		}
		if c.REQUEST.Has("month") {
			event.Month = c.REQUEST.Int("month")
		}
		if c.REQUEST.Has("day") {
			event.Day = c.REQUEST.Int("day")
		}

		// Get list of boards that can be posted in.
		boards := c.boardsAllowedTo("post_new")
		if len(boards) == 0 {
			c.fatalLangError("cannot_post_new", false)
		}
		event.Boards = c.eventBoardList(boards)
	} else {
		var idBoard, idTopic, idMember, idFirstMsg, starter int
		var title, startDate, endDate string
		err := a.DB.QueryRow(a.Q(`
			SELECT
				c.ID_BOARD, c.ID_TOPIC, c.startDate, c.endDate, c.ID_MEMBER, c.title,
				IFNULL(t.ID_FIRST_MSG, 0), IFNULL(t.ID_MEMBER_STARTED, 0)
			FROM {$db_prefix}calendar AS c
				LEFT JOIN {$db_prefix}topics AS t ON (t.ID_TOPIC = c.ID_TOPIC)
			WHERE c.ID_EVENT = ?`), eventID).Scan(&idBoard, &idTopic, &startDate, &endDate, &idMember, &title, &idFirstMsg, &starter)
		// If nothing returned, we are in poo, poo.
		if err != nil {
			c.fatalLangError("1", true)
		}

		// If it has a board, then they should be editing it within the topic.
		if idTopic != 0 && idFirstMsg != 0 {
			// We load the board up, for a check on the board access rights...
			c.Topic = idTopic
			c.loadBoard()
		}

		// Make sure the user is allowed to edit this event.
		if idMember != c.User.ID {
			c.isAllowedTo("calendar_edit_any")
		} else if !c.allowedTo("calendar_edit_any") {
			c.isAllowedTo("calendar_edit_own")
		}

		sd, _ := time.Parse("2006-01-02", startDate)
		ed, _ := time.Parse("2006-01-02", endDate)
		event.New = false
		event.ID = eventID
		event.Board = idBoard
		event.Year = sd.Year()
		event.Month = int(sd.Month())
		event.Day = sd.Day()
		event.Title = title
		event.Span = 1 + int(ed.Sub(sd).Hours()/24)
	}

	event.LastDay = lastDayOfMonth(event.Year, event.Month)

	c.SubTemplate = templateEventPost

	if hasEventID {
		c.PageTitle = c.Txt("calendar20")
	} else {
		c.PageTitle = c.Txt("calendar23")
	}
	c.LinkTree = append(c.LinkTree, Link{Name: c.PageTitle})
}

// eventBoardList builds $context['event']['boards'] for the given allowed
// board set (0 in the set means "all boards").
func (c *Ctx) eventBoardList(boards []int) []EventBoard {
	a := c.App

	q := `
		SELECT c.name AS catName, c.ID_CAT, b.ID_BOARD, b.name AS boardName, b.childLevel
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE ` + c.User.QuerySeeBoard
	if !intIn(boards, 0) {
		q += "\n\t\t\t\tAND b.ID_BOARD IN (" + joinInts(boards) + ")"
	}

	var list []EventBoard
	rows, err := a.DB.Query(a.Q(q))
	if err == nil {
		for rows.Next() {
			var catName, boardName string
			var idCat, idBoard, childLevel int
			rows.Scan(&catName, &idCat, &idBoard, &boardName, &childLevel)
			list = append(list, EventBoard{
				ID:         idBoard,
				Name:       boardName,
				ChildLevel: childLevel,
				Prefix:     strings.Repeat("&nbsp;", childLevel*3),
				CatName:    catName,
			})
		}
		rows.Close()
	}
	return list
}
