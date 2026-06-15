package app

// Phase 7: calendar admin (holidays + settings).

import (
	"net/url"
	"strings"
	"testing"
)

func TestCalSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=managecalendar;sa=settings", admin)
	w := postForm(t, a, "/index.php?action=managecalendar;sa=settings", url.Values{
		"cal_enabled":        {"on"},
		"cal_days_for_index": {"7"},
		"cal_showholidays":   {"all"},
		"cal_showbdays":      {"index"},
		"cal_showevents":     {"never"},
		"cal_defaultboard":   {"1"},
		"cal_minyear":        {"2000"},
		"cal_maxyear":        {"2030"},
		"cal_maxspan":        {"7"},
		"calendar_view[0]":   {"on"},
		"sc":                 {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("save cal settings: status %d body %.300s", w.Code, w.Body.String())
	}
	if a.Setting("cal_enabled") != "1" || a.Setting("cal_days_for_index") != "7" {
		t.Fatalf("cal settings not saved")
	}
	// showholidays=all -> both index + calendar flags on.
	if a.Setting("cal_showholidaysonindex") != "1" || a.Setting("cal_showholidaysoncalendar") != "1" {
		t.Fatalf("cal_showholidays=all not mapped to both flags")
	}
	// showbdays=index -> index on, calendar off.
	if a.Setting("cal_showbdaysonindex") != "1" || a.Setting("cal_showbdaysoncalendar") != "0" {
		t.Fatalf("cal_showbdays=index not mapped correctly")
	}
	// Inline calendar_view permission granted to members (group 0).
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 0 AND permission = 'calendar_view'`)).Scan(&n)
	if n != 1 {
		t.Errorf("calendar_view permission not granted to members")
	}
}

func TestHolidayAddListDelete(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1", "cal_enabled": "1", "cal_minyear": "2000", "cal_maxyear": "2030"})

	// Count existing seeded holidays.
	var before int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}calendar_holidays`)).Scan(&before)

	// Add a holiday.
	sc, cookies := mbForm(t, a, "/index.php?action=managecalendar;sa=editholiday", admin)
	w := postForm(t, a, "/index.php?action=managecalendar;sa=editholiday", url.Values{
		"title": {"AAA Test Day"},
		"year":  {"2025"},
		"month": {"7"},
		"day":   {"4"},
		"sc":    {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("add holiday: status %d", w.Code)
	}
	var id int
	var date string
	a.DB.QueryRow(a.Q(`SELECT ID_HOLIDAY, eventDate FROM {$db_prefix}calendar_holidays WHERE title = 'AAA Test Day'`)).Scan(&id, &date)
	if id == 0 || date != "2025-07-04" {
		t.Fatalf("holiday not added correctly: id=%d date=%q", id, date)
	}

	// It shows in the list.
	_, body := get(t, a, "/index.php?action=managecalendar;sa=holidays", admin)
	if !strings.Contains(body, "AAA Test Day") || !strings.Contains(body, "4 July 2025") {
		t.Errorf("holiday not listed:\n%.400s", body)
	}

	// Delete it via the list.
	sc, cookies = mbForm(t, a, "/index.php?action=managecalendar;sa=holidays", admin)
	w = postForm(t, a, "/index.php?action=managecalendar;sa=holidays", url.Values{
		"delete":                    {"1"},
		"holiday[" + itoa(id) + "]": {"on"},
		"sc":                        {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("delete holiday: status %d", w.Code)
	}
	var after int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}calendar_holidays`)).Scan(&after)
	if after != before {
		t.Fatalf("holiday not deleted (before=%d after=%d)", before, after)
	}
}
