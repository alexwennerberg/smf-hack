package app

// Port of Sources/Reports.php: the admin report generator (?action=reports) —
// the report-type chooser plus the boards / board permissions / member groups /
// group permissions / staff reports, all built through the newTable/addData/
// addSeperator/setKeys/finishTables table framework (ported here as
// *reportBuilder). The print-friendly view (st=print) renders through a
// reports_print layer (the same body as the PHP print template; the layer is
// named distinctly so it doesn't clash with the generic print layer).

import "strings"

func init() {
	registerAction("reports", (*Ctx).ReportsMain)
	layerFuncs["reports_print_above"] = templateReportsPrintAbove
	layerFuncs["reports_print_below"] = templateReportsPrintBelow
}

// ReportType is one entry of $context['report_types'].
type ReportType struct {
	ID          string
	Title       string
	Description string
	IsFirst     bool
}

// reportsPage backs template_report_type.
type reportsPage struct {
	ReportTypes []ReportType
}

func (c *Ctx) ReportsMain() {
	c.isAllowedTo("admin_forum")
	c.loadLanguage("Reports")
	c.adminIndex("generate_reports")
	c.PageTitle = c.Txt("generate_reports")

	order := []string{"boards", "board_perms", "member_groups", "group_perms", "staff"}
	funcs := map[string]func(){
		"boards":        c.BoardReport,
		"board_perms":   c.BoardPermissionsReport,
		"member_groups": c.MemberGroupsReport,
		"group_perms":   c.GroupPermissionsReport,
		"staff":         c.StaffReport,
	}
	var types []ReportType
	for i, k := range order {
		desc := ""
		if c.TxtHas("gr_type_desc_" + k) {
			desc = c.Txt("gr_type_desc_" + k)
		}
		types = append(types, ReportType{ID: k, Title: c.Txt("gr_type_" + k), Description: desc, IsFirst: i == 0})
	}

	rt := c.REQUEST.Str("rt")
	if rt == "" || funcs[rt] == nil {
		c.Page = &reportsPage{ReportTypes: types}
		c.SubTemplate = templateReportType
		return
	}

	c.reports = &reportBuilder{}

	st := c.REQUEST.Str("st")
	if st == "print" {
		c.TemplateLayers = []string{"reports_print"}
		c.SubTemplate = templateReportsPrint
	} else {
		c.SubTemplate = templateReportsMain
	}

	c.reportType = rt
	c.PageTitle += " - " + c.Txt("gr_type_"+rt)
	funcs[rt]()
	c.reports.finish()
	c.Page = c.reports
}

// ---- the table builder ----

type rptCell struct {
	Value     string
	Seperator bool
	Style     string
}

type rptKV struct {
	Key   string
	Value string
}

type rptTable struct {
	Title        string
	DefaultValue string
	ShadeLeft    bool
	ShadeTop     bool
	WidthNormal  string
	WidthShaded  string
	AlignNormal  string
	AlignShaded  string

	colMode  bool
	colOrder []string
	colCells map[string][]rptCell

	Data        [][]rptCell
	ColumnCount int
	MaxWidth    string
}

type reportBuilder struct {
	Tables  []*rptTable
	cur     int
	keys    []rptKV // ordered keys (value unused); empty = no keys
	hasKeys bool
	method  string // "rows" | "cols"
}

func (b *reportBuilder) newTable(title, defaultValue, shading, widthNormal, alignNormal, widthShaded, alignShaded string) {
	t := &rptTable{
		Title:        title,
		DefaultValue: defaultValue,
		ShadeLeft:    shading == "all" || shading == "left",
		ShadeTop:     shading == "all" || shading == "top",
		WidthNormal:  widthNormal,
		WidthShaded:  widthShaded,
		AlignNormal:  alignNormal,
		AlignShaded:  alignShaded,
		colCells:     map[string][]rptCell{},
	}
	b.Tables = append(b.Tables, t)
	b.cur = len(b.Tables) - 1
}

func (b *reportBuilder) setKeys(method string, keys []rptKV) {
	b.method = method
	if method != "rows" && method != "cols" {
		b.method = "cols"
	}
	b.keys = keys
	b.hasKeys = len(keys) > 0
}

