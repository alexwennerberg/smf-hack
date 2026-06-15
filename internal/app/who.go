package app

// Port of Sources/Who.php: the "Who's Online" listing (?action=who) and
// determineActions(), which turns each logged request into a human-readable
// description. The logged URL data is JSON in this port (see writeLog), so
// what PHP unserialize()s we json.Unmarshal.

import (
	"encoding/json"
	"sort"
	"strings"
)

func init() {
	registerAction("who", (*Ctx).Who)
}

// WhoMember is one row in the who's-online table.
type WhoMember struct {
	Member   *MemberCtx
	IP       string
	Time     string
	Action   string
	IsHidden bool
	Color    string
}

// WhoCtx is the page context for the Who template.
type WhoCtx struct {
	SortBy        string
	SortDirection string
	Start         int
	PageIndex     string
	CanSendPM     bool
	Members       []*WhoMember
}

// whoURL pairs a logged (JSON) URL with the member who made the request.
type whoURL struct {
	URL      string
	MemberID int
}

// Who is Who(): the who's-online page.
func (c *Ctx) Who() {
	a := c.App
	scripturl := a.ScriptURL

	c.isAllowedTo("who_view")

	// You can't do anything if this is off.
	if a.SettingEmpty("who_enabled") {
		c.fatalLangError("who_off", false)
	}

	page := &WhoCtx{}
	c.Page = page

	sortMethods := map[string]string{"user": "mem.realName", "time": "lo.logTime"}

	sortExpr := ""
	if s := c.REQUEST.Str("sort"); s == "" || sortMethods[s] == "" {
		page.SortBy = "time"
		sortExpr = "lo.logTime"
	} else {
		page.SortBy = s
		sortExpr = sortMethods[s]
	}

	page.SortDirection = "down"
	if c.REQUEST.Has("asc") {
		page.SortDirection = "up"
	}

	showOnlineGuard := ""
	if !c.allowedTo("moderate_forum") {
		showOnlineGuard = "\n\t\tWHERE IFNULL(mem.showOnline, 1) = 1"
	}

	// Get the total amount of members online.
	var totalMembers int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}log_online AS lo
			LEFT JOIN {$db_prefix}members AS mem ON (lo.ID_MEMBER = mem.ID_MEMBER)` + showOnlineGuard)).Scan(&totalMembers)

	ascParam := ""
	if c.REQUEST.Has("asc") {
		ascParam = ";asc"
	}
	page.Start = c.REQUEST.Int("start")
	page.PageIndex, page.Start = c.constructPageIndex(scripturl+"?action=who;sort="+page.SortBy+ascParam,
		page.Start, totalMembers, a.SettingInt("defaultMaxMembers"), false)

	orderDir := "DESC"
	if c.REQUEST.Has("asc") {
		orderDir = "ASC"
	}

	type whoRow struct {
		logTime     int64
		idMember    int
		url         string
		ip          string
		realName    string
		session     string
		onlineColor string
		showOnline  int
	}
	var rowsData []whoRow
	var memberIDs []int
	urlData := map[string]whoURL{}

	rows, err := a.DB.Query(a.Q(`
		SELECT
			lo.logTime AS logTime,
			lo.ID_MEMBER, lo.url, INET_NTOA(lo.ip) AS ip, IFNULL(mem.realName, '') AS realName, lo.session,
			IFNULL(mg.onlineColor, '') AS onlineColor, IFNULL(mem.showOnline, 1) AS showOnline
		FROM {$db_prefix}log_online AS lo
			LEFT JOIN {$db_prefix}members AS mem ON (lo.ID_MEMBER = mem.ID_MEMBER)
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))` + showOnlineGuard + `
		ORDER BY ` + sortExpr + ` ` + orderDir + `
		LIMIT ` + itoa(a.SettingInt("defaultMaxMembers")) + ` OFFSET ` + itoa(page.Start)))
	if err == nil {
		for rows.Next() {
			var r whoRow
			rows.Scan(&r.logTime, &r.idMember, &r.url, &r.ip, &r.realName, &r.session, &r.onlineColor, &r.showOnline)
			// @unserialize: skip rows whose URL data won't decode.
			var probe map[string]string
			if json.Unmarshal([]byte(r.url), &probe) != nil {
				continue
			}
			rowsData = append(rowsData, r)
			urlData[r.session] = whoURL{URL: r.url, MemberID: r.idMember}
			memberIDs = append(memberIDs, r.idMember)
		}
		rows.Close()
	}

	// Load the user data for these members.
	c.loadMemberData(memberIDs)

	// Load up the guest user.
	if c.memberCtx == nil {
		c.memberCtx = map[int]*MemberCtx{}
	}
	c.memberCtx[0] = &MemberCtx{
		ID:      0,
		Name:    c.Txt("28"),
		Group:   c.Txt("28"),
		Href:    "",
		Link:    c.Txt("28"),
		Email:   c.Txt("28"),
		IsGuest: true,
		Options: map[string]string{},
	}

	actions := c.determineActions(urlData)

	// Setup the linktree and page title.
	c.PageTitle = c.Txt("who_title")
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=who", Name: c.Txt("who_title")})

	todayPrefix := c.Txt("smf10")
	yesterdayPrefix := c.Txt("smf10b")

	for _, r := range rowsData {
		id := r.idMember
		var member *MemberCtx
		if id != 0 && c.loadMemberContext(id) {
			member = c.memberCtx[id]
		} else {
			member = c.memberCtx[0]
		}

		ip := ""
		if c.allowedTo("moderate_forum") {
			ip = r.ip
		}
		action, ok := actions[r.session]
		if !ok {
			action = c.Txt("who_hidden")
		}

		page.Members = append(page.Members, &WhoMember{
			Member:   member,
			IP:       ip,
			Time:     strings.NewReplacer(todayPrefix, "", yesterdayPrefix, "").Replace(c.timeformat(r.logTime)),
			Action:   action,
			IsHidden: r.showOnline == 0,
			Color:    r.onlineColor,
		})
	}

	page.CanSendPM = c.allowedTo("pm_send")
	c.SubTemplate = templateWhoMain
}

// whoAllowedActions maps actions to the permission(s) needed to see them.
var whoAllowedActions = map[string][]string{
	"admin":             {"moderate_forum", "manage_membergroups", "manage_bans", "admin_forum", "manage_permissions", "send_mail", "manage_attachments", "manage_smileys", "manage_boards", "edit_news"},
	"ban":               {"manage_bans"},
	"boardrecount":      {"admin_forum"},
	"calendar":          {"calendar_view"},
	"editnews":          {"edit_news"},
	"mailing":           {"send_mail"},
	"maintain":          {"admin_forum"},
	"manageattachments": {"manage_attachments"},
	"manageboards":      {"manage_boards"},
	"mlist":             {"view_mlist"},
	"optimizetables":    {"admin_forum"},
	"repairboards":      {"admin_forum"},
	"search":            {"search_posts"},
	"search2":           {"search_posts"},
	"setcensor":         {"moderate_forum"},
	"setreserve":        {"moderate_forum"},
	"stats":             {"view_stats"},
	"viewErrorLog":      {"admin_forum"},
	"viewmembers":       {"moderate_forum"},
}

// determineActions is determineActions($urls): build the action description
// for each session key.
func (c *Ctx) determineActions(urls map[string]whoURL) map[string]string {
	a := c.App

	data := map[string]string{}
	if !c.allowedTo("who_view") {
		return data
	}
	c.loadLanguage("Who")

	// Queue up topics/boards/profiles to resolve in batches: id -> session -> template.
	topicIDs := map[int]map[string]string{}
	boardIDs := map[int]map[string]string{}
	profileIDs := map[int]map[string]string{}

	// Deterministic order so the batch IN() clauses are stable.
	keys := make([]string, 0, len(urls))
	for k := range urls {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		url := urls[k]
		var actions map[string]string
		if json.Unmarshal([]byte(url.URL), &actions) != nil {
			continue
		}

		action := actions["action"]
		switch {
		case !hasKey(actions, "action") || action == "display":
			// No action (or display): a topic, a board, or the board index.
			if hasKey(actions, "topic") {
				data[k] = c.Txt("who_hidden")
				addWho(topicIDs, atoi(actions["topic"]), k, c.Txt("who_topic"))
			} else if hasKey(actions, "board") {
				data[k] = c.Txt("who_hidden")
				addWho(boardIDs, atoi(actions["board"]), k, c.Txt("who_board"))
			} else {
				data[k] = c.Txt("who_index")
			}
		case action == "":
			// Probably an error or some goon?
			data[k] = c.Txt("who_index")
		default:
			switch {
			case action == "profile" || action == "profile2":
				u := atoi(actions["u"])
				if u == 0 {
					u = url.MemberID
				}
				data[k] = c.Txt("who_hidden")
				label := c.Txt("who_profile")
				if action == "profile" {
					label = c.Txt("who_viewprofile")
				}
				addWho(profileIDs, u, k, label)
			case (action == "post" || action == "post2") && !hasKey(actions, "topic") && hasKey(actions, "board"):
				data[k] = c.Txt("who_hidden")
				label := c.Txt("who_post")
				if hasKey(actions, "poll") {
					label = c.Txt("who_poll")
				}
				addWho(boardIDs, atoi(actions["board"]), k, label)
			case hasKey(actions, "sa") && c.txtExists("whoall_"+action+"_"+actions["sa"]):
				data[k] = c.Txt("whoall_" + action + "_" + actions["sa"])
			case c.txtExists("whoall_" + action):
				data[k] = c.Txt("whoall_" + action)
			case c.txtExists("whotopic_" + action):
				topic := atoi(actions["topic"])
				if topic == 0 {
					topic = atoi(actions["from"])
				}
				data[k] = c.Txt("who_hidden")
				addWho(topicIDs, topic, k, c.Txt("whotopic_"+action))
			case c.txtExists("whopost_" + action):
				msgid := atoi(actions["msg"])
				if msgid == 0 {
					msgid = atoi(actions["quote"])
				}
				var idTopic int
				var subject string
				a.DB.QueryRow(a.Q(`
					SELECT m.ID_TOPIC, m.subject
					FROM ({$db_prefix}boards AS b, {$db_prefix}messages AS m)
					WHERE `+c.User.QuerySeeBoard+`
						AND m.ID_MSG = ?
						AND m.ID_BOARD = b.ID_BOARD
					LIMIT 1`), msgid).Scan(&idTopic, &subject)
				data[k] = phpSprintf(c.Txt("whopost_"+action), itoa(idTopic), subject)
				if idTopic == 0 {
					data[k] = c.Txt("who_hidden")
				}
			case c.allowedTo("moderate_forum") && c.txtExists("whoadmin_"+action):
				data[k] = c.Txt("whoadmin_" + action)
			case whoAllowedActions[action] != nil:
				if c.allowedTo(whoAllowedActions[action]...) {
					data[k] = c.Txt("whoallow_" + action)
				} else {
					data[k] = c.Txt("who_hidden")
				}
			default:
				data[k] = c.Txt("who_unknown")
			}
		}
	}

	// Load topic names.
	if len(topicIDs) != 0 {
		ids := intKeys(topicIDs)
		rows, err := a.DB.Query(a.Q(`
			SELECT t.ID_TOPIC, m.subject
			FROM ({$db_prefix}boards AS b, {$db_prefix}topics AS t, {$db_prefix}messages AS m)
			WHERE ` + c.User.QuerySeeBoard + `
				AND t.ID_TOPIC IN (` + ids + `)
				AND t.ID_BOARD = b.ID_BOARD
				AND m.ID_MSG = t.ID_FIRST_MSG
			LIMIT ` + itoa(len(topicIDs))))
		if err == nil {
			for rows.Next() {
				var idTopic int
				var subject string
				rows.Scan(&idTopic, &subject)
				for k, tpl := range topicIDs[idTopic] {
					data[k] = phpSprintf(tpl, itoa(idTopic), c.censorText(subject))
				}
			}
			rows.Close()
		}
	}

	// Load board names.
	if len(boardIDs) != 0 {
		ids := intKeys(boardIDs)
		rows, err := a.DB.Query(a.Q(`
			SELECT b.ID_BOARD, b.name
			FROM {$db_prefix}boards AS b
			WHERE ` + c.User.QuerySeeBoard + `
				AND b.ID_BOARD IN (` + ids + `)
			LIMIT ` + itoa(len(boardIDs))))
		if err == nil {
			for rows.Next() {
				var idBoard int
				var name string
				rows.Scan(&idBoard, &name)
				for k, tpl := range boardIDs[idBoard] {
					data[k] = phpSprintf(tpl, itoa(idBoard), name)
				}
			}
			rows.Close()
		}
	}

	// Load member names for the profile.
	if len(profileIDs) != 0 && (c.allowedTo("profile_view_any") || c.allowedTo("profile_view_own")) {
		ids := intKeys(profileIDs)
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, realName
			FROM {$db_prefix}members
			WHERE ID_MEMBER IN (` + ids + `)
			LIMIT ` + itoa(len(profileIDs))))
		if err == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				if !c.allowedTo("profile_view_any") && c.User.ID != id {
					continue
				}
				for k, tpl := range profileIDs[id] {
					data[k] = phpSprintf(tpl, itoa(id), name)
				}
			}
			rows.Close()
		}
	}

	return data
}

// addWho records id->session->template into one of the batch maps.
func addWho(m map[int]map[string]string, id int, session, tpl string) {
	if m[id] == nil {
		m[id] = map[string]string{}
	}
	m[id][session] = tpl
}

// intKeys returns the comma-joined keys of a batch map for an IN() clause.
func intKeys(m map[int]map[string]string) string {
	keys := make([]int, 0, len(m))
	for id := range m {
		keys = append(keys, id)
	}
	sort.Ints(keys)
	parts := make([]string, len(keys))
	for i, id := range keys {
		parts[i] = itoa(id)
	}
	return strings.Join(parts, ", ")
}

// hasKey reports whether a JSON-decoded action map has a (non-absent) key.
func hasKey(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}

// txtExists reports whether a language key resolves to a real string.
func (c *Ctx) txtExists(key string) bool {
	return c.Txt(key) != key
}
