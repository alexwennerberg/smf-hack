package app

// Port of the board/category tree machinery in Sources/Subs-Boards.php:
// getBoardTree, modifyBoard, createBoard, deleteBoards, modifyCategory,
// createCategory, deleteCategories, reorderBoards, fixChildren, recursiveBoards,
// isChildOf. The PHP nested-reference tree is flattened into ordered child-ID
// slices; reorderBoards drops the MySQL "ALTER TABLE ... ORDER BY" physical
// reorder (boardOrder drives order logically — documented).

import (
	"regexp"
	"strings"
)

// catNode is one category in the board tree ($cat_tree[id]).
type catNode struct {
	ID, Order, CanCollapse int
	Name                   string
	IsFirst                bool
	LastBoardOrder         int
	Children               []int // ordered top-level board IDs
}

// boardNode is one board in the board tree ($boards[id]).
type boardNode struct {
	ID, Category, Parent, Level, Order int
	Name, Description                  string
	MemberGroups                       []string
	CountPosts                         bool // empty(countPosts): posts count here
	Theme, OverrideTheme               int
	UseLocalPermissions                bool
	PermissionMode                     string // normal/no_polls/reply_only/read_only
	PrevBoard                          int
	Children                           []int // ordered direct-child board IDs
}

// boardTree holds the result of getBoardTree ($cat_tree/$boards/$boardList).
type boardTree struct {
	CatOrder  []int // category IDs in catOrder
	Cats      map[int]*catNode
	Boards    map[int]*boardNode
	BoardList map[int][]int // catID -> recursive ordered board IDs
}

// boardOptions mirrors the &$boardOptions array passed to modifyBoard/createBoard.
type boardOptions struct {
	moveTo            string // top/bottom/child/before/after
	hasMoveTo         bool
	targetCategory    int
	hasTargetCategory bool
	targetBoard       int
	hasTargetBoard    bool
	moveFirstChild    bool

	postsCount       *bool
	overrideTheme    *bool
	boardTheme       *int
	accessGroups     []int
	hasAccessGroups  bool
	boardName        *string
	boardDescription *string
	permissionMode   *int
	moderatorString  *string
	moderators       []int
	hasModerators    bool
	inheritPerms     bool
	hasInheritPerms  bool
}

// catOptions mirrors $catOptions for modifyCategory/createCategory.
type catOptions struct {
	moveAfter     int
	hasMoveAfter  bool
	catName       *string
	isCollapsible *bool
}

