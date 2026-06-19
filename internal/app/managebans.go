package app

// Port of Sources/ManageBans.php: the ban admin (?action=ban) — BanList,
// BanEdit (ban groups + their triggers), BanEditTrigger (single trigger),
// BanBrowseTriggers, BanLog, plus ip2range/range2ip and updateBanMembers.
//
// IP triggers are stored as four octet ranges (ip_low1..ip_high4); a '*' octet
// maps to 0-255 and 'a-b' to that range. Deviation: the reverse-DNS hostname
// suggestion (host_from_ip) is not implemented — the hostname suggestion field
// is left blank; everything else (manual hostname bans, message/error IP
// gathering) is ported.

import (
	"regexp"
	"strings"
)

func init() {
	registerAction("ban", (*Ctx).Ban)
}

func (c *Ctx) Ban() {
	a := c.App
	c.isAllowedTo("manage_bans")
	c.adminIndex("ban_members")
	c.loadLanguage("ManageBans")

	subActions := map[string]bool{"add": true, "browse": true, "edittrigger": true, "edit": true, "list": true, "log": true}
	sa := c.REQUEST.Str("sa")
	if !subActions[sa] {
		sa = "list"
	}

	c.PageTitle = c.Txt("ban_title")

	scripturl := a.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("ban_title"), Help: "ban_members", Description: c.Txt("ban_description")}
	tabs.Tabs = append(tabs.Tabs,
		AdminTab{Title: c.Txt("ban_edit_list"), Description: c.Txt("ban_description"), Href: scripturl + "?action=ban;sa=list", IsSelected: sa == "list" || sa == "edit" || sa == "edittrigger"},
		AdminTab{Title: c.Txt("ban_add_new"), Description: c.Txt("ban_description"), Href: scripturl + "?action=ban;sa=add", IsSelected: sa == "add"},
		AdminTab{Title: c.Txt("ban_trigger_browse"), Description: c.Txt("ban_trigger_browse_description"), Href: scripturl + "?action=ban;sa=browse", IsSelected: sa == "browse"},
		AdminTab{Title: c.Txt("ban_log"), Description: c.Txt("ban_log_description"), Href: scripturl + "?action=ban;sa=log", IsSelected: sa == "log", IsLast: true},
	)
	c.AdminTabs = tabs

	switch sa {
	case "add", "edit":
		c.BanEdit()
	case "browse":
		c.BanBrowseTriggers()
	case "edittrigger":
		c.BanEditTrigger()
	case "log":
		c.BanLog()
	default:
		c.BanList()
	}
}

// ---- BanList ----

// BanColumn is one column header in the ban list.
type BanColumn struct {
	Key      string
	Width    string
	Label    string
	Sortable bool
	Selected bool
	Href     string
	Link     string
}

// BanRow is one ban group in the list.
type BanRow struct {
	ID         int
	Name       string
	Added      string
	Reason     string
	Notes      string
	Expires    string
	NumEntries int
}

// BanListCtx backs template_main.
type BanListCtx struct {
	Columns       []BanColumn
	Bans          []BanRow
	PageIndex     string
	SortBy        string
	SortDirection string
}

var banDateFmtRe = regexp.MustCompile(`%[AaBbCcDdeGghjmuYy](?:[^%]*%[AaBbCcDdeGghjmuYy])*`)

