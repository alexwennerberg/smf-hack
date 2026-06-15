package app

// Port of Sources/ManageRegistration.php: the registration center
// (?action=regcenter) — admin-registers a member, edits the agreement,
// reserved names, and registration settings. Independent of ManagePermissions.
// The agreement lives in <assetdir>/agreement.txt. use_graphic_library is
// false (this port uses the no-GD per-letter captcha fallback — documented).

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registerAction("regcenter", (*Ctx).RegCenter)
}

// RegCenter is RegCenter(): the dispatcher.
func (c *Ctx) RegCenter() {
	if c.REQUEST.Str("sa") == "browse" {
		typeParam := ""
		if c.REQUEST.Has("type") {
			typeParam = ";type=" + c.REQUEST.Str("type")
		}
		c.redirectExit("action=viewmembers;sa=browse" + typeParam)
	}

	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"register":      {c.AdminRegister, "moderate_forum"},
		"agreement":     {c.EditAgreement, "admin_forum"},
		"reservednames": {c.SetReserve, "admin_forum"},
		"settings":      {c.AdminSettings, "admin_forum"},
	}

	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		if c.allowedTo("moderate_forum") {
			sa = "register"
		} else {
			sa = "settings"
		}
	}
	c.isAllowedTo(subs[sa].perm)

	c.adminIndex("registration_center")
	c.loadLanguage("Login")

	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{
		Title:       c.Txt("registration_center"),
		Help:        "registrations",
		Description: c.Txt("admin_settings_desc"),
		Tabs: []AdminTab{
			{Title: c.Txt("admin_browse_register_new"), Description: c.Txt("admin_register_desc"), Href: scripturl + "?action=regcenter;sa=register", IsSelected: sa == "register", IsLast: !c.allowedTo("admin_forum")},
		},
	}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs,
			AdminTab{Title: c.Txt("smf11"), Description: c.Txt("smf12"), Href: scripturl + "?action=regcenter;sa=agreement", IsSelected: sa == "agreement"},
			AdminTab{Title: c.Txt("341"), Description: c.Txt("699"), Href: scripturl + "?action=regcenter;sa=reservednames", IsSelected: sa == "reservednames"},
			AdminTab{Title: c.Txt("settings"), Description: c.Txt("admin_settings_desc"), Href: scripturl + "?action=regcenter;sa=settings", IsSelected: sa == "settings", IsLast: true})
	}
	c.AdminTabs = tabs

	subs[sa].fn()
}

// AdminRegisterCtx backs template_admin_register.
type AdminRegisterCtx struct {
	RegistrationDone string
	MemberGroups     [][2]string // {id, name} ordered, 0 first
}

// AdminRegister is AdminRegister(): admin registers a new member by hand.
func (c *Ctx) AdminRegister() {
	a := c.App
	scripturl := a.ScriptURL

	page := &AdminRegisterCtx{}
	c.Page = page

	if c.POST.Has("regSubmit") {
		c.checkSession("post", "", true)

		user := strings.ReplaceAll(strings.ReplaceAll(c.POST.Str("user"), "\n", ""), "\r", "")
		require := "nothing"
		if c.POST.Has("emailActivate") {
			require = "activation"
		}
		opts := &regOptions{
			Interface:             "admin",
			Username:              Htmltrim(user),
			Email:                 Htmltrim(c.POST.Str("email")),
			Password:              c.POST.Str("password"),
			PasswordCheck:         c.POST.Str("password"),
			CheckReservedName:     true,
			CheckPasswordStrength: false,
			CheckEmailBan:         false,
			SendWelcomeEmail:      c.POST.Has("emailPassword") || c.POST.Str("password") == "",
			Require:               require,
			MemberGroup:           c.POST.Int("group"),
			HasMemberGroup:        true,
		}
		memberID := c.registerMember(opts)
		if memberID != 0 {
			link := `<a href="` + scripturl + `?action=profile;u=` + itoa(memberID) + `">` + user + `</a>`
			page.RegistrationDone = phpSprintf(c.Txt("admin_register_done"), link)
		}
	}

	c.SubTemplate = templateAdminRegister
	c.PageTitle = c.Txt("registration_center")

	extra := ""
	if !c.allowedTo("admin_forum") {
		extra = "\n\t\t\tAND ID_GROUP != 1"
	}
	page.MemberGroups = [][2]string{{"0", c.Txt("admin_register_group_none")}}
	rows, err := a.DB.Query(a.Q(`
		SELECT groupName, ID_GROUP
		FROM {$db_prefix}membergroups
		WHERE ID_GROUP != 3
			AND minPosts = -1` + extra + `
		ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`))
	if err == nil {
		for rows.Next() {
			var name string
			var id int
			rows.Scan(&name, &id)
			page.MemberGroups = append(page.MemberGroups, [2]string{itoa(id), name})
		}
		rows.Close()
	}
}

// EditAgreementCtx backs template_edit_agreement.
type EditAgreementCtx struct {
	Agreement        string
	Warning          string
	RequireAgreement bool
}

