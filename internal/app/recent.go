package app

// Port of Sources/Recent.php: RecentPosts (?action=recent). getLastPost(s)
// (the board-index "latest posts" helpers) and UnreadTopics
// (?action=unread / unreadreplies, the read-tracking listing) are not yet
// ported — UnreadTopics needs the full log_topics/log_boards/log_mark_read
// machinery and arrives in a follow-up.

import "strings"

func init() {
	registerAction("recent", (*Ctx).RecentPosts)
}

// RecentPoster is a first/last poster reference on a recent post.
type RecentPoster struct {
	ID   int
	Name string
	Href string
	Link string
}

// RecentRef is a category/board reference on a recent post.
type RecentRef struct {
	ID   int
	Name string
	Href string
	Link string
}

// RecentPost is one row in the recent-posts listing.
type RecentPost struct {
	ID             int
	Counter        int
	Category       RecentRef
	Board          RecentRef
	Topic          int
	Href           string
	Link           string
	Start          int
	Subject        string
	Time           string
	FirstPoster    RecentPoster
	Poster         RecentPoster
	Message        string
	CanReply       bool
	CanMarkNotify  bool
	CanDelete      bool
	DeletePossible bool
}

// RecentCtx is the page context for the Recent template.
type RecentCtx struct {
	PageIndex string
	Posts     []*RecentPost
}

