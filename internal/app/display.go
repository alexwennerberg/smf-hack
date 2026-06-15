package app

// Port of Sources/Display.php (Display + prepareDisplayContext +
// loadAttachmentContext; Download/dlattach ports with Phase 3's attachment
// pipeline, and thumbnail creation likewise).

import (
	"math"
	"os"
	"strings"
	"time"
)

func init() {
	registerAction("*display", (*Ctx).Display)
	registerAction("display", (*Ctx).Display)
}

// DisplayAttachment is one entry from loadAttachmentContext().
type DisplayAttachment struct {
	ID         int
	Name       string
	Downloads  int
	Size       string
	ByteSize   int
	Href       string
	Link       string
	IsImage    bool
	RealWidth  int
	Width      int
	RealHeight int
	Height     int
	HasThumb   bool
	ThumbID    int
	ThumbHref  string
	ThumbJS    string
}

// DisplayMsg is one message row (the prepareDisplayContext output).
type DisplayMsg struct {
	Attachments  []DisplayAttachment
	Alternate    int
	ID           int
	Href         string
	Link         string
	Member       *MemberCtx
	Icon         string
	IconURL      string
	Subject      string
	Time         string
	Timestamp    int64
	Counter      int
	ModifiedTime string
	ModifiedName string
	Body         string
	New          bool
	FirstNew     bool
	CanModify    bool
	CanRemove    bool
	CanSeeIP     bool
}

// DisplayPollOption is one poll choice row.
type DisplayPollOption struct {
	ID         string
	Percent    string
	Votes      int
	VotedThis  bool
	Bar        string
	BarWidth   int
	Option     string
	VoteButton string
}

// DisplayPoll is $context['poll'].
type DisplayPoll struct {
	ID             int
	Image          string
	Question       string
	TotalVotes     int
	ChangeVote     bool
	IsLocked       bool
	Options        []DisplayPollOption
	Lock           bool
	Edit           bool
	AllowedWarning string
	IsExpired      bool
	ExpireTime     string
	HasVoted       bool
	StarterID      int
	StarterName    string
	StarterHref    string
	StarterLink    string
	ShowResults    bool
}

// DisplayCtx is the page context for tpl_display.go.
type DisplayCtx struct {
	PreviousNext       string
	ShowSpellchecking  bool
	NumReplies         int
	TopicFirstMessage  int
	IsMarkedNotify     bool
	PageIndex          string
	Start              int
	StartFrom          int
	HasStartFrom       bool
	LinkModerators     []string
	IsLocked           bool
	IsSticky           bool
	IsVeryHot          bool
	IsHot              bool
	IsPoll             bool
	Class              string
	UserStarted        bool
	TopicStarterID     int
	Subject            string
	NumViews           int
	MarkUnreadTime     int
	Poll               *DisplayPoll
	AllowVote          bool
	AllowPollView      bool
	AllowChangeVote    bool
	FirstMessage       int
	FirstNewMessage    bool
	AllowHideEmail     bool
	CanSticky          bool
	CanMerge           bool
	CanSplit           bool
	CalendarPost       bool
	CanMarkNotify      bool
	CanSendTopic       bool
	CanSendPM          bool
	CanReportModerator bool
	CanModerateForum   bool
	CanMove            bool
	CanLock            bool
	CanDelete          bool
	CanAddPoll         bool
	CanRemovePoll      bool
	CanReply           bool
	CanRemovePost      bool
	Messages           []*DisplayMsg
}

