package app

// Port of Sources/Subs-Members.php (Phase 2 scope: registerMember,
// isReservedName, generateValidationCode + validatePassword/isBannedEmail
// from Subs-Auth.php/Security.php; deleteMembers and BuddyListToggle port
// with Phase 4).

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// regOptions mirrors the $regOptions array of registerMember().
type regOptions struct {
	Interface             string // 'guest' or 'admin'
	Username              string
	Email                 string
	Password              string
	PasswordCheck         string
	CheckReservedName     bool
	CheckPasswordStrength bool
	CheckEmailBan         bool
	SendWelcomeEmail      bool
	Require               string            // 'nothing', 'activation', 'approval', 'coppa'
	ExtraRegisterVars     map[string]string // column -> raw value
	ThemeVars             map[string]string
	MemberGroup           int
	HasMemberGroup        bool

	// Output:
	PasswordSalt string
}

var emailRe = regexp.MustCompile(`^[0-9A-Za-z=_+\-/][0-9A-Za-z=_'+\-/\.]*@[\w\-]+(\.[\w\-]+)*(\.[\w]{2,6})$`)
var usernameStripRe = regexp.MustCompile(`[\t\n\r \x0B\x00\xA0]+`)

// generateValidationCode is generateValidationCode().
func generateValidationCode() string {
	var b [20]byte
	rand.Read(b[:])
	code := fmt.Sprintf("%x", sha1.Sum(b[:]))
	return code[:10]
}

// validatePassword is validatePassword() from Subs-Auth.php: returns the
// error suffix ('short', 'restricted_words', 'chars') or "".
func (c *Ctx) validatePassword(password, username string, restrictIn []string) string {
	a := c.App

	minLen := 4
	if !a.SettingEmpty("password_strength") {
		minLen = 8
	}
	if len(password) < minLen {
		return "short"
	}

	// Is this enough?
	if a.SettingEmpty("password_strength") {
		return ""
	}

	// Medium strength test.
	joined := strings.Join(restrictIn, " ")
	if re, err := regexp.Compile(`\b` + regexp.QuoteMeta(password) + `\b`); err == nil && re.MatchString(joined) {
		return "restricted_words"
	}
	if strings.Contains(password, username) {
		return "restricted_words"
	}

	// If just medium, we're done.
	if a.Setting("password_strength") == "1" {
		return ""
	}

	// Otherwise, hard test next: numbers and letters, uppercase too.
	good := regexp.MustCompile(`(\D\d|\d\D)`).MatchString(password)
	good = good && strings.ToLower(password) != password
	if !good {
		return "chars"
	}
	return ""
}

// isBannedEmail is isBannedEmail() from Security.php.
func (c *Ctx) isBannedEmail(email, restriction, errMsg string) {
	a := c.App
	if strings.TrimSpace(email) == "" {
		return
	}

	banReason := ""
	banned := false
	cannotAccess := false
	cannotAccessReason := ""

	rows, err := a.DB.Query(a.Q(`
		SELECT bi.ID_BAN, bg.`+restriction+`, bg.cannot_access, bg.reason
		FROM {$db_prefix}ban_items AS bi, {$db_prefix}ban_groups AS bg
		WHERE bg.ID_BAN_GROUP = bi.ID_BAN_GROUP
			AND ? LIKE bi.email_address
			AND (bg.`+restriction+` = 1 OR bg.cannot_access = 1)`), email)
	if err == nil {
		for rows.Next() {
			var idBan, restricted, access int
			var reason string
			rows.Scan(&idBan, &restricted, &access, &reason)
			if access != 0 {
				cannotAccess = true
				cannotAccessReason = reason
			}
			if restricted != 0 {
				banned = true
				banReason = reason
			}
		}
		rows.Close()
	}

	// You're in biiig trouble.  Banned for the rest of this session!
	if cannotAccess {
		c.logBanned()
		c.fatalError(phpSprintf(c.Txt("430"), c.Txt("28"))+cannotAccessReason, false)
	}

	if banned {
		c.logBanned()
		c.fatalError(errMsg+banReason, false)
	}
}

