package app

// Port of Sources/Profile.php part 3: the edit-form subactions — account,
// forumProfile, getAvatars, theme, notification, pmprefs, deleteAccount,
// rememberPostData.

import (
	"os"
	"sort"
	"strings"
	"time"
)

// ProfileGroup is one assignable membergroup on the account page.
type ProfileGroup struct {
	ID              int
	Name            string
	IsPrimary       bool
	IsAdditional    bool
	CanBeAdditional bool
}

// ProfileAccountCtx is the data for the account template.
type ProfileAccountCtx struct {
	AllowEditUsername     bool
	AllowEditMembergroups bool
	AllowEditAccount      bool
	AllowHideEmail        bool
	AllowHideOnline       bool
	AllowEditName         bool
	MemberGroups          []*ProfileGroup
}

// profileAccount is account($memID).
func (c *Ctx) profileAccount(page *ProfileCtx, memID int) {
	a := c.App

	profile := c.memberProfiles[memID]
	sub := &ProfileAccountCtx{}
	page.Sub = sub

	// Allow an administrator to edit the username?
	sub.AllowEditUsername = c.GET.Has("changeusername") && c.allowedTo("admin_forum")

	// You might be allowed to only assign the membergroups, so let's check.
	sub.AllowEditMembergroups = c.allowedTo("manage_membergroups")
	sub.AllowEditAccount = (page.IsOwner && c.allowedTo("profile_identity_own")) || c.allowedTo("profile_identity_any")

	// How about their email address... online status, and name?
	sub.AllowHideEmail = !a.SettingEmpty("allow_hideEmail") || c.allowedTo("moderate_forum")
	sub.AllowHideOnline = !a.SettingEmpty("allow_hideOnline") || c.allowedTo("moderate_forum")
	sub.AllowEditName = !a.SettingEmpty("allow_editDisplayName") || c.allowedTo("moderate_forum")

	// Load up the existing contextual data.
	page.Member.IsAdmin = profile.IDGroup == 1
	page.Member.SecretQuestion = profile.SecretQuestion

	// You need 'manage membergroups' permission for this.
	if sub.AllowEditMembergroups {
		sub.MemberGroups = []*ProfileGroup{{
			ID:        0,
			Name:      c.Txt("no_primary_membergroup"),
			IsPrimary: profile.IDGroup == 0,
		}}
		curGroups := map[string]bool{}
		for _, g := range strings.Split(profile.AdditionalGroups, ",") {
			curGroups[g] = true
		}

		// Load membergroups, but only those groups the user can assign.
		rows, err := a.DB.Query(a.Q(`
			SELECT groupName, ID_GROUP
			FROM {$db_prefix}membergroups
			WHERE ID_GROUP != 3
				AND minPosts = -1
			ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`))
		if err == nil {
			for rows.Next() {
				var name string
				var id int
				rows.Scan(&name, &id)

				// We should skip the administrator group if they don't have
				// the admin_forum permission!
				if id == 1 && !c.allowedTo("admin_forum") {
					continue
				}

				sub.MemberGroups = append(sub.MemberGroups, &ProfileGroup{
					ID:              id,
					Name:            name,
					IsPrimary:       profile.IDGroup == id,
					IsAdditional:    curGroups[itoa(id)],
					CanBeAdditional: true,
				})
			}
			rows.Close()
		}
	}

	// (userLanguage selection is single-language in this port.)

	c.loadThemeOptions(page, memID)
	c.SubTemplate = templateProfileAccount
}

// ProfileAvatarEntry is one server-stored avatar (or directory).
type ProfileAvatarEntry struct {
	Filename string
	Checked  bool
	Name     string
	IsDir    bool
	Files    []*ProfileAvatarEntry
}

// ProfileForumCtx is the data for the forumProfile template.
type ProfileForumCtx struct {
	AvatarURL          string
	MaxSignatureLength int
	AllowEditTitle     bool
	ShowSpellchecking  bool
	Avatars            []*ProfileAvatarEntry
	AvatarSelected     string
}