// Display is Display() from Display.php.
func (c *Ctx) Display() {
	a := c.App
	scripturl := a.ScriptURL

	// What are you gonna display if these are empty?!
	if c.Topic == 0 {
		c.fatalLangError("smf232", false)
	}

	page := &DisplayCtx{}
	c.Page = page

	// Not only does a prefetch make things slower for the server, but it
	// makes it impossible to know if they read it.
	if c.R.Header.Get("X-Moz") == "prefetch" {
		c.W.WriteHeader(403)
		c.exit()
	}

	// Find the previous or next topic.  Make a fuss if there are no more.
	startParam := c.REQUEST.Str("start")
	if pn := c.REQUEST.Str("prev_next"); pn == "prev" || pn == "next" {
		if c.BoardInfo.NumTopics > 1 {
			gtLt, order := ">", ""
			if pn == "next" {
				gtLt, order = "<", " DESC"
			}
			var stickyCond, stickyOrder string
			if !a.SettingEmpty("enableStickyTopics") {
				stickyCond = `
					AND ((t2.ID_LAST_MSG ` + gtLt + ` t.ID_LAST_MSG AND t2.isSticky ` + gtLt + `= t.isSticky) OR t2.isSticky ` + gtLt + ` t.isSticky)`
				stickyOrder = " t2.isSticky" + order + ","
			} else {
				stickyCond = `
					AND t2.ID_LAST_MSG ` + gtLt + ` t.ID_LAST_MSG`
			}
			var newTopic int
			err := a.DB.QueryRow(a.Q(`
				SELECT t2.ID_TOPIC
				FROM {$db_prefix}topics AS t, {$db_prefix}topics AS t2
				WHERE t.ID_TOPIC = ` + itoa(c.Topic) + stickyCond + `
					AND t2.ID_BOARD = ` + itoa(c.Board) + `
				ORDER BY` + stickyOrder + ` t2.ID_LAST_MSG` + order + `
				LIMIT 1`)).Scan(&newTopic)
			if err != nil {
				// Roll over - if we're going prev, get the last - otherwise
				// the first.
				rollSticky := ""
				if !a.SettingEmpty("enableStickyTopics") {
					rollSticky = " isSticky" + order + ","
				}
				a.DB.QueryRow(a.Q(`
					SELECT ID_TOPIC
					FROM {$db_prefix}topics
					WHERE ID_BOARD = ` + itoa(c.Board) + `
					ORDER BY` + rollSticky + ` ID_LAST_MSG` + order + `
					LIMIT 1`)).Scan(&newTopic)
			}
			c.Topic = newTopic
		}

		// Go to the newest message on this topic.
		startParam = "new"

		// Duplicate link!  Tell the robots not to link this.
		c.RobotNoIndex = true
	}

	// Add 1 to the number of views of this topic.
	if int(c.Session.GetInt("last_read_topic")) != c.Topic {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET numViews = numViews + 1 WHERE ID_TOPIC = ?`), c.Topic)
		c.Session.Set("last_read_topic", c.Topic)
	}

	// Get all the important topic info.
	newFromCol := "0"
	ltJoin := ""
	if !c.User.IsGuest {
		newFromCol = "IFNULL(lt.ID_MSG, -1) + 1"
		ltJoin = `
			LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = ` + itoa(c.Topic) + ` AND lt.ID_MEMBER = ` + itoa(c.User.ID) + `)`
	}
	var numReplies, numViews, locked, isSticky, idPoll, idMemberStarted, idFirstMsg, idLastMsg, newFrom int
	var subject string
	err := a.DB.QueryRow(a.Q(`
		SELECT
			t.numReplies, t.numViews, t.locked, ms.subject, t.isSticky, t.ID_POLL,
			t.ID_MEMBER_STARTED, t.ID_FIRST_MSG, t.ID_LAST_MSG,
			`+newFromCol+` AS new_from
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS ms`+ltJoin+`
		WHERE t.ID_TOPIC = `+itoa(c.Topic)+`
			AND ms.ID_MSG = t.ID_FIRST_MSG
		LIMIT 1`)).Scan(&numReplies, &numViews, &locked, &subject, &isSticky, &idPoll,
		&idMemberStarted, &idFirstMsg, &idLastMsg, &newFrom)
	if err != nil {
		c.fatalLangError("472", false)
	}

	// The start isn't a number; it's information about what to do, where to
	// go.
	start := c.Start
	viewNewestFirst := !empty(c.Options["view_newest_first"])
	if !isAllDigits(startParam) {
		if startParam == "new" {
			// Guests automatically go to the last topic.
			if c.User.IsGuest {
				page.StartFrom = numReplies
				page.HasStartFrom = true
				if !viewNewestFirst {
					start = page.StartFrom
				} else {
					start = 0
				}
				startParam = itoa(start)
			} else {
				// Find the earliest unread message in the topic.
				var nf int
				a.DB.QueryRow(a.Q(`
					SELECT IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, -1)) + 1 AS new_from
					FROM {$db_prefix}topics AS t
						LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = ` + itoa(c.Topic) + ` AND lt.ID_MEMBER = ` + itoa(c.User.ID) + `)
						LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = ` + itoa(c.Board) + ` AND lmr.ID_MEMBER = ` + itoa(c.User.ID) + `)
					WHERE t.ID_TOPIC = ` + itoa(c.Topic) + `
					LIMIT 1`)).Scan(&nf)
				// Fall through to the next if statement.
				startParam = "msg" + itoa(nf)
			}
		}

		if strings.HasPrefix(startParam, "from") {
			// Start from a certain time index, not a message.
			timestamp := atoi(startParam[4:])
			if timestamp == 0 {
				start = 0
			} else {
				var cnt int
				a.DB.QueryRow(a.Q(`
					SELECT COUNT(*)
					FROM {$db_prefix}messages
					WHERE posterTime < ?
						AND ID_TOPIC = ?`), timestamp, c.Topic).Scan(&cnt)
				page.StartFrom = cnt
				page.HasStartFrom = true
				if !viewNewestFirst {
					start = page.StartFrom
				} else {
					start = numReplies - page.StartFrom
				}
			}
		} else if strings.HasPrefix(startParam, "msg") {
			// Link to a message...
			virtualMsg := atoi(startParam[3:])
			if virtualMsg >= idLastMsg {
				page.StartFrom = numReplies
			} else if virtualMsg <= idFirstMsg {
				page.StartFrom = 0
			} else {
				var cnt int
				a.DB.QueryRow(a.Q(`
					SELECT COUNT(*)
					FROM {$db_prefix}messages
					WHERE ID_MSG < ?
						AND ID_TOPIC = ?`), virtualMsg, c.Topic).Scan(&cnt)
				page.StartFrom = cnt
			}
			page.HasStartFrom = true

			// We need to reverse the start as well in this case.
			if !viewNewestFirst {
				start = page.StartFrom
			} else {
				start = numReplies - page.StartFrom
			}
			c.RobotNoIndex = true
		}
	}

	// Create a previous next string if the selected theme has it as a
	// selected option.
	if !a.SettingEmpty("enablePreviousNext") {
		page.PreviousNext = `<a href="` + scripturl + `?topic=` + itoa(c.Topic) + `.0;prev_next=prev#new">` + c.Txt("previous_next_back") + `</a> <a href="` + scripturl + `?topic=` + itoa(c.Topic) + `.0;prev_next=next#new">` + c.Txt("previous_next_forward") + `</a>`
	}

	// (pspell never exists here; quick-reply spellcheck stays off.)
	page.ShowSpellchecking = false

	// Censor the title...
	subject = c.censorText(subject)
	c.PageTitle = subject

	page.NumReplies = numReplies
	page.TopicFirstMessage = idFirstMsg

	// Is this topic sticky, or can it even be?
	if a.SettingEmpty("enableStickyTopics") {
		isSticky = 0
	}

	// Guests can't mark topics read or for notifications.
	if !c.User.IsGuest {
		flag := false
		if newFrom != 0 {
			res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_topics SET ID_MSG = ? WHERE ID_MEMBER = ? AND ID_TOPIC = ?`),
				a.SettingInt("maxMsgID"), c.User.ID, c.Topic)
			if res != nil {
				n, _ := res.RowsAffected()
				flag = n != 0
			}
		}
		if !flag {
			a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_topics (ID_MSG, ID_MEMBER, ID_TOPIC) VALUES (?, ?, ?)`),
				a.SettingInt("maxMsgID"), c.User.ID, c.Topic)
		}

		// Check for notifications on this topic OR board.
		nrows, err := a.DB.Query(a.Q(`
			SELECT sent, ID_TOPIC
			FROM {$db_prefix}log_notify
			WHERE (ID_TOPIC = ? OR ID_BOARD = ?)
				AND ID_MEMBER = ?
			LIMIT 2`), c.Topic, c.Board, c.User.ID)
		if err == nil {
			doOnce := true
			for nrows.Next() {
				var sent, idTopic int
				nrows.Scan(&sent, &idTopic)
				if idTopic != 0 {
					page.IsMarkedNotify = true
				}
				if sent != 0 && doOnce {
					a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_notify SET sent = 0 WHERE (ID_TOPIC = ? OR ID_BOARD = ?) AND ID_MEMBER = ?`),
						c.Topic, c.Board, c.User.ID)
					doOnce = false
				}
			}
			nrows.Close()
		}

		// Mark board as seen if we came using last post link from BoardIndex.
		if c.REQUEST.Has("boardseen") {
			a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_boards (ID_MSG, ID_MEMBER, ID_BOARD) VALUES (?, ?, ?)`),
				a.SettingInt("maxMsgID"), c.User.ID, c.Board)
		}
	}

	// If all is set, but not allowed... just unset it.
	hasAll := c.REQUEST.Has("all")
	if hasAll && a.SettingEmpty("enableAllMessages") {
		hasAll = false
	} else if hasAll {
		start = -1
	}

	maxMessages := a.SettingInt("defaultMaxMessages")

	// Construct the page index, allowing for the .START method...
	page.PageIndex, start = c.constructPageIndex(scripturl+"?topic="+itoa(c.Topic)+".%d", start, numReplies+1, maxMessages, true)
	page.Start = start
	c.Start = start

	// If they are viewing all the posts, show all the posts.
	if !a.SettingEmpty("enableAllMessages") && numReplies+1 > maxMessages && numReplies+1 < a.SettingInt("enableAllMessages") {
		if hasAll {
			// No limit! (actually, there is a limit, but...)
			maxMessages = -1
			if a.SettingEmpty("compactTopicPagesEnable") {
				page.PageIndex += "<b>" + c.Txt("190") + "</b> "
			} else {
				page.PageIndex += "[<b>" + c.Txt("190") + "</b>] "
			}
			start = 0
			page.Start = 0
			c.Start = 0
		} else {
			page.PageIndex += `&nbsp;<a href="` + scripturl + `?topic=` + itoa(c.Topic) + `.0;all">` + c.Txt("190") + `</a> `
		}
	}

	// Build the link tree.
	extraBefore := ""
	if !c.Theme.Empty("linktree_inline") {
		extraBefore = c.Txt("118") + ": "
	}
	c.LinkTree = append(c.LinkTree, Link{
		URL:         scripturl + "?topic=" + itoa(c.Topic) + ".0",
		Name:        subject,
		ExtraBefore: extraBefore,
	})

	// Build a list of this board's moderators.
	if len(c.BoardInfo.Moderators) > 0 {
		for _, mod := range c.BoardInfo.Moderators {
			page.LinkModerators = append(page.LinkModerators,
				`<a href="`+scripturl+"?action=profile;u="+itoa(mod.ID)+`" title="`+c.Txt("62")+`">`+mod.Name+`</a>`)
		}
		modTxt := c.Txt("299")
		if len(page.LinkModerators) == 1 {
			modTxt = c.Txt("298")
		}
		c.LinkTree[len(c.LinkTree)-2].ExtraAfter = ` (` + modTxt + `: ` + strings.Join(page.LinkModerators, ", ") + `)`
	}

	// Information about the current topic...
	page.IsLocked = locked != 0
	page.IsSticky = isSticky != 0
	page.IsVeryHot = numReplies >= a.SettingInt("hotTopicVeryPosts")
	page.IsHot = numReplies >= a.SettingInt("hotTopicPosts")

	// We don't want to show the poll icon in the topic class here.
	page.Class = determineTopicClass(page.IsVeryHot, page.IsHot, false, page.IsLocked, page.IsSticky)

	page.IsPoll = idPoll > 0 && a.Setting("pollMode") == "1" && c.allowedTo("poll_view")

	// Did this user start the topic or not?
	page.UserStarted = c.User.ID == idMemberStarted && !c.User.IsGuest
	page.TopicStarterID = idMemberStarted

	// Set the topic's information for the template.
	page.Subject = subject
	page.NumViews = numViews
	page.MarkUnreadTime = newFrom

	// Create the poll info if it exists.
	if page.IsPoll {
		c.loadDisplayPoll(page, idPoll)
	}

	// Calculate the fastest way to get the messages!
	ascending := !viewNewestFirst
	qStart := start
	limit := maxMessages
	firstIndex := 0
	if qStart > numReplies/2 && maxMessages != -1 {
		ascending = !ascending
		if numReplies < qStart+limit {
			limit = numReplies - qStart + 1
		}
		if numReplies < qStart+limit {
			qStart = 0
		} else {
			qStart = numReplies - qStart - limit + 1
		}
		firstIndex = limit - 1
	}

	// Get each post and poster in this topic.
	limitClause := ""
	if maxMessages != -1 {
		limitClause = `
		LIMIT ` + itoa(limit) + ` OFFSET ` + itoa(qStart)
	}
	dirSQL := ""
	if !ascending {
		dirSQL = "DESC"
	}
	mrows, err := a.DB.Query(a.Q(`
		SELECT ID_MSG, ID_MEMBER
		FROM {$db_prefix}messages
		WHERE ID_TOPIC = ` + itoa(c.Topic) + `
		ORDER BY ID_MSG ` + dirSQL + limitClause))
	if err != nil {
		c.fatalDBError(err)
	}
	var messages []int
	var posters []int
	for mrows.Next() {
		var idMsg, idMember int
		mrows.Scan(&idMsg, &idMember)
		if idMember != 0 {
			posters = append(posters, idMember)
		}
		messages = append(messages, idMsg)
	}
	mrows.Close()

	attachmentsByMsg := map[int][]attachmentRow{}

	if len(messages) > 0 {
		// Fetch attachments.
		if !a.SettingEmpty("attachmentEnable") && c.allowedTo("view_attachments") {
			attachmentsByMsg = c.fetchDisplayAttachments(messages)
		}

		// What?  It's not like it *couldn't* be only guests in this topic...
		if len(posters) > 0 {
			c.loadMemberData(posters)
		}

		// Go to the last message if the given time is beyond the time of the
		// last message.
		if page.HasStartFrom && page.StartFrom >= numReplies {
			page.StartFrom = numReplies
		}

		// Since the anchor information is needed on the top of the page we
		// load these variables beforehand.
		if firstIndex < len(messages) {
			page.FirstMessage = messages[firstIndex]
		} else {
			page.FirstMessage = messages[0]
		}
		if !viewNewestFirst {
			page.FirstNewMessage = page.HasStartFrom && start == page.StartFrom
		} else {
			page.FirstNewMessage = page.HasStartFrom && start == numReplies-page.StartFrom
		}
	}

	// Load the "Jump to" list...
	c.loadJumpTo()

	// Basic settings....
	page.AllowHideEmail = !a.SettingEmpty("allow_hideEmail") || (c.User.IsGuest && !a.SettingEmpty("guest_hideContacts"))

	// Now set all the wonderful, wonderful permissions...
	page.CanSticky = c.allowedTo("make_sticky")
	page.CanMerge = c.allowedTo("merge_any")
	page.CanSplit = c.allowedTo("split_any")
	page.CalendarPost = c.allowedTo("calendar_post")
	page.CanMarkNotify = c.allowedTo("mark_any_notify")
	page.CanSendTopic = c.allowedTo("send_topic")
	page.CanSendPM = c.allowedTo("pm_send")
	page.CanReportModerator = c.allowedTo("report_any")
	page.CanModerateForum = c.allowedTo("moderate_forum")

	anyOwn := func(perm string) bool {
		return c.allowedTo(perm+"_any") || (page.UserStarted && c.allowedTo(perm+"_own"))
	}
	page.CanMove = anyOwn("move")
	page.CanLock = anyOwn("lock")
	page.CanDelete = anyOwn("remove")
	page.CanAddPoll = anyOwn("poll_add")
	page.CanRemovePoll = anyOwn("poll_remove")
	page.CanReply = anyOwn("post_reply")

	// Cleanup all the permissions with extra stuff...
	page.CanMarkNotify = page.CanMarkNotify && !c.User.IsGuest
	page.CanSticky = page.CanSticky && !a.SettingEmpty("enableStickyTopics")
	page.CalendarPost = page.CalendarPost && !a.SettingEmpty("cal_enabled")
	page.CanAddPoll = page.CanAddPoll && a.Setting("pollMode") == "1" && idPoll <= 0
	page.CanRemovePoll = page.CanRemovePoll && a.Setting("pollMode") == "1" && idPoll > 0
	page.CanReply = page.CanReply && (locked == 0 || c.allowedTo("moderate_board"))

	boardCount := 0
	for _, cat := range c.JumpTo {
		boardCount += len(cat.Boards)
	}
	page.CanMove = page.CanMove && boardCount > 1

	// Start this off for quick moderation - it will be or'd for each post.
	page.CanRemovePost = c.allowedTo("delete_any") || (c.allowedTo("delete_replies") && page.UserStarted)

	// Prepare all the messages (prepareDisplayContext, materialized).
	c.prepareDisplayMessages(page, messages, newFrom, attachmentsByMsg, viewNewestFirst)

	c.SubTemplate = templateDisplayMain
}

type attachmentRow struct {
	ID          int
	IDMsg       int
	Filename    string
	FileHash    string
	Filesize    int
	Downloads   int
	Width       int
	Height      int
	ThumbID     int
	ThumbWidth  int
	ThumbHeight int
}

// fetchDisplayAttachments is the attachments query in Display().
func (c *Ctx) fetchDisplayAttachments(messages []int) map[int][]attachmentRow {
	a := c.App
	ids := make([]string, len(messages))
	for i, m := range messages {
		ids[i] = itoa(m)
	}
	withThumbs := !a.SettingEmpty("attachmentShowImages") && !a.SettingEmpty("attachmentThumbnails")
	thumbCols, thumbJoin := "", ""
	if withThumbs {
		thumbCols = `,
			IFNULL(thumb.ID_ATTACH, 0) AS ID_THUMB, IFNULL(thumb.width, 0) AS thumb_width, IFNULL(thumb.height, 0) AS thumb_height`
		thumbJoin = `
			LEFT JOIN {$db_prefix}attachments AS thumb ON (thumb.ID_ATTACH = a.ID_THUMB)`
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT
			a.ID_ATTACH, a.ID_MSG, a.filename, a.file_hash, IFNULL(a.size, 0) AS filesize, a.downloads,
			a.width, a.height` + thumbCols + `
		FROM {$db_prefix}attachments AS a` + thumbJoin + `
		WHERE a.ID_MSG IN (` + strings.Join(ids, ",") + `)
			AND a.attachmentType = 0
		ORDER BY a.ID_ATTACH`))
	out := map[int][]attachmentRow{}
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var r attachmentRow
		dest := []any{&r.ID, &r.IDMsg, &r.Filename, &r.FileHash, &r.Filesize, &r.Downloads, &r.Width, &r.Height}
		if withThumbs {
			dest = append(dest, &r.ThumbID, &r.ThumbWidth, &r.ThumbHeight)
		}
		rows.Scan(dest...)
		out[r.IDMsg] = append(out[r.IDMsg], r)
	}
	return out
}

