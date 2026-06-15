package app

// Port of Sources/RemoveTopic.php: RemoveTopic2, DeleteMessage,
// RemoveOldTopics2, removeTopics, removeMessage.

import (
	"strings"
	"time"
)

func init() {
	registerAction("removetopic2", (*Ctx).RemoveTopic2)
	registerAction("deletemsg", (*Ctx).DeleteMessage)
	registerAction("removeoldtopics2", (*Ctx).RemoveOldTopics2)
}

// RemoveTopic2 is RemoveTopic2(): completely remove an entire topic.
func (c *Ctx) RemoveTopic2() {
	a := c.App

	// Make sure they aren't being lead around by someone. (:@)
	c.checkSession("get", "", true)

	var starter int
	var subject string
	a.DB.QueryRow(a.Q(`
		SELECT t.ID_MEMBER_STARTED, ms.subject
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS ms
		WHERE t.ID_TOPIC = ?
			AND ms.ID_MSG = t.ID_FIRST_MSG
		LIMIT 1`), c.Topic).Scan(&starter, &subject)

	if starter == c.User.ID && !c.allowedTo("remove_any") {
		c.isAllowedTo("remove_own")
	} else {
		c.isAllowedTo("remove_any")
	}

	// Notify people that this topic has been removed.
	c.sendNotifications(c.Topic, "remove")

	c.removeTopics([]int{c.Topic}, true, false)

	if c.allowedTo("remove_any") && (!c.allowedTo("remove_own") || starter != c.User.ID) {
		c.logAction("remove", map[string]any{"topic": c.Topic, "subject": subject, "member": starter})
	}

	c.redirectExit("board=" + itoa(c.Board) + ".0")
}

// DeleteMessage is DeleteMessage(): remove just a single post.
func (c *Ctx) DeleteMessage() {
	a := c.App

	c.checkSession("get", "", true)

	msgID := c.REQUEST.Int("msg")

	// Is $topic set?
	if c.Topic == 0 && c.REQUEST.Has("topic") {
		c.Topic = c.REQUEST.Int("topic")
	}

	var starter, poster int
	var subject string
	var postTime int64
	a.DB.QueryRow(a.Q(`
		SELECT t.ID_MEMBER_STARTED, m.ID_MEMBER, m.subject, m.posterTime
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m
		WHERE t.ID_TOPIC = ?
			AND m.ID_TOPIC = ?
			AND m.ID_MSG = ?
		LIMIT 1`), c.Topic, c.Topic, msgID).Scan(&starter, &poster, &subject, &postTime)

	if poster == c.User.ID {
		if !c.allowedTo("delete_own") {
			if starter == c.User.ID && !c.allowedTo("delete_any") {
				c.isAllowedTo("delete_replies")
			} else if !c.allowedTo("delete_any") {
				c.isAllowedTo("delete_own")
			}
		} else if !c.allowedTo("delete_any") && (starter != c.User.ID || !c.allowedTo("delete_replies")) &&
			!a.SettingEmpty("edit_disable_time") && postTime+int64(a.SettingInt("edit_disable_time"))*60 < time.Now().Unix() {
			c.fatalLangError("modify_post_time_passed", false)
		}
	} else if starter == c.User.ID && !c.allowedTo("delete_any") {
		c.isAllowedTo("delete_replies")
	} else {
		c.isAllowedTo("delete_any")
	}

	// If the full topic was removed go back to the board.
	fullTopic := c.removeMessage(msgID, true)

	if c.allowedTo("delete_any") && (!c.allowedTo("delete_own") || poster != c.User.ID) {
		c.logAction("delete", map[string]any{"topic": c.Topic, "subject": subject, "member": starter})
	}

	// We want to redirect back to recent action.
	if c.REQUEST.Has("recent") {
		c.redirectExit("action=recent")
	} else if fullTopic {
		c.redirectExit("board=" + itoa(c.Board) + ".0")
	} else {
		c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
	}
}

