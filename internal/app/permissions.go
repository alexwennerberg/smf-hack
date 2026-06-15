package app

// Port of Sources/ManagePermissions.php: the permission center
// (?action=permissions) — the membergroup permission matrix, per-group editing,
// quick actions, predefined levels, and the shared inline-permission helpers
// (init/save/theme_inline_permissions) used by other admin modules. Board-level
// permissions (PermissionByBoard/SwitchBoard/quickboard) are gated behind
// permission_enable_by_board, which is off by default in this port; those
// sub-actions redirect to the index (documented).

import "strings"

func init() {
	registerAction("permissions", (*Ctx).ModifyPermissions)
}

// ---- the static permission definition (loadAllPermissions data) ----

type permDef struct {
	id        string
	hasOwnAny bool
}
type permGroupDef struct {
	id    string
	perms []permDef
}

var permListMembergroup = []permGroupDef{
	{"general", []permDef{{"view_stats", false}, {"view_mlist", false}, {"who_view", false}, {"search_posts", false}, {"karma_edit", false}}},
	{"pm", []permDef{{"pm_read", false}, {"pm_send", false}}},
	{"maintenance", []permDef{{"admin_forum", false}, {"manage_boards", false}, {"manage_attachments", false}, {"manage_smileys", false}, {"edit_news", false}}},
	{"member_admin", []permDef{{"moderate_forum", false}, {"manage_membergroups", false}, {"manage_permissions", false}, {"manage_bans", false}, {"send_mail", false}}},
	{"profile", []permDef{{"profile_view", true}, {"profile_identity", true}, {"profile_extra", true}, {"profile_title", true}, {"profile_remove", true}, {"profile_server_avatar", false}, {"profile_upload_avatar", false}, {"profile_remote_avatar", false}}},
}

var permListBoard = []permGroupDef{
	{"general_board", []permDef{{"moderate_board", false}}},
	{"topic", []permDef{{"post_new", false}, {"merge_any", false}, {"split_any", false}, {"send_topic", false}, {"make_sticky", false}, {"move", true}, {"lock", true}, {"remove", true}, {"post_reply", true}, {"modify_replies", false}, {"delete_replies", false}, {"announce_topic", false}}},
	{"post", []permDef{{"delete", true}, {"modify", true}, {"report_any", false}}},
	{"poll", []permDef{{"poll_view", false}, {"poll_vote", false}, {"poll_post", false}, {"poll_add", true}, {"poll_edit", true}, {"poll_lock", true}, {"poll_remove", true}}},
	{"notification", []permDef{{"mark_any_notify", false}, {"mark_notify", false}}},
	{"attachment", []permDef{{"view_attachments", false}, {"post_attachment", false}}},
}

var leftPermissionGroups = map[string]bool{"general": true, "maintenance": true, "member_admin": true, "general_board": true, "topic": true, "post": true}

var nonGuestPermissions = map[string]bool{
	"karma_edit": true, "pm_read": true, "pm_send": true, "profile_identity": true, "profile_extra": true,
	"profile_title": true, "profile_remove": true, "profile_server_avatar": true, "profile_upload_avatar": true,
	"profile_remote_avatar": true, "poll_vote": true, "mark_any_notify": true, "mark_notify": true,
	"admin_forum": true, "manage_boards": true, "manage_attachments": true, "manage_smileys": true, "edit_news": true,
	"moderate_forum": true, "manage_membergroups": true, "manage_permissions": true, "manage_bans": true, "send_mail": true,
}

// ---- runtime permission-tree structures ----

type permEntry struct {
	ID        string
	Name      string
	ShowHelp  bool
	HasOwnAny bool
	OwnID     string
	OwnName   string
	AnyID     string
	AnyName   string
	Select    string // off/on/denied (no own/any)
	OwnSelect string
	AnySelect string
}

type permGroupCol struct {
	Type        string
	ID          string
	Name        string
	Icon        string
	Help        string
	Permissions []permEntry
}

type permType struct {
	ID    string
	Left  []permGroupCol
	Right []permGroupCol
	Show  bool
}

