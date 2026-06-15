package app

// Phase 7: ban administration (list/add/edit/triggers/log).

import (
	"net/url"
	"strings"
	"testing"
)

func TestBanAddListAndEdit(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Add a new ban group with an IP trigger.
	sc, cookies := mbForm(t, a, "/index.php?action=ban;sa=add", admin)
	w := postForm(t, a, "/index.php?action=ban;sa=edit", url.Values{
		"add_ban":          {"Add ban"},
		"ban_name":         {"Spammer"},
		"expiration":       {"never"},
		"reason":           {"spam"},
		"notes":            {"a note"},
		"full_ban":         {"1"},
		"ban_suggestion[]": {"main_ip"},
		"main_ip":          {"10.20.30.40"},
		"old_expire":       {"0"},
		"bg":               {"0"},
		"sc":               {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("add ban: status %d body %.300s", w.Code, w.Body.String())
	}

	var bg int
	a.DB.QueryRow(a.Q(`SELECT ID_BAN_GROUP FROM {$db_prefix}ban_groups WHERE name = 'Spammer'`)).Scan(&bg)
	if bg == 0 {
		t.Fatalf("ban group not created")
	}
	var lo1, hi1 int
	a.DB.QueryRow(a.Q(`SELECT ip_low1, ip_high1 FROM {$db_prefix}ban_items WHERE ID_BAN_GROUP = ?`), bg).Scan(&lo1, &hi1)
	if lo1 != 10 || hi1 != 10 {
		t.Fatalf("ip trigger not stored: low1=%d high1=%d", lo1, hi1)
	}

	// The list shows it.
	_, body := get(t, a, "/index.php?action=ban;sa=list", admin)
	if !strings.Contains(body, "Spammer") || !strings.Contains(body, "?action=ban;sa=edit") {
		t.Fatalf("ban list missing entry:\n%.400s", body)
	}

	// The edit page shows the trigger as an IP range.
	_, body = get(t, a, "/index.php?action=ban;sa=edit;bg="+itoa(bg), admin)
	if !strings.Contains(body, "10.20.30.40") {
		t.Errorf("edit page missing ip trigger:\n%.400s", body)
	}
}

func TestBanEmailMarksMemberBanned(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members (memberName, realName, emailAddress, ID_GROUP, ID_POST_GROUP, is_activated) VALUES ('badguy', 'Bad Guy', 'bad@evil.com', 0, 4, 1)`))
	var uid int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'badguy'`)).Scan(&uid)

	// Add a full-access email ban matching the member.
	sc, cookies := mbForm(t, a, "/index.php?action=ban;sa=add", admin)
	w := postForm(t, a, "/index.php?action=ban;sa=edit", url.Values{
		"add_ban":          {"Add ban"},
		"ban_name":         {"EvilDomain"},
		"expiration":       {"never"},
		"full_ban":         {"1"},
		"ban_suggestion[]": {"email"},
		"email":            {"bad@evil.com"},
		"old_expire":       {"0"},
		"bg":               {"0"},
		"sc":               {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("add email ban: status %d", w.Code)
	}

	var activated int
	a.DB.QueryRow(a.Q(`SELECT is_activated FROM {$db_prefix}members WHERE ID_MEMBER = ?`), uid).Scan(&activated)
	if activated < 10 {
		t.Fatalf("member not marked banned (is_activated=%d)", activated)
	}

	// Find the group + item, then remove it via the list -> member is rehabilitated.
	var bg int
	a.DB.QueryRow(a.Q(`SELECT ID_BAN_GROUP FROM {$db_prefix}ban_groups WHERE name = 'EvilDomain'`)).Scan(&bg)
	sc, cookies = mbForm(t, a, "/index.php?action=ban;sa=list", admin)
	w = postForm(t, a, "/index.php?action=ban;sa=list", url.Values{
		"removeBans": {"1"},
		"remove[]":   {itoa(bg)},
		"sc":         {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("remove ban: status %d", w.Code)
	}
	a.DB.QueryRow(a.Q(`SELECT is_activated FROM {$db_prefix}members WHERE ID_MEMBER = ?`), uid).Scan(&activated)
	if activated >= 10 {
		t.Fatalf("member still banned after removal (is_activated=%d)", activated)
	}
}

func TestBanLogRendersAndClears(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}log_banned (ID_MEMBER, ip, email, logTime) VALUES (0, '1.2.3.4', 'x@y.com', ?)`), nowUnix())

	_, body := get(t, a, "/index.php?action=ban;sa=log", admin)
	if !strings.Contains(body, "1.2.3.4") || !strings.Contains(body, "x@y.com") {
		t.Fatalf("ban log missing entry:\n%.400s", body)
	}

	sc, cookies := mbForm(t, a, "/index.php?action=ban;sa=log", admin)
	w := postForm(t, a, "/index.php?action=ban;sa=log", url.Values{
		"removeAll": {"1"},
		"sc":        {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("clear ban log: status %d", w.Code)
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_banned`)).Scan(&n)
	if n != 0 {
		t.Fatalf("ban log not cleared")
	}
}

func TestBanBrowseTriggers(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Seed a group + an IP trigger directly.
	res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}ban_groups (name, ban_time, cannot_access) VALUES ('G1', ?, 1)`), nowUnix())
	gid, _ := res.LastInsertId()
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}ban_items (ID_BAN_GROUP, ip_low1, ip_high1, ip_low2, ip_high2, ip_low3, ip_high3, ip_low4, ip_high4) VALUES (?, 192, 192, 168, 168, 0, 0, 1, 1)`), gid)

	_, body := get(t, a, "/index.php?action=ban;sa=browse;entity=ip", admin)
	if !strings.Contains(body, "192.168.0.1") {
		t.Fatalf("browse triggers missing ip:\n%.400s", body)
	}
	if !strings.Contains(body, "G1") {
		t.Errorf("browse triggers missing group link")
	}
}

func TestIP2RangeRoundTrip(t *testing.T) {
	cases := map[string][4][2]int{
		"10.20.30.40": {{10, 10}, {20, 20}, {30, 30}, {40, 40}},
		"192.168.*.*": {{192, 192}, {168, 168}, {0, 255}, {0, 255}},
		"10.0-15.0.1": {{10, 10}, {0, 15}, {0, 0}, {1, 1}},
	}
	for in, want := range cases {
		got := ip2range(in)
		if len(got) != 4 {
			t.Fatalf("ip2range(%q) len=%d", in, len(got))
		}
		var lo, hi []int
		for i := 0; i < 4; i++ {
			if got[i] != want[i] {
				t.Errorf("ip2range(%q)[%d] = %v, want %v", in, i, got[i], want[i])
			}
			lo = append(lo, got[i][0])
			hi = append(hi, got[i][1])
		}
		if back := range2ip(lo, hi); back != in {
			t.Errorf("range2ip round trip: got %q, want %q", back, in)
		}
	}
}
