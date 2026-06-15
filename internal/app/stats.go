package app

// The recount branches of updateStats() from Subs.php ('message', 'topic')
// and the posts-adjusting subset of updateMemberData() from Load.php, plus
// the DisplayStats controller (?action=stats) from Stats.php.

import (
	"math"
	"sort"
	"strings"
	"time"
)

// updateStatsMessage is updateStats('message'): recount totalMessages and
// maxMsgID from the boards table.
func (a *App) updateStatsMessage() {
	var totalMessages, maxMsgID int
	a.DB.QueryRow(a.Q(`
		SELECT IFNULL(SUM(numPosts), 0) AS totalMessages, IFNULL(MAX(ID_LAST_MSG), 0) AS maxMsgID
		FROM {$db_prefix}boards`)).Scan(&totalMessages, &maxMsgID)

	a.UpdateSettings(map[string]string{
		"totalMessages": itoa(totalMessages),
		"maxMsgID":      itoa(maxMsgID),
	})
}

// updateStatsTopic is updateStats('topic'): recount totalTopics from the
// boards table (ignoring the recycle bin).
func (a *App) updateStatsTopic() {
	cond := ""
	if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
		cond = `
		WHERE ID_BOARD != ` + itoa(a.SettingInt("recycle_board"))
	}
	var totalTopics int
	a.DB.QueryRow(a.Q(`
		SELECT IFNULL(SUM(numTopics), 0) AS totalTopics
		FROM {$db_prefix}boards` + cond)).Scan(&totalTopics)

	a.UpdateSettings(map[string]string{"totalTopics": itoa(totalTopics)})
}

// updateMemberPosts applies updateMemberData($id, array('posts' => ...))
// deltas; PHP guards posts from going negative via the unsigned column, we
// guard explicitly.
func (a *App) updateMemberPosts(memberID, delta int) {
	if delta >= 0 {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET posts = posts + ? WHERE ID_MEMBER = ?`),
			delta, memberID)
	} else {
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET posts = MAX(0, posts - ?) WHERE ID_MEMBER = ?`),
			-delta, memberID)
	}
}

func init() {
	registerAction("stats", (*Ctx).DisplayStats)
}

// StatsTop is one entry in a top-10 list (posters/boards/topics/starters/time).
type StatsTop struct {
	ID          int
	Name        string
	Link        string
	Href        string
	Board       *StatsBoard
	Subject     string
	NumPosts    int
	NumTopics   int
	NumReplies  int
	NumViews    int
	TimeOnline  string
	PostPercent int
	TimePercent int
}

// StatsBoard is the board reference inside a top-topic entry.
type StatsBoard struct {
	ID   int
	Name string
	Href string
	Link string
}

// StatsDay is one day's row in an expanded month.
type StatsDay struct {
	Day, Month        string
	Year              int
	NewTopics         int
	NewPosts          int
	NewMembers        int
	MostMembersOnline int
	Hits              int
}

// StatsMonth is one month's row in the activity history.
type StatsMonth struct {
	ID                string // YYYYMM
	DateMonth         string // MM
	DateYear          int
	Href              string
	Link              string
	Month             string // month name
	Year              int
	NewTopics         int
	NewPosts          int
	NewMembers        int
	MostMembersOnline int
	Hits              int
	NumDays           int
	Days              []*StatsDay
	Expanded          bool
}

// StatsCtx is the page context for the Stats templates.
type StatsCtx struct {
	ShowMemberList bool
	NumMembers     string
	NumPosts       string
	NumTopics      string
	NumCategories  int
	NumBoards      int
	UsersOnline    int
	OnlineToday    int
	NumHits        int
	MostOnlineNum  string
	MostOnlineDate string
	AveragePosts   string
	AverageTopics  string
	AverageMembers string
	AverageOnline  string
	AverageHits    string
	GenderRatio    string

	TopPosters       []*StatsTop
	TopBoards        []*StatsTop
	TopTopicsReplies []*StatsTop
	TopTopicsViews   []*StatsTop
	TopStarters      []*StatsTop
	TopTimeOnline    []*StatsTop

	Monthly     []*StatsMonth
	monthlyByID map[string]*StatsMonth
}