// loadAllPermissions builds the permission tree (loadAllPermissions()).
// groupID is the membergroup being edited (-1 filters guest-illegal perms);
// pass 0 when not group-specific.
func (c *Ctx) loadAllPermissions(groupID int) []*permType {
	build := func(typeID string, groups []permGroupDef) *permType {
		pt := &permType{ID: typeID}
		for _, pg := range groups {
			col := permGroupCol{
				Type: typeID, ID: pg.id,
				Name: c.Txt("permissiongroup_" + pg.id),
				Help: "",
			}
			if c.TxtHas("permissionicon_" + pg.id) {
				col.Icon = c.Txt("permissionicon_" + pg.id)
			} else {
				col.Icon = c.Txt("permissionicon")
			}
			if c.TxtHas("permissionhelp_" + pg.id) {
				col.Help = c.Txt("permissionhelp_" + pg.id)
			}
			for _, p := range pg.perms {
				if groupID == -1 && nonGuestPermissions[p.id] {
					continue
				}
				e := permEntry{
					ID: p.id, Name: c.Txt("permissionname_" + p.id),
					ShowHelp: c.TxtHas("permissionhelp_" + p.id), HasOwnAny: p.hasOwnAny,
					OwnID: p.id + "_own", AnyID: p.id + "_any",
				}
				if p.hasOwnAny {
					e.OwnName = c.Txt("permissionname_" + p.id + "_own")
					e.AnyName = c.Txt("permissionname_" + p.id + "_any")
				}
				col.Permissions = append(col.Permissions, e)
			}
			if len(col.Permissions) == 0 {
				continue
			}
			if leftPermissionGroups[pg.id] {
				pt.Left = append(pt.Left, col)
			} else {
				pt.Right = append(pt.Right, col)
			}
		}
		return pt
	}
	return []*permType{build("membergroup", permListMembergroup), build("board", permListBoard)}
}

// loadIllegalPermissions sets the permissions the current user can't grant.
func (c *Ctx) loadIllegalPermissions() {
	c.illegalPermissions = nil
	if !c.allowedTo("admin_forum") {
		c.illegalPermissions = append(c.illegalPermissions, "admin_forum")
	}
	if !c.allowedTo("manage_membergroups") {
		c.illegalPermissions = append(c.illegalPermissions, "manage_membergroups")
	}
	if !c.allowedTo("manage_permissions") {
		c.illegalPermissions = append(c.illegalPermissions, "manage_permissions")
	}
}

func (c *Ctx) isIllegalPermission(p string) bool { return inStrings(c.illegalPermissions, p) }

// ---- dispatcher ----

func (c *Ctx) ModifyPermissions() {
	c.adminIndex("edit_permissions")
	c.loadLanguage("ManagePermissions")

	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"board":      {c.permBoardRedirect, "manage_permissions"},
		"index":      {c.PermissionIndex, "manage_permissions"},
		"modify":     {c.ModifyMembergroup, "manage_permissions"},
		"modify2":    {c.ModifyMembergroup2, "manage_permissions"},
		"quick":      {c.SetQuickGroups, "manage_permissions"},
		"quickboard": {c.permBoardRedirect, "manage_permissions"},
		"settings":   {c.GeneralPermissionSettings, "admin_forum"},
		"switch":     {c.permBoardRedirect, "manage_permissions"},
	}

	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		if c.allowedTo("manage_permissions") {
			sa = "index"
		} else {
			sa = "settings"
		}
	}
	c.isAllowedTo(subs[sa].perm)

	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("permissions_title"), Help: "permissions"}
	if c.allowedTo("manage_permissions") {
		tabs.Tabs = append(tabs.Tabs,
			AdminTab{Title: c.Txt("permissions_groups"), Description: c.Txt("permission_by_membergroup_desc"), Href: scripturl + "?action=permissions", IsSelected: (sa == "modify" || sa == "index") && c.REQUEST.Int("boardid") == 0},
			AdminTab{Title: c.Txt("permissions_boards"), Description: c.Txt("permission_by_board_desc"), Href: scripturl + "?action=permissions;sa=board", IsSelected: sa == "board" || sa == "switch", IsLast: !c.allowedTo("admin_forum")})
	}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("settings"), Description: c.Txt("permission_settings_desc"), Href: scripturl + "?action=permissions;sa=settings", IsSelected: sa == "settings", IsLast: true})
	}
	c.AdminTabs = tabs

	subs[sa].fn()
}