// EditAgreement is EditAgreement(): edit <assetdir>/agreement.txt.
func (c *Ctx) EditAgreement() {
	a := c.App
	path := filepath.Join(a.Config.AssetDir, "agreement.txt")

	if c.POST.Has("agreement") {
		c.checkSession("post", "", true)
		os.WriteFile(path, []byte(strings.ReplaceAll(c.POST.Str("agreement"), "\r", "")), 0644)
		req := "0"
		if c.POST.Has("requireAgreement") {
			req = "1"
		}
		a.UpdateSettings(map[string]string{"requireAgreement": req})
		c.redirectExit("action=regcenter;sa=agreement")
	}

	page := &EditAgreementCtx{}
	c.Page = page
	if data, err := os.ReadFile(path); err == nil {
		page.Agreement = Htmlspecialchars(string(data))
	}
	if !fileWritable(path) {
		page.Warning = c.Txt("smf320")
	}
	page.RequireAgreement = !a.SettingEmpty("requireAgreement")

	c.SubTemplate = templateEditAgreement
	c.PageTitle = c.Txt("smf11")
}

// fileWritable reports whether the path (or its dir, if absent) is writable.
func fileWritable(path string) bool {
	if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		f.Close()
		return true
	} else if os.IsNotExist(err) {
		// Could be created: check the directory.
		d, derr := os.Open(filepath.Dir(path))
		if derr == nil {
			d.Close()
			return true
		}
	}
	return false
}

// ReservedCtx backs template_edit_reserved_words.
type ReservedCtx struct {
	Words     string
	MatchWord bool
	MatchCase bool
	MatchUser bool
	MatchName bool
}

// SetReserve is SetReserve(): manage reserved names.
func (c *Ctx) SetReserve() {
	a := c.App

	if c.POST.Has("save_reserved_names") {
		c.checkSession("post", "", true)
		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		a.UpdateSettings(map[string]string{
			"reserveWord":  bit("matchword"),
			"reserveCase":  bit("matchcase"),
			"reserveUser":  bit("matchuser"),
			"reserveName":  bit("matchname"),
			"reserveNames": strings.ReplaceAll(c.POST.Str("reserved"), "\r", ""),
		})
	}

	page := &ReservedCtx{
		Words:     a.Setting("reserveNames"),
		MatchWord: a.Setting("reserveWord") == "1",
		MatchCase: a.Setting("reserveCase") == "1",
		MatchUser: a.Setting("reserveUser") == "1",
		MatchName: a.Setting("reserveName") == "1",
	}
	c.Page = page
	c.SubTemplate = templateEditReservedWords
	c.PageTitle = c.Txt("341")
}

// AdminSettingsCtx backs template_admin_settings.
type AdminSettingsCtx struct {
	CoppaPost             string
	UseGraphicLibrary     bool
	VerificationImageHref string
}

// AdminSettings is AdminSettings(): registration settings.
func (c *Ctx) AdminSettings() {
	a := c.App
	scripturl := a.ScriptURL

	if c.POST.Has("save") {
		c.checkSession("post", "", true)

		if c.POST.Int("coppaAge") != 0 && c.POST.Int("coppaType") != 0 && c.POST.Str("coppaPost") == "" && c.POST.Str("coppaFax") == "" {
			c.fatalError(c.Txt("admin_setting_coppa_require_contact"), false)
		}

		coppaPost := strings.ReplaceAll(c.POST.Str("coppaPost"), "\n", "<br />")
		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		visual := "0"
		if c.POST.Has("visual_verification_type") {
			visual = itoa(c.POST.Int("visual_verification_type"))
		}
		a.UpdateSettings(map[string]string{
			"registration_method":         itoa(c.POST.Int("registration_method")),
			"notify_new_registration":     bit("notify_new_registration"),
			"send_welcomeEmail":           bit("send_welcomeEmail"),
			"password_strength":           itoa(c.POST.Int("password_strength")),
			"disable_visual_verification": visual,
			"coppaAge":                    itoa(c.POST.Int("coppaAge")),
			"coppaType":                   itoa(c.POST.Int("coppaType")),
			"coppaPost":                   coppaPost,
			"coppaFax":                    c.POST.Str("coppaFax"),
			"coppaPhone":                  c.POST.Str("coppaPhone"),
		})
		c.redirectExit("action=regcenter;sa=settings")
	}

	page := &AdminSettingsCtx{}
	c.Page = page
	if !a.SettingEmpty("coppaPost") {
		page.CoppaPost = strings.NewReplacer("<br />", "\n", "<br>", "\n").Replace(a.Setting("coppaPost"))
	}
	// This port uses the no-GD per-letter captcha fallback.
	page.UseGraphicLibrary = false
	page.VerificationImageHref = scripturl + "?action=verificationcode;rand=" + c.Sc

	c.SubTemplate = templateAdminSettings
	c.PageTitle = c.Txt("registration_center")
}
