package app

// Port of Sources/SplitTopics.php: splitting a topic into two (SplitTopics +
// sub-actions, splitTopic) and merging topics into one (MergeTopics +
// sub-actions). Templates live in tpl_splittopics.go; the XML "split"
// sub-template is here too (Xml.template.php template_split).

import "strings"

func init() {
	registerAction("splittopics", (*Ctx).SplitTopics)
	registerAction("mergetopics", (*Ctx).MergeTopics)
}

// ---- Split context structs ----

// SplitMessage is one rendered message row in the selective splitter.
type SplitMessage struct {
	ID      int
	Subject string
	Body    string
	Poster  string
}

// SplitSection is the not_selected / selected column.
type SplitSection struct {
	NumMessages int
	Start       int
	PageIndex   string
	Messages    []SplitMessage
}

// SplitChange is one XML diff entry for the AJAX selector.
type SplitChange struct {
	ID      int
	Type    string // "remove" | "insert"
	Section string // "not_selected" | "selected"
	Insert  SplitMessage
}

// SplitAskCtx backs template_ask.
type SplitAskCtx struct {
	MessageID      int
	MessageSubject string
}

// SplitMainCtx backs template_main (the all-done page).
type SplitMainCtx struct {
	OldTopic int
	NewTopic int
}

// SplitSelectCtx backs template_select and template_split (XML).
type SplitSelectCtx struct {
	TopicID      int
	TopicSubject string // urlencoded subname, for links
	NewSubject   string
	NotSelected  SplitSection
	Selected     SplitSection
	Changes      []SplitChange
}

// SplitTopics is SplitTopics(): the ?action=splittopics dispatcher.
func (c *Ctx) SplitTopics() {
	// And... which topic were you splitting, again?
	if c.Topic == 0 {
		c.fatalLangError("337", false)
	}

	// Are you allowed to split topics?
	c.isAllowedTo("split_any")

	switch c.REQUEST.Str("sa") {
	case "selectTopics":
		c.SplitSelectTopics()
	case "execute":
		c.SplitExecute()
	case "splitSelection":
		c.SplitSelectionExecute()
	case "index":
		c.SplitIndex()
	default:
		c.SplitIndex()
	}
}

// SplitIndex is SplitIndex(): ask how to split.
func (c *Ctx) SplitIndex() {
	a := c.App

	// Validate "at".
	at := c.GET.Int("at")
	if at == 0 {
		c.fatalLangError("337", false)
	}

	var subname string
	var numReplies, idFirstMsg int
	err := a.DB.QueryRow(a.Q(`
		SELECT m.subject, t.numReplies, t.ID_FIRST_MSG
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE m.ID_MSG = ?
			AND m.ID_TOPIC = ?
			AND t.ID_TOPIC = ?
		LIMIT 1`), at, c.Topic, c.Topic).Scan(&subname, &numReplies, &idFirstMsg)
	if err != nil {
		c.fatalLangError("smf272", true)
	}
	c.REQUEST.Set("subname", subname)

	// There has to be more than one message in the topic.
	if numReplies < 1 {
		c.fatalLangError("smf270", false)
	}

	// First message in the topic? Then jump straight to the selector.
	if idFirstMsg == at {
		c.SplitSelectTopics()
		return
	}

	page := &SplitAskCtx{MessageID: at, MessageSubject: subname}
	c.Page = page
	c.SubTemplate = templateSplitAsk
	c.PageTitle = c.Txt("smf251")
}