// permBoardRedirect handles the board-level sub-actions (disabled in this port).
func (c *Ctx) permBoardRedirect() { c.redirectExit("action=permissions") }

// ---- index ----

// PermIndexGroup is one membergroup row in the permission index.
type PermIndexGroup struct {
	ID          int
	Name        string
	NumMembers  string
	Link        string
	CanSearch   bool
	AllowModify bool
	NumAllowed  string
	NumDenied   string
}

// PermIndexCtx backs template_permission_index.
type PermIndexCtx struct {
	Groups      []PermIndexGroup
	Permissions []*permType
}

func (c *Ctx) PermissionIndex() {
	a := c.App
	scripturl := a.ScriptURL
	c.PageTitle = c.Txt("permissions_title")

	page := &PermIndexCtx{Permissions: c.loadAllPermissions(0)}
	c.Page = page

	var numMembers int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE ID_GROUP = 0`)).Scan(&numMembers)

	// Guests and Regular Members are synthetic.
	type grp struct {
		id          int
		name        string
		numMembers  int
		numMembersS string // override (e.g. "n/a")
		canSearch   bool
		allowModify bool
		href        string
		allowed     string
		denied      string
		isAdmin     bool
	}
	order := []int{-1, 0}
	groups := map[int]*grp{
		-1: {id: -1, name: c.Txt("membergroups_guests"), numMembersS: c.Txt("membergroups_guests_na"), allowModify: true, denied: "(" + c.Txt("permissions_none") + ")"},
		0:  {id: 0, name: c.Txt("membergroups_members"), numMembers: numMembers, canSearch: true, allowModify: true, href: scripturl + "?action=viewmembers;sa=query;group=0"},
	}

	var postGroups, normalGroups []int
	cond := ""
	if a.SettingEmpty("permission_enable_postgroups") {
		cond = " WHERE minPosts = -1"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_GROUP, groupName, minPosts
		FROM {$db_prefix}membergroups` + cond + `
		ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`))
	if err == nil {
		for rows.Next() {
			var id, minPosts int
			var name string
			rows.Scan(&id, &name, &minPosts)
			g := &grp{id: id, name: name, canSearch: id != 3, allowModify: id > 1, isAdmin: id == 1}
			if id == 3 {
				g.numMembersS = c.Txt("membergroups_guests_na")
			}
			if minPosts == -1 {
				g.href = scripturl + "?action=viewmembers;sa=query;group=" + itoa(id)
				normalGroups = append(normalGroups, id)
			} else {
				g.href = scripturl + "?action=viewmembers;sa=query;pgroup=" + itoa(id)
				postGroups = append(postGroups, id)
			}
			if id == 1 {
				g.allowed = "(" + c.Txt("permissions_all") + ")"
				g.denied = "(" + c.Txt("permissions_none") + ")"
			}
			groups[id] = g
			order = append(order, id)
		}
		rows.Close()
	}

	addCounts := func(query string, ids []int) {
		if len(ids) == 0 {
			return
		}
		r, err := a.DB.Query(a.Q(query))
		if err != nil {
			return
		}
		for r.Next() {
			var id, n int
			r.Scan(&id, &n)
			if g := groups[id]; g != nil {
				g.numMembers += n
			}
		}
		r.Close()
	}
	addCounts(`SELECT ID_POST_GROUP, COUNT(*) FROM {$db_prefix}members WHERE ID_POST_GROUP IN (`+joinInts(postGroups)+`) GROUP BY ID_POST_GROUP`, postGroups)
	addCounts(`SELECT ID_GROUP, COUNT(*) FROM {$db_prefix}members WHERE ID_GROUP IN (`+joinInts(normalGroups)+`) GROUP BY ID_GROUP`, normalGroups)
	addCounts(`SELECT mg.ID_GROUP, COUNT(*) FROM {$db_prefix}membergroups AS mg, {$db_prefix}members AS mem WHERE mg.ID_GROUP IN (`+joinInts(normalGroups)+`) AND mem.additionalGroups != '' AND mem.ID_GROUP != mg.ID_GROUP AND FIND_IN_SET(mg.ID_GROUP, mem.additionalGroups) GROUP BY mg.ID_GROUP`, normalGroups)

	// Permission counts per group (boardid=0 path).
	prows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, COUNT(*), addDeny FROM {$db_prefix}permissions GROUP BY ID_GROUP, addDeny`))
	if err == nil {
		for prows.Next() {
			var id, n, addDeny int
			prows.Scan(&id, &n, &addDeny)
			if g := groups[id]; g != nil && (addDeny != 0 || id != -1) && id != 1 {
				if addDeny == 0 {
					g.denied = itoa(n)
				} else {
					g.allowed = itoa(n)
				}
			}
		}
		prows.Close()
	}
	brows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, COUNT(*), addDeny FROM {$db_prefix}board_permissions WHERE ID_BOARD = 0 GROUP BY ID_GROUP, addDeny`))
	if err == nil {
		for brows.Next() {
			var id, n, addDeny int
			brows.Scan(&id, &n, &addDeny)
			if g := groups[id]; g != nil && (addDeny != 0 || id != -1) && id != 1 {
				if addDeny == 0 {
					g.denied = addInt(g.denied, n)
				} else {
					g.allowed = addInt(g.allowed, n)
				}
			}
		}
		brows.Close()
	}

	for _, id := range order {
		g := groups[id]
		numMembersStr := itoa(g.numMembers)
		if g.numMembersS != "" {
			numMembersStr = g.numMembersS
		}
		link := ""
		if g.href != "" {
			link = `<a href="` + g.href + `">` + numMembersStr + `</a>`
		}
		allowed := g.allowed
		if allowed == "" {
			allowed = "0"
		}
		denied := g.denied
		if denied == "" {
			denied = "0"
		}
		page.Groups = append(page.Groups, PermIndexGroup{
			ID: id, Name: g.name, NumMembers: numMembersStr, Link: link,
			CanSearch: g.canSearch, AllowModify: g.allowModify, NumAllowed: allowed, NumDenied: denied,
		})
	}

	c.SubTemplate = templatePermissionIndex
}

