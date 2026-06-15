package app

// Port of Sources/MoveTopic.php: MoveTopic (the reason/destination form),
// MoveTopic2 (execute the move) and moveTopics (the low-level mover, used
// here and by the recycle bin / quick moderation).

import "strings"

func init() {
	registerAction("movetopic", (*Ctx).MoveTopic)
	registerAction("movetopic2", (*Ctx).MoveTopic2)
}

// MoveTopicBoard is one selectable destination board.
type MoveTopicBoard struct {
	ID         int
	Name       string
	Category   string
	ChildLevel int
	Selected   bool
}

// MoveTopicCtx is the page context for MoveTopic.template.php.
type MoveTopicCtx struct {
	Subject     string
	Boards      []MoveTopicBoard
	BackToTopic bool
}

// MoveTopic is MoveTopic(): give the moderator a chance to post a reason.
func (c *Ctx) MoveTopic() {
	a := c.App

	if c.Topic == 0 {
		c.fatalLangError("1", true)
	}

	var idMemberStarted int
	var subject string
	a.DB.QueryRow(a.Q(`
		SELECT t.ID_MEMBER_STARTED, ms.subject
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS ms
		WHERE t.ID_TOPIC = ?
			AND ms.ID_MSG = t.ID_FIRST_MSG
		LIMIT 1`), c.Topic).Scan(&idMemberStarted, &subject)

	// Permission check!
	if !c.allowedTo("move_any") {
		if idMemberStarted == c.User.ID {
			c.isAllowedTo("move_own")
		} else {
			c.isAllowedTo("move_any")
		}
	}

	page := &MoveTopicCtx{Subject: subject}
	c.Page = page

	// Get a list of boards this moderator can move to.
	moveTo := c.Session.GetInt("move_to_topic")
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name, b.childLevel, c.name AS catName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE b.ID_BOARD != ?
			AND `+c.User.QuerySeeBoard), c.Board)
	if err == nil {
		for rows.Next() {
			var b MoveTopicBoard
			rows.Scan(&b.ID, &b.Name, &b.ChildLevel, &b.Category)
			b.Selected = moveTo != 0 && int64(b.ID) == moveTo
			page.Boards = append(page.Boards, b)
		}
		rows.Close()
	}

	if len(page.Boards) == 0 {
		c.fatalLangError("moveto_noboards", false)
	}

	c.PageTitle = c.Txt("132")
	page.BackToTopic = c.REQUEST.Has("goback")

	c.SubTemplate = templateMoveTopic

	// Register this form and get a sequence number in $context.
	c.checkSubmitOnce("register")
}

// MoveTopic2 is MoveTopic2(): execute the move.
func (c *Ctx) MoveTopic2() {
	a := c.App
	scripturl := a.ScriptURL

	// Make sure this form hasn't been submitted before.
	c.checkSubmitOnce("check")

	var idMemberStarted, idFirstMsg int
	a.DB.QueryRow(a.Q(`
		SELECT ID_MEMBER_STARTED, ID_FIRST_MSG
		FROM {$db_prefix}topics
		WHERE ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&idMemberStarted, &idFirstMsg)

	// Can they move topics on this board?
	if !c.allowedTo("move_any") {
		if idMemberStarted == c.User.ID {
			c.isAllowedTo("move_own")
		} else {
			c.isAllowedTo("move_any")
		}
	}

	c.checkSession("post", "", true)

	// The destination board must be numeric.
	toboard := c.POST.Int("toboard")

	// Make sure they can see the board they are trying to move to (and get
	// whether posts count in the target board).
	var pcounter int
	var boardName, subject string
	err := a.DB.QueryRow(a.Q(`
		SELECT b.countPosts, b.name, m.subject
		FROM {$db_prefix}boards AS b, {$db_prefix}topics AS t, {$db_prefix}messages AS m
		WHERE `+c.User.QuerySeeBoard+`
			AND b.ID_BOARD = ?
			AND t.ID_TOPIC = ?
			AND m.ID_MSG = t.ID_FIRST_MSG
		LIMIT 1`), toboard, c.Topic).Scan(&pcounter, &boardName, &subject)
	if err != nil {
		c.fatalLangError("smf232", false)
	}

	// Remember this for later.
	c.Session.Set("move_to_topic", toboard)

	// Rename the topic...
	if c.POST.Has("reset_subject") && c.POST.Str("custom_subject") != "" {
		customSubject := Htmlspecialchars(c.POST.Str("custom_subject"))

		if c.POST.Has("enforce_subject") {
			// Get a response prefix (single-language port: forum default ==
			// user language, so no loadLanguage dance).
			responsePrefix := c.Txt("response_prefix")
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}messages
				SET subject = ?
				WHERE ID_TOPIC = ?`), responsePrefix+customSubject, c.Topic)
		}

		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}messages
			SET subject = ?
			WHERE ID_MSG = ?`), customSubject, idFirstMsg)

		// Fix the subject cache.
		c.updateStatsSubject(c.Topic, customSubject)
	}

	// Create a link to this in the old board.
	if c.POST.Has("postRedirect") {
		reason := Htmlspecialchars(c.POST.Str("reason"))
		reason = c.preparsecode(reason, false)

		// Add a URL onto the message.
		reason = strings.NewReplacer(
			c.Txt("movetopic_auto_board"), "[url="+scripturl+"?board="+itoa(toboard)+"]"+boardName+"[/url]",
			c.Txt("movetopic_auto_topic"), "[iurl]"+scripturl+"?topic="+itoa(c.Topic)+".0[/iurl]",
		).Replace(reason)

		msg := &msgOptions{
			Subject:        c.Txt("smf56") + ": " + subject,
			Body:           reason,
			Icon:           "moved",
			SmileysEnabled: true,
		}
		lock := 1
		topicOpt := &topicOptions{
			Board:      c.Board,
			LockMode:   &lock,
			MarkAsRead: true,
		}
		poster := &posterOptions{
			ID:              c.User.ID,
			UpdatePostCount: pcounter != 0,
		}
		c.createPost(msg, topicOpt, poster)
	}

	var pcounterFrom int
	a.DB.QueryRow(a.Q(`
		SELECT countPosts
		FROM {$db_prefix}boards
		WHERE ID_BOARD = ?
		LIMIT 1`), c.Board).Scan(&pcounterFrom)

	if pcounterFrom != pcounter {
		var posters []int
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}messages
			WHERE ID_TOPIC = ?`), c.Topic)
		if err == nil {
			for rows.Next() {
				var m int
				rows.Scan(&m)
				posters = append(posters, m)
			}
			rows.Close()
		}

		// The board we're moving from counted posts, but not to: -1 each.
		// The reverse (from didn't, to did): +1 each.
		delta := 1
		if pcounterFrom == 0 {
			delta = -1
		}
		for _, m := range posters {
			a.updateMemberPosts(m, delta)
		}
	}

	// Do the move (includes statistics update needed for the redirect topic).
	c.moveTopics([]int{c.Topic}, toboard)

	// Log that they moved this topic.
	if !c.allowedTo("move_own") || idMemberStarted != c.User.ID {
		c.logAction("move", map[string]any{"topic": c.Topic, "board_from": c.Board, "board_to": toboard})
	}
	// Notify people that this topic has been moved?
	c.sendNotifications(c.Topic, "move")

	// Why not go back to the original board in case they want to keep moving?
	if !c.REQUEST.Has("goback") {
		c.redirectExit("board=" + itoa(c.Board) + ".0")
	} else {
		c.redirectExit("topic=" + itoa(c.Topic) + ".0")
	}
}

// moveTopics is moveTopics($topics, $toBoard).
func (c *Ctx) moveTopics(topics []int, toBoard int) {
	a := c.App

	// Empty array?
	if len(topics) == 0 {
		return
	}
	ids := make([]string, len(topics))
	for i, t := range topics {
		ids[i] = itoa(t)
	}
	condition := "IN (" + strings.Join(ids, ", ") + ")"

	// Destination board empty or equal to 0?
	if toBoard == 0 {
		return
	}

	// Determine the source boards...
	type boardStats struct{ numPosts, numTopics, idBoard int }
	var fromBoards []boardStats
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_BOARD, COUNT(*) AS numTopics, IFNULL(SUM(numReplies), 0) AS numReplies
		FROM {$db_prefix}topics
		WHERE ID_TOPIC ` + condition + `
		GROUP BY ID_BOARD`))
	if err != nil {
		return
	}
	for rows.Next() {
		var b boardStats
		var numReplies int
		rows.Scan(&b.idBoard, &b.numTopics, &numReplies)
		// Posts = (numReplies + 1) for each topic.
		b.numPosts = numReplies + b.numTopics
		fromBoards = append(fromBoards, b)
	}
	rows.Close()
	// Num of rows = 0 -> no topics found.
	if len(fromBoards) == 0 {
		return
	}

	// Move over the mark_read data. (because it may be read and now not by
	// some!)
	saveAServer := a.SettingInt("maxMsgID") - 50000
	if saveAServer < 0 {
		saveAServer = 0
	}
	type logTopic struct{ topic, member, msg int }
	var logTopics []logTopic
	rows, err = a.DB.Query(a.Q(`
		SELECT lmr.ID_MEMBER, lmr.ID_MSG, t.ID_TOPIC
		FROM {$db_prefix}topics AS t, {$db_prefix}log_mark_read AS lmr
			LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = lmr.ID_MEMBER)
		WHERE t.ID_TOPIC ` + condition + `
			AND lmr.ID_BOARD = t.ID_BOARD
			AND lmr.ID_MSG > t.ID_FIRST_MSG
			AND lmr.ID_MSG > ` + itoa(saveAServer) + `
			AND lmr.ID_MSG > IFNULL(lt.ID_MSG, 0)`))
	if err == nil {
		for rows.Next() {
			var lt logTopic
			rows.Scan(&lt.member, &lt.msg, &lt.topic)
			logTopics = append(logTopics, lt)
		}
		rows.Close()
	}

	// Now that we have all the topics that *should* be marked read, and by
	// which members...
	for _, lt := range logTopics {
		a.DB.Exec(a.Q(`
			REPLACE INTO {$db_prefix}log_topics
				(ID_TOPIC, ID_MEMBER, ID_MSG)
			VALUES (?, ?, ?)`), lt.topic, lt.member, lt.msg)
	}

	// Update the number of posts on each board.
	totalTopics := 0
	totalPosts := 0
	for _, stats := range fromBoards {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}boards
			SET
				numPosts = IIF(? > numPosts, 0, numPosts - ?),
				numTopics = IIF(? > numTopics, 0, numTopics - ?)
			WHERE ID_BOARD = ?`), stats.numPosts, stats.numPosts, stats.numTopics, stats.numTopics, stats.idBoard)
		totalTopics += stats.numTopics
		totalPosts += stats.numPosts
	}
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}boards
		SET
			numTopics = numTopics + ?,
			numPosts = numPosts + ?
		WHERE ID_BOARD = ?`), totalTopics, totalPosts, toBoard)

	// Move the topic.  Done.  :P
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_BOARD = ` + itoa(toBoard) + ` WHERE ID_TOPIC ` + condition))
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET ID_BOARD = ` + itoa(toBoard) + ` WHERE ID_TOPIC ` + condition))

	// Mark target board as seen, if it was already marked as seen before.
	var isSeen int
	a.DB.QueryRow(a.Q(`
		SELECT (IFNULL(lb.ID_MSG, 0) >= b.ID_MSG_UPDATED) AS isSeen
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}log_boards AS lb ON (lb.ID_BOARD = b.ID_BOARD AND lb.ID_MEMBER = ?)
		WHERE b.ID_BOARD = ?`), c.User.ID, toBoard).Scan(&isSeen)

	if isSeen != 0 && !c.User.IsGuest {
		a.DB.Exec(a.Q(`
			REPLACE INTO {$db_prefix}log_boards
				(ID_BOARD, ID_MEMBER, ID_MSG)
			VALUES (?, ?, ?)`), toBoard, c.User.ID, a.SettingInt("maxMsgID"))
	}

	// Update 'em pesky stats.
	a.updateStatsTopic()
	a.updateStatsMessage()

	updates := []int{toBoard}
	for _, stats := range fromBoards {
		updates = append(updates, stats.idBoard)
	}
	c.updateLastMessages(uniqueInts(updates), 0)
}