// SplitExecute is SplitExecute(): perform an "only this"/"after this" split.
func (c *Ctx) SplitExecute() {
	a := c.App

	// They blanked the subject name.
	subname := c.POST.Str("subname")
	if !c.POST.Has("subname") || subname == "" {
		subname = c.Txt("smf258")
		c.POST.Set("subname", subname)
	}

	// Redirect to the selector if they chose selective.
	if c.POST.Str("step2") == "selective" {
		c.REQUEST.Set("subname", subname)
		c.SplitSelectTopics()
		return
	}

	c.checkSession("post", "", true)

	at := c.POST.Int("at")
	var messagesToBeSplit []int

	switch c.POST.Str("step2") {
	case "afterthis":
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MSG
			FROM {$db_prefix}messages
			WHERE ID_TOPIC = ?
				AND ID_MSG >= ?`), c.Topic, at)
		if err == nil {
			for rows.Next() {
				var m int
				rows.Scan(&m)
				messagesToBeSplit = append(messagesToBeSplit, m)
			}
			rows.Close()
		}
	case "onlythis":
		messagesToBeSplit = append(messagesToBeSplit, at)
	default:
		c.fatalLangError("1", false)
	}

	page := &SplitMainCtx{OldTopic: c.Topic}
	c.Page = page
	page.NewTopic = c.splitTopic(c.Topic, messagesToBeSplit, subname)
	c.SubTemplate = templateSplitMain
	c.PageTitle = c.Txt("smf251")
}

// SplitSelectTopics is SplitSelectTopics(): the selective message picker
// (HTML form or XML diff for the AJAX flow).
func (c *Ctx) SplitSelectTopics() {
	a := c.App
	scripturl := a.ScriptURL
	maxMessages := a.SettingInt("defaultMaxMessages")
	if maxMessages == 0 {
		maxMessages = 15
	}

	c.PageTitle = c.Txt("smf251") + " - " + c.Txt("smf257")

	selection := c.splitSelection(c.Topic)

	page := &SplitSelectCtx{
		TopicID:      c.Topic,
		TopicSubject: urlencode(c.REQUEST.Str("subname")),
		NewSubject:   c.REQUEST.Str("subname"),
	}
	c.Page = page
	page.NotSelected.Start = c.REQUEST.Int("start")
	page.Selected.Start = c.REQUEST.Int("start2")

	isXML := c.REQUEST.Has("xml")
	if isXML {
		c.SubTemplate = templateSplitXML
	} else {
		c.SubTemplate = templateSplitSelect
	}

	// Snapshot the message IDs from before the move (for the XML diff).
	var originalNotSelected, originalSelected []int
	if isXML {
		notSelCond := ""
		if len(selection) > 0 {
			notSelCond = "\n\t\t\t\tAND ID_MSG NOT IN (" + joinInts(selection) + ")"
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MSG
			FROM {$db_prefix}messages
			WHERE ID_TOPIC = ?`+notSelCond+`
			ORDER BY ID_MSG DESC
			LIMIT ? OFFSET ?`), c.Topic, maxMessages, page.NotSelected.Start)
		n := 0
		if err == nil {
			for rows.Next() {
				var m int
				rows.Scan(&m)
				originalNotSelected = append(originalNotSelected, m)
				n++
			}
			rows.Close()
		}
		// You can't split the last message off.
		if page.NotSelected.Start == 0 && n <= 1 && c.REQUEST.Str("move") == "down" {
			c.REQUEST.Set("move", "")
		}
		if len(selection) > 0 {
			srows, err := a.DB.Query(a.Q(`
				SELECT ID_MSG
				FROM {$db_prefix}messages
				WHERE ID_TOPIC = ?
					AND ID_MSG IN (`+joinInts(selection)+`)
				ORDER BY ID_MSG DESC
				LIMIT ? OFFSET ?`), c.Topic, maxMessages, page.Selected.Start)
			if err == nil {
				for srows.Next() {
					var m int
					srows.Scan(&m)
					originalSelected = append(originalSelected, m)
				}
				srows.Close()
			}
		}
	}

	// (De)select a message.
	if move := c.REQUEST.Str("move"); move != "" {
		msg := c.REQUEST.Int("msg")
		switch move {
		case "reset":
			selection = nil
		case "up":
			selection = removeInt(selection, msg)
		default:
			selection = append(selection, msg)
		}
		c.setSplitSelection(c.Topic, selection)
	}

	// Make sure the selection is still accurate.
	if len(selection) > 0 {
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MSG
			FROM {$db_prefix}messages
			WHERE ID_TOPIC = ?
				AND ID_MSG IN (`+joinInts(selection)+`)`), c.Topic)
		var fresh []int
		if err == nil {
			for rows.Next() {
				var m int
				rows.Scan(&m)
				fresh = append(fresh, m)
			}
			rows.Close()
		}
		selection = fresh
		c.setSplitSelection(c.Topic, selection)
	}

	// Number of messages (not) selected to be split.
	selExpr := "0"
	if len(selection) > 0 {
		selExpr = "m.ID_MSG IN (" + joinInts(selection) + ")"
	}
	crows, err := a.DB.Query(a.Q(`
		SELECT `+selExpr+` AS is_selected, COUNT(*) AS num_messages
		FROM {$db_prefix}messages AS m
		WHERE m.ID_TOPIC = ?
		GROUP BY is_selected`), c.Topic)
	if err == nil {
		for crows.Next() {
			var isSelected, num int
			crows.Scan(&isSelected, &num)
			if isSelected == 0 {
				page.NotSelected.NumMessages = num
			} else {
				page.Selected.NumMessages = num
			}
		}
		crows.Close()
	}

	// Fix an oversized starting page.
	if page.Selected.Start >= page.Selected.NumMessages {
		if page.Selected.NumMessages <= maxMessages {
			page.Selected.Start = 0
		} else {
			rem := page.Selected.NumMessages % maxMessages
			if rem == 0 {
				rem = maxMessages
			}
			page.Selected.Start = page.Selected.NumMessages - rem
		}
	}

	subEsc := strings.ReplaceAll(urlencode(c.REQUEST.Str("subname")), "%", "%%")
	page.NotSelected.PageIndex, _ = c.constructPageIndex(
		scripturl+"?action=splittopics;sa=selectTopics;subname="+subEsc+";topic="+itoa(c.Topic)+".%d;start2="+itoa(page.Selected.Start),
		page.NotSelected.Start, page.NotSelected.NumMessages, maxMessages, true)
	page.Selected.PageIndex, _ = c.constructPageIndex(
		scripturl+"?action=splittopics;sa=selectTopics;subname="+subEsc+";topic="+itoa(c.Topic)+"."+itoa(page.NotSelected.Start)+";start2=%d",
		page.Selected.Start, page.Selected.NumMessages, maxMessages, true)

	// The not-selected messages.
	notSelCond := ""
	if len(selection) > 0 {
		notSelCond = "\n\t\t\tAND ID_MSG NOT IN (" + joinInts(selection) + ")"
	}
	page.NotSelected.Messages = c.splitFetchMessages(notSelCond, c.Topic, maxMessages, page.NotSelected.Start)

	// The selected messages.
	if len(selection) > 0 {
		page.Selected.Messages = c.splitFetchMessages("\n\t\t\tAND ID_MSG IN ("+joinInts(selection)+")", c.Topic, maxMessages, page.Selected.Start)
	}

	// The XML method only needs what changed: compare snapshots to results.
	if isXML {
		nowNot := messageIDs(page.NotSelected.Messages)
		nowSel := messageIDs(page.Selected.Messages)
		add := func(typ, section string, ids []int, msgs []SplitMessage) {
			for _, id := range ids {
				ch := SplitChange{ID: id, Type: typ, Section: section}
				if typ == "insert" {
					ch.Insert = findSplitMessage(msgs, id)
				}
				page.Changes = append(page.Changes, ch)
			}
		}
		// remove: in original, not in current. insert: in current, not original.
		add("remove", "not_selected", diffInts(originalNotSelected, nowNot), nil)
		add("remove", "selected", diffInts(originalSelected, nowSel), nil)
		add("insert", "not_selected", diffInts(nowNot, originalNotSelected), page.NotSelected.Messages)
		add("insert", "selected", diffInts(nowSel, originalSelected), page.Selected.Messages)
	}
}

// splitFetchMessages loads and renders messages for one selector column.
func (c *Ctx) splitFetchMessages(extraCond string, topic, limit, offset int) []SplitMessage {
	a := c.App
	rows, err := a.DB.Query(a.Q(`
		SELECT m.subject, IFNULL(mem.realName, m.posterName) AS realName, m.body, m.ID_MSG, m.smileysEnabled
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_TOPIC = ?`+extraCond+`
		ORDER BY m.ID_MSG DESC
		LIMIT ? OFFSET ?`), topic, limit, offset)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []SplitMessage
	for rows.Next() {
		var subject, realName, body string
		var idMsg, smileys int
		rows.Scan(&subject, &realName, &body, &idMsg, &smileys)
		subject = c.censorText(subject)
		body = c.censorText(body)
		body = c.parseBBCCached(body, smileys != 0, itoa(idMsg))
		out = append(out, SplitMessage{ID: idMsg, Subject: subject, Body: body, Poster: realName})
	}
	return out
}

// SplitSelectionExecute is SplitSelectionExecute(): split off the selected set.
func (c *Ctx) SplitSelectionExecute() {
	c.checkSession("post", "", true)

	subname := c.POST.Str("subname")
	if !c.POST.Has("subname") || subname == "" {
		subname = c.Txt("smf258")
	}

	selection := c.splitSelection(c.Topic)
	if len(selection) == 0 {
		c.fatalLangError("smf271", false)
	}

	page := &SplitMainCtx{OldTopic: c.Topic}
	c.Page = page
	page.NewTopic = c.splitTopic(c.Topic, selection, subname)
	c.SubTemplate = templateSplitMain
	c.PageTitle = c.Txt("smf251")
}

// splitTopic is splitTopic(): splits the given messages off into a new topic
// and returns its ID.
func (c *Ctx) splitTopic(split1Topic int, splitMessages []int, newSubject string) int {
	a := c.App

	if len(splitMessages) == 0 {
		c.fatalLangError("smf271", false)
	}
	postList := joinInts(splitMessages)

	idBoard := c.Board
	if split1Topic != c.Topic {
		a.DB.QueryRow(a.Q(`SELECT ID_BOARD FROM {$db_prefix}topics WHERE ID_TOPIC = ? LIMIT 1`), split1Topic).Scan(&idBoard)
	}

	// Find the new first and last NOT in the list (old topic).
	var split1First, split1Last, split1Replies, isSticky int
	err := a.DB.QueryRow(a.Q(`
		SELECT MIN(m.ID_MSG) AS myID_FIRST_MSG, MAX(m.ID_MSG) AS myID_LAST_MSG, COUNT(*) - 1 AS myNumReplies, t.isSticky
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE m.ID_MSG NOT IN (`+postList+`)
			AND m.ID_TOPIC = ?
			AND t.ID_TOPIC = ?
		GROUP BY m.ID_TOPIC
		LIMIT 1`), split1Topic, split1Topic).Scan(&split1First, &split1Last, &split1Replies, &isSticky)
	if err != nil {
		// You can't select ALL the messages!
		c.fatalLangError("smf271b", false)
	}
	split1FirstMem := c.getMsgMemberID(split1First)
	split1LastMem := c.getMsgMemberID(split1Last)

	// Find the first and last IN the list (new topic).
	var split2First, split2Last, split2Replies int
	a.DB.QueryRow(a.Q(`
		SELECT MIN(ID_MSG) AS myID_FIRST_MSG, MAX(ID_MSG) AS myID_LAST_MSG, COUNT(*) - 1 AS myNumReplies
		FROM {$db_prefix}messages
		WHERE ID_MSG IN (`+postList+`)
			AND ID_TOPIC = ?
		GROUP BY ID_TOPIC
		LIMIT 1`), split1Topic).Scan(&split2First, &split2Last, &split2Replies)
	split2FirstMem := c.getMsgMemberID(split2First)
	split2LastMem := c.getMsgMemberID(split2Last)

	// Sanity.
	if split1First <= 0 || split1Last <= 0 || split2First <= 0 || split2Last <= 0 || split1Replies < 0 || split2Replies < 0 {
		c.fatalLangError("smf272", true)
	}

	// You cannot split off the first message of a topic.
	if split1First > split2First {
		c.fatalLangError("smf268", false)
	}

	// Insert the new topic (0/0 for first/last to avoid UNIQUE errors).
	res, err := a.DB.Exec(a.Q(`
		INSERT INTO {$db_prefix}topics
			(ID_BOARD, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_FIRST_MSG, ID_LAST_MSG, numReplies, isSticky)
		VALUES (?, ?, ?, 0, 0, ?, ?)`), idBoard, split2FirstMem, split2LastMem, split2Replies, isSticky)
	if err != nil {
		c.fatalLangError("smf273", true)
	}
	id64, _ := res.LastInsertId()
	split2Topic := int(id64)
	if split2Topic <= 0 {
		c.fatalLangError("smf273", true)
	}

	// Move the messages over to the new topic.
	newSubject = Htmlspecialchars(newSubject)
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}messages
		SET ID_TOPIC = ?, subject = ?
		WHERE ID_MSG IN (`+postList+`)`), split2Topic, newSubject)

	// Cache the new topic's subject.
	c.updateStatsSubject(split2Topic, newSubject)

	// Fix up the old topic's first/last/replies.
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}topics
		SET numReplies = ?, ID_FIRST_MSG = ?, ID_LAST_MSG = ?, ID_MEMBER_STARTED = ?, ID_MEMBER_UPDATED = ?
		WHERE ID_TOPIC = ?`), split1Replies, split1First, split1Last, split1FirstMem, split1LastMem, split1Topic)

	// Put the new topic's first/last back to what they should be.
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}topics
		SET ID_FIRST_MSG = ?, ID_LAST_MSG = ?
		WHERE ID_TOPIC = ?`), split2First, split2Last, split2Topic)

	// The board has more topics now.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numTopics = numTopics + 1 WHERE ID_BOARD = ?`), idBoard)

	// Assume they read it before splitting it.
	if !c.User.IsGuest {
		a.DB.Exec(a.Q(`
			REPLACE INTO {$db_prefix}log_topics
				(ID_MSG, ID_MEMBER, ID_TOPIC)
			VALUES (?, ?, ?)`), a.SettingInt("maxMsgID"), c.User.ID, split2Topic)
	}

	// Housekeeping.
	a.updateStatsTopic()
	c.updateLastMessages([]int{idBoard}, 0)

	c.logAction("split", map[string]any{"topic": split1Topic, "new_topic": split2Topic})

	c.sendNotifications(split1Topic, "split")

	return split2Topic
}