func lookupKV(data []rptKV, key string) (string, bool) {
	for _, kv := range data {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

func (b *reportBuilder) addData(inc []rptKV) {
	if len(b.Tables) == 0 {
		b.newTable("", "", "all", "auto", "center", "auto", "auto")
	}
	t := b.Tables[b.cur]

	// Build the ordered cell list.
	var cells []rptCell
	var cellKeys []string
	if b.hasKeys {
		for _, k := range b.keys {
			v, _ := lookupKV(inc, k.Key)
			if empty(v) {
				v = t.DefaultValue
			}
			cells = append(cells, rptCell{Value: v, Seperator: strings.HasPrefix(k.Key, "#sep#")})
			cellKeys = append(cellKeys, k.Key)
		}
	} else {
		for _, kv := range inc {
			cells = append(cells, rptCell{Value: kv.Value, Seperator: strings.HasPrefix(kv.Key, "#sep#")})
			cellKeys = append(cellKeys, kv.Key)
		}
	}

	if b.method == "" || b.method == "rows" {
		t.Data = append(t.Data, cells)
	} else {
		t.colMode = true
		for i, k := range cellKeys {
			if _, ok := t.colCells[k]; !ok {
				t.colOrder = append(t.colOrder, k)
			}
			t.colCells[k] = append(t.colCells[k], cells[i])
		}
	}
}

func (b *reportBuilder) addSeperator(title string) {
	if len(b.Tables) == 0 {
		return
	}
	t := b.Tables[b.cur]
	t.Data = append(t.Data, []rptCell{{Seperator: true, Value: title}})
}

func (b *reportBuilder) finish() {
	for _, t := range b.Tables {
		if t.colMode {
			for _, k := range t.colOrder {
				t.Data = append(t.Data, t.colCells[k])
			}
		}
		t.ColumnCount = 0
		if len(t.Data) > 0 {
			t.ColumnCount = len(t.Data[0])
		}
		if t.ShadeLeft && t.WidthShaded != "auto" && t.WidthNormal != "auto" {
			t.MaxWidth = itoa(atoi(t.WidthShaded) + (t.ColumnCount-1)*atoi(t.WidthNormal))
		} else if t.WidthNormal != "auto" {
			t.MaxWidth = itoa(t.ColumnCount * atoi(t.WidthNormal))
		} else {
			t.MaxWidth = "auto"
		}
	}
}

// ---- reports ----

func (c *Ctx) BoardReport() {
	a := c.App
	b := c.reports

	// Moderators per board.
	moderators := map[int][]string{}
	if rows, err := a.DB.Query(a.Q(`
		SELECT mods.ID_BOARD, mem.realName
		FROM {$db_prefix}moderators AS mods, {$db_prefix}members AS mem
		WHERE mem.ID_MEMBER = mods.ID_MEMBER`)); err == nil {
		for rows.Next() {
			var bid int
			var name string
			rows.Scan(&bid, &name)
			moderators[bid] = append(moderators[bid], name)
		}
		rows.Close()
	}

	// Membergroup names.
	groups := map[string]string{"-1": c.Txt("28"), "0": c.Txt("full_member")}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, groupName, onlineColor FROM {$db_prefix}membergroups`)); err == nil {
		for rows.Next() {
			var gid int
			var name, color string
			rows.Scan(&gid, &name, &color)
			if color == "" {
				groups[itoa(gid)] = name
			} else {
				groups[itoa(gid)] = `<span style="color: ` + color + `">` + name + `</span>`
			}
		}
		rows.Close()
	}

	boardSettings := []rptKV{
		{"category", c.Txt("board_category")},
		{"parent", c.Txt("board_parent")},
		{"num_topics", c.Txt("board_num_topics")},
		{"num_posts", c.Txt("board_num_posts")},
		{"count_posts", c.Txt("board_count_posts")},
		{"theme", c.Txt("board_theme")},
		{"override_theme", c.Txt("board_override_theme")},
		{"moderators", c.Txt("board_moderators")},
		{"groups", c.Txt("board_groups")},
	}

	b.setKeys("cols", nil)

	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name, b.numPosts, b.numTopics, b.countPosts, b.memberGroups, b.override_theme,
			c.name AS catName, IFNULL(par.name, ?) AS parentName, IFNULL(th.value, ?) AS themeName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
			LEFT JOIN {$db_prefix}boards AS par ON (par.ID_BOARD = b.ID_PARENT)
			LEFT JOIN {$db_prefix}themes AS th ON (th.ID_THEME = b.ID_THEME AND th.variable = 'name')`),
		c.Txt("none"), c.Txt("none"))
	if err == nil {
		for rows.Next() {
			var bid, numPosts, numTopics, countPosts, overrideTheme int
			var name, memberGroups, catName, parentName, themeName string
			rows.Scan(&bid, &name, &numPosts, &numTopics, &countPosts, &memberGroups, &overrideTheme, &catName, &parentName, &themeName)

			b.newTable(name, "", "left", "auto", "left", "200", "left")
			b.addData(boardSettings)

			yn := func(v bool) string {
				if v {
					return c.Txt("yes")
				}
				return c.Txt("no")
			}
			mods := c.Txt("none")
			if len(moderators[bid]) > 0 {
				mods = strings.Join(moderators[bid], ", ")
			}
			var allowed []string
			for _, g := range strings.Split(memberGroups, ",") {
				if name, ok := groups[g]; ok {
					allowed = append(allowed, name)
				}
			}
			b.addData([]rptKV{
				{"category", catName},
				{"parent", parentName},
				{"num_posts", itoa(numPosts)},
				{"num_topics", itoa(numTopics)},
				{"count_posts", yn(countPosts == 0)},
				{"theme", themeName},
				{"override_theme", yn(overrideTheme != 0)},
				{"moderators", mods},
				{"groups", strings.Join(allowed, ", ")},
			})
		}
		rows.Close()
	}
}