func (c *Ctx) BanList() {
	a := c.App
	c.PageTitle = c.Txt("ban_title")

	if c.POST.Has("removeBans") && c.POST.Arr("remove") != nil {
		c.checkSession("post", "", true)
		var ids []int
		c.POST.Arr("remove").Values(func(k string, v any) {
			s, _ := v.(string)
			ids = append(ids, atoi(s))
		})
		if len(ids) > 0 {
			in := joinInts(ids)
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}ban_groups WHERE ID_BAN_GROUP IN (` + in + `)`))
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}ban_items WHERE ID_BAN_GROUP IN (` + in + `)`))
			a.UpdateSettings(map[string]string{"banLastUpdated": itoa64(nowUnix())})
			c.updateBanMembers()
		}
	}

	// Sort methods (SQLite ordering syntax).
	sortMethods := map[string]map[string]string{
		"name":        {"down": "bg.name ASC", "up": "bg.name DESC"},
		"reason":      {"down": "LENGTH(bg.reason) > 0 DESC, bg.reason ASC", "up": "LENGTH(bg.reason) > 0 ASC, bg.reason DESC"},
		"notes":       {"down": "LENGTH(bg.notes) > 0 DESC, bg.notes ASC", "up": "LENGTH(bg.notes) > 0 ASC, bg.notes DESC"},
		"expires":     {"down": "(bg.expire_time IS NULL) DESC, bg.expire_time DESC", "up": "(bg.expire_time IS NULL) ASC, bg.expire_time ASC"},
		"num_entries": {"down": "num_entries DESC", "up": "num_entries ASC"},
		"added":       {"down": "bg.ban_time ASC", "up": "bg.ban_time DESC"},
	}

	page := &BanListCtx{}
	c.Page = page
	page.Columns = []BanColumn{
		{Key: "name", Width: "20%", Label: c.Txt("ban_name"), Sortable: true},
		{Key: "notes", Width: "20%", Label: c.Txt("ban_notes"), Sortable: true},
		{Key: "reason", Width: "20%", Label: c.Txt("ban_reason"), Sortable: true},
		{Key: "added", Width: "18%", Label: c.Txt("ban_added"), Sortable: true},
		{Key: "expires", Width: "20%", Label: c.Txt("ban_expires"), Sortable: true},
		{Key: "num_entries", Label: c.Txt("ban_triggers"), Sortable: true},
		{Key: "actions", Label: c.Txt("ban_actions"), Sortable: false},
	}

	sort := c.REQUEST.Str("sort")
	if _, ok := sortMethods[sort]; !ok {
		sort = "name"
	}
	desc := c.REQUEST.Has("desc")
	scripturl := a.ScriptURL
	for i := range page.Columns {
		col := &page.Columns[i]
		col.Selected = col.Key == sort
		col.Href = scripturl + "?action=ban;sort=" + col.Key
		if !desc && col.Key == sort {
			col.Href += ";desc"
		}
		col.Link = `<a href="` + col.Href + `">` + col.Label + `</a>`
	}
	page.SortBy = sort
	if desc {
		page.SortDirection = "up"
	} else {
		page.SortDirection = "down"
	}

	var total int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}ban_groups`)).Scan(&total)

	descPart := ""
	if desc {
		descPart = ";desc"
	}
	start := c.REQUEST.Int("start")
	page.PageIndex, start = c.constructPageIndex(scripturl+"?action=ban;sort="+sort+descPart, start, total, 20, false)

	// Date format: keep only the date specifiers of the user time format.
	userFmt := c.User.TimeFormat
	if userFmt == "" {
		userFmt = a.Setting("time_format")
	}
	banFmt := ""
	if m := banDateFmtRe.FindString(userFmt); m != "" {
		banFmt = m
	}

	rows, err := a.DB.Query(a.Q(`
		SELECT bg.ID_BAN_GROUP, bg.name, bg.ban_time, bg.expire_time, bg.reason, bg.notes, COUNT(bi.ID_BAN) AS num_entries
		FROM {$db_prefix}ban_groups AS bg
			LEFT JOIN {$db_prefix}ban_items AS bi ON (bi.ID_BAN_GROUP = bg.ID_BAN_GROUP)
		GROUP BY bg.ID_BAN_GROUP
		ORDER BY `+sortMethods[sort][page.SortDirection]+`
		LIMIT ? OFFSET ?`), 20, start)
	if err == nil {
		now := nowUnix()
		for rows.Next() {
			var id, banTime, numEntries int
			var expire *int64
			var name, reason, notes string
			rows.Scan(&id, &name, &banTime, &expire, &reason, &notes, &numEntries)
			expires := c.Txt("never")
			if expire != nil {
				if *expire < now {
					expires = `<span style="color: red">` + c.Txt("ban_expired") + `</span>`
				} else {
					days := (*expire - now + 86399) / 86400
					expires = itoa64(days) + "&nbsp;" + c.Txt("ban_days")
				}
			}
			page.Bans = append(page.Bans, BanRow{
				ID:         id,
				Name:       name,
				Added:      c.timeformatFmt(int64(banTime), true, banFmt),
				Reason:     reason,
				Notes:      notes,
				Expires:    expires,
				NumEntries: numEntries,
			})
		}
		rows.Close()
	}

	c.SubTemplate = templateBanMain
}

// ---- BanEdit ----

// BanGroup is $context['ban'].
type BanGroup struct {
	ID              int
	Name            string
	ExpirationState string // never | still_active_but_we_re_counting_the_days | expired
	ExpirationDays  int64
	Reason          string
	Notes           string
	CannotAccess    bool
	CannotPost      bool
	CannotRegister  bool
	CannotLogin     bool
	IsNew           bool
}

// BanItem is one trigger row in $context['ban_items'] (ban_edit view).
type BanItem struct {
	ID       int
	Hits     int
	Type     string // ip | hostname | email | user
	IP       string
	Hostname string
	Email    string
	UserLink string
}

// BanMember is a suggested member to ban.
type BanMember struct {
	ID   int
	Name string
	Link string
}

// BanSuggestions is $context['ban_suggestions'] (only on new bans from ?u=).
type BanSuggestions struct {
	MainIP     string
	Hostname   string
	Email      string
	Member     BanMember
	HasMember  bool
	MessageIPs []string
	ErrorIPs   []string
}

// BanEditCtx backs template_ban_edit.
type BanEditCtx struct {
	Ban         BanGroup
	BanItems    []BanItem
	Suggestions *BanSuggestions
}

var hostnameBanRe = regexp.MustCompile(`[^\w.\-*]`)
var emailBanRe = regexp.MustCompile(`[^\w.\-*@]`)
var ipv4Re = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

func (c *Ctx) BanEdit() {
	a := c.App
	bg := c.REQUEST.Int("bg")

	// Adding or editing a single trigger from the ban_edit page.
	if c.POST.Has("add_new_trigger") || c.POST.Has("edit_trigger") {
		c.checkSession("post", "", true)
		newBan := c.POST.Has("add_new_trigger")
		c.banSaveTrigger(bg, c.REQUEST.Int("bi"), newBan)
		a.UpdateSettings(map[string]string{"banLastUpdated": itoa64(nowUnix())})
		c.updateBanMembers()
	} else if c.POST.Has("remove_selection") && c.POST.Arr("ban_items") != nil {
		// Remove selected triggers.
		c.checkSession("post", "", true)
		var ids []int
		c.POST.Arr("ban_items").Values(func(k string, v any) {
			s, _ := v.(string)
			ids = append(ids, atoi(s))
		})
		if len(ids) > 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}ban_items WHERE ID_BAN IN (`+joinInts(ids)+`) AND ID_BAN_GROUP = ?`), bg)
			a.UpdateSettings(map[string]string{"banLastUpdated": itoa64(nowUnix())})
			c.updateBanMembers()
		}
	} else if c.POST.Has("modify_ban") || c.POST.Has("add_ban") {
		c.checkSession("post", "", true)
		bg = c.banSaveGroup(bg)
		a.UpdateSettings(map[string]string{"banLastUpdated": itoa64(nowUnix())})
		c.updateBanMembers()
	}

	page := &BanEditCtx{}
	c.Page = page

	if bg != 0 {
		c.banLoadGroup(bg, page)
	} else {
		c.banNewGroup(page)
	}

	c.SubTemplate = templateBanEdit
}

