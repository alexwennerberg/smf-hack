package app

// Port of Sources/ManageCalendar.php: the calendar admin
// (?action=managecalendar) — holiday management + calendar settings (with the
// inline calendar_* permissions). Holiday dates are 'YYYY-MM-DD' TEXT; year
// 0004 means "every year". Event posting (the calendar itself) is elsewhere.

import "time"

func init() {
	registerAction("managecalendar", (*Ctx).ManageCalendar)
}

func (c *Ctx) ManageCalendar() {
	a := c.App
	c.isAllowedTo("admin_forum")
	c.adminIndex("manage_calendar")
	c.loadLanguage("ManageCalendar")

	sa := c.REQUEST.Str("sa")
	if sa != "holidays" && sa != "editholiday" && sa != "settings" {
		sa = "settings"
	}

	scripturl := a.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("manage_calendar"), Help: "calendar", Description: c.Txt("calendar_settings_desc")}
	if !a.SettingEmpty("cal_enabled") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("manage_holidays"), Description: c.Txt("manage_holidays_desc"), Href: scripturl + "?action=managecalendar;sa=holidays", IsSelected: sa == "holidays" || sa == "editholiday"})
	}
	tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("calendar_settings"), Description: c.Txt("calendar_settings_desc"), Href: scripturl + "?action=managecalendar;sa=settings", IsSelected: sa == "settings", IsLast: true})
	c.AdminTabs = tabs

	switch sa {
	case "holidays":
		c.ModifyHolidays()
	case "editholiday":
		c.EditHoliday()
	default:
		c.ModifyCalSettings()
	}
}

// CalHoliday is one row in the holiday list.
type CalHoliday struct {
	ID    int
	Date  string
	Title string
}

// ModifyHolidaysCtx backs template_manage_holidays.
type ModifyHolidaysCtx struct {
	Holidays  []CalHoliday
	PageIndex string
}

func (c *Ctx) ModifyHolidays() {
	a := c.App
	c.PageTitle = c.Txt("manage_holidays")

	if c.REQUEST.Has("delete") && c.REQUEST.Arr("holiday") != nil {
		c.checkSession("post", "", true)
		var ids []int
		c.REQUEST.Arr("holiday").Values(func(k string, v any) {
			ids = append(ids, atoi(k))
		})
		if len(ids) > 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}calendar_holidays WHERE ID_HOLIDAY IN (` + joinInts(ids) + `)`))
			a.updateStatsCalendar()
		}
	}

	var total int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}calendar_holidays`)).Scan(&total)

	page := &ModifyHolidaysCtx{}
	c.Page = page
	page.PageIndex, _ = c.constructPageIndex(a.ScriptURL+"?action=managecalendar;sa=holidays", c.REQUEST.Int("start"), total, 20, false)

	rows, err := a.DB.Query(a.Q(`
		SELECT ID_HOLIDAY, YEAR(eventDate) AS year, MONTH(eventDate) AS month, DAYOFMONTH(eventDate) AS day, title
		FROM {$db_prefix}calendar_holidays
		ORDER BY title
		LIMIT ? OFFSET ?`), 20, c.REQUEST.Int("start"))
	if err == nil {
		for rows.Next() {
			var id, year, month, day int
			var title string
			rows.Scan(&id, &year, &month, &day, &title)
			date := itoa(day) + " " + c.TxtListItem("months", month) + " "
			if year == 4 {
				date += "(" + c.Txt("every_year") + ")"
			} else {
				date += itoa(year)
			}
			page.Holidays = append(page.Holidays, CalHoliday{ID: id, Date: date, Title: title})
		}
		rows.Close()
	}

	c.SubTemplate = templateManageHolidays
}

// EditHolidayCtx backs template_edit_holiday.
type EditHolidayCtx struct {
	IsNew   bool
	ID      int
	Day     int
	Month   int
	Year    int
	Title   string
	LastDay int
}