// profileForumProfile is forumProfile($memID).
func (c *Ctx) profileForumProfile(page *ProfileCtx, memID int) {
	a := c.App

	profile := c.memberProfiles[memID]
	sub := &ProfileForumCtx{}
	page.Sub = sub

	sub.AvatarURL = a.Setting("avatar_url")
	sub.MaxSignatureLength = a.SettingInt("max_signatureLength")
	sub.AllowEditTitle = c.allowedTo("profile_title_any") || (c.allowedTo("profile_title_own") && page.IsOwner)

	birth := profile.Birthdate
	if birth == "" || birth == "0001-01-01" {
		birth = "0000-00-00"
	} else if strings.HasPrefix(birth, "0004") {
		birth = "0000" + birth[4:]
	}
	parts := strings.SplitN(birth, "-", 3)
	if len(parts) == 3 {
		page.Member.BirthYear = parts[0]
		page.Member.BirthMonth = parts[1]
		page.Member.BirthDay = parts[2]
	}

	page.Member.Location = profile.Location
	page.Member.Title = profile.UserTitle
	page.Member.Blurb = strings.NewReplacer("<", "&lt;", ">", "&gt;", "&amp;#039;", "&#039;").Replace(profile.PersonalText)
	page.Member.Signature = strings.NewReplacer("<br />", "\n", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#039;").Replace(profile.Signature)

	av := &page.Member.Avatar
	hasHTTP := strings.Contains(strings.ToLower(profile.Avatar), "http://")
	if profile.Avatar == "" && profile.IDAttach > 0 && av.AllowUpload {
		av.Choice = "upload"
		av.ServerPic = "blank.gif"
		av.External = "http://"
	} else if hasHTTP && av.AllowExternal {
		av.Choice = "external"
		av.ServerPic = "blank.gif"
		av.External = profile.Avatar
	} else if av.AllowServerStored && fileExists(a.Setting("avatar_directory")+"/"+profile.Avatar) {
		av.Choice = "server_stored"
		av.ServerPic = "blank.gif"
		if profile.Avatar != "" {
			av.ServerPic = profile.Avatar
		}
		av.External = "http://"
	} else {
		av.Choice = "server_stored"
		av.ServerPic = "blank.gif"
		av.External = "http://"
	}

	// Get a list of all the avatars.
	if av.AllowServerStored {
		if info, err := os.Stat(a.Setting("avatar_directory")); err == nil && info.IsDir() {
			sub.Avatars = c.getAvatars(page, "", 0)
		}
	}

	// Second level selected avatar...
	if i := strings.LastIndexByte(av.ServerPic, '/'); i >= 0 {
		sub.AvatarSelected = av.ServerPic[i+1:]
	}

	c.loadThemeOptions(page, memID)
	c.SubTemplate = templateProfileForumProfile
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getAvatars is getAvatars($directory, $level): recursive avatar listing.
func (c *Ctx) getAvatars(page *ProfileCtx, directory string, level int) []*ProfileAvatarEntry {
	a := c.App

	var result []*ProfileAvatarEntry

	// Open the directory..
	dirPath := a.Setting("avatar_directory")
	if directory != "" {
		dirPath += "/" + directory
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil
	}

	var dirs, files []string
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." || name == "blank.gif" || name == "index.php" {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, name)
		} else {
			files = append(files, name)
		}
	}

	// Sort the results... (natcasesort approximated case-insensitively.)
	caseless := func(ss []string) {
		sort.Slice(ss, func(i, j int) bool { return strings.ToLower(ss[i]) < strings.ToLower(ss[j]) })
	}
	caseless(dirs)
	caseless(files)

	serverPic := page.Member.Avatar.ServerPic

	if level == 0 {
		result = append(result, &ProfileAvatarEntry{
			Filename: "blank.gif",
			Checked:  serverPic == "" || serverPic == "blank.gif",
			Name:     c.Txt("422"),
		})
	}

	for _, line := range dirs {
		sub := directory
		if sub != "" {
			sub += "/"
		}
		tmp := c.getAvatars(page, sub+line, level+1)
		if len(tmp) > 0 {
			result = append(result, &ProfileAvatarEntry{
				Filename: Htmlspecialchars(line),
				Checked:  strings.Contains(serverPic, line+"/"),
				Name:     "[" + Htmlspecialchars(strings.ReplaceAll(line, "_", " ")) + "]",
				IsDir:    true,
				Files:    tmp,
			})
		}
	}

	for _, line := range files {
		ext := ""
		base := line
		if i := strings.LastIndexByte(line, '.'); i >= 0 {
			ext = strings.ToLower(line[i+1:])
			base = line[:i]
		}

		// Make sure it is an image.
		if ext != "gif" && ext != "jpg" && ext != "jpeg" && ext != "png" && ext != "bmp" {
			continue
		}

		result = append(result, &ProfileAvatarEntry{
			Filename: Htmlspecialchars(line),
			Checked:  line == serverPic,
			Name:     Htmlspecialchars(strings.ReplaceAll(base, "_", " ")),
		})
	}

	return result
}

