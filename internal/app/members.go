package app

// Port of loadMemberData() and loadMemberContext() from Sources/Load.php
// (lines 809–1098).

import (
	"strings"
)

// memberProfile is one $user_profile row ('normal' set).
type memberProfile struct {
	ID               int
	MemberName       string
	RealName         string
	EmailAddress     string
	HideEmail        int
	DateRegistered   int64
	Posts            int
	LastLogin        int64
	Signature        string
	PersonalText     string
	Location         string
	Gender           int
	Avatar           string
	WebsiteTitle     string
	WebsiteURL       string
	Birthdate        string
	MemberIP         string
	MemberIP2        string
	LngFile          string
	IDGroup          int
	IDPostGroup      int
	TimeOffset       float64
	ShowOnline       int
	BuddyList        string
	IsActivated      int
	UserTitle        string
	IsOnline         int64 // logTime, 0 if offline
	IDAttach         int
	AttachFilename   string
	AttachType       int
	MemberGroup      string
	MemberGroupColor string
	PostGroup        string
	PostGroupColor   string
	Stars            string

	// 'profile' set extras:
	IDTheme             int
	PmIgnoreList        string
	PmEmailNotify       int
	TimeFormat          string
	SecretQuestion      string
	AdditionalGroups    string
	SmileySet           string
	TotalTimeLoggedIn   int64
	NotifyAnnouncements int
	NotifyOnce          int
	NotifySendBody      int
	NotifyTypes         int
	OnlineURL           string
	PasswordSalt        string
	Options             map[string]string
}

// MemberCtx is $memberContext[id] — what the templates consume.
type MemberCtx struct {
	ID              int
	Username        string
	Name            string
	IsGuest         bool
	IsBuddy         bool
	IsReverseBuddy  bool
	CanViewProfile  bool
	IsTopicStarter  bool
	Title           string
	Href            string
	Link            string
	Email           string
	HideEmail       bool
	EmailPublic     bool
	Registered      string
	RegisteredTS    int64
	Blurb           string
	GenderName      string
	GenderImage     string
	WebsiteTitle    string
	WebsiteURL      string
	BirthDate       string
	Signature       string
	Location        string
	RealPosts       int
	Posts           string
	AvatarName      string
	AvatarImage     string
	AvatarHref      string
	AvatarURL       string
	LastLogin       string
	LastLoginTS     int64
	IP              string
	IP2             string
	IsOnline        bool
	OnlineText      string
	OnlineHref      string
	OnlineLink      string
	OnlineImageHref string
	OnlineLabel     string
	Language        string
	IsActivated     int
	IsBanned        bool
	Options         map[string]string
	Group           string
	GroupColor      string
	GroupID         int
	PostGroup       string
	PostGroupColor  string
	GroupStars      string
	LocalTime       string
}

// loadMemberData is loadMemberData($users) with the 'normal' set.
func (c *Ctx) loadMemberData(users []int) []int {
	return c.loadMemberDataSet(users, "", "normal")
}

