package app

// Port of Sources/Modlog.php: ViewModlog (?action=modlog) — the moderation
// log viewer with sorting, search, paging and 24h-delayed deletion.

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

func init() {
	registerAction("modlog", (*Ctx).ViewModlog)
}

// ModlogColumn is one sortable column header.
type ModlogColumn struct {
	Key      string
	Label    string
	Href     string
	Link     string
	Selected bool
}

// ModlogExtra is one rendered key/value pair from a log entry's extra data.
type ModlogExtra struct {
	Key   string
	Value string
}

// ModlogEntry is one row of the log.
type ModlogEntry struct {
	ID            int
	IP            string
	Position      string
	ModeratorLink string
	Time          string
	Editable      bool
	Action        string
	Extra         []ModlogExtra
}

// ModlogCtx is the page context for Modlog.template.php.
type ModlogCtx struct {
	Order         string
	Dir           string // ";d" or ""
	SortDirection string // "down"/"up"
	Start         int
	EntryCount    int
	PageIndex     string
	SearchParams  string // base64-encoded (this port's |"| encoding)
	SearchString  string
	SearchType    string
	SearchLabel   string
	Columns       []ModlogColumn
	Entries       []ModlogEntry
}

// modlogDescriptions maps the stored action key to its language string key.
var modlogDescriptions = []struct{ action, txt string }{
	{"lock", "modlog_ac_locked"},
	{"sticky", "modlog_ac_stickied"},
	{"modify", "modlog_ac_modified"},
	{"merge", "modlog_ac_merged"},
	{"split", "modlog_ac_split"},
	{"move", "modlog_ac_moved"},
	{"remove", "modlog_ac_removed"},
	{"delete", "modlog_ac_deleted"},
	{"delete_member", "modlog_ac_deleted_member"},
	{"ban", "modlog_ac_banned"},
	{"news", "modlog_ac_news"},
	{"profile", "modlog_ac_profile"},
	{"pruned", "modlog_ac_pruned"},
}

// modlogColumnSQL is the sortable column whitelist (key -> SQL expression).
var modlogColumnSQL = map[string]string{
	"action": "lm.action",
	"time":   "lm.logTime",
	"member": "mem.realName",
	"group":  "mg.groupName",
	"ip":     "lm.ip",
}

// modlogSearchSQL is the searchable column whitelist.
var modlogSearchSQL = map[string]string{
	"action": "lm.action",
	"member": "mem.realName",
	"group":  "mg.groupName",
	"ip":     "lm.ip",
}