// ProfileTimeFormat is one easy time format choice.
type ProfileTimeFormat struct {
	Format string
	Title  string
}

// ProfileSmileySet is one selectable smiley set.
type ProfileSmileySet struct {
	ID       string
	Name     string
	Selected bool
}

// ProfileThemeCtx is the data for the theme settings template.
type ProfileThemeCtx struct {
	EasyTimeformats      []ProfileTimeFormat
	CurrentForumTime     string
	CurrentForumTimeHour int
	SmileySets           []ProfileSmileySet
}

// profileTheme is theme($memID).
func (c *Ctx) profileTheme(page *ProfileCtx, memID int) {
	a := c.App

	profile := c.memberProfiles[memID]
	sub := &ProfileThemeCtx{}
	page.Sub = sub

	var name string
	a.DB.QueryRow(a.Q(`
		SELECT value
		FROM {$db_prefix}themes
		WHERE ID_THEME = ?
			AND variable = 'name'
		LIMIT 1`), profile.IDTheme).Scan(&name)

	page.Member.ThemeID = profile.IDTheme
	if profile.IDTheme == 0 {
		page.Member.ThemeName = c.Txt("theme_forum_default")
	} else {
		page.Member.ThemeName = name
	}
	page.Member.SmileySetID = profile.SmileySet
	page.Member.TimeFormat = profile.TimeFormat
	page.Member.TimeOffset = "0"
	if profile.TimeOffset != 0 {
		page.Member.TimeOffset = formatPercent(profile.TimeOffset)
	}

	sub.EasyTimeformats = []ProfileTimeFormat{
		{"", c.Txt("timeformat_easy0")},
		{"%B %d, %Y, %I:%M:%S %p", c.Txt("timeformat_easy1")},
		{"%B %d, %Y, %H:%M:%S", c.Txt("timeformat_easy2")},
		{"%Y-%m-%d, %H:%M:%S", c.Txt("timeformat_easy3")},
		{"%d %B %Y, %H:%M:%S", c.Txt("timeformat_easy4")},
		{"%d-%m-%Y, %H:%M:%S", c.Txt("timeformat_easy5")},
	}

	sub.CurrentForumTime = c.timeformatNoToday(time.Now().Unix() - int64(c.User.TimeOffset*3600))
	sub.CurrentForumTimeHour = atoi(c.strftimeLocal("%H", c.forumTime(false, 0)))

	sets := strings.Split("none,,"+a.Setting("smiley_sets_known"), ",")
	setNames := strings.Split(c.Txt("smileys_none")+"\n"+c.Txt("smileys_forum_board_default")+"\n"+a.Setting("smiley_sets_names"), "\n")
	for i, set := range sets {
		nameI := ""
		if i < len(setNames) {
			nameI = setNames[i]
		}
		s := ProfileSmileySet{
			ID:       Htmlspecialchars(set),
			Name:     Htmlspecialchars(nameI),
			Selected: set == page.Member.SmileySetID,
		}
		sub.SmileySets = append(sub.SmileySets, s)
		if s.Selected {
			page.Member.SmileySetName = s.Name
		}
	}

	c.loadThemeOptions(page, memID)
	c.SubTemplate = templateProfileThemeSettings
}

// ProfileNotifyBoard is one board notification row.
type ProfileNotifyBoard struct {
	ID   int
	Name string
	Href string
	Link string
	New  bool
}

// ProfileNotifyTopic is one topic notification row.
type ProfileNotifyTopic struct {
	ID         int
	PosterID   int
	PosterName string
	PosterLink string
	Subject    string
	Href       string
	Link       string
	New        bool
	NewFrom    int
	NewHref    string
	NewLink    string
	BoardID    int
	BoardName  string
	BoardLink  string
}

