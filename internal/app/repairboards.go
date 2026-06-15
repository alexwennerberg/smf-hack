package app

// Port of Sources/RepairBoards.php: the forum-integrity checker
// (?action=repairboards). Per the plan this is the pure-SQL integrity-checks
// subset — the checks/fixes that are expressible directly in SQL: orphaned
// messages/topics/boards, wrong topic first/last/reply stats, missing posters,
// dangling polls/parents, and orphaned log rows. The PHP step/substep pause
// loop is collapsed to a single pass (Go has no execution timeout). The
// detection pass lists problems; ?fixErrors applies every fix and recounts
// stats. Salvage areas (category/board) are created on demand, as in PHP.

import "strings"

func init() {
	registerAction("repairboards", (*Ctx).RepairBoards)
}

func (c *Ctx) RepairBoards() {
	c.isAllowedTo("admin_forum")
	c.adminIndex("maintain_forum")
	c.PageTitle = c.Txt("610")

	scripturl := c.App.ScriptURL

	if !c.GET.Has("fixErrors") {
		errors := c.findForumErrors()
		var raw strings.Builder
		raw.WriteString(`
			<table width="100%" border="0" cellspacing="0" cellpadding="4" class="tborder">
				<tr class="titlebg">
					<td>` + c.Txt("smf73") + `</td>
				</tr><tr>
					<td class="windowbg">`)
		if len(errors) > 0 {
			raw.WriteString(`
						` + c.Txt("smf74") + `:<br />
						` + strings.Join(errors, `
						<br />`) + `<br />
						<br />
						` + c.Txt("smf85") + `<br />
						<b><a href="` + scripturl + `?action=repairboards;fixErrors;sesc=` + c.Sc + `">` + c.Txt("163") + `</a> - <a href="` + scripturl + `?action=maintain">` + c.Txt("164") + `</a></b>`)
		} else {
			raw.WriteString(`
						` + c.Txt("maintain_no_errors") + `<br />
						<br />
						<a href="` + scripturl + `?action=maintain">` + c.Txt("maintain_return") + `</a>`)
		}
		raw.WriteString(`
					</td>
				</tr>
			</table>`)
		c.setRawData(raw.String())
		return
	}

	c.checkSession("get", "", true)
	c.applyForumFixes()
	c.setRawData(`
			<table width="100%" border="0" cellspacing="0" cellpadding="4" class="tborder">
				<tr class="titlebg">
					<td>` + c.Txt("smf86") + `</td>
				</tr><tr>
					<td class="windowbg">
						` + c.Txt("smf92") + `<br />
						<br />
						<a href="` + scripturl + `?action=maintain">` + c.Txt("maintain_return") + `</a>
					</td>
				</tr>
			</table>`)
}

func (c *Ctx) setRawData(html string) {
	c.SubTemplate = func(c *Ctx) { c.O(html) }
}

