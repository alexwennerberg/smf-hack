package app

// Port of Sources/Subs-Auth.php (Phase 2 scope: setLoginCookie, url_parts;
// plus the findmember/requestmembers popups used by the PM send form).

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"smf/internal/phpser"
)

// randomSalt is substr(md5(mt_rand()), 0, 4).
func randomSalt() string {
	var b [8]byte
	rand.Read(b[:])
	return fmt.Sprintf("%x", md5.Sum(b[:]))[:4]
}

var ipv4HostRe = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)
var globalDomainRe = regexp.MustCompile(`(?i)(?:[^\.]+\.)?([^\.]{2,}\..+)$`)

// urlParts is url_parts($local, $global): (cookie domain, cookie path).
func (a *App) urlParts(local, global bool) (string, string) {
	parsed, err := url.Parse(a.Config.BoardURL)
	host, path := "", ""
	if err == nil {
		host = parsed.Hostname()
		path = parsed.Path
	}

	// Is local cookies off?
	if path == "" || !local {
		path = ""
	}

	if global && !ipv4HostRe.MatchString(host) {
		// Globalize cookies across domains (filter out IP-addresses)?
		if m := globalDomainRe.FindStringSubmatch(host); m != nil {
			host = "." + m[1]
		}
	} else if !local && !global {
		// We shouldn't use a host at all if both options are off.
		host = ""
	} else if !strings.Contains(host, ".") {
		// The host also shouldn't be set if there aren't any dots in it.
		host = ""
	}

	return host, path + "/"
}

// setLoginCookie is setLoginCookie($cookie_length, $id, $password).
func (c *Ctx) setLoginCookie(cookieLength int, id int, password string) {
	a := c.App

	cookieState := 0
	if !a.SettingEmpty("localCookies") {
		cookieState |= 1
	}
	if !a.SettingEmpty("globalCookies") {
		cookieState |= 2
	}

	// Get the data and path to set it on.
	var data string
	if id == 0 {
		data, _ = phpser.Serialize([]any{0, "", 0})
	} else {
		data, _ = phpser.Serialize([]any{id, password, int(time.Now().Unix()) + cookieLength, cookieState})
	}
	domain, path := a.urlParts(!a.SettingEmpty("localCookies"), !a.SettingEmpty("globalCookies"))

	// Set the cookie, $_COOKIE, and session variable.
	http.SetCookie(c.W, &http.Cookie{
		Name:    a.Config.CookieName,
		Value:   url.QueryEscape(data),
		Expires: time.Now().Add(time.Duration(cookieLength) * time.Second),
		Path:    path,
		Domain:  domain,
	})

	// Make sure the user logs in with a new session ID.
	if c.Session.GetStr("login_"+a.Config.CookieName) != data {
		c.regenerateSession()
		c.Session.Set("login_"+a.Config.CookieName, data)
	}
}

// regenerateSession gives the user a fresh session ID, keeping the data
// (the session_regenerate_id dance in setLoginCookie).
func (c *Ctx) regenerateSession() {
	a := c.App
	old := c.Session.ID
	c.Session.ID = newSessionID()
	c.Session.dirty = true
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}sessions WHERE session_id = ?`), old)
	http.SetCookie(c.W, &http.Cookie{
		Name:     sessionCookieName,
		Value:    c.Session.ID,
		Path:     "/",
		HttpOnly: true,
	})
}

// destroySession is session_destroy(): drop the row and start anew.
func (c *Ctx) destroySession() {
	a := c.App
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}sessions WHERE session_id = ?`), c.Session.ID)
	c.Session.data = map[string]any{}
	c.regenerateSession()
	c.Session.Set("rand_code", newSessionID())
	c.Sc = c.Session.GetStr("rand_code")
}

