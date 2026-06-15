package app

// Port of Sources/ManageBoards.php: the board/category admin
// (?action=manageboards) — list/move, edit category, edit board, and general
// settings. Board-tree operations live in subs_boards_tree.go. Inline board
// permissions (theme_inline_permissions) land with ManagePermissions; until
// then EditBoardSettings hides that block (can_change_permissions = false).

import (
	"regexp"
	"strings"
)

func init() {
	registerAction("manageboards", (*Ctx).ManageBoards)
}

// reAmpEntity escapes a bare & not part of an entity (PHP's
// ~[&]([^;]{8}|[^;]{0,8}$)~ -> &amp;$1).
var reAmpEntity = regexp.MustCompile(`[&]([^;]{8}|[^;]{0,8}$)`)

func ampEscape(s string) string { return reAmpEntity.ReplaceAllString(s, "&amp;$1") }

// ManageBoards is ManageBoards(): the dispatcher.
func (c *Ctx) ManageBoards() {
	c.loadLanguage("ManageBoards")

	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"board":    {c.EditBoard, "manage_boards"},
		"board2":   {c.EditBoard2, "manage_boards"},
		"cat":      {c.EditCategory, "manage_boards"},
		"cat2":     {c.EditCategory2, "manage_boards"},
		"main":     {c.ManageBoardsMain, "manage_boards"},
		"move":     {c.ManageBoardsMain, "manage_boards"},
		"newcat":   {c.EditCategory, "manage_boards"},
		"newboard": {c.EditBoard, "manage_boards"},
		"settings": {c.EditBoardSettings, "admin_forum"},
	}

	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		if c.allowedTo("manage_boards") {
			sa = "main"
		} else {
			sa = "settings"
		}
	}
	c.REQUEST.Set("sa", sa)

	c.isAllowedTo(subs[sa].perm)
	c.adminIndex("manage_boards")

	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("41"), Help: "manage_boards", Description: c.Txt("677")}
	if c.allowedTo("manage_boards") {
		tabs.Tabs = append(tabs.Tabs,
			AdminTab{Title: c.Txt("boardsEdit"), Description: c.Txt("677"), Href: scripturl + "?action=manageboards", IsSelected: sa != "newcat" && sa != "settings"},
			AdminTab{Title: c.Txt("mboards_new_cat"), Description: c.Txt("677"), Href: scripturl + "?action=manageboards;sa=newcat", IsSelected: sa == "newcat", IsLast: !c.allowedTo("admin_forum")})
	}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("settings"), Description: c.Txt("mboards_settings_desc"), Href: scripturl + "?action=manageboards;sa=settings", IsSelected: sa == "settings", IsLast: true})
	}
	c.AdminTabs = tabs

	subs[sa].fn()
}

// ---- list / move ----

// MBMoveLink is one move-destination spot.
type MBMoveLink struct {
	ChildLevel int
	Label      string
	Href       string
}

// MBBoard is one board row in the list.
type MBBoard struct {
	ID               int
	Name             string
	Description      string
	ChildLevel       int
	LocalPermissions bool
	Move             bool
	MoveLinks        []MBMoveLink
}

// MBCategory is one category in the list.
type MBCategory struct {
	ID       int
	Name     string
	Boards   []MBBoard
	MoveLink *MBMoveLink
}

// ManageBoardsCtx backs template_main.
type ManageBoardsCtx struct {
	Categories           []MBCategory
	MoveBoard            int
	MoveTitle            string
	CanManagePermissions bool
}

