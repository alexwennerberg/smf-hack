package app

// Port of Sources/ManageAttachments.php: the attachment/avatar admin
// (?action=manageattachments) — attachment + avatar settings (avatar settings
// carry the inline profile_*_avatar permissions), file browser, maintenance
// (size report + bulk removal by age/size), MoveAvatars, and RepairAttachments.
//
// Deviations:
//   - GD is not available, so $context['gd_installed'] is always false (the GD
//     warnings show, matching a no-GD PHP host).
//   - RepairAttachments is collapsed to a single pass (Go has no execution
//     timeout) instead of the PHP step/substep pause-and-continue loop.

import (
	"os"
	"regexp"
)

func init() {
	registerAction("manageattachments", (*Ctx).ManageAttachments)
	layerFuncs["manage_files_above"] = templateManageFilesAbove
	layerFuncs["manage_files_below"] = templateManageFilesBelow
}

func (c *Ctx) ManageAttachments() {
	c.isAllowedTo("manage_attachments")
	c.adminIndex("manage_attachments")
	c.TemplateLayers = append(c.TemplateLayers, "manage_files")

	sa := c.REQUEST.Str("sa")
	switch sa {
	case "avatars":
		c.ManageAvatarSettings()
	case "browse":
		c.BrowseFiles()
	case "byAge":
		c.RemoveAttachmentByAge()
	case "bySize":
		c.RemoveAttachmentBySize()
	case "maintenance":
		c.MaintainFiles()
	case "moveAvatars":
		c.MoveAvatars()
	case "repair":
		c.RepairAttachments()
	case "remove":
		c.RemoveAttachment()
	case "removeall":
		c.RemoveAllAttachments()
	default:
		c.ManageAttachmentSettings()
	}
}

// ---- settings ----

type attachSettingsPage struct {
	ValidUploadDir bool
}

func (c *Ctx) ManageAttachmentSettings() {
	a := c.App
	c.PageTitle = c.Txt("smf201")
	c.attachSelected = "attachment_settings"

	if c.POST.Has("attachmentSettings") {
		c.checkSession("post", "", true)
		a.UpdateSettings(map[string]string{
			"attachmentEnable":          itoa(c.POST.Int("attachmentEnable")),
			"attachmentCheckExtensions": boolSetting(c.POST.Has("attachmentCheckExtensions")),
			"attachmentExtensions":      c.POST.Str("attachmentExtensions"),
			"attachmentShowImages":      boolSetting(c.POST.Has("attachmentShowImages")),
			"attachmentUploadDir":       c.POST.Str("attachmentUploadDir"),
			"attachmentDirSizeLimit":    itoa(c.POST.Int("attachmentDirSizeLimit")),
			"attachmentPostLimit":       itoa(c.POST.Int("attachmentPostLimit")),
			"attachmentSizeLimit":       itoa(c.POST.Int("attachmentSizeLimit")),
			"attachmentNumPerPostLimit": itoa(c.POST.Int("attachmentNumPerPostLimit")),
			"attachmentThumbnails":      boolSetting(c.POST.Has("attachmentThumbnails")),
			"attachmentThumbWidth":      itoa(c.POST.Int("attachmentThumbWidth")),
			"attachmentThumbHeight":     itoa(c.POST.Int("attachmentThumbHeight")),
		})
	}

	c.Page = &attachSettingsPage{ValidUploadDir: isWritableDir(a.Setting("attachmentUploadDir"))}
	c.SubTemplate = templateAttachments
}

type avatarSettingsPage struct {
	GDInstalled          bool
	ValidAvatarDir       bool
	ValidCustomAvatarDir bool
}