// loadDisplayPoll fills page.Poll (the poll block of Display()).
func (c *Ctx) loadDisplayPoll(page *DisplayCtx, idPoll int) {
	a := c.App
	scripturl := a.ScriptURL

	var question, posterName string
	var votingLocked, hideResults, maxVotes, changeVote, idMember, total int
	var expireTime int64
	err := a.DB.QueryRow(a.Q(`
		SELECT
			p.question, p.votingLocked, p.hideResults, p.expireTime, p.maxVotes, p.changeVote,
			p.ID_MEMBER, IFNULL(mem.realName, p.posterName) AS posterName,
			COUNT(DISTINCT lp.ID_MEMBER) AS total
		FROM {$db_prefix}polls AS p
			LEFT JOIN {$db_prefix}log_polls AS lp ON (lp.ID_POLL = p.ID_POLL)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = p.ID_MEMBER)
		WHERE p.ID_POLL = ?
		GROUP BY p.ID_POLL
		LIMIT 1`), idPoll).Scan(&question, &votingLocked, &hideResults, &expireTime, &maxVotes,
		&changeVote, &idMember, &posterName, &total)
	if err != nil {
		return
	}

	// Get all the options, and calculate the total votes.
	rows, err := a.DB.Query(a.Q(`
		SELECT pc.ID_CHOICE, pc.label, pc.votes, IFNULL(lp.ID_CHOICE, -1) AS votedThis
		FROM {$db_prefix}poll_choices AS pc
			LEFT JOIN {$db_prefix}log_polls AS lp ON (lp.ID_CHOICE = pc.ID_CHOICE AND lp.ID_POLL = ? AND lp.ID_MEMBER = ?)
		WHERE pc.ID_POLL = ?`), idPoll, c.User.ID, idPoll)
	if err != nil {
		return
	}
	type pollOpt struct {
		id, votes, votedThis int
		label                string
	}
	var opts []pollOpt
	realTotal := 0
	hasVoted := false
	for rows.Next() {
		var o pollOpt
		rows.Scan(&o.id, &o.label, &o.votes, &o.votedThis)
		o.label = c.censorText(o.label)
		opts = append(opts, o)
		realTotal += o.votes
		hasVoted = hasVoted || o.votedThis != -1
	}
	rows.Close()

	poll := &DisplayPoll{
		ID:          idPoll,
		Question:    c.parseBBC(question, true),
		TotalVotes:  total,
		ChangeVote:  changeVote != 0,
		IsLocked:    votingLocked != 0,
		Lock:        c.allowedTo("poll_lock_any") || (page.UserStarted && c.allowedTo("poll_lock_own")),
		Edit:        c.allowedTo("poll_edit_any") || (page.UserStarted && c.allowedTo("poll_edit_own")),
		IsExpired:   expireTime != 0 && expireTime < time.Now().Unix(),
		HasVoted:    hasVoted,
		StarterID:   idMember,
		StarterName: posterName,
	}
	poll.Image = "normal_poll"
	if votingLocked != 0 {
		poll.Image = "normal_locked_poll"
	}
	if maxVotes > 1 {
		poll.AllowedWarning = phpSprintf(c.Txt("poll_options6"), itoa(maxVotes))
	}
	if expireTime != 0 {
		poll.ExpireTime = c.timeformat(expireTime)
	}
	if idMember != 0 {
		poll.StarterHref = scripturl + "?action=profile;u=" + itoa(idMember)
		poll.StarterLink = `<a href="` + poll.StarterHref + `">` + posterName + `</a>`
	} else {
		poll.StarterLink = posterName
	}

	page.AllowVote = !poll.IsExpired && !c.User.IsGuest && votingLocked == 0 && c.allowedTo("poll_vote") && !hasVoted
	page.AllowPollView = c.allowedTo("moderate_board") || hideResults == 0 || (hideResults == 1 && hasVoted) || poll.IsExpired
	poll.ShowResults = page.AllowPollView && c.REQUEST.Has("viewResults")
	page.AllowChangeVote = !poll.IsExpired && !c.User.IsGuest && votingLocked == 0 && c.allowedTo("poll_vote") && hasVoted && poll.ChangeVote

	// Calculate the percentages and bar lengths...
	divisor := realTotal
	if divisor == 0 {
		divisor = 1
	}
	// Determine if a decimal point is needed.
	precision := 1
	if realTotal == 100 {
		precision = 0
	}

	mult := 1.0
	for i := 0; i < precision; i++ {
		mult *= 10
	}
	for _, o := range opts {
		raw := float64(o.votes*100) / float64(divisor)
		bar := phpRound(raw, precision)
		// PHP: $barWide = $bar == 0 ? 1 : floor(($bar * 8) / 3), where $bar is the
		// rounded percent as a FLOAT (e.g. 54.5) — not its integer part.
		rounded := math.Floor(raw*mult+0.5) / mult
		barWide := 1
		if rounded != 0 {
			barWide = int(math.Floor(rounded * 8 / 3))
		}
		voteType := "radio"
		if maxVotes > 1 {
			voteType = "checkbox"
		}
		poll.Options = append(poll.Options, DisplayPollOption{
			ID:         "options-" + itoa(o.id),
			Percent:    bar,
			Votes:      o.votes,
			VotedThis:  o.votedThis != -1,
			Bar:        `<span style="white-space: nowrap;"><img src="` + c.Theme.ImagesURL() + `/poll_left.gif" alt="" /><img src="` + c.Theme.ImagesURL() + `/poll_middle.gif" width="` + itoa(barWide) + `" height="12" alt="-" /><img src="` + c.Theme.ImagesURL() + `/poll_right.gif" alt="" /></span>`,
			BarWidth:   barWide,
			Option:     c.parseBBC(o.label, true),
			VoteButton: `<input type="` + voteType + `" name="options[]" id="options-` + itoa(o.id) + `" value="` + itoa(o.id) + `" class="check" />`,
		})
	}

	page.Poll = poll
}