// addInt adds n to a numeric string (treating non-numeric/"" as 0).
func addInt(s string, n int) string {
	return itoa(atoi(s) + n)
}

// ---- per-group editing ----

// PermModifyCtx backs template_modify_group.
type PermModifyCtx struct {
	GroupID     int
	GroupName   string
	BoardID     int
	BoardName   string
	Local       bool
	Permissions []*permType
}

func (c *Ctx) ModifyMembergroup() {
	a := c.App
	groupID := c.GET.Int("group")
	if groupID == 1 {
		c.redirectExit("action=permissions")
	}

	page := &PermModifyCtx{GroupID: groupID, Permissions: c.loadAllPermissions(groupID)}
	c.Page = page

	if groupID > 0 {
		a.DB.QueryRow(a.Q(`SELECT groupName FROM {$db_prefix}membergroups WHERE ID_GROUP = ? LIMIT 1`), groupID).Scan(&page.GroupName)
	} else if groupID == -1 {
		page.GroupName = c.Txt("membergroups_guests")
	} else {
		page.GroupName = c.Txt("membergroups_members")
	}

	// Current permissions for this group (boardid=0 / global only in this port).
	allowed := map[string]bool{}
	denied := map[string]bool{}
	showMembergroup := groupID != 3
	if showMembergroup {
		rows, err := a.DB.Query(a.Q(`SELECT permission, addDeny FROM {$db_prefix}permissions WHERE ID_GROUP = ?`), groupID)
		if err == nil {
			for rows.Next() {
				var perm string
				var addDeny int
				rows.Scan(&perm, &addDeny)
				if addDeny == 0 {
					denied[perm] = true
				} else {
					allowed[perm] = true
				}
			}
			rows.Close()
		}
	}
	boardAllowed := map[string]bool{}
	boardDenied := map[string]bool{}
	rows, err := a.DB.Query(a.Q(`SELECT permission, addDeny FROM {$db_prefix}board_permissions WHERE ID_GROUP = ? AND ID_BOARD = 0`), groupID)
	if err == nil {
		for rows.Next() {
			var perm string
			var addDeny int
			rows.Scan(&perm, &addDeny)
			if addDeny == 0 {
				boardDenied[perm] = true
			} else {
				boardAllowed[perm] = true
			}
		}
		rows.Close()
	}

	sel := func(perm string, al, dn map[string]bool) string {
		if al[perm] {
			return "on"
		}
		if dn[perm] {
			return "denied"
		}
		return "off"
	}

	for _, pt := range page.Permissions {
		var al, dn map[string]bool
		if pt.ID == "membergroup" {
			pt.Show = showMembergroup
			al, dn = allowed, denied
		} else {
			pt.Show = true
			al, dn = boardAllowed, boardDenied
		}
		for ci := range [2][]permGroupCol{pt.Left, pt.Right} {
			cols := pt.Left
			if ci == 1 {
				cols = pt.Right
			}
			for gi := range cols {
				for pi := range cols[gi].Permissions {
					e := &cols[gi].Permissions[pi]
					if e.HasOwnAny {
						e.AnySelect = sel(e.ID+"_any", al, dn)
						e.OwnSelect = sel(e.ID+"_own", al, dn)
					} else {
						e.Select = sel(e.ID, al, dn)
					}
				}
			}
		}
	}

	c.SubTemplate = templateModifyGroup
	c.PageTitle = c.Txt("permissions_modify_group")
}