// banSaveTrigger handles add_new_trigger / edit_trigger.
func (c *Ctx) banSaveTrigger(bg, bi int, newBan bool) {
	a := c.App
	bantype := c.POST.Str("bantype")

	// Column values to set: ip quad, hostname, email, member.
	var lo, hi [4]int
	hostname, email := "", ""
	member := 0

	switch bantype {
	case "ip_ban":
		parts := ip2range(c.POST.Str("ip"))
		if len(parts) != 4 {
			c.fatalLangError("invalid_ip", false)
		}
		for i := 0; i < 4; i++ {
			lo[i], hi[i] = parts[i][0], parts[i][1]
		}
	case "hostname_ban":
		h := c.POST.Str("hostname")
		if hostnameBanRe.MatchString(h) {
			c.fatalLangError("invalid_hostname", false)
		}
		hostname = strings.ReplaceAll(h, "*", "%")
	case "email_ban":
		e := c.POST.Str("email")
		if emailBanRe.MatchString(e) {
			c.fatalLangError("invalid_email", false)
		}
		email = strings.ToLower(strings.ReplaceAll(e, "*", "%"))
		var found int
		a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER FROM {$db_prefix}members
			WHERE (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups)) AND emailAddress LIKE ?
			LIMIT 1`), email).Scan(&found)
		if found != 0 {
			c.fatalLangError("no_ban_admin", true)
		}
	case "user_ban":
		user := Htmlspecialchars(c.POST.Str("user"))
		var isAdmin int
		err := a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER, (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups)) AS isAdmin
			FROM {$db_prefix}members WHERE memberName = ? OR realName = ? LIMIT 1`), user, user).Scan(&member, &isAdmin)
		if err != nil {
			c.fatalLangError("invalid_username", false)
		}
		if isAdmin != 0 {
			c.fatalLangError("no_ban_admin", true)
		}
	default:
		c.fatalLangError("no_bantype_selected", false)
	}

	if newBan {
		a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}ban_items
				(ID_BAN_GROUP, ip_low1, ip_high1, ip_low2, ip_high2, ip_low3, ip_high3, ip_low4, ip_high4, hostname, email_address, ID_MEMBER)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
			bg, lo[0], hi[0], lo[1], hi[1], lo[2], hi[2], lo[3], hi[3], hostname, email, member)
	} else {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}ban_items SET
				ip_low1 = ?, ip_high1 = ?, ip_low2 = ?, ip_high2 = ?,
				ip_low3 = ?, ip_high3 = ?, ip_low4 = ?, ip_high4 = ?,
				hostname = ?, email_address = ?, ID_MEMBER = ?
			WHERE ID_BAN = ? AND ID_BAN_GROUP = ?`),
			lo[0], hi[0], lo[1], hi[1], lo[2], hi[2], lo[3], hi[3], hostname, email, member, bi, bg)
	}

	c.logAction("ban", map[string]any{"new": newBan, "type": bantype})
}

// banSaveGroup handles modify_ban / add_ban and returns the (possibly new) bg.
func (c *Ctx) banSaveGroup(bg int) int {
	a := c.App
	addBan := c.POST.Has("add_ban")
	name := c.POST.Str("ban_name")
	if name == "" {
		c.fatalError(c.Txt("ban_name_empty"), false)
	}

	// Reject a duplicate ban name.
	var dup int
	if addBan {
		a.DB.QueryRow(a.Q(`SELECT ID_BAN_GROUP FROM {$db_prefix}ban_groups WHERE name = ? LIMIT 1`), name).Scan(&dup)
	} else {
		a.DB.QueryRow(a.Q(`SELECT ID_BAN_GROUP FROM {$db_prefix}ban_groups WHERE name = ? AND ID_BAN_GROUP != ? LIMIT 1`), name, bg).Scan(&dup)
	}
	if dup != 0 {
		c.fatalError(phpSprintf(c.Txt("ban_name_exists"), Htmlspecialchars(name)), false)
	}

	reason := Htmlspecialchars(c.POST.Str("reason"))
	notes := Htmlspecialchars(c.POST.Str("notes"))
	notes = strings.NewReplacer("\r", "", "\n", "<br />", "  ", "&nbsp; ").Replace(notes)

	// expiration -> a SQL fragment.
	now := nowUnix()
	var expirationExpr string
	switch c.POST.Str("expiration") {
	case "never":
		expirationExpr = "NULL"
	case "expired":
		expirationExpr = "0"
	default:
		if c.POST.Str("expire_date") != c.POST.Str("old_expire") {
			expirationExpr = itoa64(now + 24*60*60*int64(c.POST.Int("expire_date")))
		} else {
			expirationExpr = "expire_time"
		}
	}

	fullBan := c.POST.Has("full_ban")
	b2i := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	cannotAccess := b2i(fullBan)
	cannotPost := b2i(!fullBan && c.POST.Has("cannot_post"))
	cannotRegister := b2i(!fullBan && c.POST.Has("cannot_register"))
	cannotLogin := b2i(!fullBan && c.POST.Has("cannot_login"))

	if addBan {
		triggers := c.banCollectTriggers()
		res, _ := a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}ban_groups
				(name, ban_time, expire_time, cannot_access, cannot_register, cannot_post, cannot_login, reason, notes)
			VALUES (substr(?, 1, 20), ?, `+expirationExpr+`, ?, ?, ?, ?, substr(?, 1, 255), substr(?, 1, 65534))`),
			name, now, cannotAccess, cannotRegister, cannotPost, cannotLogin, reason, notes)
		id64, _ := res.LastInsertId()
		bg = int(id64)
		for _, t := range triggers {
			a.DB.Exec(a.Q(`
				INSERT INTO {$db_prefix}ban_items
					(ID_BAN_GROUP, ip_low1, ip_high1, ip_low2, ip_high2, ip_low3, ip_high3, ip_low4, ip_high4, hostname, email_address, ID_MEMBER)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
				bg, t.lo[0], t.hi[0], t.lo[1], t.hi[1], t.lo[2], t.hi[2], t.lo[3], t.hi[3], t.hostname, t.email, t.member)
		}
	} else {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}ban_groups SET
				name = ?, reason = ?, notes = ?, expire_time = `+expirationExpr+`,
				cannot_access = ?, cannot_post = ?, cannot_register = ?, cannot_login = ?
			WHERE ID_BAN_GROUP = ?`),
			name, reason, notes, cannotAccess, cannotPost, cannotRegister, cannotLogin, bg)
	}
	return bg
}