// phpRound formats round($v, $precision) the way PHP prints it (trailing
// zeros dropped).
func phpRound(v float64, precision int) string {
	// Format from the scaled integer, not from a re-derived float: r = 182/10
	// lands on 18.19999… in float64, and decimal formatting then truncates it
	// back to "18.1" instead of "18.2". PHP's round() echoes as a float, so
	// trailing zeros are dropped (18.20 -> "18.2", 50.0 -> "50").
	mult := int64(1)
	for i := 0; i < precision; i++ {
		mult *= 10
	}
	neg := v < 0
	if neg {
		v = -v
	}
	n := int64(v*float64(mult) + 0.5) // PHP rounds halves away from zero
	intPart := n / mult
	s := itoa(int(intPart))
	if precision > 0 {
		frac := strings.TrimRight(zeroPad(int(n%mult), precision), "0")
		if frac != "" {
			s += "." + frac
		}
	}
	if neg {
		s = "-" + s
	}
	return s
}

func floatToStr(v float64, precision int) string {
	// strconv-free tiny formatter for the poll percentages.
	neg := v < 0
	if neg {
		v = -v
	}
	intPart := int64(v)
	frac := v - float64(intPart)
	out := itoa(int(intPart))
	if precision > 0 {
		fracDigits := ""
		for i := 0; i < precision; i++ {
			frac *= 10
			d := int(frac)
			fracDigits += itoa(d)
			frac -= float64(d)
		}
		out += "." + fracDigits
	}
	if neg {
		out = "-" + out
	}
	return out
}