// RecentPosts is RecentPosts(): the most recent posts on the forum
// (optionally scoped to a category, a board set, or the current board).
func (c *Ctx) RecentPosts() {
	a := c.App
	scripturl := a.ScriptURL

	c.PageTitle = c.Txt("214")
	page := &RecentCtx{}
	c.Page = page

	start := c.REQUEST.Int("start")
	if c.REQUEST.Has("start") && start > 95 {
		start = 95
	}

	maxMsgID := a.SettingInt("maxMsgID")
	totalMessages := a.SettingInt("totalMessages")

	var queryThisBoard string
	catParam := ""

	switch {
	case !empty(c.REQUEST.Str("c")) && c.Board == 0:
		var cats []string
		for _, cv := range strings.Split(c.REQUEST.Str("c"), ",") {
			cats = append(cats, itoa(atoi(cv)))
		}
		catParam = strings.Join(cats, ",")

		if len(cats) == 1 {
			var name string
			a.DB.QueryRow(a.Q(`SELECT name FROM {$db_prefix}categories WHERE ID_CAT = `+cats[0]+` LIMIT 1`)).Scan(&name)
			if name == "" {
				c.fatalLangError("1", false)
			}
			// (int)$_REQUEST['c'] of the exploded array is 1 in PHP; preserve it.
			c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "#1", Name: name})
		}

		boards, totalCatPosts := c.recentBoardsByCat(cats)
		if len(boards) == 0 {
			c.fatalLangError("error_no_boards_selected", false)
		}
		queryThisBoard = "b.ID_BOARD IN (" + joinInts(boards) + ")"
		if totalCatPosts > 100 && totalCatPosts > totalMessages/15 {
			queryThisBoard += "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-400-start*7))
		}
		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=recent;c="+catParam, start, minInt(100, totalCatPosts), 10, false)

	case !empty(c.REQUEST.Str("boards")):
		var want []string
		for _, bv := range strings.Split(c.REQUEST.Str("boards"), ",") {
			want = append(want, itoa(atoi(bv)))
		}
		boards, totalPosts := c.recentBoardsByID(want)
		if len(boards) == 0 {
			c.fatalLangError("error_no_boards_selected", false)
		}
		queryThisBoard = "b.ID_BOARD IN (" + joinInts(boards) + ")"
		if totalPosts > 100 && totalPosts > totalMessages/12 {
			queryThisBoard += "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-500-start*9))
		}
		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=recent;boards="+strings.Join(want, ","), start, minInt(100, totalPosts), 10, false)

	case c.Board != 0:
		var totalPosts int
		a.DB.QueryRow(a.Q(`SELECT numPosts FROM {$db_prefix}boards WHERE ID_BOARD = ? LIMIT 1`), c.Board).Scan(&totalPosts)
		queryThisBoard = "b.ID_BOARD = " + itoa(c.Board)
		if totalPosts > 80 && totalPosts > totalMessages/10 {
			queryThisBoard += "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-600-start*10))
		}
		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=recent;board="+itoa(c.Board)+".%d", start, minInt(100, totalPosts), 10, true)

	default:
		recycle := ""
		if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
			recycle = "\n\t\t\tAND b.ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
		}
		queryThisBoard = c.User.QuerySeeBoard + recycle + "\n\t\t\tAND m.ID_MSG >= " + itoa(maxInt(0, maxMsgID-100-start*6))
		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=recent", start, minInt(100, totalMessages), 10, false)
	}

	linkURL := scripturl + "?action=recent"
	if c.Board != 0 {
		linkURL += ";board=" + itoa(c.Board) + ".0"
	} else if catParam != "" {
		linkURL += ";c=1" // PHP casts the array to (int) here.
	}
	c.LinkTree = append(c.LinkTree, Link{URL: linkURL, Name: c.PageTitle})

	// Find the 10 most recent messages they can *view*.
	var messages []int
	rows, err := a.DB.Query(a.Q(`
		SELECT m.ID_MSG
		FROM ({$db_prefix}messages AS m, {$db_prefix}boards AS b)
		WHERE b.ID_BOARD = m.ID_BOARD
			AND ` + queryThisBoard + `
		ORDER BY m.ID_MSG DESC
		LIMIT 10 OFFSET ` + itoa(start)))
	if err == nil {
		for rows.Next() {
			var id int
			rows.Scan(&id)
			messages = append(messages, id)
		}
		rows.Close()
	}
	if len(messages) == 0 {
		c.SubTemplate = templateRecentMain
		return
	}

	// own/any board -> message ids, for the permission pass.
	boardIDsOwn := map[int][]int{}
	boardIDsAny := map[int][]int{}
	byID := map[int]*RecentPost{}

	counter := start + 1
	prows, err := a.DB.Query(a.Q(`
		SELECT
			m.ID_MSG, m.subject, m.smileysEnabled, m.posterTime, m.body, m.ID_TOPIC, t.ID_BOARD, b.ID_CAT,
			b.name AS bname, c.name AS cname, t.numReplies, m.ID_MEMBER, m2.ID_MEMBER AS ID_FIRST_MEMBER,
			IFNULL(mem2.realName, m2.posterName) AS firstPosterName, t.ID_FIRST_MSG,
			IFNULL(mem.realName, m.posterName) AS posterName, t.ID_LAST_MSG
		FROM ({$db_prefix}messages AS m, {$db_prefix}messages AS m2, {$db_prefix}topics AS t, {$db_prefix}boards AS b, {$db_prefix}categories AS c)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
			LEFT JOIN {$db_prefix}members AS mem2 ON (mem2.ID_MEMBER = m2.ID_MEMBER)
		WHERE m2.ID_MSG = t.ID_FIRST_MSG
			AND t.ID_TOPIC = m.ID_TOPIC
			AND b.ID_BOARD = t.ID_BOARD
			AND c.ID_CAT = b.ID_CAT
			AND m.ID_MSG IN (` + joinInts(messages) + `)
		ORDER BY m.ID_MSG DESC
		LIMIT ` + itoa(len(messages))))
	if err == nil {
		editDisable := a.SettingInt("edit_disable_time")
		for prows.Next() {
			var idMsg, smileys, idTopic, idBoard, idCat, numReplies, idMember, idFirstMember, idFirstMsg, idLastMsg int
			var subject, body, bname, cname, firstPosterName, posterName string
			var posterTime int64
			prows.Scan(&idMsg, &subject, &smileys, &posterTime, &body, &idTopic, &idBoard, &idCat,
				&bname, &cname, &numReplies, &idMember, &idFirstMember, &firstPosterName, &idFirstMsg, &posterName, &idLastMsg)

			body = c.censorText(body)
			subject = c.censorText(subject)
			body = c.parseBBCCached(body, smileys != 0, itoa(idMsg))

			deletePossible := (idFirstMsg != idMsg || idLastMsg == idMsg) &&
				(editDisable == 0 || posterTime+int64(editDisable)*60 >= nowUnix())

			fp := RecentPoster{ID: idFirstMember, Name: firstPosterName}
			if idFirstMember == 0 {
				fp.Link = firstPosterName
			} else {
				fp.Href = scripturl + "?action=profile;u=" + itoa(idFirstMember)
				fp.Link = `<a href="` + fp.Href + `">` + firstPosterName + `</a>`
			}
			pp := RecentPoster{ID: idMember, Name: posterName}
			if idMember == 0 {
				pp.Link = posterName
			} else {
				pp.Href = scripturl + "?action=profile;u=" + itoa(idMember)
				pp.Link = `<a href="` + pp.Href + `">` + posterName + `</a>`
			}

			post := &RecentPost{
				ID:      idMsg,
				Counter: counter,
				Category: RecentRef{ID: idCat, Name: cname,
					Href: scripturl + "#" + itoa(idCat),
					Link: `<a href="` + scripturl + `#` + itoa(idCat) + `">` + cname + `</a>`},
				Board: RecentRef{ID: idBoard, Name: bname,
					Href: scripturl + "?board=" + itoa(idBoard) + ".0",
					Link: `<a href="` + scripturl + `?board=` + itoa(idBoard) + `.0">` + bname + `</a>`},
				Topic:          idTopic,
				Href:           scripturl + "?topic=" + itoa(idTopic) + ".msg" + itoa(idMsg) + "#msg" + itoa(idMsg),
				Link:           `<a href="` + scripturl + `?topic=` + itoa(idTopic) + `.msg` + itoa(idMsg) + `#msg` + itoa(idMsg) + `">` + subject + `</a>`,
				Start:          numReplies,
				Subject:        subject,
				Time:           c.timeformat(posterTime),
				FirstPoster:    fp,
				Poster:         pp,
				Message:        body,
				DeletePossible: deletePossible,
			}
			counter++
			page.Posts = append(page.Posts, post)
			byID[idMsg] = post

			if c.User.ID != 0 && c.User.ID == idFirstMember {
				boardIDsOwn[idBoard] = append(boardIDsOwn[idBoard], idMsg)
			}
			boardIDsAny[idBoard] = append(boardIDsAny[idBoard], idMsg)
		}
		prows.Close()
	}

	// Apply per-board permissions (own then any).
	applyPerm := func(boardMsgs map[int][]int, permission string, set func(*RecentPost), anyType bool) {
		boards := c.boardsAllowedTo(permission)
		if len(boards) != 0 && boards[0] == 0 {
			boards = nil
			for b := range boardMsgs {
				boards = append(boards, b)
			}
		}
		for _, b := range boards {
			msgs, ok := boardMsgs[b]
			if !ok {
				continue
			}
			for _, msg := range msgs {
				post := byID[msg]
				if anyType || post.Poster.ID == c.User.ID {
					set(post)
				}
			}
		}
	}
	applyPerm(boardIDsOwn, "post_reply_own", func(p *RecentPost) { p.CanReply = true }, false)
	applyPerm(boardIDsOwn, "delete_own", func(p *RecentPost) { p.CanDelete = true }, false)
	applyPerm(boardIDsAny, "post_reply_any", func(p *RecentPost) { p.CanReply = true }, true)
	applyPerm(boardIDsAny, "mark_any_notify", func(p *RecentPost) { p.CanMarkNotify = true }, true)
	applyPerm(boardIDsAny, "delete_any", func(p *RecentPost) { p.CanDelete = true }, true)

	// Some posts - the first posts - can't just be deleted.
	for _, p := range page.Posts {
		p.CanDelete = p.CanDelete && p.DeletePossible
	}

	c.SubTemplate = templateRecentMain
}

