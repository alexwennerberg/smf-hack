package app

// Port of Sources/Register.php. The GD captcha image (showCodeImage) is
// replaced by the no-GD fallback PHP itself uses when GD is absent: the
// per-letter font gifs from Themes/default/fonts/ (a legitimate PHP-side
// configuration). The WAV captcha is ported in subs_sound.go.

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registerAction("register", (*Ctx).Register)
	registerAction("register2", (*Ctx).Register2)
	registerAction("activate", (*Ctx).Activate)
	registerAction("coppa", (*Ctx).CoppaForm)
	registerAction("verificationcode", (*Ctx).VerificationCode)
}

// RegisterCtx is the page context for the Register templates.
type RegisterCtx struct {
	AllowHideEmail        bool
	RequireAgreement      bool
	Agreement             string
	ShowCoppa             bool
	CoppaDesc             string
	VisualVerification    bool
	UseGraphicLibrary     bool
	VerificationImageHref string
	Description           string
	// Coppa form bits:
	ForumContacts    string
	CoppaBody        string
	CoppaPost        string
	CoppaFax         string
	CoppaPhone       string
	CoppaManyOptions bool
	CoppaMemberID    int
	// Activation:
	MemberID        int
	CanActivate     bool
	DefaultUsername string
	// Sound:
	VerificationSoundHref string
}

// Register is Register(): the registration form.
func (c *Ctx) Register() {
	a := c.App

	// Check if the administrator has it disabled.
	if a.Setting("registration_method") == "3" {
		c.fatalLangError("registration_disabled", false)
	}

	// If this user is an admin - redirect them to the admin registration
	// page.
	if c.allowedTo("moderate_forum") && !c.User.IsGuest {
		c.redirectExit("action=regcenter;sa=register")
	} else if !c.User.IsGuest {
		c.redirectExit("")
	}

	c.loadLanguage("Login")

	page := &RegisterCtx{
		AllowHideEmail:   !a.SettingEmpty("allow_hideEmail"),
		RequireAgreement: !a.SettingEmpty("requireAgreement"),
	}
	c.Page = page
	c.SubTemplate = templateRegisterBefore

	// Under age restrictions?
	if !a.SettingEmpty("coppaAge") {
		page.ShowCoppa = true
		page.CoppaDesc = phpSprintf(c.Txt("register_age_confirmation"), a.Setting("coppaAge"))
	}

	c.PageTitle = c.Txt("97")

	// If you have to agree to the agreement, it needs to be fetched from the
	// file.
	if page.RequireAgreement {
		if data, err := os.ReadFile(filepath.Join(a.Config.AssetDir, "agreement.txt")); err == nil {
			page.Agreement = c.parseBBCCached(string(data), true, "agreement")
		}
	}

	// Generate a visual verification code to make sure the user is no bot.
	page.VisualVerification = a.Setting("disable_visual_verification") != "1"
	if page.VisualVerification {
		page.UseGraphicLibrary = false // no-GD path: per-letter images
		page.VerificationImageHref = a.ScriptURL + "?action=verificationcode;rand=" + md5hex(itoa(rand.Int()))

		// Only generate a new code if one hasn't been set yet.
		if !c.Session.Has("visual_verification_code") {
			// Skip I, J, L, O and Q.
			chars := "ABCDEFGHKMNPRSTUVWXYZ"
			code := make([]byte, 5)
			for i := range code {
				code[i] = chars[rand.Intn(len(chars))]
			}
			c.Session.Set("visual_verification_code", string(code))
		}
	}
}

