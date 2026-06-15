package app

// Port of Sources/ManageMembergroups.php: the membergroup admin
// (?action=membergroups) — list, add, edit, delete, member assignment, and
// settings (inline manage_membergroups permission). Group helpers live in
// subs_membergroups.go; predefined permission levels / copy use permissions.go.

import (
	"regexp"
	"strings"
)

func init() {
	registerAction("membergroups", (*Ctx).ModifyMembergroups)
}

func (c *Ctx) ModifyMembergroups() {
	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"add":      {c.AddMembergroup, "manage_membergroups"},
		"delete":   {c.DeleteMembergroup, "manage_membergroups"},
		"edit":     {c.EditMembergroup, "manage_membergroups"},
		"index":    {c.MembergroupIndex, "manage_membergroups"},
		"members":  {c.MembergroupMembers, "manage_membergroups"},
		"settings": {c.ModifyMembergroupSettings, "admin_forum"},
	}
	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		if c.allowedTo("manage_membergroups") {
			sa = "index"
		} else {
			sa = "settings"
		}
	}
	c.isAllowedTo(subs[sa].perm)

	c.adminIndex("edit_groups")
	c.loadLanguage("ManageMembers")

	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("membergroups_title"), Help: "membergroups", Description: c.Txt("membergroups_description")}
	if c.allowedTo("manage_membergroups") {
		tabs.Tabs = append(tabs.Tabs,
			AdminTab{Title: c.Txt("membergroups_edit_groups"), Description: c.Txt("membergroups_description"), Href: scripturl + "?action=membergroups", IsSelected: sa != "add" && sa != "settings"},
			AdminTab{Title: c.Txt("membergroups_new_group"), Description: c.Txt("membergroups_description"), Href: scripturl + "?action=membergroups;sa=add", IsSelected: sa == "add", IsLast: !c.allowedTo("admin_forum")})
	}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("settings"), Description: c.Txt("membergroups_description"), Href: scripturl + "?action=membergroups;sa=settings", IsSelected: sa == "settings", IsLast: true})
	}
	c.AdminTabs = tabs

	subs[sa].fn()
}

// ---- index ----

// MGIndexGroup is one group row.
type MGIndexGroup struct {
	ID         int
	Name       string
	Color      string
	Link       string
	NumMembers string
	Stars      string
	MinPosts   string
	CanSearch  bool
}

type MGIndexCtx struct {
	Regular []MGIndexGroup
	Post    []MGIndexGroup
}

// renderStars turns "count#image" into repeated <img> tags.
func (c *Ctx) renderStars(stars string) string {
	parts := strings.SplitN(stars, "#", 2)
	if len(parts) != 2 || parts[0] == "" || parts[0] == "0" || parts[1] == "" {
		return ""
	}
	n := atoi(parts[0])
	img := `<img src="` + c.Theme.ImagesURL() + `/` + parts[1] + `" alt="*" border="0" />`
	return strings.Repeat(img, n)
}

