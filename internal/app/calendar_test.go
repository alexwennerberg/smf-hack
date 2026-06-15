package app

// Smoke tests for the calendar month-grid view (?action=calendar).

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCalendarGrid(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"cal_enabled": "1"})

	// cal_maxyear defaults to 2010; ask for a month within range.
	w, body := get(t, a, "/index.php?action=calendar;year=2010;month=6", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("calendar status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{
		"June 2010",                          // month title
		`?action=calendar;year=2010;month=5`, // previous month
		`?action=calendar;year=2010;month=7`, // next month
		`<select name="month">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("calendar grid missing %q", want)
		}
	}
}

func TestCalendarBirthday(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"cal_enabled": "1", "cal_showbdaysoncalendar": "1"})
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET birthdate = '1990-06-15' WHERE ID_MEMBER = 1`))

	_, body := get(t, a, "/index.php?action=calendar;year=2010;month=6", adminCookie(t, a))
	// The admin's birthday should appear with their age (20 in 2010), inside
	// the profile link.
	if !strings.Contains(body, "?action=profile;u=1\">admin (20)</a>") {
		t.Errorf("calendar birthday not shown:\n%.1200s", body)
	}
}

func TestCalendarDisabled(t *testing.T) {
	a := newTestApp(t)
	// cal_enabled defaults to 0 -> calendar_off fatal error (still 200).
	w, _ := get(t, a, "/index.php?action=calendar;year=2010;month=6", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("disabled calendar status %d", w.Code)
	}
}