func (c *Ctx) ManageAvatarSettings() {
	a := c.App
	c.PageTitle = c.Txt("smf201")
	c.attachSelected = "avatar_settings"

	perms := []string{"profile_server_avatar", "profile_upload_avatar", "profile_remote_avatar"}

	if c.POST.Has("avatarSettings") {
		c.checkSession("post", "", true)
		a.UpdateSettings(map[string]string{
			"avatar_directory":         c.POST.Str("avatar_directory"),
			"avatar_url":               c.POST.Str("avatar_url"),
			"avatar_download_external": boolSetting(c.POST.Has("avatar_download_external")),
			"avatar_max_width_upload":  itoa(c.POST.Int("avatar_max_width_upload")),
			"avatar_max_height_upload": itoa(c.POST.Int("avatar_max_height_upload")),
			"avatar_resize_upload":     boolSetting(c.POST.Has("avatar_resize_upload")),
			"avatar_download_png":      boolSetting(c.POST.Has("avatar_download_png")),
			"custom_avatar_enabled":    boolSetting(c.POST.Has("custom_avatar_enabled")),
		})
		if !c.POST.Has("avatar_download_external") {
			a.UpdateSettings(map[string]string{
				"avatar_max_width_external":  itoa(c.POST.Int("avatar_max_width_external")),
				"avatar_max_height_external": itoa(c.POST.Int("avatar_max_height_external")),
				"avatar_action_too_large":    c.POST.Str("avatar_action_too_large"),
			})
		}
		if c.POST.Has("custom_avatar_enabled") {
			a.UpdateSettings(map[string]string{
				"custom_avatar_dir": c.POST.Str("custom_avatar_dir"),
				"custom_avatar_url": c.POST.Str("custom_avatar_url"),
			})
		}
		c.saveInlinePermissions(perms)
	}

	c.initInlinePermissions(perms, []int{-1})

	page := &avatarSettingsPage{
		GDInstalled:    false, // no GD in this port
		ValidAvatarDir: isDir(a.Setting("avatar_directory")),
	}
	page.ValidCustomAvatarDir = a.SettingEmpty("custom_avatar_enabled") || isWritableDir(a.Setting("custom_avatar_dir"))
	c.Page = page
	c.SubTemplate = templateAvatars
}

// ---- browse ----

// AttachPost is one entry of $context['posts'].
type AttachPost struct {
	PosterLink     string
	Time           string
	AttachmentID   int
	AttachmentLink string
	Width          int
	Height         int
	Size           string
	TopicLink      string
}

type browsePage struct {
	BrowseType    string
	SortBy        string
	SortDirection string
	PageIndex     string
	Start         int
	Posts         []AttachPost
}

