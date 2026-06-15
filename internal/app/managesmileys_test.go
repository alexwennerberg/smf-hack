package app

// Phase 7: smiley + message-icon administration.

import (
	"net/url"
	"strings"
	"testing"
)

func TestSmileySettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=smileys;sa=settings", admin)
	w := postForm(t, a, "/index.php?action=smileys;sa=settings", url.Values{
		"default_smiley_set":  {"1"}, // classic
		"smiley_enable":       {"on"},
		"messageIcons_enable": {"on"},
		"smiley_sets_url":     {"http://example.com/smileys"},
		"smiley_sets_dir":     {"/tmp/smileys"},
		"sc":                  {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("save smiley settings: status %d body %.200s", w.Code, w.Body.String())
	}
	if a.Setting("smiley_sets_default") != "classic" {
		t.Errorf("default set not saved: %q", a.Setting("smiley_sets_default"))
	}
	if a.Setting("smileys_url") != "http://example.com/smileys" || a.Setting("smiley_enable") != "1" {
		t.Errorf("smiley settings not saved")
	}
}

func TestSmileyAddListModifyDelete(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1", "smiley_enable": "1"})

	// Add a smiley (existing-file method).
	sc, cookies := mbForm(t, a, "/index.php?action=smileys;sa=addsmiley", admin)
	w := postForm(t, a, "/index.php?action=smileys;sa=addsmiley", url.Values{
		"method":             {"existing"},
		"smiley_code":        {":testsmiley:"},
		"smiley_filename":    {"test.gif"},
		"smiley_description": {"Test Smiley"},
		"smiley_location":    {"0"},
		"sc":                 {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("add smiley: status %d body %.200s", w.Code, w.Body.String())
	}
	var id int
	a.DB.QueryRow(a.Q(`SELECT ID_SMILEY FROM {$db_prefix}smileys WHERE code = ':testsmiley:'`)).Scan(&id)
	if id == 0 {
		t.Fatalf("smiley not inserted")
	}

	// List shows it.
	_, body := get(t, a, "/index.php?action=smileys;sa=editsmileys", admin)
	if !strings.Contains(body, ":testsmiley:") || !strings.Contains(body, "test.gif") {
		t.Fatalf("editsmileys missing smiley:\n%.400s", body)
	}

	// Modify it.
	sc, cookies = mbForm(t, a, "/index.php?action=smileys;sa=modifysmiley;smiley="+itoa(id), admin)
	w = postForm(t, a, "/index.php?action=smileys;sa=editsmileys", url.Values{
		"smiley":             {itoa(id)},
		"smiley_code":        {":testsmiley:"},
		"smiley_filename":    {"test.gif"},
		"smiley_description": {"Changed"},
		"smiley_location":    {"2"},
		"sc":                 {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("modify smiley: status %d", w.Code)
	}
	var hidden int
	var desc string
	a.DB.QueryRow(a.Q(`SELECT hidden, description FROM {$db_prefix}smileys WHERE ID_SMILEY = ?`), id).Scan(&hidden, &desc)
	if hidden != 2 || desc != "Changed" {
		t.Fatalf("smiley not modified (hidden=%d desc=%q)", hidden, desc)
	}

	// Bulk delete.
	sc, cookies = mbForm(t, a, "/index.php?action=smileys;sa=editsmileys", admin)
	w = postForm(t, a, "/index.php?action=smileys;sa=editsmileys", url.Values{
		"smiley_action":     {"delete"},
		"checked_smileys[]": {itoa(id)},
		"sc":                {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("delete smiley: status %d", w.Code)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}smileys WHERE ID_SMILEY = ?`), id).Scan(&n)
	if n != 0 {
		t.Fatalf("smiley not deleted")
	}
}

func TestMessageIconsListAddDelete(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1", "messageIcons_enable": "1"})

	// The seeded icons are listed.
	_, body := get(t, a, "/index.php?action=smileys;sa=editicons", admin)
	if !strings.Contains(body, `name="checked_icons[]"`) {
		t.Fatalf("editicons list missing checkboxes:\n%.400s", body)
	}

	var before int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}message_icons`)).Scan(&before)

	// Delete the first icon.
	var firstID int
	a.DB.QueryRow(a.Q(`SELECT ID_ICON FROM {$db_prefix}message_icons ORDER BY iconOrder LIMIT 1`)).Scan(&firstID)
	sc, cookies := mbForm(t, a, "/index.php?action=smileys;sa=editicons", admin)
	w := postForm(t, a, "/index.php?action=smileys;sa=editicons", url.Values{
		"delete":          {"1"},
		"checked_icons[]": {itoa(firstID)},
		"sc":              {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("delete icon: status %d", w.Code)
	}
	var after int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}message_icons`)).Scan(&after)
	if after != before-1 {
		t.Fatalf("icon not deleted (before=%d after=%d)", before, after)
	}
}

func TestSmileySetsListRenders(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=smileys;sa=editsets", admin)
	if !strings.Contains(body, "Default") || !strings.Contains(body, "Classic") {
		t.Fatalf("editsets missing set names:\n%.400s", body)
	}
	if !strings.Contains(body, "sa=modifyset;set=") {
		t.Errorf("editsets missing modify links")
	}

	// The modify form for a new set renders.
	_, body = get(t, a, "/index.php?action=smileys;sa=modifyset;set=-1", admin)
	if !strings.Contains(body, `name="smiley_sets_name"`) {
		t.Errorf("modifyset form missing fields")
	}
}
