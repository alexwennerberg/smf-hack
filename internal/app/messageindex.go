package app

// Port of Sources/MessageIndex.php.

import (
	"os"
	"regexp"
	"strings"
)

func init() {
	registerAction("*messageindex", (*Ctx).MessageIndex)
}

// MITopic is one row of $context['topics'].
type MITopic struct {
	ID         int
	FirstPost  MIPostInfo
	LastPost   MIPostInfo
	IsSticky   bool
	IsLocked   bool
	IsPoll     bool
	IsHot      bool
	IsVeryHot  bool
	IsPostedIn bool
	Icon       string
	IconURL    string
	Subject    string
	New        bool
	NewFrom    int
	NewHref    string
	Pages      string
	Replies    int
	Views      int
	Class      string
	Board      RecentRef // set by the Recent/Unread listings (topics span boards)
}

// MIPostInfo is first_post/last_post inside a topic row.
type MIPostInfo struct {
	ID         int
	MemberID   int
	Username   string
	Name       string
	MemberHref string
	MemberLink string
	Time       string
	Timestamp  int64
	Subject    string
	Preview    string
	Icon       string
	IconURL    string
	Href       string
	Link       string
}

// MessageIndexCtx is the page context for tpl_messageindex.go.
type MessageIndexCtx struct {
	Name           string
	Description    string
	PageIndex      string
	Start          int
	LinkModerators []string
	IsMarkedNotify bool
	CanMarkNotify  bool
	CanPostNew     bool
	CanPostPoll    bool
	Boards         []*BIBoard // child boards (same shape as board index)
	Topics         []*MITopic
	SortBy         string
	SortDirection  string
	NoTopicListing bool
}

var stripTagsRe = regexp.MustCompile(`<[^>]*>`)

// stripTags approximates PHP strip_tags for the generated previews (input
// here is parse_bbc output, which is well-formed).
func stripTags(s string) string {
	return stripTagsRe.ReplaceAllString(s, "")
}

// entityLen / entitySubstr are $func['strlen'] / $func['substr'].
func entityLen(s string) int { return len(entitySplit(s)) }

func entitySubstr(s string, start, length int) string {
	units := entitySplit(s)
	if start >= len(units) {
		return ""
	}
	end := start + length
	if end > len(units) {
		end = len(units)
	}
	return strings.Join(units[start:end], "")
}