func (c *Ctx) BoardPermissionsReport() {
	a := c.App
	b := c.reports

	boardClause := "1"
	if c.REQUEST.Has("boards") {
		boardClause = "ID_BOARD IN (" + joinInts(c.reqIntList("boards")) + ")"
	}
	reqGroups, hasGroups := c.reqIntListOpt("groups")
	groupClause := "1"
	if hasGroups {
		groupClause = "ID_GROUP IN (" + joinInts(reqGroups) + ")"
	}

	type boardInfo struct {
		name           string
		localPerms     bool
		permissionMode string
	}
	boards := map[int]boardInfo{0: {name: c.Txt("global_boards"), localPerms: true}}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name, permission_mode FROM {$db_prefix}boards WHERE ` + boardClause)); err == nil {
		for rows.Next() {
			var bid, pmode int
			var name string
			rows.Scan(&bid, &name, &pmode)
			pm := "normal"
			if a.SettingEmpty("permission_enable_by_board") {
				switch pmode {
				case 0:
					pm = "normal"
				case 2:
					pm = "no_polls"
				case 3:
					pm = "reply_only"
				default:
					pm = "read_only"
				}
			}
			boards[bid] = boardInfo{name: name, localPerms: !a.SettingEmpty("permission_enable_by_board") && pmode == 1, permissionMode: pm}
		}
		rows.Close()
	}

	// Membergroups (ordered).
	var memberGroups []rptKV
	memberGroups = append(memberGroups, rptKV{"col", ""})
	if !hasGroups || inInts(reqGroups, -1) || inInts(reqGroups, 0) {
		memberGroups = append(memberGroups, rptKV{"-1", c.Txt("membergroups_guests")}, rptKV{"0", c.Txt("membergroups_members")})
	}
	pgClause := ""
	if a.SettingEmpty("permission_enable_postgroups") {
		pgClause = " AND minPosts = -1"
	}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, groupName FROM {$db_prefix}membergroups WHERE ` + groupClause + ` AND ID_GROUP != 1` + pgClause + ` ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`)); err == nil {
		for rows.Next() {
			var gid int
			var name string
			rows.Scan(&gid, &name)
			memberGroups = append(memberGroups, rptKV{itoa(gid), name})
		}
		rows.Close()
	}

	b.setKeys("rows", memberGroups)

	// board -> group -> perm -> addDeny
	boardPerms := map[int]map[int]map[string]int{}
	permTitles := []rptKV{} // ordered unique permissions
	permSeen := map[string]bool{}
	denyClause := ""
	if a.SettingEmpty("permission_enable_deny") {
		denyClause = " AND addDeny = 1"
	}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, ID_GROUP, addDeny, permission FROM {$db_prefix}board_permissions WHERE ` + boardClause + ` AND ` + groupClause + denyClause + ` ORDER BY ID_BOARD, permission`)); err == nil {
		for rows.Next() {
			var bid, gid, addDeny int
			var perm string
			rows.Scan(&bid, &gid, &addDeny, &perm)
			if boardPerms[bid] == nil {
				boardPerms[bid] = map[int]map[string]int{}
			}
			if boardPerms[bid][gid] == nil {
				boardPerms[bid][gid] = map[string]int{}
			}
			boardPerms[bid][gid][perm] = addDeny
			if !permSeen[perm] {
				permSeen[perm] = true
				title := perm
				if c.TxtHas("board_perms_name_" + perm) {
					title = c.Txt("board_perms_name_" + perm)
				}
				permTitles = append(permTitles, rptKV{perm, title})
			}
		}
		rows.Close()
	}

	// Board ID order for stable output.
	for _, board := range orderedBoardIDs(boardPerms) {
		groupsForBoard := boardPerms[board]
		if board != 0 && !boards[board].localPerms {
			continue
		}
		b.newTable(boards[board].name, "x", "all", "100", "center", "200", "left")
		b.addData(memberGroups)
		b.addSeperator(c.Txt("board_perms_permission"))

		for _, permKV := range permTitles {
			perm := permKV.Key
			identicalGlobal := board != 0
			curData := []rptKV{{"col", permKV.Value}}

			for _, mg := range memberGroups {
				if mg.Key == "col" {
					continue
				}
				gid := atoi(mg.Key)
				gp := groupsForBoard[gid]
				var cellVal string
				localVal, hasLocal := gp[perm]
				if hasLocal {
					if globalVal, ok := boardPerms[0][gid][perm]; !ok || globalVal != localVal {
						identicalGlobal = false
					}
					if localVal == 0 {
						cellVal = `<span style="color: red;">` + c.Txt("board_perms_deny") + `</span>`
					} else if localVal == 1 {
						cellVal = `<span style="color: darkgreen;">` + c.Txt("board_perms_allow") + `</span>`
					} else {
						cellVal = "x"
					}
				} else {
					if globalVal, ok := boardPerms[0][gid][perm]; ok && globalVal != 0 {
						identicalGlobal = false
					}
					cellVal = "x"
				}
				// Embolden when different from the global board.
				globalVal, hasGlobal := boardPerms[0][gid][perm]
				if !(hasGlobal == hasLocal && globalVal == localVal) {
					cellVal = "<b>" + cellVal + "</b>"
				}
				curData = append(curData, rptKV{mg.Key, cellVal})
			}
			if !identicalGlobal || !c.REQUEST.Has("show_differences") {
				b.addData(curData)
			}
		}
	}

	// "Simple" local-permission boards.
	b.setKeys("rows", nil)
	for _, id := range orderedBoardInfoIDs(boards) {
		board := boards[id]
		if id != 0 && board.permissionMode != "" && board.permissionMode != "normal" {
			b.newTable(board.name, "x", "top", "auto", "center", "auto", "auto")
			b.addData([]rptKV{{"0", "<b>" + c.Txt("board_perms_group_"+board.permissionMode) + "</b>"}})
		}
	}
}

func (c *Ctx) MemberGroupsReport() {
	a := c.App
	b := c.reports

	type boardRow struct {
		id     int
		name   string
		pmode  string
		groups []int
	}
	var boardOrder []int
	boardsByID := map[int]boardRow{}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name, memberGroups, permission_mode FROM {$db_prefix}boards`)); err == nil {
		for rows.Next() {
			var bid, pmode int
			var name, mg string
			rows.Scan(&bid, &name, &mg, &pmode)
			pm := c.Txt("permission_mode_normal")
			if a.SettingEmpty("permission_enable_by_board") {
				switch pmode {
				case 0:
					pm = c.Txt("permission_mode_normal")
				case 2:
					pm = c.Txt("permission_mode_no_polls")
				case 3:
					pm = c.Txt("permission_mode_reply_only")
				default:
					pm = c.Txt("permission_mode_read_only")
				}
			}
			grps := []int{1, 3}
			for _, g := range strings.Split(mg, ",") {
				if g != "" {
					grps = append(grps, atoi(g))
				}
			}
			boardOrder = append(boardOrder, bid)
			boardsByID[bid] = boardRow{id: bid, name: name, pmode: pm, groups: grps}
		}
		rows.Close()
	}

	mgSettings := []rptKV{
		{"name", ""},
		{"#sep#1", c.Txt("member_group_settings")},
		{"color", c.Txt("member_group_color")},
		{"minPosts", c.Txt("member_group_minPosts")},
		{"maxMessages", c.Txt("member_group_maxMessages")},
		{"stars", c.Txt("member_group_stars")},
		{"#sep#2", c.Txt("member_group_access")},
	}
	for _, bid := range boardOrder {
		mgSettings = append(mgSettings, rptKV{"board_" + itoa(bid), boardsByID[bid].name})
	}

	b.setKeys("cols", mgSettings)
	b.newTable(c.Txt("gr_type_member_groups"), "-", "all", "100", "center", "200", "left")
	b.addData(mgSettings)

	type groupRow struct {
		id          int
		name        string
		color       string
		minPosts    int
		maxMessages string
		stars       string
		canModerate int
	}
	rowsData := []groupRow{
		{id: -1, name: c.Txt("membergroups_guests"), color: "-", minPosts: -1},
		{id: 0, name: c.Txt("membergroups_members"), color: "-", minPosts: -1},
	}
	canModSel := ""
	canModJoin := ""
	if a.SettingEmpty("permission_enable_by_board") {
		canModSel = ", IIF(bp.permission IS NOT NULL OR mg.ID_GROUP = 1, 1, 0) AS can_moderate"
		canModJoin = ` LEFT JOIN {$db_prefix}board_permissions AS bp ON (bp.ID_GROUP = mg.ID_GROUP AND bp.ID_BOARD = 0 AND bp.permission = 'moderate_board')`
	}
	if rows, err := a.DB.Query(a.Q(`SELECT mg.ID_GROUP, mg.groupName, mg.onlineColor, mg.minPosts, mg.maxMessages, mg.stars` + canModSel + ` FROM {$db_prefix}membergroups AS mg` + canModJoin + ` ORDER BY mg.minPosts, IIF(mg.ID_GROUP < 4, mg.ID_GROUP, 4), mg.groupName`)); err == nil {
		for rows.Next() {
			var g groupRow
			var maxMsg *int
			if a.SettingEmpty("permission_enable_by_board") {
				rows.Scan(&g.id, &g.name, &g.color, &g.minPosts, &maxMsg, &g.stars, &g.canModerate)
			} else {
				rows.Scan(&g.id, &g.name, &g.color, &g.minPosts, &maxMsg, &g.stars)
			}
			if maxMsg != nil {
				g.maxMessages = itoa(*maxMsg)
			}
			rowsData = append(rowsData, g)
		}
		rows.Close()
	}

	for _, g := range rowsData {
		color := "-"
		if g.color != "" && g.color != "-" {
			color = `<span style="color: ` + g.color + `;">` + g.color + `</span>`
		}
		minPosts := itoa(g.minPosts)
		if g.minPosts == -1 {
			minPosts = "N/A"
		}
		stars := ""
		parts := strings.Split(g.stars, "#")
		if len(parts) >= 2 && parts[0] != "" && parts[0] != "0" && parts[1] != "" {
			stars = strings.Repeat(`<img src="`+c.Theme.ImagesURL()+`/`+parts[1]+`" alt="*" border="0" />`, atoi(parts[0]))
		}
		group := []rptKV{
			{"name", g.name},
			{"color", color},
			{"minPosts", minPosts},
			{"maxMessages", g.maxMessages},
			{"stars", stars},
		}
		for _, bid := range boardOrder {
			board := boardsByID[bid]
			cell := "x"
			if inInts(board.groups, g.id) {
				var label string
				if a.SettingEmpty("permission_enable_by_board") {
					if g.canModerate == 0 {
						label = board.pmode
					} else {
						label = c.Txt("permission_mode_normal")
					}
				} else {
					label = c.Txt("board_perms_allow")
				}
				cell = `<span style="color: darkgreen;">` + label + `</span>`
			}
			group = append(group, rptKV{"board_" + itoa(bid), cell})
		}
		b.addData(group)
	}
}