// prepareDisplayMessages materializes prepareDisplayContext() over all rows.
func (c *Ctx) prepareDisplayMessages(page *DisplayCtx, messages []int, newFrom int, attachmentsByMsg map[int][]attachmentRow, viewNewestFirst bool) {
	a := c.App
	scripturl := a.ScriptURL

	if len(messages) == 0 {
		return
	}

	ids := make([]string, len(messages))
	for i, m := range messages {
		ids[i] = itoa(m)
	}
	order := ""
	if viewNewestFirst {
		order = " DESC"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT
			ID_MSG, icon, subject, posterTime, posterIP, ID_MEMBER, modifiedTime, modifiedName, body,
			smileysEnabled, posterName, posterEmail,
			ID_MSG_MODIFIED < ` + itoa(newFrom) + ` AS isRead
		FROM {$db_prefix}messages
		WHERE ID_MSG IN (` + strings.Join(ids, ",") + `)
		ORDER BY ID_MSG` + order))
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()

	// Remember which message this is.  (ie. reply #83)
	counter := page.Start
	if viewNewestFirst {
		counter = page.NumReplies - page.Start
	}

	iconSources := map[string]string{}
	for _, icon := range []string{"xx", "thumbup", "thumbdown", "exclamation", "question", "lamp",
		"smiley", "angry", "cheesy", "grin", "sad", "wink", "moved", "recycled", "wireless"} {
		iconSources[icon] = "images_url"
	}

	for rows.Next() {
		var idMsg, idMember, smileysEnabled, isRead int
		var posterTime, modifiedTime int64
		var icon, subject, posterIP, modifiedName, body, posterName, posterEmail string
		if err := rows.Scan(&idMsg, &icon, &subject, &posterTime, &posterIP, &idMember,
			&modifiedTime, &modifiedName, &body, &smileysEnabled, &posterName, &posterEmail, &isRead); err != nil {
			c.fatalDBError(err)
		}

		// Message Icon Management... check the images exist.
		if _, ok := iconSources[icon]; !ok {
			if a.SettingEmpty("messageIconChecks_disable") {
				if _, err := os.Stat(c.Theme.Get("theme_dir") + "/images/post/" + icon + ".gif"); err == nil {
					iconSources[icon] = "images_url"
				} else {
					iconSources[icon] = "default_images_url"
				}
			} else {
				iconSources[icon] = "images_url"
			}
		}

		// If you're a lazy bum, you probably didn't give a subject...
		if subject == "" {
			subject = c.Txt("24")
		}

		// Are you allowed to remove at least a single reply?
		page.CanRemovePost = page.CanRemovePost || (c.allowedTo("delete_own") &&
			(a.SettingEmpty("edit_disable_time") || posterTime+int64(a.SettingInt("edit_disable_time"))*60 >= time.Now().Unix()) &&
			idMember == c.User.ID)

		// If it couldn't load, or the user was a guest....
		var member *MemberCtx
		if !c.loadMemberContext(idMember) {
			member = &MemberCtx{
				Name:      posterName,
				ID:        0,
				Group:     c.Txt("28"),
				Link:      posterName,
				Email:     posterEmail,
				HideEmail: posterEmail == "" || (!a.SettingEmpty("guest_hideContacts") && c.User.IsGuest),
				IsGuest:   true,
				Options:   map[string]string{},
			}
		} else {
			member = c.memberCtx[idMember]
			member.CanViewProfile = c.allowedTo("profile_view_any") || (idMember == c.User.ID && c.allowedTo("profile_view_own"))
			member.IsTopicStarter = idMember == page.TopicStarterID
		}
		member.IP = posterIP

		// Do the censor thang.
		body = c.censorText(body)
		subject = c.censorText(subject)

		// Run BBC interpreter on the message.
		body = c.parseBBCCached(body, smileysEnabled != 0, itoa(idMsg))

		msg := &DisplayMsg{
			Attachments:  c.loadAttachmentContext(idMsg, attachmentsByMsg),
			Alternate:    counter % 2,
			ID:           idMsg,
			Href:         scripturl + "?topic=" + itoa(c.Topic) + ".msg" + itoa(idMsg) + "#msg" + itoa(idMsg),
			Link:         `<a href="` + scripturl + "?topic=" + itoa(c.Topic) + ".msg" + itoa(idMsg) + "#msg" + itoa(idMsg) + `">` + subject + `</a>`,
			Member:       member,
			Icon:         icon,
			IconURL:      c.Theme.Get(iconSources[icon]) + "/post/" + icon + ".gif",
			Subject:      subject,
			Time:         c.timeformat(posterTime),
			Timestamp:    c.forumTime(true, posterTime),
			Counter:      counter,
			ModifiedTime: c.timeformat(modifiedTime),
			ModifiedName: modifiedName,
			Body:         body,
			New:          isRead == 0,
			FirstNew:     page.HasStartFrom && page.StartFrom == counter,
			CanModify: (!page.IsLocked || c.allowedTo("moderate_board")) &&
				(c.allowedTo("modify_any") || (c.allowedTo("modify_replies") && page.UserStarted) ||
					(c.allowedTo("modify_own") && idMember == c.User.ID &&
						(a.SettingEmpty("edit_disable_time") || posterTime+int64(a.SettingInt("edit_disable_time"))*60 > time.Now().Unix()))),
			CanRemove: c.allowedTo("delete_any") || (c.allowedTo("delete_replies") && page.UserStarted) ||
				(c.allowedTo("delete_own") && idMember == c.User.ID &&
					(a.SettingEmpty("edit_disable_time") || posterTime+int64(a.SettingInt("edit_disable_time"))*60 > time.Now().Unix())),
			CanSeeIP: c.allowedTo("moderate_forum") || (idMember == c.User.ID && c.User.ID != 0),
		}

		if !viewNewestFirst {
			counter++
		} else {
			counter--
		}

		page.Messages = append(page.Messages, msg)
	}
}

