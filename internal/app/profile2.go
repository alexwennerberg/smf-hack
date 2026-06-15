package app

// Port of Sources/Profile.php part 4: the save path — ModifyProfile2,
// saveProfileChanges, makeThemeChanges, makeNotificationChanges,
// makeAvatarChanges, deleteAccount2.

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// profile2SAAllowed: (own perms, any perms, session type, require password).
var profile2SAAllowed = map[string]struct {
	own, any    []string
	sessionType string
	requirePass bool
}{
	"account":         {[]string{"manage_membergroups", "profile_identity_any", "profile_identity_own"}, []string{"manage_membergroups", "profile_identity_any"}, "post", true},
	"forumProfile":    {[]string{"profile_extra_any", "profile_extra_own"}, []string{"profile_extra_any"}, "post", false},
	"theme":           {[]string{"profile_extra_any", "profile_extra_own"}, []string{"profile_extra_any"}, "post", false},
	"notification":    {[]string{"profile_extra_any", "profile_extra_own"}, []string{"profile_extra_any"}, "post", false},
	"pmprefs":         {[]string{"profile_extra_any", "profile_extra_own"}, []string{"profile_extra_any"}, "post", false},
	"deleteAccount":   {[]string{"profile_remove_any", "profile_remove_own"}, []string{"profile_remove_any"}, "post", true},
	"activateAccount": {nil, []string{"moderate_forum"}, "get", false},
}

// profileSaveState carries the working state of a profile save.
type profileSaveState struct {
	vars           map[string]any // profile_vars
	errors         []string
	newPassEmail   bool
	validationCode string
	isOwner        bool
	memID          int
}