// ProfileNotifyCtx is the data for the notification template.
type ProfileNotifyCtx struct {
	BoardNotifications []ProfileNotifyBoard
	TopicNotifications []ProfileNotifyTopic
	PageIndex          string
	Start              int
}

// profileNotification is notification($memID).
func (c *Ctx) profileNotification(page *ProfileCtx, memID int) {
	a := c.App
	scripturl := a.ScriptURL

	profile := c.memberProfiles[memID]
	sub := &ProfileNotifyCtx{Start: c.Start}
	page.Sub = sub

	// All the boards with notification on..
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name, IFNULL(lb.ID_MSG, 0) AS boardRead, b.ID_MSG_UPDATED
		FROM {$db_prefix}log_notify AS ln, {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}log_boards AS lb ON (lb.ID_BOARD = b.ID_BOARD AND lb.ID_MEMBER = ` + itoa(c.User.ID) + `)
		WHERE ln.ID_MEMBER = ` + itoa(memID) + `
			AND b.ID_BOARD = ln.ID_BOARD
			AND ` + c.User.QuerySeeBoard + `
		ORDER BY b.boardOrder`))
	if err == nil {
		for rows.Next() {
			var id, boardRead, msgUpdated int
			var name string
			rows.Scan(&id, &name, &boardRead, &msgUpdated)
			sub.BoardNotifications = append(sub.BoardNotifications, ProfileNotifyBoard{
				ID:   id,
				Name: name,
				Href: scripturl + "?board=" + itoa(id) + ".0",
				Link: `<a href="` + scripturl + `?board=` + itoa(id) + `.0">` + name + `</a>`,
				New:  boardRead < msgUpdated,
			})
		}
		rows.Close()
	}

	var numTopics int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}log_notify AS ln, {$db_prefix}boards AS b, {$db_prefix}topics AS t
		WHERE ln.ID_MEMBER = ` + itoa(memID) + `
			AND t.ID_TOPIC = ln.ID_TOPIC
			AND b.ID_BOARD = t.ID_BOARD
			AND ` + c.User.QuerySeeBoard)).Scan(&numTopics)

	sub.PageIndex, _ = c.constructPageIndex(scripturl+"?action=profile;u="+itoa(memID)+";sa=notification",
		c.Start, numTopics, a.SettingInt("defaultMaxMessages"), false)

	// All the topics with notification on...
	rows, err = a.DB.Query(a.Q(`
		SELECT
			IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, -1)) + 1 AS new_from, b.ID_BOARD, b.name,
			t.ID_TOPIC, ms.subject, IFNULL(ms.ID_MEMBER, 0), IFNULL(mem.realName, ms.posterName) AS realName,
			ml.ID_MSG_MODIFIED
		FROM {$db_prefix}log_notify AS ln, {$db_prefix}boards AS b, {$db_prefix}topics AS t, {$db_prefix}messages AS ms, {$db_prefix}messages AS ml
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = ms.ID_MEMBER)
			LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(c.User.ID) + `)
			LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = b.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(c.User.ID) + `)
		WHERE ln.ID_MEMBER = ` + itoa(memID) + `
			AND t.ID_TOPIC = ln.ID_TOPIC
			AND ms.ID_MSG = t.ID_FIRST_MSG
			AND ml.ID_MSG = t.ID_LAST_MSG
			AND b.ID_BOARD = t.ID_BOARD
			AND ` + c.User.QuerySeeBoard + `
		ORDER BY ms.ID_MSG DESC
		LIMIT ` + itoa(a.SettingInt("defaultMaxMessages")) + ` OFFSET ` + itoa(c.Start)))
	if err == nil {
		for rows.Next() {
			var newFrom, boardID, topicID, posterID, msgModified int
			var boardName, subject, realName string
			rows.Scan(&newFrom, &boardID, &boardName, &topicID, &subject, &posterID, &realName, &msgModified)

			subject = c.censorText(subject)

			posterLink := realName
			if posterID != 0 {
				posterLink = `<a href="` + scripturl + `?action=profile;u=` + itoa(posterID) + `">` + realName + `</a>`
			}
			sub.TopicNotifications = append(sub.TopicNotifications, ProfileNotifyTopic{
				ID:         topicID,
				PosterID:   posterID,
				PosterName: realName,
				PosterLink: posterLink,
				Subject:    subject,
				Href:       scripturl + "?topic=" + itoa(topicID) + ".0",
				Link:       `<a href="` + scripturl + `?topic=` + itoa(topicID) + `.0">` + subject + `</a>`,
				New:        newFrom <= msgModified,
				NewFrom:    newFrom,
				NewHref:    scripturl + "?topic=" + itoa(topicID) + ".msg" + itoa(newFrom) + "#new",
				NewLink:    `<a href="` + scripturl + `?topic=` + itoa(topicID) + `.msg` + itoa(newFrom) + `#new">` + subject + `</a>`,
				BoardID:    boardID,
				BoardName:  boardName,
				BoardLink:  `<a href="` + scripturl + `?board=` + itoa(boardID) + `.0">` + boardName + `</a>`,
			})
		}
		rows.Close()
	}

	// What options are set?
	page.Member.NotifyAnnouncement = profile.NotifyAnnouncements
	page.Member.NotifyOnce = profile.NotifyOnce
	page.Member.NotifySendBody = profile.NotifySendBody
	page.Member.NotifyTypes = profile.NotifyTypes

	c.loadThemeOptions(page, memID)
	c.SubTemplate = templateProfileNotification
}