func (c *Ctx) GroupPermissionsReport() {
	a := c.App
	b := c.reports

	reqGroups, hasGroups := c.reqIntListOpt("groups")
	clause := "ID_GROUP != 3"
	if hasGroups {
		reqGroups = diffInts(reqGroups, []int{3})
		clause = "ID_GROUP IN (" + joinInts(reqGroups) + ")"
	}

	var groups []rptKV
	groups = append(groups, rptKV{"col", ""})
	if !hasGroups || inInts(reqGroups, -1) || inInts(reqGroups, 0) {
		groups = append(groups, rptKV{"-1", c.Txt("membergroups_guests")}, rptKV{"0", c.Txt("membergroups_members")})
	}
	pgClause := ""
	if a.SettingEmpty("permission_enable_postgroups") {
		pgClause = " AND minPosts = -1"
	}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, groupName FROM {$db_prefix}membergroups WHERE ` + clause + ` AND ID_GROUP != 1` + pgClause + ` ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`)); err == nil {
		for rows.Next() {
			var gid int
			var name string
			rows.Scan(&gid, &name)
			groups = append(groups, rptKV{itoa(gid), name})
		}
		rows.Close()
	}

	b.setKeys("rows", groups)
	b.newTable(c.Txt("gr_type_group_perms"), "-", "all", "100", "center", "200", "left")
	b.addData(groups)
	b.addSeperator(c.Txt("board_perms_permission"))

	denyClause := ""
	if a.SettingEmpty("permission_enable_deny") {
		denyClause = " AND addDeny = 1"
	}
	rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, addDeny, permission FROM {$db_prefix}permissions WHERE ` + clause + denyClause + ` ORDER BY permission`))
	lastPermission := ""
	started := false
	var curData []rptKV
	if err == nil {
		for rows.Next() {
			var gid, addDeny int
			var perm string
			rows.Scan(&gid, &addDeny, &perm)
			if perm != lastPermission {
				if started {
					b.addData(curData)
				}
				title := perm
				if c.TxtHas("group_perms_name_" + perm) {
					title = c.Txt("group_perms_name_" + perm)
				}
				curData = []rptKV{{"col", title}}
				lastPermission = perm
				started = true
			}
			if addDeny != 0 {
				curData = append(curData, rptKV{itoa(gid), `<span style="color: darkgreen;">` + c.Txt("board_perms_allow") + `</span>`})
			} else {
				curData = append(curData, rptKV{itoa(gid), `<span style="color: red;">` + c.Txt("board_perms_deny") + `</span>`})
			}
		}
		rows.Close()
	}
	if started {
		b.addData(curData)
	}
}