// isReservedName is isReservedName($name, $current_ID_MEMBER, $is_name,
// $fatal).
func (c *Ctx) isReservedName(name string, currentIDMember int, isName bool, fatal bool) bool {
	a := c.App

	checkName := strings.ToLower(name)

	// Administrators are never restricted ;).
	if !c.allowedTo("moderate_forum") &&
		((!a.SettingEmpty("reserveName") && isName) || (!a.SettingEmpty("reserveUser") && !isName)) {
		for _, reserved := range strings.Split(a.Setting("reserveNames"), "\n") {
			if reserved == "" {
				continue
			}
			reservedCheck := reserved
			checkMe := name
			// Case sensitive check?
			if a.SettingEmpty("reserveCase") {
				reservedCheck = strings.ToLower(reservedCheck)
				checkMe = checkName
			}

			// If it's not just entire word, check for it in there somewhere...
			if checkMe == reservedCheck ||
				(strings.Contains(checkMe, reservedCheck) && a.SettingEmpty("reserveWord")) {
				if fatal {
					c.fatalLangError("244", true, reserved)
				}
				return true
			}
		}

		if c.censorText(name) != name {
			if fatal {
				c.fatalLangError("name_censored", true, name)
			}
			return true
		}
	}

	// Make sure they don't want someone else's name.
	cond := ""
	if currentIDMember != 0 {
		cond = "ID_MEMBER != " + itoa(currentIDMember) + " AND "
	}
	var dummy int
	err := a.DB.QueryRow(a.Q(`
		SELECT ID_MEMBER
		FROM {$db_prefix}members
		WHERE `+cond+`(realName LIKE ? OR memberName LIKE ?)
		LIMIT 1`), name, name).Scan(&dummy)
	if err == nil {
		return true
	}

	// Does name case insensitive match a member group name?
	err = a.DB.QueryRow(a.Q(`
		SELECT ID_GROUP
		FROM {$db_prefix}membergroups
		WHERE groupName LIKE ?
		LIMIT 1`), name).Scan(&dummy)
	return err == nil
}