func (c *Ctx) EditHoliday() {
	a := c.App
	isNew := !c.REQUEST.Has("holiday")
	holidayID := c.REQUEST.Int("holiday")

	if c.POST.Has("sc") && (c.REQUEST.Has("delete") || c.POST.Str("title") != "") {
		c.checkSession("post", "", true)
		if c.REQUEST.Has("delete") {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}calendar_holidays WHERE ID_HOLIDAY = ?`), holidayID)
		} else {
			yr := c.POST.Int("year")
			if yr <= 4 {
				yr = 4
			}
			date := zeroPad(yr, 4) + "-" + zeroPad(c.POST.Int("month"), 2) + "-" + zeroPad(c.POST.Int("day"), 2)
			if c.POST.Has("edit") {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}calendar_holidays SET eventDate = ?, title = ? WHERE ID_HOLIDAY = ?`), date, c.POST.Str("title"), holidayID)
			} else {
				a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}calendar_holidays (eventDate, title) VALUES (?, SUBSTR(?, 1, 48))`), date, c.POST.Str("title"))
			}
		}
		a.updateStatsCalendar()
		c.redirectExit("action=managecalendar;sa=holidays")
	}

	page := &EditHolidayCtx{IsNew: isNew}
	c.Page = page
	if isNew {
		page.Title = c.Txt("holidays_add")
		now := time.Now()
		page.Day = now.Day()
		page.Month = int(now.Month())
		page.Year = 0
	} else {
		c.PageTitle = c.Txt("holidays_edit")
		var year, month, day int
		var title string
		a.DB.QueryRow(a.Q(`
			SELECT ID_HOLIDAY, YEAR(eventDate), MONTH(eventDate), DAYOFMONTH(eventDate), title
			FROM {$db_prefix}calendar_holidays WHERE ID_HOLIDAY = ? LIMIT 1`), holidayID).Scan(&page.ID, &year, &month, &day, &title)
		if year <= 4 {
			year = 0
		}
		page.Year = year
		page.Month = month
		page.Day = day
		page.Title = title
	}
	if c.PageTitle == "" {
		if isNew {
			c.PageTitle = c.Txt("holidays_add")
		} else {
			c.PageTitle = c.Txt("holidays_edit")
		}
	}

	// Last day of the selected month (for the day dropdown).
	yr := page.Year
	if yr == 0 {
		yr = 2004 // a leap year for "every year" so 29 Feb is allowed
	}
	page.LastDay = time.Date(yr, time.Month(page.Month)+1, 0, 0, 0, 0, 0, time.UTC).Day()

	c.SubTemplate = templateEditHoliday
}

// zeroPad left-pads n to width with zeros.
func zeroPad(n, width int) string {
	s := itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// CalSettingsCtx backs template_modify_settings.
type CalSettingsCtx struct {
	CalBoards    [][2]string // {id, "Cat - Board"}, with a blank entry 0
	ShowHolidays string
	ShowBdays    string
	ShowEvents   string
}

func (c *Ctx) ModifyCalSettings() {
	a := c.App
	c.PageTitle = c.Txt("calendar_settings")
	calPerms := []string{"calendar_view", "calendar_post", "calendar_edit_own", "calendar_edit_any"}

	if c.POST.Has("cal_days_for_index") && c.POST.Has("sc") {
		c.checkSession("post", "", true)

		showFlag := func(field, mode string) string {
			v := c.POST.Str(field)
			if v == mode || v == "all" {
				return "1"
			}
			return "0"
		}
		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		a.UpdateSettings(map[string]string{
			"cal_enabled":                bit("cal_enabled"),
			"cal_daysaslink":             bit("cal_daysaslink"),
			"cal_showweeknum":            bit("cal_showweeknum"),
			"cal_days_for_index":         itoa(c.POST.Int("cal_days_for_index")),
			"cal_showholidaysonindex":    showFlag("cal_showholidays", "index"),
			"cal_showbdaysonindex":       showFlag("cal_showbdays", "index"),
			"cal_showeventsonindex":      showFlag("cal_showevents", "index"),
			"cal_showholidaysoncalendar": showFlag("cal_showholidays", "cal"),
			"cal_showbdaysoncalendar":    showFlag("cal_showbdays", "cal"),
			"cal_showeventsoncalendar":   showFlag("cal_showevents", "cal"),
			"cal_defaultboard":           itoa(c.POST.Int("cal_defaultboard")),
			"cal_allow_unlinked":         bit("cal_allow_unlinked"),
			"cal_minyear":                itoa(c.POST.Int("cal_minyear")),
			"cal_maxyear":                itoa(c.POST.Int("cal_maxyear")),
			"cal_bdaycolor":              c.POST.Str("cal_bdaycolor"),
			"cal_eventcolor":             c.POST.Str("cal_eventcolor"),
			"cal_holidaycolor":           c.POST.Str("cal_holidaycolor"),
			"cal_allowspan":              bit("cal_allowspan"),
			"cal_maxspan":                itoa(c.POST.Int("cal_maxspan")),
			"cal_showInTopic":            bit("cal_showInTopic"),
		})
		c.saveInlinePermissions(calPerms)
		a.updateStatsCalendar()
		c.redirectExit("action=managecalendar;sa=settings")
	}

	page := &CalSettingsCtx{}
	c.Page = page
	// PHP builds $context['cal_boards'] keyed by board ID with a leading key 0
	// (empty name) for the "no default" choice, so the option value is "0".
	page.CalBoards = [][2]string{{"0", ""}}
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name AS bName, c.name AS cName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)`))
	if err == nil {
		for rows.Next() {
			var id int
			var bName, cName string
			rows.Scan(&id, &bName, &cName)
			page.CalBoards = append(page.CalBoards, [2]string{itoa(id), cName + " - " + bName})
		}
		rows.Close()
	}

	// SMF's ManageCalendar calls init_inline_permissions($calendarPermissions)
	// with no excluded groups, so Guests (-1) appear in the calendar perms.
	c.initInlinePermissions(calPerms, nil)

	showState := func(onIndex, onCal string) string {
		if a.SettingEmpty(onIndex) {
			if a.SettingEmpty(onCal) {
				return "never"
			}
			return "cal"
		}
		if a.SettingEmpty(onCal) {
			return "index"
		}
		return "all"
	}
	page.ShowHolidays = showState("cal_showholidaysonindex", "cal_showholidaysoncalendar")
	page.ShowBdays = showState("cal_showbdaysonindex", "cal_showbdaysoncalendar")
	page.ShowEvents = showState("cal_showeventsonindex", "cal_showeventsoncalendar")

	c.SubTemplate = templateModifyCalSettings
}