// ManageBoardsMain is ManageBoardsMain(): the board list (+ move handling).
func (c *Ctx) ManageBoardsMain() {
	// Execute a move requested via the UI.
	if c.REQUEST.Str("sa") == "move" && inStrings([]string{"child", "before", "after", "top"}, c.REQUEST.Str("move_to")) {
		c.checkSession("get", "", true)
		opts := &boardOptions{moveTo: c.REQUEST.Str("move_to"), hasMoveTo: true, moveFirstChild: true}
		if c.REQUEST.Str("move_to") == "top" {
			opts.targetCategory = c.REQUEST.Int("target_cat")
			opts.hasTargetCategory = true
		} else {
			opts.targetBoard = c.REQUEST.Int("target_board")
			opts.hasTargetBoard = true
		}
		c.modifyBoard(c.REQUEST.Int("src_board"), opts)
	}

	t := c.getBoardTree()

	page := &ManageBoardsCtx{}
	c.Page = page

	moveBoard := 0
	if c.REQUEST.Has("move") && t.Boards[c.REQUEST.Int("move")] != nil {
		moveBoard = c.REQUEST.Int("move")
	}
	page.MoveBoard = moveBoard

	for _, catID := range t.CatOrder {
		cat := t.Cats[catID]
		mbCat := MBCategory{ID: cat.ID, Name: cat.Name}
		moveCat := moveBoard != 0 && t.Boards[moveBoard].Category == catID
		boardIdx := map[int]int{}
		for _, boardID := range t.BoardList[catID] {
			b := t.Boards[boardID]
			boardIdx[boardID] = len(mbCat.Boards)
			mbCat.Boards = append(mbCat.Boards, MBBoard{
				ID: b.ID, Name: b.Name, Description: b.Description, ChildLevel: b.Level,
				LocalPermissions: b.UseLocalPermissions,
				Move:             moveCat && (boardID == moveBoard || t.isChildOf(boardID, moveBoard)),
			})
		}

		// Build the move-destination spots.
		if moveBoard != 0 {
			c.buildMoveLinks(&mbCat, t, catID, moveBoard, boardIdx)
		}
		page.Categories = append(page.Categories, mbCat)
	}

	if moveBoard != 0 {
		page.MoveTitle = phpSprintf(c.Txt("mboards_select_destination"), Htmlspecialchars(t.Boards[moveBoard].Name))
	}

	c.SubTemplate = templateManageBoardsMain
	c.PageTitle = c.Txt("41")
	page.CanManagePermissions = c.allowedTo("manage_permissions")
}

// buildMoveLinks ports the move-spot stack logic from ManageBoardsMain.
func (c *Ctx) buildMoveLinks(mbCat *MBCategory, t *boardTree, catID, moveBoard int, boardIdx map[int]int) {
	scripturl := c.App.ScriptURL
	boardList := t.BoardList[catID]

	href := func(moveTo string, targetBoard int) string {
		return scripturl + "?action=manageboards;sa=move;src_board=" + itoa(moveBoard) + ";target_board=" + itoa(targetBoard) + ";move_to=" + moveTo + ";sesc=" + c.Sc
	}

	prevChildLevel := 0
	prevIdx := -1
	var stack []MBMoveLink

	for _, boardID := range boardList {
		b := t.Boards[boardID]
		idx := boardIdx[boardID]

		if mbCat.MoveLink == nil {
			mbCat.MoveLink = &MBMoveLink{
				ChildLevel: 0,
				Label:      c.Txt("mboards_order_before") + " '" + Htmlspecialchars(b.Name) + "'",
				Href:       href("before", boardID),
			}
		}

		if !mbCat.Boards[idx].Move {
			mbCat.Boards[idx].MoveLinks = []MBMoveLink{
				{ChildLevel: b.Level, Label: c.Txt("mboards_order_after") + "'" + Htmlspecialchars(b.Name) + "'", Href: href("after", boardID)},
				{ChildLevel: b.Level + 1, Label: c.Txt("mboards_order_child_of") + " '" + Htmlspecialchars(b.Name) + "'", Href: href("child", boardID)},
			}
		}

		difference := b.Level - prevChildLevel
		if difference == 1 && prevIdx >= 0 && len(mbCat.Boards[prevIdx].MoveLinks) > 0 {
			// array_push(stack, array_shift(prev.move_links))
			stack = append(stack, mbCat.Boards[prevIdx].MoveLinks[0])
			mbCat.Boards[prevIdx].MoveLinks = mbCat.Boards[prevIdx].MoveLinks[1:]
		} else if difference < 0 && prevIdx >= 0 {
			for i := 0; i < -difference; i++ {
				if len(stack) == 0 {
					break
				}
				// array_unshift(prev.move_links, array_pop(stack))
				popped := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				mbCat.Boards[prevIdx].MoveLinks = append([]MBMoveLink{popped}, mbCat.Boards[prevIdx].MoveLinks...)
			}
		}

		prevIdx = idx
		prevChildLevel = b.Level
	}

	if len(stack) > 0 && prevIdx >= 0 {
		mbCat.Boards[prevIdx].MoveLinks = append(append([]MBMoveLink{}, stack...), mbCat.Boards[prevIdx].MoveLinks...)
	}

	if len(boardList) == 0 {
		mbCat.MoveLink = &MBMoveLink{
			ChildLevel: 0,
			Label:      c.Txt("mboards_order_before") + " '" + Htmlspecialchars(t.Cats[catID].Name) + "'",
			Href:       scripturl + "?action=manageboards;sa=move;src_board=" + itoa(moveBoard) + ";target_cat=" + itoa(catID) + ";move_to=top;sesc=" + c.Sc,
		}
	}
}