// banTrigger is a resolved trigger row for a freshly-added ban group.
type banTrigger struct {
	lo, hi   [4]int
	hostname string
	email    string
	member   int
}

// banCollectTriggers reads the ban_suggestion[] checkboxes when adding a ban.
func (c *Ctx) banCollectTriggers() []banTrigger {
	a := c.App
	arr := c.POST.Arr("ban_suggestion")
	if arr == nil {
		return nil
	}
	sel := map[string]bool{}
	arr.Values(func(k string, v any) {
		if s, ok := v.(string); ok {
			sel[s] = true
		}
	})

	var out []banTrigger
	mainIP := c.POST.Str("main_ip")
	if sel["main_ip"] && mainIP != "" {
		parts := ip2range(mainIP)
		if len(parts) != 4 {
			c.fatalLangError("invalid_ip", false)
		}
		out = append(out, ipTrigger(parts))
	}
	if sel["hostname"] && c.POST.Str("hostname") != "" {
		h := c.POST.Str("hostname")
		if hostnameBanRe.MatchString(h) {
			c.fatalLangError("invalid_hostname", false)
		}
		out = append(out, banTrigger{hostname: strings.ReplaceAll(h, "*", "%")})
	}
	if sel["email"] && c.POST.Str("email") != "" {
		e := c.POST.Str("email")
		if emailBanRe.MatchString(e) {
			c.fatalLangError("invalid_email", false)
		}
		out = append(out, banTrigger{email: strings.ToLower(strings.ReplaceAll(e, "*", "%"))})
	}
	if sel["user"] && (c.POST.Str("bannedUser") != "" || c.POST.Str("user") != "") {
		member := c.POST.Int("bannedUser")
		if member == 0 {
			user := Htmlspecialchars(c.POST.Str("user"))
			var isAdmin int
			err := a.DB.QueryRow(a.Q(`
				SELECT ID_MEMBER, (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups)) AS isAdmin
				FROM {$db_prefix}members WHERE memberName = ? OR realName = ? LIMIT 1`), user, user).Scan(&member, &isAdmin)
			if err != nil {
				c.fatalLangError("invalid_username", false)
			}
			if isAdmin != 0 {
				c.fatalLangError("no_ban_admin", true)
			}
		}
		out = append(out, banTrigger{member: member})
	}

	// Additional IPs gathered from messages/errors.
	if ips := arr.Arr("ips"); ips != nil {
		seen := map[string]bool{}
		ips.Values(func(k string, v any) {
			ip, _ := v.(string)
			if ip == "" || ip == mainIP || seen[ip] {
				return
			}
			seen[ip] = true
			parts := ip2range(ip)
			if len(parts) != 4 {
				c.fatalLangError("invalid_ip", false)
			}
			out = append(out, ipTrigger(parts))
		})
	}
	return out
}

