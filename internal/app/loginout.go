package app

// Port of Sources/LogInOut.php.

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/subtle"
	"fmt"
	"regexp"
	"strings"
	"time"

	"smf/internal/phpser"
)

func init() {
	registerAction("login", (*Ctx).Login)
	registerAction("login2", (*Ctx).Login2)
	registerAction("logout", func(c *Ctx) { c.Logout(false) })
}

// LoginCtx is the page context for template_login.
type LoginCtx struct {
	DefaultUsername string
	DefaultPassword string
	NeverExpire     bool
	LoginError      string
	HasError        bool
	Description     string
	ShowUndelete    bool
}

func sha1hex(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

// Login is Login(): show the login form.
func (c *Ctx) Login() {
	c.loadLanguage("Login")
	c.SubTemplate = templateLogin
	c.PageTitle = c.Txt("34")

	c.Page = &LoginCtx{
		DefaultUsername: c.REQUEST.Str("u"),
	}

	// Set the login URL - will be used when the login process is done.
	if old := c.Session.GetStr("old_url"); old != "" && loginURLRe.MatchString(old) {
		c.Session.Set("login_url", old)
	} else {
		c.Session.Delete("login_url")
	}
}

var loginURLRe = regexp.MustCompile(`(board|topic)[=,]`)

// Login2 is Login2(): perform the actual logging-in.
func (c *Ctx) Login2() {
	a := c.App

	if c.GET.Str("sa") == "salt" && !c.User.IsGuest {
		// Refresh the password salt and re-issue the cookie.
		var timeout int64
		if cookie, err := c.R.Cookie(a.Config.CookieName); err == nil {
			val := phpUrldecode(cookie.Value)
			if loginCookieRe.MatchString(val) {
				if parts, err := phpser.Unserialize(val); err == nil && len(parts) >= 3 {
					timeout, _ = parts[2].(int64)
				}
			}
		} else if v := c.Session.GetStr("login_" + a.Config.CookieName); v != "" {
			if parts, err := phpser.Unserialize(v); err == nil && len(parts) >= 3 {
				timeout, _ = parts[2].(int64)
			}
		} else {
			c.fatalError("Login2(): Cannot be logged in without a session or cookie", true)
		}

		salt := randomSalt()
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET passwordSalt = ? WHERE ID_MEMBER = ?`), salt, c.User.ID)
		c.User.PasswordSalt = salt

		c.setLoginCookie(int(timeout-time.Now().Unix()), c.User.ID, sha1hex(c.User.Passwd+salt))

		c.redirectExit("action=login2;sa=check;member=" + itoa(c.User.ID))
	} else if c.GET.Str("sa") == "check" {
		// Double check the cookie...
		if c.GET.Int("member") != c.User.ID {
			c.fatalLangError("login_cookie_error", false)
		}

		// Some whitelisting for login_url...
		if c.Session.GetStr("login_url") == "" {
			c.redirectExit("")
		} else {
			temp := c.Session.GetStr("login_url")
			c.Session.Delete("login_url")
			c.redirectExit(temp)
		}
	}

	// Beyond this point you are assumed to be a guest trying to login.
	if !c.User.IsGuest {
		c.redirectExit("")
	}

	// Set the login_url if it's not already set.
	if c.Session.GetStr("login_url") == "" {
		if old := c.Session.GetStr("old_url"); old != "" && loginURLRe.MatchString(old) {
			c.Session.Set("login_url", old)
		}
	}

	// Are you guessing with a script that doesn't keep the session id?
	c.spamProtection("login")

	// Been guessing a lot, haven't we?
	if int(c.Session.GetInt("failed_login")) >= a.SettingInt("failed_login_threshold")*3 {
		c.fatalLangError("login_threshold_fail", true)
	}

	// Set up the cookie length.
	cookieTime := a.SettingInt("cookieTime")
	if c.POST.Has("cookieneverexp") || (!empty(c.POST.Str("cookielength")) && c.POST.Int("cookielength") == -1) {
		cookieTime = 3153600
	} else if !empty(c.POST.Str("cookielength")) && (c.POST.Int("cookielength") >= 1 || c.POST.Int("cookielength") <= 525600) {
		cookieTime = c.POST.Int("cookielength")
	}

	c.loadLanguage("Login")
	c.SubTemplate = templateLogin
	if a.Config.Maintenance != 0 || a.SettingEmpty("allow_guestAccess") {
		c.SubTemplate = templateKickGuest
	}

	// Set up the default/fallback stuff.
	page := &LoginCtx{
		DefaultUsername: Htmlspecialchars(c.REQUEST.Str("user")),
		NeverExpire:     cookieTime == 525600 || cookieTime == 3153600,
		LoginError:      c.Txt("106"),
		HasError:        true,
	}
	c.Page = page
	c.PageTitle = c.Txt("34")

	// You forgot to type your username, dummy!
	if c.REQUEST.Str("user") == "" {
		page.LoginError = c.Txt("37")
		return
	}

	// Hmm... maybe 'admin' will login with no password. Uhh... NO!
	hashPasswrd := c.POST.Str("hash_passwrd")
	if c.REQUEST.Str("passwrd") == "" && len(hashPasswrd) != 40 {
		page.LoginError = c.Txt("38")
		return
	}

	// No funky symbols either.
	if funkyUserRe.MatchString(unHtmlspecialchars(c.REQUEST.Str("user"))) {
		page.LoginError = c.Txt("240")
		return
	}

	// Load the data up!
	user := c.REQUEST.Str("user")
	var passwd, lngfile, emailAddress, additionalGroups, memberName, passwordSalt string
	var idMember, idGroup, isActivated int
	scan := func(where string) error {
		return a.DB.QueryRow(a.Q(`
			SELECT passwd, ID_MEMBER, ID_GROUP, lngfile, is_activated, emailAddress, additionalGroups, memberName, passwordSalt
			FROM {$db_prefix}members
			WHERE `+where+` = ?
			LIMIT 1`), user).Scan(&passwd, &idMember, &idGroup, &lngfile, &isActivated,
			&emailAddress, &additionalGroups, &memberName, &passwordSalt)
	}
	if err := scan("memberName"); err != nil {
		// Probably mistyped or their email, try it as an email address.
		if err := scan("emailAddress"); err != nil {
			page.LoginError = c.Txt("40")
			return
		}
	}

	// What is the true activation status of this account?
	activationStatus := isActivated
	if activationStatus > 10 {
		activationStatus -= 10
	}

	// Check if the account is activated - COPPA first...
	if activationStatus == 5 {
		page.LoginError = c.Txt("coppa_not_completed1") + ` <a href="` + a.ScriptURL + `?action=coppa;member=` + itoa(idMember) + `">` + c.Txt("coppa_not_completed2") + `</a>`
		return
	} else if activationStatus == 3 {
		// Awaiting approval still?
		c.fatalLangError("still_awaiting_approval", true)
	} else if activationStatus == 4 {
		// Awaiting deletion, changed their mind?
		if !c.REQUEST.Has("undelete") {
			page.LoginError = c.Txt("awaiting_delete_account")
			page.ShowUndelete = true
			return
		}
		// Otherwise reactivate!
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET is_activated = 1 WHERE ID_MEMBER = ?`), idMember)
		unapproved := a.SettingInt("unapprovedMembers")
		if unapproved > 0 {
			unapproved--
		}
		a.UpdateSettings(map[string]string{"unapprovedMembers": itoa(unapproved)})
	} else if activationStatus != 1 {
		// Standard activation?
		c.logError(c.Txt("activate_not_completed1") + ` - <span class="remove">` + memberName + `</span>`)
		page.LoginError = c.Txt("activate_not_completed1") + ` <a href="` + a.ScriptURL + `?action=activate;sa=resend;u=` + itoa(idMember) + `">` + c.Txt("activate_not_completed2") + `</a>`
		return
	}

	// Figure out the password using SMF's encryption.
	var shaPasswd string
	if len(hashPasswrd) == 40 {
		// Needs upgrading?
		if len(passwd) != 40 {
			page.LoginError = c.Txt("login_hash_error")
			c.DisableLoginHash = true
			return
		} else if subtle.ConstantTimeCompare([]byte(hashPasswrd), []byte(sha1hex(passwd+c.Sc))) == 1 {
			// Challenge passed.
			shaPasswd = passwd
		} else {
			c.Session.Set("failed_login", c.Session.GetInt("failed_login")+1)
			if int(c.Session.GetInt("failed_login")) >= a.SettingInt("failed_login_threshold") {
				c.redirectExit("action=reminder")
			}
			c.logError(c.Txt("39") + ` - <span class="remove">` + memberName + `</span>`)
			c.DisableLoginHash = true
			page.LoginError = c.Txt("39")
			return
		}
	} else {
		shaPasswd = sha1hex(strings.ToLower(memberName) + unHtmlspecialchars(c.REQUEST.Str("passwrd")))
	}

	// Bad password!  Thought you could fool the database?!
	if passwd != shaPasswd {
		// Maybe we were too hasty... try some other authentication methods.
		plain := unHtmlspecialchars(c.REQUEST.Str("passwrd"))
		var otherPasswords []string
		if passwordSalt == "" {
			otherPasswords = append(otherPasswords,
				fmt.Sprintf("%x", md5.Sum([]byte(plain))), // md5
				sha1hex(plain), // plain sha1
				md5HMAC(plain, strings.ToLower(memberName)),
				fmt.Sprintf("%x", md5.Sum([]byte(plain+strings.ToLower(memberName)))),
				plain, // none at all
			)
			// (crypt()-based legacy formats are not portable to Go's stdlib
			// and predate SMF 1.0; intentionally skipped.)
		} else if len(passwd) == 32 {
			otherPasswords = append(otherPasswords,
				fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x", md5.Sum([]byte(plain)))+passwordSalt))),
				fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x", md5.Sum([]byte(passwordSalt)))+fmt.Sprintf("%x", md5.Sum([]byte(plain)))))),
			)
		}
		// Maybe they are using a hash from before the password fix.
		otherPasswords = append(otherPasswords, sha1hex(strings.ToLower(memberName)+addslashes(plain)))

		matched := false
		for _, op := range otherPasswords {
			if passwd == op {
				matched = true
				break
			}
		}

		if matched {
			// Whichever encryption it was using, let's make it use SMF's now.
			passwd = shaPasswd
			passwordSalt = randomSalt()
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET passwd = ?, passwordSalt = ? WHERE ID_MEMBER = ?`),
				passwd, passwordSalt, idMember)
		} else {
			// They've messed up again - keep a count to see if they need a hand.
			c.Session.Set("failed_login", c.Session.GetInt("failed_login")+1)

			// Hmm... don't remember it, do you?  Try the password reminder ;).
			if int(c.Session.GetInt("failed_login")) >= a.SettingInt("failed_login_threshold") {
				c.redirectExit("action=reminder")
			}
			c.logError(c.Txt("39") + ` - <span class="remove">` + memberName + `</span>`)
			page.LoginError = c.Txt("39")
			return
		}
	} else if passwordSalt == "" {
		// Correct password, but they've got no salt; fix it!
		passwordSalt = randomSalt()
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET passwordSalt = ? WHERE ID_MEMBER = ?`), passwordSalt, idMember)
	}

	// Get ready to set the cookie...
	c.User.ID = idMember
	c.User.Username = memberName
	c.User.Passwd = passwd
	c.User.PasswordSalt = passwordSalt
	c.User.Email = emailAddress

	// Bam!  Cookie set.  A session too, just incase.
	c.setLoginCookie(60*cookieTime, idMember, sha1hex(passwd+passwordSalt))

	// Reset the login threshold.
	c.Session.Delete("failed_login")

	c.User.IsGuest = false
	c.User.Groups = []int{idGroup}
	if additionalGroups != "" {
		for _, g := range strings.Split(additionalGroups, ",") {
			c.User.Groups = append(c.User.Groups, atoi(g))
		}
	}
	c.User.IsAdmin = c.User.InGroup(1)

	// Are you banned?
	c.isNotBanned(true)

	// An administrator, set up the login so they don't have to type it again.
	if c.User.IsAdmin {
		c.Session.Set("admin_time", time.Now().Unix())
		c.Session.Delete("just_registered")
	}

	// Don't stick the language or theme after this point.
	c.Session.Delete("language")
	c.Session.Delete("ID_THEME")

	// You've logged in, haven't you?
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET lastLogin = ?, memberIP = ?, memberIP2 = ? WHERE ID_MEMBER = ?`),
		time.Now().Unix(), c.User.IP, c.BanCheckIP, idMember)

	// Get rid of the online entry for that old guest....
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE session = ?`), "ip"+c.User.IP)
	c.Session.Set("log_time", 0)

	// Just log you back out if it's in maintenance mode and you AREN'T an admin.
	if a.Config.Maintenance == 0 || c.allowedTo("admin_forum") {
		c.redirectExit("action=login2;sa=check;member=" + itoa(idMember))
	} else {
		c.redirectExit("action=logout;sesc=" + c.Sc)
	}
}

