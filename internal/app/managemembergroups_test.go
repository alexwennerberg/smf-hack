package app

// Phase 7: membergroup administration.

import (
	"net/url"
	"strings"
	"testing"
)

func TestMembergroupIndex(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=membergroups", admin)
	if !strings.Contains(body, "Administrator") || !strings.Contains(body, "?action=membergroups;sa=edit;group=1") {
		t.Errorf("membergroup index missing groups:\n%.400s", body)
	}
	// Regular + post group sections.
	if !strings.Contains(body, `name="postgroup" value="1"`) {
		t.Errorf("membergroup index missing post-group section")
	}
}

func TestMembergroupAddAndEditAndDelete(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Add a new general group with standard permissions.
	sc, cookies := mbForm(t, a, "/index.php?action=membergroups;sa=add;generalgroup", admin)
	w := postForm(t, a, "/index.php?action=membergroups;sa=add", url.Values{
		"group_name": {"Testers"},
		"perm_type":  {"predefined"},
		"level":      {"standard"},
		"sc":         {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("add group: status %d body %.300s", w.Code, w.Body.String())
	}
	var gid int
	a.DB.QueryRow(a.Q(`SELECT ID_GROUP FROM {$db_prefix}membergroups WHERE groupName = 'Testers'`)).Scan(&gid)
	if gid == 0 {
		t.Fatalf("group not created")
	}
	// Standard level granted pm_read globally.
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = ? AND permission = 'pm_read'`), gid).Scan(&n)
	if n != 1 {
		t.Errorf("predefined standard level not applied to new group")
	}

	// Edit the group's name + color.
	sc, cookies = mbForm(t, a, "/index.php?action=membergroups;sa=edit;group="+itoa(gid), admin)
	w = postForm(t, a, "/index.php?action=membergroups;sa=edit;group="+itoa(gid), url.Values{
		"submit":       {"1"},
		"group_name":   {"QA Team"},
		"online_color": {"#ff0000"},
		"max_messages": {"0"},
		"star_count":   {"0"},
		"star_image":   {""},
		"sc":           {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("edit group: status %d body %.300s", w.Code, w.Body.String())
	}
	var name, color string
	a.DB.QueryRow(a.Q(`SELECT groupName, onlineColor FROM {$db_prefix}membergroups WHERE ID_GROUP = ?`), gid).Scan(&name, &color)
	if name != "QA Team" || color != "#ff0000" {
		t.Fatalf("group edit not saved: name=%q color=%q", name, color)
	}

	// Delete it.
	sc, cookies = mbForm(t, a, "/index.php?action=membergroups;sa=edit;group="+itoa(gid), admin)
	w = postForm(t, a, "/index.php?action=membergroups;sa=edit;group="+itoa(gid), url.Values{
		"delete": {"1"},
		"sc":     {sc},
	}, cookies...)
	if w.Code != 302 {
		t.Fatalf("delete group: status %d", w.Code)
	}
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}membergroups WHERE ID_GROUP = ?`), gid).Scan(&n)
	if n != 0 {
		t.Fatalf("group not deleted")
	}
}

func TestMembergroupMembersAddRemove(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// Add an assignable general group.
	sc, cookies := mbForm(t, a, "/index.php?action=membergroups;sa=add;generalgroup", admin)
	postForm(t, a, "/index.php?action=membergroups;sa=add", url.Values{
		"group_name": {"Helpers"}, "perm_type": {"predefined"}, "level": {"standard"}, "sc": {sc},
	}, cookies...)
	var gid int
	a.DB.QueryRow(a.Q(`SELECT ID_GROUP FROM {$db_prefix}membergroups WHERE groupName = 'Helpers'`)).Scan(&gid)

	// Create a member to assign.
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members (memberName, realName, emailAddress, ID_GROUP, ID_POST_GROUP, is_activated) VALUES ('joe', 'joe', 'joe@x.com', 0, 4, 1)`))
	var uid int
	a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE memberName = 'joe'`)).Scan(&uid)

	// Add joe to the group.
	sc, cookies = mbForm(t, a, "/index.php?action=membergroups;sa=members;group="+itoa(gid), admin)
	w := postForm(t, a, "/index.php?action=membergroups;sa=members;group="+itoa(gid), url.Values{
		"add":   {"1"},
		"toAdd": {"joe"},
		"sc":    {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("add member: status %d", w.Code)
	}
	var grp int
	a.DB.QueryRow(a.Q(`SELECT ID_GROUP FROM {$db_prefix}members WHERE ID_MEMBER = ?`), uid).Scan(&grp)
	if grp != gid {
		t.Fatalf("joe not assigned to group (ID_GROUP=%d want %d)", grp, gid)
	}

	// The member shows in the list.
	_, body := get(t, a, "/index.php?action=membergroups;sa=members;group="+itoa(gid), admin)
	if !strings.Contains(body, "joe@x.com") {
		t.Errorf("member not listed in group")
	}

	// Remove joe.
	sc, cookies = mbForm(t, a, "/index.php?action=membergroups;sa=members;group="+itoa(gid), admin)
	w = postForm(t, a, "/index.php?action=membergroups;sa=members;group="+itoa(gid), url.Values{
		"remove": {"1"},
		"rem[]":  {itoa(uid)},
		"sc":     {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("remove member: status %d", w.Code)
	}
	a.DB.QueryRow(a.Q(`SELECT ID_GROUP FROM {$db_prefix}members WHERE ID_MEMBER = ?`), uid).Scan(&grp)
	if grp != 0 {
		t.Fatalf("joe not removed from group (ID_GROUP=%d)", grp)
	}
}
