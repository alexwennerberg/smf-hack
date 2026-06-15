package app

// Port of Sources/Profile.php part 1: the ModifyProfile dispatcher and the
// shared profile member context. The view subactions live in
// profile_views.go, the edit-form subactions in profile_forms.go and the
// save path (ModifyProfile2) in profile2.go.

import "strings"

func init() {
	registerAction("profile", (*Ctx).ModifyProfile)
	registerAction("viewprofile", (*Ctx).ModifyProfile) // index.php alias
	registerAction("profile2", (*Ctx).ModifyProfile2)
	registerAction("trackip", func(c *Ctx) { c.TrackIP(0) })
}

// ProfileAvatar is $context['member']['avatar'] on the profile pages.
type ProfileAvatar struct {
	Name              string
	Href              string
	Custom            string
	Selection         string
	IDAttach          int
	Filename          string
	AllowServerStored bool
	AllowUpload       bool
	AllowExternal     bool
	Choice            string
	ServerPic         string
	External          string
}

// ProfileMember is the dispatcher-level $context['member'] data.
type ProfileMember struct {
	ID           int
	Username     string
	Name         string
	Email        string
	Posts        int
	HideEmail    int
	ShowOnline   int
	Registered   string
	Group        int
	GenderName   string
	Avatar       ProfileAvatar
	WebsiteTitle string
	WebsiteURL   string

	// account()/forumProfile()/theme()/notification() extras:
	IsAdmin            bool
	SecretQuestion     string
	BirthYear          string
	BirthMonth         string
	BirthDay           string
	Location           string
	Title              string
	Blurb              string
	Signature          string
	ThemeID            int
	ThemeName          string
	SmileySetID        string
	SmileySetName      string
	TimeFormat         string
	TimeOffset         string
	NotifyAnnouncement int
	NotifyOnce         int
	NotifySendBody     int
	NotifyTypes        int
	Options            map[string]string
}

// ProfileCtx is the page context shared by all the Profile templates.
type ProfileCtx struct {
	MemID           int
	IsOwner         bool
	Areas           []ProfileAreaGroup
	MenuSelected    string
	RequirePass     bool
	Member          *ProfileMember
	MemberFull      *MemberCtx // summary()
	ModifyError     map[string]bool
	ModifyErrorList []string

	// Filled per-subaction; see profile_views.go / profile_forms.go.
	Sub any
}

// ProfileAreaGroup is one block of the profile left menu.
type ProfileAreaGroup struct {
	Title string
	Areas []string // pre-built links
	Keys  []string // parallel area keys
}

func (g *ProfileAreaGroup) has(key string) bool {
	for _, k := range g.Keys {
		if k == key {
			return true
		}
	}
	return false
}

// profileSubActions: permission lists per subaction (own, any).
var profileSAAllowed = map[string][2][]string{
	"summary":         {{"profile_view_any", "profile_view_own"}, {"profile_view_any"}},
	"statPanel":       {{"profile_view_any", "profile_view_own"}, {"profile_view_any"}},
	"showPosts":       {{"profile_view_any", "profile_view_own"}, {"profile_view_any"}},
	"editBuddies":     {{"profile_extra_any", "profile_extra_own"}, {}},
	"trackUser":       {{"moderate_forum"}, {"moderate_forum"}},
	"trackIP":         {{"moderate_forum"}, {"moderate_forum"}},
	"showPermissions": {{"manage_permissions"}, {"manage_permissions"}},
	"account":         {{"manage_membergroups", "profile_identity_any", "profile_identity_own"}, {"manage_membergroups", "profile_identity_any"}},
	"forumProfile":    {{"profile_extra_any", "profile_extra_own"}, {"profile_extra_any"}},
	"theme":           {{"profile_extra_any", "profile_extra_own"}, {"profile_extra_any"}},
	"notification":    {{"profile_extra_any", "profile_extra_own"}, {"profile_extra_any"}},
	"pmprefs":         {{"profile_extra_any", "profile_extra_own"}, {"profile_extra_any"}},
	"deleteAccount":   {{"profile_remove_any", "profile_remove_own"}, {"profile_remove_any"}},
}

// ModifyProfile is ModifyProfile($post_errors): show/edit profiles.
func (c *Ctx) ModifyProfile() {
	c.modifyProfileErrors(nil)
}