func ipTrigger(parts [][2]int) banTrigger {
	var t banTrigger
	for i := 0; i < 4; i++ {
		t.lo[i], t.hi[i] = parts[i][0], parts[i][1]
	}
	return t
}

// banLoadGroup loads an existing ban group + its triggers for the edit page.
func (c *Ctx) banLoadGroup(bg int, page *BanEditCtx) {
	a := c.App
	scripturl := a.ScriptURL
	rows, err := a.DB.Query(a.Q(`
		SELECT
			bi.ID_BAN, bi.hostname, bi.email_address, bi.hits,
			bi.ip_low1, bi.ip_high1, bi.ip_low2, bi.ip_high2, bi.ip_low3, bi.ip_high3, bi.ip_low4, bi.ip_high4,
			bg.ID_BAN_GROUP, bg.name, bg.ban_time, bg.expire_time, bg.reason, bg.notes,
			bg.cannot_access, bg.cannot_register, bg.cannot_login, bg.cannot_post,
			IFNULL(mem.ID_MEMBER, 0) AS mem_id, IFNULL(mem.realName, '') AS realName
		FROM {$db_prefix}ban_groups AS bg
			LEFT JOIN {$db_prefix}ban_items AS bi ON (bi.ID_BAN_GROUP = bg.ID_BAN_GROUP)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = bi.ID_MEMBER)
		WHERE bg.ID_BAN_GROUP = ?`), bg)
	if err != nil {
		c.fatalLangError("ban_not_found", false)
	}
	defer rows.Close()
	now := nowUnix()
	loaded := false
	var orphans []int
	for rows.Next() {
		var banID, hits int
		var hostname, email string
		var lo, hi [4]int
		var grpID, banTime, cannotAccess, cannotRegister, cannotLogin, cannotPost int
		var expire *int64
		var name, reason, notes string
		var memID int
		var realName string
		var banIDNull *int
		rows.Scan(&banIDNull, &hostname, &email, &hits,
			&lo[0], &hi[0], &lo[1], &hi[1], &lo[2], &hi[2], &lo[3], &hi[3],
			&grpID, &name, &banTime, &expire, &reason, &notes,
			&cannotAccess, &cannotRegister, &cannotLogin, &cannotPost,
			&memID, &realName)
		if banIDNull != nil {
			banID = *banIDNull
		}

		if !loaded {
			loaded = true
			page.Ban = BanGroup{
				ID:             grpID,
				Name:           name,
				Reason:         reason,
				Notes:          notes,
				CannotAccess:   cannotAccess != 0,
				CannotPost:     cannotPost != 0,
				CannotRegister: cannotRegister != 0,
				CannotLogin:    cannotLogin != 0,
				IsNew:          false,
			}
			if expire == nil {
				page.Ban.ExpirationState = "never"
			} else if *expire < now {
				page.Ban.ExpirationState = "expired"
			} else {
				page.Ban.ExpirationState = "still_active_but_we_re_counting_the_days"
				page.Ban.ExpirationDays = (*expire - now) / 86400
			}
		}

		if banIDNull == nil || banID == 0 {
			continue
		}
		item := BanItem{ID: banID, Hits: hits}
		switch {
		case hi[0] != 0:
			item.Type = "ip"
			item.IP = range2ip(lo[:], hi[:])
		case hostname != "":
			item.Type = "hostname"
			item.Hostname = strings.ReplaceAll(hostname, "%", "*")
		case email != "":
			item.Type = "email"
			item.Email = strings.ReplaceAll(email, "%", "*")
		case memID != 0:
			item.Type = "user"
			item.UserLink = `<a href="` + scripturl + `?action=profile;u=` + itoa(memID) + `">` + realName + `</a>`
		default:
			orphans = append(orphans, banID)
			continue
		}
		page.BanItems = append(page.BanItems, item)
	}
	if !loaded {
		c.fatalLangError("ban_not_found", false)
	}
	for _, id := range orphans {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}ban_items WHERE ID_BAN = ?`), id)
	}
}

// banNewGroup builds the blank form, optionally pre-filled from ?u=.
func (c *Ctx) banNewGroup(page *BanEditCtx) {
	a := c.App
	scripturl := a.ScriptURL
	page.Ban = BanGroup{IsNew: true, ExpirationState: "never", CannotAccess: true}
	sug := &BanSuggestions{}
	page.Suggestions = sug

	u := c.REQUEST.Int("u")
	if u == 0 {
		return
	}
	var id int
	var realName, mainIP, email string
	err := a.DB.QueryRow(a.Q(`SELECT ID_MEMBER, realName, memberIP, emailAddress FROM {$db_prefix}members WHERE ID_MEMBER = ? LIMIT 1`), u).Scan(&id, &realName, &mainIP, &email)
	if err != nil || id == 0 {
		return
	}
	sug.Member = BanMember{ID: id, Name: realName}
	sug.MainIP = mainIP
	sug.Email = email
	sug.HasMember = true
	sug.Member.Link = `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + realName + `</a>`
	page.Ban.Name = realName

	// Hostname suggestion (reverse DNS) is not implemented in this port.

	// Additional IPs from this member's posts and errors.
	if rows, err := a.DB.Query(a.Q(`
		SELECT DISTINCT posterIP FROM {$db_prefix}messages
		WHERE ID_MEMBER = ? AND posterIP != '' ORDER BY posterIP`), u); err == nil {
		for rows.Next() {
			var ip string
			rows.Scan(&ip)
			if ipv4Re.MatchString(ip) {
				sug.MessageIPs = append(sug.MessageIPs, ip)
			}
		}
		rows.Close()
	}
	if rows, err := a.DB.Query(a.Q(`
		SELECT DISTINCT ip FROM {$db_prefix}log_errors
		WHERE ID_MEMBER = ? AND ip != '' ORDER BY ip`), u); err == nil {
		for rows.Next() {
			var ip string
			rows.Scan(&ip)
			if ipv4Re.MatchString(ip) {
				sug.ErrorIPs = append(sug.ErrorIPs, ip)
			}
		}
		rows.Close()
	}
	c.loadLanguage("Profile")
}

