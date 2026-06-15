package app

// Phase 7: attachment + avatar administration.

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestAttachmentSettingsSave(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=manageattachments", admin)
	w := postForm(t, a, "/index.php?action=manageattachments", url.Values{
		"attachmentSettings":   {"1"},
		"attachmentEnable":     {"1"},
		"attachmentExtensions": {"jpg,png,gif"},
		"attachmentUploadDir":  {"/tmp/attach"},
		"attachmentPostLimit":  {"192"},
		"attachmentThumbnails": {"on"},
		"attachmentThumbWidth": {"100"},
		"sc":                   {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("save attachment settings: status %d", w.Code)
	}
	if a.Setting("attachmentExtensions") != "jpg,png,gif" || a.Setting("attachmentThumbnails") != "1" || a.Setting("attachmentPostLimit") != "192" {
		t.Fatalf("attachment settings not saved")
	}
}

func TestAvatarSettingsSaveInlinePerm(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	sc, cookies := mbForm(t, a, "/index.php?action=manageattachments;sa=avatars", admin)
	w := postForm(t, a, "/index.php?action=manageattachments;sa=avatars", url.Values{
		"avatarSettings":           {"1"},
		"avatar_directory":         {"/tmp/avatars"},
		"avatar_url":               {"http://example.com/avatars"},
		"avatar_max_width_upload":  {"65"},
		"avatar_max_height_upload": {"65"},
		"profile_server_avatar[2]": {"on"},
		"sc":                       {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("save avatar settings: status %d", w.Code)
	}
	if a.Setting("avatar_url") != "http://example.com/avatars" || a.Setting("avatar_max_width_upload") != "65" {
		t.Fatalf("avatar settings not saved")
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}permissions WHERE ID_GROUP = 2 AND permission = 'profile_server_avatar'`)).Scan(&n)
	if n != 1 {
		t.Errorf("profile_server_avatar inline permission not granted to group 2")
	}
}

func TestAttachmentBrowseRenders(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	_, body := get(t, a, "/index.php?action=manageattachments;sa=browse", admin)
	if !strings.Contains(body, "attachment_manager_browse_files") && !strings.Contains(body, "sa=browse;avatars") {
		t.Fatalf("browse page missing sub-tabs:\n%.400s", body)
	}
}

func TestAttachmentMaintenanceRenders(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})
	// The upload dir must exist for MaintainFiles to scan it.
	os.MkdirAll(a.Setting("attachmentUploadDir"), 0755)

	_, body := get(t, a, "/index.php?action=manageattachments;sa=maintenance", admin)
	if !strings.Contains(body, "sa=repair") {
		t.Fatalf("maintenance page missing repair link:\n%.400s", body)
	}
}

func TestAttachmentRepairOrphan(t *testing.T) {
	a := newTestApp(t)
	admin := adminCookie(t, a)
	a.UpdateSettings(map[string]string{"securityDisable": "1"})

	// An attachment pointing at a non-existent message (attachment_no_msg).
	a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}attachments (ID_MSG, ID_MEMBER, attachmentType, filename, size) VALUES (99999, 0, 0, 'orphan.dat', 10)`))
	var aid int
	a.DB.QueryRow(a.Q(`SELECT ID_ATTACH FROM {$db_prefix}attachments WHERE filename = 'orphan.dat'`)).Scan(&aid)

	sc, cookies := mbForm(t, a, "/index.php?action=manageattachments", admin)

	// First a dry-run scan finds the error.
	_, body := get(t, a, "/index.php?action=manageattachments;sa=repair;sesc="+sc, cookies...)
	if !strings.Contains(body, "attachment_no_msg") {
		t.Fatalf("repair did not report the orphan:\n%.500s", body)
	}

	// Now fix it.
	w := postForm(t, a, "/index.php?action=manageattachments;sa=repair;fixErrors=1;step=0;substep=0;sesc="+sc, url.Values{
		"to_fix[]": {"attachment_no_msg"},
		"sc":       {sc},
	}, cookies...)
	if w.Code != 200 {
		t.Fatalf("repair fix: status %d body %.300s", w.Code, w.Body.String())
	}
	var n int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}attachments WHERE ID_ATTACH = ?`), aid).Scan(&n)
	if n != 0 {
		t.Fatalf("orphan attachment not removed")
	}
}