// modifyProfileErrors is the real ModifyProfile, optionally with the save
// errors from ModifyProfile2.
func (c *Ctx) modifyProfileErrors(postErrors []string) {
	a := c.App
	scripturl := a.ScriptURL

	c.loadLanguage("Profile")

	// Set the profile layer to be displayed.
	c.TemplateLayers = append(c.TemplateLayers, "profile")

	// Did we get the user by name... or by ID_MEMBER?
	var memberResult []int
	if c.REQUEST.Has("user") {
		memberResult = c.loadMemberDataSet(nil, c.REQUEST.Str("user"), "profile")
	} else if !empty(c.REQUEST.Str("u")) {
		memberResult = c.loadMemberDataSet([]int{c.REQUEST.Int("u")}, "", "profile")
	} else {
		memberResult = c.loadMemberDataSet([]int{c.User.ID}, "", "profile")
	}

	// Check if loadMemberData() has returned a valid result.
	if len(memberResult) == 0 {
		c.fatalLangError("453", false)
	}

	// If all went well, we have a valid member ID!
	memID := memberResult[0]
	profile := c.memberProfiles[memID]

	page := &ProfileCtx{MemID: memID, ModifyError: map[string]bool{}}
	c.Page = page

	// Is this the profile of the user himself or herself?
	page.IsOwner = memID == c.User.ID

	// No Subaction?
	sa := c.REQUEST.Str("sa")
	if _, ok := profileSAAllowed[sa]; !ok {
		// Pick the first subaction you're allowed to see.
		if (c.allowedTo("profile_view_own") && page.IsOwner) || c.allowedTo("profile_view_any") {
			sa = "summary"
		} else if c.allowedTo("moderate_forum") {
			sa = "trackUser"
		} else if c.allowedTo("manage_permissions") {
			sa = "showPermissions"
		} else if (c.allowedTo("profile_identity_own") && page.IsOwner) || c.allowedTo("profile_identity_any") || c.allowedTo("manage_membergroups") {
			sa = "account"
		} else if (c.allowedTo("profile_extra_own") && page.IsOwner) || c.allowedTo("profile_extra_any") {
			sa = "forumProfile"
		} else if (c.allowedTo("profile_remove_own") && page.IsOwner) || c.allowedTo("profile_remove_any") {
			sa = "deleteAccount"
		} else if page.IsOwner {
			c.isAllowedTo("profile_view_own")
		} else {
			c.isAllowedTo("profile_view_any")
		}
		c.REQUEST.Set("sa", sa)
	}

	// Check the permissions for the given sub action.
	perms := profileSAAllowed[sa][0]
	if !page.IsOwner {
		perms = profileSAAllowed[sa][1]
	}
	if !c.allowedTo(perms...) {
		if len(perms) > 0 {
			c.isAllowedTo(perms[0])
		} else {
			c.fatalLangError("1", false)
		}
	}

	// Set the menu items in the left bar...
	if !c.User.IsGuest && ((page.IsOwner && c.allowedTo("profile_view_own")) ||
		c.allowedTo("profile_view_any", "moderate_forum", "manage_permissions")) {
		info := ProfileAreaGroup{Title: c.Txt("profileInfo")}

		if (page.IsOwner && c.allowedTo("profile_view_own")) || c.allowedTo("profile_view_any") {
			for _, area := range []string{"summary", "statPanel", "showPosts"} {
				info.Areas = append(info.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=`+area+`">`+c.Txt(area)+`</a>`)
				info.Keys = append(info.Keys, area)
			}
		}

		// Groups with moderator permissions can also....
		if c.allowedTo("moderate_forum") {
			for _, area := range []string{"trackUser", "trackIP"} {
				info.Areas = append(info.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=`+area+`">`+c.Txt(area)+`</a>`)
				info.Keys = append(info.Keys, area)
			}
		}
		if c.allowedTo("manage_permissions") {
			info.Areas = append(info.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=showPermissions">`+c.Txt("showPermissions")+`</a>`)
			info.Keys = append(info.Keys, "showPermissions")
		}
		page.Areas = append(page.Areas, info)
	}

	// Edit your/this person's profile?
	if (page.IsOwner && c.allowedTo("profile_identity_own", "profile_extra_own")) ||
		c.allowedTo("profile_identity_any", "profile_extra_any", "manage_membergroups") {
		edit := ProfileAreaGroup{Title: c.Txt("profileEdit")}

		if (page.IsOwner && c.allowedTo("profile_identity_own")) || c.allowedTo("profile_identity_any", "manage_membergroups") {
			edit.Areas = append(edit.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=account">`+c.Txt("account")+`</a>`)
			edit.Keys = append(edit.Keys, "account")
		}

		if (page.IsOwner && c.allowedTo("profile_extra_own")) || c.allowedTo("profile_extra_any") {
			for _, area := range []string{"forumProfile", "theme", "notification", "pmprefs"} {
				edit.Areas = append(edit.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=`+area+`">`+c.Txt(area)+`</a>`)
				edit.Keys = append(edit.Keys, area)
			}
		}

		if !a.SettingEmpty("enable_buddylist") && page.IsOwner && c.allowedTo("profile_extra_own", "profile_extra_any") {
			edit.Areas = append(edit.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=editBuddies">`+c.Txt("editBuddies")+`</a>`)
			edit.Keys = append(edit.Keys, "editBuddies")
		}
		page.Areas = append(page.Areas, edit)
	}

	// If you have permission to do something with this profile, you'll see
	// one or more actions.
	if (page.IsOwner && c.allowedTo("profile_remove_own")) || c.allowedTo("profile_remove_any") ||
		(!page.IsOwner && c.allowedTo("pm_send")) {
		action := ProfileAreaGroup{Title: c.Txt("profileAction")}

		// You shouldn't PM (or ban really..) yourself!!
		if !page.IsOwner && c.allowedTo("pm_send") {
			action.Areas = append(action.Areas, `<a href="`+scripturl+`?action=pm;sa=send;u=`+itoa(memID)+`">`+c.Txt("profileSendIm")+`</a>`)
			action.Keys = append(action.Keys, "send_pm")
		}
		// We don't wanna ban admins, do we?
		isAdminMember := profile.IDGroup == 1
		for _, g := range strings.Split(profile.AdditionalGroups, ",") {
			if g == "1" {
				isAdminMember = true
			}
		}
		if c.allowedTo("manage_bans") && !isAdminMember {
			action.Areas = append(action.Areas, `<a href="`+scripturl+`?action=ban;sa=add;u=`+itoa(memID)+`">`+c.Txt("profileBanUser")+`</a>`)
			action.Keys = append(action.Keys, "banUser")
		}

		// You may remove your own account 'cuz it's yours or you're an
		// admin.
		if (page.IsOwner && c.allowedTo("profile_remove_own")) || c.allowedTo("profile_remove_any") {
			action.Areas = append(action.Areas, `<a href="`+scripturl+`?action=profile;u=`+itoa(memID)+`;sa=deleteAccount">`+c.Txt("deleteAccount")+`</a>`)
			action.Keys = append(action.Keys, "deleteAccount")
		}
		page.Areas = append(page.Areas, action)
	}

	// This is here so the menu won't be shown unless it's actually needed.
	needed := false
	for _, g := range page.Areas {
		if g.has("trackUser") || g.has("showPermissions") || g.has("banUser") || g.has("deleteAccount") {
			needed = true
		}
		if g.Title == c.Txt("profileEdit") {
			needed = true
		}
	}
	if !needed {
		page.Areas = nil
	}

	// Set the selected items.
	page.MenuSelected = sa

	// All the subactions that require a user password in order to validate.
	page.RequirePass = sa == "account"

	page.Member = c.buildProfileMember(memID)

	// Call the appropriate subaction function.
	switch sa {
	case "summary":
		c.profileSummary(page, memID)
	case "statPanel":
		c.profileStatPanel(page, memID)
	case "showPosts":
		c.profileShowPosts(page, memID)
	case "editBuddies":
		c.profileEditBuddies(page, memID)
	case "trackUser":
		c.profileTrackUser(page, memID)
	case "trackIP":
		c.TrackIP(memID)
	case "showPermissions":
		c.profileShowPermissions(page, memID)
	case "account":
		c.profileAccount(page, memID)
	case "forumProfile":
		c.profileForumProfile(page, memID)
	case "theme":
		c.profileTheme(page, memID)
	case "notification":
		c.profileNotification(page, memID)
	case "pmprefs":
		c.profilePmprefs(page, memID)
	case "deleteAccount":
		c.profileDeleteAccount(page, memID)
	}

	if len(postErrors) > 0 {
		// Set all the errors so the template knows what went wrong.
		for _, e := range postErrors {
			page.ModifyError[e] = true
		}
		page.ModifyErrorList = postErrors
		c.rememberPostData(page)
	}

	// Set the page title if it's not already set...
	if c.PageTitle == "" {
		c.PageTitle = c.Txt("79") + " - " + c.Txt(sa)
	}
}

// buildProfileMember fills the dispatcher-level $context['member'].
func (c *Ctx) buildProfileMember(memID int) *ProfileMember {
	a := c.App
	scripturl := a.ScriptURL
	p := c.memberProfiles[memID]

	m := &ProfileMember{
		ID:         memID,
		Username:   p.MemberName,
		Name:       p.RealName,
		Email:      p.EmailAddress,
		Posts:      p.Posts,
		HideEmail:  p.HideEmail,
		ShowOnline: p.ShowOnline,
		Group:      p.IDGroup,
		Options:    p.Options,
	}
	if p.DateRegistered == 0 {
		m.Registered = c.Txt("470")
	} else {
		m.Registered = c.strftimeLocal("%Y-%m-%d",
			p.DateRegistered+int64((c.User.TimeOffset+float64(a.SettingInt("time_offset")))*3600))
	}
	if p.Gender == 2 {
		m.GenderName = "f"
	} else if p.Gender != 0 {
		m.GenderName = "m"
	}

	av := &m.Avatar
	av.Name = p.Avatar
	if p.IDAttach != 0 {
		if p.AttachType == 0 {
			av.Href = scripturl + "?action=dlattach;attach=" + itoa(p.IDAttach) + ";type=avatar"
		} else {
			av.Href = a.Setting("custom_avatar_url") + "/" + p.AttachFilename
		}
	}
	av.Custom = "http://"
	if strings.Contains(strings.ToLower(p.Avatar), "http://") {
		av.Custom = p.Avatar
	} else if p.Avatar != "" {
		av.Selection = p.Avatar
	}
	av.IDAttach = p.IDAttach
	av.Filename = p.AttachFilename
	av.AllowServerStored = c.allowedTo("profile_server_avatar") || memID != c.User.ID
	av.AllowUpload = c.allowedTo("profile_upload_avatar") || memID != c.User.ID
	av.AllowExternal = c.allowedTo("profile_remote_avatar") || memID != c.User.ID

	m.WebsiteTitle = p.WebsiteTitle
	m.WebsiteURL = p.WebsiteURL

	return m
}

// loadThemeOptions is loadThemeOptions($memID).
func (c *Ctx) loadThemeOptions(page *ProfileCtx, memID int) {
	a := c.App

	postOpts := c.POST.Arr("options")
	if postOpts != nil && c.POST.Arr("default_options") != nil {
		defs := c.POST.Arr("default_options")
		for _, k := range defs.Keys() {
			if !postOpts.Has(k) {
				postOpts.Set(k, defs.Get(k))
			}
		}
	}

	if page.IsOwner {
		page.Member.Options = c.Options
	} else {
		p := c.memberProfiles[memID]
		opts := map[string]string{}
		defaults := map[string]string{}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, variable, value
			FROM {$db_prefix}themes
			WHERE ID_THEME IN (1, ?)
				AND ID_MEMBER IN (-1, ?)`), p.IDTheme, memID)
		if err == nil {
			for rows.Next() {
				var id int
				var k, v string
				rows.Scan(&id, &k, &v)
				if id == -1 {
					defaults[k] = v
					continue
				}
				if postOpts != nil && postOpts.Has(k) {
					v = postOpts.Str(k)
				}
				opts[k] = v
			}
			rows.Close()
		}
		// Load up the default theme options for any missing.
		for k, v := range defaults {
			if _, ok := opts[k]; !ok {
				opts[k] = v
			}
		}
		page.Member.Options = opts
	}
}