// ---- edit category ----

// ECOrder is one entry in the category-position select.
type ECOrder struct {
	ID       int
	Name     string
	Selected bool
	TrueName string
}

// EditCategoryCtx backs template_modify_category / confirm_category_delete.
type EditCategoryCtx struct {
	ID           int
	Name         string
	EditableName string
	CanCollapse  bool
	IsNew        bool
	IsEmpty      bool
	Children     []string
	Order        []ECOrder
	Delete       bool
}

// EditCategory is EditCategory().
func (c *Ctx) EditCategory() {
	t := c.getBoardTree()
	catID := c.REQUEST.Int("cat")

	page := &EditCategoryCtx{}
	c.Page = page

	// Start with "in first place".
	firstSelected := false
	if catID != 0 && t.Cats[catID] != nil {
		firstSelected = t.Cats[catID].IsFirst
	}
	page.Order = []ECOrder{{ID: 0, Name: c.Txt("mboards_order_first"), Selected: firstSelected}}

	if c.REQUEST.Str("sa") == "newcat" {
		page.ID = 0
		page.Name = c.Txt("mboards_new_cat_name")
		page.EditableName = Htmlspecialchars(c.Txt("mboards_new_cat_name"))
		page.CanCollapse = true
		page.IsNew = true
		page.IsEmpty = true
	} else if t.Cats[catID] == nil {
		c.redirectExit("action=manageboards")
	} else {
		cat := t.Cats[catID]
		page.ID = catID
		page.Name = cat.Name
		page.EditableName = Htmlspecialchars(cat.Name)
		page.CanCollapse = cat.CanCollapse != 0
		page.IsEmpty = len(cat.Children) == 0
		for _, child := range t.BoardList[catID] {
			b := t.Boards[child]
			page.Children = append(page.Children, strings.Repeat("-", b.Level)+" "+b.Name)
		}
	}

	prevCat := 0
	for _, cid := range t.CatOrder {
		tree := t.Cats[cid]
		if cid == catID && prevCat > 0 {
			for i := range page.Order {
				if page.Order[i].ID == prevCat {
					page.Order[i].Selected = true
				}
			}
		} else {
			page.Order = append(page.Order, ECOrder{ID: cid, Name: c.Txt("mboards_order_after") + tree.Name, TrueName: tree.Name})
		}
		prevCat = cid
	}

	if !c.REQUEST.Has("delete") {
		c.SubTemplate = templateModifyCategory
		if c.REQUEST.Str("sa") == "newcat" {
			c.PageTitle = c.Txt("mboards_new_cat_name")
		} else {
			c.PageTitle = c.Txt("catEdit")
		}
	} else {
		page.Delete = true
		c.SubTemplate = templateConfirmCategoryDelete
		c.PageTitle = c.Txt("mboards_delete_cat")
	}
}