// ---- Merge context structs ----

// MergeBoard is one board option in the merge UI.
type MergeBoard struct {
	ID       int
	Name     string
	Category string
	Selected bool
}

// MergeListTopic is one topic in MergeIndex's candidate list.
type MergeListTopic struct {
	ID         int
	Subject    string
	PosterLink string
}

// MergePerson is a started/updated descriptor in the extra-options table.
type MergePerson struct {
	Time string
	Link string
}

// MergeTopicInfo is one topic's data in the extra-options table.
type MergeTopicInfo struct {
	ID       int
	Subject  string
	Started  MergePerson
	Updated  MergePerson
	Selected bool
}

// MergePoll is one selectable poll in the extra-options table.
type MergePoll struct {
	ID           int
	TopicID      int
	TopicSubject string
	Question     string
	Selected     bool
}

// MergeIndexCtx backs template_merge.
type MergeIndexCtx struct {
	OriginTopic   int
	OriginSubject string
	TargetBoard   int
	PageIndex     string
	Boards        []MergeBoard
	Topics        []MergeListTopic
}

// MergeExtraCtx backs template_merge_extra_options.
type MergeExtraCtx struct {
	Topics []MergeTopicInfo
	Boards []MergeBoard
	Polls  []MergePoll
}

// MergeDoneCtx backs template_merge_done.
type MergeDoneCtx struct {
	TargetBoard int
	TargetTopic int
}