var funkyUserRe = regexp.MustCompile(`[<>&"'=\\]`)

// addslashes is PHP addslashes (for the legacy pre-password-fix hash).
func addslashes(s string) string {
	return strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`, "\x00", `\0`).Replace(s)
}

// Logout is Logout($internal).
func (c *Ctx) Logout(internal bool) {
	a := c.App

	// Make sure they aren't being auto-logged out.
	if !internal {
		c.checkSession("get", "", true)
	}

	// Just ensure they aren't a guest!
	if !c.User.IsGuest {
		// If you log out, you aren't online anymore :P.
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE ID_MEMBER = ?`), c.User.ID)
	}

	c.Session.Set("log_time", 0)

	// Empty the cookie! (set it in the past, and for ID_MEMBER = 0)
	c.setLoginCookie(-3600, 0, "")
	logoutURL := c.Session.GetStr("logout_url")
	c.destroySession()
	if c.User.ID != 0 {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET passwordSalt = ? WHERE ID_MEMBER = ?`), randomSalt(), c.User.ID)
	}

	// Off to the merry board index we go!
	if logoutURL == "" {
		c.redirectExit("")
	} else {
		c.redirectExit(logoutURL)
	}
}

// md5HMAC is md5_hmac($data, $key): old style SMF 1.0.x/YaBB SE hashing.
func md5HMAC(data, key string) string {
	if len(key) > 64 {
		sum := md5.Sum([]byte(key))
		key = string(sum[:])
	}
	for len(key) < 64 {
		key += "\x00"
	}
	inner := make([]byte, 64)
	outer := make([]byte, 64)
	for i := 0; i < 64; i++ {
		inner[i] = key[i] ^ 0x36
		outer[i] = key[i] ^ 0x5c
	}
	innerSum := md5.Sum(append(inner, data...))
	return fmt.Sprintf("%x", md5.Sum(append(outer, innerSum[:]...)))
}