// MessageIndex is MessageIndex() from MessageIndex.php.
func (c *Ctx) MessageIndex() {
	a := c.App
	scripturl := a.ScriptURL
	bi := c.BoardInfo
	page := &MessageIndexCtx{
		Name:        bi.Name,
		Description: bi.Description,
	}
	c.Page = page

	// View all the topics, or just a few?
	maxindex := a.SettingInt("defaultMaxTopics")
	if c.REQUEST.Has("all") && !a.SettingEmpty("enableAllMessages") {
		maxindex = bi.NumTopics
	}

	// We only know these.
	sortParam := c.REQUEST.Str("sort")
	knownSorts := map[string]bool{"subject": true, "starter": true, "last_poster": true,
		"replies": true, "views": true, "first_post": true, "last_post": true}
	if c.REQUEST.Has("sort") && !knownSorts[sortParam] {
		sortParam = "last_post"
		c.REQUEST.Set("sort", sortParam)
	}

	// Make sure the starting place makes sense and construct the page index.
	start := c.Start
	if c.REQUEST.Has("sort") {
		desc := ""
		if c.REQUEST.Has("desc") {
			desc = ";desc"
		}
		page.PageIndex, start = c.constructPageIndex(scripturl+"?board="+itoa(c.Board)+".%d;sort="+sortParam+desc, start, bi.NumTopics, maxindex, true)
	} else {
		page.PageIndex, start = c.constructPageIndex(scripturl+"?board="+itoa(c.Board)+".%d", start, bi.NumTopics, maxindex, true)
	}
	c.Start = start
	page.Start = start

	if c.REQUEST.Has("all") && !a.SettingEmpty("enableAllMessages") && maxindex > a.SettingInt("enableAllMessages") {
		maxindex = a.SettingInt("enableAllMessages")
		start = 0
		c.Start = 0
		page.Start = 0
	}

	// Build a list of the board's moderators.
	if len(bi.Moderators) > 0 {
		for _, mod := range bi.Moderators {
			page.LinkModerators = append(page.LinkModerators,
				`<a href="`+scripturl+"?action=profile;u="+itoa(mod.ID)+`" title="`+c.Txt("62")+`">`+mod.Name+`</a>`)
		}
		modTxt := c.Txt("299")
		if len(page.LinkModerators) == 1 {
			modTxt = c.Txt("298")
		}
		c.LinkTree[len(c.LinkTree)-1].ExtraAfter = ` (` + modTxt + `: ` + strings.Join(page.LinkModerators, ", ") + `)`
	}

	// Mark current and parent boards as seen.
	if !c.User.IsGuest {
		// We can't know they read it if we allow prefetches.
		if c.R.Header.Get("X-Moz") == "prefetch" {
			c.W.WriteHeader(403)
			c.exit()
		}

		a.DB.Exec(a.Q(`REPLACE INTO {$db_prefix}log_boards (ID_MSG, ID_MEMBER, ID_BOARD) VALUES (?, ?, ?)`),
			a.SettingInt("maxMsgID"), c.User.ID, c.Board)

		var sent int
		err := a.DB.QueryRow(a.Q(`SELECT sent FROM {$db_prefix}log_notify WHERE ID_BOARD = ? AND ID_MEMBER = ? LIMIT 1`),
			c.Board, c.User.ID).Scan(&sent)
		page.IsMarkedNotify = err == nil
		if page.IsMarkedNotify && sent != 0 {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_notify SET sent = 0 WHERE ID_BOARD = ? AND ID_MEMBER = ?`),
				c.Board, c.User.ID)
		}
	}

	// 'Print' the header and board info.
	c.PageTitle = stripTags(bi.Name)

	// Set the variables up for the template.
	page.CanMarkNotify = c.allowedTo("mark_notify") && !c.User.IsGuest
	page.CanPostNew = c.allowedTo("post_new")
	page.CanPostPoll = a.Setting("pollMode") == "1" && c.allowedTo("poll_post")

	// Aren't children wonderful things?
	c.loadChildBoards(page)

	// Default sort methods.
	sortMethods := map[string]string{
		"subject":     "mf.subject",
		"starter":     "IFNULL(memf.realName, mf.posterName)",
		"last_poster": "IFNULL(meml.realName, ml.posterName)",
		"replies":     "t.numReplies",
		"views":       "t.numViews",
		"first_post":  "t.ID_TOPIC",
		"last_post":   "t.ID_LAST_MSG",
	}

	var sortColumn string
	var ascending bool
	if col, ok := sortMethods[sortParam]; !ok || !c.REQUEST.Has("sort") {
		page.SortBy = "last_post"
		sortColumn = "ID_LAST_MSG"
		ascending = c.REQUEST.Has("asc")
	} else {
		page.SortBy = sortParam
		sortColumn = col
		ascending = !c.REQUEST.Has("desc")
	}
	page.SortDirection = "down"
	if ascending {
		page.SortDirection = "up"
	}

	// Calculate the fastest way to get the topics.
	fakeAscending := false
	if start > (bi.NumTopics-1)/2 {
		ascending = !ascending
		fakeAscending = true
		if bi.NumTopics < start+maxindex+1 {
			maxindex = bi.NumTopics - start
			start = 0
		} else {
			start = bi.NumTopics - start - maxindex
		}
	}

	// Setup the default topic icons...
	iconSources := map[string]string{}
	for _, icon := range []string{"xx", "thumbup", "thumbdown", "exclamation", "question", "lamp",
		"smiley", "angry", "cheesy", "grin", "sad", "wink", "moved", "recycled", "wireless"} {
		iconSources[icon] = "images_url"
	}

	stickyOrder := ""
	if !a.SettingEmpty("enableStickyTopics") {
		stickyOrder = "isSticky"
		if !fakeAscending {
			stickyOrder += " DESC"
		}
		stickyOrder += ", "
	}
	dir := ""
	if !ascending {
		dir = " DESC"
	}

	var topicIDs []int

	// Sequential pages are often not optimized, so we add an additional
	// query.
	preQuery := start > 0
	if preQuery {
		fromExtra, whereExtra := "", ""
		switch page.SortBy {
		case "last_poster":
			fromExtra = `, {$db_prefix}messages AS ml`
			whereExtra = `
				AND ml.ID_MSG = t.ID_LAST_MSG`
		case "starter", "subject":
			fromExtra = `, {$db_prefix}messages AS mf`
			whereExtra = `
				AND mf.ID_MSG = t.ID_FIRST_MSG`
		}
		joinExtra := ""
		if page.SortBy == "starter" {
			joinExtra = `
				LEFT JOIN {$db_prefix}members AS memf ON (memf.ID_MEMBER = mf.ID_MEMBER)`
		} else if page.SortBy == "last_poster" {
			joinExtra = `
				LEFT JOIN {$db_prefix}members AS meml ON (meml.ID_MEMBER = ml.ID_MEMBER)`
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT t.ID_TOPIC
			FROM {$db_prefix}topics AS t` + fromExtra + joinExtra + `
			WHERE t.ID_BOARD = ` + itoa(c.Board) + whereExtra + `
			ORDER BY ` + stickyOrder + sortColumn + dir + `
			LIMIT ` + itoa(maxindex) + ` OFFSET ` + itoa(start)))
		if err != nil {
			c.fatalDBError(err)
		}
		for rows.Next() {
			var id int
			rows.Scan(&id)
			topicIDs = append(topicIDs, id)
		}
		rows.Close()
	}

	// Grab the appropriate topic information...
	if !preQuery || len(topicIDs) > 0 {
		newFromCol := "0"
		memberJoins := ""
		if !c.User.IsGuest {
			newFromCol = "IFNULL(lt.ID_MSG, IFNULL(lmr.ID_MSG, -1)) + 1"
			memberJoins = `
				LEFT JOIN {$db_prefix}log_topics AS lt ON (lt.ID_TOPIC = t.ID_TOPIC AND lt.ID_MEMBER = ` + itoa(c.User.ID) + `)
				LEFT JOIN {$db_prefix}log_mark_read AS lmr ON (lmr.ID_BOARD = ` + itoa(c.Board) + ` AND lmr.ID_MEMBER = ` + itoa(c.User.ID) + `)`
		}
		var where, order, limit string
		if preQuery {
			ids := make([]string, len(topicIDs))
			for i, id := range topicIDs {
				ids[i] = itoa(id)
			}
			where = "t.ID_TOPIC IN (" + strings.Join(ids, ", ") + ")"
			order = "FIND_IN_SET(t.ID_TOPIC, '" + strings.Join(ids, ",") + "')"
			limit = itoa(maxindex)
		} else {
			where = "t.ID_BOARD = " + itoa(c.Board)
			order = stickyOrder + sortColumn + dir
			limit = itoa(maxindex) + " OFFSET " + itoa(start)
		}

		rows, err := a.DB.Query(a.Q(`
			SELECT
				t.ID_TOPIC, t.numReplies, t.locked, t.numViews, t.isSticky, t.ID_POLL,
				` + newFromCol + ` AS new_from,
				t.ID_LAST_MSG, ml.posterTime AS lastPosterTime, ml.ID_MSG_MODIFIED,
				ml.subject AS lastSubject, ml.icon AS lastIcon, ml.posterName AS lastMemberName,
				ml.ID_MEMBER AS lastID_MEMBER, IFNULL(meml.realName, ml.posterName) AS lastDisplayName,
				t.ID_FIRST_MSG, mf.posterTime AS firstPosterTime,
				mf.subject AS firstSubject, mf.icon AS firstIcon, mf.posterName AS firstMemberName,
				mf.ID_MEMBER AS firstID_MEMBER, IFNULL(memf.realName, mf.posterName) AS firstDisplayName,
				SUBSTR(ml.body, 1, 384) AS lastBody, SUBSTR(mf.body, 1, 384) AS firstBody, ml.smileysEnabled AS lastSmileys,
				mf.smileysEnabled AS firstSmileys
			FROM {$db_prefix}topics AS t, {$db_prefix}messages AS ml, {$db_prefix}messages AS mf
				LEFT JOIN {$db_prefix}members AS meml ON (meml.ID_MEMBER = ml.ID_MEMBER)
				LEFT JOIN {$db_prefix}members AS memf ON (memf.ID_MEMBER = mf.ID_MEMBER)` + memberJoins + `
			WHERE ` + where + `
				AND ml.ID_MSG = t.ID_LAST_MSG
				AND mf.ID_MSG = t.ID_FIRST_MSG
			ORDER BY ` + order + `
			LIMIT ` + limit))
		if err != nil {
			c.fatalDBError(err)
		}

		for rows.Next() {
			var idTopic, numReplies, locked, numViews, isSticky, idPoll, newFrom int
			var idLastMsg, idMsgModified, lastIDMember, idFirstMsg, firstIDMember int
			var lastPosterTime, firstPosterTime int64
			var lastSubject, lastIcon, lastMemberName, lastDisplayName string
			var firstSubject, firstIcon, firstMemberName, firstDisplayName string
			var lastBody, firstBody string
			var lastSmileys, firstSmileys int
			if err := rows.Scan(&idTopic, &numReplies, &locked, &numViews, &isSticky, &idPoll,
				&newFrom, &idLastMsg, &lastPosterTime, &idMsgModified,
				&lastSubject, &lastIcon, &lastMemberName, &lastIDMember, &lastDisplayName,
				&idFirstMsg, &firstPosterTime, &firstSubject, &firstIcon, &firstMemberName,
				&firstIDMember, &firstDisplayName, &lastBody, &firstBody,
				&lastSmileys, &firstSmileys); err != nil {
				rows.Close()
				c.fatalDBError(err)
			}

			if idPoll > 0 && a.Setting("pollMode") == "0" {
				continue
			}

			if !preQuery {
				topicIDs = append(topicIDs, idTopic)
			}

			// Limit them to 128 characters.
			firstBody = stripTags(strings.ReplaceAll(c.parseBBCCached(firstBody, firstSmileys != 0, itoa(idFirstMsg)), "<br />", "&#10;"))
			if entityLen(firstBody) > 128 {
				firstBody = entitySubstr(firstBody, 0, 128) + "..."
			}
			lastBody = stripTags(strings.ReplaceAll(c.parseBBCCached(lastBody, lastSmileys != 0, itoa(idLastMsg)), "<br />", "&#10;"))
			if entityLen(lastBody) > 128 {
				lastBody = entitySubstr(lastBody, 0, 128) + "..."
			}

			// Censor the subject and message preview.
			firstSubject = c.censorText(firstSubject)
			firstBody = c.censorText(firstBody)

			// Don't censor them twice!
			if idFirstMsg == idLastMsg {
				lastSubject = firstSubject
				lastBody = firstBody
			} else {
				lastSubject = c.censorText(lastSubject)
				lastBody = c.censorText(lastBody)
			}

			// Decide how many pages the topic should have.
			pages := ""
			topicLength := numReplies + 1
			maxMessages := a.SettingInt("defaultMaxMessages")
			if topicLength > maxMessages {
				var tmppages []string
				tmpa := 1
				for tmpb := 0; tmpb < topicLength; tmpb += maxMessages {
					tmppages = append(tmppages, `<a href="`+scripturl+"?topic="+itoa(idTopic)+"."+itoa(tmpb)+`">`+itoa(tmpa)+`</a>`)
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

			// We need to check the topic icons exist...
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
					MemberID:  firstIDMember,
					Username:  firstMemberName,
					Name:      firstDisplayName,
					Time:      c.timeformat(firstPosterTime),
					Timestamp: c.forumTime(true, firstPosterTime),
					Subject:   firstSubject,
					Preview:   firstBody,
					Icon:      firstIcon,
					IconURL:   c.Theme.Get(iconSources[firstIcon]) + "/post/" + firstIcon + ".gif",
					Href:      scripturl + "?topic=" + itoa(idTopic) + ".0",
					Link:      `<a href="` + scripturl + "?topic=" + itoa(idTopic) + `.0">` + firstSubject + `</a>`,
				},
				LastPost: MIPostInfo{
					ID:        idLastMsg,
					MemberID:  lastIDMember,
					Username:  lastMemberName,
					Name:      lastDisplayName,
					Time:      c.timeformat(lastPosterTime),
					Timestamp: c.forumTime(true, lastPosterTime),
					Subject:   lastSubject,
					Preview:   lastBody,
					Icon:      lastIcon,
					IconURL:   c.Theme.Get(iconSources[lastIcon]) + "/post/" + lastIcon + ".gif",
					Href:      scripturl + "?topic=" + itoa(idTopic) + lastHrefPart + "#new",
					Link:      `<a href="` + scripturl + "?topic=" + itoa(idTopic) + lastHrefPart + `#new">` + lastSubject + `</a>`,
				},
				IsSticky:  !a.SettingEmpty("enableStickyTopics") && isSticky != 0,
				IsLocked:  locked != 0,
				IsPoll:    a.Setting("pollMode") == "1" && idPoll > 0,
				IsHot:     numReplies >= a.SettingInt("hotTopicPosts"),
				IsVeryHot: numReplies >= a.SettingInt("hotTopicVeryPosts"),
				Icon:      firstIcon,
				IconURL:   c.Theme.Get(iconSources[firstIcon]) + "/post/" + firstIcon + ".gif",
				Subject:   firstSubject,
				New:       newFrom <= idMsgModified,
				NewFrom:   newFrom,
				NewHref:   scripturl + "?topic=" + itoa(idTopic) + ".msg" + itoa(newFrom) + "#new",
				Pages:     pages,
				Replies:   numReplies,
				Views:     numViews,
			}
			if firstIDMember != 0 {
				topic.FirstPost.MemberHref = scripturl + "?action=profile;u=" + itoa(firstIDMember)
				topic.FirstPost.MemberLink = `<a href="` + topic.FirstPost.MemberHref + `" title="` + c.Txt("92") + ` ` + firstDisplayName + `">` + firstDisplayName + `</a>`
			} else {
				topic.FirstPost.MemberLink = firstDisplayName
			}
			if lastIDMember != 0 {
				topic.LastPost.MemberHref = scripturl + "?action=profile;u=" + itoa(lastIDMember)
				topic.LastPost.MemberLink = `<a href="` + topic.LastPost.MemberHref + `">` + lastDisplayName + `</a>`
			} else {
				topic.LastPost.MemberLink = lastDisplayName
			}

			topic.Class = determineTopicClass(topic.IsVeryHot, topic.IsHot, topic.IsPoll, topic.IsLocked, topic.IsSticky)
			page.Topics = append(page.Topics, topic)
		}
		rows.Close()

		// Fix the sequence of topics if they were retrieved in the wrong
		// order. (for speed reasons...)
		if fakeAscending {
			for i, j := 0, len(page.Topics)-1; i < j; i, j = i+1, j-1 {
				page.Topics[i], page.Topics[j] = page.Topics[j], page.Topics[i]
			}
		}

		if !a.SettingEmpty("enableParticipation") && !c.User.IsGuest && len(topicIDs) > 0 {
			ids := make([]string, len(topicIDs))
			for i, id := range topicIDs {
				ids[i] = itoa(id)
			}
			rows, err := a.DB.Query(a.Q(`
				SELECT ID_TOPIC
				FROM {$db_prefix}messages
				WHERE ID_TOPIC IN (` + strings.Join(ids, ", ") + `)
					AND ID_MEMBER = ` + itoa(c.User.ID) + `
				GROUP BY ID_TOPIC
				LIMIT ` + itoa(len(topicIDs))))
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					for _, t := range page.Topics {
						if t.ID == id {
							t.IsPostedIn = true
							t.Class = "my_" + t.Class
						}
					}
				}
				rows.Close()
			}
		}
	}

	c.loadJumpTo()

	// (Quick Moderation [display_quick_mod theme option] arrives with
	// Phase 6's moderation port.)

	// If there are children, but no topics and no ability to post topics...
	page.NoTopicListing = len(page.Boards) > 0 && len(page.Topics) == 0 && !page.CanPostNew

	c.SubTemplate = templateMessageIndexMain
}