func (c *Ctx) MembergroupIndex() {
	a := c.App
	scripturl := a.ScriptURL
	c.PageTitle = c.Txt("membergroups_title")

	page := &MGIndexCtx{}
	c.Page = page

	type grp struct {
		g        MGIndexGroup
		isPost   bool
		numCount int
	}
	var order []int
	groups := map[int]*grp{}

	rows, err := a.DB.Query(a.Q(`
		SELECT ID_GROUP, groupName, minPosts, onlineColor, stars
		FROM {$db_prefix}membergroups
		ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`))
	if err == nil {
		for rows.Next() {
			var id, minPosts int
			var name, color, stars string
			rows.Scan(&id, &name, &minPosts, &color, &stars)
			g := &grp{isPost: minPosts != -1}
			g.g = MGIndexGroup{
				ID: id, Name: name, Color: color, CanSearch: id != 3,
				Stars: c.renderStars(stars),
			}
			if minPosts == -1 {
				g.g.MinPosts = "-"
			} else {
				g.g.MinPosts = itoa(minPosts)
			}
			if id == 3 {
				g.g.NumMembers = c.Txt("membergroups_guests_na")
			}
			if id != 3 {
				g.g.Link = `<a href="` + scripturl + `?action=membergroups;sa=members;group=` + itoa(id) + `">` + name + `</a>`
			}
			groups[id] = g
			order = append(order, id)
		}
		rows.Close()
	}

	// Member counts.
	var postIDs, normalIDs []int
	for id, g := range groups {
		if g.isPost {
			postIDs = append(postIDs, id)
		} else {
			normalIDs = append(normalIDs, id)
		}
	}
	addCount := func(q string) {
		r, err := a.DB.Query(a.Q(q))
		if err != nil {
			return
		}
		for r.Next() {
			var id, n int
			r.Scan(&id, &n)
			if g := groups[id]; g != nil {
				g.numCount += n
			}
		}
		r.Close()
	}
	if len(postIDs) > 0 {
		addCount(`SELECT ID_POST_GROUP, COUNT(*) FROM {$db_prefix}members WHERE ID_POST_GROUP IN (` + joinInts(postIDs) + `) GROUP BY ID_POST_GROUP`)
	}
	if len(normalIDs) > 0 {
		addCount(`SELECT ID_GROUP, COUNT(*) FROM {$db_prefix}members WHERE ID_GROUP IN (` + joinInts(normalIDs) + `) GROUP BY ID_GROUP`)
		addCount(`SELECT mg.ID_GROUP, COUNT(*) FROM {$db_prefix}membergroups AS mg, {$db_prefix}members AS mem WHERE mg.ID_GROUP IN (` + joinInts(normalIDs) + `) AND mem.additionalGroups != '' AND mem.ID_GROUP != mg.ID_GROUP AND FIND_IN_SET(mg.ID_GROUP, mem.additionalGroups) GROUP BY mg.ID_GROUP`)
	}

	for _, id := range order {
		g := groups[id]
		if g.g.NumMembers == "" {
			g.g.NumMembers = itoa(g.numCount)
		}
		if g.isPost {
			page.Post = append(page.Post, g.g)
		} else {
			page.Regular = append(page.Regular, g.g)
		}
	}

	c.SubTemplate = templateMembergroupsMain
}

// ---- add ----

// MGBoard is a board access checkbox.
type MGBoard struct {
	ID         int
	Name       string
	ChildLevel int
	Selected   bool
}

type MGNewCtx struct {
	PostGroup      bool
	UndefinedGroup bool
	Groups         [][2]string
	Boards         []MGBoard
}