// ModifyProfile2 is ModifyProfile2(): execute the modifications.
func (c *Ctx) ModifyProfile2() {
	a := c.App
	scripturl := a.ScriptURL

	c.loadLanguage("Profile")

	sa := c.REQUEST.Str("sa")
	conf, ok := profile2SAAllowed[sa]
	if empty(sa) || !ok {
		c.fatalLangError("453", false)
	}

	c.checkSession(conf.sessionType, "", true)

	st := &profileSaveState{vars: map[string]any{}}

	// Clean up the POST variables: htmltrim + htmlspecialchars on values.
	for _, k := range c.POST.Keys() {
		if v, isStr := c.POST.Get(k).(string); isStr {
			c.POST.Set(k, Htmlspecialchars(Htmltrim(v)))
		}
	}

	// Search for the member being edited.
	memberResult := c.loadMemberDataSet([]int{c.REQUEST.Int("userID")}, "", "profile")
	if len(memberResult) == 0 {
		c.fatalLangError("453", false)
	}
	st.memID = memberResult[0]
	profile := c.memberProfiles[st.memID]

	// Are you modifying your own, or someone else's?
	st.isOwner = c.User.ID == st.memID
	if !st.isOwner {
		c.validateSession()
	}

	// Check profile editing permissions.
	perms := conf.own
	if !st.isOwner {
		perms = conf.any
	}
	if !c.allowedTo(perms...) {
		if len(perms) > 0 {
			c.isAllowedTo(perms[0])
		} else {
			c.fatalLangError("1", false)
		}
	}

	// If this is yours, check the password.
	if st.isOwner && conf.requirePass {
		// You didn't even enter a password!
		if strings.TrimSpace(c.POST.Str("oldpasswrd")) == "" {
			st.errors = append(st.errors, "no_password")
		}

		// Undo the POST cleaning for the password check.
		oldPass := unHtmlspecialchars(c.POST.Str("oldpasswrd"))

		// Bad password!!!
		if c.User.Passwd != fmt.Sprintf("%x", sha1.Sum([]byte(strings.ToLower(profile.MemberName)+oldPass))) {
			st.errors = append(st.errors, "bad_password")
		}
	}

	// If the user is an admin - see if they are resetting someones
	// username.
	if c.User.IsAdmin && c.POST.Has("memberName") {
		// Do the reset... this will send them an email too.
		c.resetPassword(st.memID, c.POST.Str("memberName"))
	}

	// Change the IP address in the database.
	if st.isOwner {
		st.vars["memberIP"] = c.User.IP
	}

	// Now call the sub-action function...
	if c.POST.Str("sa") == "deleteAccount" {
		c.deleteAccount2(st)
		if len(st.errors) == 0 {
			c.redirectExit("")
		}
	} else {
		c.saveProfileChanges(st)
	}

	// There was a problem, let them try to re-enter.
	if len(st.errors) > 0 {
		// Load the language file so we can give a nice explanation of the
		// errors.
		c.loadLanguage("Errors")

		c.REQUEST.Set("sa", c.POST.Str("sa"))
		c.REQUEST.Set("u", itoa(st.memID))
		c.modifyProfileErrors(st.errors)
		return
	}

	if len(st.vars) > 0 {
		a.updateMemberDataMap(st.memID, st.vars)
	}

	// What if this is the newest member?
	if a.SettingInt("latestMember") == st.memID {
		c.updateStatsMember(0, "")
	} else if _, ok := st.vars["realName"]; ok {
		a.UpdateSettings(map[string]string{"memberlist_updated": i64toa(time.Now().Unix())})
	}

	// If the member changed his/her birthdate, update calendar statistics.
	if _, ok := st.vars["birthdate"]; ok {
		a.updateStatsCalendar()
	} else if _, ok := st.vars["realName"]; ok {
		a.updateStatsCalendar()
	}

	// Send an email?
	if st.newPassEmail {
		// Send off the email.
		c.sendmail([]string{c.POST.Str("emailAddress")}, c.Txt("activate_reactivate_title")+" "+a.Config.MbName,
			c.Txt("activate_reactivate_mail")+"\n\n"+
				scripturl+"?action=activate;u="+itoa(st.memID)+";code="+st.validationCode+"\n\n"+
				c.Txt("activate_code")+": "+st.validationCode+"\n\n"+
				c.Txt("130"), "")

		// Log the user out.
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE ID_MEMBER = ?`), st.memID)
		c.Session.Set("log_time", 0)
		c.Session.Delete("login_" + a.Config.CookieName)

		c.destroyLoginCookie()
		c.loadUserSettings()

		// Send them to the done-with-registration-login screen.
		page := &RegisterCtx{Description: c.Txt("activate_changed_email")}
		c.Page = page
		c.PageTitle = c.Txt("79")
		c.SubTemplate = templateRegisterAfter
		return
	} else if st.isOwner {
		// Log them back in.
		if pass1 := c.POST.Str("passwrd1"); pass1 != "" {
			plain := unHtmlspecialchars(pass1)
			hash := fmt.Sprintf("%x", sha1.Sum([]byte(strings.ToLower(profile.MemberName)+plain)))
			c.setLoginCookie(60*a.SettingInt("cookieTime"), st.memID,
				fmt.Sprintf("%x", sha1.Sum([]byte(hash+profile.PasswordSalt))))
		}

		c.loadUserSettings()
		c.writeLog(false)
	}

	// Back to same subaction page..
	c.redirectExit("action=profile;u=" + itoa(st.memID) + ";sa=" + c.REQUEST.Str("sa"))
}

// destroyLoginCookie blanks the login cookie (the $_COOKIE reset in PHP).
func (c *Ctx) destroyLoginCookie() {
	c.setLoginCookie(-3600, 0, "")
}

var (
	birthdateRe  = regexp.MustCompile(`(\d{4})[\-\., ](\d{2})[\-\., ](\d{2})`)
	langCleanRe  = regexp.MustCompile(`[^\w\-]`)
	msnMatchRe   = emailRe
	floatCommaRe = strings.NewReplacer(",", ".")
)

// checkdate is PHP checkdate($month, $day, $year).
func checkdate(month, day, year int) bool {
	if month < 1 || month > 12 || day < 1 || year < 1 {
		return false
	}
	daysIn := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	max := daysIn[month-1]
	if month == 2 && (year%4 == 0 && (year%100 != 0 || year%400 == 0)) {
		max = 29
	}
	return day <= max
}

// saveProfileChanges is saveProfileChanges(&$profile_vars, &$post_errors,
// $memID).
func (c *Ctx) saveProfileChanges(st *profileSaveState) {
	a := c.App
	memID := st.memID
	oldProfile := c.memberProfiles[memID]

	// Permissions...
	var changeIdentity, changeOther bool
	if st.isOwner {
		changeIdentity = c.allowedTo("profile_identity_any", "profile_identity_own")
		changeOther = c.allowedTo("profile_extra_any", "profile_extra_own")
	} else {
		changeIdentity = c.allowedTo("profile_identity_any")
		changeOther = c.allowedTo("profile_extra_any")
	}

	// Arrays of all the changes - makes things easier.
	profileBools := []string{"notifyAnnouncements", "notifyOnce", "notifySendBody"}
	profileInts := []string{"pm_email_notify", "notifyTypes", "ICQ", "gender", "ID_THEME"}
	profileFloats := []string{"timeOffset"}
	profileStrings := []string{
		"websiteUrl", "websiteTitle",
		"AIM", "YIM",
		"location", "birthdate",
		"timeFormat",
		"buddy_list",
		"pm_ignore_list",
		"smileySet",
		"signature", "personalText", "avatar",
	}

	// Fix the spaces in messenger screennames...
	for _, v := range []string{"MSN", "AIM", "YIM"} {
		if c.POST.Has(v) {
			c.POST.Set(v, strings.ReplaceAll(c.POST.Str(v), " ", "+"))
		}
	}

	// Make sure the MSN one is an email address, not something like 'none'
	// :P.
	if c.POST.Has("MSN") && (c.POST.Str("MSN") == "" || msnMatchRe.MatchString(c.POST.Str("MSN"))) {
		profileStrings = append(profileStrings, "MSN")
	}

	// Validate the title...
	if !a.SettingEmpty("titlesEnable") && (c.allowedTo("profile_title_any") || (c.allowedTo("profile_title_own") && st.isOwner)) {
		profileStrings = append(profileStrings, "usertitle")
	}

	// Validate the timeOffset...
	if c.POST.Has("timeOffset") {
		c.POST.Set("timeOffset", floatCommaRe.Replace(c.POST.Str("timeOffset")))
		off := toFloat(c.POST.Get("timeOffset"))
		if off < -23.5 || off > 23.5 {
			st.errors = append(st.errors, "bad_offset")
		}
	}

	// Fix the URL...
	if c.POST.Has("websiteUrl") {
		url := c.POST.Str("websiteUrl")
		if len(strings.TrimSpace(url)) > 0 && !strings.Contains(url, "://") {
			url = "http://" + url
		}
		if len(url) < 8 || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
			url = ""
		}
		c.POST.Set("websiteUrl", url)
	}

	// Birthdate handling.
	if c.POST.Has("birthdate") {
		if m := birthdateRe.FindStringSubmatch(c.POST.Str("birthdate")); m != nil {
			year, month, day := atoi(m[1]), atoi(m[2]), atoi(m[3])
			if year < 4 {
				year = 4
			}
			if checkdate(month, day, year) {
				c.POST.Set("birthdate", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
			} else {
				c.POST.Set("birthdate", "0001-01-01")
			}
		} else {
			c.POST.Delete("birthdate")
		}
	} else if c.POST.Has("bday1") && c.POST.Has("bday2") && c.POST.Has("bday3") &&
		c.POST.Int("bday1") > 0 && c.POST.Int("bday2") > 0 {
		year := c.POST.Int("bday3")
		if year < 4 {
			year = 4
		}
		if checkdate(c.POST.Int("bday1"), c.POST.Int("bday2"), year) {
			c.POST.Set("birthdate", fmt.Sprintf("%04d-%02d-%02d", year, c.POST.Int("bday1"), c.POST.Int("bday2")))
		} else {
			c.POST.Set("birthdate", "0001-01-01")
		}
	} else if c.POST.Has("bday1") || c.POST.Has("bday2") || c.POST.Has("bday3") {
		c.POST.Set("birthdate", "0001-01-01")
	}

	if c.POST.Has("im_email_notify") {
		c.POST.Set("pm_email_notify", c.POST.Str("im_email_notify"))
	}

	// Validate and set the ignorelist (and buddy list): turn names into IDs.
	resolveNames := func(list string) string {
		list = strings.NewReplacer("\n", "', '", "\r", "", "&quot;", "").Replace(Htmltrim(list))
		if strings.TrimSpace(list) == "" {
			return ""
		}
		var quoted []string
		for _, name := range strings.Split(list, "', '") {
			quoted = append(quoted, "'"+strings.ReplaceAll(name, "'", "''")+"'")
		}
		in := strings.Join(quoted, ", ")
		var ids []string
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}members
			WHERE memberName IN (` + in + `) OR realName IN (` + in + `)`))
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				ids = append(ids, itoa(id))
			}
			rows.Close()
		}
		return strings.Join(ids, ",")
	}

	if c.POST.Has("pm_ignore_list") || c.POST.Has("im_ignore_list") {
		if !c.POST.Has("pm_ignore_list") {
			c.POST.Set("pm_ignore_list", c.POST.Str("im_ignore_list"))
		}
		raw := c.POST.Str("pm_ignore_list")
		if regexp.MustCompile(`(\A|,)\*(\z|,)`).MatchString(strings.NewReplacer("\n", ",", "\r", "").Replace(Htmltrim(raw))) {
			c.POST.Set("pm_ignore_list", "*")
		} else {
			c.POST.Set("pm_ignore_list", resolveNames(raw))
		}
	}

	// Similarly, do the same for the buddy list
	if c.POST.Has("buddy_list") {
		c.POST.Set("buddy_list", resolveNames(c.POST.Str("buddy_list")))
	}

	// Validate the smiley set.
	if c.POST.Has("smileySet") {
		valid := false
		for _, set := range strings.Split(a.Setting("smiley_sets_known"), ",") {
			if c.POST.Str("smileySet") == set {
				valid = true
			}
		}
		if !valid && c.POST.Str("smileySet") != "none" {
			c.POST.Set("smileySet", "")
		}
	}

	// Make sure the signature isn't too long.
	if c.POST.Has("signature") {
		sig := c.POST.Str("signature")
		if !a.SettingEmpty("max_signatureLength") && entityLen(sig) > a.SettingInt("max_signatureLength") {
			sig = entitySubstr(sig, 0, a.SettingInt("max_signatureLength"))
		}
		sig = strings.NewReplacer("&quot;", "\\&quot;", "&#039;", "\\&#39;", "&#39;", "\\&#39;").Replace(sig)
		sig = c.preparsecode(sig, false)
		c.POST.Set("signature", sig)
	}

	// Identity-only changes...
	if changeIdentity {
		// This block is only concerned with display name validation.
		if c.POST.Has("realName") && (!a.SettingEmpty("allow_editDisplayName") || c.allowedTo("moderate_forum")) &&
			strings.TrimSpace(c.POST.Str("realName")) != oldProfile.RealName {
			name := strings.TrimSpace(latin1NameClean(c.POST.Str("realName")))
			if name == "" {
				st.errors = append(st.errors, "no_name")
			} else if entityLen(name) > 60 {
				st.errors = append(st.errors, "name_too_long")
			} else if c.isReservedName(name, memID, true, false) {
				st.errors = append(st.errors, "name_taken")
			}
			st.vars["realName"] = name
		}

		// Change the registration date. (admin only; strtotime parsing is
		// limited to YYYY-MM-DD here.)
		if !empty(c.POST.Str("dateRegistered")) && c.allowedTo("admin_forum") {
			t, err := time.ParseInLocation("2006-01-02", c.POST.Str("dateRegistered"), time.Local)
			if err != nil {
				c.fatalError(c.Txt("smf233")+" "+c.strftimeLocal("%d %b %Y %H:%M:%S", c.forumTime(false, 0)), false)
			}
			newDate := t.Unix() - int64((c.User.TimeOffset+float64(a.SettingInt("time_offset")))*3600)
			oldDate := oldProfile.DateRegistered
			if t.Unix() != oldDate {
				st.vars["dateRegistered"] = newDate
			}
		}

		// Change the number of posts.
		if c.POST.Has("posts") && c.allowedTo("moderate_forum") {
			cleaned := strings.NewReplacer(",", "", ".", "", " ", "").Replace(c.POST.Str("posts"))
			st.vars["posts"] = atoi(cleaned)
		}

		// This block is only concerned with email address validation..
		if c.POST.Has("emailAddress") && strings.ToLower(c.POST.Str("emailAddress")) != strings.ToLower(oldProfile.EmailAddress) {
			email := c.POST.Str("emailAddress")

			// Prepare the new password, or check if they want to change
			// their own.
			if !a.SettingEmpty("send_validation_onChange") && !c.allowedTo("moderate_forum") {
				st.validationCode = generateValidationCode()
				st.vars["validation_code"] = st.validationCode
				st.vars["is_activated"] = 2
				st.newPassEmail = true
			}

			// Check the name and email for validity.
			if strings.TrimSpace(email) == "" {
				st.errors = append(st.errors, "no_email")
			}
			if !emailRe.MatchString(email) {
				st.errors = append(st.errors, "bad_email")
			}

			// Email addresses should be and stay unique.
			var dummy int
			if err := a.DB.QueryRow(a.Q(`
				SELECT ID_MEMBER
				FROM {$db_prefix}members
				WHERE ID_MEMBER != ?
					AND emailAddress = ?
				LIMIT 1`), memID, email).Scan(&dummy); err == nil {
				st.errors = append(st.errors, "email_taken")
			}

			st.vars["emailAddress"] = email
		}

		// Hide email address?
		if c.POST.Has("hideEmail") && (!a.SettingEmpty("allow_hideEmail") || c.allowedTo("moderate_forum")) {
			st.vars["hideEmail"] = boolToInt(!empty(c.POST.Str("hideEmail")))
		}

		// Are they allowed to change their hide status?
		if c.POST.Has("showOnline") && (!a.SettingEmpty("allow_hideOnline") || c.allowedTo("moderate_forum")) {
			st.vars["showOnline"] = boolToInt(!empty(c.POST.Str("showOnline")))
		}

		// If they're trying to change the password, let's check they pick a
		// sensible one.
		if pass1 := c.POST.Str("passwrd1"); c.POST.Has("passwrd1") && pass1 != "" {
			// Do the two entries for the password even match?
			if pass1 != c.POST.Str("passwrd2") {
				st.errors = append(st.errors, "bad_new_password")
			}

			// Let's get the validation function into play...
			if passErr := c.validatePassword(unHtmlspecialchars(pass1), c.User.Username, []string{c.User.Name, c.User.Email}); passErr != "" {
				st.errors = append(st.errors, "password_"+passErr)
			}

			// Set up the new password variable... ready for storage.
			st.vars["passwd"] = fmt.Sprintf("%x", sha1.Sum([]byte(strings.ToLower(oldProfile.MemberName)+unHtmlspecialchars(pass1))))
		}

		if c.POST.Has("secretQuestion") {
			st.vars["secretQuestion"] = c.POST.Str("secretQuestion")
		}

		// Do you have a *secret* password?
		if c.POST.Has("secretAnswer") && c.POST.Str("secretAnswer") != "" {
			st.vars["secretAnswer"] = fmt.Sprintf("%x", md5.Sum([]byte(c.POST.Str("secretAnswer"))))
		}
	}

	// Things they can do if they are a forum moderator.
	if c.allowedTo("moderate_forum") {
		if (c.REQUEST.Str("sa") == "activateAccount" || !empty(c.POST.Str("is_activated"))) && oldProfile.IsActivated != 1 {
			// If we are approving the deletion of an account, we do
			// something special ;)
			if oldProfile.IsActivated == 4 {
				c.deleteMembers([]int{memID})
				c.redirectExit("")
			}

			// Actually update this member now.
			newActivated := 1
			if oldProfile.IsActivated >= 10 {
				newActivated = 11
			}
			a.updateMemberDataMap(memID, map[string]any{"is_activated": newActivated, "validation_code": ""})

			// If we are doing approval, update the stats for the member just
			// incase.
			switch oldProfile.IsActivated {
			case 3, 4, 13, 14:
				n := a.SettingInt("unapprovedMembers") - 1
				if n < 0 {
					n = 0
				}
				a.UpdateSettings(map[string]string{"unapprovedMembers": itoa(n)})
			}

			// Make sure we update the stats too.
			c.updateStatsMemberRecount()
		}

		if c.POST.Has("karmaGood") {
			st.vars["karmaGood"] = c.POST.Int("karmaGood")
		}
		if c.POST.Has("karmaBad") {
			st.vars["karmaBad"] = c.POST.Int("karmaBad")
		}
	}

	// Assigning membergroups (you need admin_forum permissions to change an
	// admins' membergroups).
	if c.allowedTo("manage_membergroups") {
		// The account page allows the change of your ID_GROUP - but not to
		// admin!
		if c.POST.Has("ID_GROUP") && (c.allowedTo("admin_forum") || (c.POST.Int("ID_GROUP") != 1 && oldProfile.IDGroup != 1)) {
			st.vars["ID_GROUP"] = c.POST.Int("ID_GROUP")
		}

		// Find the additional membergroups (if any)
		var newAdditional []int
		hasAdditional := false
		if arr := c.POST.Arr("additionalGroups"); arr != nil {
			hasAdditional = true
			for _, k := range arr.Keys() {
				groupID := atoi(arr.Str(k))
				if groupID == 0 || (!c.allowedTo("admin_forum") && groupID == 1) {
					continue
				}
				newAdditional = append(newAdditional, groupID)
			}

			// Put admin back in there if you don't have permission to take
			// it away.
			wasAdmin := false
			for _, g := range strings.Split(oldProfile.AdditionalGroups, ",") {
				if g == "1" {
					wasAdmin = true
				}
			}
			if !c.allowedTo("admin_forum") && wasAdmin {
				newAdditional = append(newAdditional, 1)
			}

			ids := make([]string, len(newAdditional))
			for i, g := range newAdditional {
				ids[i] = itoa(g)
			}
			st.vars["additionalGroups"] = strings.Join(ids, ",")
		}

		// Too often, people remove delete their own account, or something.
		wasAdmin := oldProfile.IDGroup == 1
		for _, g := range strings.Split(oldProfile.AdditionalGroups, ",") {
			if g == "1" {
				wasAdmin = true
			}
		}
		if wasAdmin {
			stillAdmin := true
			if newGroup, ok := st.vars["ID_GROUP"]; ok && newGroup != 1 {
				stillAdmin = false
			}
			if hasAdditional {
				inAdditional := false
				for _, g := range newAdditional {
					if g == 1 {
						inAdditional = true
					}
				}
				if !stillAdmin && inAdditional {
					stillAdmin = true
				}
			}

			// If they would no longer be an admin, look for any other...
			if !stillAdmin {
				var another int
				a.DB.QueryRow(a.Q(`
					SELECT ID_MEMBER
					FROM {$db_prefix}members
					WHERE (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups))
						AND ID_MEMBER != ?
					LIMIT 1`), memID).Scan(&another)

				if another == 0 {
					c.fatalLangError("at_least_one_admin", true)
				}
			}
		}
	}

	// (lngfile validation: single-language port, skipped.)

	// Here's where we sort out all the 'other' values...
	if changeOther {
		themeID := oldProfile.IDTheme
		if c.POST.Has("ID_THEME") {
			themeID = c.POST.Int("ID_THEME")
		}
		c.makeThemeChanges(memID, themeID)
		c.makeAvatarChanges(memID, st)
		c.makeNotificationChanges(memID)

		for _, v := range profileBools {
			if c.POST.Has(v) {
				st.vars[v] = boolToInt(!empty(c.POST.Str(v)))
			}
		}
		for _, v := range profileInts {
			if c.POST.Has(v) {
				st.vars[v] = c.POST.Int(v)
			}
		}
		for _, v := range profileFloats {
			if c.POST.Has(v) {
				st.vars[v] = toFloat(c.POST.Get(v))
			}
		}
		for _, v := range profileStrings {
			if c.POST.Has(v) {
				st.vars[v] = c.POST.Str(v)
			}
		}
	}

	if icq, ok := st.vars["ICQ"]; ok {
		if n, isInt := icq.(int); isInt && n == 0 {
			st.vars["ICQ"] = ""
		}
	}
}