// getBoardTree is getBoardTree(): build the full category/board tree.
func (c *Ctx) getBoardTree() *boardTree {
	a := c.App
	t := &boardTree{Cats: map[int]*catNode{}, Boards: map[int]*boardNode{}, BoardList: map[int][]int{}}

	rows, err := a.DB.Query(a.Q(`
		SELECT
			IFNULL(b.ID_BOARD, 0) AS ID_BOARD, IFNULL(b.ID_PARENT, 0), IFNULL(b.name, ''), IFNULL(b.description, ''), IFNULL(b.childLevel, 0),
			IFNULL(b.boardOrder, 0), IFNULL(b.countPosts, 0), IFNULL(b.memberGroups, ''), IFNULL(b.ID_THEME, 0), IFNULL(b.override_theme, 0),
			IFNULL(b.permission_mode, 0), c.ID_CAT, c.name AS cName, c.catOrder, c.canCollapse
		FROM {$db_prefix}categories AS c
			LEFT JOIN {$db_prefix}boards AS b ON (b.ID_CAT = c.ID_CAT)
		ORDER BY c.catOrder, b.childLevel, b.boardOrder`))
	if err != nil {
		return t
	}
	defer rows.Close()

	lastBoardOrder := 0
	prevBoard := 0
	curLevel := 0
	enableByBoard := !a.SettingEmpty("permission_enable_by_board")

	for rows.Next() {
		var idBoard, idParent, childLevel, boardOrder, countPosts, idTheme, overrideTheme, permMode int
		var catOrder, idCat, canCollapse int
		var bName, description, memberGroups, cName string
		rows.Scan(&idBoard, &idParent, &bName, &description, &childLevel,
			&boardOrder, &countPosts, &memberGroups, &idTheme, &overrideTheme,
			&permMode, &idCat, &cName, &catOrder, &canCollapse)

		if _, ok := t.Cats[idCat]; !ok {
			t.Cats[idCat] = &catNode{
				ID: idCat, Name: cName, Order: catOrder, CanCollapse: canCollapse,
				IsFirst:        len(t.CatOrder) == 0,
				LastBoardOrder: lastBoardOrder,
			}
			t.CatOrder = append(t.CatOrder, idCat)
			prevBoard = 0
			curLevel = 0
		}

		if idBoard == 0 {
			continue
		}

		if childLevel != curLevel {
			prevBoard = 0
		}

		permModeStr := "normal"
		if !enableByBoard {
			switch permMode {
			case 0:
				permModeStr = "normal"
			case 2:
				permModeStr = "no_polls"
			case 3:
				permModeStr = "reply_only"
			default:
				permModeStr = "read_only"
			}
		}

		t.Boards[idBoard] = &boardNode{
			ID: idBoard, Category: idCat, Parent: idParent, Level: childLevel, Order: boardOrder,
			Name: bName, Description: description, MemberGroups: strings.Split(memberGroups, ","),
			CountPosts: countPosts == 0, Theme: idTheme, OverrideTheme: overrideTheme,
			UseLocalPermissions: enableByBoard && permMode == 1,
			PermissionMode:      permModeStr,
			PrevBoard:           prevBoard,
		}
		prevBoard = idBoard
		lastBoardOrder = boardOrder

		if childLevel == 0 {
			t.Cats[idCat].Children = append(t.Cats[idCat].Children, idBoard)
		} else {
			parent := t.Boards[idParent]
			if parent == nil {
				c.fatalLangError("no_valid_parent", false, bName)
				return t
			}
			// Silently fix a wrong childLevel.
			if parent.Level != childLevel-1 {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET childLevel = ? WHERE ID_BOARD = ?`), parent.Level+1, idBoard)
			}
			parent.Children = append(parent.Children, idBoard)
		}
	}

	for _, catID := range t.CatOrder {
		var list []int
		t.recursiveBoards(&list, t.Cats[catID].Children)
		t.BoardList[catID] = list
	}
	return t
}

// recursiveBoards appends board IDs depth-first in order.
func (t *boardTree) recursiveBoards(out *[]int, children []int) {
	for _, id := range children {
		*out = append(*out, id)
		if b := t.Boards[id]; b != nil {
			t.recursiveBoards(out, b.Children)
		}
	}
}

// isChildOf is isChildOf($child, $parent).
func (t *boardTree) isChildOf(child, parent int) bool {
	b := t.Boards[child]
	if b == nil || b.Parent == 0 {
		return false
	}
	if b.Parent == parent {
		return true
	}
	return t.isChildOf(b.Parent, parent)
}

// modifyBoard is modifyBoard($board_id, &$boardOptions).
func (c *Ctx) modifyBoard(boardID int, opts *boardOptions) {
	a := c.App
	t := c.getBoardTree()

	if t.Boards[boardID] == nil ||
		(opts.hasTargetBoard && t.Boards[opts.targetBoard] == nil) ||
		(opts.hasTargetCategory && t.Cats[opts.targetCategory] == nil) {
		c.fatalLangError("smf232", false)
	}

	var boardUpdates []string
	var args []any

	if opts.hasMoveTo {
		var idCat, childLevel, idParent, after int
		switch opts.moveTo {
		case "top":
			idCat = opts.targetCategory
			childLevel = 0
			idParent = 0
			after = t.Cats[idCat].LastBoardOrder
		case "bottom":
			idCat = opts.targetCategory
			childLevel = 0
			idParent = 0
			after = 0
			for _, id := range t.Cats[idCat].Children {
				if t.Boards[id].Order > after {
					after = t.Boards[id].Order
				}
			}
		case "child":
			tb := t.Boards[opts.targetBoard]
			idCat = tb.Category
			childLevel = tb.Level + 1
			idParent = opts.targetBoard
			if t.isChildOf(idParent, boardID) {
				c.fatalError("Unable to make a parent its own child", false)
			}
			after = tb.Order
			if len(tb.Children) > 0 && !opts.moveFirstChild {
				for _, id := range tb.Children {
					if t.Boards[id].Order > after {
						after = t.Boards[id].Order
					}
				}
			}
		case "before", "after":
			tb := t.Boards[opts.targetBoard]
			idCat = tb.Category
			childLevel = tb.Level
			idParent = tb.Parent
			after = tb.Order
			if opts.moveTo == "before" {
				after--
			}
		}

		// Children of this board.
		var childList []int
		t.recursiveBoards(&childList, t.Boards[boardID].Children)

		var childUpdates []string
		levelDiff := childLevel - t.Boards[boardID].Level
		if levelDiff != 0 {
			op := ""
			if levelDiff > 0 {
				op = "+ "
			}
			childUpdates = append(childUpdates, "childLevel = childLevel "+op+itoa(levelDiff))
		}
		if idCat != t.Boards[boardID].Category {
			childUpdates = append(childUpdates, "ID_CAT = "+itoa(idCat))
		}
		if len(childList) > 0 && len(childUpdates) > 0 {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ` + strings.Join(childUpdates, ", ") + ` WHERE ID_BOARD IN (` + joinInts(childList) + `)`))
		}

		// Make room for this spot.
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET boardOrder = boardOrder + ? WHERE boardOrder > ? AND ID_BOARD != ?`),
			1+len(childList), after, boardID)

		boardUpdates = append(boardUpdates, "ID_CAT = ?", "ID_PARENT = ?", "childLevel = ?", "boardOrder = ?")
		args = append(args, idCat, idParent, childLevel, after+1)
	}

	if opts.postsCount != nil {
		v := "1"
		if *opts.postsCount {
			v = "0"
		}
		boardUpdates = append(boardUpdates, "countPosts = "+v)
	}
	if opts.boardTheme != nil {
		boardUpdates = append(boardUpdates, "ID_THEME = ?")
		args = append(args, *opts.boardTheme)
	}
	if opts.overrideTheme != nil {
		v := "0"
		if *opts.overrideTheme {
			v = "1"
		}
		boardUpdates = append(boardUpdates, "override_theme = "+v)
	}
	if opts.hasAccessGroups {
		groups := make([]string, len(opts.accessGroups))
		for i, g := range opts.accessGroups {
			groups[i] = itoa(g)
		}
		boardUpdates = append(boardUpdates, "memberGroups = ?")
		args = append(args, strings.Join(groups, ","))
	}
	if opts.boardName != nil {
		boardUpdates = append(boardUpdates, "name = ?")
		args = append(args, *opts.boardName)
	}
	if opts.boardDescription != nil {
		boardUpdates = append(boardUpdates, "description = ?")
		args = append(args, *opts.boardDescription)
	}
	if opts.permissionMode != nil && a.SettingEmpty("permission_enable_by_board") {
		boardUpdates = append(boardUpdates, "permission_mode = ?")
		args = append(args, *opts.permissionMode)
	}

	if len(boardUpdates) > 0 {
		args = append(args, boardID)
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET `+strings.Join(boardUpdates, ", ")+` WHERE ID_BOARD = ?`), args...)
	}

	// Moderators.
	if opts.hasModerators || opts.moderatorString != nil {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}moderators WHERE ID_BOARD = ?`), boardID)

		if opts.moderatorString != nil && strings.TrimSpace(*opts.moderatorString) != "" {
			names := parseModeratorString(*opts.moderatorString)
			if len(names) > 0 {
				quoted := make([]string, len(names))
				for i, n := range names {
					quoted[i] = "'" + strings.ReplaceAll(n, "'", "''") + "'"
				}
				in := strings.Join(quoted, ", ")
				mrows, err := a.DB.Query(a.Q(`
					SELECT ID_MEMBER FROM {$db_prefix}members
					WHERE memberName IN (` + in + `) OR realName IN (` + in + `)`))
				if err == nil {
					for mrows.Next() {
						var id int
						mrows.Scan(&id)
						opts.moderators = append(opts.moderators, id)
					}
					mrows.Close()
				}
			}
		}

		for _, mod := range opts.moderators {
			a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}moderators (ID_BOARD, ID_MEMBER) VALUES (?, ?)`), boardID, mod)
		}
	}

	if opts.hasMoveTo {
		c.reorderBoards()
	}
}