// checkSession is checkSession() from Security.php.
func (c *Ctx) checkSession(typ string, fromAction string, isFatal bool) string {
	a := c.App
	errKey := ""

	switch typ {
	case "post":
		if c.POST.Str("sc") != c.Sc {
			errKey = "smf304"
		}
	case "get":
		if c.GET.Str("sesc") != c.Sc {
			errKey = "smf305"
		}
	case "request":
		if c.GET.Str("sesc") != c.Sc && c.POST.Str("sc") != c.Sc {
			errKey = "smf305"
		}
	}

	// Verify that they aren't changing user agents on us - that could be bad.
	if c.Session.GetStr("USER_AGENT") != c.UserAgent && a.SettingEmpty("disableCheckUA") {
		errKey = "smf305"
	}

	// Make sure a page with session check requirement is not being
	// prefetched.
	if c.R.Header.Get("X-Moz") == "prefetch" {
		c.W.WriteHeader(403)
		c.exit()
	}

	logError := false

	// Check the referring site - it should be the same server at least!
	referrerHost := ""
	if ref := c.R.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil {
			referrerHost = u.Hostname()
		}
	}
	if referrerHost != "" {
		realHost := c.R.Host
		if i := strings.IndexByte(realHost, ':'); i >= 0 {
			realHost = realHost[:i]
		}
		parsedHost := ""
		if u, err := url.Parse(a.Config.BoardURL); err == nil {
			parsedHost = u.Hostname()
		}

		// Are global cookies on?  If so, let's check them ;).
		if !a.SettingEmpty("globalCookies") {
			trim := func(h string) string {
				if m := regexp.MustCompile(`(?i)(?:[^\.]+\.)?([^\.]{3,}\..+)$`).FindStringSubmatch(h); m != nil {
					return m[1]
				}
				return h
			}
			parsedHost = trim(parsedHost)
			referrerHost = trim(referrerHost)
			realHost = trim(realHost)
		}

		// Okay: referrer must either match parsed_url or real_host.
		if parsedHost != "" && !strings.EqualFold(referrerHost, parsedHost) && !strings.EqualFold(referrerHost, realHost) {
			errKey = "smf306"
			logError = true
		}
	}

	// If a from_action is specified you'd better have an old_url.
	if fromAction != "" {
		oldURL := c.Session.GetStr("old_url")
		re := regexp.MustCompile(`[?;&]action=` + regexp.QuoteMeta(fromAction) + `([;&]|$)`)
		if oldURL == "" || !re.MatchString(oldURL) {
			errKey = "smf306"
			logError = true
		}
	}

	if strings.EqualFold(c.UserAgent, "hacker") {
		c.fatalError("Sound the alarm!  It's a hacker!  Close the castle gates!!", false)
	}

	// Everything is ok, return an empty string.
	if errKey == "" {
		return ""
	}
	if isFatal {
		c.fatalLangError(errKey, logError)
	}
	return errKey
}

// spamProtection is spamProtection() from Subs.php (flood control).
func (c *Ctx) spamProtection(errorType string) bool {
	a := c.App
	now := time.Now().Unix()
	waitTime := int64(a.SettingInt("spamWaitTime"))

	// Delete old entries...
	if errorType == "spam" && !c.allowedTo("moderate_board") {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_floodcontrol WHERE logTime < ?`), now-waitTime)
	} else {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_floodcontrol WHERE (logTime < ? AND ip = ?) OR logTime < ?`),
			now-2, c.User.IP, now-waitTime)
	}

	// Add a new entry, deleting the old if necessary.
	ip := c.User.IP
	if len(ip) > 16 {
		ip = ip[:16]
	}
	// REPLACE reports 1 affected row on plain insert, 2 (delete+insert) when
	// it replaced — PHP treats != 1 as "was already there".
	var existed int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_floodcontrol WHERE ip = ?`), ip).Scan(&existed)
	a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_floodcontrol (ip, logTime) VALUES (?, ?)`), ip, now)
	if existed != 0 {
		// Spammer!  You only have to wait a *few* seconds!
		c.fatalLangError(errorType+"WaitTime_broken", false, itoa(a.SettingInt("spamWaitTime")))
		return true
	}

	// They haven't posted within the limit.
	return false
}

// validateSession is validateSession() from Security.php: requires an
// admin-session password re-entry within the last hour.
func (c *Ctx) validateSession() {
	a := c.App

	// We don't care if the option is off, because Guests should NEVER get
	// past here.
	c.isNotGuest("")

	// Is the security option off?  Or are they already logged in?
	if !a.SettingEmpty("securityDisable") {
		return
	}
	if adminTime := int64(toFloat(c.Session.Get("admin_time"))); adminTime != 0 && adminTime+3600 >= time.Now().Unix() {
		return
	}

	// Hashed password, ahoy!
	if hash := c.POST.Str("admin_hash_pass"); c.POST.Has("admin_hash_pass") && len(hash) == 40 {
		c.checkSession("post", "", true)

		if hash == fmt.Sprintf("%x", sha1.Sum([]byte(c.User.Passwd+c.Sc))) {
			c.Session.Set("admin_time", time.Now().Unix())
			c.Session.Delete("request_referer")
			return
		}
	}
	// Posting the password... check it.
	if c.POST.Has("admin_pass") {
		c.checkSession("post", "", true)

		// Password correct?
		if fmt.Sprintf("%x", sha1.Sum([]byte(strings.ToLower(c.User.Username)+c.POST.Str("admin_pass")))) == c.User.Passwd {
			c.Session.Set("admin_time", time.Now().Unix())
			c.Session.Delete("request_referer")
			return
		}
	}

	// Better be sure to remember the real referer
	if c.Session.GetStr("request_referer") == "" {
		c.Session.Set("request_referer", c.R.Header.Get("Referer"))
	} else if c.POST.Len() == 0 {
		c.Session.Delete("request_referer")
	}
	// Need to type in a password for that, man.
	c.adminLogin()
}

