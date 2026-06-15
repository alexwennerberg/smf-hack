package app

// Port of Sources/Security.php (Phase 1 scope: allowedTo/isAllowedTo,
// is_not_guest, banPermissions and the is_not_banned access check; the
// session-check/login machinery lands with Phase 2).

import (
	"regexp"
	"strings"
	"time"
)

// allowedTo is allowedTo() from Security.php for the current board.
func (c *Ctx) allowedTo(permissions ...string) bool {
	if len(permissions) == 0 {
		return true
	}
	// You're never allowed to do something if your data hasn't been loaded yet!
	if c.User == nil {
		return false
	}
	// Administrators are supermen :P.
	if c.User.IsAdmin {
		return true
	}
	for _, p := range permissions {
		for _, have := range c.User.Permissions {
			if p == have {
				return true
			}
		}
	}
	return false
}

// isAllowedTo is isAllowedTo(): error out if the permission is missing.
func (c *Ctx) isAllowedTo(permission ...string) {
	if c.allowedTo(permission...) {
		return
	}
	// PHP uses cannot_<first> when passed an array of alternatives.
	first := permission[0]
	// If they are a guest, show a login. (because the error might be gone if
	// they do!)
	if c.User.IsGuest {
		c.loadLanguage("Errors")
		c.isNotGuest(c.Txt("cannot_" + first))
	}

	// Clear the action because they aren't really doing that!
	c.GET.Set("action", "")
	c.GET.Set("board", "")
	c.GET.Set("topic", "")
	c.writeLog(true)

	c.fatalLangError("cannot_"+first, false)
}

// isNotGuest is is_not_guest(): guests get the kick_guest login screen.
func (c *Ctx) isNotGuest(message string) {
	if !c.User.IsGuest {
		return
	}

	// People always worry when they see people doing things they aren't
	// actually doing...
	c.GET.Set("action", "")
	c.GET.Set("board", "")
	c.GET.Set("topic", "")
	c.writeLog(true)

	// Just die.
	if c.REQUEST.Has("xml") {
		c.obExit(boolPtr(false), nil, false)
	}

	c.Session.Set("login_url", c.RequestURL)

	// Load the Login template and language file.
	c.loadLanguage("Login")
	c.SubTemplate = templateKickGuest
	c.KickMessage = message
	c.PageTitle = c.Txt("34")

	c.obExit(nil, nil, false)
}

var ipPartsRe = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)