func (c *Ctx) BrowseFiles() {
	a := c.App
	scripturl := a.ScriptURL
	c.PageTitle = c.Txt("smf201")
	c.attachSelected = "browse"

	browseType := "attachments"
	if c.REQUEST.Has("avatars") {
		browseType = "avatars"
	} else if c.REQUEST.Has("thumbs") {
		browseType = "thumbs"
	}

	// Counts.
	numAttachments, numThumbs := 0, 0
	rows, _ := a.DB.Query(a.Q(`SELECT attachmentType, COUNT(*) FROM {$db_prefix}attachments WHERE attachmentType IN (0, 3) AND ID_MEMBER = 0 GROUP BY attachmentType`))
	if rows != nil {
		for rows.Next() {
			var t, n int
			rows.Scan(&t, &n)
			if t == 0 {
				numAttachments = n
			} else {
				numThumbs = n
			}
		}
		rows.Close()
	}
	var numAvatars int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}attachments WHERE ID_MEMBER != 0`)).Scan(&numAvatars)

	sortMethods := map[string]string{"name": "a.filename", "size": "a.size", "member": "mem.realName"}
	if browseType == "avatars" {
		sortMethods["date"] = "mem.lastLogin"
	} else {
		sortMethods["date"] = "m.ID_MSG"
	}

	sortBy := c.GET.Str("sort")
	var descending bool
	if _, ok := sortMethods[sortBy]; !ok {
		sortBy = "date"
		descending = !empty(c.Options["view_newest_first"])
	} else {
		descending = c.GET.Has("desc")
	}
	sortDir := "up"
	if descending {
		sortDir = "down"
	}

	total := numAttachments
	switch browseType {
	case "avatars":
		total = numAvatars
	case "thumbs":
		total = numThumbs
	}

	perPage := a.SettingInt("defaultMaxMessages")
	if perPage == 0 {
		perPage = 30
	}
	descPart := ""
	if descending {
		descPart = ";desc"
	}
	typePart := ""
	if browseType != "attachments" {
		typePart = ";" + browseType
	}
	start := c.REQUEST.Int("start")
	pageIndex, start := c.constructPageIndex(scripturl+"?action=manageattachments;sa="+c.REQUEST.Str("sa")+typePart+";sort="+sortBy+descPart, start, total, perPage, false)

	page := &browsePage{BrowseType: browseType, SortBy: sortBy, SortDirection: sortDir, PageIndex: pageIndex, Start: start}
	c.Page = page

	dir := "ASC"
	if descending {
		dir = "DESC"
	}
	var q string
	if browseType == "avatars" {
		q = `
			SELECT '' AS ID_MSG, IFNULL(mem.realName, ?) AS posterName, mem.lastLogin AS posterTime, 0 AS ID_TOPIC, a.ID_MEMBER,
				a.ID_ATTACH, a.filename, a.attachmentType, a.size, a.width, a.height, a.downloads, '' AS subject, 0 AS ID_BOARD
			FROM {$db_prefix}attachments AS a
				LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = a.ID_MEMBER)
			WHERE a.ID_MEMBER != 0
			ORDER BY ` + sortMethods[sortBy] + ` ` + dir + `
			LIMIT ? OFFSET ?`
		rows, _ = a.DB.Query(a.Q(q), c.Txt("470"), perPage, start)
	} else {
		attachType := "0"
		if browseType == "thumbs" {
			attachType = "3"
		}
		q = `
			SELECT m.ID_MSG, IFNULL(mem.realName, m.posterName) AS posterName, m.posterTime, m.ID_TOPIC, m.ID_MEMBER,
				a.ID_ATTACH, a.filename, a.attachmentType, a.size, a.width, a.height, a.downloads, mf.subject, t.ID_BOARD
			FROM {$db_prefix}attachments AS a, {$db_prefix}messages AS m, {$db_prefix}topics AS t, {$db_prefix}messages AS mf
				LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
			WHERE a.ID_MSG = m.ID_MSG
				AND a.attachmentType = ` + attachType + `
				AND t.ID_TOPIC = m.ID_TOPIC
				AND mf.ID_MSG = t.ID_FIRST_MSG
			ORDER BY ` + sortMethods[sortBy] + ` ` + dir + `
			LIMIT ? OFFSET ?`
		rows, _ = a.DB.Query(a.Q(q), perPage, start)
	}
	if rows != nil {
		for rows.Next() {
			var idMsg, idTopic, idMember, idAttach, attachType, size, width, height, downloads, idBoard int
			var posterName, filename, subject string
			var posterTime int64
			rows.Scan(&idMsg, &posterName, &posterTime, &idTopic, &idMember, &idAttach, &filename, &attachType, &size, &width, &height, &downloads, &subject, &idBoard)

			posterLink := posterName
			if idMember != 0 {
				posterLink = `<a href="` + scripturl + `?action=profile;u=` + itoa(idMember) + `">` + posterName + `</a>`
			}
			tm := c.Txt("never")
			if posterTime != 0 {
				tm = c.timeformat(posterTime)
			}
			var href string
			if attachType == 1 {
				href = a.Setting("custom_avatar_url") + "/" + filename
			} else if browseType == "avatars" {
				href = scripturl + "?action=dlattach;type=avatar;id=" + itoa(idAttach)
			} else {
				href = scripturl + "?action=dlattach;topic=" + itoa(idTopic) + ".0;id=" + itoa(idAttach)
			}
			onclick := ""
			if width != 0 && height != 0 {
				imgPart := ";image"
				if attachType == 1 {
					imgPart = ""
				}
				onclick = ` onclick="return reqWin(this.href + '` + imgPart + `', ` + itoa(width+20) + `, ` + itoa(height+20) + `, true);"`
			}
			link := `<a href="` + href + `"` + onclick + `>` + Htmlspecialchars(filename) + `</a>`

			page.Posts = append(page.Posts, AttachPost{
				PosterLink:     posterLink,
				Time:           tm,
				AttachmentID:   idAttach,
				AttachmentLink: link,
				Width:          width,
				Height:         height,
				Size:           phpRound(float64(size)/1024, 2),
				TopicLink:      `<a href="` + scripturl + `?topic=` + itoa(idTopic) + `.0">` + subject + `</a>`,
			})
		}
		rows.Close()
	}

	c.SubTemplate = templateBrowse
}

// ---- maintenance ----

type maintenancePage struct {
	NumAttachments      int
	NumAvatars          int
	AttachmentTotalSize string
	AttachmentSpace     string
	HasSpaceLimit       bool
}

var postTmpRe = regexp.MustCompile(`^post_tmp_\d+_\d+$`)

func (c *Ctx) MaintainFiles() {
	a := c.App
	c.PageTitle = c.Txt("smf201")
	c.attachSelected = "maintenance"

	page := &maintenancePage{}
	c.Page = page
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}attachments WHERE attachmentType = 0 AND ID_MEMBER = 0`)).Scan(&page.NumAttachments)
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}attachments WHERE ID_MEMBER != 0`)).Scan(&page.NumAvatars)

	uploadDir := a.Setting("attachmentUploadDir")
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		c.fatalLangError("smf115b", true)
	}
	var dirSize float64
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." {
			continue
		}
		if postTmpRe.MatchString(name) {
			if info, err := e.Info(); err == nil && info.ModTime().Unix() < nowUnix()-18000 {
				os.Remove(uploadDir + "/" + name)
			}
			continue
		}
		if info, err := e.Info(); err == nil {
			dirSize += float64(info.Size())
		}
	}
	dirSize /= 1024

	if !a.SettingEmpty("attachmentDirSizeLimit") {
		page.HasSpaceLimit = true
		space := float64(a.SettingInt("attachmentDirSizeLimit")) - dirSize
		if space < 0 {
			space = 0
		}
		page.AttachmentSpace = phpRound(space, 2)
	}
	page.AttachmentTotalSize = phpRound(dirSize, 2)

	c.SubTemplate = templateMaintenance
}

// ---- removal ----

func (c *Ctx) RemoveAttachmentByAge() {
	a := c.App
	c.checkSession("post", "manageattachments", true)
	age := int64(c.POST.Int("age"))

	if c.REQUEST.Str("type") != "avatars" {
		messages := c.removeAttachments("a.attachmentType = 0 AND m.posterTime < "+itoa64(nowUnix()-24*60*60*age), "messages", true)
		if len(messages) > 0 && c.POST.Has("notice") {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET body = body || ? WHERE ID_MSG IN (`+joinInts(messages)+`)`), "<br /><br />"+c.POST.Str("notice"))
		}
	} else {
		c.removeAttachments("a.ID_MEMBER != 0 AND mem.lastLogin < "+itoa64(nowUnix()-24*60*60*age), "members", false)
	}
	tail := ""
	if c.REQUEST.Has("avatars") {
		tail = ";avatars"
	}
	c.redirectExit("action=manageattachments" + tail)
}