// Register2 is Register2(): actually register the member.
func (c *Ctx) Register2() {
	a := c.App

	// Well, if you don't agree, you can't register.
	if !a.SettingEmpty("requireAgreement") && (empty(c.POST.Str("regagree")) || c.POST.Str("regagree") == "no") {
		c.redirectExit("")
	}

	// Make sure they came from *somewhere*, have a session.
	if c.Session.GetStr("old_url") == "" {
		c.redirectExit("action=register")
	}

	// You can't register if it's disabled.
	if a.Setting("registration_method") == "3" {
		c.fatalLangError("registration_disabled", false)
	}

	// Clean newlines from scalar POST values.
	post := func(key string) string {
		return Htmltrim(strings.NewReplacer("\n", "", "\r", "").Replace(c.POST.Str(key)))
	}

	// Are they under age, and under age users are banned?
	if !a.SettingEmpty("coppaAge") && a.SettingEmpty("coppaType") && !c.POST.Has("skip_coppa") {
		c.loadLanguage("Login")
		c.fatalLangError("under_age_registration_prohibited", false, a.Setting("coppaAge"))
	}

	// Check whether the visual verification code was entered correctly.
	if a.Setting("disable_visual_verification") != "1" {
		given := strings.ToUpper(c.REQUEST.Str("visual_verification_code"))
		want := c.Session.GetStr("visual_verification_code")
		if given == "" || given != want {
			errs := c.Session.GetInt("visual_errors") + 1
			c.Session.Set("visual_errors", errs)
			if errs > 3 && c.Session.Has("visual_verification_code") {
				c.Session.Delete("visual_verification_code")
			}
			c.fatalLangError("visual_verification_failed", false)
		}
		c.Session.Delete("visual_errors")
	}

	// Collect all extra registration fields someone might have filled in.
	extra := map[string]string{}
	for _, v := range []string{"websiteUrl", "websiteTitle", "location", "birthdate",
		"timeFormat", "buddy_list", "pm_ignore_list", "smileySet", "personalText", "avatar",
		"secretQuestion", "secretAnswer"} {
		if c.POST.Has(v) {
			extra[v] = Htmlspecialchars(post(v))
		}
	}
	// Normalise the website URL so it can't be a javascript:/data: link when
	// later rendered in an href (Profile2 does this; stock Register.php does
	// not — a divergence to close that XSS vector).
	if c.POST.Has("websiteUrl") {
		website := strings.TrimSpace(c.POST.Str("websiteUrl"))
		if len(website) > 0 && !strings.Contains(website, "://") {
			website = "http://" + website
		}
		if len(website) < 8 || (!strings.HasPrefix(website, "http://") && !strings.HasPrefix(website, "https://")) {
			website = ""
		}
		extra["websiteUrl"] = Htmlspecialchars(website)
	}
	for _, v := range []string{"pm_email_notify", "notifyTypes", "gender", "ID_THEME"} {
		if c.POST.Has(v) {
			extra[v] = itoa(c.POST.Int(v))
		}
	}
	for _, v := range []string{"notifyAnnouncements", "notifyOnce", "notifySendBody", "hideEmail", "showOnline"} {
		if c.POST.Has(v) {
			if empty(c.POST.Str(v)) {
				extra[v] = "0"
			} else {
				extra[v] = "1"
			}
		}
	}

	if c.POST.Str("secretAnswer") != "" {
		extra["secretAnswer"] = md5hex(post("secretAnswer"))
	}

	// Validation... even if we're not a mall.
	if c.POST.Has("realName") && (!a.SettingEmpty("allow_editDisplayName") || c.allowedTo("moderate_forum")) {
		realName := strings.TrimSpace(usernameStripRe.ReplaceAllString(post("realName"), " "))
		if strings.TrimSpace(realName) != "" && !c.isReservedName(realName, 0, true, false) && entityLen(realName) <= 60 {
			extra["realName"] = Htmlspecialchars(realName)
		}
	}

	// Birthdate parts...
	if !empty(c.POST.Str("bday1")) && !empty(c.POST.Str("bday2")) {
		extra["birthdate"] = padInt(c.POST.Int("bday3"), 4) + "-" + padInt(c.POST.Int("bday1"), 2) + "-" + padInt(c.POST.Int("bday2"), 2)
	}

	// Set the options needed for registration.
	require := "nothing"
	if !a.SettingEmpty("coppaAge") && !c.POST.Has("skip_coppa") {
		require = "coppa"
	} else if a.Setting("registration_method") == "1" {
		require = "activation"
	} else if !a.SettingEmpty("registration_method") {
		require = "approval"
	}

	opts := &regOptions{
		Interface:             "guest",
		Username:              post("user"),
		Email:                 post("email"),
		Password:              c.POST.Str("passwrd1"),
		PasswordCheck:         c.POST.Str("passwrd2"),
		CheckReservedName:     true,
		CheckPasswordStrength: true,
		CheckEmailBan:         true,
		SendWelcomeEmail:      !a.SettingEmpty("send_welcomeEmail"),
		Require:               require,
		ExtraRegisterVars:     extra,
		ThemeVars:             map[string]string{},
	}

	// Registration options are always default options...
	if defOpts := c.POST.Arr("default_options"); defOpts != nil {
		defOpts.Values(func(k string, v any) {
			if s, ok := v.(string); ok {
				opts.ThemeVars[k] = Htmlspecialchars(s)
			}
		})
	}
	if userOpts := c.POST.Arr("options"); userOpts != nil {
		userOpts.Values(func(k string, v any) {
			if s, ok := v.(string); ok {
				opts.ThemeVars[k] = Htmlspecialchars(s)
			}
		})
	}

	memberID := c.registerMember(opts)

	// If COPPA has been selected then things get complicated.
	if !a.SettingEmpty("coppaAge") && !c.POST.Has("skip_coppa") {
		c.redirectExit("action=coppa;member=" + itoa(memberID))
	} else if !a.SettingEmpty("registration_method") {
		// Basic template variable setup.
		c.loadLanguage("Login")
		page := &RegisterCtx{}
		if a.Setting("registration_method") == "2" {
			page.Description = c.Txt("approval_after_registration")
		} else {
			page.Description = c.Txt("activate_after_registration")
		}
		c.Page = page
		c.PageTitle = c.Txt("97")
		c.SubTemplate = templateRegisterAfter
	} else {
		c.setLoginCookie(60*a.SettingInt("cookieTime"), memberID,
			sha1hex(sha1hex(strings.ToLower(opts.Username)+opts.Password)+opts.PasswordSalt))
		c.redirectExit("action=login2;sa=check;member=" + itoa(memberID))
	}
}