// isNotBanned is is_not_banned() from Security.php.
func (c *Ctx) isNotBanned(forceCheck bool) {
	a := c.App

	// You cannot be banned if you are an admin.
	if c.User.IsAdmin {
		return
	}

	ban, _ := c.Session.Get("ban").(map[string]any)
	banStr := func(k string) string { v, _ := ban[k].(string); return v }
	banInt := func(k string) int64 {
		if f, ok := ban[k].(float64); ok {
			return int64(f)
		}
		return 0
	}

	// Only check the ban every so often. (to reduce load.)
	if forceCheck || ban == nil || a.SettingEmpty("banLastUpdated") ||
		banInt("last_checked") < int64(a.SettingInt("banLastUpdated")) ||
		int(banInt("ID_MEMBER")) != c.User.ID ||
		banStr("ip") != c.User.IP || banStr("ip2") != c.User.IP2 ||
		banStr("email") != c.User.Email {

		// Innocent until proven guilty. (but we know you are! :P)
		newBan := map[string]any{
			"last_checked": time.Now().Unix(),
			"ID_MEMBER":    c.User.ID,
			"ip":           c.User.IP,
			"ip2":          c.User.IP2,
			"email":        c.User.Email,
		}

		var banQuery []string
		var args []any
		for _, ip := range []string{c.User.IP, c.User.IP2} {
			if m := ipPartsRe.FindStringSubmatch(ip); m != nil {
				banQuery = append(banQuery, `((`+m[1]+` BETWEEN bi.ip_low1 AND bi.ip_high1)
					AND (`+m[2]+` BETWEEN bi.ip_low2 AND bi.ip_high2)
					AND (`+m[3]+` BETWEEN bi.ip_low3 AND bi.ip_high3)
					AND (`+m[4]+` BETWEEN bi.ip_low4 AND bi.ip_high4))`)
				// (hostname lookups are skipped unless enabled; same gate as
				// PHP's disableHostnameLookup, applied conservatively.)
			} else if ip == "unknown" {
				banQuery = append(banQuery, `(bi.ip_low1 = 255 AND bi.ip_high1 = 255
					AND bi.ip_low2 = 255 AND bi.ip_high2 = 255
					AND bi.ip_low3 = 255 AND bi.ip_high3 = 255
					AND bi.ip_low4 = 255 AND bi.ip_high4 = 255)`)
			}
		}
		if c.User.Email != "" {
			banQuery = append(banQuery, `(? LIKE bi.email_address)`)
			args = append(args, c.User.Email)
		}
		if !c.User.IsGuest && c.User.ID != 0 {
			banQuery = append(banQuery, `bi.ID_MEMBER = `+itoa(c.User.ID))
		}

		if len(banQuery) > 0 {
			query := a.Q(`
				SELECT bi.ID_BAN, bi.email_address, bi.ID_MEMBER, bg.cannot_access, bg.cannot_register,
					bg.cannot_post, bg.cannot_login, bg.reason
				FROM {$db_prefix}ban_groups AS bg, {$db_prefix}ban_items AS bi
				WHERE bg.ID_BAN_GROUP = bi.ID_BAN_GROUP
					AND (bg.expire_time IS NULL OR bg.expire_time > ` + itoa(int(time.Now().Unix())) + `)
					AND (` + strings.Join(banQuery, " OR ") + `)`)
			rows, err := a.DB.Query(query, args...)
			if err == nil {
				for rows.Next() {
					var idBan, idMember, cannotAccess, cannotRegister, cannotPost, cannotLogin int
					var emailAddress, reason string
					rows.Scan(&idBan, &emailAddress, &idMember, &cannotAccess, &cannotRegister, &cannotPost, &cannotLogin, &reason)
					for restriction, hit := range map[string]int{
						"cannot_access": cannotAccess, "cannot_login": cannotLogin,
						"cannot_post": cannotPost, "cannot_register": cannotRegister,
					} {
						if hit != 0 {
							entry, _ := newBan[restriction].(map[string]any)
							if entry == nil {
								entry = map[string]any{"reason": reason, "ids": []any{}}
							}
							entry["ids"] = append(entry["ids"].([]any), idBan)
							newBan[restriction] = entry
						}
					}
				}
				rows.Close()
			}
		}
		c.Session.Set("ban", newBan)
		ban = newBan
	}

	// They are banned from accessing!
	if banned, ok := ban["cannot_access"].(map[string]any); ok {
		// Log them out, if logged in. (Phase 2 expands this.)
		c.logBanned()
		reason, _ := banned["reason"].(string)
		c.loadLanguage("Errors")
		c.Page = &ErrorCtx{Title: c.Txt("your_ban"), Message: phpSprintf(c.bannedNameMessage(), c.bannedName()) + reason}
		c.fatalError(reason, false)
	}

	// Banned from logging in but logged in anyway? Phase 2 handles the
	// logout flow; cannot_post is surfaced by template_header.
}

func (c *Ctx) bannedName() string {
	if c.User.IsGuest {
		return c.Txt("28")
	}
	return c.User.Name
}

func (c *Ctx) bannedNameMessage() string {
	return c.Txt("your_ban")
}