// reModeratorQuoted matches "quoted" names in a moderator string.
var reModeratorQuoted = regexp.MustCompile(`"([^"]+)"`)

// parseModeratorString splits a "quoted", comma-separated list of moderator
// names (htmlspecialchars'd, then &quot; restored, as PHP does).
func parseModeratorString(s string) []string {
	s = strings.ReplaceAll(Htmlspecialchars(s), "&quot;", `"`)
	var names []string
	for _, m := range reModeratorQuoted.FindAllStringSubmatch(s, -1) {
		names = append(names, m[1])
	}
	rest := reModeratorQuoted.ReplaceAllString(s, "")
	for _, n := range strings.Split(rest, ",") {
		names = append(names, n)
	}
	var out []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

// createBoard is createBoard($boardOptions): returns the new board ID.
func (c *Ctx) createBoard(opts *boardOptions) int {
	a := c.App

	if opts.boardName == nil || strings.TrimSpace(*opts.boardName) == "" || !opts.hasMoveTo || !opts.hasTargetCategory {
		return 0
	}

	res, err := a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}boards (ID_CAT, name, description, boardOrder, memberGroups)
		VALUES (?, SUBSTR(?, 1, 255), '', 0, '-1,0')`), opts.targetCategory, *opts.boardName)
	if err != nil {
		return 0
	}
	id64, _ := res.LastInsertId()
	boardID := int(id64)
	if boardID == 0 {
		return 0
	}

	c.modifyBoard(boardID, opts)

	// Inherit parent permissions (permission_enable_by_board off in this port).
	inherit := true
	if opts.hasInheritPerms {
		inherit = opts.inheritPerms
	}
	if inherit {
		t := c.getBoardTree()
		b := t.Boards[boardID]
		if a.SettingEmpty("permission_enable_by_board") && b != nil && b.Parent != 0 && t.Boards[b.Parent] != nil && !t.Boards[b.Parent].UseLocalPermissions {
			var mode int
			a.DB.QueryRow(a.Q(`SELECT permission_mode FROM {$db_prefix}boards WHERE ID_BOARD = ? LIMIT 1`), b.Parent).Scan(&mode)
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET permission_mode = ? WHERE ID_BOARD = ?`), mode, boardID)
		}
	}

	return boardID
}