// RemoveOldTopics2 is RemoveOldTopics2(): prune old topics (admin
// maintenance).
func (c *Ctx) RemoveOldTopics2() {
	a := c.App

	c.isAllowedTo("admin_forum")
	c.checkSession("post", "maintain", true)

	// No boards at all?  Forget it then :/.
	boards := c.POST.Arr("boards")
	if boards == nil || boards.Len() == 0 {
		c.redirectExit("action=maintain")
	}

	// This should exist, but we can make sure.
	deleteType := c.POST.Str("delete_type")
	if !c.POST.Has("delete_type") {
		deleteType = "nothing"
	}

	// Custom conditions.
	condition := ""

	if deleteType == "moved" {
		// Just moved notice topics?
		condition += `
			AND m.icon = 'moved'
			AND t.locked = 1`
	} else if deleteType == "locked" {
		// Otherwise, maybe locked topics only?
		condition += `
			AND t.locked = 1`
	}

	// Exclude stickies?
	if c.POST.Has("delete_old_not_sticky") {
		condition += `
			AND t.isSticky = 0`
	}

	var boardIDs []string
	for _, k := range boards.Keys() {
		boardIDs = append(boardIDs, itoa(atoi(k)))
	}

	// All we're gonna do here is grab the ID_TOPICs and send them to
	// removeTopics().
	cutoff := time.Now().Unix() - 3600*24*int64(c.POST.Int("maxdays"))
	var topics []int
	rows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m
		WHERE m.ID_MSG = t.ID_LAST_MSG
			AND m.posterTime < ` + i64toa(cutoff) + condition + `
			AND t.ID_BOARD IN (` + strings.Join(boardIDs, ", ") + `)`))
	if err == nil {
		for rows.Next() {
			var t int
			rows.Scan(&t)
			topics = append(topics, t)
		}
		rows.Close()
	}

	c.removeTopics(topics, false, true)

	// Log an action into the moderation log.
	c.logAction("pruned", map[string]any{"days": c.POST.Int("maxdays")})

	c.redirectExit("action=maintain;done")
}

// removeTopics is removeTopics($topics, $decreasePostCount,
// $ignoreRecycling). Permissions are NOT checked here!
func (c *Ctx) removeTopics(topics []int, decreasePostCount, ignoreRecycling bool) {
	a := c.App

	// Nothing to do?
	if len(topics) == 0 {
		return
	}

	inList := func(ts []int) string {
		ids := make([]string, len(ts))
		for i, t := range ts {
			ids[i] = itoa(t)
		}
		return strings.Join(ids, ", ")
	}
	condition := "IN (" + inList(topics) + ")"

	// Decrease the post counts.
	if decreasePostCount {
		rows, err := a.DB.Query(a.Q(`
			SELECT m.ID_MEMBER, COUNT(*) AS posts
			FROM {$db_prefix}messages AS m, {$db_prefix}boards AS b
			WHERE m.ID_TOPIC ` + condition + `
				AND b.ID_BOARD = m.ID_BOARD
				AND m.icon != 'recycled'
				AND b.countPosts = 0
			GROUP BY m.ID_MEMBER`))
		if err == nil {
			type memberPosts struct{ member, posts int }
			var mps []memberPosts
			for rows.Next() {
				var mp memberPosts
				rows.Scan(&mp.member, &mp.posts)
				mps = append(mps, mp)
			}
			rows.Close()
			for _, mp := range mps {
				a.updateMemberPosts(mp.member, -mp.posts)
			}
		}
	}

	// Recycle topics that aren't in the recycle board...
	if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 && !ignoreRecycling {
		var recycleTopics []int
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_TOPIC
			FROM {$db_prefix}topics
			WHERE ID_TOPIC ` + condition + `
				AND ID_BOARD != ` + itoa(a.SettingInt("recycle_board"))))
		if err == nil {
			for rows.Next() {
				var t int
				rows.Scan(&t)
				recycleTopics = append(recycleTopics, t)
			}
			rows.Close()
		}

		if len(recycleTopics) > 0 {
			// Mark recycled topics as recycled.
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}messages
				SET icon = 'recycled'
				WHERE ID_TOPIC IN (` + inList(recycleTopics) + `)`))

			// De-sticky and unlock topics.
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}topics
				SET
					locked = 0,
					isSticky = 0
				WHERE ID_TOPIC IN (` + inList(recycleTopics) + `)`))

			// Move the topics to the recycle board.
			c.moveTopics(recycleTopics, a.SettingInt("recycle_board"))

			// Topics that were recycled don't need to be deleted, so
			// subtract them.
			recycled := map[int]bool{}
			for _, t := range recycleTopics {
				recycled[t] = true
			}
			var left []int
			for _, t := range topics {
				if !recycled[t] {
					left = append(left, t)
				}
			}
			topics = left

			// Topic list has changed, so does the condition to select
			// topics.
			condition = "IN (" + inList(topics) + ")"
		}
	}

	// Still topics left to delete?
	if len(topics) == 0 {
		return
	}

	type boardStats struct{ numPosts, numTopics, idBoard int }
	var adjustBoards []boardStats

	// Find out how many posts we are deleting.
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_BOARD, COUNT(*) AS numTopics, IFNULL(SUM(numReplies), 0) AS numReplies
		FROM {$db_prefix}topics
		WHERE ID_TOPIC ` + condition + `
		GROUP BY ID_BOARD`))
	if err == nil {
		for rows.Next() {
			var b boardStats
			var numReplies int
			rows.Scan(&b.idBoard, &b.numTopics, &numReplies)
			// The numReplies is only the *replies*.  There're also the first
			// posts in the topics.
			b.numPosts = numReplies + b.numTopics
			adjustBoards = append(adjustBoards, b)
		}
		rows.Close()
	}

	// Decrease the posts/topics...
	for _, stats := range adjustBoards {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}boards
			SET
				numTopics = IIF(? > numTopics, 0, numTopics - ?),
				numPosts = IIF(? > numPosts, 0, numPosts - ?)
			WHERE ID_BOARD = ?`), stats.numTopics, stats.numTopics, stats.numPosts, stats.numPosts, stats.idBoard)
	}

	// Remove Polls.
	var polls []string
	rows, err = a.DB.Query(a.Q(`
		SELECT ID_POLL
		FROM {$db_prefix}topics
		WHERE ID_TOPIC ` + condition + `
			AND ID_POLL > 0`))
	if err == nil {
		for rows.Next() {
			var p int
			rows.Scan(&p)
			polls = append(polls, itoa(p))
		}
		rows.Close()
	}

	if len(polls) > 0 {
		pollCondition := "IN (" + strings.Join(polls, ", ") + ")"
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}polls WHERE ID_POLL ` + pollCondition))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}poll_choices WHERE ID_POLL ` + pollCondition))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_polls WHERE ID_POLL ` + pollCondition))
	}

	// Get rid of the attachment, if it exists.
	a.removeAttachments("a.attachmentType = 0 AND m.ID_TOPIC "+condition, "messages", false, true)

	// (The custom search index branch is dropped: LIKE search only.)

	// Delete anything related to the topic.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}messages WHERE ID_TOPIC ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_TOPIC ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_notify WHERE ID_TOPIC ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}topics WHERE ID_TOPIC ` + condition))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_subjects WHERE ID_TOPIC ` + condition))

	// Update the totals...
	a.updateStatsMessage()
	a.updateStatsTopic()

	var updates []int
	for _, stats := range adjustBoards {
		updates = append(updates, stats.idBoard)
	}
	c.updateLastMessages(updates, 0)
}

// removeMessage is removeMessage($message, $decreasePostCount): remove a
// specific message (including permission checks).
func (c *Ctx) removeMessage(message int, decreasePostCount bool) bool {
	a := c.App

	if message <= 0 {
		return false
	}

	var rowMember, idTopic, firstMsg, lastMsg, numReplies, idBoard, starterID, countPosts int
	var icon, subject string
	var posterTime int64
	err := a.DB.QueryRow(a.Q(`
		SELECT
			m.ID_MEMBER, m.icon, m.posterTime, m.subject,
			t.ID_TOPIC, t.ID_FIRST_MSG, t.ID_LAST_MSG, t.numReplies, t.ID_BOARD,
			t.ID_MEMBER_STARTED AS ID_MEMBER_POSTER,
			b.countPosts
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t, {$db_prefix}boards AS b
		WHERE m.ID_MSG = ?
			AND t.ID_TOPIC = m.ID_TOPIC
			AND b.ID_BOARD = t.ID_BOARD
		LIMIT 1`), message).Scan(&rowMember, &icon, &posterTime, &subject,
		&idTopic, &firstMsg, &lastMsg, &numReplies, &idBoard, &starterID, &countPosts)
	if err != nil {
		return false
	}

	inBoards := func(list []int, board int) bool {
		for _, b := range list {
			if b == 0 || b == board {
				return true
			}
		}
		return false
	}

	if c.Board == 0 || idBoard != c.Board {
		deleteAny := c.boardsAllowedTo("delete_any")

		if !inBoards(deleteAny, idBoard) {
			deleteOwn := inBoards(c.boardsAllowedTo("delete_own"), idBoard)
			deleteReplies := inBoards(c.boardsAllowedTo("delete_replies"), idBoard)

			if rowMember == c.User.ID {
				if !deleteOwn {
					if starterID == c.User.ID {
						if !deleteReplies {
							c.fatalLangError("cannot_delete_replies", true)
						}
					} else {
						c.fatalLangError("cannot_delete_own", true)
					}
				} else if (starterID != c.User.ID || !deleteReplies) &&
					!a.SettingEmpty("edit_disable_time") && posterTime+int64(a.SettingInt("edit_disable_time"))*60 < time.Now().Unix() {
					c.fatalLangError("modify_post_time_passed", false)
				}
			} else if starterID == c.User.ID {
				if !deleteReplies {
					c.fatalLangError("cannot_delete_replies", true)
				}
			} else {
				c.fatalLangError("cannot_delete_any", true)
			}
		}
	} else {
		// Check permissions to delete this message.
		if rowMember == c.User.ID {
			if !c.allowedTo("delete_own") {
				if starterID == c.User.ID && !c.allowedTo("delete_any") {
					c.isAllowedTo("delete_replies")
				} else if !c.allowedTo("delete_any") {
					c.isAllowedTo("delete_own")
				}
			} else if !c.allowedTo("delete_any") && (starterID != c.User.ID || !c.allowedTo("delete_replies")) &&
				!a.SettingEmpty("edit_disable_time") && posterTime+int64(a.SettingInt("edit_disable_time"))*60 < time.Now().Unix() {
				c.fatalLangError("modify_post_time_passed", false)
			}
		} else if starterID == c.User.ID && !c.allowedTo("delete_any") {
			c.isAllowedTo("delete_replies")
		} else {
			c.isAllowedTo("delete_any")
		}
	}

	// Delete the *whole* topic, but only if the topic consists of one
	// message.
	if firstMsg == message {
		if c.Board == 0 || idBoard != c.Board {
			removeAny := inBoards(c.boardsAllowedTo("remove_any"), idBoard)
			removeOwn := false
			if !removeAny {
				removeOwn = inBoards(c.boardsAllowedTo("remove_own"), idBoard)
			}

			if rowMember != c.User.ID && !removeAny {
				c.fatalLangError("cannot_remove_any", true)
			} else if !removeAny && !removeOwn {
				c.fatalLangError("cannot_remove_own", true)
			}
		} else {
			// Check permissions to delete a whole topic.
			if rowMember != c.User.ID {
				c.isAllowedTo("remove_any")
			} else if !c.allowedTo("remove_any") {
				c.isAllowedTo("remove_own")
			}
		}

		// ...if there is only one post.
		if numReplies != 0 {
			c.fatalLangError("delFirstPost", false)
		}

		c.removeTopics([]int{idTopic}, true, false)
		return true
	}

	// Default recycle to false.
	recycle := false

	// If recycle topics has been set, make a copy of this message in the
	// recycle board. Make sure we're not recycling messages that are already
	// on the recycle board.
	if !a.SettingEmpty("recycle_enable") && idBoard != a.SettingInt("recycle_board") && icon != "recycled" {
		// Check if the recycle board exists and if so get the read status.
		var isRead int
		err := a.DB.QueryRow(a.Q(`
			SELECT (IFNULL(lb.ID_MSG, 0) >= b.ID_MSG_UPDATED) AS isSeen
			FROM {$db_prefix}boards AS b
				LEFT JOIN {$db_prefix}log_boards AS lb ON (lb.ID_BOARD = b.ID_BOARD AND lb.ID_MEMBER = ?)
			WHERE b.ID_BOARD = ?`), c.User.ID, a.SettingInt("recycle_board")).Scan(&isRead)
		if err != nil {
			c.fatalLangError("recycle_no_valid_board", true)
		}

		// Insert a new topic in the recycle board.
		res, err := a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}topics
				(ID_BOARD, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_FIRST_MSG, ID_LAST_MSG)
			VALUES (?, ?, ?, ?, ?)`), a.SettingInt("recycle_board"), rowMember, rowMember, message, message)

		// Capture the ID of the new topic...
		topicID := 0
		if err == nil {
			id, _ := res.LastInsertId()
			topicID = int(id)
		}

		// If the topic creation went successful, move the message.
		if topicID > 0 {
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}messages
				SET
					ID_TOPIC = ?,
					ID_BOARD = ?,
					icon = 'recycled'
				WHERE ID_MSG = ?`), topicID, a.SettingInt("recycle_board"), message)

			// Mark recycled topic as read.
			if !c.User.IsGuest {
				a.DB.Exec(a.Q(`
					REPLACE INTO {$db_prefix}log_topics
						(ID_TOPIC, ID_MEMBER, ID_MSG)
					VALUES (?, ?, ?)`), topicID, c.User.ID, a.SettingInt("maxMsgID"))
			}

			// Mark recycle board as seen, if it was marked as seen before.
			if isRead != 0 && !c.User.IsGuest {
				a.DB.Exec(a.Q(`
					REPLACE INTO {$db_prefix}log_boards
						(ID_BOARD, ID_MEMBER, ID_MSG)
					VALUES (?, ?, ?)`), a.SettingInt("recycle_board"), c.User.ID, a.SettingInt("maxMsgID"))
			}

			// Add one topic and post to the recycle bin board.
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}boards
				SET
					numTopics = numTopics + 1,
					numPosts = numPosts + 1
				WHERE ID_BOARD = ?`), a.SettingInt("recycle_board"))

			// Make sure this message isn't getting deleted later on.
			recycle = true

			// Make sure we update the search subject index.
			c.updateStatsSubject(topicID, subject)
		}
	}

	// Deleting a recycled message can not lower anyone's post count.
	if icon == "recycled" {
		decreasePostCount = false
	}

	// This is the last post, update the last post on the board.
	if lastMsg == message {
		// Find the last message, set it, and decrease the post count.
		var newLastMsg, newLastMember int
		a.DB.QueryRow(a.Q(`
			SELECT ID_MSG, ID_MEMBER
			FROM {$db_prefix}messages
			WHERE ID_TOPIC = ?
				AND ID_MSG != ?
			ORDER BY ID_MSG DESC
			LIMIT 1`), idTopic, message).Scan(&newLastMsg, &newLastMember)

		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET
				ID_LAST_MSG = ?,
				numReplies = IIF(numReplies = 0, 0, numReplies - 1),
				ID_MEMBER_UPDATED = ?
			WHERE ID_TOPIC = ?`), newLastMsg, newLastMember, idTopic)
	} else {
		// Only decrease post counts.
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET numReplies = IIF(numReplies = 0, 0, numReplies - 1)
			WHERE ID_TOPIC = ?`), idTopic)
	}

	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}boards
		SET numPosts = IIF(numPosts = 0, 0, numPosts - 1)
		WHERE ID_BOARD = ?`), idBoard)

	// If the poster was registered and the board this message was on
	// incremented the member's posts when it was posted, decrease his or her
	// post count.
	if rowMember != 0 && decreasePostCount && countPosts == 0 {
		// SMF's updateMemberData('posts' => '-') both decrements the count and
		// recomputes the member's post group (updateStats('postgroups', ...)).
		a.updateMemberPosts(rowMember, -1)
		c.updateStatsPostgroups(rowMember)
	}

	// Only remove posts if they're not recycled.
	if !recycle {
		// Remove the message!
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}messages WHERE ID_MSG = ?`), message)

		// Delete attachment(s) if they exist.
		a.removeAttachments("a.attachmentType = 0 AND a.ID_MSG = "+itoa(message), "", false, true)
	}

	// Update the pesky statistics.
	a.updateStatsMessage()
	a.updateStatsTopic()

	// And now to update the last message of each board we messed with.
	if recycle {
		c.updateLastMessages([]int{idBoard, a.SettingInt("recycle_board")}, 0)
	} else {
		c.updateLastMessages([]int{idBoard}, 0)
	}

	return false
}