// makeThemeChanges is makeThemeChanges($memID, $ID_THEME).
func (c *Ctx) makeThemeChanges(memID, themeID int) {
	a := c.App

	reservedVars := map[string]bool{
		"actual_theme_url": true, "actual_images_url": true, "base_theme_dir": true,
		"base_theme_url": true, "default_images_url": true, "default_theme_dir": true,
		"default_theme_url": true, "default_template": true, "images_url": true,
		"number_recent_posts": true, "smiley_sets_default": true, "theme_dir": true,
		"theme_id": true, "theme_layers": true, "theme_templates": true, "theme_url": true,
	}

	opts := c.POST.Arr("options")
	defaults := c.POST.Arr("default_options")

	// Can't change reserved vars.
	check := func(arr *PArray) {
		if arr == nil {
			return
		}
		for _, k := range arr.Keys() {
			if reservedVars[k] {
				c.fatalLangError("1", true)
			}
		}
	}
	check(opts)
	check(defaults)

	flat := func(v any) string {
		switch val := v.(type) {
		case string:
			return val
		case *PArray:
			var parts []string
			for _, k := range val.Keys() {
				if s, ok := val.Get(k).(string); ok {
					parts = append(parts, s)
				}
			}
			return strings.Join(parts, ",")
		}
		return ""
	}

	// These are the theme changes...
	if opts != nil {
		for _, opt := range opts.Keys() {
			a.DB.Exec(a.Q(`
				REPLACE INTO {$db_prefix}themes
					(ID_MEMBER, ID_THEME, variable, value)
				VALUES (?, ?, SUBSTR(?, 1, 255), SUBSTR(?, 1, 65534))`),
				memID, themeID, opt, flat(opts.Get(opt)))
		}
	}

	var eraseOptions []string
	if defaults != nil {
		for _, opt := range defaults.Keys() {
			a.DB.Exec(a.Q(`
				REPLACE INTO {$db_prefix}themes
					(ID_MEMBER, ID_THEME, variable, value)
				VALUES (?, 1, SUBSTR(?, 1, 255), SUBSTR(?, 1, 65534))`),
				memID, opt, flat(defaults.Get(opt)))
			eraseOptions = append(eraseOptions, "'"+strings.ReplaceAll(opt, "'", "''")+"'")
		}
	}

	if len(eraseOptions) > 0 {
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}themes
			WHERE ID_THEME != 1
				AND variable IN (` + strings.Join(eraseOptions, ", ") + `)
				AND ID_MEMBER = ` + itoa(memID)))
	}
}

// makeNotificationChanges is makeNotificationChanges($memID).
func (c *Ctx) makeNotificationChanges(memID int) {
	a := c.App

	collectInts := func(arr *PArray) []string {
		var out []string
		if arr == nil {
			return out
		}
		for _, k := range arr.Keys() {
			id := atoi(arr.Str(k))
			if id != 0 {
				out = append(out, itoa(id))
			}
		}
		return out
	}

	// Update the boards they are being notified on.
	if c.POST.Has("edit_notify_boards") && c.POST.Arr("notify_boards") != nil && c.POST.Arr("notify_boards").Len() > 0 {
		boards := collectInts(c.POST.Arr("notify_boards"))
		if len(boards) > 0 {
			a.DB.Exec(a.Q(`
				DELETE FROM {$db_prefix}log_notify
				WHERE ID_BOARD IN (` + strings.Join(boards, ", ") + `)
					AND ID_MEMBER = ` + itoa(memID)))
		}
	} else if c.POST.Has("edit_notify_topics") && c.POST.Arr("notify_topics") != nil && c.POST.Arr("notify_topics").Len() > 0 {
		// We are editing topic notifications......
		topics := collectInts(c.POST.Arr("notify_topics"))
		if len(topics) > 0 {
			a.DB.Exec(a.Q(`
				DELETE FROM {$db_prefix}log_notify
				WHERE ID_TOPIC IN (` + strings.Join(topics, ", ") + `)
					AND ID_MEMBER = ` + itoa(memID)))
		}
	}
}

var avatarActionRe = regexp.MustCompile(`(?i)action(=|%3d)`)
var avatarServerRe = regexp.MustCompile(`^([\w _!@%*=\-#()\[\]&.,]+/)?[\w _!@%*=\-#()\[\]&.,]+$`)

// makeAvatarChanges is makeAvatarChanges($memID, &$post_errors).
func (c *Ctx) makeAvatarChanges(memID int, st *profileSaveState) {
	a := c.App

	if !c.POST.Has("avatar_choice") || memID == 0 {
		return
	}

	uploadDir := a.Setting("attachmentUploadDir")
	if !a.SettingEmpty("custom_avatar_enabled") {
		uploadDir = a.Setting("custom_avatar_dir")
	}

	choice := c.POST.Str("avatar_choice")
	personal := c.POST.Str("userpicpersonal")

	// (avatar_download_external fetches the remote image server-side; that
	// branch is dropped — external avatars stay external links.)

	if choice == "server_stored" && c.allowedTo("profile_server_avatar") {
		avatar := c.POST.Str("file")
		if avatar == "" {
			avatar = c.POST.Str("cat")
		}
		avatar = strings.ReplaceAll(avatar, "&amp;", "&")
		if avatarServerRe.MatchString(avatar) && !strings.Contains(avatar, "..") &&
			fileExists(a.Setting("avatar_directory")+"/"+avatar) {
			if avatar == "blank.gif" {
				avatar = ""
			}
		} else {
			avatar = ""
		}
		c.POST.Set("avatar", avatar)

		// Get rid of their old avatar. (if uploaded.)
		a.removeAttachments("a.ID_MEMBER = "+itoa(memID), "", false, true)
	} else if choice == "external" && c.allowedTo("profile_remote_avatar") &&
		strings.HasPrefix(strings.ToLower(personal), "http://") {
		// Remove any attached avatar...
		a.removeAttachments("a.ID_MEMBER = "+itoa(memID), "", false, true)

		avatar := replaceAvatarAction(personal)

		if avatar == "http://" || avatar == "http:///" {
			avatar = ""
		} else if !strings.HasPrefix(avatar, "http://") {
			// Trying to make us do something we'll regret?
			st.errors = append(st.errors, "bad_avatar")
		} else if !a.SettingEmpty("avatar_max_height_external") || !a.SettingEmpty("avatar_max_width_external") {
			// Should we check dimensions? Validate the avatar's real size.
			maxW := a.SettingInt("avatar_max_width_external")
			maxH := a.SettingInt("avatar_max_height_external")
			w, h, ok := urlImageSize(avatar)
			if ok && ((w > maxW && maxW != 0) || (h > maxH && maxH != 0)) {
				// Houston, we have a problem. The avatar is too large!!
				// PHP's option_download_and_resize fetches + shrinks the image
				// server-side via downloadAvatar(); that external-download
				// branch is dropped in this port (single-binary, no GD-backed
				// resize of remote images), so both too-large actions collapse
				// to refusal — the same outcome PHP yields when the download
				// fails.
				st.errors = append(st.errors, "bad_avatar")
			}
		}
		c.POST.Set("avatar", avatar)
	} else if choice == "upload" && c.allowedTo("profile_upload_avatar") {
		var fileData []byte
		var ok bool
		if c.R.MultipartForm != nil {
			if fhs := c.R.MultipartForm.File["attachment"]; len(fhs) > 0 && fhs[0].Filename != "" {
				if f, err := fhs[0].Open(); err == nil {
					fileData, _ = io.ReadAll(f)
					f.Close()
					ok = true
				}
			}
		}
		if ok {
			os.MkdirAll(uploadDir, 0755)
			tmpPath := uploadDir + "/avatar_tmp_" + itoa(memID)
			if err := os.WriteFile(tmpPath, fileData, 0644); err != nil {
				c.fatalLangError("attachments_no_write", true)
			}

			width, height, isImage := imageSize(tmpPath)

			// No size, then it's probably not a valid pic.
			if !isImage {
				st.errors = append(st.errors, "bad_avatar")
			} else if (!a.SettingEmpty("avatar_max_width_upload") && width > a.SettingInt("avatar_max_width_upload")) ||
				(!a.SettingEmpty("avatar_max_height_upload") && height > a.SettingInt("avatar_max_height_upload")) {
				// Check whether the image is too large.
				st.errors = append(st.errors, "bad_avatar")
			} else {
				// Though not an exhaustive list, better safe than sorry.
				if regexp.MustCompile(`(?i)(iframe|<\?|<%|html|eval|body|script\W|[CF]WS[\x01-\x0C])`).Match(fileData) {
					os.Remove(tmpPath)
					c.fatalLangError("smf124", true)
				}

				ext := ".bmp"
				switch imageContentType(fileData) {
				case "image/gif":
					ext = ".gif"
				case "image/jpeg":
					ext = ".jpg"
				case "image/png":
					ext = ".png"
				}

				destName := "avatar_" + itoa(memID) + ext

				// Remove previous attachments this member might have had.
				a.removeAttachments("a.ID_MEMBER = "+itoa(memID), "", false, true)

				fileHash := ""
				attachType := 1
				if a.SettingEmpty("custom_avatar_enabled") {
					fileHash = a.getAttachmentFilename(destName, 0, true, "")
					attachType = 0
				}

				res, err := a.DB.Exec(a.Q(`
					INSERT INTO {$db_prefix}attachments
						(ID_MEMBER, attachmentType, filename, file_hash, size, width, height)
					VALUES (?, ?, ?, ?, ?, ?, ?)`),
					memID, attachType, destName, fileHash, len(fileData), width, height)
				if err == nil {
					id, _ := res.LastInsertId()
					attachID := int(id)

					// Try to move this avatar.
					destinationPath := uploadDir + "/" + destName
					if fileHash != "" {
						destinationPath = uploadDir + "/" + itoa(attachID) + "_" + fileHash
					}
					if err := os.Rename(tmpPath, destinationPath); err != nil {
						a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}attachments WHERE ID_ATTACH = ?`), attachID)
						c.fatalLangError("smf124", true)
					}
					os.Chmod(destinationPath, 0644)
				}
			}
			c.POST.Set("avatar", "")

			// Delete any temporary file.
			os.Remove(tmpPath)
		} else {
			// Selected the upload avatar option and had one already uploaded
			// before or didn't upload one.
			c.POST.Set("avatar", "")
		}
	} else {
		c.POST.Set("avatar", "")
	}
}

// replaceAvatarAction is preg_replace('~action(=|%3d)(?!dlattach)~i',
// 'action-', ...) with the lookahead emulated.
func replaceAvatarAction(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		loc := avatarActionRe.FindStringIndex(s[i:])
		if loc == nil {
			b.WriteString(s[i:])
			break
		}
		start, end := i+loc[0], i+loc[1]
		b.WriteString(s[i:start])
		rest := s[end:]
		if len(rest) >= 8 && strings.EqualFold(rest[:8], "dlattach") {
			b.WriteString(s[start:end])
		} else {
			b.WriteString("action-")
		}
		i = end
	}
	return b.String()
}

// deleteAccount2 is deleteAccount2($profile_vars, $post_errors, $memID).
func (c *Ctx) deleteAccount2(st *profileSaveState) {
	a := c.App
	memID := st.memID

	if !st.isOwner {
		c.isAllowedTo("profile_remove_any")
	} else if !c.allowedTo("profile_remove_any") {
		c.isAllowedTo("profile_remove_own")
	}

	c.checkSession("post", "", true)

	oldProfile := c.memberProfiles[memID]

	// Too often, people remove/delete their own only account.
	wasAdmin := oldProfile.IDGroup == 1
	for _, g := range strings.Split(oldProfile.AdditionalGroups, ",") {
		if g == "1" {
			wasAdmin = true
		}
	}
	if wasAdmin {
		// Are you allowed to administrate the forum, as they are?
		c.isAllowedTo("admin_forum")

		var another int
		a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}members
			WHERE (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups))
				AND ID_MEMBER != ?
			LIMIT 1`), memID).Scan(&another)

		if another == 0 {
			c.fatalLangError("at_least_one_admin", true)
		}
	}

	// Do you have permission to delete others profiles, or is that your
	// profile you wanna delete?
	if memID != c.User.ID {
		c.isAllowedTo("profile_remove_any")

		// Now, have you been naughty and need your posts deleting?
		if c.POST.Str("remove_type") != "none" && c.allowedTo("moderate_forum") {
			// First off we delete any topics the member has started - if
			// they wanted topics being done.
			if c.POST.Str("remove_type") == "topics" {
				// Fetch all topics started by this user.
				var topicIDs []int
				rows, err := a.DB.Query(a.Q(`
					SELECT t.ID_TOPIC
					FROM {$db_prefix}topics AS t
					WHERE t.ID_MEMBER_STARTED = ?`), memID)
				if err == nil {
					for rows.Next() {
						var t int
						rows.Scan(&t)
						topicIDs = append(topicIDs, t)
					}
					rows.Close()
				}

				// Actually remove the topics.
				c.removeTopics(topicIDs, true, false)
			}

			// Now delete the remaining messages.
			var msgs []int
			rows, err := a.DB.Query(a.Q(`
				SELECT m.ID_MSG
				FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
				WHERE m.ID_MEMBER = ?
					AND m.ID_TOPIC = t.ID_TOPIC
					AND t.ID_FIRST_MSG != m.ID_MSG`), memID)
			if err == nil {
				for rows.Next() {
					var m int
					rows.Scan(&m)
					msgs = append(msgs, m)
				}
				rows.Close()
			}
			// This could take a while... but ya know it's gonna be worth it
			// in the end.
			for _, m := range msgs {
				c.removeMessage(m, true)
			}
		}

		// Only delete this poor members account if they are actually being
		// booted out of camp.
		if c.POST.Has("deleteAccount") {
			c.deleteMembers([]int{memID})
		}
	} else if len(st.errors) == 0 && !a.SettingEmpty("approveAccountDeletion") && !c.allowedTo("moderate_forum") {
		// Do they need approval to delete? Setup their account for deletion
		// ;)
		a.updateMemberDataMap(memID, map[string]any{"is_activated": 4})
		// Another account needs approval...
		a.UpdateSettings(map[string]string{"unapprovedMembers": itoa(a.SettingInt("unapprovedMembers") + 1)})
	} else if len(st.errors) == 0 {
		// Also check if you typed your password correctly.
		c.deleteMembers([]int{memID})
	}
}