// deleteBoards is deleteBoards($boards_to_remove, $moveChildrenTo). Pass
// moveChildrenTo = -1 to include children (PHP null).
func (c *Ctx) deleteBoards(boardsToRemove []int, moveChildrenTo int) {
	a := c.App
	if len(boardsToRemove) == 0 {
		return
	}
	t := c.getBoardTree()

	if moveChildrenTo == -1 {
		var childBoards []int
		for _, b := range boardsToRemove {
			if t.Boards[b] != nil {
				t.recursiveBoards(&childBoards, t.Boards[b].Children)
			}
		}
		if len(childBoards) > 0 {
			boardsToRemove = uniqueInts(append(boardsToRemove, childBoards...))
		}
	} else {
		for _, idBoard := range boardsToRemove {
			if moveChildrenTo == 0 {
				c.fixChildren(idBoard, 0, 0)
			} else {
				c.fixChildren(idBoard, t.Boards[moveChildrenTo].Level+1, moveChildrenTo)
			}
		}
	}

	in := joinInts(boardsToRemove)

	// Remove all topics in these boards first.
	var topics []int
	trows, err := a.DB.Query(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}topics WHERE ID_BOARD IN (` + in + `)`))
	if err == nil {
		for trows.Next() {
			var id int
			trows.Scan(&id)
			topics = append(topics, id)
		}
		trows.Close()
	}
	c.removeTopics(topics, false, false)

	for _, tbl := range []string{"log_mark_read", "log_boards", "log_notify", "moderators", "calendar", "board_permissions", "message_icons"} {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}` + tbl + ` WHERE ID_BOARD IN (` + in + `)`))
	}
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}boards WHERE ID_BOARD IN (` + in + `)`))

	a.updateStatsMessage()
	a.updateStatsTopic()
	a.updateStatsCalendar()

	if !a.SettingEmpty("recycle_board") && inInts(boardsToRemove, a.SettingInt("recycle_board")) {
		a.UpdateSettings(map[string]string{"recycle_board": "0", "recycle_enable": "0"})
	}

	c.reorderBoards()
}

// modifyCategory is modifyCategory($category_id, $catOptions).
func (c *Ctx) modifyCategory(categoryID int, opts *catOptions) {
	a := c.App
	var catUpdates []string
	var args []any

	if opts.hasMoveAfter {
		var cats []int
		catOrder := map[int]int{}
		if opts.moveAfter == 0 {
			cats = append(cats, categoryID)
		}
		rows, err := a.DB.Query(a.Q(`SELECT ID_CAT, catOrder FROM {$db_prefix}categories ORDER BY catOrder`))
		if err == nil {
			for rows.Next() {
				var id, order int
				rows.Scan(&id, &order)
				if id != categoryID {
					cats = append(cats, id)
				}
				if id == opts.moveAfter {
					cats = append(cats, categoryID)
				}
				catOrder[id] = order
			}
			rows.Close()
		}
		for index, cat := range cats {
			if index != catOrder[cat] {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}categories SET catOrder = ? WHERE ID_CAT = ?`), index, cat)
			}
		}
		c.reorderBoards()
	}

	if opts.catName != nil {
		catUpdates = append(catUpdates, "name = ?")
		args = append(args, *opts.catName)
	}
	if opts.isCollapsible != nil {
		v := "0"
		if *opts.isCollapsible {
			v = "1"
		}
		catUpdates = append(catUpdates, "canCollapse = "+v)
	}

	if len(catUpdates) > 0 {
		args = append(args, categoryID)
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}categories SET `+strings.Join(catUpdates, ", ")+` WHERE ID_CAT = ?`), args...)
	}
}

// createCategory is createCategory($catOptions): returns the new category ID.
func (c *Ctx) createCategory(opts *catOptions) int {
	a := c.App
	if opts.catName == nil || strings.TrimSpace(*opts.catName) == "" {
		return 0
	}
	res, err := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}categories (name) VALUES (SUBSTR(?, 1, 48))`), *opts.catName)
	if err != nil {
		return 0
	}
	id64, _ := res.LastInsertId()
	categoryID := int(id64)
	if !opts.hasMoveAfter {
		opts.moveAfter = 0
		opts.hasMoveAfter = true
	}
	if opts.isCollapsible == nil {
		tru := true
		opts.isCollapsible = &tru
	}
	c.modifyCategory(categoryID, opts)
	return categoryID
}