// recentBoardsByCat returns the visible boards in the given categories and
// their combined post count.
func (c *Ctx) recentBoardsByCat(cats []string) ([]int, int) {
	a := c.App
	var boards []int
	total := 0
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.numPosts
		FROM {$db_prefix}boards AS b
		WHERE b.ID_CAT IN (` + strings.Join(cats, ", ") + `)
			AND ` + c.User.QuerySeeBoard))
	if err == nil {
		for rows.Next() {
			var id, numPosts int
			rows.Scan(&id, &numPosts)
			boards = append(boards, id)
			total += numPosts
		}
		rows.Close()
	}
	return boards, total
}

// recentBoardsByID returns the visible boards among the given ids and their
// combined post count.
func (c *Ctx) recentBoardsByID(want []string) ([]int, int) {
	a := c.App
	var boards []int
	total := 0
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_BOARD, b.numPosts
		FROM {$db_prefix}boards AS b
		WHERE b.ID_BOARD IN (` + strings.Join(want, ", ") + `)
			AND ` + c.User.QuerySeeBoard + `
		LIMIT ` + itoa(len(want))))
	if err == nil {
		for rows.Next() {
			var id, numPosts int
			rows.Scan(&id, &numPosts)
			boards = append(boards, id)
			total += numPosts
		}
		rows.Close()
	}
	return boards, total
}