// MergeTopics is MergeTopics(): the ?action=mergetopics dispatcher.
func (c *Ctx) MergeTopics() {
	switch c.REQUEST.Str("sa") {
	case "done":
		c.MergeDone()
	case "execute":
		c.MergeExecute(nil)
	case "options":
		c.MergeExecute(nil)
	case "index":
		c.MergeIndex()
	default:
		c.MergeIndex()
	}
}

// MergeIndex is MergeIndex(): pick a topic to merge with.
func (c *Ctx) MergeIndex() {
	a := c.App
	scripturl := a.ScriptURL
	maxTopics := a.SettingInt("defaultMaxTopics")
	if maxTopics == 0 {
		maxTopics = 20
	}

	targetBoard := c.Board
	if c.REQUEST.Has("targetboard") {
		targetBoard = c.REQUEST.Int("targetboard")
	}

	if !c.GET.Has("from") {
		c.fatalLangError("1", true)
	}
	from := c.GET.Int("from")

	page := &MergeIndexCtx{OriginTopic: from, TargetBoard: targetBoard}
	c.Page = page

	// How many topics are on this board? (for paging)
	var topicCount int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}topics WHERE ID_BOARD = ?`), targetBoard).Scan(&topicCount)

	page.PageIndex, _ = c.constructPageIndex(
		scripturl+"?action=mergetopics;from="+itoa(from)+";targetboard="+itoa(targetBoard)+";board="+itoa(c.Board)+".%d",
		c.REQUEST.Int("start"), topicCount, maxTopics, true)

	// Get the origin topic's subject.
	var subject string
	err := a.DB.QueryRow(a.Q(`
		SELECT m.subject
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE m.ID_MSG = t.ID_FIRST_MSG
			AND t.ID_TOPIC = ?
			AND t.ID_BOARD = ?
		LIMIT 1`), from, c.Board).Scan(&subject)
	if err != nil {
		c.fatalLangError("smf232", false)
	}
	page.OriginSubject = subject
	c.PageTitle = c.Txt("smf252")

	// Boards the user has merge permission on.
	mergeBoards := c.boardsAllowedTo("merge_any")
	if len(mergeBoards) == 0 {
		c.fatalLangError("cannot_merge_any", true)
	}

	mergeCond := ""
	if !inInts(mergeBoards, 0) {
		mergeCond = "\n\t\t\t\tAND b.ID_BOARD IN (" + joinInts(mergeBoards) + ")"
	}
	brows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.name AS bName, c.name AS cName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE ` + c.User.QuerySeeBoard + mergeCond))
	if err == nil {
		for brows.Next() {
			var b MergeBoard
			brows.Scan(&b.ID, &b.Name, &b.Category)
			page.Boards = append(page.Boards, b)
		}
		brows.Close()
	}

	// Candidate topics to merge with.
	stickyOrder := ""
	if !a.SettingEmpty("enableStickyTopics") {
		stickyOrder = "t.isSticky DESC, "
	}
	trows, err := a.DB.Query(a.Q(`
		SELECT t.ID_TOPIC, m.subject, m.ID_MEMBER, IFNULL(mem.realName, m.posterName) AS posterName
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_MSG = t.ID_FIRST_MSG
			AND t.ID_BOARD = ?
			AND t.ID_TOPIC != ?
		ORDER BY `+stickyOrder+`t.ID_LAST_MSG DESC
		LIMIT ? OFFSET ?`), targetBoard, from, maxTopics, c.REQUEST.Int("start"))
	if err == nil {
		for trows.Next() {
			var id, idMember int
			var subj, posterName string
			trows.Scan(&id, &subj, &idMember, &posterName)
			subj = c.censorText(subj)
			link := posterName
			if idMember != 0 {
				link = `<a href="` + scripturl + `?action=profile;u=` + itoa(idMember) + `" target="_blank">` + posterName + `</a>`
			}
			page.Topics = append(page.Topics, MergeListTopic{ID: id, Subject: subj, PosterLink: link})
		}
		trows.Close()
	}

	if len(page.Topics) == 0 && len(page.Boards) <= 1 {
		c.fatalLangError("merge_need_more_topics", true)
	}

	c.SubTemplate = templateMerge
}