func (c *Ctx) AddMembergroup() {
	a := c.App

	if c.POST.Str("group_name") != "" {
		c.checkSession("post", "", true)

		postCountBased := c.POST.Has("min_posts") && (!c.POST.Has("postgroup_based") || c.POST.Str("postgroup_based") != "0")

		var maxGroup int
		a.DB.QueryRow(a.Q(`SELECT IFNULL(MAX(ID_GROUP), 0) FROM {$db_prefix}membergroups`)).Scan(&maxGroup)
		newID := maxGroup + 1

		minPosts := -1
		if postCountBased {
			minPosts = c.POST.Int("min_posts")
		}
		a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}membergroups (ID_GROUP, groupName, minPosts, stars, onlineColor)
			VALUES (?, SUBSTR(?, 1, 80), ?, '1#star.gif', '')`), newID, c.POST.Str("group_name"), minPosts)

		if c.POST.Has("min_posts") {
			c.updateStatsPostgroups(0)
		}

		permType := c.POST.Str("perm_type")
		if postCountBased && a.SettingEmpty("permission_enable_postgroups") {
			permType = ""
		}

		if permType == "predefined" {
			c.setPermissionLevel(c.POST.Str("level"), newID, -1)
		} else if permType == "copy" {
			c.loadIllegalPermissions()
			copyID := c.POST.Int("copyperm")
			// Copy general permissions.
			prows, err := a.DB.Query(a.Q(`SELECT permission, addDeny FROM {$db_prefix}permissions WHERE ID_GROUP = ?`), copyID)
			if err == nil {
				type pp struct {
					perm    string
					addDeny int
				}
				var perms []pp
				for prows.Next() {
					var p pp
					prows.Scan(&p.perm, &p.addDeny)
					perms = append(perms, p)
				}
				prows.Close()
				for _, p := range perms {
					if c.isIllegalPermission(p.perm) {
						continue
					}
					a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}permissions (ID_GROUP, permission, addDeny) VALUES (?, ?, ?)`), newID, p.perm, p.addDeny)
				}
			}
			// Copy board permissions (global only in this port).
			brows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, permission, addDeny FROM {$db_prefix}board_permissions WHERE ID_GROUP = ? AND ID_BOARD = 0`), copyID)
			if err == nil {
				type bp struct {
					board   int
					perm    string
					addDeny int
				}
				var perms []bp
				for brows.Next() {
					var p bp
					brows.Scan(&p.board, &p.perm, &p.addDeny)
					perms = append(perms, p)
				}
				brows.Close()
				for _, p := range perms {
					a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}board_permissions (ID_GROUP, ID_BOARD, permission, addDeny) VALUES (?, ?, ?, ?)`), newID, p.board, p.perm, p.addDeny)
				}
			}
			// Copy membergroup look (if not from guests).
			if copyID > 0 {
				var color, stars string
				var maxMessages int
				a.DB.QueryRow(a.Q(`SELECT onlineColor, maxMessages, stars FROM {$db_prefix}membergroups WHERE ID_GROUP = ? LIMIT 1`), copyID).Scan(&color, &maxMessages, &stars)
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}membergroups SET onlineColor = ?, maxMessages = ?, stars = ? WHERE ID_GROUP = ?`), color, maxMessages, stars, newID)
			}
		}

		// Board access.
		var boardAccess []int
		if arr := c.POST.Arr("boardaccess"); arr != nil {
			arr.Values(func(k string, v any) {
				s, _ := v.(string)
				boardAccess = append(boardAccess, atoi(s))
			})
		}
		if len(boardAccess) > 0 {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET memberGroups = IIF(memberGroups = '', '` + itoa(newID) + `', memberGroups || ',` + itoa(newID) + `') WHERE ID_BOARD IN (` + joinInts(boardAccess) + `)`))
		}

		c.redirectExit("action=membergroups;sa=edit;group=" + itoa(newID))
	}

	page := &MGNewCtx{
		PostGroup:      c.REQUEST.Has("postgroup"),
		UndefinedGroup: !c.REQUEST.Has("postgroup") && !c.REQUEST.Has("generalgroup"),
	}
	c.Page = page
	c.PageTitle = c.Txt("membergroups_new_group")
	c.SubTemplate = templateMembergroupNew

	cond := ""
	if a.SettingEmpty("permission_enable_postgroups") {
		cond = " AND minPosts = -1"
	}
	grows, err := a.DB.Query(a.Q(`
		SELECT ID_GROUP, groupName FROM {$db_prefix}membergroups
		WHERE (ID_GROUP > 3 OR ID_GROUP = 2)` + cond + `
		ORDER BY minPosts, ID_GROUP != 2, groupName`))
	if err == nil {
		for grows.Next() {
			var id int
			var name string
			grows.Scan(&id, &name)
			page.Groups = append(page.Groups, [2]string{itoa(id), name})
		}
		grows.Close()
	}

	brows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name, childLevel FROM {$db_prefix}boards`))
	if err == nil {
		for brows.Next() {
			var b MGBoard
			brows.Scan(&b.ID, &b.Name, &b.ChildLevel)
			page.Boards = append(page.Boards, b)
		}
		brows.Close()
	}
}

// ---- delete (URL) ----

func (c *Ctx) DeleteMembergroup() {
	c.checkSession("get", "", true)
	c.deleteMembergroups([]int{c.REQUEST.Int("group")})
	c.redirectExit("action=membergroups;")
}

// ---- edit ----

type MGEditCtx struct {
	ID             int
	Name           string
	EditableName   string
	Color          string
	MinPosts       int
	MaxMessages    int
	StarCount      int
	StarImage      string
	IsPostGroup    bool
	AllowPostGroup bool
	AllowDelete    bool
	Boards         []MGBoard
}

func (c *Ctx) EditMembergroup() {
	a := c.App
	groupID := c.REQUEST.Int("group")
	if c.REQUEST.Int("group") < 1 {
		c.fatalLangError("membergroup_does_not_exist", false)
	}

	if c.POST.Has("delete") {
		c.checkSession("post", "", true)
		c.deleteMembergroups([]int{groupID})
		c.redirectExit("action=membergroups;")
	} else if c.POST.Has("submit") {
		c.checkSession("post", "", true)

		maxMessages := c.POST.Int("max_messages")
		minPosts := -1
		if c.POST.Has("min_posts") && c.POST.Str("post_group") == "1" && groupID > 3 {
			minPosts = absInt(c.POST.Int("min_posts"))
		} else if groupID == 4 {
			minPosts = 0
		}
		stars := ""
		if c.POST.Int("star_count") > 0 {
			n := c.POST.Int("star_count")
			if n > 99 {
				n = 99
			}
			stars = itoa(n) + "#" + c.POST.Str("star_image")
		}
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}membergroups
			SET groupName = ?, onlineColor = ?, maxMessages = ?, minPosts = ?, stars = ?
			WHERE ID_GROUP = ?`), c.POST.Str("group_name"), c.POST.Str("online_color"), maxMessages, minPosts, stars, groupID)

		// Board access.
		if groupID == 2 || groupID > 3 {
			var boardAccess []int
			if arr := c.POST.Arr("boardaccess"); arr != nil {
				arr.Values(func(k string, v any) {
					s, _ := v.(string)
					boardAccess = append(boardAccess, atoi(s))
				})
			}
			// Remove from boards no longer selected.
			notIn := ""
			if len(boardAccess) > 0 {
				notIn = " AND ID_BOARD NOT IN (" + joinInts(boardAccess) + ")"
			}
			c.rewriteCSVColumnWhere("boards", "ID_BOARD", "memberGroups", []int{groupID}, "FIND_IN_SET("+itoa(groupID)+", memberGroups)"+notIn)
			// Add to newly selected boards.
			if len(boardAccess) > 0 {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET memberGroups = IIF(memberGroups = '', '` + itoa(groupID) + `', memberGroups || ',` + itoa(groupID) + `') WHERE ID_BOARD IN (` + joinInts(boardAccess) + `) AND NOT FIND_IN_SET(` + itoa(groupID) + `, memberGroups)`))
			}
		}

		// If this became a post group, clear explicit membership.
		if minPosts != -1 {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_GROUP = 0 WHERE ID_GROUP = ?`), groupID)
			c.rewriteCSVColumn("members", "ID_MEMBER", "additionalGroups", []int{groupID})
		}

		c.updateStatsPostgroups(0)
		c.redirectExit("action=membergroups")
	}

	var name, color, stars string
	var minPosts, maxMessages int
	err := a.DB.QueryRow(a.Q(`SELECT groupName, minPosts, onlineColor, maxMessages, stars FROM {$db_prefix}membergroups WHERE ID_GROUP = ? LIMIT 1`), groupID).Scan(&name, &minPosts, &color, &maxMessages, &stars)
	if err != nil {
		c.fatalLangError("membergroup_does_not_exist", false)
	}
	starParts := strings.SplitN(stars, "#", 2)
	starImage := ""
	if len(starParts) > 1 {
		starImage = starParts[1]
	}

	page := &MGEditCtx{
		ID: groupID, Name: name, EditableName: Htmlspecialchars(name), Color: color,
		MinPosts: minPosts, MaxMessages: maxMessages, StarCount: atoi(starParts[0]), StarImage: starImage,
		IsPostGroup:    minPosts != -1,
		AllowPostGroup: groupID == 2 || groupID > 4,
		AllowDelete:    groupID == 2 || groupID > 4,
	}
	c.Page = page

	if groupID == 2 || groupID > 3 {
		rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name, childLevel, FIND_IN_SET(?, memberGroups) AS can_access FROM {$db_prefix}boards`), groupID)
		if err == nil {
			for rows.Next() {
				var b MGBoard
				var canAccess int
				rows.Scan(&b.ID, &b.Name, &b.ChildLevel, &canAccess)
				b.Selected = canAccess != 0
				page.Boards = append(page.Boards, b)
			}
			rows.Close()
		}
	}

	c.SubTemplate = templateMembergroupEdit
	c.PageTitle = c.Txt("membergroups_edit_group")
}

// rewriteCSVColumnWhere is like rewriteCSVColumn but with a custom WHERE.
func (c *Ctx) rewriteCSVColumnWhere(table, idCol, csvCol string, groups []int, where string) {
	a := c.App
	rows, err := a.DB.Query(a.Q(`SELECT ` + idCol + `, ` + csvCol + ` FROM {$db_prefix}` + table + ` WHERE ` + where))
	if err != nil {
		return
	}
	type upd struct {
		id  int
		csv string
	}
	var updates []upd
	for rows.Next() {
		var u upd
		rows.Scan(&u.id, &u.csv)
		updates = append(updates, u)
	}
	rows.Close()
	for _, u := range updates {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}`+table+` SET `+csvCol+` = ? WHERE `+idCol+` = ?`), csvDiff(u.csv, groups), u.id)
	}
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ---- members ----