// ---- BanEditTrigger ----

// BanTriggerField is one of the IP/host/email/user radios in the trigger form.
type BanTriggerField struct {
	Value    string
	Selected bool
}

// BanEditTriggerCtx backs template_ban_edit_trigger.
type BanEditTriggerCtx struct {
	ID         int
	Group      int
	IP         BanTriggerField
	Hostname   BanTriggerField
	Email      BanTriggerField
	BannedUser BanTriggerField
	IsNew      bool
}

func (c *Ctx) BanEditTrigger() {
	a := c.App
	bg := c.REQUEST.Int("bg")
	if bg == 0 {
		c.fatalLangError("ban_not_found", false)
	}
	page := &BanEditTriggerCtx{Group: bg}
	c.Page = page
	c.SubTemplate = templateBanEditTrigger

	bi := c.REQUEST.Int("bi")
	if bi == 0 {
		page.IsNew = true
		page.IP.Selected = true
		return
	}

	var lo, hi [4]int
	var hostname, email, memberName, realName string
	err := a.DB.QueryRow(a.Q(`
		SELECT bi.hostname, bi.email_address,
			bi.ip_low1, bi.ip_high1, bi.ip_low2, bi.ip_high2, bi.ip_low3, bi.ip_high3, bi.ip_low4, bi.ip_high4,
			IFNULL(mem.memberName, ''), IFNULL(mem.realName, '')
		FROM {$db_prefix}ban_items AS bi
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = bi.ID_MEMBER)
		WHERE bi.ID_BAN = ? AND bi.ID_BAN_GROUP = ? LIMIT 1`), bi, bg).Scan(
		&hostname, &email, &lo[0], &hi[0], &lo[1], &hi[1], &lo[2], &hi[2], &lo[3], &hi[3], &memberName, &realName)
	if err != nil {
		c.fatalLangError("ban_not_found", false)
	}
	_ = realName
	page.ID = bi
	if lo[0] != 0 {
		page.IP = BanTriggerField{Value: range2ip(lo[:], hi[:]), Selected: true}
	}
	page.Hostname = BanTriggerField{Value: strings.ReplaceAll(hostname, "%", "*"), Selected: hostname != ""}
	page.Email = BanTriggerField{Value: strings.ReplaceAll(email, "%", "*"), Selected: email != ""}
	page.BannedUser = BanTriggerField{Value: memberName, Selected: memberName != ""}
}

// ---- BanBrowseTriggers ----

// BanBrowseItem is one row in the trigger browser.
type BanBrowseItem struct {
	ID        int
	Hits      int
	Entity    string
	GroupName string
	GroupHref string
	GroupLink string
}

// BanBrowseCtx backs template_browse_triggers.
type BanBrowseCtx struct {
	SelectedEntity string
	BanItems       []BanBrowseItem
	PageIndex      string
	Start          int
}

