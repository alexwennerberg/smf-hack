package app

// Port of Sources/Subs-Boards.php (Phase 2 scope: MarkRead, markBoardsRead,
// CollapseCategory; Phase 6: QuickModeration/QuickModeration2).

import "strings"

func init() {
	registerAction("markasread", (*Ctx).MarkRead)
	registerAction("collapse", (*Ctx).CollapseCategory)
	registerAction("quickmod", (*Ctx).QuickModeration)
	registerAction("quickmod2", (*Ctx).QuickModeration2)
}

// getMsgMemberID is getMsgMemberID($messageID): the (still-existing) member
// who posted a message, or 0.
func (c *Ctx) getMsgMemberID(messageID int) int {
	var memberID int
	c.App.DB.QueryRow(c.App.Q(`
		SELECT IFNULL(mem.ID_MEMBER, 0)
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_MSG = ?
		LIMIT 1`), messageID).Scan(&memberID)
	return memberID
}

// inInts reports whether want is in xs (in_array equivalent).
func inInts(xs []int, want int) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// markBoardsRead is markBoardsRead($boards, $unread).
func (c *Ctx) markBoardsRead(boards []int, unread bool) {
	a := c.App
	boards = uniqueInts(boards)
	if len(boards) == 0 {
		return
	}
	ids := make([]string, len(boards))
	for i, b := range boards {
		ids[i] = itoa(b)
	}
	in := strings.Join(ids, ", ")

	if unread {
		// Clear out all the places where this lovely info is stored.
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_mark_read WHERE ID_BOARD IN (` + in + `) AND ID_MEMBER = ` + itoa(c.User.ID)))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_boards WHERE ID_BOARD IN (` + in + `) AND ID_MEMBER = ` + itoa(c.User.ID)))
	} else {
		// Otherwise mark the board as read.
		maxMsg := a.SettingInt("maxMsgID")
		for _, b := range boards {
			a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_mark_read (ID_MSG, ID_MEMBER, ID_BOARD) VALUES (?, ?, ?)`),
				maxMsg, c.User.ID, b)
			a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_boards (ID_MSG, ID_MEMBER, ID_BOARD) VALUES (?, ?, ?)`),
				maxMsg, c.User.ID, b)
		}
	}
}