// loadMemberDataSet is the full loadMemberData($users, $is_name, $set):
// pass name != "" to look up by memberName, set is "normal" or "profile".
func (c *Ctx) loadMemberDataSet(users []int, name string, set string) []int {
	a := c.App
	if len(users) == 0 && name == "" {
		return nil
	}
	if c.memberProfiles == nil {
		c.memberProfiles = map[int]*memberProfile{}
	}

	ids := make([]string, 0, len(users))
	seen := map[int]bool{}
	for _, u := range users {
		if !seen[u] && u != 0 {
			seen[u] = true
			ids = append(ids, itoa(u))
		}
	}
	if len(ids) == 0 && name == "" {
		return nil
	}

	where := "mem.ID_MEMBER IN (" + strings.Join(ids, ", ") + ")"
	var args []any
	if name != "" {
		where = "mem.memberName = ?"
		args = append(args, name)
	}

	titlesEnabled := !a.SettingEmpty("titlesEnable")
	titleCol := ""
	if titlesEnabled {
		titleCol = `,
			mem.usertitle`
	}

	columns := `
			IFNULL(lo.logTime, 0) AS isOnline, IFNULL(a.ID_ATTACH, 0) AS ID_ATTACH, IFNULL(a.filename, '') AS filename, IFNULL(a.attachmentType, 0) AS attachmentType,
			mem.signature, mem.personalText, mem.location, mem.gender, mem.avatar, mem.ID_MEMBER, mem.memberName,
			mem.realName, mem.emailAddress, mem.hideEmail, mem.dateRegistered, mem.websiteTitle, mem.websiteUrl,
			mem.birthdate, mem.memberIP, mem.memberIP2, mem.posts, mem.lastLogin,
			mem.ID_POST_GROUP, mem.lngfile, mem.ID_GROUP, mem.timeOffset, mem.showOnline,
			mem.buddy_list, IFNULL(mg.onlineColor, '') AS member_group_color, IFNULL(mg.groupName, '') AS member_group,
			IFNULL(pg.onlineColor, '') AS post_group_color, IFNULL(pg.groupName, '') AS post_group, mem.is_activated,
			CASE WHEN mem.ID_GROUP = 0 OR IFNULL(mg.stars, '') = '' THEN IFNULL(pg.stars, '') ELSE mg.stars END AS stars` + titleCol
	if set == "profile" {
		columns += `,
			mem.ID_THEME, mem.pm_ignore_list, mem.pm_email_notify, mem.timeFormat, mem.secretQuestion,
			mem.additionalGroups, mem.smileySet, mem.totalTimeLoggedIn, mem.notifyAnnouncements,
			mem.notifyOnce, mem.notifySendBody, mem.notifyTypes, IFNULL(lo.url, '') AS online_url, mem.passwordSalt`
	}

	rows, err := a.DB.Query(a.Q(`
		SELECT`+columns+`
		FROM {$db_prefix}members AS mem
			LEFT JOIN {$db_prefix}log_online AS lo ON (lo.ID_MEMBER = mem.ID_MEMBER)
			LEFT JOIN {$db_prefix}attachments AS a ON (a.ID_MEMBER = mem.ID_MEMBER)
			LEFT JOIN {$db_prefix}membergroups AS pg ON (pg.ID_GROUP = mem.ID_POST_GROUP)
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = mem.ID_GROUP)
		WHERE `+where), args...)
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()

	var loaded []int
	for rows.Next() {
		p := &memberProfile{Options: map[string]string{}}
		dest := []any{
			&p.IsOnline, &p.IDAttach, &p.AttachFilename, &p.AttachType,
			&p.Signature, &p.PersonalText, &p.Location, &p.Gender, &p.Avatar, &p.ID, &p.MemberName,
			&p.RealName, &p.EmailAddress, &p.HideEmail, &p.DateRegistered, &p.WebsiteTitle, &p.WebsiteURL,
			&p.Birthdate, &p.MemberIP, &p.MemberIP2, &p.Posts, &p.LastLogin,
			&p.IDPostGroup, &p.LngFile, &p.IDGroup, &p.TimeOffset, &p.ShowOnline,
			&p.BuddyList, &p.MemberGroupColor, &p.MemberGroup,
			&p.PostGroupColor, &p.PostGroup, &p.IsActivated, &p.Stars,
		}
		if titlesEnabled {
			dest = append(dest, &p.UserTitle)
		}
		if set == "profile" {
			dest = append(dest,
				&p.IDTheme, &p.PmIgnoreList, &p.PmEmailNotify, &p.TimeFormat, &p.SecretQuestion,
				&p.AdditionalGroups, &p.SmileySet, &p.TotalTimeLoggedIn, &p.NotifyAnnouncements,
				&p.NotifyOnce, &p.NotifySendBody, &p.NotifyTypes, &p.OnlineURL, &p.PasswordSalt)
		}
		if err := rows.Scan(dest...); err != nil {
			c.fatalDBError(err)
		}
		c.memberProfiles[p.ID] = p
		loaded = append(loaded, p.ID)
	}

	// Load this user's theme options.
	if len(loaded) > 0 {
		lids := make([]string, len(loaded))
		for i, id := range loaded {
			lids[i] = itoa(id)
		}
		orows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, variable, value
			FROM {$db_prefix}themes
			WHERE ID_MEMBER IN (` + strings.Join(lids, ", ") + `)`))
		if err == nil {
			for orows.Next() {
				var id int
				var k, v string
				orows.Scan(&id, &k, &v)
				if p, ok := c.memberProfiles[id]; ok {
					p.Options[k] = v
				}
			}
			orows.Close()
		}
	}

	// Are we loading any moderators?  If so, fix their group data...
	if len(loaded) > 0 && c.BoardInfo != nil && len(c.BoardInfo.Moderators) > 0 {
		var modGroup, modColor, modStars string
		err := a.DB.QueryRow(a.Q(`
			SELECT groupName, onlineColor, stars
			FROM {$db_prefix}membergroups
			WHERE ID_GROUP = 3
			LIMIT 1`)).Scan(&modGroup, &modColor, &modStars)
		if err == nil {
			for _, id := range loaded {
				if !c.BoardInfo.IsModerator(id) {
					continue
				}
				p := c.memberProfiles[id]
				// By popular demand, don't show admins or global moderators
				// as moderators.
				if p.IDGroup != 1 && p.IDGroup != 2 {
					p.MemberGroup = modGroup
				}
				if modStars != "" {
					p.Stars = modStars
				}
				if modColor != "" {
					p.MemberGroupColor = modColor
				}
			}
		}
	}

	return loaded
}

// loadMemberContext is loadMemberContext($user).
func (c *Ctx) loadMemberContext(user int) bool {
	a := c.App
	scripturl := a.ScriptURL

	if c.memberCtx == nil {
		c.memberCtx = map[int]*MemberCtx{}
	}
	if _, done := c.memberCtx[user]; done {
		return true
	}
	if user == 0 {
		return false
	}
	p, ok := c.memberProfiles[user]
	if !ok {
		return false
	}

	// Censor everything.
	signature := c.censorText(p.Signature)
	blurb := c.censorText(p.PersonalText)
	location := c.censorText(p.Location)

	// Set things up to be used before hand.
	gendertxt := ""
	if p.Gender == 2 {
		gendertxt = c.Txt("239")
	} else if p.Gender == 1 {
		gendertxt = c.Txt("238")
	}
	signature = strings.ReplaceAll(strings.ReplaceAll(signature, "\n", "<br />"), "\r", "")
	signature = c.parseBBCCached(signature, true, "sig"+itoa(p.ID))

	isOnline := (p.ShowOnline != 0 || c.allowedTo("moderate_forum")) && p.IsOnline > 0
	starsCount, starsImage := "", ""
	if p.Stars != "" {
		parts := strings.SplitN(p.Stars, "#", 2)
		starsCount = parts[0]
		if len(parts) > 1 {
			starsImage = parts[1]
		}
	}
	isBuddy := false
	for _, b := range c.User.Buddies {
		if b == p.ID {
			isBuddy = true
			break
		}
	}
	var buddyList []int
	isReverseBuddy := false
	if p.BuddyList != "" {
		for _, b := range strings.Split(p.BuddyList, ",") {
			id := atoi(b)
			buddyList = append(buddyList, id)
			if id == c.User.ID {
				isReverseBuddy = true
			}
		}
	}

	// If we're always html resizing, assume it's too large.
	avatarWidth, avatarHeight := "", ""
	if tooLarge := a.Setting("avatar_action_too_large"); tooLarge == "option_html_resize" || tooLarge == "option_js_resize" {
		if !a.SettingEmpty("avatar_max_width_external") {
			avatarWidth = ` width="` + a.Setting("avatar_max_width_external") + `"`
		}
		if !a.SettingEmpty("avatar_max_height_external") {
			avatarHeight = ` height="` + a.Setting("avatar_max_height_external") + `"`
		}
	}

	m := &MemberCtx{
		ID:             p.ID,
		Username:       p.MemberName,
		Name:           p.RealName,
		IsBuddy:        isBuddy,
		IsReverseBuddy: isReverseBuddy,
		Href:           scripturl + "?action=profile;u=" + itoa(p.ID),
		Link:           `<a href="` + scripturl + "?action=profile;u=" + itoa(p.ID) + `" title="` + c.Txt("92") + ` ` + p.RealName + `">` + p.RealName + `</a>`,
		Email:          p.EmailAddress,
		HideEmail: p.EmailAddress == "" || (!a.SettingEmpty("guest_hideContacts") && c.User.IsGuest) ||
			(p.HideEmail != 0 && !a.SettingEmpty("allow_hideEmail") && !c.allowedTo("moderate_forum") && c.User.ID != p.ID),
		EmailPublic: (p.HideEmail == 0 || a.SettingEmpty("allow_hideEmail")) &&
			(a.SettingEmpty("guest_hideContacts") || !c.User.IsGuest),
		Blurb:          blurb,
		GenderName:     gendertxt,
		WebsiteTitle:   p.WebsiteTitle,
		WebsiteURL:     p.WebsiteURL,
		Signature:      signature,
		Location:       location,
		RealPosts:      p.Posts,
		IP:             Htmlspecialchars(p.MemberIP),
		IP2:            Htmlspecialchars(p.MemberIP2),
		IsOnline:       isOnline,
		IsActivated:    p.IsActivated,
		IsBanned:       p.IsActivated >= 10,
		Options:        p.Options,
		Group:          p.MemberGroup,
		GroupColor:     p.MemberGroupColor,
		GroupID:        p.IDGroup,
		PostGroup:      p.PostGroup,
		PostGroupColor: p.PostGroupColor,
		LocalTime:      c.timeformatNoToday(nowUnix() + int64((p.TimeOffset-c.User.TimeOffset)*3600)),
	}

	if !a.SettingEmpty("titlesEnable") {
		m.Title = p.UserTitle
	}

	if p.Gender != 0 {
		gimg := "Female"
		if p.Gender == 1 {
			gimg = "Male"
		}
		m.GenderImage = `<img src="` + c.Theme.ImagesURL() + `/` + gimg + `.gif" alt="` + gendertxt + `" border="0" />`
	}

	if p.DateRegistered == 0 {
		m.Registered = c.Txt("470")
		m.RegisteredTS = 0
	} else {
		m.Registered = c.timeformat(p.DateRegistered)
		m.RegisteredTS = c.forumTime(true, p.DateRegistered)
	}

	if p.Birthdate == "" || p.Birthdate == "0001-01-01" {
		m.BirthDate = "0000-00-00"
	} else if strings.HasPrefix(p.Birthdate, "0004") {
		m.BirthDate = "0000" + p.Birthdate[4:]
	} else {
		m.BirthDate = p.Birthdate
	}

	if p.Posts > 100000 {
		m.Posts = c.Txt("683")
	} else if p.Posts == 1337 {
		m.Posts = "leet"
	} else {
		m.Posts = c.commaFormat(p.Posts)
	}

	// Avatar.
	m.AvatarName = p.Avatar
	if p.Avatar == "" {
		if p.IDAttach > 0 {
			href := scripturl + "?action=dlattach;attach=" + itoa(p.IDAttach) + ";type=avatar"
			if p.AttachType != 0 {
				href = a.Setting("custom_avatar_url") + "/" + p.AttachFilename
			}
			m.AvatarImage = `<img src="` + href + `" alt="" class="avatar" border="0" />`
			m.AvatarHref = href
		}
	} else if containsCI(p.Avatar, "http://") {
		m.AvatarImage = `<img src="` + p.Avatar + `"` + avatarWidth + avatarHeight + ` alt="" class="avatar" border="0" />`
		m.AvatarHref = p.Avatar
		m.AvatarURL = p.Avatar
	} else {
		m.AvatarImage = `<img src="` + a.Setting("avatar_url") + `/` + Htmlspecialchars(p.Avatar) + `" alt="" class="avatar" border="0" />`
		m.AvatarHref = a.Setting("avatar_url") + "/" + p.Avatar
		m.AvatarURL = a.Setting("avatar_url") + "/" + p.Avatar
	}

	if p.LastLogin == 0 {
		m.LastLogin = c.Txt("never")
	} else {
		m.LastLogin = c.timeformat(p.LastLogin)
		m.LastLoginTS = c.forumTime(false, p.LastLogin)
	}

	onlineKey := "online3"
	onlineLabelKey := "online5"
	if isOnline {
		onlineKey = "online2"
		onlineLabelKey = "online4"
	}
	m.OnlineText = c.Txt(onlineKey)
	m.OnlineHref = scripturl + "?action=pm;sa=send;u=" + itoa(p.ID)
	m.OnlineLink = `<a href="` + m.OnlineHref + `">` + c.Txt(onlineKey) + `</a>`
	buddyPrefix := ""
	if isBuddy {
		buddyPrefix = "buddy_"
	}
	onOff := "useroff"
	if isOnline {
		onOff = "useron"
	}
	m.OnlineImageHref = c.Theme.ImagesURL() + "/" + buddyPrefix + onOff + ".gif"
	m.OnlineLabel = c.Txt(onlineLabelKey)

	m.Language = ucwords(strings.NewReplacer("_", " ", "-utf8", "").Replace(p.LngFile))

	if starsCount != "" && starsImage != "" {
		star := `<img src="` + strings.ReplaceAll(c.Theme.ImagesURL()+"/"+starsImage, "$language", c.User.Language) + `" alt="*" border="0" />`
		m.GroupStars = strings.Repeat(star, atoi(starsCount))
	}

	c.memberCtx[user] = m
	return true
}

func containsCI(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func ucwords(s string) string {
	out := []byte(s)
	up := true
	for i := 0; i < len(out); i++ {
		ch := out[i]
		if up && ch >= 'a' && ch <= 'z' {
			out[i] = ch - 32
		}
		up = ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
	}
	return string(out)
}