// statsExpanded reads $_SESSION['expanded_stats'] as year(string)->[]month.
func statsExpanded(c *Ctx) map[string][]int {
	out := map[string][]int{}
	if v, ok := c.Session.Get("expanded_stats").(map[string]any); ok {
		for y, list := range v {
			if arr, ok := list.([]any); ok {
				for _, m := range arr {
					out[y] = append(out[y], int(toFloat(m)))
				}
			}
		}
	}
	return out
}

// statsSetExpanded writes the expanded-stats map back to the session.
func statsSetExpanded(c *Ctx, exp map[string][]int) {
	m := map[string]any{}
	for y, months := range exp {
		var a []any
		for _, mo := range months {
			a = append(a, mo)
		}
		m[y] = a
	}
	c.Session.Set("expanded_stats", m)
}

// statsIsExpanded reports whether month is expanded in year.
func statsIsExpanded(exp map[string][]int, year, month int) bool {
	for _, m := range exp[itoa(year)] {
		if m == month {
			return true
		}
	}
	return false
}

// DisplayStats is DisplayStats(): the forum statistics page (?action=stats).
func (c *Ctx) DisplayStats() {
	a := c.App
	scripturl := a.ScriptURL

	exp := statsExpanded(c)

	var year, month int
	if e := c.REQUEST.Str("expand"); e != "" {
		month = atoi(substr(e, 4, -1))
		year = atoi(substr(e, 0, 4))
		if year > 1900 && year < 2200 && month >= 1 && month <= 12 {
			exp[itoa(year)] = append(exp[itoa(year)], month)
			statsSetExpanded(c, exp)
		}
	} else if col := c.REQUEST.Str("collapse"); col != "" {
		month = atoi(substr(col, 4, -1))
		year = atoi(substr(col, 0, 4))
		if len(exp[itoa(year)]) != 0 {
			var kept []int
			for _, m := range exp[itoa(year)] {
				if m != month {
					kept = append(kept, m)
				}
			}
			exp[itoa(year)] = kept
			statsSetExpanded(c, exp)
		}
	}

	page := &StatsCtx{monthlyByID: map[string]*StatsMonth{}}
	c.Page = page

	// Handle the XMLHttpRequest.
	if c.REQUEST.Has("xml") {
		// Collapsing stats only needs adjustments of the session variables.
		if c.REQUEST.Str("collapse") != "" {
			c.exit()
		}

		c.SubTemplate = templateStatsXML
		c.getDailyStats(page, "YEAR(date) = "+itoa(year)+" AND MONTH(date) = "+itoa(month))
		id := itoa(year) + sprintf02(month)
		if m, ok := page.monthlyByID[id]; ok {
			m.DateMonth = sprintf02(month)
			m.DateYear = year
		} else {
			m := &StatsMonth{ID: id, DateMonth: sprintf02(month), DateYear: year}
			page.monthlyByID[id] = m
			page.Monthly = append(page.Monthly, m)
		}
		return
	}

	c.loadLanguage("Stats")
	c.SubTemplate = templateStatsMain

	c.isAllowedTo("view_stats")

	// Build the link tree......
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=stats", Name: c.Txt("smf_stats_1")})
	c.PageTitle = a.Config.MbName + " - " + c.Txt("smf_stats_1")

	page.ShowMemberList = c.allowedTo("view_mlist")

	// Get averages...
	var sumPosts, sumTopics, sumRegisters, sumMostOn, sumHits int
	var minDate string
	a.DB.QueryRow(a.Q(`
		SELECT
			IFNULL(SUM(posts), 0) AS posts, IFNULL(SUM(topics), 0) AS topics, IFNULL(SUM(registers), 0) AS registers,
			IFNULL(SUM(mostOn), 0) AS mostOn, IFNULL(MIN(date), '') AS date, IFNULL(SUM(hits), 0) AS hits
		FROM {$db_prefix}log_activity`)).Scan(&sumPosts, &sumTopics, &sumRegisters, &sumMostOn, &minDate, &sumHits)

	// This would be the amount of time the forum has been up... in days...
	startTS := time.Now().Unix()
	if t, err := time.Parse("2006-01-02", minDate); err == nil {
		startTS = t.Unix()
	}
	totalDaysUp := int(math.Ceil(float64(time.Now().Unix()-startTS) / (60 * 60 * 24)))
	if totalDaysUp < 1 {
		totalDaysUp = 1
	}

	page.AveragePosts = phpRound(float64(sumPosts)/float64(totalDaysUp), 2)
	page.AverageTopics = phpRound(float64(sumTopics)/float64(totalDaysUp), 2)
	page.AverageMembers = phpRound(float64(sumRegisters)/float64(totalDaysUp), 2)
	page.AverageOnline = phpRound(float64(sumMostOn)/float64(totalDaysUp), 2)
	page.AverageHits = phpRound(float64(sumHits)/float64(totalDaysUp), 2)
	page.NumHits = sumHits

	// How many users are online now.
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_online`)).Scan(&page.UsersOnline)

	// Statistics such as number of boards, categories, etc.
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}boards AS b`)).Scan(&page.NumBoards)
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}categories AS c`)).Scan(&page.NumCategories)

	page.NumMembers = a.Setting("totalMembers")
	page.NumPosts = a.Setting("totalMessages")
	page.NumTopics = a.Setting("totalTopics")
	page.MostOnlineNum = a.Setting("mostOnline")
	page.MostOnlineDate = c.timeformat(int64(a.SettingInt("mostDate")))

	// Male vs. female ratio.
	var males, females int
	grows, err := a.DB.Query(a.Q(`
		SELECT COUNT(*) AS totalMembers, gender
		FROM {$db_prefix}members
		GROUP BY gender`))
	if err == nil {
		for grows.Next() {
			var total, gender int
			grows.Scan(&total, &gender)
			if gender != 0 {
				if gender == 2 {
					females = total
				} else {
					males = total
				}
			}
		}
		grows.Close()
	}
	switch {
	case males == females:
		page.GenderRatio = "1:1"
	case males == 0:
		page.GenderRatio = "0:1"
	case females == 0:
		page.GenderRatio = "1:0"
	case males > females:
		page.GenderRatio = phpRound(float64(males)/float64(females), 1) + ":1"
	default:
		page.GenderRatio = "1:" + phpRound(float64(females)/float64(males), 1)
	}

	// Members online so far today.
	today := time.Unix(c.forumTime(false, 0), 0).Format("2006-01-02")
	a.DB.QueryRow(a.Q(`
		SELECT mostOn
		FROM {$db_prefix}log_activity
		WHERE date = ?
		LIMIT 1`), today).Scan(&page.OnlineToday)

	recycleCond := ""
	if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
		recycleCond = "\n\t\t\tAND b.ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
	}

	// Poster top 10.
	page.TopPosters = c.statsTopMembers(`
		SELECT ID_MEMBER, realName, posts
		FROM {$db_prefix}members
		WHERE posts > 0
		ORDER BY posts DESC
		LIMIT 10`, "posts")

	// Board top 10.
	maxBoardPosts := 1
	brows, err := a.DB.Query(a.Q(`
		SELECT ID_BOARD, name, numPosts
		FROM {$db_prefix}boards AS b
		WHERE ` + c.User.QuerySeeBoard + recycleCond + `
		ORDER BY numPosts DESC
		LIMIT 10`))
	if err == nil {
		for brows.Next() {
			var id, numPosts int
			var name string
			brows.Scan(&id, &name, &numPosts)
			page.TopBoards = append(page.TopBoards, &StatsTop{
				ID:       id,
				Name:     name,
				NumPosts: numPosts,
				Href:     scripturl + "?board=" + itoa(id) + ".0",
				Link:     `<a href="` + scripturl + `?board=` + itoa(id) + `.0">` + name + `</a>`,
			})
			if maxBoardPosts < numPosts {
				maxBoardPosts = numPosts
			}
		}
		brows.Close()
	}
	for _, b := range page.TopBoards {
		b.PostPercent = phpRoundInt(float64(b.NumPosts*100) / float64(maxBoardPosts))
	}

	// Topic replies / views top 10.
	page.TopTopicsReplies = c.statsTopTopics("t.numReplies", recycleCond, "replies")
	page.TopTopicsViews = c.statsTopTopics("t.numViews", recycleCond, "views")

	// Topic starter top 10.
	page.TopStarters = c.statsTopStarters(recycleCond)

	// Time online top 10.
	page.TopTimeOnline = c.statsTopTimeOnline()

	// Activity by month.
	mrows, err := a.DB.Query(a.Q(`
		SELECT
			YEAR(date) AS stats_year, MONTH(date) AS stats_month, SUM(hits) AS hits, SUM(registers) AS registers, SUM(topics) AS topics, SUM(posts) AS posts, MAX(mostOn) AS mostOn, COUNT(*) AS numDays
		FROM {$db_prefix}log_activity
		GROUP BY stats_year, stats_month`))
	if err == nil {
		for mrows.Next() {
			var statsYear, statsMonth, hits, registers, topics, posts, mostOn, numDays int
			mrows.Scan(&statsYear, &statsMonth, &hits, &registers, &topics, &posts, &mostOn, &numDays)
			id := itoa(statsYear) + sprintf02(statsMonth)
			expanded := statsIsExpanded(exp, statsYear, statsMonth)
			ec := "expand"
			if expanded {
				ec = "collapse"
			}
			m := &StatsMonth{
				ID:                id,
				DateMonth:         sprintf02(statsMonth),
				DateYear:          statsYear,
				Href:              scripturl + "?action=stats;" + ec + "=" + id + "#" + id,
				Link:              `<a href="` + scripturl + `?action=stats;` + ec + `=` + id + `#` + id + `">` + c.TxtListItem("months", statsMonth) + ` ` + itoa(statsYear) + `</a>`,
				Month:             c.TxtListItem("months", statsMonth),
				Year:              statsYear,
				NewTopics:         topics,
				NewPosts:          posts,
				NewMembers:        registers,
				MostMembersOnline: mostOn,
				Hits:              hits,
				NumDays:           numDays,
				Expanded:          expanded,
			}
			page.monthlyByID[id] = m
			page.Monthly = append(page.Monthly, m)
		}
		mrows.Close()
	}

	// krsort: descending by id.
	sort.Slice(page.Monthly, func(i, j int) bool { return page.Monthly[i].ID > page.Monthly[j].ID })

	if len(exp) == 0 {
		return
	}

	var conditions []string
	for year, months := range exp {
		if len(months) != 0 {
			ms := make([]string, len(months))
			for i, m := range months {
				ms[i] = itoa(m)
			}
			conditions = append(conditions, "YEAR(date) = "+year+" AND MONTH(date) IN ("+strings.Join(ms, ", ")+")")
		}
	}
	// No daily stats to even look at?
	if len(conditions) == 0 {
		return
	}
	sort.Strings(conditions) // deterministic OR ordering
	c.getDailyStats(page, strings.Join(conditions, " OR "))
}

// statsTopMembers runs a "top members by N" query (posters).
func (c *Ctx) statsTopMembers(query, _ string) []*StatsTop {
	a := c.App
	scripturl := a.ScriptURL
	var out []*StatsTop
	maxNum := 1
	rows, err := a.DB.Query(a.Q(query))
	if err == nil {
		for rows.Next() {
			var id, posts int
			var name string
			rows.Scan(&id, &name, &posts)
			out = append(out, &StatsTop{
				ID:       id,
				Name:     name,
				NumPosts: posts,
				Href:     scripturl + "?action=profile;u=" + itoa(id),
				Link:     `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + name + `</a>`,
			})
			if maxNum < posts {
				maxNum = posts
			}
		}
		rows.Close()
	}
	for _, p := range out {
		p.PostPercent = phpRoundInt(float64(p.NumPosts*100) / float64(maxNum))
	}
	return out
}

