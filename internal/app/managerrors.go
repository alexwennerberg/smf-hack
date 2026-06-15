package app

// Port of Sources/ManageErrors.php: ViewErrorLog (?action=viewErrorLog) and
// deleteErrors — the error-log viewer with column filters, paging, sort
// direction and selective/all deletion.

import (
	"encoding/base64"
	"regexp"
	"strings"
)

func init() {
	registerAction("viewErrorLog", (*Ctx).ViewErrorLog)
}

// ErrorLogFilter is $filter / $context['filter'].
type ErrorLogFilter struct {
	Variable  string // column name
	ValueSQL  string // LIKE pattern
	Href      string // ";filter=...;value=..."
	Entity    string // human label
	ValueHTML string
}

// ErrorLogEntry is one row of the log.
type ErrorLogEntry struct {
	ID            int
	MemberID      int
	MemberIP      string
	MemberSession string
	MemberLink    string
	Time          string
	URLHTML       string
	URLHref       string
	MessageHTML   string
	MessageHref   string
}

// ErrorLogCtx backs template_error_log.
type ErrorLogCtx struct {
	SortDirection string // "up" / "down"
	Start         int
	PageIndex     string
	Errors        []ErrorLogEntry
	HasFilter     bool
	Filter        ErrorLogFilter
}

var errLogColumns = map[string]string{
	"ID_MEMBER": "35",
	"ip":        "ip_address",
	"session":   "session",
	"url":       "error_url",
	"message":   "error_message",
}

var (
	reRemoveSpanEsc = regexp.MustCompile(`&lt;span class=&quot;remove&quot;&gt;(.+?)&lt;/span&gt;`)
)

// addcslashesLike escapes \, _ and % for a LIKE pattern (addcslashes($s,'\_%')).
func addcslashesLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `_`, `\_`, `%`, `\%`).Replace(s)
}