// registerMember is registerMember(&$regOptions): returns the new member ID.
func (c *Ctx) registerMember(opts *regOptions) int {
	a := c.App

	c.loadLanguage("Login")

	// Registration from the admin center, let them sweat a little more.
	if opts.Interface == "admin" {
		c.isNotGuest("")
		c.isAllowedTo("moderate_forum")
	} else if opts.Interface == "guest" {
		c.spamProtection("register")

		// You cannot register twice...
		if !c.User.IsGuest {
			c.redirectExit("")
		}

		// Make sure they didn't just register with this session.
		if c.Session.GetInt("just_registered") != 0 && a.SettingEmpty("disableRegisterCheck") {
			c.fatalLangError("register_only_once", false)
		}
	}

	// No name?!  How can you register with no name?
	if opts.Username == "" {
		c.fatalLangError("37", false)
	}

	// Spaces and other odd characters are evil...
	opts.Username = strings.TrimSpace(usernameStripRe.ReplaceAllString(opts.Username, " "))

	// Don't use too long a name.
	if entityLen(opts.Username) > 25 {
		opts.Username = Htmltrim(entitySubstr(opts.Username, 0, 25))
	}

	// Only these characters are permitted.
	if funkyUserRe.MatchString(opts.Username) || opts.Username == "_" || opts.Username == "|" ||
		strings.Contains(opts.Username, "[code") || strings.Contains(opts.Username, "[/code") {
		c.fatalLangError("240", false)
	}

	if containsCI(opts.Username, c.Txt("28")) {
		c.fatalLangError("244", true, c.Txt("28"))
	}

	if opts.Email == "" || !emailRe.MatchString(opts.Email) || len(opts.Email) > 255 {
		c.fatalError(phpSprintf(c.Txt("500"), opts.Username), false)
	}

	if opts.CheckReservedName && c.isReservedName(opts.Username, 0, false, true) {
		if opts.Password == "chocolate cake" {
			c.fatalError("Sorry, I don't take bribes... you'll need to come up with a different name.", false)
		}
		c.fatalError("("+Htmlspecialchars(opts.Username)+") "+c.Txt("473"), false)
	}

	// Generate a validation code if it's supposed to be emailed.
	validationCode := ""
	if opts.Require == "activation" {
		validationCode = generateValidationCode()
	}

	// If you haven't put in a password generated one.
	if opts.Interface == "admin" && opts.Password == "" {
		opts.Password = generateValidationCode()
		opts.PasswordCheck = opts.Password
	} else if opts.Password != opts.PasswordCheck {
		// Does the first password match the second?
		c.fatalLangError("213", false)
	}

	// That's kind of easy to guess...
	if opts.Password == "" {
		c.fatalLangError("91", false)
	}

	// Now perform hard password validation as required.
	if opts.CheckPasswordStrength {
		if pwErr := c.validatePassword(opts.Password, opts.Username, []string{opts.Email}); pwErr != "" {
			c.fatalLangError("profile_error_password_"+pwErr, false)
		}
	}

	// You may not be allowed to register this email.
	if opts.CheckEmailBan {
		c.isBannedEmail(opts.Email, "cannot_register", c.Txt("ban_register_prohibited"))
	}

	// Check if the email address is in use.
	var dummy int
	err := a.DB.QueryRow(a.Q(`
		SELECT ID_MEMBER
		FROM {$db_prefix}members
		WHERE emailAddress = ?
			OR emailAddress = ?
		LIMIT 1`), opts.Email, opts.Username).Scan(&dummy)
	if err == nil {
		c.fatalError(phpSprintf(c.Txt("730"), Htmlspecialchars(opts.Email)), false)
	}

	opts.PasswordSalt = randomSalt()

	// Some of these might be overwritten.
	registerVars := map[string]string{
		"memberName":      opts.Username,
		"emailAddress":    opts.Email,
		"passwd":          sha1hex(strings.ToLower(opts.Username) + opts.Password),
		"passwordSalt":    opts.PasswordSalt,
		"posts":           "0",
		"dateRegistered":  itoa(int(time.Now().Unix())),
		"memberIP":        c.User.IP,
		"memberIP2":       c.BanCheckIP,
		"validation_code": validationCode,
		"realName":        opts.Username,
		"pm_email_notify": "1",
		"ID_THEME":        "0",
		"ID_POST_GROUP":   "4",
	}

	// Setup the activation status on this new account.
	switch opts.Require {
	case "coppa":
		registerVars["is_activated"] = "5"
		registerVars["validation_code"] = ""
	case "nothing":
		registerVars["is_activated"] = "1"
	case "activation":
		registerVars["is_activated"] = "0"
	default:
		registerVars["is_activated"] = "3"
	}

	if opts.HasMemberGroup {
		// Make sure the ID_GROUP will be valid, if this is an administator.
		group := opts.MemberGroup
		if group == 1 && !c.allowedTo("admin_forum") {
			group = 0
		}
		// Check if this group is assignable.
		unassignable := map[int]bool{-1: true, 3: true}
		grows, err := a.DB.Query(a.Q(`SELECT ID_GROUP FROM {$db_prefix}membergroups WHERE minPosts != -1`))
		if err == nil {
			for grows.Next() {
				var g int
				grows.Scan(&g)
				unassignable[g] = true
			}
			grows.Close()
		}
		if unassignable[group] {
			group = 0
		}
		registerVars["ID_GROUP"] = itoa(group)
	}

	// Integrate optional member settings to be set.
	for k, v := range opts.ExtraRegisterVars {
		registerVars[k] = v
	}

	// Register them into the database.
	var cols, marks []string
	var vals []any
	for k, v := range registerVars {
		cols = append(cols, k)
		marks = append(marks, "?")
		vals = append(vals, v)
	}
	res, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}members (`+strings.Join(cols, ", ")+`) VALUES (`+strings.Join(marks, ", ")+`)`), vals...)
	if err != nil {
		c.fatalDBError(err)
	}
	id64, _ := res.LastInsertId()
	memberID := int(id64)

	realName := opts.Username

	// Update the number of members and latest member's info.
	c.updateStatsMember(memberID, realName)

	// Theme variables too?
	for k, v := range opts.ThemeVars {
		a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}themes (ID_MEMBER, variable, value) VALUES (?, SUBSTR(?, 1, 255), SUBSTR(?, 1, 65534))`),
			memberID, k, v)
	}

	// If it's enabled, increase the registrations for today.
	c.trackStatsRegister()

	scripturl := a.ScriptURL

	// Administrative registrations are a bit different...
	if opts.Interface == "admin" {
		var emailMessage string
		if opts.Require == "activation" {
			emailMessage = "register_activate_message"
		} else if opts.SendWelcomeEmail {
			emailMessage = "register_immediate_message"
		}
		if emailMessage != "" {
			c.sendmail([]string{opts.Email}, c.Txt("register_subject"),
				phpSprintf(c.Txt(emailMessage), realName, opts.Username, opts.Password, validationCode,
					scripturl+"?action=activate;u="+itoa(memberID)+";code="+validationCode), "")
		}
		return memberID
	}

	switch opts.Require {
	case "nothing":
		// Can post straight away - welcome them.
		if opts.SendWelcomeEmail {
			c.sendmail([]string{opts.Email}, c.Txt("register_subject"),
				phpSprintf(c.Txt("register_immediate_message"), realName, opts.Username, opts.Password), "")
		}
		c.adminNotify("standard", memberID, opts.Username)
	case "activation", "coppa":
		c.sendmail([]string{opts.Email}, c.Txt("register_subject"),
			phpSprintf(c.Txt("register_activate_message"), realName, opts.Username, opts.Password, validationCode,
				scripturl+"?action=activate;u="+itoa(memberID)+";code="+validationCode), "")
	default:
		// Must be awaiting approval.
		c.sendmail([]string{opts.Email}, c.Txt("register_subject"),
			phpSprintf(c.Txt("register_pending_message"), realName, opts.Username, opts.Password), "")
		c.adminNotify("approval", memberID, opts.Username)
	}

	// Okay, they're for sure registered...
	c.Session.Set("just_registered", 1)

	return memberID
}