func (c *Ctx) ModifyMembergroup2() {
	a := c.App
	c.checkSession("post", "", true)
	c.loadIllegalPermissions()

	groupID := c.GET.Int("group")

	type pv struct {
		permission string
		addDeny    string
	}
	give := map[string][]pv{"membergroup": nil, "board": nil}
	if perm := c.POST.Arr("perm"); perm != nil {
		for _, ptype := range []string{"membergroup", "board"} {
			if sub := perm.Arr(ptype); sub != nil {
				sub.Values(func(permission string, v any) {
					s, _ := v.(string)
					if s == "on" || s == "deny" {
						if c.isIllegalPermission(permission) {
							return
						}
						ad := "1"
						if s == "deny" {
							ad = "0"
						}
						give[ptype] = append(give[ptype], pv{permission, ad})
					}
				})
			}
		}
	}

	// General permissions.
	if groupID != 3 {
		notIn := ""
		if len(c.illegalPermissions) > 0 {
			notIn = " AND permission NOT IN (" + quoteList(c.illegalPermissions) + ")"
		}
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP = ?`+notIn), groupID)
		for _, p := range give["membergroup"] {
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}permissions (ID_GROUP, permission, addDeny) VALUES (?, ?, ?)`), groupID, p.permission, p.addDeny)
		}
	}

	// Board permissions (global, board 0).
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}board_permissions WHERE ID_GROUP = ? AND ID_BOARD = 0`), groupID)
	for _, p := range give["board"] {
		a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}board_permissions (ID_GROUP, ID_BOARD, permission, addDeny) VALUES (?, 0, ?, ?)`), groupID, p.permission, p.addDeny)
	}

	c.redirectExit("action=permissions")
}

// quoteList renders a comma-separated quoted SQL list.
func quoteList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return strings.Join(q, ", ")
}

// ---- quick actions (membergroup level) ----

