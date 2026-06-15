package app

// Port of Sources/Load.php: loadUserSettings, loadBoard, loadPermissions,
// loadTheme, getBoardParents, censorText, loadJumpTo.

import (
	"crypto/sha1"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"smf/internal/phpser"
)

// User is $user_info (plus the bits of $user_settings carried forward).
type User struct {
	ID                int // $ID_MEMBER
	Username          string
	Name              string // realName
	Email             string
	Passwd            string
	Language          string
	IsGuest           bool
	IsAdmin           bool
	IsMod             bool
	Groups            []int
	Theme             int
	LastLogin         int64
	IP, IP2           string
	Posts             int
	TimeFormat        string
	TimeOffset        float64
	AvatarURL         string
	AvatarFilename    string
	AvatarCustomDir   bool
	AvatarIDAttach    int
	SmileySet         string
	Messages          int
	UnreadMessages    int
	TotalTimeLoggedIn int64
	Buddies           []int
	MessageLabels     string
	Permissions       []string
	QuerySeeBoard     string // SQL fragment referencing b.memberGroups

	// Extra $user_settings columns some callers need later.
	PasswordSalt   string
	IDMsgLastVisit int
	IsActivated    int
}

func (u *User) InGroup(g int) bool {
	for _, x := range u.Groups {
		if x == g {
			return true
		}
	}
	return false
}

// BoardInfo is $board_info.
type BoardInfo struct {
	ID                  int
	Name                string
	Description         string
	NumTopics           int
	CatID               int
	CatName             string
	Parent              int
	ChildLevel          int
	Theme               int
	OverrideTheme       bool
	PostsCount          bool   // empty(countPosts)
	PermissionMode      string // 'normal', 'no_polls', 'reply_only', 'read_only'
	UseLocalPermissions bool
	Groups              []int
	Moderators          []Moderator // ordered
	ParentBoards        []ParentBoard
	Error               string // '', 'exist' or 'access'
}

func (b *BoardInfo) IsModerator(id int) bool {
	for _, m := range b.Moderators {
		if m.ID == id {
			return true
		}
	}
	return false
}

type Moderator struct {
	ID         int
	Name       string
	Href, Link string
}

type ParentBoard struct {
	URL, Name string
	Level     int
}

// ThemeSettings is $settings — the default theme's rows from the themes
// table plus computed values. show_* values follow PHP's bool conversion.
type ThemeSettings struct {
	m map[string]string
}

func (t *ThemeSettings) Get(k string) string     { return t.m[k] }
func (t *ThemeSettings) Empty(k string) bool     { return empty(t.m[k]) }
func (t *ThemeSettings) Int(k string) int        { return atoi(t.m[k]) }
func (t *ThemeSettings) ThemeURL() string        { return t.m["theme_url"] }
func (t *ThemeSettings) ImagesURL() string       { return t.m["images_url"] }
func (t *ThemeSettings) DefaultThemeURL() string { return t.m["default_theme_url"] }

var loginCookieRe = regexp.MustCompile(`^a:[34]:\{i:0;(i:\d{1,6}|s:[1-8]:"\d{1,8}");i:1;s:(0|40):"([a-fA-F0-9]{40})?";i:2;[id]:\d{1,14};(i:3;i:\d;)?\}$`)