// ProfilePMPrefsCtx is the data for the pmprefs template.
type ProfilePMPrefsCtx struct {
	SendEmail  int
	BuddyList  string
	IgnoreList string
}

// profilePmprefs is pmprefs($memID).
func (c *Ctx) profilePmprefs(page *ProfileCtx, memID int) {
	a := c.App

	profile := c.memberProfiles[memID]
	sub := &ProfilePMPrefsCtx{}
	page.Sub = sub

	// Tell the template what they are....
	sub.SendEmail = profile.PmEmailNotify

	nameList := func(list string) string {
		if list == "" {
			return ""
		}
		var names []string
		rows, err := a.DB.Query(a.Q(`
			SELECT realName
			FROM {$db_prefix}members
			WHERE FIND_IN_SET(ID_MEMBER, '` + strings.ReplaceAll(list, "'", "''") + `')`))
		if err == nil {
			for rows.Next() {
				var name string
				rows.Scan(&name)
				names = append(names, name)
			}
			rows.Close()
		}
		return strings.Join(names, "\n")
	}

	if profile.PmIgnoreList != "*" {
		sub.IgnoreList = nameList(profile.PmIgnoreList)
	} else {
		sub.IgnoreList = "*"
	}

	// Get all their "buddies"...
	sub.BuddyList = nameList(profile.BuddyList)
	c.PageTitle = c.Txt("pmprefs") + ": " + c.Txt("144")

	c.loadThemeOptions(page, memID)
	c.SubTemplate = templateProfilePmprefs
}

// ProfileDeleteCtx is the data for the deleteAccount template.
type ProfileDeleteCtx struct {
	CanDeletePosts bool
	NeedsApproval  bool
}

// profileDeleteAccount is deleteAccount($memID): present a screen to make
// sure the user wants to be deleted.
func (c *Ctx) profileDeleteAccount(page *ProfileCtx, memID int) {
	a := c.App

	if !page.IsOwner {
		c.isAllowedTo("profile_remove_any")
	} else if !c.allowedTo("profile_remove_any") {
		c.isAllowedTo("profile_remove_own")
	}

	profile := c.memberProfiles[memID]
	sub := &ProfileDeleteCtx{}
	page.Sub = sub

	// Permissions for removing stuff...
	sub.CanDeletePosts = !page.IsOwner && c.allowedTo("moderate_forum")

	// Can they do this, or will they need approval?
	sub.NeedsApproval = page.IsOwner && !a.SettingEmpty("approveAccountDeletion") && !c.allowedTo("moderate_forum")
	c.PageTitle = c.Txt("deleteAccount") + ": " + profile.RealName

	c.SubTemplate = templateProfileDeleteAccount
}