// updateStatsMember is updateStats('member', id, name).
func (c *Ctx) updateStatsMember(memberID int, realName string) {
	a := c.App
	changes := map[string]string{
		"memberlist_updated": itoa(int(time.Now().Unix())),
	}

	if a.Setting("registration_method") == "2" {
		var total, latest int
		a.DB.QueryRow(a.Q(`SELECT COUNT(*), IFNULL(MAX(ID_MEMBER), 0) FROM {$db_prefix}members WHERE is_activated = 1`)).Scan(&total, &latest)
		var latestName string
		a.DB.QueryRow(a.Q(`SELECT realName FROM {$db_prefix}members WHERE ID_MEMBER = ? LIMIT 1`), latest).Scan(&latestName)
		var unapproved int
		a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE is_activated IN (3, 4)`)).Scan(&unapproved)
		changes["totalMembers"] = itoa(total)
		changes["latestMember"] = itoa(latest)
		changes["latestRealName"] = latestName
		changes["unapprovedMembers"] = itoa(unapproved)
	} else if memberID != 0 {
		changes["latestMember"] = itoa(memberID)
		changes["latestRealName"] = realName
		changes["totalMembers"] = itoa(a.SettingInt("totalMembers") + 1)
	} else {
		var total, latest int
		a.DB.QueryRow(a.Q(`SELECT COUNT(*), IFNULL(MAX(ID_MEMBER), 0) FROM {$db_prefix}members`)).Scan(&total, &latest)
		var latestName string
		a.DB.QueryRow(a.Q(`SELECT realName FROM {$db_prefix}members WHERE ID_MEMBER = ? LIMIT 1`), latest).Scan(&latestName)
		changes["totalMembers"] = itoa(total)
		changes["latestMember"] = itoa(latest)
		changes["latestRealName"] = latestName
	}

	a.UpdateSettings(changes)
}

// updateStatsMemberRecount is updateStats('member', false): full recount.
func (c *Ctx) updateStatsMemberRecount() {
	a := c.App
	changes := map[string]string{
		"memberlist_updated": itoa(int(time.Now().Unix())),
	}
	if a.Setting("registration_method") == "2" {
		var total, latest int
		a.DB.QueryRow(a.Q(`SELECT COUNT(*), IFNULL(MAX(ID_MEMBER), 0) FROM {$db_prefix}members WHERE is_activated = 1`)).Scan(&total, &latest)
		var latestName string
		a.DB.QueryRow(a.Q(`SELECT realName FROM {$db_prefix}members WHERE ID_MEMBER = ? LIMIT 1`), latest).Scan(&latestName)
		var unapproved int
		a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE is_activated IN (3, 4)`)).Scan(&unapproved)
		changes["totalMembers"] = itoa(total)
		changes["latestMember"] = itoa(latest)
		changes["latestRealName"] = latestName
		changes["unapprovedMembers"] = itoa(unapproved)
	}
	a.UpdateSettings(changes)
}