// statsTopTopics runs the top-topics-by-replies/views query.
func (c *Ctx) statsTopTopics(orderCol, recycleCond, kind string) []*StatsTop {
	a := c.App
	scripturl := a.ScriptURL

	// Limit the topics scanned on large forums, as PHP does.
	topicIn := ""
	if a.SettingInt("totalMessages") > 100000 {
		col := "numReplies"
		if kind == "views" {
			col = "numViews"
		}
		var ids []string
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_TOPIC
			FROM {$db_prefix}topics
			WHERE ` + col + ` != 0
			ORDER BY ` + col + ` DESC
			LIMIT 100`))
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				ids = append(ids, itoa(id))
			}
			rows.Close()
		}
		if len(ids) != 0 {
			topicIn = "\n\t\t\tAND t.ID_TOPIC IN (" + strings.Join(ids, ", ") + ")"
		}
	}

	var out []*StatsTop
	maxNum := 1
	rows, err := a.DB.Query(a.Q(`
		SELECT m.subject, ` + orderCol + ` AS num, t.ID_BOARD, t.ID_TOPIC, b.name
		FROM ({$db_prefix}topics AS t, {$db_prefix}messages AS m, {$db_prefix}boards AS b)
		WHERE m.ID_MSG = t.ID_FIRST_MSG
			AND ` + c.User.QuerySeeBoard + recycleCond + `
			AND t.ID_BOARD = b.ID_BOARD` + topicIn + `
		ORDER BY ` + orderCol + ` DESC
		LIMIT 10`))
	if err == nil {
		for rows.Next() {
			var subject, boardName string
			var num, boardID, topicID int
			rows.Scan(&subject, &num, &boardID, &topicID, &boardName)
			subject = c.censorText(subject)
			t := &StatsTop{
				ID: topicID,
				Board: &StatsBoard{
					ID:   boardID,
					Name: boardName,
					Href: scripturl + "?board=" + itoa(boardID) + ".0",
					Link: `<a href="` + scripturl + `?board=` + itoa(boardID) + `.0">` + boardName + `</a>`,
				},
				Subject: subject,
				Href:    scripturl + "?topic=" + itoa(topicID) + ".0",
				Link:    `<a href="` + scripturl + `?topic=` + itoa(topicID) + `.0">` + subject + `</a>`,
			}
			if kind == "views" {
				t.NumViews = num
			} else {
				t.NumReplies = num
			}
			out = append(out, t)
			if maxNum < num {
				maxNum = num
			}
		}
		rows.Close()
	}
	for _, t := range out {
		n := t.NumReplies
		if kind == "views" {
			n = t.NumViews
		}
		t.PostPercent = phpRoundInt(float64(n*100) / float64(maxNum))
	}
	return out
}

// statsTopStarters runs the topic-starter top-10 query.
func (c *Ctx) statsTopStarters(recycleCond string) []*StatsTop {
	a := c.App
	scripturl := a.ScriptURL

	// Count topics started per member.
	starterRecycle := ""
	if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
		starterRecycle = "\n\t\t\tWHERE ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
	}
	counts := map[int]int{}
	var order []int
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER_STARTED, COUNT(*) AS hits
		FROM {$db_prefix}topics` + starterRecycle + `
		GROUP BY ID_MEMBER_STARTED
		ORDER BY hits DESC
		LIMIT 20`))
	if err == nil {
		for rows.Next() {
			var id, hits int
			rows.Scan(&id, &hits)
			counts[id] = hits
			order = append(order, id)
		}
		rows.Close()
	}
	if len(order) == 0 {
		order = []int{0}
		counts[0] = 0
	}

	ids := make([]string, len(order))
	for i, id := range order {
		ids[i] = itoa(id)
	}
	in := strings.Join(ids, ", ")

	var out []*StatsTop
	maxNum := 1
	mrows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, realName
		FROM {$db_prefix}members
		WHERE ID_MEMBER IN (` + in + `)
		ORDER BY FIND_IN_SET(ID_MEMBER, '` + strings.Join(ids, ",") + `')
		LIMIT 10`))
	if err == nil {
		for mrows.Next() {
			var id int
			var name string
			mrows.Scan(&id, &name)
			out = append(out, &StatsTop{
				ID:        id,
				Name:      name,
				NumTopics: counts[id],
				Href:      scripturl + "?action=profile;u=" + itoa(id),
				Link:      `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + name + `</a>`,
			})
			if maxNum < counts[id] {
				maxNum = counts[id]
			}
		}
		mrows.Close()
	}
	for _, t := range out {
		t.PostPercent = phpRoundInt(float64(t.NumTopics*100) / float64(maxNum))
	}
	return out
}

