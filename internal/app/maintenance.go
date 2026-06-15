package app

// Port of the maintenance tools in Sources/Admin.php: Maintenance() (the
// maintenance home + log clearing) and AdminBoardRecount(). The PHP time-based
// continuation (chunking work to dodge execution timeouts) is collapsed into a
// single pass — local SQLite is fast and Go has no max_execution_time.
// CleanupPermissions (file chmod via the dropped package-FTP system),
// OptimizeTables, ConvertUtf8/Entities and dumpdb are MySQL/PHP-specific and
// dropped; the maintain template still links to them byte-for-byte.

import "sort"

func init() {
	registerAction("maintain", (*Ctx).Maintenance)
	registerAction("boardrecount", (*Ctx).AdminBoardRecount)
}

// MaintainBoard is one board in the prune picker.
type MaintainBoard struct {
	ID         int
	Name       string
	ChildLevel int
}

// MaintainCategory groups boards under a category.
type MaintainCategory struct {
	Name   string
	Boards []MaintainBoard
}

// NotDoneCtx backs template_not_done (the progress/continuation page used by
// chunked operations).
type NotDoneCtx struct {
	ContinueGetData   string
	ContinuePostData  string
	ContinuePercent   int
	ContinueCountdown int
}

// MaintainCtx backs template_maintain.
type MaintainCtx struct {
	Finished      bool
	Categories    []MaintainCategory
	ConvertUTF8   bool
	ConvertEntity bool
}