func padInt(n, width int) string {
	s := itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

// Activate is Activate(): activate an account via emailed code.
func (c *Ctx) Activate() {
	a := c.App

	c.loadLanguage("Login")

	if empty(c.REQUEST.Str("u")) && c.POST.Str("user") == "" {
		if a.SettingEmpty("registration_method") || a.Setting("registration_method") == "3" {
			c.fatalLangError("1", true)
		}
		page := &RegisterCtx{
			MemberID:        0,
			CanActivate:     a.SettingEmpty("registration_method") || a.Setting("registration_method") == "1",
			DefaultUsername: c.GET.Str("user"),
		}
		c.Page = page
		c.SubTemplate = templateResendActivate
		c.PageTitle = c.Txt("invalid_activation_resend")
		return
	}

	// Get the code from the database...
	var idMember, isActivated int
	var validationCode, memberName, realName, emailAddress, passwd string
	var err error
	if empty(c.REQUEST.Str("u")) {
		err = a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER, validation_code, memberName, realName, emailAddress, is_activated, passwd
			FROM {$db_prefix}members
			WHERE memberName = ? OR emailAddress = ?
			LIMIT 1`), c.POST.Str("user"), c.POST.Str("user")).Scan(
			&idMember, &validationCode, &memberName, &realName, &emailAddress, &isActivated, &passwd)
	} else {
		err = a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER, validation_code, memberName, realName, emailAddress, is_activated, passwd
			FROM {$db_prefix}members
			WHERE ID_MEMBER = ?
			LIMIT 1`), c.REQUEST.Int("u")).Scan(
			&idMember, &validationCode, &memberName, &realName, &emailAddress, &isActivated, &passwd)
	}

	// Does this user exist at all?
	if err != nil {
		page := &RegisterCtx{MemberID: 0}
		c.Page = page
		c.SubTemplate = templateRetryActivate
		c.PageTitle = c.Txt("invalid_userid")
		return
	}

	emailChange := false

	// Change their email address? (they probably tried a fake one first :P.)
	if c.POST.Has("new_email") && c.REQUEST.Has("passwd") &&
		sha1hex(strings.ToLower(memberName)+c.REQUEST.Str("passwd")) == passwd &&
		(isActivated == 0 || isActivated == 2) {
		if a.SettingEmpty("registration_method") || a.Setting("registration_method") == "3" {
			c.fatalLangError("1", true)
		}

		newEmail := c.POST.Str("new_email")
		if !emailRe.MatchString(newEmail) {
			c.fatalError(phpSprintf(c.Txt("500"), Htmlspecialchars(newEmail)), false)
		}

		// Make sure their email isn't banned.
		c.isBannedEmail(newEmail, "cannot_register", c.Txt("ban_register_prohibited"))

		// Ummm... don't even dare try to take someone else's email!!
		var dummy int
		if a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE emailAddress = ? LIMIT 1`), newEmail).Scan(&dummy) == nil {
			c.fatalError(phpSprintf(c.Txt("730"), Htmlspecialchars(newEmail)), false)
		}

		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET emailAddress = ? WHERE ID_MEMBER = ?`), newEmail, idMember)
		emailAddress = newEmail
		emailChange = true
	}

	// Resend the password, but only if the account wasn't activated yet.
	if c.REQUEST.Str("sa") == "resend" && (isActivated == 0 || isActivated == 2) && c.REQUEST.Str("code") == "" {
		msgKey := "resend_activate_message"
		if !a.SettingEmpty("registration_method") && a.Setting("registration_method") != "1" {
			msgKey = "resend_pending_message"
		}
		c.sendmail([]string{emailAddress}, c.Txt("register_subject"),
			phpSprintf(c.Txt(msgKey), realName, memberName, validationCode,
				a.ScriptURL+"?action=activate;u="+itoa(idMember)+";code="+validationCode), "")

		c.PageTitle = c.Txt("invalid_activation_resend")
		if emailChange {
			c.fatalError(c.Txt("change_email_success"), false)
		}
		c.fatalError(c.Txt("resend_email_success"), false)
	}

	// Quit if this code is not right.
	if empty(c.REQUEST.Str("code")) || validationCode != c.REQUEST.Str("code") {
		if isActivated != 0 {
			c.fatalLangError("already_activated", false)
		} else if validationCode == "" {
			c.loadLanguage("Profile")
			c.fatalError(c.Txt("registration_not_approved")+` <a href="`+a.ScriptURL+`?action=activate;user=`+memberName+`">`+c.Txt("662")+`</a>.`, false)
		}

		page := &RegisterCtx{MemberID: idMember}
		c.Page = page
		c.SubTemplate = templateRetryActivate
		c.PageTitle = c.Txt("invalid_activation_code")
		return
	}

	// Validation complete - update the database!
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET is_activated = 1, validation_code = '' WHERE ID_MEMBER = ?`), idMember)

	// Also do a proper member stat re-evaluation.
	c.updateStatsMemberRecount()

	if !c.POST.Has("new_email") && isActivated != 2 {
		c.adminNotify("activation", idMember, memberName)
	}

	c.Page = &LoginCtx{
		DefaultUsername: memberName,
		Description:     c.Txt("activate_success"),
	}
	c.PageTitle = c.Txt("245")
	c.SubTemplate = templateLogin
}

// CoppaForm is CoppaForm(): under-age registration paperwork.
func (c *Ctx) CoppaForm() {
	a := c.App

	c.loadLanguage("Login")

	// No User ID??
	if !c.GET.Has("member") {
		c.fatalLangError("1", true)
	}

	// Get the user details...
	var username string
	err := a.DB.QueryRow(a.Q(`
		SELECT memberName
		FROM {$db_prefix}members
		WHERE ID_MEMBER = ?
			AND is_activated = 5`), c.GET.Int("member")).Scan(&username)
	if err != nil {
		c.fatalLangError("1", true)
	}

	if c.GET.Has("form") {
		// Some simple contact stuff for the forum.
		contacts := ""
		if !a.SettingEmpty("coppaPost") {
			contacts += a.Setting("coppaPost") + "<br /><br />"
		}
		if !a.SettingEmpty("coppaFax") {
			contacts += a.Setting("coppaFax") + "<br />"
		}
		if contacts != "" {
			contacts = a.Config.MbName + "<br />" + contacts
		}

		if !c.GET.Has("dl") {
			// Showing template?
			ul := `<u>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</u>`
			c.TemplateLayers = []string{}
			page := &RegisterCtx{
				ForumContacts: contacts,
				CoppaBody: strings.NewReplacer("{PARENT_NAME}", ul, "{CHILD_NAME}", ul,
					"{USER_NAME}", username).Replace(c.Txt("coppa_form_body")),
			}
			c.Page = page
			c.SubTemplate = templateCoppaForm
			c.PageTitle = c.Txt("coppa_form_title")
		} else {
			// Downloading.
			ul := "                "
			crlf := "\r\n"
			data := contacts + crlf + c.Txt("coppa_form_address") + ":" + crlf + c.Txt("coppa_form_date") + ":" + crlf + crlf + crlf + c.Txt("coppa_form_body")
			data = strings.NewReplacer("{PARENT_NAME}", ul, "{CHILD_NAME}", ul, "{USER_NAME}", username,
				"<br>", crlf, "<br />", crlf).Replace(data)

			c.W.Header().Set("Connection", "close")
			c.W.Header().Set("Content-Disposition", `attachment; filename="approval.txt"`)
			c.W.Header().Set("Content-Type", "application/octet-stream")
			c.Out.WriteString(data)
			c.exit()
		}
	} else {
		page := &RegisterCtx{
			CoppaBody:        strings.ReplaceAll(c.Txt("coppa_after_registration"), "{MINIMUM_AGE}", a.Setting("coppaAge")),
			CoppaManyOptions: !a.SettingEmpty("coppaPost") && !a.SettingEmpty("coppaFax"),
			CoppaPost:        a.Setting("coppaPost"),
			CoppaFax:         a.Setting("coppaFax"),
			CoppaMemberID:    c.GET.Int("member"),
		}
		if !a.SettingEmpty("coppaPhone") {
			page.CoppaPhone = strings.ReplaceAll(c.Txt("coppa_send_by_phone"), "{PHONE_NUMBER}", a.Setting("coppaPhone"))
		}
		c.Page = page
		c.PageTitle = c.Txt("coppa_title")
		c.SubTemplate = templateCoppa
	}
}

// VerificationCode is VerificationCode(): serve the captcha (letter images
// or WAV sound).
func (c *Ctx) VerificationCode() {
	a := c.App
	code := c.Session.GetStr("visual_verification_code")

	// Somehow no code was generated or the session was lost.
	if code == "" {
		c.W.WriteHeader(408)
		c.exit()
	}

	// Show a window that will play the verification code.
	if c.REQUEST.Has("sound") {
		c.loadLanguage("Login")
		page := &RegisterCtx{
			VerificationSoundHref: a.ScriptURL + "?action=verificationcode;rand=" + md5hex(itoa(rand.Int())) + ";format=.wav",
		}
		c.Page = page
		c.SubTemplate = templateVerificationSound
		c.TemplateLayers = []string{}
		c.obExit(nil, nil, false)
	}

	if c.REQUEST.Str("format") == ".wav" {
		if !c.createWaveFile(code) {
			c.W.WriteHeader(400)
		}
		c.exit()
	}

	// Letter image (the no-GD path).
	if c.REQUEST.Has("letter") {
		letter := c.REQUEST.Int("letter")
		if letter > 0 && letter <= len(code) {
			if c.showLetterImage(strings.ToLower(string(code[letter-1]))) {
				c.exit()
			}
		}
	}
	c.W.WriteHeader(400)
	c.exit()
}

// showLetterImage is showLetterImage($letter) from Subs-Graphics.php.
func (c *Ctx) showLetterImage(letter string) bool {
	fontsDir := filepath.Join(c.App.Config.AssetDir, "Themes/default/fonts")
	entries, err := os.ReadDir(fontsDir)
	if err != nil {
		return false
	}
	var fontList []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			if _, err := os.Stat(filepath.Join(fontsDir, e.Name()+".gdf")); err == nil {
				fontList = append(fontList, e.Name())
			}
		}
	}
	if len(fontList) == 0 {
		return false
	}

	// Pick a random font.
	font := fontList[rand.Intn(len(fontList))]

	data, err := os.ReadFile(filepath.Join(fontsDir, font, letter+".gif"))
	if err != nil {
		return false
	}
	c.W.Header().Set("Content-Type", "image/gif")
	c.W.Write(data)
	return true
}