// MarkRead is MarkRead() (?action=markasread).
func (c *Ctx) MarkRead() {
	a := c.App

	// No Guests allowed!
	c.isNotGuest("")

	c.checkSession("get", "", true)

	sa := c.REQUEST.Str("sa")
	switch sa {
	case "all":
		// Find all the boards this user can see.
		var boards []int
		rows, err := a.DB.Query(a.Q(`SELECT b.ID_BOARD FROM {$db_prefix}boards AS b WHERE ` + c.User.QuerySeeBoard))
		if err == nil {
			for rows.Next() {
				var b int
				rows.Scan(&b)
				boards = append(boards, b)
			}
			rows.Close()
		}
		if len(boards) > 0 {
			c.markBoardsRead(boards, c.REQUEST.Has("unread"))
		}

		c.Session.Set("ID_MSG_LAST_VISIT", a.SettingInt("maxMsgID"))
		if old := c.Session.GetStr("old_url"); old != "" && strings.Contains(old, "action=unread") {
			c.redirectExit("action=unread")
		}
		c.redirectExit("")

	case "unreadreplies":
		// Make sure all the topics are integers!
		for _, t := range strings.Split(c.REQUEST.Str("topics"), "-") {
			a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_topics (ID_MSG, ID_MEMBER, ID_TOPIC) VALUES (?, ?, ?)`),
				a.SettingInt("maxMsgID"), c.User.ID, atoi(t))
		}
		c.redirectExit("action=unreadreplies")

	case "topic":
		// Special case: mark a topic unread!
		var earlyMsg int
		if !empty(c.GET.Str("t")) {
			// Get the latest message before this one.
			a.DB.QueryRow(a.Q(`
				SELECT IFNULL(MAX(ID_MSG), 0)
				FROM {$db_prefix}messages
				WHERE ID_TOPIC = ?
					AND ID_MSG < ?`), c.Topic, c.GET.Int("t")).Scan(&earlyMsg)
		}
		if earlyMsg == 0 {
			a.DB.QueryRow(a.Q(`
				SELECT ID_MSG
				FROM {$db_prefix}messages
				WHERE ID_TOPIC = ?
				ORDER BY ID_MSG
				LIMIT 1 OFFSET ?`), c.Topic, c.Start).Scan(&earlyMsg)
		}

		earlyMsg--

		// Use a time one second earlier than the first time: blam, unread!
		a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_topics (ID_MSG, ID_MEMBER, ID_TOPIC) VALUES (?, ?, ?)`),
			earlyMsg, c.User.ID, c.Topic)

		c.redirectExit("board=" + itoa(c.Board) + ".0")

	default:
		var categories, boards []int
		if c.REQUEST.Has("c") {
			for _, cc := range strings.Split(c.REQUEST.Str("c"), ",") {
				categories = append(categories, atoi(cc))
			}
		}
		if c.REQUEST.Has("boards") {
			for _, b := range strings.Split(c.REQUEST.Str("boards"), ",") {
				boards = append(boards, atoi(b))
			}
		}
		if c.Board != 0 {
			boards = append(boards, c.Board)
		}

		var clauses []string
		if len(categories) > 0 {
			ids := make([]string, len(categories))
			for i, v := range categories {
				ids[i] = itoa(v)
			}
			clauses = append(clauses, "ID_CAT IN ("+strings.Join(ids, ", ")+")")
		}
		if len(boards) > 0 {
			ids := make([]string, len(boards))
			for i, v := range boards {
				ids[i] = itoa(v)
			}
			clauses = append(clauses, "ID_BOARD IN ("+strings.Join(ids, ", ")+")")
		}

		if len(clauses) == 0 {
			c.redirectExit("")
		}

		var seeBoards []int
		rows, err := a.DB.Query(a.Q(`
			SELECT b.ID_BOARD
			FROM {$db_prefix}boards AS b
			WHERE ` + c.User.QuerySeeBoard + `
				AND b.` + strings.Join(clauses, " OR b.")))
		if err == nil {
			for rows.Next() {
				var b int
				rows.Scan(&b)
				seeBoards = append(seeBoards, b)
			}
			rows.Close()
		}

		if len(seeBoards) == 0 {
			c.redirectExit("")
		}

		c.markBoardsRead(seeBoards, c.REQUEST.Has("unread"))

		if !c.REQUEST.Has("unread") {
			// Mark all child boards as read too.
			ids := make([]string, len(seeBoards))
			for i, v := range seeBoards {
				ids[i] = itoa(v)
			}
			crows, err := a.DB.Query(a.Q(`
				SELECT b.ID_BOARD
				FROM {$db_prefix}boards AS b
				WHERE b.ID_PARENT IN (` + strings.Join(ids, ", ") + `)
					AND ` + c.User.QuerySeeBoard))
			if err == nil {
				for crows.Next() {
					var b int
					crows.Scan(&b)
					a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_boards (ID_MSG, ID_MEMBER, ID_BOARD) VALUES (?, ?, ?)`),
						a.SettingInt("maxMsgID"), c.User.ID, b)
				}
				crows.Close()
			}

			if c.Board == 0 {
				c.redirectExit("")
			}
			c.redirectExit("board=" + itoa(c.Board) + ".0")
		} else {
			if c.BoardInfo == nil || c.BoardInfo.Parent == 0 {
				c.redirectExit("")
			}
			c.redirectExit("board=" + itoa(c.BoardInfo.Parent) + ".0")
		}
	}
}

// CollapseCategory is CollapseCategory() (?action=collapse).
func (c *Ctx) CollapseCategory() {
	a := c.App

	c.checkSession("request", "", true)

	cat := c.REQUEST.Int("c")

	// Not very complicated... just make sure the value is there.
	if c.REQUEST.Str("sa") == "collapse" {
		a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}collapsed_categories (ID_CAT, ID_MEMBER) VALUES (?, ?)`),
			cat, c.User.ID)
	} else if c.REQUEST.Str("sa") == "expand" {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}collapsed_categories WHERE ID_MEMBER = ? AND ID_CAT = ?`),
			c.User.ID, cat)
	}

	// And go back to the back to board index.
	c.BoardIndex()
}

// QuickModeration is QuickModeration() from Subs-Boards.php: batch moderation
// of selected topics from the message index / recent / search lists. Actions
// are sticky/move/remove/lock/markread/merge, validated against the user's
// per-board permissions.
func (c *Ctx) QuickModeration() {
	a := c.App

	// Check the session = get or post.
	c.checkSession("request", "", true)

	if c.Session.Has("topicseen_cache") {
		c.Session.Set("topicseen_cache", []any{})
	}

	// Remember the last board they moved things to.
	if c.REQUEST.Has("move_to") {
		c.Session.Set("move_to_topic", c.REQUEST.Int("move_to"))
	}

	// Only a few possible actions.
	possibleActions := []string{"markread"}

	var boardsCan map[string][]int
	var redirectURL string
	if c.Board != 0 {
		can := func(perm string) []int {
			if c.allowedTo(perm) {
				return []int{c.Board}
			}
			return nil
		}
		boardsCan = map[string][]int{
			"make_sticky": can("make_sticky"),
			"move_any":    can("move_any"),
			"move_own":    can("move_own"),
			"remove_any":  can("remove_any"),
			"remove_own":  can("remove_own"),
			"lock_any":    can("lock_any"),
			"lock_own":    can("lock_own"),
			"merge_any":   can("merge_any"),
		}
		redirectURL = "board=" + itoa(c.Board) + "." + c.REQUEST.Str("start")
	} else {
		// !!! Ugly.  There's no getting around this, is there?
		boardsCan = map[string][]int{
			"make_sticky": c.boardsAllowedTo("make_sticky"),
			"move_any":    c.boardsAllowedTo("move_any"),
			"move_own":    c.boardsAllowedTo("move_own"),
			"remove_any":  c.boardsAllowedTo("remove_any"),
			"remove_own":  c.boardsAllowedTo("remove_own"),
			"lock_any":    c.boardsAllowedTo("lock_any"),
			"lock_own":    c.boardsAllowedTo("lock_own"),
			"merge_any":   c.boardsAllowedTo("merge_any"),
		}
		if c.POST.Has("redirect_url") {
			redirectURL = c.POST.Str("redirect_url")
		} else {
			redirectURL = c.Session.GetStr("old_url")
		}
	}

	if len(boardsCan["make_sticky"]) > 0 && !a.SettingEmpty("enableStickyTopics") {
		possibleActions = append(possibleActions, "sticky")
	}
	if len(boardsCan["move_any"]) > 0 || len(boardsCan["move_own"]) > 0 {
		possibleActions = append(possibleActions, "move")
	}
	if len(boardsCan["remove_any"]) > 0 || len(boardsCan["remove_own"]) > 0 {
		possibleActions = append(possibleActions, "remove")
	}
	if len(boardsCan["lock_any"]) > 0 || len(boardsCan["lock_own"]) > 0 {
		possibleActions = append(possibleActions, "lock")
	}
	if len(boardsCan["merge_any"]) > 0 {
		possibleActions = append(possibleActions, "merge")
	}

	// actions: ID_TOPIC => action, kept in request order via 'order'.
	order := []int{}
	actions := map[int]string{}
	addAction := func(t int, act string) {
		if _, ok := actions[t]; !ok {
			order = append(order, t)
		}
		actions[t] = act
	}

	// The direct method: $_REQUEST['actions'][ID_TOPIC] = action.
	if actionsArr := c.REQUEST.Arr("actions"); actionsArr != nil {
		actionsArr.Values(func(k string, v any) {
			s, _ := v.(string)
			addAction(atoi(k), s)
		})
	}

	// The other method: $_REQUEST['topics'] + $_REQUEST['qaction'].
	if topicsArr := c.REQUEST.Arr("topics"); topicsArr != nil {
		qaction := c.REQUEST.Str("qaction")
		// If the action isn't valid, just quit now.
		if qaction == "" || !inStrings(possibleActions, qaction) {
			c.redirectExit(redirectURL)
		}

		var topicIDs []int
		topicsArr.Values(func(k string, v any) {
			s, _ := v.(string)
			topicIDs = append(topicIDs, atoi(s))
		})

		// Merge requires all topics as one parameter and can be done at once.
		if qaction == "merge" {
			if len(topicIDs) < 2 {
				c.redirectExit(redirectURL)
			}
			c.MergeExecute(topicIDs)
			return
		}

		for _, t := range topicIDs {
			addAction(t, qaction)
		}
	}

	// Weird... how'd you get here?
	if len(order) == 0 {
		c.redirectExit(redirectURL)
	}

	// Validate each action (drop unknown ones).
	{
		var nOrder []int
		nActions := map[int]string{}
		for _, t := range order {
			if inStrings(possibleActions, actions[t]) {
				nOrder = append(nOrder, t)
				nActions[t] = actions[t]
			}
		}
		order, actions = nOrder, nActions
	}

	if len(order) > 0 {
		// Find all topics that *aren't* on this board (and validate perms).
		extra := ""
		if c.Board != 0 {
			extra = "\n\t\t\t\tAND ID_BOARD != " + itoa(c.Board)
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_TOPIC, ID_MEMBER_STARTED, ID_BOARD, locked
			FROM {$db_prefix}topics
			WHERE ID_TOPIC IN (` + joinInts(order) + `)` + extra))
		if err == nil {
			for rows.Next() {
				var topicID, memberStarted, boardID, locked int
				rows.Scan(&topicID, &memberStarted, &boardID, &locked)
				if c.Board != 0 {
					delete(actions, topicID)
					continue
				}
				// Goodness, this is fun.  We need to validate the action.
				switch actions[topicID] {
				case "sticky":
					if !inInts(boardsCan["make_sticky"], 0) && !inInts(boardsCan["make_sticky"], boardID) {
						delete(actions, topicID)
					}
				case "move":
					if !inInts(boardsCan["move_any"], 0) && !inInts(boardsCan["move_any"], boardID) &&
						(memberStarted != c.User.ID || (!inInts(boardsCan["move_own"], 0) && !inInts(boardsCan["move_own"], boardID))) {
						delete(actions, topicID)
					}
				case "remove":
					if !inInts(boardsCan["remove_any"], 0) && !inInts(boardsCan["remove_any"], boardID) &&
						(memberStarted != c.User.ID || (!inInts(boardsCan["remove_own"], 0) && !inInts(boardsCan["remove_own"], boardID))) {
						delete(actions, topicID)
					}
				case "lock":
					// NOTE: SMF 1.1 references an undefined $locked here (always
					// false); replicated by omitting that OR term.
					if !inInts(boardsCan["lock_any"], 0) && !inInts(boardsCan["lock_any"], boardID) &&
						(memberStarted != c.User.ID || (!inInts(boardsCan["lock_own"], 0) && !inInts(boardsCan["lock_own"], boardID))) {
						delete(actions, topicID)
					}
				}
			}
			rows.Close()
		}
	}

	var stickyCache, removeCache, lockCache, markCache []int
	moveTo := map[int]int{}
	var moveOrder []int

	type bpDelta struct{ topics, posts int }
	affected := map[int]*bpDelta{}
	var affectedOrder []int
	ensure := func(b int) *bpDelta {
		if _, ok := affected[b]; !ok {
			affected[b] = &bpDelta{}
			affectedOrder = append(affectedOrder, b)
		}
		return affected[b]
	}
	if c.Board != 0 {
		ensure(c.Board)
	}

	// Separate the actions.
	for _, topic := range order {
		action, ok := actions[topic]
		if !ok {
			continue
		}
		switch action {
		case "markread":
			markCache = append(markCache, topic)
		case "sticky":
			stickyCache = append(stickyCache, topic)
		case "move":
			to := c.REQUEST.Int("move_to")
			if mt := c.REQUEST.Arr("move_tos"); mt != nil && mt.Has(itoa(topic)) {
				to = atoi(mt.Str(itoa(topic)))
			}
			moveTo[topic] = to
			if to == 0 {
				continue
			}
			moveOrder = append(moveOrder, topic)
		case "remove":
			removeCache = append(removeCache, topic)
		case "lock":
			lockCache = append(lockCache, topic)
		}
	}

	// Do all the stickies...
	if len(stickyCache) > 0 {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET isSticky = IIF(isSticky = 1, 0, 1)
			WHERE ID_TOPIC IN (` + joinInts(stickyCache) + `)`))
	}

	// Move sucka! (the most complicated part....)
	var moveCache [][3]int // {topic, board_from, board_to}
	if len(moveOrder) > 0 {
		extra := ""
		if c.Board != 0 && !c.allowedTo("move_any") {
			extra = "\n\t\t\t\tAND ID_MEMBER_STARTED = " + itoa(c.User.ID)
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT numReplies, ID_TOPIC, ID_BOARD
			FROM {$db_prefix}topics
			WHERE ID_TOPIC IN (` + joinInts(moveOrder) + `)` + extra))
		if err == nil {
			for rows.Next() {
				var numReplies, topicID, boardID int
				rows.Scan(&numReplies, &topicID, &boardID)
				to := moveTo[topicID]
				numReplies++ // posts = numReplies + 1
				if to == 0 {
					continue
				}
				ensure(to)
				ensure(boardID)
				affected[boardID].topics--
				affected[boardID].posts -= numReplies
				affected[to].topics++
				affected[to].posts += numReplies

				a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_BOARD = ? WHERE ID_TOPIC = ?`), to, topicID)
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_BOARD = ? WHERE ID_TOPIC = ?`), to, topicID)
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}calendar SET ID_BOARD = ? WHERE ID_TOPIC = ?`), to, topicID)

				moveCache = append(moveCache, [3]int{topicID, boardID, to})
			}
			rows.Close()
		}

		for _, b := range affectedOrder {
			d := affected[b]
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}boards
				SET numPosts = numPosts + ?, numTopics = numTopics + ?
				WHERE ID_BOARD = ?`), d.posts, d.topics, b)
		}
	}

	// Now delete the topics...
	if len(removeCache) > 0 {
		// They can only delete their own topics. (we wouldn't be here if they
		// couldn't do that..)
		if c.Board != 0 && !c.allowedTo("remove_any") {
			rows, err := a.DB.Query(a.Q(`
				SELECT ID_TOPIC
				FROM {$db_prefix}topics
				WHERE ID_TOPIC IN (`+joinInts(removeCache)+`)
					AND ID_MEMBER_STARTED = ?`), c.User.ID)
			removeCache = nil
			if err == nil {
				for rows.Next() {
					var t int
					rows.Scan(&t)
					removeCache = append(removeCache, t)
				}
				rows.Close()
			}
		}

		if len(removeCache) > 0 {
			// Gotta send the notifications *first*!
			for _, topic := range removeCache {
				c.logAction("remove", map[string]any{"topic": topic})
				c.sendNotifications(topic, "remove")
			}
			c.removeTopics(removeCache, true, false)
		}
	}

	// And lastly, lock the topics...
	lockStatus := map[int]bool{}
	if len(lockCache) > 0 {
		if c.Board != 0 && !c.allowedTo("lock_any") {
			rows, err := a.DB.Query(a.Q(`
				SELECT ID_TOPIC, locked
				FROM {$db_prefix}topics
				WHERE ID_TOPIC IN (`+joinInts(lockCache)+`)
					AND ID_MEMBER_STARTED = ?
					AND locked IN (2, 0)`), c.User.ID)
			lockCache = nil
			if err == nil {
				for rows.Next() {
					var t, locked int
					rows.Scan(&t, &locked)
					lockCache = append(lockCache, t)
					lockStatus[t] = locked == 0
				}
				rows.Close()
			}
		} else {
			rows, err := a.DB.Query(a.Q(`
				SELECT ID_TOPIC, locked
				FROM {$db_prefix}topics
				WHERE ID_TOPIC IN (` + joinInts(lockCache) + `)`))
			if err == nil {
				for rows.Next() {
					var t, locked int
					rows.Scan(&t, &locked)
					lockStatus[t] = locked == 0
				}
				rows.Close()
			}
		}

		if len(lockCache) > 0 {
			newVal := "2"
			if c.allowedTo("lock_any") {
				newVal = "1"
			}
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}topics
				SET locked = IIF(locked = 0, ` + newVal + `, 0)
				WHERE ID_TOPIC IN (` + joinInts(lockCache) + `)`))
		}
	}

	if len(markCache) > 0 {
		maxMsg := a.SettingInt("maxMsgID")
		for _, topic := range markCache {
			a.DB.Exec(a.Q(`
				REPLACE INTO {$db_prefix}log_topics
					(ID_MSG, ID_MEMBER, ID_TOPIC)
				VALUES (?, ?, ?)`), maxMsg, c.User.ID, topic)
		}
	}

	for _, mc := range moveCache {
		c.logAction("move", map[string]any{"topic": mc[0], "board_from": mc[1], "board_to": mc[2]})
		c.sendNotifications(mc[0], "move")
	}
	// NOTE: SMF tests the whole $lockStatus array, so a non-empty batch always
	// notifies 'lock' (replicated).
	lockNotify := "unlock"
	if len(lockStatus) > 0 {
		lockNotify = "lock"
	}
	for _, topic := range lockCache {
		c.logAction("lock", map[string]any{"topic": topic})
		c.sendNotifications(topic, lockNotify)
	}
	for _, topic := range stickyCache {
		c.logAction("sticky", map[string]any{"topic": topic})
		c.sendNotifications(topic, "sticky")
	}

	a.updateStatsTopic()
	a.updateStatsMessage()
	a.updateStatsCalendar()

	if len(affectedOrder) > 0 {
		c.updateLastMessages(affectedOrder, 0)
	}

	c.redirectExit(redirectURL)
}

// QuickModeration2 is QuickModeration2() from Subs-Boards.php: in-topic
// quick deletion of selected messages.
func (c *Ctx) QuickModeration2() {
	a := c.App

	// Check the session = get or post.
	c.checkSession("request", "", true)

	msgsArr := c.REQUEST.Arr("msgs")
	if msgsArr == nil {
		c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
	}

	var messages []int
	msgsArr.Values(func(k string, v any) {
		s, _ := v.(string)
		messages = append(messages, atoi(s))
	})

	// Allowed to delete any message?
	allowedAll := false
	if c.allowedTo("delete_any") {
		allowedAll = true
	} else if c.allowedTo("delete_replies") {
		// Allowed to delete replies to their messages?
		var starter int
		a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER_STARTED
			FROM {$db_prefix}topics
			WHERE ID_TOPIC = ?
			LIMIT 1`), c.Topic).Scan(&starter)
		allowedAll = starter == c.User.ID
	}

	// Make sure they're allowed to delete their own messages, if not any.
	if !allowedAll {
		c.isAllowedTo("delete_own")
	}

	// Allowed to remove which messages?
	extra := ""
	if !allowedAll {
		extra = "\n\t\t\t\tAND ID_MEMBER = " + itoa(c.User.ID)
	}
	type msgInfo struct {
		subject string
		member  int
	}
	msgs := map[int]msgInfo{}
	var msgOrder []int
	editDisable := a.SettingInt("edit_disable_time")
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MSG, subject, ID_MEMBER, posterTime
		FROM {$db_prefix}messages
		WHERE ID_MSG IN (`+joinInts(messages)+`)
			AND ID_TOPIC = ?`+extra), c.Topic)
	if err == nil {
		for rows.Next() {
			var idMsg, member int
			var subject string
			var posterTime int64
			rows.Scan(&idMsg, &subject, &member, &posterTime)
			if !allowedAll && editDisable != 0 && posterTime+int64(editDisable)*60 < nowUnix() {
				continue
			}
			msgs[idMsg] = msgInfo{subject, member}
			msgOrder = append(msgOrder, idMsg)
		}
		rows.Close()
	}

	// Get the first message in the topic - because you can't delete that!
	var firstMessage, lastMessage int
	a.DB.QueryRow(a.Q(`
		SELECT ID_FIRST_MSG, ID_LAST_MSG
		FROM {$db_prefix}topics
		WHERE ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&firstMessage, &lastMessage)

	// Delete all the messages we know they can delete.
	for _, message := range msgOrder {
		// Just skip the first message.
		if message == firstMessage && message != lastMessage {
			continue
		}
		c.removeMessage(message, true)

		info := msgs[message]
		// Log this moderation action ;).
		if c.allowedTo("delete_any") && (!c.allowedTo("delete_own") || info.member != c.User.ID) {
			c.logAction("delete", map[string]any{"topic": c.Topic, "subject": info.subject, "member": info.member})
		}
	}

	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}