func (c *Ctx) SetQuickGroups() {
	a := c.App
	c.checkSession("post", "", true)
	c.loadIllegalPermissions()

	var groupIDs []int
	if arr := c.POST.Arr("group"); arr != nil {
		arr.Values(func(k string, v any) {
			s, _ := v.(string)
			groupIDs = append(groupIDs, atoi(s))
		})
	}
	groupIDs = uniqueInts(groupIDs)
	if len(groupIDs) == 0 {
		c.redirectExit("action=permissions;boardid=0")
	}

	predefined := c.POST.Str("predefined")
	copyFrom := c.POST.Str("copy_from")
	permissions := c.POST.Str("permissions")

	if predefined != "" {
		if !inStrings([]string{"restrict", "standard", "moderator", "maintenance"}, predefined) {
			c.redirectExit("action=permissions;boardid=0")
		}
		for _, g := range groupIDs {
			c.setPermissionLevel(predefined, g, -1)
		}
	} else if copyFrom != "" && copyFrom != "empty" {
		from := atoi(copyFrom)
		// Don't copy onto the source group.
		var targets []int
		for _, g := range groupIDs {
			if g != from {
				targets = append(targets, g)
			}
		}
		if len(targets) == 0 {
			c.redirectExit("action=permissions;boardid=0")
		}
		type tp struct {
			perm    string
			addDeny int
		}
		var src []tp
		rows, err := a.DB.Query(a.Q(`SELECT permission, addDeny FROM {$db_prefix}permissions WHERE ID_GROUP = ?`), from)
		if err == nil {
			for rows.Next() {
				var t tp
				rows.Scan(&t.perm, &t.addDeny)
				src = append(src, t)
			}
			rows.Close()
		}
		notIn := ""
		if len(c.illegalPermissions) > 0 {
			notIn = " AND permission NOT IN (" + quoteList(c.illegalPermissions) + ")"
		}
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP IN (` + joinInts(targets) + `)` + notIn))
		for _, g := range targets {
			for _, t := range src {
				if c.isIllegalPermission(t.perm) {
					continue
				}
				a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}permissions (permission, ID_GROUP, addDeny) VALUES (?, ?, ?)`), t.perm, g, t.addDeny)
			}
		}
	} else if permissions != "" {
		parts := strings.SplitN(permissions, "/", 2)
		if len(parts) != 2 {
			c.redirectExit("action=permissions;boardid=0")
		}
		permissionType, permission := parts[0], parts[1]
		addRemove := c.POST.Str("add_remove")
		if !inStrings([]string{"add", "clear", "deny"}, addRemove) || (permissionType != "membergroup" && permissionType != "board") {
			c.redirectExit("action=permissions;boardid=0")
		}
		if addRemove == "clear" {
			if permissionType == "membergroup" {
				notIn := ""
				if len(c.illegalPermissions) > 0 {
					notIn = " AND permission NOT IN (" + quoteList(c.illegalPermissions) + ")"
				}
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP IN (`+joinInts(groupIDs)+`) AND permission = ?`+notIn), permission)
			} else {
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}board_permissions WHERE ID_GROUP IN (`+joinInts(groupIDs)+`) AND ID_BOARD = 0 AND permission = ?`), permission)
			}
		} else {
			addDeny := "1"
			if addRemove == "deny" {
				addDeny = "0"
			}
			if permissionType == "membergroup" && !c.isIllegalPermission(permission) {
				for _, g := range groupIDs {
					a.DB.Exec(a.Q(`INSERT OR REPLACE INTO {$db_prefix}permissions (permission, ID_GROUP, addDeny) VALUES (?, ?, ?)`), permission, g, addDeny)
				}
			} else if permissionType != "membergroup" {
				for _, g := range groupIDs {
					a.DB.Exec(a.Q(`INSERT OR REPLACE INTO {$db_prefix}board_permissions (permission, ID_GROUP, ID_BOARD, addDeny) VALUES (?, ?, 0, ?)`), permission, g, addDeny)
				}
			}
		}
	}

	// Guests may never have these.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP = -1 AND (permission = 'manage_membergroups' OR permission = 'manage_permissions' OR permission = 'admin_forum')`))

	c.redirectExit("action=permissions;boardid=0")
}

// ---- predefined permission levels ----