// loadUserSettings is loadUserSettings() from Load.php.
func (c *Ctx) loadUserSettings() {
	a := c.App
	idMember := 0
	password := ""

	// Check the cookie first...
	if cookie, err := c.R.Cookie(a.Config.CookieName); err == nil {
		val := phpUrldecode(cookie.Value)
		// The PHP 4.3.9 security-hole regex, kept as the format gate.
		if loginCookieRe.MatchString(val) {
			if parts, err := phpser.Unserialize(val); err == nil && len(parts) >= 2 {
				if id, ok := parts[0].(int64); ok {
					idMember = int(id)
				} else if s, ok := parts[0].(string); ok {
					idMember = atoi(s)
				}
				password, _ = parts[1].(string)
				if idMember == 0 || len(password) == 0 {
					idMember = 0
				}
			}
		}
	} else if v := c.Session.GetStr("login_" + a.Config.CookieName); v != "" &&
		(c.Session.GetStr("USER_AGENT") == c.UserAgent || !a.SettingEmpty("disableCheckUA")) {
		// ... then the session.
		if parts, err := phpser.Unserialize(v); err == nil && len(parts) >= 3 {
			id, _ := parts[0].(int64)
			password, _ = parts[1].(string)
			span, _ := parts[2].(int64)
			if id != 0 && len(password) == 40 && span > time.Now().Unix() {
				idMember = int(id)
			}
		}
	}

	var u *User
	if idMember != 0 {
		u = a.fetchUserSettings(idMember)
		if u != nil {
			// SHA-1 passwords should be 40 characters long.
			check := false
			if len(password) == 40 {
				check = fmt.Sprintf("%x", sha1.Sum([]byte(u.Passwd+u.PasswordSalt))) == password
			}
			// Wrong password or not activated - either way, you're going nowhere.
			if !(check && (u.IsActivated == 1 || u.IsActivated == 11)) {
				u = nil
			}
		}
		if u == nil {
			idMember = 0
		}
	}

	if u != nil {
		// Update the last visit time when needed (5+ hours since the post
		// they last saw was made).
		isXML := c.REQUEST.Has("xml") || c.REQUEST.Str("action") == ".xml"
		if !isXML && !c.Session.Has("ID_MSG_LAST_VISIT") {
			var visitTime int64
			a.DB.QueryRow(a.Q(`SELECT posterTime FROM {$db_prefix}messages WHERE ID_MSG = ? LIMIT 1`), u.IDMsgLastVisit).Scan(&visitTime)
			c.Session.Set("ID_MSG_LAST_VISIT", u.IDMsgLastVisit)
			if visitTime < time.Now().Unix()-5*3600 {
				now := time.Now().Unix()
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_MSG_LAST_VISIT = ?, lastLogin = ?, memberIP = ?, memberIP2 = ? WHERE ID_MEMBER = ?`),
					a.SettingInt("maxMsgID"), now, c.IP, c.BanCheckIP, u.ID)
				u.LastLogin = now
			}
		} else if !c.Session.Has("ID_MSG_LAST_VISIT") {
			c.Session.Set("ID_MSG_LAST_VISIT", u.IDMsgLastVisit)
		}
	} else {
		// This is what a guest's variables should be.
		u = &User{IsGuest: true, Groups: []int{-1}, Language: a.Config.Language, TimeFormat: a.Setting("time_format")}
	}

	u.IsAdmin = u.InGroup(1)
	u.IP = c.IP
	u.IP2 = c.BanCheckIP
	if u.TimeFormat == "" {
		u.TimeFormat = a.Setting("time_format")
	}

	// ?language= / session language switching (userLanguage setting). Only
	// English is generated so far, but keep the session semantics.
	if !a.SettingEmpty("userLanguage") && !empty(c.REQUEST.Str("language")) {
		u.Language = strings.NewReplacer(".", "_", "/", "_", "\\", "_", ":", "_").Replace(c.REQUEST.Str("language"))
		c.Session.Set("language", u.Language)
	} else if !a.SettingEmpty("userLanguage") && c.Session.GetStr("language") != "" {
		u.Language = strings.NewReplacer(".", "_", "/", "_", "\\", "_", ":", "_").Replace(c.Session.GetStr("language"))
	}

	// Just build this here, it makes it easier to change/use.
	if u.IsGuest {
		u.QuerySeeBoard = "FIND_IN_SET(-1, b.memberGroups)"
	} else if u.IsAdmin {
		u.QuerySeeBoard = "1"
	} else {
		parts := make([]string, len(u.Groups))
		for i, g := range u.Groups {
			parts[i] = itoa(g)
		}
		u.QuerySeeBoard = "(FIND_IN_SET(" + strings.Join(parts, ", b.memberGroups) OR FIND_IN_SET(") + ", b.memberGroups))"
	}

	c.User = u
}

// fetchUserSettings loads the members row (plus avatar attachment) into a
// User, like the mem.* query in loadUserSettings.
func (a *App) fetchUserSettings(id int) *User {
	u := &User{ID: id}
	var additionalGroups, buddyList string
	var groupID, postGroupID int
	var attachType int
	var avatarFilename sql.NullString
	err := a.DB.QueryRow(a.Q(`
		SELECT mem.memberName, mem.realName, mem.emailAddress, mem.passwd, mem.passwordSalt, mem.is_activated,
			mem.ID_GROUP, mem.ID_POST_GROUP, mem.additionalGroups, mem.lngfile, mem.ID_THEME, mem.lastLogin,
			mem.posts, mem.timeFormat, mem.timeOffset, mem.avatar, mem.smileySet, mem.instantMessages,
			mem.unreadMessages, mem.totalTimeLoggedIn, mem.buddy_list, mem.ID_MSG_LAST_VISIT, mem.messageLabels,
			IFNULL(a.ID_ATTACH, 0) AS ID_ATTACH, IFNULL(a.filename, ''), IFNULL(a.attachmentType, 0)
		FROM {$db_prefix}members AS mem
			LEFT JOIN {$db_prefix}attachments AS a ON (a.ID_MEMBER = mem.ID_MEMBER)
		WHERE mem.ID_MEMBER = ?
		LIMIT 1`), id).Scan(
		&u.Username, &u.Name, &u.Email, &u.Passwd, &u.PasswordSalt, &u.IsActivated,
		&groupID, &postGroupID, &additionalGroups, &u.Language, &u.Theme, &u.LastLogin,
		&u.Posts, &u.TimeFormat, &u.TimeOffset, &u.AvatarURL, &u.SmileySet, &u.Messages,
		&u.UnreadMessages, &u.TotalTimeLoggedIn, &buddyList, &u.IDMsgLastVisit, &u.MessageLabels,
		&u.AvatarIDAttach, &avatarFilename, &attachType)
	if err != nil {
		return nil
	}
	u.AvatarFilename = avatarFilename.String
	u.AvatarCustomDir = attachType == 1
	u.Groups = []int{groupID, postGroupID}
	if additionalGroups != "" {
		for _, g := range strings.Split(additionalGroups, ",") {
			u.Groups = append(u.Groups, atoi(g))
		}
	}
	u.Groups = uniqueInts(u.Groups)
	if u.Language == "" || a.SettingEmpty("userLanguage") {
		u.Language = a.Config.Language
	}
	if u.Theme < 0 {
		u.Theme = 0
	}
	if empty(buddyList) || a.SettingEmpty("enable_buddylist") {
		u.Buddies = nil
	} else {
		for _, b := range strings.Split(buddyList, ",") {
			u.Buddies = append(u.Buddies, atoi(b))
		}
	}
	return u
}

func uniqueInts(in []int) []int {
	seen := make(map[int]bool, len(in))
	out := in[:0]
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// loadBoard is loadBoard() from Load.php.
func (c *Ctx) loadBoard() {
	a := c.App

	c.User.IsMod = false
	c.LinkTree = nil

	if c.Board == 0 && c.Topic == 0 {
		c.BoardInfo = &BoardInfo{}
		return
	}

	var rows *sql.Rows
	var err error
	if c.Topic != 0 {
		rows, err = a.DB.Query(a.Q(`
			SELECT
				c.ID_CAT, b.name AS bname, b.description, b.numTopics, b.memberGroups,
				b.ID_PARENT, c.name AS cname, IFNULL(mem.ID_MEMBER, 0) AS ID_MODERATOR,
				IFNULL(mem.realName, ''), b.ID_BOARD, b.childLevel,
				b.ID_THEME, b.override_theme, b.permission_mode, b.countPosts
			FROM {$db_prefix}boards AS b, {$db_prefix}topics AS t
				LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
				LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = t.ID_BOARD)
				LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = mods.ID_MEMBER)
			WHERE b.ID_BOARD = t.ID_BOARD
				AND t.ID_TOPIC = ?`), c.Topic)
	} else {
		rows, err = a.DB.Query(a.Q(`
			SELECT
				c.ID_CAT, b.name AS bname, b.description, b.numTopics, b.memberGroups,
				b.ID_PARENT, c.name AS cname, IFNULL(mem.ID_MEMBER, 0) AS ID_MODERATOR,
				IFNULL(mem.realName, ''), b.ID_BOARD, b.childLevel,
				b.ID_THEME, b.override_theme, b.permission_mode, b.countPosts
			FROM {$db_prefix}boards AS b
				LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
				LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = b.ID_BOARD)
				LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = mods.ID_MEMBER)
			WHERE b.ID_BOARD = ?`), c.Board)
	}
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()

	bi := &BoardInfo{}
	found := false
	for rows.Next() {
		var memberGroups string
		var overrideTheme, permissionMode, countPosts, modID int
		var modName string
		if err := rows.Scan(&bi.CatID, &bi.Name, &bi.Description, &bi.NumTopics, &memberGroups,
			&bi.Parent, &bi.CatName, &modID, &modName, &bi.ID, &bi.ChildLevel,
			&bi.Theme, &overrideTheme, &permissionMode, &countPosts); err != nil {
			c.fatalDBError(err)
		}
		if !found {
			found = true
			c.Board = bi.ID
			bi.OverrideTheme = overrideTheme != 0
			bi.PostsCount = countPosts == 0
			bi.UseLocalPermissions = !a.SettingEmpty("permission_enable_by_board") && permissionMode == 1
			if a.SettingEmpty("permission_enable_by_board") {
				switch permissionMode {
				case 0:
					bi.PermissionMode = "normal"
				case 2:
					bi.PermissionMode = "no_polls"
				case 3:
					bi.PermissionMode = "reply_only"
				default:
					bi.PermissionMode = "read_only"
				}
			} else {
				bi.PermissionMode = "normal"
			}
			if memberGroups != "" {
				for _, g := range strings.Split(memberGroups, ",") {
					bi.Groups = append(bi.Groups, atoi(g))
				}
			}
			bi.ParentBoards = c.getBoardParents(bi.Parent)
		}
		if modID != 0 && !bi.IsModerator(modID) {
			bi.Moderators = append(bi.Moderators, Moderator{
				ID:   modID,
				Name: modName,
				Href: a.ScriptURL + "?action=profile;u=" + itoa(modID),
				Link: `<a href="` + a.ScriptURL + "?action=profile;u=" + itoa(modID) + `" title="` + c.Txt("62") + `">` + modName + `</a>`,
			})
		}
	}
	if !found {
		// Otherwise the topic is invalid, there are no moderators, etc.
		bi = &BoardInfo{Error: "exist"}
		c.Topic = 0
		c.Board = 0
	}
	c.BoardInfo = bi

	if c.Topic != 0 {
		c.GET.Set("board", itoa(c.Board))
	}

	if c.Board != 0 {
		c.User.IsMod = bi.IsModerator(c.User.ID)

		if len(intersectInts(c.User.Groups, bi.Groups)) == 0 && !c.User.IsAdmin {
			bi.Error = "access"
		}

		// Build up the linktree.
		c.LinkTree = append(c.LinkTree, Link{URL: a.ScriptURL + "#" + itoa(bi.CatID), Name: bi.CatName})
		for i := len(bi.ParentBoards) - 1; i >= 0; i-- {
			c.LinkTree = append(c.LinkTree, Link{URL: bi.ParentBoards[i].URL, Name: bi.ParentBoards[i].Name})
		}
		c.LinkTree = append(c.LinkTree, Link{URL: a.ScriptURL + "?board=" + itoa(c.Board) + ".0", Name: bi.Name})
	}

	// Hacker... you can't see this topic, I'll tell you that. (but
	// moderators can!)
	if bi.Error != "" && (bi.Error != "access" || !c.User.IsMod) {
		c.loadPermissions()
		c.loadTheme()

		c.GET.Set("board", "")
		c.GET.Set("topic", "")

		// If it's a prefetching agent, just make clear they're not allowed.
		if c.R.Header.Get("X-Moz") == "prefetch" {
			c.W.WriteHeader(403)
			c.exit()
		} else if c.User.IsGuest {
			c.loadLanguage("Errors")
			c.isNotGuest(c.Txt("topic_gone"))
		} else {
			c.fatalLangError("topic_gone", false)
		}
	}

	if c.User.IsMod {
		c.User.Groups = append(c.User.Groups, 3)
	}
}

func intersectInts(a, b []int) []int {
	var out []int
	for _, x := range a {
		for _, y := range b {
			if x == y {
				out = append(out, x)
				break
			}
		}
	}
	return out
}

// getBoardParents is getBoardParents() from Load.php (moderator decoration
// is omitted here; the linktree only uses url/name/level).
func (c *Ctx) getBoardParents(idParent int) []ParentBoard {
	a := c.App
	var boards []ParentBoard
	for idParent != 0 {
		var name string
		var nextParent, childLevel int
		err := a.DB.QueryRow(a.Q(`
			SELECT b.ID_PARENT, b.name, b.childLevel
			FROM {$db_prefix}boards AS b
			WHERE b.ID_BOARD = ?`), idParent).Scan(&nextParent, &name, &childLevel)
		if err != nil {
			c.fatalLangError("parent_not_found", true)
		}
		boards = append(boards, ParentBoard{
			URL:   a.ScriptURL + "?board=" + itoa(idParent) + ".0",
			Name:  name,
			Level: childLevel,
		})
		idParent = nextParent
	}
	return boards
}

// loadPermissions is loadPermissions() from Load.php.
func (c *Ctx) loadPermissions() {
	a := c.App
	c.User.Permissions = nil

	if c.User.IsAdmin {
		return
	}

	groups := make([]string, len(c.User.Groups))
	for i, g := range c.User.Groups {
		groups[i] = itoa(g)
	}
	groupList := strings.Join(groups, ", ")

	var removals []string
	rows, err := a.DB.Query(a.Q(`
		SELECT permission, addDeny
		FROM {$db_prefix}permissions
		WHERE ID_GROUP IN (` + groupList + `)`))
	if err != nil {
		c.fatalDBError(err)
	}
	for rows.Next() {
		var perm string
		var addDeny int
		rows.Scan(&perm, &addDeny)
		if addDeny == 0 {
			removals = append(removals, perm)
		} else {
			c.User.Permissions = append(c.User.Permissions, perm)
		}
	}
	rows.Close()

	// Get the board permissions.
	if c.Board != 0 {
		if c.BoardInfo == nil {
			c.fatalLangError("smf232", true)
		}
		permBoard := 0
		if c.BoardInfo.UseLocalPermissions {
			permBoard = c.Board
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT permission, addDeny
			FROM {$db_prefix}board_permissions
			WHERE ID_GROUP IN (`+groupList+`)
				AND ID_BOARD = ?`), permBoard)
		if err != nil {
			c.fatalDBError(err)
		}
		for rows.Next() {
			var perm string
			var addDeny int
			rows.Scan(&perm, &addDeny)
			if addDeny == 0 {
				removals = append(removals, perm)
			} else {
				c.User.Permissions = append(c.User.Permissions, perm)
			}
		}
		rows.Close()
	}

	// Remove all the permissions they shouldn't have ;).
	if !a.SettingEmpty("permission_enable_deny") {
		c.User.Permissions = diffStrings(c.User.Permissions, removals)
	}

	// Remove some board permissions if the board is read-only or reply-only.
	if a.SettingEmpty("permission_enable_by_board") && c.Board != 0 &&
		c.BoardInfo.PermissionMode != "normal" && !c.allowedTo("moderate_board") {
		var blocked []string
		switch c.BoardInfo.PermissionMode {
		case "read_only":
			blocked = []string{"post_new", "poll_post", "post_reply_own", "post_reply_any"}
		case "reply_only":
			blocked = []string{"post_new", "poll_post"}
		case "no_polls":
			blocked = []string{"poll_post"}
		}
		c.User.Permissions = diffStrings(c.User.Permissions, blocked)
	}

	// Banned?  Watch, don't touch.. (banPermissions arrives with Phase 2.)
	c.banPermissions()
}