// EditCategory2 is EditCategory2().
func (c *Ctx) EditCategory2() {
	c.checkSession("post", "", true)
	catID := c.POST.Int("cat")

	if c.POST.Has("edit") || c.POST.Has("add") {
		opts := &catOptions{}
		if c.POST.Has("cat_order") {
			opts.moveAfter = c.POST.Int("cat_order")
			opts.hasMoveAfter = true
		}
		name := ampEscape(c.POST.Str("cat_name"))
		opts.catName = &name
		coll := c.POST.Has("collapse")
		opts.isCollapsible = &coll

		if c.POST.Has("add") {
			c.createCategory(opts)
		} else {
			c.modifyCategory(catID, opts)
		}
	} else if c.POST.Has("delete") && !c.POST.Has("confirmation") && !c.POST.Has("empty") {
		c.EditCategory()
		return
	} else if c.POST.Has("delete") {
		if c.POST.Has("delete_action") && c.POST.Int("delete_action") == 1 {
			if c.POST.Int("cat_to") == 0 {
				c.fatalLangError("mboards_delete_error", true)
			}
			c.deleteCategories([]int{catID}, c.POST.Int("cat_to"))
		} else {
			c.deleteCategories([]int{catID}, -1)
		}
	}

	c.redirectExit("action=manageboards")
}

// ---- edit board ----

// EBGroup is one membergroup access option.
type EBGroup struct {
	ID          string
	Name        string
	Checked     bool
	IsPostGroup bool
}

// EBOrder is one entry in the board-position select.
type EBOrder struct {
	ID       int
	Name     string
	IsChild  bool
	Selected bool
}

// EBCategoryOpt is one category option for moving a board.
type EBCategoryOpt struct {
	ID       int
	Name     string
	Selected bool
}

// EBTheme is one theme option.
type EBTheme struct {
	ID   int
	Name string
}

// EditBoardCtx backs template_modify_board / confirm_board_delete.
type EditBoardCtx struct {
	IsNew           bool
	ID              int
	Name            string
	Description     string
	CountPosts      bool
	Theme           int
	OverrideTheme   int
	Category        int
	NoChildren      bool
	PermissionMode  string
	ModeratorList   string
	Groups          []EBGroup
	BoardOrder      []EBOrder
	Categories      []EBCategoryOpt
	Themes          []EBTheme
	Children        []string
	CanMoveChildren bool
	Delete          bool
}