func (c *Ctx) BanBrowseTriggers() {
	a := c.App
	scripturl := a.ScriptURL

	if c.POST.Has("remove_triggers") && c.POST.Arr("remove") != nil {
		c.checkSession("post", "", true)
		var ids []int
		c.POST.Arr("remove").Values(func(k string, v any) {
			s, _ := v.(string)
			ids = append(ids, atoi(s))
		})
		if len(ids) > 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}ban_items WHERE ID_BAN IN (` + joinInts(ids) + `)`))
			if c.REQUEST.Str("entity") == "member" {
				c.updateBanMembers()
			}
			a.UpdateSettings(map[string]string{"banLastUpdated": itoa64(nowUnix())})
		}
	}

	type qspec struct{ sel, where, orderby string }
	query := map[string]qspec{
		"ip":       {"bi.ip_low1, bi.ip_high1, bi.ip_low2, bi.ip_high2, bi.ip_low3, bi.ip_high3, bi.ip_low4, bi.ip_high4", "bi.ip_low1 > 0", "bi.ip_low1, bi.ip_high1, bi.ip_low2, bi.ip_high2, bi.ip_low3, bi.ip_high3, bi.ip_low4, bi.ip_high4"},
		"hostname": {"bi.hostname", "bi.hostname != ''", "bi.hostname"},
		"email":    {"bi.email_address", "bi.email_address != ''", "bi.email_address"},
		"member":   {"mem.ID_MEMBER, mem.realName", "mem.ID_MEMBER = bi.ID_MEMBER", "mem.realName"},
	}

	entity := c.REQUEST.Str("entity")
	if _, ok := query[entity]; !ok {
		entity = "ip"
	}
	page := &BanBrowseCtx{SelectedEntity: entity}
	c.Page = page

	memJoin := ""
	if entity == "member" {
		memJoin = ", " + a.Q("{$db_prefix}") + "members AS mem"
	}

	var total int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}ban_items AS bi` + memJoin + ` WHERE ` + query[entity].where)).Scan(&total)

	perPage := a.SettingInt("defaultMaxMessages")
	if perPage == 0 {
		perPage = 30
	}
	start := c.REQUEST.Int("start")
	page.PageIndex, start = c.constructPageIndex(scripturl+"?action=ban;sa=browse;entity="+entity, start, total, perPage, false)
	page.Start = start

	if total == 0 {
		c.SubTemplate = templateBanBrowseTriggers
		return
	}

	rows, err := a.DB.Query(a.Q(`
		SELECT bi.ID_BAN, `+query[entity].sel+`, bi.hits, bg.ID_BAN_GROUP, bg.name
		FROM {$db_prefix}ban_items AS bi, {$db_prefix}ban_groups AS bg`+memJoin+`
		WHERE `+query[entity].where+` AND bg.ID_BAN_GROUP = bi.ID_BAN_GROUP
		ORDER BY `+query[entity].orderby+`
		LIMIT ? OFFSET ?`), perPage, start)
	if err == nil {
		for rows.Next() {
			var item BanBrowseItem
			switch entity {
			case "ip":
				var banID int
				var lo, hi [4]int
				var hits, grpID int
				var name string
				rows.Scan(&banID, &lo[0], &hi[0], &lo[1], &hi[1], &lo[2], &hi[2], &lo[3], &hi[3], &hits, &grpID, &name)
				item = BanBrowseItem{ID: banID, Hits: hits, Entity: range2ip(lo[:], hi[:]), GroupName: name}
				item.fillGroup(scripturl, grpID, name)
			case "hostname", "email":
				var banID, hits, grpID int
				var val, name string
				rows.Scan(&banID, &val, &hits, &grpID, &name)
				item = BanBrowseItem{ID: banID, Hits: hits, Entity: strings.ReplaceAll(val, "%", "*")}
				item.fillGroup(scripturl, grpID, name)
			default: // member
				var banID, memID, hits, grpID int
				var realName, name string
				rows.Scan(&banID, &memID, &realName, &hits, &grpID, &name)
				link := `<a href="` + scripturl + `?action=profile;u=` + itoa(memID) + `">` + realName + `</a>`
				item = BanBrowseItem{ID: banID, Hits: hits, Entity: link}
				item.fillGroup(scripturl, grpID, name)
			}
			page.BanItems = append(page.BanItems, item)
		}
		rows.Close()
	}

	c.SubTemplate = templateBanBrowseTriggers
}

func (it *BanBrowseItem) fillGroup(scripturl string, grpID int, name string) {
	it.GroupName = name
	it.GroupHref = scripturl + "?action=ban;sa=edit;bg=" + itoa(grpID)
	it.GroupLink = `<a href="` + it.GroupHref + `">` + name + `</a>`
}

// ---- BanLog ----

// BanLogRow is one log_banned entry.
type BanLogRow struct {
	ID         int
	MemberID   int
	MemberLink string
	IP         string
	Email      string
	Date       string
}

// BanLogCtx backs template_ban_log.
type BanLogCtx struct {
	Entries       []BanLogRow
	Sort          string
	SortDirection string
	Start         int
	PageIndex     string
}

func (c *Ctx) BanLog() {
	a := c.App
	scripturl := a.ScriptURL

	sortColumns := map[string]string{
		"name":  "mem.realName",
		"ip":    "lb.ip",
		"email": "lb.email",
		"date":  "lb.logTime",
	}
	entriesPerPage := 30

	if c.POST.Has("removeAll") || (c.POST.Has("removeSelected") && c.POST.Arr("remove") != nil) {
		c.checkSession("post", "", true)
		if c.POST.Has("removeAll") {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_banned`))
		} else {
			var ids []int
			c.POST.Arr("remove").Values(func(k string, v any) {
				s, _ := v.(string)
				ids = append(ids, atoi(s))
			})
			if len(ids) > 0 {
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_banned WHERE ID_BAN_LOG IN (` + joinInts(ids) + `)`))
			}
		}
	}

	var total int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_banned`)).Scan(&total)

	sort := c.REQUEST.Str("sort")
	desc := c.REQUEST.Has("desc")
	if _, ok := sortColumns[sort]; !ok || sort == "" {
		sort = "date"
		desc = true
	}

	page := &BanLogCtx{Sort: sort}
	c.Page = page
	if desc {
		page.SortDirection = "down"
	} else {
		page.SortDirection = "up"
	}
	descPart := ""
	if desc {
		descPart = ";desc"
	}
	start := c.REQUEST.Int("start")
	page.PageIndex, start = c.constructPageIndex(scripturl+"?action=ban;sa=log;sort="+sort+descPart, start, total, entriesPerPage, false)
	page.Start = start

	dir := ""
	if desc {
		dir = " DESC"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT lb.ID_BAN_LOG, lb.ID_MEMBER, IFNULL(lb.ip, '-') AS ip, IFNULL(lb.email, '-') AS email, lb.logTime, IFNULL(mem.realName, '') AS realName
		FROM {$db_prefix}log_banned AS lb
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = lb.ID_MEMBER)
		ORDER BY `+sortColumns[sort]+dir+`
		LIMIT ? OFFSET ?`), entriesPerPage, start)
	if err == nil {
		for rows.Next() {
			var id, memID int
			var ip, email, realName string
			var logTime int64
			rows.Scan(&id, &memID, &ip, &email, &logTime, &realName)
			page.Entries = append(page.Entries, BanLogRow{
				ID:         id,
				MemberID:   memID,
				MemberLink: `<a href="` + scripturl + `?action=profile;u=` + itoa(memID) + `">` + realName + `</a>`,
				IP:         ip,
				Email:      email,
				Date:       c.timeformat(logTime),
			})
		}
		rows.Close()
	}

	c.SubTemplate = templateBanLog
}