// trackStatsRegister bumps today's registers count (trackStats register +).
func (c *Ctx) trackStatsRegister() {
	a := c.App
	if a.SettingEmpty("trackStats") {
		return
	}
	date := time.Unix(c.forumTime(false, 0), 0).Format("2006-01-02")
	res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_activity SET registers = registers + 1 WHERE date = ?`), date)
	if res != nil {
		if n, _ := res.RowsAffected(); n == 0 {
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}log_activity (date, registers) VALUES (?, 1)`), date)
		}
	}
}

// groupsAllowedTo is groupsAllowedTo($permission, $board_id). boardID < 0
// means null (regular permissions).
func (c *Ctx) groupsAllowedTo(permission string, boardID int) (allowed, denied []int) {
	a := c.App

	// Admins are allowed to do anything.
	allowed = []int{1}

	if boardID < 0 {
		// Assume we're dealing with regular permissions (like
		// profile_view_own).
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_GROUP, addDeny
			FROM {$db_prefix}permissions
			WHERE permission = ?`), permission)
		if err == nil {
			for rows.Next() {
				var group, addDeny int
				rows.Scan(&group, &addDeny)
				if addDeny == 1 {
					allowed = append(allowed, group)
				} else {
					denied = append(denied, group)
				}
			}
			rows.Close()
		}
	} else {
		// Let's do the global/local board permissions. First get the
		// permission mode of the given board.
		permissionMode := 0
		if c.BoardInfo != nil && c.BoardInfo.ID == boardID {
			switch c.BoardInfo.PermissionMode {
			case "no_polls":
				permissionMode = 2
			case "reply_only":
				permissionMode = 3
			case "read_only":
				permissionMode = 4
			}
		} else if boardID != 0 {
			if err := a.DB.QueryRow(a.Q(`
				SELECT permission_mode
				FROM {$db_prefix}boards
				WHERE ID_BOARD = ?
				LIMIT 1`), boardID).Scan(&permissionMode); err != nil {
				c.fatalLangError("smf232", true)
			}
		}

		// Without permissions-by-board, you might need moderator
		// permission.
		moderatorOnly := false
		if boardID != 0 && a.SettingEmpty("permission_enable_by_board") &&
			(permission == "post_reply_own" || permission == "post_reply_any" ||
				permission == "post_new" || permission == "poll_post") {
			maxAllowableMode := 3
			if permission == "post_new" {
				maxAllowableMode = 2
			} else if permission == "poll_post" {
				maxAllowableMode = 0
			}
			if permissionMode > maxAllowableMode {
				moderatorOnly = true
			}
		}

		fromExtra := ""
		modCond := ""
		if moderatorOnly {
			fromExtra = ", {$db_prefix}board_permissions AS modperm"
			modCond = `
				AND modperm.ID_GROUP = bp.ID_GROUP
				AND modperm.ID_BOARD = 0
				AND modperm.permission = 'moderate_board'
				AND modperm.addDeny = 1`
		}
		boardCond := "= 0"
		if !a.SettingEmpty("permission_enable_by_board") && permissionMode != 0 && boardID != 0 {
			boardCond = "IN (0, " + itoa(boardID) + ")"
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT bp.ID_GROUP, bp.addDeny
			FROM {$db_prefix}board_permissions AS bp`+fromExtra+`
			WHERE bp.permission = ?`+modCond+`
				AND bp.ID_BOARD `+boardCond), permission)
		if err == nil {
			for rows.Next() {
				var group, addDeny int
				rows.Scan(&group, &addDeny)
				if addDeny == 1 {
					allowed = append(allowed, group)
				} else {
					denied = append(denied, group)
				}
			}
			rows.Close()
		}
	}

	// Denied is never allowed.
	deniedSet := map[int]bool{}
	for _, g := range denied {
		deniedSet[g] = true
	}
	var kept []int
	for _, g := range allowed {
		if !deniedSet[g] {
			kept = append(kept, g)
		}
	}
	return kept, denied
}