// EditBoard is EditBoard().
func (c *Ctx) EditBoard() {
	a := c.App
	t := c.getBoardTree()

	boardID := c.REQUEST.Int("boardid")
	sa := c.REQUEST.Str("sa")
	if t.Boards[boardID] == nil {
		boardID = 0
		sa = "newboard"
	}

	page := &EditBoardCtx{}
	c.Page = page

	var curGroups []string // current memberGroups
	var curCategory int
	if sa == "newboard" {
		page.IsNew = true
		page.ID = 0
		page.Name = c.Txt("mboards_new_board_name")
		page.Description = ""
		page.CountPosts = true
		page.Theme = 0
		page.OverrideTheme = 0
		page.Category = c.REQUEST.Int("cat")
		page.NoChildren = true
		page.PermissionMode = "normal"
		curGroups = []string{"0", "-1"}
		curCategory = c.REQUEST.Int("cat")
	} else {
		b := t.Boards[boardID]
		page.ID = b.ID
		page.Name = Htmlspecialchars(b.Name)
		page.Description = Htmlspecialchars(b.Description)
		page.CountPosts = b.CountPosts
		page.Theme = b.Theme
		page.OverrideTheme = b.OverrideTheme
		page.Category = b.Category
		page.NoChildren = len(b.Children) == 0
		page.PermissionMode = b.PermissionMode
		curGroups = b.MemberGroups
		curCategory = b.Category
	}

	hasGroup := func(g string) bool { return inStrings(curGroups, g) }

	page.Groups = []EBGroup{
		{ID: "-1", Name: c.Txt("parent_guests_only"), Checked: hasGroup("-1")},
		{ID: "0", Name: c.Txt("parent_members_only"), Checked: hasGroup("0")},
	}

	rows, err := a.DB.Query(a.Q(`
		SELECT groupName, ID_GROUP, minPosts
		FROM {$db_prefix}membergroups
		WHERE ID_GROUP > 3 OR ID_GROUP = 2
		ORDER BY minPosts, ID_GROUP != 2, groupName`))
	if err == nil {
		for rows.Next() {
			var groupName string
			var idGroup, minPosts int
			rows.Scan(&groupName, &idGroup, &minPosts)
			if sa == "newboard" && minPosts == -1 {
				curGroups = append(curGroups, itoa(idGroup))
			}
			page.Groups = append(page.Groups, EBGroup{
				ID: itoa(idGroup), Name: strings.TrimSpace(groupName),
				Checked: inStrings(curGroups, itoa(idGroup)), IsPostGroup: minPosts != -1,
			})
		}
		rows.Close()
	}

	for _, bid := range t.BoardList[curCategory] {
		b := t.Boards[bid]
		if bid == boardID {
			page.BoardOrder = append(page.BoardOrder, EBOrder{
				ID: bid, Name: strings.Repeat("-", b.Level) + " (" + c.Txt("mboards_current_position") + ")", Selected: true,
			})
		} else {
			isChild := false
			if boardID != 0 {
				isChild = t.isChildOf(bid, boardID)
			}
			page.BoardOrder = append(page.BoardOrder, EBOrder{
				ID: bid, Name: strings.Repeat("-", b.Level) + " " + b.Name, IsChild: isChild,
			})
		}
	}

	if boardID != 0 {
		page.CanMoveChildren = false
		for _, ch := range t.Boards[boardID].Children {
			page.Children = append(page.Children, t.Boards[ch].Name)
		}
		for _, o := range page.BoardOrder {
			if !o.IsChild && !o.Selected {
				page.CanMoveChildren = true
			}
		}
	}

	for _, catID := range t.CatOrder {
		id := catID
		if catID == curCategory {
			id = 0
		}
		page.Categories = append(page.Categories, EBCategoryOpt{ID: id, Name: t.Cats[catID].Name, Selected: catID == curCategory})
	}

	var mods []string
	if boardID != 0 {
		mrows, err := a.DB.Query(a.Q(`
			SELECT mem.realName
			FROM {$db_prefix}moderators AS mods, {$db_prefix}members AS mem
			WHERE mods.ID_BOARD = ? AND mem.ID_MEMBER = mods.ID_MEMBER`), boardID)
		if err == nil {
			for mrows.Next() {
				var n string
				mrows.Scan(&n)
				mods = append(mods, n)
			}
			mrows.Close()
		}
	}
	if len(mods) > 0 {
		page.ModeratorList = "&quot;" + strings.Join(mods, "&quot;, &quot;") + "&quot;"
	}

	trows, err := a.DB.Query(a.Q(`SELECT ID_THEME AS id, value AS name FROM {$db_prefix}themes WHERE variable = 'name'`))
	if err == nil {
		for trows.Next() {
			var th EBTheme
			trows.Scan(&th.ID, &th.Name)
			page.Themes = append(page.Themes, th)
		}
		trows.Close()
	}

	if !c.REQUEST.Has("delete") {
		c.SubTemplate = templateModifyBoard
		c.PageTitle = c.Txt("boardsEdit")
	} else {
		page.Delete = true
		c.SubTemplate = templateConfirmBoardDelete
		c.PageTitle = c.Txt("mboards_delete_board")
	}
}