// adminLogin is adminLogin() from Subs-Auth.php: show the admin-session
// password prompt, replaying the request's GET/POST data.
func (c *Ctx) adminLogin() {
	a := c.App

	c.loadLanguage("Admin")

	// Start with nothing for get data and post data.
	getData := "?"
	postData := ""

	// Add up all the data from $_GET into get_data. (Our $scripturl never
	// contains GET stuff.)
	for _, k := range c.GET.Keys() {
		if v, ok := c.GET.Get(k).(string); ok {
			getData += url.QueryEscape(k) + "=" + url.QueryEscape(v) + ";"
		}
	}
	getData = getData[:len(getData)-1]

	// They used a wrong password, log it and unset that.
	if c.POST.Has("admin_hash_pass") || c.POST.Has("admin_pass") {
		c.logError(c.Txt("security_wrong"))
		c.POST.Delete("admin_hash_pass")
		c.POST.Delete("admin_pass")
	}

	// Now go through $_POST.  Make sure the session hash is sent.
	c.POST.Set("sc", c.Sc)
	for _, k := range c.POST.Keys() {
		postData += adminLoginOutputPostVars(k, c.POST.Get(k))
	}

	page := &AdminLoginCtx{GetData: getData, PostData: postData}
	c.Page = page

	// Now we'll use the admin_login sub template of the Login template.
	c.SubTemplate = templateAdminLogin

	// And title the page something like "Login".
	if c.PageTitle == "" {
		c.PageTitle = c.Txt("34")
	}

	c.obExit(nil, nil, false)
	_ = a
}

// adminLoginOutputPostVars is adminLogin_outputPostVars().
func adminLoginOutputPostVars(k string, v any) string {
	switch val := v.(type) {
	case string:
		return "\n<input type=\"hidden\" name=\"" + Htmlspecialchars(k) + "\" value=\"" +
			strings.NewReplacer(`"`, "&quot;", "<", "&lt;", ">", "&gt;").Replace(val) + "\" />"
	case *PArray:
		ret := ""
		for _, k2 := range val.Keys() {
			ret += adminLoginOutputPostVars(k+"["+k2+"]", val.Get(k2))
		}
		return ret
	}
	return ""
}

// AdminLoginCtx is the page context for template_admin_login.
type AdminLoginCtx struct {
	GetData  string
	PostData string
}

// templateAdminLogin is template_admin_login() from Login.template.php.
func templateAdminLogin(c *Ctx) {
	page := c.Page.(*AdminLoginCtx)
	scripturl := c.App.ScriptURL

	// Since this should redirect to whatever they were doing, send all the
	// get data.
	c.O(`
<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/sha1.js"></script>

<form action="`, scripturl, page.GetData, `" method="post" accept-charset="`, c.CharacterSet, `" name="frmLogin" id="frmLogin" onsubmit="hashAdminPassword(this, '`, c.User.Username, `', '`, c.Sc, `');">
	<table border="0" width="400" cellspacing="0" cellpadding="3" class="tborder" align="center">
		<tr class="titlebg">
			<td align="left">
				<img src="`, c.Theme.ImagesURL(), `/icons/login_sm.gif" alt="" align="top" /> `, c.Txt("34"), `
			</td>
		</tr>`)

	// We just need the password.
	c.O(`
		<tr class="windowbg">
			<td align="center" style="padding: 1ex 0;">
				<b>`, c.Txt("36"), `:</b> <input type="password" name="admin_pass" size="24" /> <a href="`, scripturl, `?action=helpadmin;help=securityDisable_why" onclick="return reqWin(this.href);" class="help"><img src="`, c.Theme.ImagesURL(), `/helptopics.gif" alt="`, c.Txt("119"), `" align="middle" /></a><br />
				<input type="submit" value="`, c.Txt("34"), `" style="margin-top: 2ex;" />
			</td>
		</tr>
	</table>`)

	// Make sure to output all the old post data.
	c.O(page.PostData, `

	<input type="hidden" name="admin_hash_pass" value="" />
</form>`)

	// Focus on the password box.
	c.O(`
<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
	document.forms.frmLogin.admin_pass.focus();
// ]]></script>`)
}