// ---- IP range helpers ----

// range2ip is range2ip($low, $high): turn two octet quads into a display IP.
func range2ip(low, high []int) string {
	if len(low) != 4 || len(high) != 4 {
		return ""
	}
	parts := make([]string, 4)
	for i := 0; i < 4; i++ {
		if low[i] == high[i] {
			parts[i] = itoa(low[i])
		} else if low[i] == 0 && high[i] == 255 {
			parts[i] = "*"
		} else {
			parts[i] = itoa(low[i]) + "-" + itoa(high[i])
		}
	}
	if low[0] == 255 && low[1] == 255 && low[2] == 255 && low[3] == 255 &&
		high[0] == 255 && high[1] == 255 && high[2] == 255 && high[3] == 255 {
		return "unknown"
	}
	return strings.Join(parts, ".")
}

var ipRangePartRe = regexp.MustCompile(`^(\d{1,3})\-(\d{1,3})$`)

// ip2range is ip2range($fullip): parse an IP (with *, a-b octets) to ranges.
// Returns 4 [low,high] pairs, or fewer (invalid) if any octet is unparseable.
func ip2range(fullip string) [][2]int {
	if fullip == "unknown" {
		fullip = "255.255.255.255"
	}
	ipParts := strings.Split(fullip, ".")
	if len(ipParts) != 4 {
		return nil
	}
	var out [][2]int
	for i := 0; i < 4; i++ {
		p := ipParts[i]
		if p == "*" {
			out = append(out, [2]int{0, 255})
		} else if m := ipRangePartRe.FindStringSubmatch(p); m != nil {
			out = append(out, [2]int{atoi(m[1]), atoi(m[2])})
		} else if isAllDigits(p) && p != "" {
			out = append(out, [2]int{atoi(p), atoi(p)})
		}
		// else: skip (leaves len < 4 -> treated as invalid by callers)
	}
	return out
}

// updateBanMembers syncs members.is_activated with the active access bans.
func (c *Ctx) updateBanMembers() {
	a := c.App
	now := nowUnix()

	updates := map[int][]int{} // new_value -> member ids
	var newMembers []int

	// Members that should be marked banned but aren't yet.
	if rows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER, mem.is_activated + 10 AS new_value
		FROM {$db_prefix}ban_groups AS bg, {$db_prefix}ban_items AS bi, {$db_prefix}members AS mem
		WHERE bg.ID_BAN_GROUP = bi.ID_BAN_GROUP
			AND bg.cannot_access = 1
			AND (bg.expire_time IS NULL OR bg.expire_time > ?)
			AND (mem.ID_MEMBER = bi.ID_MEMBER OR (bi.email_address != '' AND mem.emailAddress LIKE bi.email_address))
			AND mem.is_activated < 10`), now); err == nil {
		for rows.Next() {
			var id, newVal int
			rows.Scan(&id, &newVal)
			updates[newVal] = append(updates[newVal], id)
			newMembers = append(newMembers, id)
		}
		rows.Close()
	}

	if len(newMembers) > 0 {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online WHERE ID_MEMBER IN (` + joinInts(newMembers) + `)`))
	}

	// Members wrongly marked as banned.
	if rows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER, mem.is_activated - 10 AS new_value
		FROM {$db_prefix}members AS mem
			LEFT JOIN {$db_prefix}ban_items AS bi ON (bi.ID_MEMBER = mem.ID_MEMBER OR (bi.email_address != '' AND mem.emailAddress LIKE bi.email_address))
			LEFT JOIN {$db_prefix}ban_groups AS bg ON (bg.ID_BAN_GROUP = bi.ID_BAN_GROUP AND bg.cannot_access = 1 AND (bg.expire_time IS NULL OR bg.expire_time > ?))
		WHERE (bi.ID_BAN IS NULL OR bg.ID_BAN_GROUP IS NULL)
			AND mem.is_activated >= 10`), now); err == nil {
		for rows.Next() {
			var id, newVal int
			rows.Scan(&id, &newVal)
			updates[newVal] = append(updates[newVal], id)
		}
		rows.Close()
	}

	for newStatus, members := range updates {
		if len(members) == 0 {
			continue
		}
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET is_activated = ? WHERE ID_MEMBER IN (`+joinInts(members)+`)`), newStatus)
	}

	c.updateStatsMemberRecount()
}