func (c *Ctx) RemoveAttachmentBySize() {
	a := c.App
	c.checkSession("post", "manageattachments", true)
	size := int64(c.POST.Int("size"))
	messages := c.removeAttachments("a.attachmentType = 0 AND a.size > "+itoa64(1024*size), "messages", true)
	if len(messages) > 0 && c.POST.Has("notice") {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET body = body || ? WHERE ID_MSG IN (`+joinInts(messages)+`)`), "<br /><br />"+c.POST.Str("notice"))
	}
	c.redirectExit("action=manageattachments;sa=maintenance")
}

func (c *Ctx) RemoveAttachment() {
	a := c.App
	c.checkSession("post", "", true)

	if c.POST.Arr("remove") != nil {
		var attachments []int
		c.POST.Arr("remove").Values(func(k string, v any) {
			attachments = append(attachments, atoi(k))
		})
		if len(attachments) > 0 {
			if c.REQUEST.Str("type") == "avatars" {
				c.removeAttachments("a.ID_ATTACH IN ("+joinInts(attachments)+")", "", false)
			} else {
				messages := c.removeAttachments("a.ID_ATTACH IN ("+joinInts(attachments)+")", "messages", true)
				if len(messages) > 0 {
					a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET body = body || ? WHERE ID_MSG IN (`+joinInts(messages)+`)`), "<br /><br />"+c.Txt("smf216"))
				}
			}
		}
	}

	sort := c.GET.Str("sort")
	if sort == "" {
		sort = "date"
	}
	descPart := ""
	if c.GET.Has("desc") {
		descPart = ";desc"
	}
	c.redirectExit("action=manageattachments;sa=browse;" + c.REQUEST.Str("type") + ";sort=" + sort + descPart + ";start=" + itoa(c.REQUEST.Int("start")))
}