// TestCalendarEventPostUnlinked exercises the standalone CalendarPost flow:
// the new-event form, an unlinked insert, the edit form, an update, a delete.
func TestCalendarEventPostUnlinked(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"cal_enabled": "1", "cal_allow_unlinked": "1"})
	admin := adminCookie(t, a)

	// The new-event form renders (postevent form, not the post form).
	w, body := get(t, a, "/index.php?action=calendar;sa=post", admin)
	if w.Code != 200 {
		t.Fatalf("event form status %d:\n%.400s", w.Code, body)
	}
	for _, want := range []string{`name="postevent"`, `name="evtitle"`, `name="link_to_board"`, `<select name="year"`} {
		if !strings.Contains(body, want) {
			t.Errorf("event post form missing %q", want)
		}
	}
	m := scRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no sc in event form:\n%.400s", body)
	}
	sc := m[1]
	cookies := append([]*http.Cookie{admin}, cookiesFrom(w)...)

	// Post a new UNLINKED event (no link_to_board checkbox).
	w = postForm(t, a, "/index.php?action=calendar;sa=post", url.Values{
		"eventid":  {"-1"},
		"evtitle":  {"Go Release Day"},
		"calendar": {"1"},
		"year":     {"2010"},
		"month":    {"6"},
		"day":      {"15"},
		"sc":       {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("event insert: status %d body %.400s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); !strings.Contains(loc, "action=calendar;month=6;year=2010") {
		t.Fatalf("event insert redirect %q", loc)
	}

	var id, board, topic int
	var title, start string
	if err := a.DB.QueryRow(a.Q(`SELECT ID_EVENT, ID_BOARD, ID_TOPIC, title, startDate FROM {$db_prefix}calendar ORDER BY ID_EVENT DESC LIMIT 1`)).Scan(&id, &board, &topic, &title, &start); err != nil {
		t.Fatalf("event not inserted: %v", err)
	}
	if title != "Go Release Day" || board != 0 || topic != 0 || start != "2010-06-15" {
		t.Fatalf("bad event: id=%d board=%d topic=%d title=%q start=%q", id, board, topic, title, start)
	}

	// The edit form prefills the existing event.
	_, body = get(t, a, "/index.php?action=calendar;sa=post;eventid="+itoa(id)+";sesc="+sc, cookies...)
	if !strings.Contains(body, `value="Go Release Day"`) {
		t.Errorf("edit form not prefilled:\n%.600s", body)
	}

	// Update it.
	w = postForm(t, a, "/index.php?action=calendar;sa=post", url.Values{
		"eventid":  {itoa(id)},
		"evtitle":  {"Go 2.0 Day"},
		"calendar": {"1"},
		"year":     {"2010"},
		"month":    {"7"},
		"day":      {"4"},
		"sc":       {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("event update: status %d body %.400s", w.Code, w.Body.String())
	}
	a.DB.QueryRow(a.Q(`SELECT title, startDate FROM {$db_prefix}calendar WHERE ID_EVENT = ?`), id).Scan(&title, &start)
	if title != "Go 2.0 Day" || start != "2010-07-04" {
		t.Fatalf("event not updated: title=%q start=%q", title, start)
	}

	// Delete it.
	w = postForm(t, a, "/index.php?action=calendar;sa=post", url.Values{
		"eventid":     {itoa(id)},
		"deleteevent": {"1"},
		"year":        {"2010"},
		"month":       {"7"},
		"sc":          {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("event delete: status %d", w.Code)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}calendar WHERE ID_EVENT = ?`), id).Scan(&n)
	if n != 0 {
		t.Fatalf("event not deleted")
	}
}

// TestCalendarEventLinkedPost exercises the make_event integration in the post
// form + Post2: posting a new topic that also creates a linked calendar event.
func TestCalendarEventLinkedPost(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{"cal_enabled": "1"})
	admin := adminCookie(t, a)

	// The post form, calendar-enabled, shows the event fields. (cal_maxyear
	// defaults to 2010, so request an in-range year for the make_event form.)
	formPath := "/index.php?action=post;board=1.0;calendar;year=2010;month=8"
	sc, seq, cookies := openPostForm(t, a, formPath, admin)
	{
		_, body := get(t, a, formPath, admin)
		for _, want := range []string{`name="evtitle"`, `name="calendar" value="1"`, `<select name="board">`} {
			if !strings.Contains(body, want) {
				t.Errorf("make_event post form missing %q", want)
			}
		}
	}

	w := postForm(t, a, "/index.php?action=post2;start=0;board=1.0", url.Values{
		"topic":              {"0"},
		"subject":            {"Linked event topic"},
		"message":            {"This topic has a calendar event."},
		"icon":               {"xx"},
		"calendar":           {"1"},
		"eventid":            {"-1"},
		"evtitle":            {"Linked Event"},
		"year":               {"2010"},
		"month":              {"8"},
		"day":                {"20"},
		"additional_options": {"0"},
		"sc":                 {sc},
		"seqnum":             {seq},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("post2 linked event: status %d body %.500s", w.Code, w.Body.String())
	}

	var board, topic int
	var title, start string
	if err := a.DB.QueryRow(a.Q(`SELECT ID_BOARD, ID_TOPIC, title, startDate FROM {$db_prefix}calendar ORDER BY ID_EVENT DESC LIMIT 1`)).Scan(&board, &topic, &title, &start); err != nil {
		t.Fatalf("linked event not inserted: %v", err)
	}
	if board != 1 || topic == 0 || title != "Linked Event" || start != "2010-08-20" {
		t.Fatalf("bad linked event: board=%d topic=%d title=%q start=%q", board, topic, title, start)
	}
}

// TestBoardIndexCalendar exercises calendarDoIndex(): the board-index
// info-center block that lists today's holidays, birthdays, and events when
// the cal_show*onindex flags are on. Dates are pinned to the real current day
// so the block (which uses forum_time, not a URL date) has something to show.
func TestBoardIndexCalendar(t *testing.T) {
	a := newTestApp(t)
	a.UpdateSettings(map[string]string{
		"cal_enabled":             "1",
		"cal_showholidaysonindex": "1",
		"cal_showbdaysonindex":    "1",
		"cal_showeventsonindex":   "1",
	})

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	md := today[4:] // "-MM-DD"
	// Birth year chosen so the age is a fixed 25 regardless of the wall clock.
	birthYear := now.Year() - 25

	// A recurring holiday (year 0004) today, a birthday today, an event today.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}calendar_holidays (eventDate, title) VALUES ('0004` + md + `', 'Test Holiday')`))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET birthdate = '` + itoa(birthYear) + md + `' WHERE ID_MEMBER = 1`))
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}calendar (startDate, endDate, ID_BOARD, ID_TOPIC, title, ID_MEMBER) VALUES ('` + today + `', '` + today + `', 0, 0, 'Test Event', 1)`))

	w, body := get(t, a, "/index.php", adminCookie(t, a))
	if w.Code != 200 {
		t.Fatalf("board index status %d", w.Code)
	}
	for _, want := range []string{
		`/icons/calendar.gif`,                        // the block header icon
		`style="color: #000080;"`,                    // cal_holidaycolor span (holidays)
		`Test Holiday`,                               // the holiday title
		`style="color: #920AC4;"`,                    // cal_bdaycolor span (birthdays)
		`?action=profile;u=1"><b>admin</b> (25)</a>`, // today's birthday (bolded) with age
		`style="color: #078907;"`,                    // cal_eventcolor span (events)
		`Test Event`,                                 // the event title
	} {
		if !strings.Contains(body, want) {
			t.Errorf("board-index calendar block missing %q:\n%.2000s", want, body)
		}
	}
}