// LatestPost is one entry in the board-index "recent posts" list
// (getLastPosts). Only the fields the board index renders are kept.
type LatestPost struct {
	Link       string
	PosterLink string
	BoardLink  string
	Time       string
	Timestamp  int64
}

// getLastPosts is getLastPosts($showlatestcount): the N most recent posts the
// user can see, for the board-index info-center list. (The guest cache is
// skipped — recomputed each request, identical output.)
func (c *Ctx) getLastPosts(showLatestCount int) []*LatestPost {
	a := c.App
	scripturl := a.ScriptURL

	recycle := ""
	if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
		recycle = "\n\t\t\tAND b.ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
	}
	floor := maxInt(0, a.SettingInt("maxMsgID")-20*showLatestCount)

	var posts []*LatestPost
	rows, err := a.DB.Query(a.Q(`
		SELECT
			m.posterTime, m.subject, m.ID_TOPIC, m.ID_MEMBER, m.ID_MSG,
			IFNULL(mem.realName, m.posterName) AS posterName, t.ID_BOARD, b.name AS bName,
			SUBSTR(m.body, 1, 384) AS body, m.smileysEnabled
		FROM ({$db_prefix}messages AS m, {$db_prefix}topics AS t, {$db_prefix}boards AS b)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_MSG >= ` + itoa(floor) + `
			AND t.ID_TOPIC = m.ID_TOPIC
			AND b.ID_BOARD = t.ID_BOARD` + recycle + `
			AND ` + c.User.QuerySeeBoard + `
		ORDER BY m.ID_MSG DESC
		LIMIT ` + itoa(showLatestCount)))
	if err != nil {
		return nil
	}
	for rows.Next() {
		var posterTime int64
		var subject, posterName, bName, body string
		var idTopic, idMember, idMsg, idBoard, smileys int
		rows.Scan(&posterTime, &subject, &idTopic, &idMember, &idMsg, &posterName, &idBoard, &bName, &body, &smileys)

		subject = c.censorText(subject)
		// (preview is censored/parsed in PHP but the board-index list doesn't
		// render it, so we skip that work.)

		posterLink := posterName
		if idMember != 0 {
			posterLink = `<a href="` + scripturl + `?action=profile;u=` + itoa(idMember) + `">` + posterName + `</a>`
		}
		href := scripturl + "?topic=" + itoa(idTopic) + ".msg" + itoa(idMsg) + ";topicseen#msg" + itoa(idMsg)
		posts = append(posts, &LatestPost{
			Link:       `<a href="` + href + `">` + subject + `</a>`,
			PosterLink: posterLink,
			BoardLink:  `<a href="` + scripturl + `?board=` + itoa(idBoard) + `.0">` + bName + `</a>`,
			Time:       c.timeformat(posterTime),
			Timestamp:  c.forumTime(true, posterTime),
		})
	}
	rows.Close()
	return posts
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = itoa(x)
	}
	return strings.Join(parts, ", ")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