// MergeExecute is MergeExecute($topics): merge two or more topics into one.
func (c *Ctx) MergeExecute(topics []int) {
	a := c.App
	scripturl := a.ScriptURL

	// A non-empty topics arg means an internal call (from QuickModeration).
	if len(topics) > 0 {
		c.isAllowedTo("merge_any")
	}
	c.checkSession("request", "", true)

	// Handle URLs from MergeIndex.
	if c.GET.Int("from") != 0 && c.GET.Int("to") != 0 {
		topics = []int{c.GET.Int("from"), c.GET.Int("to")}
	}
	// From the form, the topic IDs come by post.
	if arr := c.POST.Arr("topics"); arr != nil {
		topics = nil
		arr.Values(func(k string, v any) {
			s, _ := v.(string)
			topics = append(topics, atoi(s))
		})
	}

	// Need more than one topic.
	if len(topics) <= 1 {
		c.fatalLangError("merge_need_more_topics", true)
	}

	// Get info about the topics (and polls) to merge.
	rows, err := a.DB.Query(a.Q(`
		SELECT
			t.ID_TOPIC, t.ID_BOARD, t.ID_POLL, t.numViews, t.isSticky,
			m1.subject, m1.posterTime AS time_started, IFNULL(mem1.ID_MEMBER, 0) AS ID_MEMBER_STARTED, IFNULL(mem1.realName, m1.posterName) AS name_started,
			m2.posterTime AS time_updated, IFNULL(mem2.ID_MEMBER, 0) AS ID_MEMBER_UPDATED, IFNULL(mem2.realName, m2.posterName) AS name_updated
		FROM {$db_prefix}topics AS t, {$db_prefix}messages AS m1, {$db_prefix}messages AS m2
			LEFT JOIN {$db_prefix}members AS mem1 ON (mem1.ID_MEMBER = m1.ID_MEMBER)
			LEFT JOIN {$db_prefix}members AS mem2 ON (mem2.ID_MEMBER = m2.ID_MEMBER)
		WHERE t.ID_TOPIC IN (` + joinInts(topics) + `)
			AND m1.ID_MSG = t.ID_FIRST_MSG
			AND m2.ID_MSG = t.ID_LAST_MSG
		ORDER BY t.ID_FIRST_MSG`))
	if err != nil {
		c.fatalLangError("smf263", true)
	}

	type topicData struct {
		id, board, poll, numViews int
		subject                   string
		started, updated          MergePerson
	}
	tdata := map[int]*topicData{}
	var tdataOrder []int
	numViews := 0
	isSticky := 0
	var boards []int
	var polls []int
	firstTopic := 0
	for rows.Next() {
		var id, board, poll, nv, sticky, memStarted, memUpdated int
		var subject, nameStarted, nameUpdated string
		var timeStarted, timeUpdated int64
		rows.Scan(&id, &board, &poll, &nv, &sticky, &subject, &timeStarted, &memStarted, &nameStarted, &timeUpdated, &memUpdated, &nameUpdated)
		td := &topicData{id: id, board: board, poll: poll, numViews: nv, subject: subject}
		td.started = MergePerson{Time: c.timeformat(timeStarted), Link: profileLinkOr(scripturl, memStarted, nameStarted)}
		td.updated = MergePerson{Time: c.timeformat(timeUpdated), Link: profileLinkOr(scripturl, memUpdated, nameUpdated)}
		tdata[id] = td
		tdataOrder = append(tdataOrder, id)
		numViews += nv
		boards = append(boards, board)
		if poll > 0 {
			polls = append(polls, poll)
		}
		if firstTopic == 0 {
			firstTopic = id
		}
		if sticky > isSticky {
			isSticky = sticky
		}
	}
	rows.Close()
	if len(tdataOrder) < 2 {
		c.fatalLangError("smf263", true)
	}

	boards = uniqueInts(boards)

	// Boards the user can merge in.
	mergeBoards := c.boardsAllowedTo("merge_any")
	if len(mergeBoards) == 0 {
		c.fatalLangError("cannot_merge_any", true)
	}

	// Make sure they can see all boards involved.
	mergeCond := ""
	if !inInts(mergeBoards, 0) {
		mergeCond = "\n\t\t\tAND b.ID_BOARD IN (" + joinInts(mergeBoards) + ")"
	}
	var seen int
	vrows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD
		FROM {$db_prefix}boards AS b
		WHERE b.ID_BOARD IN (` + joinInts(boards) + `)
			AND ` + c.User.QuerySeeBoard + mergeCond))
	if err == nil {
		for vrows.Next() {
			var b int
			vrows.Scan(&b)
			seen++
		}
		vrows.Close()
	}
	if seen != len(boards) {
		c.fatalLangError("smf232", false)
	}

	// First pass: show the extra-options form.
	sa := c.REQUEST.Str("sa")
	if sa == "" || sa == "options" {
		page := &MergeExtraCtx{}
		c.Page = page

		if len(polls) > 1 {
			prows, err := a.DB.Query(a.Q(`
				SELECT t.ID_TOPIC, t.ID_POLL, m.subject, p.question
				FROM {$db_prefix}polls AS p, {$db_prefix}topics AS t, {$db_prefix}messages AS m
				WHERE p.ID_POLL IN (` + joinInts(polls) + `)
					AND t.ID_POLL = p.ID_POLL
					AND m.ID_MSG = t.ID_FIRST_MSG`))
			if err == nil {
				for prows.Next() {
					var tID, pID int
					var subj, question string
					prows.Scan(&tID, &pID, &subj, &question)
					page.Polls = append(page.Polls, MergePoll{ID: pID, TopicID: tID, TopicSubject: subj, Question: question, Selected: tID == firstTopic})
				}
				prows.Close()
			}
		}
		if len(boards) > 1 {
			brows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name FROM {$db_prefix}boards WHERE ID_BOARD IN (` + joinInts(boards) + `) ORDER BY name`))
			if err == nil {
				for brows.Next() {
					var id int
					var name string
					brows.Scan(&id, &name)
					page.Boards = append(page.Boards, MergeBoard{ID: id, Name: name, Selected: id == tdata[firstTopic].board})
				}
				brows.Close()
			}
		}

		for _, id := range tdataOrder {
			td := tdata[id]
			page.Topics = append(page.Topics, MergeTopicInfo{
				ID: td.id, Subject: td.subject, Started: td.started, Updated: td.updated, Selected: td.id == firstTopic,
			})
		}

		c.PageTitle = c.Txt("smf252")
		c.SubTemplate = templateMergeExtraOptions
		return
	}

	// Determine target board.
	targetBoard := boards[0]
	if len(boards) > 1 {
		targetBoard = c.REQUEST.Int("board")
	}
	if !inInts(boards, targetBoard) {
		c.fatalLangError("smf232", false)
	}

	// Determine which poll survives.
	targetPoll := 0
	if len(polls) > 1 {
		targetPoll = c.POST.Int("poll")
	} else if len(polls) == 1 {
		targetPoll = polls[0]
	}
	if targetPoll > 0 && !inInts(polls, targetPoll) {
		c.fatalLangError("1", false)
	}
	var deletedPolls []int
	if targetPoll == 0 {
		deletedPolls = polls
	} else {
		deletedPolls = diffInts(polls, []int{targetPoll})
	}

	// Determine the merged subject (PHP empty(): "0" counts as blank, the
	// custom-subject sentinel from the form's select).
	var targetSubject string
	if empty(c.POST.Str("subject")) && c.POST.Has("custom_subject") && c.POST.Str("custom_subject") != "" {
		targetSubject = Htmlspecialchars(c.POST.Str("custom_subject"))
	} else if td, ok := tdata[c.POST.Int("subject")]; ok && !empty(td.subject) {
		targetSubject = td.subject
	} else {
		targetSubject = tdata[firstTopic].subject
	}

	// First/last message and the reply count.
	var firstMsg, lastMsg, numReplies int
	a.DB.QueryRow(a.Q(`
		SELECT MIN(ID_MSG), MAX(ID_MSG), COUNT(ID_MSG) - 1
		FROM {$db_prefix}messages
		WHERE ID_TOPIC IN (`+joinInts(topics)+`)`)).Scan(&firstMsg, &lastMsg, &numReplies)

	// Member IDs of the first and last message.
	var memberStarted, memberUpdated int
	mrows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER
		FROM {$db_prefix}messages
		WHERE ID_MSG IN (?, ?)
		ORDER BY ID_MSG
		LIMIT 2`), firstMsg, lastMsg)
	if err == nil {
		if mrows.Next() {
			mrows.Scan(&memberStarted)
		}
		if mrows.Next() {
			mrows.Scan(&memberUpdated)
		}
		mrows.Close()
	}

	// The lowest topic ID becomes the merged topic.
	idTopic := topics[0]
	for _, t := range topics {
		if t < idTopic {
			idTopic = t
		}
	}
	deletedTopics := diffInts(topics, []int{idTopic})

	// Delete the remaining topics.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}topics WHERE ID_TOPIC IN (` + joinInts(deletedTopics) + `)`))
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_subjects WHERE ID_TOPIC IN (` + joinInts(deletedTopics) + `)`))

	// Assign the properties of the merged topic.
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}topics
		SET ID_BOARD = ?, ID_MEMBER_STARTED = ?, ID_MEMBER_UPDATED = ?, ID_FIRST_MSG = ?, ID_LAST_MSG = ?, ID_POLL = ?, numReplies = ?, numViews = ?, isSticky = ?
		WHERE ID_TOPIC = ?`),
		targetBoard, memberStarted, memberUpdated, firstMsg, lastMsg, targetPoll, numReplies, numViews, isSticky, idTopic)

	// Response prefix (single-language port: forum default == user language).
	responsePrefix := c.Txt("response_prefix")

	// Move all messages to the merged topic; optionally enforce the subject.
	if c.POST.Has("enforce_subject") {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}messages
			SET ID_TOPIC = ?, ID_BOARD = ?, subject = ?
			WHERE ID_TOPIC IN (`+joinInts(topics)+`)`), idTopic, targetBoard, responsePrefix+targetSubject)
	} else {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}messages
			SET ID_TOPIC = ?, ID_BOARD = ?
			WHERE ID_TOPIC IN (`+joinInts(topics)+`)`), idTopic, targetBoard)
	}

	// The first message keeps the bare subject.
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}messages SET subject = ? WHERE ID_MSG = ?`), targetSubject, firstMsg)

	// Merge log_topics (read markers): keep the earliest ID_MSG per member.
	lrows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, MIN(ID_MSG) AS new_ID_MSG
		FROM {$db_prefix}log_topics
		WHERE ID_TOPIC IN (` + joinInts(topics) + `)
		GROUP BY ID_MEMBER`))
	if err == nil {
		type lt struct{ member, msg int }
		var entries []lt
		for lrows.Next() {
			var m, msg int
			lrows.Scan(&m, &msg)
			entries = append(entries, lt{m, msg})
		}
		lrows.Close()
		if len(entries) > 0 {
			for _, e := range entries {
				a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_topics (ID_MEMBER, ID_TOPIC, ID_MSG) VALUES (?, ?, ?)`), e.member, idTopic, e.msg)
			}
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_TOPIC IN (` + joinInts(deletedTopics) + `)`))
		}
	}

	// Merge topic notifications.
	if narr := c.POST.Arr("notifications"); narr != nil {
		var notifs []int
		narr.Values(func(k string, v any) {
			s, _ := v.(string)
			notifs = append(notifs, atoi(s))
		})
		// Must reference valid topics.
		if len(diffInts(notifs, topics)) > 0 {
			c.fatalLangError("smf232", false)
		}
		nrows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, MAX(sent) AS sent
			FROM {$db_prefix}log_notify
			WHERE ID_TOPIC IN (` + joinInts(notifs) + `)
			GROUP BY ID_MEMBER`))
		if err == nil {
			type ln struct{ member, sent int }
			var entries []ln
			for nrows.Next() {
				var m, sent int
				nrows.Scan(&m, &sent)
				entries = append(entries, ln{m, sent})
			}
			nrows.Close()
			if len(entries) > 0 {
				for _, e := range entries {
					a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_notify (ID_MEMBER, ID_TOPIC, ID_BOARD, sent) VALUES (?, ?, 0, ?)`), e.member, idTopic, e.sent)
				}
				a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_topics WHERE ID_TOPIC IN (` + joinInts(deletedTopics) + `)`))
			}
		}
	}

	// Get rid of the redundant polls.
	if len(deletedPolls) > 0 {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}polls WHERE ID_POLL IN (` + joinInts(deletedPolls) + `)`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}poll_choices WHERE ID_POLL IN (` + joinInts(deletedPolls) + `)`))
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_polls WHERE ID_POLL IN (` + joinInts(deletedPolls) + `)`))
	}

	// Fix the board totals.
	if len(boards) > 1 {
		frows, err := a.DB.Query(a.Q(`
			SELECT ID_BOARD, COUNT(*) AS numTopics, SUM(numReplies) + COUNT(*) AS numPosts
			FROM {$db_prefix}topics
			WHERE ID_BOARD IN (` + joinInts(boards) + `)
			GROUP BY ID_BOARD`))
		if err == nil {
			type bt struct{ board, nt, np int }
			var bts []bt
			for frows.Next() {
				var b, nt, np int
				frows.Scan(&b, &nt, &np)
				bts = append(bts, bt{b, nt, np})
			}
			frows.Close()
			for _, b := range bts {
				a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numPosts = ?, numTopics = ? WHERE ID_BOARD = ?`), b.np, b.nt, b.board)
			}
		}
	} else {
		dec := len(topics) - 1
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}boards SET numTopics = IIF(? > numTopics, 0, numTopics - ?) WHERE ID_BOARD = ?`), dec, dec, targetBoard)
	}

	// Update statistics.
	a.updateStatsTopic()
	c.updateStatsSubject(idTopic, targetSubject)
	c.updateLastMessages(boards, 0)

	c.logAction("merge", map[string]any{"topic": idTopic})

	c.sendNotifications(idTopic, "merge")

	c.redirectExit("action=mergetopics;sa=done;to=" + itoa(idTopic) + ";targetboard=" + itoa(targetBoard))
}