// loadChildBoards fills page.Boards with this board's children (the first
// child-board query in MessageIndex()).
func (c *Ctx) loadChildBoards(page *MessageIndexCtx) {
	a := c.App
	scripturl := a.ScriptURL

	isReadCol := "1 AS isRead"
	lbJoin := ""
	if !c.User.IsGuest {
		isReadCol = "(IFNULL(lb.ID_MSG, 0) >= b.ID_MSG_UPDATED) AS isRead"
		lbJoin = `
			LEFT JOIN {$db_prefix}log_boards AS lb ON (lb.ID_BOARD = b.ID_BOARD AND lb.ID_MEMBER = ` + itoa(c.User.ID) + `)`
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT
			b.ID_BOARD, b.name, b.description, b.numTopics, b.numPosts,
			IFNULL(m.posterName, '') AS posterName, IFNULL(m.posterTime, 0) AS posterTime, IFNULL(m.subject, '') AS subject,
			IFNULL(m.ID_MSG, 0) AS ID_MSG, IFNULL(m.ID_TOPIC, 0) AS ID_TOPIC,
			IFNULL(mem.realName, IFNULL(m.posterName, '')) AS realName, ` + isReadCol + `,
			IFNULL(mem.ID_MEMBER, 0) AS ID_MEMBER, IFNULL(mem2.ID_MEMBER, 0) AS ID_MODERATOR,
			IFNULL(mem2.realName, '') AS modRealName
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_MSG = b.ID_LAST_MSG)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)` + lbJoin + `
			LEFT JOIN {$db_prefix}moderators AS mods ON (mods.ID_BOARD = b.ID_BOARD)
			LEFT JOIN {$db_prefix}members AS mem2 ON (mem2.ID_MEMBER = mods.ID_MEMBER)
		WHERE b.ID_PARENT = ` + itoa(c.Board) + `
			AND ` + c.User.QuerySeeBoard))
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()

	byID := map[int]*BIBoard{}
	for rows.Next() {
		var idBoard, numTopics, numPosts, idMsg, idTopic, isRead, idMember, idModerator int
		var posterTime int64
		var name, description, posterName, subject, realName, modRealName string
		rows.Scan(&idBoard, &name, &description, &numTopics, &numPosts,
			&posterName, &posterTime, &subject, &idMsg, &idTopic,
			&realName, &isRead, &idMember, &idModerator, &modRealName)

		board, seen := byID[idBoard]
		if !seen {
			subject = c.censorText(subject)
			shortSubject := shortenSubject(subject, 24)

			lp := &BILastPost{
				ID:        idMsg,
				Timestamp: c.forumTime(true, posterTime),
				Subject:   shortSubject,
				Start:     "new",
				Topic:     idTopic,
				MemberID:  idMember,
			}
			if posterTime > 0 {
				lp.Time = c.timeformat(posterTime)
			} else {
				lp.Time = c.Txt("470")
			}
			if posterName != "" {
				lp.MemberName = realName
				if idMember != 0 {
					lp.MemberLink = `<a href="` + scripturl + "?action=profile;u=" + itoa(idMember) + `">` + realName + `</a>`
				} else {
					lp.MemberLink = realName
				}
			} else {
				lp.MemberName = c.Txt("470")
				lp.MemberLink = c.Txt("470")
			}
			seenPart := ""
			if isRead == 0 {
				seenPart = ";boardseen"
			}
			if subject != "" {
				lp.Href = scripturl + "?topic=" + itoa(idTopic) + ".new" + seenPart + "#new"
				lp.Link = `<a href="` + lp.Href + `" title="` + subject + `">` + shortSubject + `</a>`
			} else {
				lp.Link = c.Txt("470")
			}

			board = &BIBoard{
				ID:          idBoard,
				LastPost:    lp,
				New:         isRead == 0 && posterName != "",
				Name:        name,
				Description: description,
				Topics:      numTopics,
				Posts:       numPosts,
				Href:        scripturl + "?board=" + itoa(idBoard) + ".0",
				Link:        `<a href="` + scripturl + "?board=" + itoa(idBoard) + `.0">` + name + `</a>`,
			}
			byID[idBoard] = board
			page.Boards = append(page.Boards, board)
		}
		if idModerator != 0 && !boardHasModerator(board, idModerator) {
			board.Moderators = append(board.Moderators, Moderator{
				ID:   idModerator,
				Name: modRealName,
				Href: scripturl + "?action=profile;u=" + itoa(idModerator),
				Link: `<a href="` + scripturl + "?action=profile;u=" + itoa(idModerator) + `" title="` + c.Txt("62") + `">` + modRealName + `</a>`,
			})
			board.LinkModerators = append(board.LinkModerators,
				`<a href="`+scripturl+"?action=profile;u="+itoa(idModerator)+`" title="`+c.Txt("62")+`">`+modRealName+`</a>`)
		}
	}

	// (Grandchild boards roll-up for countChildPosts, and the second child
	// query, follow the same pattern; with the default settings the
	// child-of-child query only affects boards two levels deep. Ported when
	// a fixture needs it — flagged in PORTING.md.)
	if len(page.Boards) > 0 {
		c.loadGrandchildBoards(page)
	}
}

// loadGrandchildBoards is the "Load up the child boards" query (children of
// the children, for new-status and link_children).
func (c *Ctx) loadGrandchildBoards(page *MessageIndexCtx) {
	a := c.App
	scripturl := a.ScriptURL

	byID := map[int]*BIBoard{}
	var ids []string
	for _, b := range page.Boards {
		byID[b.ID] = b
		ids = append(ids, itoa(b.ID))
	}

	isReadCol := "1"
	lbJoin := ""
	if !c.User.IsGuest {
		isReadCol = "(IFNULL(lb.ID_MSG, 0) >= b.ID_MSG_UPDATED)"
		lbJoin = `
			LEFT JOIN {$db_prefix}log_boards AS lb ON (lb.ID_BOARD = b.ID_BOARD AND lb.ID_MEMBER = ` + itoa(c.User.ID) + `)`
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT
			b.ID_BOARD, b.ID_PARENT, b.name, b.description, b.numTopics, b.numPosts,
			IFNULL(m.posterName, '') AS posterName, IFNULL(m.posterTime, 0) AS posterTime, IFNULL(m.subject, '') AS subject,
			IFNULL(m.ID_MSG, 0), IFNULL(m.ID_TOPIC, 0),
			IFNULL(mem.realName, IFNULL(m.posterName, '')) AS realName,
			` + isReadCol + ` AS isRead,
			IFNULL(mem.ID_MEMBER, 0) AS ID_MEMBER
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}messages AS m ON (m.ID_MSG = b.ID_LAST_MSG)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)` + lbJoin + `
		WHERE b.ID_PARENT IN (` + strings.Join(ids, ",") + `)
			AND ` + c.User.QuerySeeBoard))
	if err != nil {
		c.fatalDBError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var idBoard, idParent, numTopics, numPosts, idMsg, idTopic, isRead, idMember int
		var posterTime int64
		var name, description, posterName, subject, realName string
		rows.Scan(&idBoard, &idParent, &name, &description, &numTopics, &numPosts,
			&posterName, &posterTime, &subject, &idMsg, &idTopic, &realName, &isRead, &idMember)

		parent, ok := byID[idParent]
		if !ok {
			continue
		}

		if parent.LastPost != nil && parent.LastPost.Timestamp < c.forumTime(true, posterTime) {
			subject = c.censorText(subject)
			shortSubject := shortenSubject(subject, 24)
			lp := &BILastPost{
				ID:        idMsg,
				Timestamp: c.forumTime(true, posterTime),
				Subject:   shortSubject,
				Start:     "new",
				Topic:     idTopic,
				MemberID:  idMember,
			}
			if posterTime > 0 {
				lp.Time = c.timeformat(posterTime)
			} else {
				lp.Time = c.Txt("470")
			}
			if posterName != "" {
				lp.MemberName = realName
				if idMember != 0 {
					lp.MemberLink = `<a href="` + scripturl + "?action=profile;u=" + itoa(idMember) + `">` + realName + `</a>`
				} else {
					lp.MemberLink = realName
				}
			} else {
				lp.MemberName = c.Txt("470")
				lp.MemberLink = c.Txt("470")
			}
			seenPart := ""
			if isRead == 0 {
				seenPart = ";boardseen"
			}
			lp.Href = scripturl + "?topic=" + itoa(idTopic) + ".new" + seenPart + "#new"
			if subject != "" {
				lp.Link = `<a href="` + lp.Href + `" title="` + subject + `">` + shortSubject + `</a>`
			} else {
				lp.Link = c.Txt("470")
			}
			parent.LastPost = lp
		}

		child := &BIBoard{
			ID:          idBoard,
			Name:        name,
			Description: description,
			New:         isRead == 0 && posterName != "",
			Topics:      numTopics,
			Posts:       numPosts,
			Href:        scripturl + "?board=" + itoa(idBoard) + ".0",
			Link:        `<a href="` + scripturl + "?board=" + itoa(idBoard) + `.0">` + name + `</a>`,
		}
		parent.Children = append(parent.Children, child)
		parent.LinkChildren = append(parent.LinkChildren, child.Link)
		parent.ChildrenNew = parent.ChildrenNew || (isRead == 0 && posterName != "")
	}
}