// Maintenance is Maintenance(): the maintenance center.
func (c *Ctx) Maintenance() {
	a := c.App

	c.isAllowedTo("admin_forum")
	c.adminIndex("maintain_forum")

	page := &MaintainCtx{}
	c.Page = page

	switch c.GET.Str("sa") {
	case "logs":
		// No one's online now.... MUHAHAHAHA :P.
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_online`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_banned`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_errors`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_floodcontrol`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_karma`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_topics`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_messages`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_results`))
		a.UpdateSettings(map[string]string{"search_pointer": "0"})
		page.Finished = true
	case "destroy":
		c.O(`<html><head><title>`, c.App.Config.MbName, ` deleted!</title></head>
			<body style="background-color: orange; font-family: arial, sans-serif; text-align: center;">
			<div style="margin-top: 8%; font-size: 400%; color: black;">Oh my, you killed `, c.App.Config.MbName, `!</div>
			<div style="margin-top: 7%; font-size: 500%; color: red;"><b>You lazy bum!</b></div>
			</body></html>`)
		f := false
		c.obExit(&f, nil, false)
		return
	default:
		page.Finished = c.GET.Has("done")
	}

	// Grab some boards maintenance can be done on (ordered by category).
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name, b.childLevel, c.name AS catName, c.ID_CAT
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE ` + c.User.QuerySeeBoard))
	catIndex := map[int]int{}
	if err == nil {
		for rows.Next() {
			var idBoard, childLevel, idCat int
			var name, catName string
			rows.Scan(&idBoard, &name, &childLevel, &catName, &idCat)
			idx, ok := catIndex[idCat]
			if !ok {
				idx = len(page.Categories)
				catIndex[idCat] = idx
				page.Categories = append(page.Categories, MaintainCategory{Name: catName})
			}
			page.Categories[idx].Boards = append(page.Categories[idx].Boards, MaintainBoard{ID: idBoard, Name: name, ChildLevel: childLevel})
		}
		rows.Close()
	}

	// ISO-8859-1 port: never UTF-8 (these links stay hidden).
	page.ConvertUTF8 = false
	page.ConvertEntity = false

	c.SubTemplate = templateMaintain
	c.PageTitle = c.Txt("maintain_title")
}

// AdminBoardRecount is AdminBoardRecount(): recompute topic/board/member
// totals and the per-board latest message. Done in a single pass.
func (c *Ctx) AdminBoardRecount() {
	a := c.App

	c.isAllowedTo("admin_forum")
	c.adminIndex("maintain_forum")

	// Step 1: fix each topic's reply count.
	type tr struct{ topic, real int }
	var fixes []tr
	rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC, COUNT(*) - 1 AS realNumReplies
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m
		WHERE m.ID_TOPIC = t.ID_TOPIC
		GROUP BY t.ID_TOPIC
		HAVING realNumReplies != t.numReplies`))
	if err == nil {
		for rows.Next() {
			var f tr
			rows.Scan(&f.topic, &f.real)
			fixes = append(fixes, f)
		}
		rows.Close()
	}
	for _, f := range fixes {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET numReplies = ? WHERE ID_TOPIC = ?`), f.real, f.topic)
	}

	// Step 2: recompute each board's post and topic counts.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numPosts = 0, numTopics = 0`))
	type bc struct{ board, posts, topics int }
	var bcs []bc
	brows, err := a.DB.Query(a.Q(`
		SELECT t.ID_BOARD, COUNT(*) AS realNumPosts, COUNT(DISTINCT t.ID_TOPIC) AS realNumTopics
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m
		WHERE m.ID_TOPIC = t.ID_TOPIC
		GROUP BY t.ID_BOARD`))
	if err == nil {
		for brows.Next() {
			var b bc
			brows.Scan(&b.board, &b.posts, &b.topics)
			bcs = append(bcs, b)
		}
		brows.Close()
	}
	for _, b := range bcs {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numPosts = numPosts + ?, numTopics = numTopics + ? WHERE ID_BOARD = ?`), b.posts, b.topics, b.board)
	}

	// Step 3: fix members' PM counts.
	type mc struct {
		member int
		col    string
		real   int
	}
	var mcs []mc
	mrows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER, COUNT(pmr.ID_PM) AS realNum, mem.instantMessages
		FROM {$db_prefix}members AS mem
			LEFT JOIN {$db_prefix}pm_recipients AS pmr ON (mem.ID_MEMBER = pmr.ID_MEMBER AND pmr.deleted = 0)
		GROUP BY mem.ID_MEMBER
		HAVING realNum != mem.instantMessages`))
	if err == nil {
		for mrows.Next() {
			var id, real, cur int
			mrows.Scan(&id, &real, &cur)
			mcs = append(mcs, mc{id, "instantMessages", real})
		}
		mrows.Close()
	}
	urows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER, COUNT(pmr.ID_PM) AS realNum, mem.unreadMessages
		FROM {$db_prefix}members AS mem
			LEFT JOIN {$db_prefix}pm_recipients AS pmr ON (mem.ID_MEMBER = pmr.ID_MEMBER AND pmr.deleted = 0 AND pmr.is_read = 0)
		GROUP BY mem.ID_MEMBER
		HAVING realNum != mem.unreadMessages`))
	if err == nil {
		for urows.Next() {
			var id, real, cur int
			urows.Scan(&id, &real, &cur)
			mcs = append(mcs, mc{id, "unreadMessages", real})
		}
		urows.Close()
	}
	for _, m := range mcs {
		a.updateMemberDataMap(m.member, map[string]any{m.col: m.real})
	}

	// Step 4: fix messages pointing to the wrong board.
	type mb struct{ board, msg int }
	boardMsgs := map[int][]int{}
	xrows, err := a.DB.Query(a.Q(`
		SELECT t.ID_BOARD, m.ID_MSG
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE t.ID_TOPIC = m.ID_TOPIC
			AND m.ID_BOARD != t.ID_BOARD`))
	if err == nil {
		for xrows.Next() {
			var x mb
			xrows.Scan(&x.board, &x.msg)
			boardMsgs[x.board] = append(boardMsgs[x.board], x.msg)
		}
		xrows.Close()
	}
	for board, msgs := range boardMsgs {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_BOARD = ? WHERE ID_MSG IN (`+joinInts(msgs)+`)`), board)
	}

	// Step 5: recompute each board's latest message (children bubble up).
	type boardRow struct {
		board, parent, lastMsg, localLast, childLevel int
	}
	byLevel := map[int][]boardRow{}
	lrows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.ID_PARENT, b.ID_LAST_MSG, MAX(m.ID_MSG) AS localLastMsg, b.childLevel
		FROM {$db_prefix}boards AS b, {$db_prefix}messages AS m
		WHERE b.ID_BOARD = m.ID_BOARD
		GROUP BY b.ID_BOARD`))
	if err == nil {
		for lrows.Next() {
			var r boardRow
			lrows.Scan(&r.board, &r.parent, &r.lastMsg, &r.localLast, &r.childLevel)
			byLevel[r.childLevel] = append(byLevel[r.childLevel], r)
		}
		lrows.Close()
	}
	// Process deepest child levels first (krsort).
	levels := make([]int, 0, len(byLevel))
	for lvl := range byLevel {
		levels = append(levels, lvl)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(levels)))
	lastMsg := map[int]int{}
	for _, lvl := range levels {
		for _, r := range byLevel[lvl] {
			curLast := r.localLast
			if v, ok := lastMsg[r.board]; ok && v > curLast {
				curLast = v
			}
			if curLast != r.lastMsg {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_LAST_MSG = ? WHERE ID_BOARD = ?`), curLast, r.board)
			}
			if v, ok := lastMsg[r.parent]; !ok || r.localLast > v {
				lastMsg[r.parent] = r.localLast
			}
		}
	}

	// Update the basic statistics.
	c.updateStatsMemberRecount()
	a.updateStatsMessage()
	a.updateStatsTopic()

	c.redirectExit("action=maintain;done")
}