func (c *Ctx) RemoveAllAttachments() {
	a := c.App
	c.checkSession("get", "manageattachments", true)
	messages := c.removeAttachments("a.attachmentType = 0", "", true)
	notice := c.POST.Str("notice")
	if !c.POST.Has("notice") {
		notice = c.Txt("smf216")
	}
	if len(messages) > 0 {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET body = body || ? WHERE ID_MSG IN (`+joinInts(messages)+`)`), "<br /><br />"+notice)
	}
	c.redirectExit("action=manageattachments;sa=maintenance")
}

// removeAttachments is removeAttachments($condition, $query_type, $return_affected).
func (c *Ctx) removeAttachments(condition, queryType string, returnAffected bool) []int {
	a := c.App
	var msgs, attach, parents []int

	from := "{$db_prefix}attachments AS a"
	extraWhere := ""
	msgCol := "a.ID_MSG"
	if queryType == "members" {
		from += ", {$db_prefix}members AS mem"
		extraWhere = " AND mem.ID_MEMBER = a.ID_MEMBER"
	} else if queryType == "messages" {
		from += ", {$db_prefix}messages AS m"
		extraWhere = " AND m.ID_MSG = a.ID_MSG"
		msgCol = "m.ID_MSG"
	}

	q := `
		SELECT a.filename, a.file_hash, a.attachmentType, a.ID_ATTACH, a.ID_MEMBER, ` + msgCol + ` AS ID_MSG,
			IFNULL(thumb.ID_ATTACH, 0) AS ID_THUMB, IFNULL(thumb.filename, '') AS thumb_filename,
			IFNULL(thumb_parent.ID_ATTACH, 0) AS ID_PARENT, IFNULL(thumb.file_hash, '') AS thumb_file_hash
		FROM (` + from + `)
			LEFT JOIN {$db_prefix}attachments AS thumb ON (thumb.ID_ATTACH = a.ID_THUMB)
			LEFT JOIN {$db_prefix}attachments AS thumb_parent ON (a.attachmentType = 3 AND thumb_parent.ID_THUMB = a.ID_ATTACH)
		WHERE ` + condition + extraWhere

	rows, err := a.DB.Query(a.Q(q))
	if err != nil {
		return nil
	}
	for rows.Next() {
		var filename, fileHash, thumbFilename, thumbFileHash string
		var attachType, idAttach, idMember, idMsg, idThumb, idParent int
		rows.Scan(&filename, &fileHash, &attachType, &idAttach, &idMember, &idMsg, &idThumb, &thumbFilename, &idParent, &thumbFileHash)

		if attachType == 1 {
			os.Remove(a.Setting("custom_avatar_dir") + "/" + filename)
		} else {
			os.Remove(a.getAttachmentFilename(filename, idAttach, false, fileHash))
			if idParent != 0 {
				parents = append(parents, idParent)
			}
			if idThumb != 0 {
				os.Remove(a.getAttachmentFilename(thumbFilename, idThumb, false, thumbFileHash))
				attach = append(attach, idThumb)
			}
		}
		if returnAffected && attachType == 0 {
			msgs = append(msgs, idMsg)
		}
		attach = append(attach, idAttach)
	}
	rows.Close()

	// Parents whose thumb we deleted (and that aren't themselves deleted).
	parents = diffInts(parents, attach)
	if len(parents) > 0 {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}attachments SET ID_THUMB = 0 WHERE ID_ATTACH IN (` + joinInts(parents) + `)`))
	}
	if len(attach) > 0 {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}attachments WHERE ID_ATTACH IN (` + joinInts(attach) + `)`))
	}
	if returnAffected {
		return uniqueInts(msgs)
	}
	return nil
}

// ---- MoveAvatars ----

func (c *Ctx) MoveAvatars() {
	a := c.App
	customDir := a.Setting("custom_avatar_dir")
	if !isWritableDir(customDir) {
		os.Chmod(customDir, 0777)
		if !isWritableDir(customDir) {
			c.fatalLangError("attachments_no_write", true)
		}
	}

	rows, err := a.DB.Query(a.Q(`SELECT ID_ATTACH, ID_MEMBER, filename, file_hash FROM {$db_prefix}attachments WHERE attachmentType = 0 AND ID_MEMBER > 0`))
	var updated []int
	if err == nil {
		for rows.Next() {
			var idAttach, idMember int
			var filename, fileHash string
			rows.Scan(&idAttach, &idMember, &filename, &fileHash)
			src := a.getAttachmentFilename(filename, idAttach, false, fileHash)
			if os.Rename(src, customDir+"/"+filename) == nil {
				updated = append(updated, idAttach)
			}
		}
		rows.Close()
	}
	if len(updated) > 0 {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}attachments SET attachmentType = 1 WHERE ID_ATTACH IN (` + joinInts(updated) + `)`))
	}
	c.redirectExit("action=manageattachments;sa=maintenance")
}

