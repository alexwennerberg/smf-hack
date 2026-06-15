package app

// Phase 7: theme settings/options subset (SetThemeSettings / SetThemeOptions).

import (
	"net/url"
	"strings"
	"testing"
)

// TestThemeSettingsSave renders the default theme's layout-settings form and
// round-trips a couple of values into the themes table (ID_MEMBER = 0).
func TestThemeSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=maintain", admin)

	// The settings form renders for the default theme.
	_, body := get(t, a, "/index.php?action=theme;sesc="+sc+";sa=settings;th=1", cookies...)
	if !strings.Contains(body, `name="options[number_recent_posts]"`) || !strings.Contains(body, `name="options[show_gender]"`) {
		t.Fatalf("theme settings form missing fields:\n%.500s", body)
	}
	// The list tab should be selected (SetThemeSettings forces it).
	if !strings.Contains(body, `sa=settings;th=1`) {
		t.Fatalf("settings form action wrong:\n%.300s", body)
	}

	// Save: a number setting and a checkbox.
	w := postForm(t, a, "/index.php?action=theme;sa=settings;th=1", url.Values{
		"submit":                       {"Save"},
		"options[number_recent_posts]": {"7"},
		"options[show_gender]":         {"1"},
		"sc":                           {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("save theme settings: status %d body %.200s", w.Code, w.Body.String())
	}

	var v string
	a.DB.QueryRow(a.Q(`SELECT value FROM {$db_prefix}themes WHERE ID_MEMBER = 0 AND ID_THEME = 1 AND variable = 'number_recent_posts'`)).Scan(&v)
	if v != "7" {
		t.Errorf("number_recent_posts not saved, got %q", v)
	}
	v = ""
	a.DB.QueryRow(a.Q(`SELECT value FROM {$db_prefix}themes WHERE ID_MEMBER = 0 AND ID_THEME = 1 AND variable = 'show_gender'`)).Scan(&v)
	if v != "1" {
		t.Errorf("show_gender not saved, got %q", v)
	}

	// Re-render: the saved value is reflected in the form.
	_, body = get(t, a, "/index.php?action=theme;sesc="+sc+";sa=settings;th=1", cookies...)
	if !strings.Contains(body, `name="options[number_recent_posts]" id="number_recent_posts" value="7"`) {
		t.Errorf("saved number_recent_posts not shown in form")
	}
}

// TestThemeOptionsResetList renders the reset list and the set-defaults form,
// and round-trips a default option.
func TestThemeOptionsResetList(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=maintain", admin)

	// The reset list shows the default theme with reset links.
	_, body := get(t, a, "/index.php?action=theme;sesc="+sc+";sa=reset", cookies...)
	if !strings.Contains(body, `sa=reset;who=1`) || !strings.Contains(body, `sa=reset;who=2`) {
		t.Fatalf("reset list missing reset links:\n%.500s", body)
	}

	// The set-defaults form (no who) renders option fields with default_ prefix.
	_, body = get(t, a, "/index.php?action=theme;sesc="+sc+";sa=reset;th=1", cookies...)
	if !strings.Contains(body, `name="default_options[show_board_desc]"`) {
		t.Fatalf("set-options form missing default option field:\n%.500s", body)
	}

	// Save a default option (ID_MEMBER = -1, ID_THEME = 1).
	w := postForm(t, a, "/index.php?action=theme;sa=reset;th=1", url.Values{
		"submit":                              {"Save"},
		"who":                                 {"0"},
		"default_options[show_board_desc]":    {"1"},
		"default_options[calendar_start_day]": {"1"},
		"sc":                                  {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("save theme options: status %d body %.200s", w.Code, w.Body.String())
	}

	var v string
	a.DB.QueryRow(a.Q(`SELECT value FROM {$db_prefix}themes WHERE ID_MEMBER = -1 AND ID_THEME = 1 AND variable = 'calendar_start_day'`)).Scan(&v)
	if v != "1" {
		t.Errorf("calendar_start_day default not saved, got %q", v)
	}
}

// TestThemeOptionsResetMembers exercises the who=1 (apply to all members) path.
func TestThemeOptionsResetMembers(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=maintain", admin)

	// The reset-members form (who=1) renders the master selects.
	_, body := get(t, a, "/index.php?action=theme;sesc="+sc+";sa=reset;th=1;who=1", cookies...)
	if !strings.Contains(body, `options_master[show_board_desc]`) && !strings.Contains(body, `default_options_master[show_board_desc]`) {
		t.Fatalf("reset-members form missing master select:\n%.500s", body)
	}

	// Apply a value to all members (master=1 means "change").
	w := postForm(t, a, "/index.php?action=theme;sa=reset;th=1", url.Values{
		"submit":                             {"Save"},
		"who":                                {"1"},
		"default_options[view_newest_first]": {"1"},
		"default_options_master[view_newest_first]": {"1"},
		"sc": {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("apply to members: status %d body %.200s", w.Code, w.Body.String())
	}

	// Every member now has a view_newest_first row under theme 1.
	var members, withOpt int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members`)).Scan(&members)
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}themes WHERE ID_MEMBER > 0 AND ID_THEME = 1 AND variable = 'view_newest_first'`)).Scan(&withOpt)
	if members == 0 || withOpt != members {
		t.Errorf("view_newest_first not applied to all members (members=%d withOpt=%d)", members, withOpt)
	}
}