// MergeDone is MergeDone(): the all-done confirmation page.
func (c *Ctx) MergeDone() {
	page := &MergeDoneCtx{
		TargetBoard: c.GET.Int("targetboard"),
		TargetTopic: c.GET.Int("to"),
	}
	c.Page = page
	c.PageTitle = c.Txt("smf252")
	c.SubTemplate = templateMergeDone
}

// ---- small helpers ----

// splitSelection returns the session's selected message IDs for a topic.
func (c *Ctx) splitSelection(topic int) []int {
	raw, _ := c.Session.Get("split_selection_" + itoa(topic)).([]any)
	out := make([]int, 0, len(raw))
	for _, v := range raw {
		out = append(out, int(toFloat(v)))
	}
	return out
}

// setSplitSelection stores the session's selected message IDs for a topic.
func (c *Ctx) setSplitSelection(topic int, sel []int) {
	vals := make([]any, len(sel))
	for i, s := range sel {
		vals[i] = s
	}
	c.Session.Set("split_selection_"+itoa(topic), vals)
}

// profileLinkOr returns a profile link, or the bare name for guests.
func profileLinkOr(scripturl string, memberID int, name string) string {
	if memberID == 0 {
		return name
	}
	return `<a href="` + scripturl + `?action=profile;u=` + itoa(memberID) + `">` + name + `</a>`
}

// removeInt returns xs without the first occurrence of v.
func removeInt(xs []int, v int) []int {
	out := xs[:0:0]
	for _, x := range xs {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// diffInts returns the elements of a not present in b (array_diff).
func diffInts(a, b []int) []int {
	set := make(map[int]bool, len(b))
	for _, x := range b {
		set[x] = true
	}
	var out []int
	for _, x := range a {
		if !set[x] {
			out = append(out, x)
		}
	}
	return out
}

// messageIDs extracts the IDs from a slice of SplitMessage.
func messageIDs(msgs []SplitMessage) []int {
	out := make([]int, len(msgs))
	for i, m := range msgs {
		out[i] = m.ID
	}
	return out
}

// findSplitMessage returns the message with the given ID (zero value if none).
func findSplitMessage(msgs []SplitMessage, id int) SplitMessage {
	for _, m := range msgs {
		if m.ID == id {
			return m
		}
	}
	return SplitMessage{}
}