// statsTopTimeOnline runs the time-online top-10 query.
func (c *Ctx) statsTopTimeOnline() []*StatsTop {
	a := c.App
	scripturl := a.ScriptURL

	var out []*StatsTop
	maxTime := 1
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, realName, totalTimeLoggedIn
		FROM {$db_prefix}members
		ORDER BY totalTimeLoggedIn DESC
		LIMIT 20`))
	if err == nil {
		for rows.Next() {
			var id, total int
			var name string
			rows.Scan(&id, &name, &total)
			if len(out) >= 10 {
				continue
			}
			timeDays := total / 86400
			timeHours := (total % 86400) / 3600
			timelogged := ""
			if timeDays > 0 {
				timelogged += itoa(timeDays) + c.Txt("totalTimeLogged5")
			}
			if timeHours > 0 {
				timelogged += itoa(timeHours) + c.Txt("totalTimeLogged6")
			}
			timelogged += itoa((total%3600)/60) + c.Txt("totalTimeLogged7")
			out = append(out, &StatsTop{
				ID:         id,
				Name:       name,
				TimeOnline: timelogged,
				NumPosts:   total, // seconds_online, reused for the percent calc
				Href:       scripturl + "?action=profile;u=" + itoa(id),
				Link:       `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + name + `</a>`,
			})
			if maxTime < total {
				maxTime = total
			}
		}
		rows.Close()
	}
	for _, t := range out {
		t.TimePercent = phpRoundInt(float64(t.NumPosts*100) / float64(maxTime))
	}
	return out
}