// rememberPostData is rememberPostData(): overwrite member settings with the
// erroneous input so the form redisplays it.
func (c *Ctx) rememberPostData(page *ProfileCtx) {
	a := c.App
	scripturl := a.ScriptURL
	memID := c.REQUEST.Int("userID")
	profile := c.memberProfiles[memID]
	m := page.Member

	m.ID = memID
	m.Username = profile.MemberName
	if !empty(c.POST.Str("realName")) {
		m.Name = c.POST.Str("realName")
	} else {
		m.Name = profile.MemberName
	}
	m.Title = c.POST.Str("usertitle")
	m.Email = c.POST.Str("emailAddress")
	m.HideEmail = boolToInt(!empty(c.POST.Str("hideEmail")))
	m.ShowOnline = boolToInt(!empty(c.POST.Str("showOnline")))
	if empty(c.POST.Str("dateRegistered")) {
		m.Registered = c.Txt("470")
	} else {
		m.Registered = c.POST.Str("dateRegistered")
	}
	m.Blurb = strings.NewReplacer("<", "&lt;", ">", "&gt;", "&amp;#039;", "&#039;").Replace(c.POST.Str("personalText"))
	if c.POST.Int("gender") == 2 {
		m.GenderName = "f"
	} else if !empty(c.POST.Str("gender")) {
		m.GenderName = "m"
	} else {
		m.GenderName = ""
	}
	m.WebsiteTitle = c.POST.Str("websiteTitle")
	m.WebsiteURL = c.POST.Str("websiteUrl")
	m.BirthMonth = "00"
	if !empty(c.POST.Str("bday1")) {
		m.BirthMonth = itoa(c.POST.Int("bday1"))
	}
	m.BirthDay = "00"
	if !empty(c.POST.Str("bday2")) {
		m.BirthDay = itoa(c.POST.Int("bday2"))
	}
	m.BirthYear = "0000"
	if !empty(c.POST.Str("bday3")) {
		m.BirthYear = itoa(c.POST.Int("bday3"))
	}
	m.Signature = strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(c.POST.Str("signature"))
	m.Location = c.POST.Str("location")
	m.Posts = c.POST.Int("posts")

	av := &m.Avatar
	av.Name = c.POST.Str("avatar")
	if strings.Contains(strings.ToLower(av.Name), "http://") {
		av.Custom = av.Name
		av.Selection = ""
	} else {
		av.Custom = "http://"
		av.Selection = av.Name
	}
	av.Choice = "server_stored"
	if !empty(c.POST.Str("avatar_choice")) {
		av.Choice = c.POST.Str("avatar_choice")
	}
	av.External = "http://"
	if !empty(c.POST.Str("userpicpersonal")) {
		av.External = c.POST.Str("userpicpersonal")
	}

	m.TimeFormat = c.POST.Str("timeFormat")
	m.TimeOffset = "0"
	if !empty(c.POST.Str("timeOffset")) {
		m.TimeOffset = c.POST.Str("timeOffset")
	}
	m.SecretQuestion = c.POST.Str("secretQuestion")
	m.NotifyAnnouncement = boolToInt(!empty(c.POST.Str("notifyAnnouncements")))
	m.NotifyOnce = boolToInt(!empty(c.POST.Str("notifyOnce")))
	m.NotifySendBody = c.POST.Int("notifySendBody")
	m.NotifyTypes = c.POST.Int("notifyTypes")
	m.Group = c.POST.Int("ID_GROUP")
	if c.POST.Has("smileySet") {
		m.SmileySetID = c.POST.Str("smileySet")
	}

	// Overwrite the currently set membergroups with those you just selected.
	if sub, ok := page.Sub.(*ProfileAccountCtx); ok && c.allowedTo("manage_membergroups") && c.POST.Has("ID_GROUP") {
		addGroups := map[int]bool{}
		if arr := c.POST.Arr("additionalGroups"); arr != nil {
			for _, k := range arr.Keys() {
				addGroups[atoi(arr.Str(k))] = true
			}
		}
		for _, g := range sub.MemberGroups {
			g.IsPrimary = g.ID == c.POST.Int("ID_GROUP")
			g.IsAdditional = addGroups[g.ID]
		}
	}

	c.loadThemeOptions(page, memID)
	_ = scripturl
}