// ViewErrorLog is ViewErrorLog().
func (c *Ctx) ViewErrorLog() {
	a := c.App
	scripturl := a.ScriptURL

	c.isAllowedTo("admin_forum")
	c.adminIndex("view_errors")

	// Set up the filtering...
	var filter *ErrorLogFilter
	if c.GET.Has("value") && c.GET.Has("filter") {
		fvar := c.GET.Str("filter")
		if entityKey, ok := errLogColumns[fvar]; ok {
			var sqlVal string
			if fvar == "message" || fvar == "url" {
				if dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.GET.Str("value"), " ", "+")); err == nil {
					sqlVal = string(dec)
				}
			} else {
				sqlVal = addcslashesLike(c.GET.Str("value"))
			}
			filter = &ErrorLogFilter{
				Variable: fvar,
				ValueSQL: sqlVal,
				Href:     ";filter=" + c.GET.Str("filter") + ";value=" + c.GET.Str("value"),
				Entity:   c.Txt(entityKey),
			}
		}
	}

	// Deleting, are we?
	if c.POST.Has("delall") || c.POST.Has("delete") {
		c.deleteErrors(filter)
	}

	desc := c.REQUEST.Has("desc")

	whereSQL := ""
	var whereArgs []any
	if filter != nil {
		whereSQL = "\n\t\tWHERE " + filter.Variable + " LIKE ?"
		whereArgs = append(whereArgs, filter.ValueSQL)
	}

	var numErrors int
	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}log_errors`+whereSQL), whereArgs...).Scan(&numErrors)

	// If this filter is empty...
	if numErrors == 0 && filter != nil {
		if desc {
			c.redirectExit("action=viewErrorLog;desc")
		}
		c.redirectExit("action=viewErrorLog")
	}

	start := c.GET.Int("start")
	if start < 0 {
		start = 0
	}

	page := &ErrorLogCtx{}
	c.Page = page
	if desc {
		page.SortDirection = "down"
	} else {
		page.SortDirection = "up"
	}

	maxMessages := a.SettingInt("defaultMaxMessages")
	if maxMessages == 0 {
		maxMessages = 15
	}

	indexBase := scripturl + "?action=viewErrorLog"
	if page.SortDirection == "down" {
		indexBase += ";desc"
	}
	if filter != nil {
		indexBase += filter.Href
	}
	page.PageIndex, _ = c.constructPageIndex(indexBase, start, numErrors, maxMessages, false)
	page.Start = start

	orderDir := ""
	if page.SortDirection == "down" {
		orderDir = "DESC"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_ERROR, ID_MEMBER, ip, url, logTime, message, session
		FROM {$db_prefix}log_errors`+whereSQL+`
		ORDER BY ID_ERROR `+orderDir+`
		LIMIT ? OFFSET ?`), append(append([]any{}, whereArgs...), maxMessages, start)...)
	members := map[int]bool{}
	if err == nil {
		for rows.Next() {
			var idErr, idMember int
			var ip, urlVal, message, session string
			var logTime int64
			rows.Scan(&idErr, &idMember, &ip, &urlVal, &logTime, &message, &session)

			searchMessage := reRemoveSpanEsc.ReplaceAllString(addcslashesLike(message), "%")
			if filter != nil && searchMessage == filter.ValueSQL {
				searchMessage = addcslashesLike(message)
			}
			showMessage := reRemoveSpanEsc.ReplaceAllString(message, "$1")
			showMessage = strings.NewReplacer("\r", "", "<br />", "\n", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(showMessage)
			showMessage = strings.ReplaceAll(showMessage, "\n", "<br />")

			page.Errors = append(page.Errors, ErrorLogEntry{
				ID:            idErr,
				MemberID:      idMember,
				MemberIP:      ip,
				MemberSession: session,
				Time:          c.timeformat(logTime),
				URLHTML:       Htmlspecialchars(scripturl + urlVal),
				URLHref:       base64.StdEncoding.EncodeToString([]byte(addcslashesLike(urlVal))),
				MessageHTML:   showMessage,
				MessageHref:   base64.StdEncoding.EncodeToString([]byte(searchMessage)),
			})
			members[idMember] = true
		}
		rows.Close()
	}

	// Resolve member names.
	if len(members) > 0 {
		ids := make([]int, 0, len(members))
		for id := range members {
			ids = append(ids, id)
		}
		names := map[int]string{}
		mrows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER, realName FROM {$db_prefix}members WHERE ID_MEMBER IN (` + joinInts(ids) + `)`))
		if err == nil {
			for mrows.Next() {
				var id int
				var name string
				mrows.Scan(&id, &name)
				names[id] = name
			}
			mrows.Close()
		}
		for i := range page.Errors {
			memID := page.Errors[i].MemberID
			if memID == 0 {
				page.Errors[i].MemberLink = c.Txt("28")
			} else {
				page.Errors[i].MemberLink = `<a href="` + scripturl + `?action=profile;u=` + itoa(memID) + `">` + names[memID] + `</a>`
			}
		}
	}

	// Filtering context.
	if filter != nil {
		page.HasFilter = true
		switch filter.Variable {
		case "ID_MEMBER":
			var name string
			a.DB.QueryRow(a.Q(`SELECT realName FROM {$db_prefix}members WHERE ID_MEMBER = ?`), atoi(filter.ValueSQL)).Scan(&name)
			filter.ValueHTML = `<a href="` + scripturl + `?action=profile;u=` + filter.ValueSQL + `">` + name + `</a>`
		case "url":
			filter.ValueHTML = "'" + Htmlspecialchars(scripturl+filter.ValueSQL) + "'"
		case "message":
			h := strings.NewReplacer("\n", "<br />", "&lt;br /&gt;", "<br />", "\t", "&nbsp;&nbsp;&nbsp;", `\_`, "_", `\%`, "%", `\\`, `\`).Replace(Htmlspecialchars(filter.ValueSQL))
			filter.ValueHTML = "'" + h + "'"
		default:
			filter.ValueHTML = filter.ValueSQL
		}
		page.Filter = *filter
	}

	c.PageTitle = c.Txt("errlog1")
	c.SubTemplate = templateErrorLog
}

// deleteErrors is deleteErrors(): remove all/filtered/selected errors.
func (c *Ctx) deleteErrors(filter *ErrorLogFilter) {
	a := c.App
	c.checkSession("post", "", true)

	if c.POST.Has("delall") && filter == nil {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_errors`))
	} else if c.POST.Has("delall") && filter != nil {
		a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_errors WHERE `+filter.Variable+` LIKE ?`), filter.ValueSQL)
	} else if c.POST.Arr("delete") != nil {
		var ids []int
		c.POST.Arr("delete").Values(func(k string, v any) {
			s, _ := v.(string)
			ids = append(ids, atoi(s))
		})
		ids = uniqueInts(ids)
		if len(ids) > 0 {
			a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_errors WHERE ID_ERROR IN (` + joinInts(ids) + `)`))
		}
		desc := ""
		if c.REQUEST.Has("desc") {
			desc = ";desc"
		}
		extra := ""
		if filter != nil {
			extra = ";filter=" + c.GET.Str("filter") + ";value=" + c.GET.Str("value")
		}
		c.redirectExit("action=viewErrorLog" + desc + ";start=" + itoa(c.GET.Int("start")) + extra)
	}

	desc := ""
	if c.REQUEST.Has("desc") {
		desc = ";desc"
	}
	c.redirectExit("action=viewErrorLog" + desc)
}