// deleteCategories is deleteCategories($categories, $moveBoardsTo). Pass
// moveBoardsTo = -1 to delete the boards too (PHP null).
func (c *Ctx) deleteCategories(categories []int, moveBoardsTo int) {
	a := c.App
	in := joinInts(categories)

	if moveBoardsTo == -1 {
		var inside []int
		rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD FROM {$db_prefix}boards WHERE ID_CAT IN (` + in + `)`))
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				inside = append(inside, id)
			}
			rows.Close()
		}
		if len(inside) > 0 {
			c.deleteBoards(inside, -1)
		}
	} else if inInts(categories, moveBoardsTo) {
		return
	} else {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_CAT = ? WHERE ID_CAT IN (`+in+`)`), moveBoardsTo)
	}

	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}collapsed_categories WHERE ID_CAT IN (` + in + `)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}categories WHERE ID_CAT IN (` + in + `)`))

	c.reorderBoards()
}

// reorderBoards is reorderBoards(): renumber boardOrder per category in tree
// order. (The MySQL physical "ALTER TABLE ... ORDER BY" is dropped.)
func (c *Ctx) reorderBoards() {
	a := c.App
	t := c.getBoardTree()
	boardOrder := 0
	for _, catID := range t.CatOrder {
		for _, boardID := range t.BoardList[catID] {
			boardOrder++
			if t.Boards[boardID].Order != boardOrder {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET boardOrder = ? WHERE ID_BOARD = ?`), boardOrder, boardID)
			}
		}
	}
}

// fixChildren is fixChildren($parent, $newLevel, $newParent).
func (c *Ctx) fixChildren(parent, newLevel, newParent int) {
	a := c.App
	var children []int
	rows, err := a.DB.Query(a.Q(`SELECT ID_BOARD FROM {$db_prefix}boards WHERE ID_PARENT = ?`), parent)
	if err == nil {
		for rows.Next() {
			var id int
			rows.Scan(&id)
			children = append(children, id)
		}
		rows.Close()
	}
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_PARENT = ?, childLevel = ? WHERE ID_PARENT = ?`), newParent, newLevel, parent)
	for _, child := range children {
		c.fixChildren(child, newLevel+1, child)
	}
}
