package app

// Port of Sources/Recent.php UnreadTopics() — ?action=unread (new since last
// visit / all) and ?action=unreadreplies (topics you've posted in that are
// unread). The >100k-message temporary-table optimizations are NOT ported:
// the non-temp-table query paths produce identical results, just without the
// large-forum speedups (documented deviation).

import (
	"database/sql"
	"os"
	"strings"
)

func init() {
	registerAction("unread", (*Ctx).UnreadTopics)
	registerAction("unreadreplies", (*Ctx).UnreadTopics)
}

// UnreadCtx is the page context for the unread / replies templates.
type UnreadCtx struct {
	ShowingAllTopics       bool
	Action                 string // "unread" or "unreadreplies"
	PageIndex              string
	SortBy                 string
	SortDirection          string
	QuerystringBoardLimits string
	QuerystringSortLimits  string
	NoBoardLimits          bool
	TopicsToMark           string
	Topics                 []*MITopic
}

// unreadSelectClause is the shared SELECT list for the topic detail query.
const unreadSelectClause = `
			ms.subject AS firstSubject, ms.posterTime AS firstPosterTime, ms.ID_TOPIC, t.ID_BOARD, b.name AS bname,
			t.numReplies, t.numViews, ms.ID_MEMBER AS ID_FIRST_MEMBER, ml.ID_MEMBER AS ID_LAST_MEMBER,
			ml.posterTime AS lastPosterTime, IFNULL(mems.realName, ms.posterName) AS firstPosterName,
			IFNULL(meml.realName, ml.posterName) AS lastPosterName, ml.subject AS lastSubject,
			ml.icon AS lastIcon, ms.icon AS firstIcon, t.ID_POLL, t.isSticky, t.locked, ml.modifiedTime AS lastModifiedTime,
			IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, -1)) + 1 AS new_from, SUBSTR(ml.body, 1, 384) AS lastBody, SUBSTR(ms.body, 1, 384) AS firstBody,
			ml.smileysEnabled AS lastSmileys, ms.smileysEnabled AS firstSmileys, t.ID_FIRST_MSG, t.ID_LAST_MSG`