func (c *Ctx) setPermissionLevel(level string, group, board int) {
	a := c.App
	c.loadIllegalPermissions()

	globalLevels := map[string][]string{
		"restrict": {"search_posts", "view_stats", "who_view", "profile_view_own", "profile_identity_own"},
	}
	globalLevels["standard"] = append(append([]string{}, globalLevels["restrict"]...), "view_mlist", "karma_edit", "pm_read", "pm_send", "profile_view_any", "profile_extra_own", "profile_server_avatar", "profile_upload_avatar", "profile_remote_avatar", "profile_remove_own")
	globalLevels["moderator"] = append([]string{}, globalLevels["standard"]...)
	globalLevels["maintenance"] = append(append([]string{}, globalLevels["moderator"]...), "manage_attachments", "manage_smileys", "manage_boards", "moderate_forum", "manage_membergroups", "manage_bans", "admin_forum", "manage_permissions", "edit_news", "profile_identity_any", "profile_extra_any", "profile_title_any")

	boardLevels := map[string][]string{
		"restrict": {"poll_view", "post_new", "post_reply_own", "post_reply_any", "delete_own", "modify_own", "mark_any_notify", "mark_notify", "report_any", "send_topic"},
	}
	boardLevels["standard"] = append(append([]string{}, boardLevels["restrict"]...), "poll_vote", "poll_edit_own", "poll_post", "poll_add_own", "post_attachment", "lock_own", "remove_own", "view_attachments")
	boardLevels["moderator"] = append(append([]string{}, boardLevels["standard"]...), "make_sticky", "poll_edit_any", "delete_any", "modify_any", "lock_any", "remove_any", "move_any", "merge_any", "split_any", "poll_lock_any", "poll_remove_any", "poll_add_any")
	boardLevels["maintenance"] = append([]string{}, boardLevels["moderator"]...)

	gl := globalLevels[level]
	// Remove illegal permissions.
	var glFiltered []string
	for _, p := range gl {
		if !c.isIllegalPermission(p) {
			glFiltered = append(glFiltered, p)
		}
	}
	gl = glFiltered

	if board == -1 && group != -999 {
		// Group permissions (global).
		if len(gl) == 0 {
			return
		}
		notIn := ""
		if len(c.illegalPermissions) > 0 {
			notIn = " AND permission NOT IN (" + quoteList(c.illegalPermissions) + ")"
		}
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP = ?`+notIn), group)
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}board_permissions WHERE ID_GROUP = ? AND ID_BOARD = 0`), group)
		for _, p := range gl {
			a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}permissions (ID_GROUP, permission) VALUES (?, ?)`), group, p)
		}
		for _, p := range boardLevels[level] {
			a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}board_permissions (ID_BOARD, ID_GROUP, permission) VALUES (0, ?, ?)`), group, p)
		}
	}
	// Board-specific levels are gated behind permission_enable_by_board (off).
}

// ---- general settings ----