func (c *Ctx) StaffReport() {
	a := c.App
	b := c.reports

	boards := map[int]string{}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name FROM {$db_prefix}boards`)); err == nil {
		for rows.Next() {
			var bid int
			var name string
			rows.Scan(&bid, &name)
			boards[bid] = name
		}
		rows.Close()
	}

	moderators := map[int][]int{}
	var localMods []int
	if rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, ID_MEMBER FROM {$db_prefix}moderators`)); err == nil {
		for rows.Next() {
			var bid, mid int
			rows.Scan(&bid, &mid)
			moderators[mid] = append(moderators[mid], bid)
			localMods = append(localMods, mid)
		}
		rows.Close()
	}

	globalMods := intersectInts(intersectInts(intersectInts(
		c.membersAllowedTo("moderate_board", 0),
		c.membersAllowedTo("post_new", 0)),
		c.membersAllowedTo("remove_any", 0)),
		c.membersAllowedTo("modify_any", 0))

	allStaff := uniqueInts(concatInts(
		c.membersAllowedTo("admin_forum", -1),
		c.membersAllowedTo("manage_membergroups", -1),
		c.membersAllowedTo("manage_permissions", -1),
		localMods, globalMods))

	if len(allStaff) > 300 {
		c.fatalLangError("report_error_too_many_staff", true)
	}

	groups := map[int]string{0: c.Txt("full_member")}
	if rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, groupName, onlineColor FROM {$db_prefix}membergroups`)); err == nil {
		for rows.Next() {
			var gid int
			var name, color string
			rows.Scan(&gid, &name, &color)
			if color == "" {
				groups[gid] = name
			} else {
				groups[gid] = `<span style="color: ` + color + `">` + name + `</span>`
			}
		}
		rows.Close()
	}

	staffSettings := []rptKV{
		{"position", c.Txt("report_staff_position")},
		{"moderates", c.Txt("report_staff_moderates")},
		{"posts", c.Txt("report_staff_posts")},
		{"last_login", c.Txt("report_staff_last_login")},
	}
	b.setKeys("cols", nil)

	if len(allStaff) == 0 {
		return
	}
	rows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER, realName, ID_GROUP, posts, lastLogin FROM {$db_prefix}members WHERE ID_MEMBER IN (` + joinInts(allStaff) + `) ORDER BY realName`))
	if err == nil {
		for rows.Next() {
			var mid, gid, posts int
			var realName string
			var lastLogin int64
			rows.Scan(&mid, &realName, &gid, &posts, &lastLogin)

			b.newTable(realName, "", "left", "auto", "left", "200", "center")
			b.addData(staffSettings)

			position := groups[0]
			if g, ok := groups[gid]; ok {
				position = g
			}
			var moderates string
			if inInts(globalMods, mid) {
				moderates = "<i>" + c.Txt("report_staff_all_boards") + "</i>"
			} else if bids, ok := moderators[mid]; ok {
				var names []string
				for _, bid := range bids {
					if n, ok := boards[bid]; ok {
						names = append(names, n)
					}
				}
				moderates = strings.Join(names, ", ")
			} else {
				moderates = "<i>" + c.Txt("report_staff_no_boards") + "</i>"
			}
			b.addData([]rptKV{
				{"position", position},
				{"moderates", moderates},
				{"posts", itoa(posts)},
				{"last_login", c.timeformat(lastLogin)},
			})
		}
		rows.Close()
	}
}

// ---- small helpers ----

func (c *Ctx) reqIntList(key string) []int {
	var out []int
	if arr := c.REQUEST.Arr(key); arr != nil {
		arr.Values(func(k string, v any) {
			s, _ := v.(string)
			out = append(out, atoi(s))
		})
	} else if s := c.REQUEST.Str(key); s != "" {
		for _, p := range strings.Split(s, ",") {
			out = append(out, atoi(p))
		}
	}
	return out
}

func (c *Ctx) reqIntListOpt(key string) ([]int, bool) {
	if !c.REQUEST.Has(key) {
		return nil, false
	}
	return c.reqIntList(key), true
}

func concatInts(lists ...[]int) []int {
	var out []int
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

func orderedBoardIDs(m map[int]map[int]map[string]int) []int {
	var ids []int
	for k := range m {
		ids = append(ids, k)
	}
	sortIntsAsc(ids)
	return ids
}

func orderedBoardInfoIDs[T any](m map[int]T) []int {
	var ids []int
	for k := range m {
		ids = append(ids, k)
	}
	sortIntsAsc(ids)
	return ids
}

func sortIntsAsc(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