func diffStrings(a, b []string) []string {
	out := a[:0]
outer:
	for _, x := range a {
		for _, y := range b {
			if x == y {
				continue outer
			}
		}
		out = append(out, x)
	}
	return out
}

// censorText is censorText() from Load.php.
func (c *Ctx) censorText(text string) string {
	a := c.App
	if (!empty(c.Options["show_no_censored"]) && !c.Theme.Empty("allow_no_censored")) || a.SettingEmpty("censor_vulgar") {
		return text
	}
	for _, re := range a.censorRegexps() {
		text = re.re.ReplaceAllString(text, re.proper)
	}
	return text
}

type censorPair struct {
	re     *regexp.Regexp
	proper string
}

// censorRegexps builds (and caches) the censor patterns the way
// censorText() quotes them.
func (a *App) censorRegexps() []censorPair {
	if v, ok := a.cache.Get("censor_regexps"); ok {
		return v.([]censorPair)
	}
	vulgar := strings.Split(a.Setting("censor_vulgar"), "\n")
	proper := strings.Split(a.Setting("censor_proper"), "\n")
	var pairs []censorPair
	for i, word := range vulgar {
		if word == "" {
			continue
		}
		p := ""
		if i < len(proper) {
			p = proper[i]
		}
		quoted := regexp.QuoteMeta(word)
		quoted = strings.NewReplacer(`\\\*`, `[*]`, `\*`, `[^\s]*?`, "&", "&amp;").Replace(quoted)
		var pattern string
		if a.SettingEmpty("censorWholeWord") {
			pattern = quoted
		} else {
			// PHP uses lookbehind/lookahead; Go RE2 has neither, so use word
			// boundaries against \W (close enough for the golden corpus; any
			// divergence will show in diffs and can be revisited).
			pattern = `(?:^|\b)` + quoted + `(?:\b|$)`
		}
		if !a.SettingEmpty("censorIgnoreCase") {
			pattern = "(?i)" + pattern
		}
		if re, err := regexp.Compile(pattern); err == nil {
			pairs = append(pairs, censorPair{re, p})
		}
		if strings.Contains(quoted, "'") {
			q2 := strings.ReplaceAll(pattern, "'", "&#039;")
			if re, err := regexp.Compile(q2); err == nil {
				pairs = append(pairs, censorPair{re, p})
			}
		}
	}
	a.cache.Put("censor_regexps", pairs, 120*time.Second)
	return pairs
}

// loadJumpTo is loadJumpTo() from Load.php.
func (c *Ctx) loadJumpTo() {
	if c.JumpTo != nil {
		return
	}
	a := c.App
	rows, err := a.DB.Query(a.Q(`
		SELECT c.name AS catName, c.ID_CAT, b.ID_BOARD, b.name AS boardName, b.childLevel
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE ` + c.User.QuerySeeBoard))
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()
	c.JumpTo = []JumpToCat{}
	for rows.Next() {
		var catName, boardName string
		var catID, boardID, childLevel int
		rows.Scan(&catName, &catID, &boardID, &boardName, &childLevel)
		if len(c.JumpTo) == 0 || c.JumpTo[len(c.JumpTo)-1].ID != catID {
			c.JumpTo = append(c.JumpTo, JumpToCat{ID: catID, Name: catName})
		}
		cat := &c.JumpTo[len(c.JumpTo)-1]
		cat.Boards = append(cat.Boards, JumpToBoard{
			ID:         boardID,
			Name:       boardName,
			ChildLevel: childLevel,
			IsCurrent:  boardID == c.Board,
		})
	}
}