func (c *Ctx) GeneralPermissionSettings() {
	a := c.App
	c.PageTitle = c.Txt("permission_settings_title")

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)

		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		a.UpdateSettings(map[string]string{
			"permission_enable_deny":       bit("permission_enable_deny"),
			"permission_enable_postgroups": bit("permission_enable_postgroups"),
			"permission_enable_by_board":   bit("permission_enable_by_board"),
		})

		// Clear deny permissions if disabled.
		if a.SettingEmpty("permission_enable_deny") {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE addDeny = 0`))
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}board_permissions WHERE addDeny = 0`))
		}
		// Remove postgroup permissions if disabled.
		if a.SettingEmpty("permission_enable_postgroups") {
			var pg []int
			rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP FROM {$db_prefix}membergroups WHERE minPosts != -1`))
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					pg = append(pg, id)
				}
				rows.Close()
			}
			if len(pg) > 0 {
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE ID_GROUP IN (` + joinInts(pg) + `)`))
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}board_permissions WHERE ID_GROUP IN (` + joinInts(pg) + `)`))
			}
		}

		c.saveInlinePermissions([]string{"manage_permissions"})
	}

	c.initInlinePermissions([]string{"manage_permissions"}, []int{-1})
	c.SubTemplate = templateGeneralPermissionSettings
}

// ---- inline permissions (the unblocker) ----

// InlinePermGroup is one group row in an inline permission box.
type InlinePermGroup struct {
	ID          int
	Name        string
	IsPostgroup bool
	Status      string // on/off/deny
}

// initInlinePermissions is init_inline_permissions().
func (c *Ctx) initInlinePermissions(permissions []string, excludedGroups []int) {
	a := c.App
	c.loadLanguage("ManagePermissions")
	c.CanChangePermissions = c.allowedTo("manage_permissions")
	if c.InlinePermissions == nil {
		c.InlinePermissions = map[string][]InlinePermGroup{}
	}
	if !c.CanChangePermissions {
		return
	}

	// group id -> index within each permission's slice
	type key struct {
		perm string
		grp  int
	}
	idx := map[key]int{}
	for _, perm := range permissions {
		c.InlinePermissions[perm] = []InlinePermGroup{
			{ID: -1, Name: c.Txt("membergroups_guests"), Status: "off"},
			{ID: 0, Name: c.Txt("membergroups_members"), Status: "off"},
		}
		idx[key{perm, -1}] = 0
		idx[key{perm, 0}] = 1
	}

	permIn := quoteList(permissions)

	// Guests / members status.
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_GROUP, IIF(addDeny = 0, 'deny', 'on') AS status, permission
		FROM {$db_prefix}permissions
		WHERE ID_GROUP IN (-1, 0) AND permission IN (` + permIn + `)`))
	if err == nil {
		for rows.Next() {
			var id int
			var status, perm string
			rows.Scan(&id, &status, &perm)
			if i, ok := idx[key{perm, id}]; ok {
				c.InlinePermissions[perm][i].Status = status
			}
		}
		rows.Close()
	}

	// Other groups.
	cond := ""
	if a.SettingEmpty("permission_enable_postgroups") {
		cond = " AND mg.minPosts = -1"
	}
	grows, err := a.DB.Query(a.Q(`
		SELECT mg.ID_GROUP, mg.groupName, mg.minPosts, IFNULL(p.addDeny, -1) AS status, p.permission
		FROM {$db_prefix}membergroups AS mg
			LEFT JOIN {$db_prefix}permissions AS p ON (p.ID_GROUP = mg.ID_GROUP AND p.permission IN (` + permIn + `))
		WHERE mg.ID_GROUP NOT IN (1, 3)` + cond + `
		ORDER BY mg.minPosts, IIF(mg.ID_GROUP < 4, mg.ID_GROUP, 4), mg.groupName`))
	if err == nil {
		for grows.Next() {
			var id, minPosts, status int
			var name string
			var perm *string
			grows.Scan(&id, &name, &minPosts, &status, &perm)
			for _, p := range permissions {
				if _, ok := idx[key{p, id}]; !ok {
					c.InlinePermissions[p] = append(c.InlinePermissions[p], InlinePermGroup{ID: id, Name: name, IsPostgroup: minPosts != -1, Status: "off"})
					idx[key{p, id}] = len(c.InlinePermissions[p]) - 1
				}
			}
			if perm != nil {
				st := "off"
				if status == 0 {
					st = "deny"
				} else if status == 1 {
					st = "on"
				}
				if i, ok := idx[key{*perm, id}]; ok {
					c.InlinePermissions[*perm][i].Status = st
				}
			}
		}
		grows.Close()
	}

	// Remove excluded groups.
	for _, group := range excludedGroups {
		for _, perm := range permissions {
			list := c.InlinePermissions[perm]
			var out []InlinePermGroup
			for _, g := range list {
				if g.ID != group {
					out = append(out, g)
				}
			}
			c.InlinePermissions[perm] = out
		}
	}
}

// saveInlinePermissions is save_inline_permissions().
func (c *Ctx) saveInlinePermissions(permissions []string) {
	a := c.App
	if !c.allowedTo("manage_permissions") {
		return
	}
	c.loadIllegalPermissions()

	type row struct {
		group   int
		perm    string
		addDeny string
	}
	var inserts []row
	for _, perm := range permissions {
		arr := c.POST.Arr(perm)
		if arr == nil {
			continue
		}
		arr.Values(func(k string, v any) {
			s, _ := v.(string)
			if (s == "on" || s == "deny") && !c.isIllegalPermission(perm) {
				ad := "1"
				if s == "deny" {
					ad = "0"
				}
				inserts = append(inserts, row{atoi(k), perm, ad})
			}
		})
	}

	notIn := ""
	if len(c.illegalPermissions) > 0 {
		notIn = " AND permission NOT IN (" + quoteList(c.illegalPermissions) + ")"
	}
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}permissions WHERE permission IN (` + quoteList(permissions) + `)` + notIn))
	for _, r := range inserts {
		a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}permissions (ID_GROUP, permission, addDeny) VALUES (?, ?, ?)`), r.group, r.perm, r.addDeny)
	}
}