// EditBoard2 is EditBoard2().
func (c *Ctx) EditBoard2() {
	a := c.App
	c.checkSession("post", "", true)
	boardID := c.POST.Int("boardid")

	if c.POST.Has("edit") || c.POST.Has("add") {
		opts := &boardOptions{}

		if c.POST.Int("new_cat") != 0 {
			opts.moveTo = "bottom"
			opts.hasMoveTo = true
			opts.targetCategory = c.POST.Int("new_cat")
			opts.hasTargetCategory = true
		} else if c.POST.Str("placement") != "" && c.POST.Int("board_order") != 0 {
			if !inStrings([]string{"before", "after", "child"}, c.POST.Str("placement")) {
				c.fatalLangError("mangled_post", false)
			}
			opts.moveTo = c.POST.Str("placement")
			opts.hasMoveTo = true
			opts.targetBoard = c.POST.Int("board_order")
			opts.hasTargetBoard = true
		}

		postsCount := c.POST.Has("count")
		opts.postsCount = &postsCount
		overrideTheme := c.POST.Has("override_theme")
		opts.overrideTheme = &overrideTheme
		boardTheme := c.POST.Int("boardtheme")
		opts.boardTheme = &boardTheme

		opts.hasAccessGroups = true
		if arr := c.POST.Arr("groups"); arr != nil {
			arr.Values(func(k string, v any) {
				s, _ := v.(string)
				opts.accessGroups = append(opts.accessGroups, atoi(s))
			})
		}

		boardName := ampEscape(c.POST.Str("board_name"))
		opts.boardName = &boardName
		boardDesc := ampEscape(c.POST.Str("desc"))
		opts.boardDescription = &boardDesc

		if a.SettingEmpty("permission_enable_by_board") {
			pm := c.POST.Int("permission_mode")
			opts.permissionMode = &pm
			opts.inheritPerms = false
			opts.hasInheritPerms = true
		}

		modStr := c.POST.Str("moderators")
		opts.moderatorString = &modStr

		if c.POST.Has("add") {
			if c.POST.Int("new_cat") == 0 {
				opts.targetCategory = c.POST.Int("cur_cat")
				opts.hasTargetCategory = true
			}
			if !opts.hasMoveTo {
				opts.moveTo = "bottom"
				opts.hasMoveTo = true
			}
			c.createBoard(opts)
		} else {
			c.modifyBoard(boardID, opts)
		}
	} else if c.POST.Has("delete") && !c.POST.Has("confirmation") && !c.POST.Has("no_children") {
		c.EditBoard()
		return
	} else if c.POST.Has("delete") {
		if c.POST.Has("delete_action") && c.POST.Int("delete_action") == 1 {
			if c.POST.Int("board_to") == 0 {
				c.fatalError(c.Txt("mboards_delete_board_error"), false)
			}
			c.deleteBoards([]int{boardID}, c.POST.Int("board_to"))
		} else {
			c.deleteBoards([]int{boardID}, 0)
		}
	}

	c.redirectExit("action=manageboards")
}

// ---- general settings ----

// MBSettingsBoard is one board option in the recycle picker.
type MBSettingsBoard struct {
	ID           int
	Name         string
	IsRecycle    bool
	CategoryName string
}

// BoardSettingsCtx backs template_modify_general_settings.
type BoardSettingsCtx struct {
	Boards               []MBSettingsBoard
	CanChangePermissions bool
}

// EditBoardSettings is EditBoardSettings().
func (c *Ctx) EditBoardSettings() {
	a := c.App
	c.PageTitle = c.Txt("41") + " - " + c.Txt("settings")
	c.SubTemplate = templateModifyGeneralSettings

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)
		recycleEnable := "0"
		if c.POST.Has("recycle_enable") {
			recycleEnable = "1"
		}
		countChild := "0"
		if c.POST.Has("countChildPosts") {
			countChild = "1"
		}
		a.UpdateSettings(map[string]string{
			"countChildPosts": countChild,
			"recycle_enable":  recycleEnable,
			"recycle_board":   itoa(c.POST.Int("recycle_board")),
		})
		c.saveInlinePermissions([]string{"manage_boards"})
	}

	page := &BoardSettingsCtx{}
	c.Page = page
	c.initInlinePermissions([]string{"manage_boards"}, []int{-1})
	page.CanChangePermissions = c.CanChangePermissions

	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name AS bName, c.ID_CAT, c.name AS cName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)`))
	if err == nil {
		for rows.Next() {
			var idBoard, idCat int
			var bName, cName string
			rows.Scan(&idBoard, &bName, &idCat, &cName)
			page.Boards = append(page.Boards, MBSettingsBoard{
				ID: idBoard, Name: bName, CategoryName: cName,
				IsRecycle: !a.SettingEmpty("recycle_board") && a.SettingInt("recycle_board") == idBoard,
			})
		}
		rows.Close()
	}
}