// UnreadTopics is UnreadTopics().
func (c *Ctx) UnreadTopics() {
	a := c.App
	scripturl := a.ScriptURL
	me := c.User.ID

	// Guests can't have unread things.
	c.isNotGuest("")

	// Prefetching + lots of work = bad mojo.
	if c.R.Header.Get("X-Moz") == "prefetch" {
		c.W.WriteHeader(403)
		c.exit()
	}

	action := c.REQUEST.Str("action")
	isTopics := action == "unread"

	page := &UnreadCtx{Action: action, ShowingAllTopics: c.GET.Has("all")}
	c.Page = page

	if action == "unread" {
		if page.ShowingAllTopics {
			c.PageTitle = c.Txt("unread_topics_all")
		} else {
			c.PageTitle = c.Txt("unread_topics_visit")
		}
	} else {
		c.PageTitle = c.Txt("unread_replies")
	}

	// Board scoping -> queryThisBoard (a bare "ID_BOARD ..." condition) +
	// querystring_board_limits.
	var queryThisBoard, qblFmt string
	switch {
	case c.Board != 0:
		queryThisBoard = "ID_BOARD = " + itoa(c.Board)
		qblFmt = ";board=" + itoa(c.Board) + ".%d"
	case !empty(c.REQUEST.Str("boards")):
		var want []string
		for _, bv := range strings.Split(c.REQUEST.Str("boards"), ",") {
			want = append(want, itoa(atoi(bv)))
		}
		boards := c.unreadVisibleBoards("b.ID_BOARD IN (" + strings.Join(want, ", ") + ")")
		if len(boards) == 0 {
			c.fatalLangError("error_no_boards_selected", true)
		}
		queryThisBoard = "ID_BOARD IN (" + joinInts(boards) + ")"
		qblFmt = ";boards=" + joinInts2(boards) + ";start=%d"
	case !empty(c.REQUEST.Str("c")):
		var cats []string
		for _, cv := range strings.Split(c.REQUEST.Str("c"), ",") {
			cats = append(cats, itoa(atoi(cv)))
		}
		boards := c.unreadVisibleBoards("b.ID_CAT IN (" + strings.Join(cats, ", ") + ")")
		if len(boards) == 0 {
			c.fatalLangError("error_no_boards_selected", true)
		}
		queryThisBoard = "ID_BOARD IN (" + joinInts(boards) + ")"
		qblFmt = ";c=" + strings.Join(cats, ",") + ";start=%d"

		if len(cats) == 1 {
			var name string
			a.DB.QueryRow(a.Q(`SELECT name FROM {$db_prefix}categories WHERE ID_CAT = ` + cats[0] + ` LIMIT 1`)).Scan(&name)
			c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "#" + cats[0], Name: name})
		}
	default:
		recycle := ""
		if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
			recycle = " AND b.ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
		}
		boards := c.unreadVisibleBoards(c.User.QuerySeeBoard + recycle)
		if len(boards) == 0 {
			c.fatalLangError("error_no_boards_selected", true)
		}
		queryThisBoard = "ID_BOARD IN (" + joinInts(boards) + ")"
		qblFmt = ";start=%d"
		page.NoBoardLimits = true
	}

	// Sorting.
	sortMethods := map[string]string{
		"subject":    "ms.subject",
		"starter":    "IFNULL(mems.realName, ms.posterName)",
		"replies":    "t.numReplies",
		"views":      "t.numViews",
		"first_post": "t.ID_TOPIC",
		"last_post":  "t.ID_LAST_MSG",
	}
	var sortExpr string
	var ascending bool
	if s := c.REQUEST.Str("sort"); s == "" || sortMethods[s] == "" {
		page.SortBy = "last_post"
		sortExpr = "t.ID_LAST_MSG"
		ascending = c.REQUEST.Has("asc")
		if ascending {
			page.QuerystringSortLimits = ";asc"
		}
	} else {
		page.SortBy = s
		sortExpr = sortMethods[s]
		ascending = !c.REQUEST.Has("desc")
		page.QuerystringSortLimits = ";sort=" + page.SortBy
		if !ascending {
			page.QuerystringSortLimits += ";desc"
		}
	}
	page.SortDirection = "down"
	if ascending {
		page.SortDirection = "up"
	}
	dir := " DESC"
	if ascending {
		dir = ""
	}

	// Link tree.
	allParam := ""
	if page.ShowingAllTopics {
		allParam = ";all"
	}
	c.LinkTree = append(c.LinkTree, Link{
		URL:  scripturl + "?action=" + action + sprintfD(qblFmt, 0) + page.QuerystringSortLimits,
		Name: map[bool]string{true: c.Txt("unread_topics_visit"), false: c.Txt("unread_replies")}[action == "unread"],
	})
	if page.ShowingAllTopics {
		c.LinkTree = append(c.LinkTree, Link{
			URL:  scripturl + "?action=" + action + ";all" + sprintfD(qblFmt, 0) + page.QuerystringSortLimits,
			Name: c.Txt("unread_topics_all"),
		})
	}

	c.SubTemplate = templateUnread
	if !isTopics {
		c.SubTemplate = templateReplies
	}

	maxTopics := a.SettingInt("defaultMaxTopics")
	start := c.REQUEST.Int("start")
	lastVisit := int(c.Session.GetInt("ID_MSG_LAST_VISIT"))

	// earliest_msg, for showing-all.
	earliestMsg := 0
	if page.ShowingAllTopics {
		if c.Board != 0 {
			a.DB.QueryRow(a.Q(`SELECT IFNULL(MIN(ID_MSG), 0) FROM {$db_prefix}log_mark_read WHERE ID_MEMBER = ? AND ID_BOARD = ?`), me, c.Board).Scan(&earliestMsg)
		} else {
			a.DB.QueryRow(a.Q(`
				SELECT IFNULL(MIN(lmr.ID_MSG), 0)
				FROM {$db_prefix}boards AS b
					LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = b.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(me) + `)
				WHERE ` + c.User.QuerySeeBoard)).Scan(&earliestMsg)
		}
		if earliestMsg != 0 {
			earliestMsg2 := 0
			a.DB.QueryRow(a.Q(`SELECT IFNULL(MIN(ID_MSG), 0) FROM {$db_prefix}log_topics WHERE ID_MEMBER = ?`), me).Scan(&earliestMsg2)
			if earliestMsg2 == 0 {
				earliestMsg2 = -1
			}
			earliestMsg = minInt(earliestMsg2, earliestMsg)
		}
	}

	var numTopics, minMessage int

	if isTopics {
		// The "new since last visit / all" listing.
		whereLast := "t.ID_LAST_MSG > " + itoa(lastVisit)
		if page.ShowingAllTopics && earliestMsg != 0 {
			whereLast = "t.ID_LAST_MSG > " + itoa(earliestMsg)
		}
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*), IFNULL(MIN(t.ID_LAST_MSG), 0)
			FROM {$db_prefix}topics AS t
				LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(me) + `)
				LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = t.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(me) + `)
			WHERE t.` + queryThisBoard + `
				AND ` + whereLast + `
				AND IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, 0)) < t.ID_LAST_MSG`)).Scan(&numTopics, &minMessage)

		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action="+action+allParam+qblFmt+page.QuerystringSortLimits, start, numTopics, maxTopics, true)
		if numTopics == 0 {
			page.finishEmpty(qblFmt, start)
			return
		}

		rows, err := a.DB.Query(a.Q(`
			SELECT ` + unreadSelectClause + `
			FROM ({$db_prefix}messages AS ms, {$db_prefix}messages AS ml, {$db_prefix}topics AS t, {$db_prefix}boards AS b)
				LEFT JOIN {$db_prefix}members AS mems ON (mems.ID_MEMBER = ms.ID_MEMBER)
				LEFT JOIN {$db_prefix}members AS meml ON (meml.ID_MEMBER = ml.ID_MEMBER)
				LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(me) + `)
				LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = t.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(me) + `)
			WHERE t.ID_TOPIC = ms.ID_TOPIC
				AND b.ID_BOARD = t.ID_BOARD
				AND t.` + queryThisBoard + `
				AND ms.ID_MSG = t.ID_FIRST_MSG
				AND ml.ID_MSG = t.ID_LAST_MSG
				AND t.ID_LAST_MSG >= ` + itoa(minMessage) + `
				AND IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, 0)) < ml.ID_MSG
			ORDER BY ` + sortExpr + dir + `
			LIMIT ` + itoa(maxTopics) + ` OFFSET ` + itoa(start)))
		c.buildUnreadTopics(page, rows, err)
	} else {
		// The "topics I've replied to that are unread" listing.
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(DISTINCT t.ID_TOPIC), IFNULL(MIN(t.ID_LAST_MSG), 0)
			FROM ({$db_prefix}topics AS t, {$db_prefix}messages AS m)
				LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(me) + `)
				LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = t.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(me) + `)
			WHERE t.` + queryThisBoard + `
				AND m.ID_TOPIC = t.ID_TOPIC
				AND m.ID_MEMBER = ` + itoa(me) + `
				AND IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, 0)) < t.ID_LAST_MSG`)).Scan(&numTopics, &minMessage)

		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action="+action+qblFmt+page.QuerystringSortLimits, start, numTopics, maxTopics, true)
		if numTopics == 0 {
			page.finishEmpty(qblFmt, start)
			return
		}

		// Find the topic ids on this page (sorted).
		msJoin, msCond := "", ""
		if strings.Contains(sortExpr, "ms.") {
			msJoin = ", {$db_prefix}messages AS ms"
			msCond = "\n\t\t\t\tAND ms.ID_MSG = t.ID_FIRST_MSG"
		}
		memsJoin := ""
		if strings.Contains(sortExpr, "mems.") {
			memsJoin = "\n\t\t\t\tLEFT JOIN {$db_prefix}members AS mems ON (mems.ID_MEMBER = ms.ID_MEMBER)"
		}
		var topics []int
		trows, err := a.DB.Query(a.Q(`
			SELECT DISTINCT t.ID_TOPIC, ` + sortExpr + ` AS sortKey
			FROM ({$db_prefix}topics AS t, {$db_prefix}messages AS m` + msJoin + `)` + memsJoin + `
				LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(me) + `)
				LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = t.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(me) + `)
			WHERE m.ID_TOPIC = t.ID_TOPIC
				AND m.ID_MEMBER = ` + itoa(me) + `
				AND t.` + queryThisBoard + `
				AND t.ID_LAST_MSG >= ` + itoa(minMessage) + `
				AND IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, 0)) < t.ID_LAST_MSG` + msCond + `
			ORDER BY sortKey` + dir + `
			LIMIT ` + itoa(maxTopics) + ` OFFSET ` + itoa(start)))
		if err == nil {
			for trows.Next() {
				var id int
				var sortKey any
				trows.Scan(&id, &sortKey)
				topics = append(topics, id)
			}
			trows.Close()
		}
		if len(topics) == 0 {
			page.finishEmpty(qblFmt, start)
			return
		}

		rows, err := a.DB.Query(a.Q(`
			SELECT ` + unreadSelectClause + `
			FROM ({$db_prefix}messages AS ms, {$db_prefix}messages AS ml, {$db_prefix}topics AS t, {$db_prefix}boards AS b)
				LEFT JOIN {$db_prefix}members AS mems ON (mems.ID_MEMBER = ms.ID_MEMBER)
				LEFT JOIN {$db_prefix}members AS meml ON (meml.ID_MEMBER = ml.ID_MEMBER)
				LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(me) + `)
				LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = t.ID_BOARD AND lmr.ID_MEMBER = ` + itoa(me) + `)
			WHERE t.ID_TOPIC IN (` + joinInts(topics) + `)
				AND t.ID_TOPIC = ms.ID_TOPIC
				AND b.ID_BOARD = t.ID_BOARD
				AND ms.ID_MSG = t.ID_FIRST_MSG
				AND ml.ID_MSG = t.ID_LAST_MSG
			ORDER BY ` + sortExpr + dir + `
			LIMIT ` + itoa(len(topics))))
		c.buildUnreadTopics(page, rows, err)
	}

	// Mark which ones the user has participated in.
	if isTopics && !a.SettingEmpty("enableParticipation") && len(page.Topics) != 0 {
		var ids []int
		byID := map[int]*MITopic{}
		for _, t := range page.Topics {
			ids = append(ids, t.ID)
			byID[t.ID] = t
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_TOPIC
			FROM {$db_prefix}messages
			WHERE ID_TOPIC IN (` + joinInts(ids) + `)
				AND ID_MEMBER = ?`), me)
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				if t := byID[id]; t != nil && !t.IsPostedIn {
					t.IsPostedIn = true
					t.Class = "my_" + t.Class
				}
			}
			rows.Close()
		}
	}

	page.QuerystringBoardLimits = sprintfD(qblFmt, start)
	var markIDs []int
	for _, t := range page.Topics {
		markIDs = append(markIDs, t.ID)
	}
	page.TopicsToMark = joinIntsSep(markIDs, "-")
}

// finishEmpty handles the num_topics==0 early returns.
func (p *UnreadCtx) finishEmpty(qblFmt string, start int) {
	if qblFmt == ";start=%d" {
		p.QuerystringBoardLimits = ""
	} else {
		p.QuerystringBoardLimits = sprintfD(qblFmt, start)
	}
}

// unreadVisibleBoards returns the visible board ids matching an extra
// condition (already including query_see_board where needed).
func (c *Ctx) unreadVisibleBoards(where string) []int {
	a := c.App
	// The board-set / category cases AND in query_see_board; the default case
	// passes query_see_board itself. Detect by whether it already references it.
	cond := where
	if !strings.Contains(where, "memberGroups") && !strings.Contains(where, "FIND_IN_SET") {
		cond = c.User.QuerySeeBoard + " AND " + where
	}
	var boards []int
	rows, err := a.DB.Query(a.Q(`SELECT b.ID_BOARD FROM {$db_prefix}boards AS b WHERE ` + cond))
	if err == nil {
		for rows.Next() {
			var id int
			rows.Scan(&id)
			boards = append(boards, id)
		}
		rows.Close()
	}
	return boards
}

// buildUnreadTopics turns the detail query into MITopic rows.
func (c *Ctx) buildUnreadTopics(page *UnreadCtx, rows *sql.Rows, err error) {
	if err != nil {
		return
	}
	defer rows.Close()
	a := c.App
	scripturl := a.ScriptURL
	maxMessages := a.SettingInt("defaultMaxMessages")

	iconSources := map[string]string{}
	for _, icon := range []string{"xx", "thumbup", "thumbdown", "exclamation", "question", "lamp",
		"smiley", "angry", "cheesy", "grin", "sad", "wink", "moved", "recycled", "wireless"} {
		iconSources[icon] = "images_url"
	}

	for rows.Next() {
		var firstSubject, bname, firstPosterName, lastPosterName, lastSubject, lastIcon, firstIcon, lastBody, firstBody string
		var firstPosterTime, lastPosterTime, lastModifiedTime int64
		var idTopic, idBoard, numReplies, numViews, idFirstMember, idLastMember, idPoll, isSticky, locked, newFrom, lastSmileys, firstSmileys, idFirstMsg, idLastMsg int
		if err := rows.Scan(&firstSubject, &firstPosterTime, &idTopic, &idBoard, &bname,
			&numReplies, &numViews, &idFirstMember, &idLastMember, &lastPosterTime, &firstPosterName,
			&lastPosterName, &lastSubject, &lastIcon, &firstIcon, &idPoll, &isSticky, &locked, &lastModifiedTime,
			&newFrom, &lastBody, &firstBody, &lastSmileys, &firstSmileys, &idFirstMsg, &idLastMsg); err != nil {
			continue
		}

		if idPoll > 0 && a.Setting("pollMode") == "0" {
			continue
		}

		// Clip + parse the body previews.
		firstBody = stripTags(strings.ReplaceAll(c.parseBBCCached(firstBody, firstSmileys != 0, itoa(idFirstMsg)), "<br />", "&#10;"))
		if len(firstBody) > 128 {
			firstBody = substr(firstBody, 0, 128) + "..."
		}
		lastBody = stripTags(strings.ReplaceAll(c.parseBBCCached(lastBody, lastSmileys != 0, itoa(idLastMsg)), "<br />", "&#10;"))
		if len(lastBody) > 128 {
			lastBody = substr(lastBody, 0, 128) + "..."
		}

		firstSubject = c.censorText(firstSubject)
		firstBody = c.censorText(firstBody)
		if idFirstMsg == idLastMsg {
			lastSubject = firstSubject
			lastBody = firstBody
		} else {
			lastSubject = c.censorText(lastSubject)
			lastBody = c.censorText(lastBody)
		}

		// Pagination links for the topic.
		pages := ""
		if topicLength := numReplies + 1; topicLength > maxMessages {
			var tmppages []string
			tmpa := 1
			for tmpb := 0; tmpb < topicLength; tmpb += maxMessages {
				tmppages = append(tmppages, `<a href="`+scripturl+"?topic="+itoa(idTopic)+"."+itoa(tmpb)+`;topicseen">`+itoa(tmpa)+`</a>`)
				tmpa++
			}
			if len(tmppages) <= 5 {
				pages = "&#171; " + strings.Join(tmppages, " ")
			} else {
				pages = "&#171; " + tmppages[0] + " " + tmppages[1] + " ... " + tmppages[len(tmppages)-2] + " " + tmppages[len(tmppages)-1]
			}
			if !a.SettingEmpty("enableAllMessages") && topicLength < a.SettingInt("enableAllMessages") {
				pages += ` &nbsp;<a href="` + scripturl + "?topic=" + itoa(idTopic) + `.0;all">` + c.Txt("190") + `</a>`
			}
			pages += " &#187;"
		}

		// Topic icons exist?
		for _, icon := range []string{firstIcon, lastIcon} {
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
		}

		lastHrefPart := ".msg" + itoa(idLastMsg)
		if numReplies == 0 {
			lastHrefPart = ".0"
		}

		topic := &MITopic{
			ID: idTopic,
			FirstPost: MIPostInfo{
				ID:        idFirstMsg,
				MemberID:  idFirstMember,
				Name:      firstPosterName,
				Time:      c.timeformat(firstPosterTime),
				Timestamp: c.forumTime(true, firstPosterTime),
				Subject:   firstSubject,
				Preview:   firstBody,
				Icon:      firstIcon,
				IconURL:   c.Theme.Get(iconSources[firstIcon]) + "/post/" + firstIcon + ".gif",
				Href:      scripturl + "?topic=" + itoa(idTopic) + ".0;topicseen",
				Link:      `<a href="` + scripturl + "?topic=" + itoa(idTopic) + `.0;topicseen">` + firstSubject + `</a>`,
			},
			LastPost: MIPostInfo{
				ID:        idLastMsg,
				MemberID:  idLastMember,
				Name:      lastPosterName,
				Time:      c.timeformat(lastPosterTime),
				Timestamp: c.forumTime(true, lastPosterTime),
				Subject:   lastSubject,
				Preview:   lastBody,
				Icon:      lastIcon,
				IconURL:   c.Theme.Get(iconSources[lastIcon]) + "/post/" + lastIcon + ".gif",
				Href:      scripturl + "?topic=" + itoa(idTopic) + lastHrefPart + ";topicseen#msg" + itoa(idLastMsg),
				Link:      `<a href="` + scripturl + "?topic=" + itoa(idTopic) + lastHrefPart + ";topicseen#msg" + itoa(idLastMsg) + `">` + lastSubject + `</a>`,
			},
			NewFrom:   newFrom,
			NewHref:   scripturl + "?topic=" + itoa(idTopic) + ".msg" + itoa(newFrom) + ";topicseen#new",
			IsSticky:  !a.SettingEmpty("enableStickyTopics") && isSticky != 0,
			IsLocked:  locked != 0,
			IsPoll:    a.Setting("pollMode") == "1" && idPoll > 0,
			IsHot:     numReplies >= a.SettingInt("hotTopicPosts"),
			IsVeryHot: numReplies >= a.SettingInt("hotTopicVeryPosts"),
			Icon:      firstIcon,
			IconURL:   c.Theme.Get(iconSources[firstIcon]) + "/post/" + firstIcon + ".gif",
			Subject:   firstSubject,
			Pages:     pages,
			Replies:   numReplies,
			Views:     numViews,
			Board: RecentRef{
				ID:   idBoard,
				Name: bname,
				Href: scripturl + "?board=" + itoa(idBoard) + ".0",
				Link: `<a href="` + scripturl + "?board=" + itoa(idBoard) + `.0">` + bname + `</a>`,
			},
		}
		if idFirstMember != 0 {
			topic.FirstPost.MemberHref = scripturl + "?action=profile;u=" + itoa(idFirstMember)
			topic.FirstPost.MemberLink = `<a href="` + topic.FirstPost.MemberHref + `" title="` + c.Txt("92") + ` ` + firstPosterName + `">` + firstPosterName + `</a>`
		} else {
			topic.FirstPost.MemberLink = firstPosterName
		}
		if idLastMember != 0 {
			topic.LastPost.MemberHref = scripturl + "?action=profile;u=" + itoa(idLastMember)
			topic.LastPost.MemberLink = `<a href="` + topic.LastPost.MemberHref + `">` + lastPosterName + `</a>`
		} else {
			topic.LastPost.MemberLink = lastPosterName
		}
		topic.Class = determineTopicClass(topic.IsVeryHot, topic.IsHot, topic.IsPoll, topic.IsLocked, topic.IsSticky)

		page.Topics = append(page.Topics, topic)
	}
}

// sprintfD substitutes a single %d in a format string (sprintf($s, $n)).
func sprintfD(format string, n int) string {
	return strings.Replace(format, "%d", itoa(n), 1)
}

// joinInts2 is implode(',', xs) — no spaces.
func joinInts2(xs []int) string {
	return joinIntsSep(xs, ",")
}

// joinIntsSep joins ints with an arbitrary separator.
func joinIntsSep(xs []int, sep string) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = itoa(x)
	}
	return strings.Join(parts, sep)
}