// getDailyStats is getDailyStats($condition): expand a month into days.
func (c *Ctx) getDailyStats(page *StatsCtx, condition string) {
	a := c.App
	rows, err := a.DB.Query(a.Q(`
		SELECT YEAR(date) AS stats_year, MONTH(date) AS stats_month, DAYOFMONTH(date) AS stats_day, topics, posts, registers, mostOn, hits
		FROM {$db_prefix}log_activity
		WHERE ` + condition + `
		ORDER BY stats_day ASC`))
	if err != nil {
		return
	}
	for rows.Next() {
		var statsYear, statsMonth, statsDay, topics, posts, registers, mostOn, hits int
		rows.Scan(&statsYear, &statsMonth, &statsDay, &topics, &posts, &registers, &mostOn, &hits)
		id := itoa(statsYear) + sprintf02(statsMonth)
		m, ok := page.monthlyByID[id]
		if !ok {
			m = &StatsMonth{ID: id}
			page.monthlyByID[id] = m
			page.Monthly = append(page.Monthly, m)
		}
		m.Days = append(m.Days, &StatsDay{
			Day:               sprintf02(statsDay),
			Month:             sprintf02(statsMonth),
			Year:              statsYear,
			NewTopics:         topics,
			NewPosts:          posts,
			NewMembers:        registers,
			MostMembersOnline: mostOn,
			Hits:              hits,
		})
	}
	rows.Close()
}

// phpRoundInt is round($v) with no precision (nearest integer).
func phpRoundInt(v float64) int {
	return int(math.Floor(v + 0.5))
}

// sprintf02 is sprintf('%02d', n).
func sprintf02(n int) string {
	if n < 10 && n >= 0 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