// logBanned records the hit in log_banned, like log_ban() in Security.php.
func (c *Ctx) logBanned() {
	a := c.App
	a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}log_banned (ID_MEMBER, ip, email, logTime)
		VALUES (?, SUBSTR(?, 1, 16), ?, ?)`),
		c.User.ID, c.User.IP, c.User.Email, time.Now().Unix())
}

// banPermissions is banPermissions(): strip permissions from post-banned
// users.
func (c *Ctx) banPermissions() {
	if c.Session == nil {
		return
	}
	ban, _ := c.Session.Get("ban").(map[string]any)
	if ban == nil {
		return
	}
	if _, banned := ban["cannot_post"]; banned {
		denied := map[string]bool{}
		for _, p := range []string{
			"pm_send", "poll_post",
			"poll_add_own", "poll_add_any", "poll_edit_own", "poll_edit_any", "poll_lock_own",
			"poll_lock_any", "poll_remove_own", "poll_remove_any", "manage_attachments",
			"manage_smileys", "manage_boards", "admin_forum", "manage_permissions",
			"moderate_forum", "manage_membergroups", "manage_bans", "send_mail", "edit_news",
			"delete_own", "delete_any", "delete_replies", "make_sticky", "merge_any",
			"split_any", "modify_own", "modify_any", "modify_replies", "move_any", "send_topic",
			"lock_own", "lock_any", "remove_own", "remove_any", "post_new", "post_reply_own",
			"post_reply_any", "post_unapproved_topics", "post_unapproved_replies_own",
			"post_unapproved_replies_any",
		} {
			denied[p] = true
		}
		var kept []string
		for _, p := range c.User.Permissions {
			if !denied[p] {
				kept = append(kept, p)
			}
		}
		c.User.Permissions = kept
	}
}

// boardsAllowedTo is boardsAllowedTo($permission): the boards on which the
// user may use a permission. Returns []int{0} for admins (all boards).
func (c *Ctx) boardsAllowedTo(permission string) []int {
	a := c.App

	// Administrators are all powerful, sorry.
	if c.User.IsAdmin {
		return []int{0}
	}

	// All groups the user is in except 'moderator'.
	var groups []string
	for _, g := range c.User.Groups {
		if g != 3 {
			groups = append(groups, itoa(g))
		}
	}

	// With no local permissions, there might be some other restrictions.
	maxAllowableMode := -1
	if a.SettingEmpty("permission_enable_by_board") && !inStrings(c.User.Permissions, "moderate_board") {
		switch permission {
		case "post_reply_own", "post_reply_any":
			maxAllowableMode = 3
		case "post_new":
			maxAllowableMode = 2
		case "poll_post":
			maxAllowableMode = 0
		}
	}

	bpBoard := "0"
	if !a.SettingEmpty("permission_enable_by_board") {
		bpBoard = "IIF(b.permission_mode = 1, b.ID_BOARD, 0)"
	}
	modeCond := ""
	if maxAllowableMode >= 0 {
		modeCond = `
			AND (mods.ID_MEMBER IS NOT NULL OR b.permission_mode <= ` + itoa(maxAllowableMode) + `)`
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.permission_mode, bp.addDeny
		FROM {$db_prefix}boards AS b, {$db_prefix}board_permissions AS bp
			LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = b.ID_BOARD AND mods.ID_MEMBER = ` + itoa(c.User.ID) + `)
		WHERE bp.ID_BOARD = ` + bpBoard + `
			AND bp.ID_GROUP IN (` + strings.Join(groups, ", ") + `, 3)
			AND bp.permission = '` + permission + `'` + modeCond + `
			AND (mods.ID_MEMBER IS NOT NULL OR bp.ID_GROUP != 3)`))
	if err != nil {
		return nil
	}
	defer rows.Close()

	var boards, denyBoards []int
	for rows.Next() {
		var board, mode, addDeny int
		rows.Scan(&board, &mode, &addDeny)
		if addDeny == 0 {
			denyBoards = append(denyBoards, board)
		} else {
			boards = append(boards, board)
		}
	}

	var out []int
	for _, b := range boards {
		denied := false
		for _, d := range denyBoards {
			if d == b {
				denied = true
				break
			}
		}
		if !denied {
			out = append(out, b)
		}
	}
	return out
}

func inStrings(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