// loadAttachmentContext is loadAttachmentContext($ID_MSG) (thumbnail
// *creation* deferred to Phase 3; existing thumbs are used).
func (c *Ctx) loadAttachmentContext(idMsg int, attachmentsByMsg map[int][]attachmentRow) []DisplayAttachment {
	a := c.App
	scripturl := a.ScriptURL

	var out []DisplayAttachment
	if a.SettingEmpty("attachmentEnable") {
		return out
	}
	for _, att := range attachmentsByMsg[idMsg] {
		d := DisplayAttachment{
			ID:        att.ID,
			Name:      Htmlspecialchars(att.Filename),
			Downloads: att.Downloads,
			Size:      phpRound(float64(att.Filesize)/1024, 2) + " " + c.Txt("smf211"),
			ByteSize:  att.Filesize,
			Href:      scripturl + "?action=dlattach;topic=" + itoa(c.Topic) + ".0;attach=" + itoa(att.ID),
			IsImage:   att.Width != 0 && att.Height != 0 && !a.SettingEmpty("attachmentShowImages"),
		}
		d.Link = `<a href="` + d.Href + `">` + Htmlspecialchars(att.Filename) + `</a>`

		if !d.IsImage {
			out = append(out, d)
			continue
		}

		d.RealWidth, d.Width = att.Width, att.Width
		d.RealHeight, d.Height = att.Height, att.Height

		if att.ThumbID != 0 {
			d.Width = att.ThumbWidth
			d.Height = att.ThumbHeight
			d.HasThumb = true
			d.ThumbID = att.ThumbID
			d.ThumbHref = scripturl + "?action=dlattach;topic=" + itoa(c.Topic) + ".0;attach=" + itoa(att.ThumbID) + ";image"
		}

		if !d.HasThumb && ((!a.SettingEmpty("max_image_width") && att.Width > a.SettingInt("max_image_width")) ||
			(!a.SettingEmpty("max_image_height") && att.Height > a.SettingInt("max_image_height"))) {
			maxW, maxH := a.SettingInt("max_image_width"), a.SettingInt("max_image_height")
			if maxW != 0 && (maxH == 0 || att.Height*maxW/att.Width <= maxH) {
				d.Width = maxW
				d.Height = att.Height * maxW / att.Width
			} else if maxW != 0 {
				d.Width = att.Width * maxH / att.Height
				d.Height = maxH
			}
		} else if d.HasThumb {
			if (!a.SettingEmpty("max_image_width") && d.RealWidth > a.SettingInt("max_image_width")) ||
				(!a.SettingEmpty("max_image_height") && d.RealHeight > a.SettingInt("max_image_height")) {
				d.ThumbJS = "return reqWin('" + d.Href + ";image', " + itoa(att.Width+20) + ", " + itoa(att.Height+20) + ", true);"
			} else {
				d.ThumbJS = "return expandThumb(" + itoa(att.ID) + ");"
			}
		}

		if !d.HasThumb {
			d.Downloads++
		}
		out = append(out, d)
	}
	return out
}