func init() {
	registerAction("findmember", (*Ctx).JSMembers)
	registerAction("requestmembers", (*Ctx).RequestMembers)
}

// FindMembersCtx is the page context for the find_members popup.
type FindMembersCtx struct {
	InputBoxName string
	Delimiter    string
	QuoteResults bool
	LastSearch   string
	HasSearch    bool
	ShowBuddies  bool
	BuddySearch  bool
	Results      []*foundMember
	PageIndex    string
}

var inputBoxNameRe = regexp.MustCompile(`^[\w-]+$`)

// JSMembers is JSMembers() (?action=findmember): the member-search popup that
// feeds the PM send form's To/Bcc boxes.
func (c *Ctx) JSMembers() {
	a := c.App
	scripturl := a.ScriptURL

	c.checkSession("get", "", true)

	// Why is this in the Help template, you ask? Well, erm... it helps you.
	c.TemplateLayers = []string{}
	c.SubTemplate = templateFindMembers

	page := &FindMembersCtx{}
	c.Page = page

	start := 0
	if c.REQUEST.Has("search") {
		page.HasSearch = true
		page.LastSearch = Htmlspecialchars(c.REQUEST.Str("search"))
		start = c.REQUEST.Int("start")
	}

	// Allow the user to pass the input to be added to to the box.
	page.InputBoxName = "to"
	if input := c.REQUEST.Str("input"); inputBoxNameRe.MatchString(input) {
		page.InputBoxName = input
	}

	// Take the delimiter over GET in case it's \n or something.
	page.Delimiter = ", "
	if c.REQUEST.Has("delim") {
		page.Delimiter = Htmlspecialchars(c.REQUEST.Str("delim"))
	}
	page.QuoteResults = !empty(c.REQUEST.Str("quote"))

	// Some buddy related settings ;)
	page.ShowBuddies = len(c.User.Buddies) != 0
	page.BuddySearch = c.REQUEST.Has("buddies")

	// If the user has done a search, well - search.
	if page.HasSearch {
		found := c.findMembers([]string{page.LastSearch}, true, page.BuddySearch, 0)
		totalResults := len(found)

		quoteParam := ""
		if page.QuoteResults {
			quoteParam = ";quote=1"
		}
		buddyParam := ""
		if page.BuddySearch {
			buddyParam = ";buddies"
		}
		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=findmember;search="+page.LastSearch+";sesc="+c.Sc+";input="+page.InputBoxName+quoteParam+buddyParam,
			start, totalResults, 7, false)

		// findMembers returns a map; iterate in a stable (ID-ascending) order
		// to mirror MySQL's storage order, then take this page's slice.
		ids := make([]int, 0, len(found))
		for id := range found {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for i := start; i < start+7 && i < len(ids); i++ {
			page.Results = append(page.Results, found[ids[i]])
		}
	}
}

// RequestMembers is RequestMembers() (?action=requestmembers): the plain-text
// realName list that drives the To/Bcc autocompleter.
func (c *Ctx) RequestMembers() {
	a := c.App

	c.checkSession("get", "", true)

	search := Htmlspecialchars(c.REQUEST.Str("search")) + "*"
	search = strings.TrimSpace(strings.ToLower(search))
	search = strings.NewReplacer("%", `\%`, "_", `\_`, "*", "%", "?", "_", "&#038;", "&amp;").Replace(search)

	buddiesCond := ""
	if c.REQUEST.Has("buddies") {
		ids := make([]string, len(c.User.Buddies))
		for i, b := range c.User.Buddies {
			ids[i] = itoa(b)
		}
		buddiesCond = `
			AND ID_MEMBER IN (` + strings.Join(ids, ", ") + `)`
	}
	limit := 800
	if len(search) <= 2 {
		limit = 100
	}

	c.W.Header().Set("Content-Type", "text/plain; charset="+c.CharacterSet)

	rows, err := a.DB.Query(a.Q(`
		SELECT realName
		FROM {$db_prefix}members
		WHERE realName LIKE ? ESCAPE '\'`+buddiesCond+`
			AND is_activated IN (1, 11)
		LIMIT `+itoa(limit)), search)
	if err == nil {
		// ISO-8859-1 port: skip PHP's iconv/UTF-8 re-encoding of numeric
		// entities; emit names in the forum charset with the same entity
		// normalization PHP applies.
		repl := strings.NewReplacer("&amp;", "&#038;", "&lt;", "&#060;", "&gt;", "&#062;", "&quot;", "&#034;")
		for rows.Next() {
			var realName string
			rows.Scan(&realName)
			c.O(repl.Replace(realName), "\n")
		}
		rows.Close()
	}

	// obExit(false): flush what we have, no chrome.
	c.exit()
}