type MGMember struct {
	ID          int
	Name        string
	Email       string
	IP          string
	Registered  string
	LastOnline  string
	Posts       int
	IsActivated bool
}

type MGMembersCtx struct {
	GroupID       int
	GroupName     string
	Assignable    bool
	IsPostGroup   bool
	Members       []MGMember
	SortBy        string
	SortDirection string
	PageIndex     string
	Start         int
	PageTitle     string
}

var reQuotedName = regexp.MustCompile(`"([^"]+)"`)

func (c *Ctx) MembergroupMembers() {
	a := c.App
	scripturl := a.ScriptURL
	groupID := c.REQUEST.Int("group")

	if groupID == -1 || groupID == 0 || groupID == 3 {
		c.fatalLangError("membergroup_does_not_exist", false)
	}

	var name string
	var assignable, isPostGroup int
	err := a.DB.QueryRow(a.Q(`SELECT groupName, IIF(minPosts = -1, 1, 0), IIF(minPosts != -1, 1, 0) FROM {$db_prefix}membergroups WHERE ID_GROUP = ? LIMIT 1`), groupID).Scan(&name, &assignable, &isPostGroup)
	if err != nil {
		c.fatalLangError("membergroup_does_not_exist", false)
	}
	if groupID == 1 && !c.allowedTo("admin_forum") {
		assignable = 0
	}

	page := &MGMembersCtx{GroupID: groupID, GroupName: name, Assignable: assignable != 0, IsPostGroup: isPostGroup != 0}
	c.Page = page

	if c.POST.Has("remove") && c.REQUEST.Arr("rem") != nil && page.Assignable {
		c.checkSession("post", "", true)
		var rem []int
		c.REQUEST.Arr("rem").Values(func(k string, v any) {
			s, _ := v.(string)
			rem = append(rem, atoi(s))
		})
		c.removeMembersFromGroups(rem, []int{groupID}, false)
	} else if c.REQUEST.Has("add") && c.REQUEST.Str("toAdd") != "" && page.Assignable {
		c.checkSession("post", "", true)
		toAdd := strings.ReplaceAll(Htmlspecialchars(c.REQUEST.Str("toAdd")), "&quot;", `"`)
		var names []string
		for _, m := range reQuotedName.FindAllStringSubmatch(toAdd, -1) {
			names = append(names, m[1])
		}
		rest := reQuotedName.ReplaceAllString(toAdd, "")
		names = append(names, strings.Split(rest, ",")...)
		var lower []string
		seen := map[string]bool{}
		for _, n := range names {
			n = strings.ToLower(strings.TrimSpace(n))
			if n != "" && !seen[n] {
				seen[n] = true
				lower = append(lower, n)
			}
		}
		if len(lower) > 0 {
			var members []int
			rows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE LOWER(memberName) IN (` + quoteList(lower) + `) OR LOWER(realName) IN (` + quoteList(lower) + `)`))
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					members = append(members, id)
				}
				rows.Close()
			}
			typ := "auto"
			if c.POST.Has("additional") {
				typ = "only_additional"
			}
			c.addMembersToGroup(members, groupID, typ)
		}
	}

	sortMethods := map[string]string{"name": "realName", "email": "emailAddress", "active": "lastLogin", "registered": "dateRegistered", "posts": "posts"}
	sortBy := "name"
	querySort := "realName"
	if s := c.REQUEST.Str("sort"); s != "" {
		if col, ok := sortMethods[s]; ok {
			sortBy = s
			querySort = col
		}
	}
	page.SortBy = sortBy
	page.SortDirection = "up"
	dir := "ASC"
	if c.REQUEST.Has("desc") {
		page.SortDirection = "down"
		dir = "DESC"
	}

	whereGroup := "ID_GROUP = " + itoa(groupID) + " OR FIND_IN_SET(" + itoa(groupID) + ", additionalGroups)"
	if isPostGroup != 0 {
		whereGroup = "ID_POST_GROUP = " + itoa(groupID)
	}

	var total int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE ` + whereGroup)).Scan(&total)

	maxMembers := a.SettingInt("defaultMaxMembers")
	if maxMembers == 0 {
		maxMembers = 30
	}
	descParam := ""
	if c.REQUEST.Has("desc") {
		descParam = ";desc"
	}
	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=membergroups;sa=members;group="+itoa(groupID)+";sort="+sortBy+descParam, c.REQUEST.Int("start"), total, maxMembers, false)
	page.Start = c.REQUEST.Int("start")

	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, realName, emailAddress, memberIP, dateRegistered, lastLogin, posts, is_activated
		FROM {$db_prefix}members
		WHERE `+whereGroup+`
		ORDER BY `+querySort+` `+dir+`
		LIMIT ? OFFSET ?`), maxMembers, page.Start)
	if err == nil {
		for rows.Next() {
			var id, posts, isActivated int
			var realName, email, ip string
			var dateReg, lastLogin int64
			rows.Scan(&id, &realName, &email, &ip, &dateReg, &lastLogin, &posts, &isActivated)
			lastOnline := c.Txt("never")
			if lastLogin != 0 {
				lastOnline = c.timeformat(lastLogin)
			}
			if isActivated%10 != 1 {
				lastOnline = `<i title="` + c.Txt("not_activated") + `">` + lastOnline + `</i>`
			}
			page.Members = append(page.Members, MGMember{
				ID: id, Posts: posts, IsActivated: isActivated%10 == 1,
				Name:       `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + realName + `</a>`,
				Email:      `<a href="mailto:` + email + `">` + email + `</a>`,
				IP:         `<a href="` + scripturl + `?action=trackip;searchip=` + ip + `">` + ip + `</a>`,
				Registered: c.timeformat(dateReg), LastOnline: lastOnline,
			})
		}
		rows.Close()
	}

	page.PageTitle = c.Txt("membergroups_members_title") + ": " + name
	c.PageTitle = page.PageTitle
	c.SubTemplate = templateMembergroupMembers
}

// ---- settings ----

func (c *Ctx) ModifyMembergroupSettings() {
	c.PageTitle = c.Txt("membergroups_settings")
	c.SubTemplate = templateMembergroupSettings

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)
		c.saveInlinePermissions([]string{"manage_membergroups"})
	}
	c.initInlinePermissions([]string{"manage_membergroups"}, []int{-1})
}