// ---- RepairAttachments (single pass) ----

// RepairError is one row of $context['repair_errors'].
type RepairError struct {
	Key   string
	Count int
}

type repairPage struct {
	Completed    bool
	ErrorsFound  bool
	RepairErrors []RepairError
}

func (c *Ctx) RepairAttachments() {
	a := c.App
	c.PageTitle = c.Txt("repair_attachments")
	c.attachSelected = "maintenance"
	c.checkSession("get", "", true)

	if c.POST.Has("cancel") {
		c.redirectExit("action=manageattachments;sa=maintenance")
		return
	}

	fixErrors := c.GET.Has("fixErrors")
	toFix := map[string]bool{}
	if fixErrors {
		if c.POST.Arr("to_fix") == nil {
			c.redirectExit("action=manageattachments;sa=maintenance")
			return
		}
		c.POST.Arr("to_fix").Values(func(k string, v any) {
			if s, ok := v.(string); ok {
				toFix[s] = true
			}
		})
	}

	errs := map[string]int{}

	// 1. Stranded thumbnails (attachmentType 3 with no parent).
	{
		var toRemove []int
		rows, _ := a.DB.Query(a.Q(`
			SELECT thumb.ID_ATTACH, thumb.filename, thumb.file_hash
			FROM {$db_prefix}attachments AS thumb
				LEFT JOIN {$db_prefix}attachments AS tparent ON (tparent.ID_THUMB = thumb.ID_ATTACH)
			WHERE thumb.attachmentType = 3 AND tparent.ID_ATTACH IS NULL`))
		if rows != nil {
			for rows.Next() {
				var id int
				var fn, fh string
				rows.Scan(&id, &fn, &fh)
				toRemove = append(toRemove, id)
				errs["missing_thumbnail_parent"]++
				if fixErrors && toFix["missing_thumbnail_parent"] {
					os.Remove(a.getAttachmentFilename(fn, id, false, fh))
				}
			}
			rows.Close()
		}
		if fixErrors && len(toRemove) > 0 && toFix["missing_thumbnail_parent"] {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}attachments WHERE ID_ATTACH IN (` + joinInts(toRemove) + `) AND attachmentType = 3`))
		}
	}

	// 2. Parents that think they have thumbnails, but don't.
	{
		var toUpdate []int
		rows, _ := a.DB.Query(a.Q(`
			SELECT a.ID_ATTACH
			FROM {$db_prefix}attachments AS a
				LEFT JOIN {$db_prefix}attachments AS thumb ON (thumb.ID_ATTACH = a.ID_THUMB)
			WHERE a.ID_THUMB != 0 AND thumb.ID_ATTACH IS NULL`))
		if rows != nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				toUpdate = append(toUpdate, id)
				errs["parent_missing_thumbnail"]++
			}
			rows.Close()
		}
		if fixErrors && len(toUpdate) > 0 && toFix["parent_missing_thumbnail"] {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}attachments SET ID_THUMB = 0 WHERE ID_ATTACH IN (` + joinInts(toUpdate) + `)`))
		}
	}

	// 3. Recount/verify every attachment file.
	{
		var toRemove []int
		rows, _ := a.DB.Query(a.Q(`SELECT ID_ATTACH, filename, file_hash, size, attachmentType FROM {$db_prefix}attachments`))
		if rows != nil {
			for rows.Next() {
				var id, size, attachType int
				var fn, fh string
				rows.Scan(&id, &fn, &fh, &size, &attachType)
				var filename string
				if attachType == 1 {
					filename = a.Setting("custom_avatar_dir") + "/" + fn
				} else {
					filename = a.getAttachmentFilename(fn, id, false, fh)
				}
				info, statErr := os.Stat(filename)
				if statErr != nil {
					errs["file_missing_on_disk"]++
					if fixErrors && toFix["file_missing_on_disk"] {
						toRemove = append(toRemove, id)
					}
				} else if info.Size() == 0 {
					errs["file_size_of_zero"]++
					if fixErrors && toFix["file_size_of_zero"] {
						toRemove = append(toRemove, id)
						os.Remove(filename)
					}
				} else if info.Size() != int64(size) {
					errs["file_wrong_size"]++
					if fixErrors && toFix["file_wrong_size"] {
						a.DB.Exec(a.Q(`UPDATE {$db_prefix}attachments SET size = ? WHERE ID_ATTACH = ?`), info.Size(), id)
					}
				}
			}
			rows.Close()
		}
		if fixErrors && len(toRemove) > 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}attachments WHERE ID_ATTACH IN (` + joinInts(toRemove) + `)`))
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}attachments SET ID_THUMB = 0 WHERE ID_THUMB IN (` + joinInts(toRemove) + `)`))
		}
	}

	// 4. Avatars with no member.
	{
		var toRemove []int
		rows, _ := a.DB.Query(a.Q(`
			SELECT a.ID_ATTACH, a.filename, a.file_hash, a.attachmentType
			FROM {$db_prefix}attachments AS a
				LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = a.ID_MEMBER)
			WHERE a.ID_MEMBER != 0 AND a.ID_MSG = 0 AND mem.ID_MEMBER IS NULL`))
		if rows != nil {
			for rows.Next() {
				var id, attachType int
				var fn, fh string
				rows.Scan(&id, &fn, &fh, &attachType)
				toRemove = append(toRemove, id)
				errs["avatar_no_member"]++
				if fixErrors && toFix["avatar_no_member"] {
					if attachType == 1 {
						os.Remove(a.Setting("custom_avatar_dir") + "/" + fn)
					} else {
						os.Remove(a.getAttachmentFilename(fn, id, false, fh))
					}
				}
			}
			rows.Close()
		}
		if fixErrors && len(toRemove) > 0 && toFix["avatar_no_member"] {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}attachments WHERE ID_ATTACH IN (` + joinInts(toRemove) + `) AND ID_MEMBER != 0 AND ID_MSG = 0`))
		}
	}

	// 5. Attachments missing a message.
	{
		var toRemove []int
		rows, _ := a.DB.Query(a.Q(`
			SELECT a.ID_ATTACH, a.filename, a.file_hash
			FROM {$db_prefix}attachments AS a
				LEFT JOIN {$db_prefix}messages AS m ON (m.ID_MSG = a.ID_MSG)
			WHERE a.ID_MEMBER = 0 AND a.ID_MSG != 0 AND m.ID_MSG IS NULL`))
		if rows != nil {
			for rows.Next() {
				var id int
				var fn, fh string
				rows.Scan(&id, &fn, &fh)
				toRemove = append(toRemove, id)
				errs["attachment_no_msg"]++
				if fixErrors && toFix["attachment_no_msg"] {
					os.Remove(a.getAttachmentFilename(fn, id, false, fh))
				}
			}
			rows.Close()
		}
		if fixErrors && len(toRemove) > 0 && toFix["attachment_no_msg"] {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}attachments WHERE ID_ATTACH IN (` + joinInts(toRemove) + `) AND ID_MEMBER = 0 AND ID_MSG != 0`))
		}
	}

	// Build the ordered repair_errors list and totals.
	order := []string{"missing_thumbnail_parent", "parent_missing_thumbnail", "file_missing_on_disk", "file_wrong_size", "file_size_of_zero", "attachment_no_msg", "avatar_no_member"}
	page := &repairPage{Completed: fixErrors}
	total := 0
	for _, k := range order {
		page.RepairErrors = append(page.RepairErrors, RepairError{Key: k, Count: errs[k]})
		total += errs[k]
	}
	page.ErrorsFound = total > 0
	c.Page = page
	c.SubTemplate = templateAttachmentRepair
}

// ---- helpers ----

func isDir(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isWritableDir(path string) bool {
	if !isDir(path) {
		return false
	}
	// Probe writability by creating a temp file.
	f, err := os.CreateTemp(path, ".smfwrite_*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}