// ViewModlog is ViewModlog().
func (c *Ctx) ViewModlog() {
	a := c.App
	scripturl := a.ScriptURL

	c.isAllowedTo("admin_forum")

	c.adminIndex("view_moderation_log")

	c.PageTitle = c.Txt("modlog_view")

	const displayPage = 30
	const hoursDisable = 24
	cutoff := nowUnix() - hoursDisable*3600

	// Handle deletion...
	if c.POST.Has("removeall") {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_actions WHERE logTime < ?`), cutoff)
	} else if c.POST.Has("remove") && c.POST.Arr("delete") != nil {
		var ids []int
		c.POST.Arr("delete").Values(func(k string, v any) {
			s, _ := v.(string)
			ids = append(ids, atoi(s))
		})
		ids = uniqueInts(ids)
		if len(ids) > 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_actions WHERE ID_ACTION IN (`+joinInts(ids)+`) AND logTime < ?`), cutoff)
		}
	}

	page := &ModlogCtx{}
	c.Page = page

	// Pass order and direction so they survive a remove command.
	if c.REQUEST.Has("d") {
		page.Dir = ";d"
		page.SortDirection = "up"
	} else {
		page.Dir = ""
		page.SortDirection = "down"
	}

	// Setup the direction stuff...
	order := c.REQUEST.Str("order")
	if _, ok := modlogColumnSQL[order]; !ok {
		order = "time"
	}
	page.Order = order
	orderType := modlogColumnSQL[order]

	// Decode incoming search params (this port: type|"|string, base64).
	var paramString, paramType string
	if c.REQUEST.Has("params") {
		if dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.REQUEST.Str("params"), " ", "+")); err == nil {
			parts := strings.SplitN(string(dec), `|"|`, 2)
			if len(parts) == 2 {
				paramType, paramString = parts[0], parts[1]
			}
		}
	}

	// Reconcile request overrides with decoded params.
	searchString := paramString
	if paramString == "" || (c.REQUEST.Str("search") != "" && paramString != c.REQUEST.Str("search")) {
		searchString = c.REQUEST.Str("search")
	}

	searchType := paramType
	if _, ok := modlogSearchSQL[searchType]; c.REQUEST.Has("search_type") || paramType == "" || !ok {
		if rt := c.REQUEST.Str("search_type"); rt != "" {
			if _, ok := modlogSearchSQL[rt]; ok {
				searchType = rt
			} else {
				searchType = "member"
			}
		} else if _, ok := modlogSearchSQL[order]; ok {
			searchType = order
		} else {
			searchType = "member"
		}
	}

	searchColumn := modlogSearchSQL[searchType]

	// Setup the search context.
	if searchString != "" {
		page.SearchParams = base64.StdEncoding.EncodeToString([]byte(searchType + `|"|` + searchString))
	}
	// Escape for display: search arrives via the POST form (or base64 params),
	// neither of which is htmlspecialchars'd at parse time the way GET is.
	page.SearchString = Htmlspecialchars(searchString)
	page.SearchType = searchType
	page.SearchLabel = c.Txt(modlogSearchLabelKey(searchType))

	// Provide extra info about each column - the link, whether selected, etc.
	colOrder := []string{"action", "time", "member", "group", "ip"}
	colLabelKey := map[string]string{"action": "modlog_action", "time": "modlog_date", "member": "modlog_member", "group": "modlog_position", "ip": "modlog_ip"}
	for _, col := range colOrder {
		href := scripturl + "?action=modlog;order=" + col + ";start=0"
		if page.SearchParams != "" {
			href += ";params=" + page.SearchParams
		}
		if !c.REQUEST.Has("d") && col == order {
			href += ";d"
		}
		label := c.Txt(colLabelKey[col])
		page.Columns = append(page.Columns, ModlogColumn{
			Key:      col,
			Label:    label,
			Href:     href,
			Link:     `<a href="` + href + `">` + label + `</a>`,
			Selected: order == col,
		})
	}

	// Searching by action requires mapping the localized text back to a key.
	searchKey := searchString
	if searchType == "action" && searchString != "" {
		for _, d := range modlogDescriptions {
			if strings.Contains(c.Txt(d.txt), searchString) {
				searchKey = d.action
				break
			}
		}
	}

	whereClause := ""
	var whereArgs []any
	if searchString != "" {
		whereClause = "\n\t\tWHERE INSTR(" + searchColumn + ", ?)"
		whereArgs = append(whereArgs, searchKey)
	}

	// Count the total number of entries for pagination.
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}log_actions AS lm
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = lm.ID_MEMBER)
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))`+whereClause), whereArgs...).Scan(&page.EntryCount)

	// Create the page index.
	indexBase := scripturl + "?action=modlog;order=" + order + page.Dir
	if page.SearchParams != "" {
		indexBase += ";params=" + page.SearchParams
	}
	page.PageIndex, _ = c.constructPageIndex(indexBase, c.Start, page.EntryCount, displayPage, false)
	page.Start = c.Start

	// The descending modifier unless ascending (;d) is requested.
	dirSQL := " DESC"
	if c.REQUEST.Has("d") {
		dirSQL = ""
	}

	// Fetch the log details.
	rows, err := a.DB.Query(a.Q(`
		SELECT
			lm.ID_ACTION, lm.ID_MEMBER, lm.ip, lm.logTime, lm.action, lm.extra,
			mem.realName, mg.groupName
		FROM {$db_prefix}log_actions AS lm
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = lm.ID_MEMBER)
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))`+whereClause+`
		ORDER BY `+orderType+dirSQL+`
		LIMIT ? OFFSET ?`), append(append([]any{}, whereArgs...), displayPage, page.Start)...)

	topics := map[int][]*modlogRow{}
	boards := map[int][]*modlogRow{}
	members := map[int][]*modlogRow{}
	var allRows []*modlogRow

	if err == nil {
		for rows.Next() {
			var idAction, idMember int
			var ip, action, extraJSON string
			var logTime int64
			var realName, groupName *string
			rows.Scan(&idAction, &idMember, &ip, &logTime, &action, &extraJSON, &realName, &groupName)

			extra := map[string]any{}
			if extraJSON != "" {
				if err := json.Unmarshal([]byte(extraJSON), &extra); err != nil {
					extra = map[string]any{}
				}
			}

			// IP range / email decorations.
			if v, ok := extra["ip_range"]; ok {
				s := modlogStr(v)
				extra["ip_range"] = `<a href="` + scripturl + `?action=trackip;searchip=` + s + `">` + s + `</a>`
			}
			if v, ok := extra["email"]; ok {
				s := modlogStr(v)
				extra["email"] = `<a href="mailto:` + s + `">` + s + `</a>`
			}

			name := ""
			if realName != nil {
				name = *realName
			}
			pos := ""
			if groupName != nil {
				pos = *groupName
			}

			desc := action
			for _, d := range modlogDescriptions {
				if d.action == action {
					desc = c.Txt(d.txt)
					break
				}
			}

			entry := &ModlogEntry{
				ID:            idAction,
				IP:            ip,
				Position:      pos,
				ModeratorLink: `<a href="` + scripturl + `?action=profile;u=` + itoa(idMember) + `">` + name + `</a>`,
				Time:          c.timeformat(logTime),
				Editable:      nowUnix() > logTime+hoursDisable*3600,
				Action:        desc,
			}
			mr := &modlogRow{entry: entry, extra: extra}
			allRows = append(allRows, mr)

			if v, ok := extra["topic"]; ok {
				topics[modlogInt(v)] = append(topics[modlogInt(v)], mr)
			}
			if v, ok := extra["new_topic"]; ok {
				topics[modlogInt(v)] = append(topics[modlogInt(v)], mr)
			}
			if v, ok := extra["member"]; ok {
				members[modlogInt(v)] = append(members[modlogInt(v)], mr)
			}
			if v, ok := extra["board_to"]; ok {
				boards[modlogInt(v)] = append(boards[modlogInt(v)], mr)
			}
			if v, ok := extra["board_from"]; ok {
				boards[modlogInt(v)] = append(boards[modlogInt(v)], mr)
			}
		}
		rows.Close()
	}

	// Resolve board names into links.
	if len(boards) > 0 {
		brows, err := a.DB.Query(a.Q(`SELECT ID_BOARD, name FROM {$db_prefix}boards WHERE ID_BOARD IN (` + joinInts(mapIntKeys(boards)) + `)`))
		if err == nil {
			for brows.Next() {
				var id int
				var name string
				brows.Scan(&id, &name)
				link := `<a href="` + scripturl + `?board=` + itoa(id) + `">` + name + `</a>`
				for _, mr := range boards[id] {
					if v, ok := mr.extra["board_to"]; ok && modlogInt(v) == id {
						mr.extra["board_to"] = link
					} else if v, ok := mr.extra["board_from"]; ok && modlogInt(v) == id {
						mr.extra["board_from"] = link
					}
				}
			}
			brows.Close()
		}
	}

	// Resolve topic subjects into links.
	if len(topics) > 0 {
		trows, err := a.DB.Query(a.Q(`
			SELECT ms.subject, t.ID_TOPIC
			FROM {$db_prefix}topics AS t, {$db_prefix}messages AS ms
			WHERE t.ID_TOPIC IN (` + joinInts(mapIntKeys(topics)) + `)
				AND ms.ID_MSG = t.ID_FIRST_MSG`))
		if err == nil {
			for trows.Next() {
				var subject string
				var id int
				trows.Scan(&subject, &id)
				for _, mr := range topics[id] {
					anchor := "0"
					if m, ok := mr.extra["message"]; ok {
						ms := modlogStr(m)
						anchor = "msg" + ms + "#msg" + ms
					}
					link := `<a href="` + scripturl + `?topic=` + itoa(id) + `.` + anchor + `">` + subject + `</a>`
					if v, ok := mr.extra["topic"]; ok && modlogInt(v) == id {
						mr.extra["topic"] = link
					} else if v, ok := mr.extra["new_topic"]; ok && modlogInt(v) == id {
						mr.extra["new_topic"] = link
					}
				}
			}
			trows.Close()
		}
	}

	// Resolve member ids into links.
	if len(members) > 0 {
		mrows, err := a.DB.Query(a.Q(`SELECT realName, ID_MEMBER FROM {$db_prefix}members WHERE ID_MEMBER IN (` + joinInts(mapIntKeys(members)) + `)`))
		if err == nil {
			for mrows.Next() {
				var name string
				var id int
				mrows.Scan(&name, &id)
				link := `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + name + `</a>`
				for _, mr := range members[id] {
					mr.extra["member"] = link
				}
			}
			mrows.Close()
		}
	}

	// Finalize each entry's extra into sorted key/value pairs. NOTE: extra is
	// stored as JSON in this port (documented deviation); JSON object key order
	// is not preserved, so pairs render in sorted-key order rather than PHP's
	// insertion order.
	for _, mr := range allRows {
		keys := make([]string, 0, len(mr.extra))
		for k := range mr.extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			mr.entry.Extra = append(mr.entry.Extra, ModlogExtra{Key: k, Value: modlogStr(mr.extra[k])})
		}
		page.Entries = append(page.Entries, *mr.entry)
	}

	c.SubTemplate = templateModlog
}

// modlogRow pairs a built entry with its still-mutable extra map during the
// link-resolution passes.
type modlogRow struct {
	entry *ModlogEntry
	extra map[string]any
}

// modlogSearchLabelKey returns the label lang key for a search type.
func modlogSearchLabelKey(t string) string {
	switch t {
	case "action":
		return "modlog_action"
	case "member":
		return "modlog_member"
	case "group":
		return "modlog_position"
	case "ip":
		return "modlog_ip"
	}
	return "modlog_member"
}

// modlogInt coerces a JSON-decoded extra value to an int.
func modlogInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		return atoi(x)
	}
	return 0
}

// modlogStr stringifies a JSON-decoded extra value the way PHP would echo it.
func modlogStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return itoa64(int64(x))
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "1"
		}
		return ""
	case nil:
		return ""
	}
	return ""
}

// mapIntKeys returns the keys of a map[int][]*modlogRow.
func mapIntKeys(m map[int][]*modlogRow) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}