// findForumErrors runs the detection queries and returns the human-readable
// error lines (the PHP $context['repair_errors']).
func (c *Ctx) findForumErrors() []string {
	a := c.App
	var errors []string

	// zero_ids
	var zeroTopics, zeroMessages int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}topics WHERE ID_TOPIC = 0`)).Scan(&zeroTopics)
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}messages WHERE ID_MSG = 0`)).Scan(&zeroMessages)
	if zeroTopics != 0 || zeroMessages != 0 {
		errors = append(errors, c.Txt("repair_zero_ids"))
	}

	// missing_topics: messages whose topic is gone.
	if rows, err := a.DB.Query(a.Q(`
		SELECT m.ID_TOPIC, m.ID_MSG
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}topics AS t ON (t.ID_TOPIC = m.ID_TOPIC)
		WHERE t.ID_TOPIC IS NULL
		ORDER BY m.ID_TOPIC, m.ID_MSG`)); err == nil {
		for rows.Next() {
			var tid, mid int
			rows.Scan(&tid, &mid)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_topics"), mid, tid))
		}
		rows.Close()
	}

	// missing_messages: topics with no messages.
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_TOPIC = t.ID_TOPIC)
		GROUP BY t.ID_TOPIC
		HAVING COUNT(m.ID_MSG) = 0`)); err == nil {
		for rows.Next() {
			var tid int
			rows.Scan(&tid)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_messages"), tid))
		}
		rows.Close()
	}

	// stats_topics: wrong first/last/numReplies.
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC, t.ID_FIRST_MSG, t.ID_LAST_MSG, t.numReplies,
			MIN(m.ID_MSG) AS myFirst, MAX(m.ID_MSG) AS myLast, COUNT(m.ID_MSG) - 1 AS myReplies
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_TOPIC = t.ID_TOPIC)
		GROUP BY t.ID_TOPIC
		HAVING t.ID_FIRST_MSG != myFirst OR t.ID_LAST_MSG != myLast OR t.numReplies != myReplies
		ORDER BY t.ID_TOPIC`)); err == nil {
		for rows.Next() {
			var tid, first, last, replies, myFirst, myLast, myReplies int
			rows.Scan(&tid, &first, &last, &replies, &myFirst, &myLast, &myReplies)
			if first != myFirst {
				errors = append(errors, phpSprintf(c.Txt("repair_stats_topics_1"), tid, first))
			}
			if last != myLast {
				errors = append(errors, phpSprintf(c.Txt("repair_stats_topics_2"), tid, last))
			}
			if replies != myReplies {
				errors = append(errors, phpSprintf(c.Txt("repair_stats_topics_3"), tid, replies))
			}
		}
		rows.Close()
	}

	// missing_boards: topics on a board that's gone.
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC, t.ID_BOARD
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = t.ID_BOARD)
		WHERE b.ID_BOARD IS NULL
		ORDER BY t.ID_BOARD, t.ID_TOPIC`)); err == nil {
		for rows.Next() {
			var tid, bid int
			rows.Scan(&tid, &bid)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_boards"), tid, bid))
		}
		rows.Close()
	}

	// missing_categories: boards in a category that's gone.
	if rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.ID_CAT
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE c.ID_CAT IS NULL
		ORDER BY b.ID_CAT, b.ID_BOARD`)); err == nil {
		for rows.Next() {
			var bid, cid int
			rows.Scan(&bid, &cid)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_categories"), bid, cid))
		}
		rows.Close()
	}

	// missing_posters: messages by a non-existent member.
	if rows, err := a.DB.Query(a.Q(`
		SELECT m.ID_MSG, m.ID_MEMBER
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE mem.ID_MEMBER IS NULL AND m.ID_MEMBER != 0
		ORDER BY m.ID_MSG`)); err == nil {
		for rows.Next() {
			var mid, member int
			rows.Scan(&mid, &member)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_posters"), mid, member))
		}
		rows.Close()
	}

	// missing_parents: boards whose parent is gone.
	if rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.ID_PARENT
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}boards AS p ON (p.ID_BOARD = b.ID_PARENT)
		WHERE b.ID_PARENT != 0 AND (p.ID_BOARD IS NULL OR p.ID_BOARD = b.ID_BOARD)
		ORDER BY b.ID_PARENT, b.ID_BOARD`)); err == nil {
		for rows.Next() {
			var bid, pid int
			rows.Scan(&bid, &pid)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_parents"), bid, pid))
		}
		rows.Close()
	}

	// missing_polls: topics referencing a poll that's gone.
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC, t.ID_POLL
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}polls AS p ON (p.ID_POLL = t.ID_POLL)
		WHERE t.ID_POLL != 0 AND p.ID_POLL IS NULL
		ORDER BY t.ID_POLL, t.ID_TOPIC`)); err == nil {
		for rows.Next() {
			var tid, pid int
			rows.Scan(&tid, &pid)
			errors = append(errors, phpSprintf(c.Txt("repair_missing_polls"), tid, pid))
		}
		rows.Close()
	}

	// missing_log_topics / missing_log_boards: orphaned log rows.
	if n := c.countOrphans(`{$db_prefix}log_topics AS lt LEFT JOIN {$db_prefix}topics AS t ON (t.ID_TOPIC = lt.ID_TOPIC) WHERE t.ID_TOPIC IS NULL`); n > 0 {
		errors = append(errors, c.Txt("repair_missing_log_topics"))
	}
	if n := c.countOrphans(`{$db_prefix}log_boards AS lb LEFT JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = lb.ID_BOARD) WHERE b.ID_BOARD IS NULL`); n > 0 {
		errors = append(errors, c.Txt("repair_missing_log_boards"))
	}

	return errors
}

func (c *Ctx) countOrphans(fromWhere string) int {
	var n int
	c.App.DB.QueryRow(c.App.Q(`SELECT COUNT(*) FROM ` + fromWhere)).Scan(&n)
	return n
}

// salvage lazily creates the Salvage Category / Board and returns their IDs.
type salvage struct {
	c       *Ctx
	created bool
	catID   int
	boardID int
}