// membersAllowedTo is membersAllowedTo($permission, $board_id). boardID < 0
// means null.
func (c *Ctx) membersAllowedTo(permission string, boardID int) []int {
	a := c.App

	allowed, denied := c.groupsAllowedTo(permission, boardID)

	includeModerators := false
	excludeModerators := false
	filter3 := func(groups []int, found *bool) []int {
		var out []int
		for _, g := range groups {
			if g == 3 {
				*found = true
				continue
			}
			out = append(out, g)
		}
		return out
	}
	if boardID >= 0 {
		allowed = filter3(allowed, &includeModerators)
		denied = filter3(denied, &excludeModerators)
	}

	groupClause := func(groups []int) string {
		ids := make([]string, len(groups))
		var finds []string
		for i, g := range groups {
			ids[i] = itoa(g)
			finds = append(finds, "FIND_IN_SET("+itoa(g)+", mem.additionalGroups)")
		}
		cl := "ID_GROUP IN (" + strings.Join(ids, ", ") + ")"
		if len(finds) > 0 {
			cl += " OR " + strings.Join(finds, " OR ")
		}
		return cl
	}

	joinMods := ""
	if includeModerators || excludeModerators {
		joinMods = `
			LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_MEMBER = mem.ID_MEMBER AND ID_BOARD = ` + itoa(boardID) + `)`
	}
	includeClause := ""
	if includeModerators {
		includeClause = "mods.ID_MEMBER IS NOT NULL OR "
	}
	// (As in PHP, no denied clause at all when the denied list is empty -
	// even if moderators are excluded.)
	deniedClause := ""
	if len(denied) > 0 {
		excludeClause := ""
		if excludeModerators {
			excludeClause = "mods.ID_MEMBER IS NOT NULL OR "
		}
		deniedClause = `
			AND NOT (` + excludeClause + groupClause(denied) + `)`
	}

	var members []int
	rows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER
		FROM {$db_prefix}members AS mem` + joinMods + `
		WHERE (` + includeClause + groupClause(allowed) + `)` + deniedClause))
	if err == nil {
		for rows.Next() {
			var m int
			rows.Scan(&m)
			members = append(members, m)
		}
		rows.Close()
	}
	return members
}