// templateFindMembers is template_find_members() from Help.template.php.
func templateFindMembers(c *Ctx) {
	page := c.Page.(*FindMembersCtx)
	scripturl := c.App.ScriptURL

	rtl := ""
	if c.RightToLeft {
		rtl = ` dir="rtl"`
	}
	// theTextBox.value assignment differs for quoted results.
	quoteEmpty := "name"
	quoteAppend := "name"
	if page.QuoteResults {
		quoteEmpty = `"\"" + name + "\""`
		quoteAppend = `"\"" + name + "\""`
	}

	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"`, rtl, `>
	<head>
		<title>`, c.Txt("find_members"), `</title>
		<meta http-equiv="Content-Type" content="text/html; charset=`, c.CharacterSet, `" />
		<link rel="stylesheet" type="text/css" href="`, c.Theme.ThemeURL(), `/style.css" />
		<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/script.js"></script>
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var membersAdded = [];
			function addMember(name)
			{
				var theTextBox = window.opener.document.getElementById("`, page.InputBoxName, `");

				if (typeof(membersAdded[name]) != "undefined")
					return;
				membersAdded[name] = true;

				if (theTextBox.value.length < 1)
					theTextBox.value = `, quoteEmpty, `;
				else
					theTextBox.value += "`, page.Delimiter, `" + `, quoteAppend, `;

				window.focus();
			}
		// ]]></script>
	</head>
	<body>
		<form action="`, scripturl, `?action=findmember;sesc=`, c.Sc, `" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" width="100%" cellpadding="4" cellspacing="0" class="tborder">
				<tr class="titlebg">
					<td align="center" colspan="2">`, c.Txt("find_members"), `</td>
				</tr>
				<tr class="windowbg">
					<td align="left" colspan="2">
						<b>`, c.Txt("find_username"), `:</b><br />
						<input type="text" name="search" id="search" value="`, page.LastSearch, `" style="margin-top: 4px; width: 96%;" /><br />
					</td>
				</tr>
				<tr class="windowbg" valign="top">`)

	// Only offer to search for buddies if we have some!
	if page.ShowBuddies {
		buddyChecked := ""
		if page.BuddySearch {
			buddyChecked = ` checked="checked"`
		}
		c.O(`
					<td align="left">
						<span class="smalltext"><label for="buddies"><input type="checkbox" class="check" name="buddies" id="buddies"`, buddyChecked, ` /> `, c.Txt("find_buddies"), `</label></span>
					</td>
					<td align="right">`)
	} else {
		c.O(`
					<td>`)
	}

	c.O(`
						<span class="smalltext"><i>`, c.Txt("find_wildcards"), `</i></span>
					</td>
				</tr>
				<tr class="windowbg">
					<td align="right" colspan="2">
						<input type="submit" value="`, c.Txt("182"), `" />
						<input type="button" value="`, c.Txt("find_close"), `" onclick="window.close();" />
					</td>
				</tr>
			</table>

			<br />

			<table border="0" width="100%" cellpadding="4" cellspacing="0" class="tborder">
				<tr class="titlebg">
					<td align="center">`, c.Txt("find_results"), `</td>
				</tr>`)

	if len(page.Results) == 0 {
		c.O(`
				<tr class="windowbg">
					<td align="center">`, c.Txt("find_no_results"), `</td>
				</tr>`)
	} else {
		alternate := true
		for _, result := range page.Results {
			css := "windowbg"
			if alternate {
				css = "windowbg2"
			}
			c.O(`
				<tr class="`, css, `" valign="middle">
					<td align="left">
						<a href="`, result.Href, `" target="_blank"><img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="`, c.Txt("27"), `" title="`, c.Txt("27"), `" border="0" /></a>
						<a href="javascript:void(0);" onclick="addMember(this.title); return false;" title="`, result.Username, `">`, result.Name, `</a>
					</td>
				</tr>`)
			alternate = !alternate
		}

		c.O(`
				<tr class="titlebg">
					<td align="left">`, c.Txt("139"), `: `, page.PageIndex, `</td>
				</tr>`)
	}

	quoteHidden := "0"
	if page.QuoteResults {
		quoteHidden = "1"
	}
	c.O(`
			</table>
			<input type="hidden" name="input" value="`, page.InputBoxName, `" />
			<input type="hidden" name="delim" value="`, page.Delimiter, `" />
			<input type="hidden" name="quote" value="`, quoteHidden, `" />
		</form>`)

	if len(page.Results) == 0 {
		c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			document.getElementById("search").focus();
		// ]]></script>`)
	}

	c.O(`
	</body>
</html>`)
}