func (s *salvage) ensure() {
	if s.created {
		return
	}
	s.created = true
	a := s.c.App
	catName := s.c.Txt("salvaged_category_name")
	a.DB.QueryRow(a.Q(`SELECT ID_CAT FROM {$db_prefix}categories WHERE name = ? LIMIT 1`), catName).Scan(&s.catID)
	if s.catID == 0 {
		res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}categories (name, catOrder) VALUES (substr(?, 1, 255), -1)`), catName)
		id, _ := res.LastInsertId()
		s.catID = int(id)
	}
	boardName := s.c.Txt("salvaged_board_name")
	a.DB.QueryRow(a.Q(`SELECT ID_BOARD FROM {$db_prefix}boards WHERE ID_CAT = ? AND name = ? LIMIT 1`), s.catID, boardName).Scan(&s.boardID)
	if s.boardID == 0 {
		res, _ := a.DB.Exec(a.Q(`INSERT INTO {$db_prefix}boards (name, description, ID_CAT, memberGroups, boardOrder) VALUES (substr(?, 1, 255), substr(?, 1, 255), ?, '1', -1)`),
			boardName, s.c.Txt("salvaged_board_description"), s.catID)
		id, _ := res.LastInsertId()
		s.boardID = int(id)
	}
}

// applyForumFixes runs every fix (the to_fix-empty branch of PHP, i.e. fix all).
func (c *Ctx) applyForumFixes() {
	a := c.App
	sal := &salvage{c: c}

	// zero_ids (MySQL-ism; harmless no-op when the IDs are PK aliases).
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_TOPIC = NULL WHERE ID_TOPIC = 0`))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_MSG = NULL WHERE ID_MSG = 0`))

	// missing_messages: delete empty topics and their log rows.
	var emptyTopics []int
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_TOPIC = t.ID_TOPIC)
		GROUP BY t.ID_TOPIC HAVING COUNT(m.ID_MSG) = 0`)); err == nil {
		for rows.Next() {
			var tid int
			rows.Scan(&tid)
			emptyTopics = append(emptyTopics, tid)
		}
		rows.Close()
	}
	if len(emptyTopics) > 0 {
		in := joinInts(emptyTopics)
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}topics WHERE ID_TOPIC IN (` + in + `)`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_TOPIC IN (` + in + `)`))
	}

	// missing_topics: salvage orphaned messages into freshly-created topics.
	type orphanGroup struct {
		idBoard, idTopic, first, last, replies int
	}
	var groups []orphanGroup
	if rows, err := a.DB.Query(a.Q(`
		SELECT m.ID_BOARD, m.ID_TOPIC, MIN(m.ID_MSG) AS myFirst, MAX(m.ID_MSG) AS myLast, COUNT(*) - 1 AS myReplies
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}topics AS t ON (t.ID_TOPIC = m.ID_TOPIC)
		WHERE t.ID_TOPIC IS NULL
		GROUP BY m.ID_TOPIC`)); err == nil {
		for rows.Next() {
			var g orphanGroup
			rows.Scan(&g.idBoard, &g.idTopic, &g.first, &g.last, &g.replies)
			groups = append(groups, g)
		}
		rows.Close()
	}
	for _, g := range groups {
		if g.idBoard == 0 {
			sal.ensure()
			g.idBoard = sal.boardID
		}
		memberStarted := c.getMsgMemberID(g.first)
		memberUpdated := c.getMsgMemberID(g.last)
		res, _ := a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}topics (ID_BOARD, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_FIRST_MSG, ID_LAST_MSG, numReplies)
			VALUES (?, ?, ?, ?, ?, ?)`), g.idBoard, memberStarted, memberUpdated, g.first, g.last, g.replies)
		newID, _ := res.LastInsertId()
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_TOPIC = ?, ID_BOARD = ? WHERE ID_TOPIC = ?`), newID, g.idBoard, g.idTopic)
	}

	// stats_topics: fix first/last msg, starter/updater and numReplies.
	type statFix struct {
		tid, myFirst, myLast, myReplies int
	}
	var statFixes []statFix
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC, MIN(m.ID_MSG) AS myFirst, MAX(m.ID_MSG) AS myLast, COUNT(m.ID_MSG) - 1 AS myReplies
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_TOPIC = t.ID_TOPIC)
		GROUP BY t.ID_TOPIC
		HAVING t.ID_FIRST_MSG != myFirst OR t.ID_LAST_MSG != myLast OR t.numReplies != myReplies`)); err == nil {
		for rows.Next() {
			var s statFix
			rows.Scan(&s.tid, &s.myFirst, &s.myLast, &s.myReplies)
			statFixes = append(statFixes, s)
		}
		rows.Close()
	}
	for _, s := range statFixes {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET ID_FIRST_MSG = ?, ID_MEMBER_STARTED = ?, ID_LAST_MSG = ?, ID_MEMBER_UPDATED = ?, numReplies = ?
			WHERE ID_TOPIC = ?`), s.myFirst, c.getMsgMemberID(s.myFirst), s.myLast, c.getMsgMemberID(s.myLast), s.myReplies, s.tid)
	}

	// missing_boards: salvage topics whose board is gone.
	type boardFix struct {
		idBoard, numTopics, numPosts int
	}
	var boardFixes []boardFix
	if rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_BOARD, COUNT(*) AS myNumTopics, COUNT(m.ID_MSG) AS myNumPosts
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}boards AS b ON (b.ID_BOARD = t.ID_BOARD)
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_TOPIC = t.ID_TOPIC)
		WHERE b.ID_BOARD IS NULL
		GROUP BY t.ID_BOARD`)); err == nil {
		for rows.Next() {
			var bf boardFix
			rows.Scan(&bf.idBoard, &bf.numTopics, &bf.numPosts)
			boardFixes = append(boardFixes, bf)
		}
		rows.Close()
	}
	if len(boardFixes) > 0 {
		sal.ensure()
		for _, bf := range boardFixes {
			res, _ := a.DB.Exec(a.Q(`
				INSERT INTO {$db_prefix}boards (ID_CAT, name, description, numTopics, numPosts, memberGroups)
				VALUES (?, 'Salvaged board', '', ?, ?, '1')`), sal.catID, bf.numTopics, bf.numPosts)
			newID, _ := res.LastInsertId()
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_BOARD = ? WHERE ID_BOARD = ?`), newID, bf.idBoard)
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_BOARD = ? WHERE ID_BOARD = ?`), newID, bf.idBoard)
		}
	}

	// missing_categories: move boards into the salvage category.
	var orphanCats []int
	if rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_CAT FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE c.ID_CAT IS NULL GROUP BY b.ID_CAT`)); err == nil {
		for rows.Next() {
			var cid int
			rows.Scan(&cid)
			orphanCats = append(orphanCats, cid)
		}
		rows.Close()
	}
	if len(orphanCats) > 0 {
		sal.ensure()
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_CAT = ? WHERE ID_CAT IN (`+joinInts(orphanCats)+`)`), sal.catID)
	}

	// missing_posters: re-assign to guest.
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}messages SET ID_MEMBER = 0
		WHERE ID_MEMBER != 0 AND ID_MEMBER NOT IN (SELECT ID_MEMBER FROM {$db_prefix}members)`))

	// missing_parents: re-home boards under the salvage board.
	var orphanParents []int
	if rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_PARENT FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}boards AS p ON (p.ID_BOARD = b.ID_PARENT)
		WHERE b.ID_PARENT != 0 AND (p.ID_BOARD IS NULL OR p.ID_BOARD = b.ID_BOARD)
		GROUP BY b.ID_PARENT`)); err == nil {
		for rows.Next() {
			var pid int
			rows.Scan(&pid)
			orphanParents = append(orphanParents, pid)
		}
		rows.Close()
	}
	if len(orphanParents) > 0 {
		sal.ensure()
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET ID_PARENT = ?, ID_CAT = ?, childLevel = 1 WHERE ID_PARENT IN (`+joinInts(orphanParents)+`)`), sal.boardID, sal.catID)
	}

	// missing_polls: clear dangling poll references.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_POLL = 0 WHERE ID_POLL != 0 AND ID_POLL NOT IN (SELECT ID_POLL FROM {$db_prefix}polls)`))

	// Orphaned log rows.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_TOPIC NOT IN (SELECT ID_TOPIC FROM {$db_prefix}topics)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_MEMBER NOT IN (SELECT ID_MEMBER FROM {$db_prefix}members)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_boards WHERE ID_BOARD NOT IN (SELECT ID_BOARD FROM {$db_prefix}boards)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_polls WHERE ID_MEMBER NOT IN (SELECT ID_MEMBER FROM {$db_prefix}members)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_polls WHERE ID_POLL NOT IN (SELECT ID_POLL FROM {$db_prefix}polls)`))

	// Recount the forum-wide stats.
	a.updateStatsMessage()
	a.updateStatsTopic()
}
