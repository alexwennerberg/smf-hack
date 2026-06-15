package app

// Port of Sources/ManageMembers.php: the member admin (?action=viewmembers) —
// ViewMemberlist (all/filtered list with sort + delete + the advanced search
// query builder), SearchMembers (the search form), MembersAwaitingActivation
// and AdminApprove (the approval/activation queue). Search params round-trip as
// base64(JSON) for pagination (PHP used base64(serialize) — documented).

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// reYMD matches the YYYY-M-D date format used by the member search.
var reYMD = regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}$`)

func init() {
	registerAction("viewmembers", (*Ctx).ViewMembers)
}

func (c *Ctx) ViewMembers() {
	a := c.App

	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"all":     {c.ViewMemberlist, "moderate_forum"},
		"approve": {c.AdminApprove, "moderate_forum"},
		"browse":  {c.MembersAwaitingActivation, "moderate_forum"},
		"search":  {c.SearchMembers, "moderate_forum"},
		"query":   {c.ViewMemberlist, "moderate_forum"},
	}
	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		sa = "all"
	}
	c.isAllowedTo(subs[sa].perm)
	c.adminIndex("view_members")
	c.loadLanguage("ManageMembers")

	// Activation counts (for the tab labels + the approval queue).
	awaitingActivation, awaitingApproval := 0, 0
	for typ, n := range c.activationNumbers() {
		if typ == 0 || typ == 2 {
			awaitingActivation += n
		} else if typ == 3 || typ == 4 || typ == 5 {
			awaitingApproval += n
		}
	}
	showActivate := a.SettingInt("registration_method") == 1 || awaitingActivation != 0
	showApprove := a.SettingInt("registration_method") == 2 || awaitingApproval != 0

	scripturl := a.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("9"), Help: "view_members", Description: c.Txt("11")}
	tabs.Tabs = append(tabs.Tabs,
		AdminTab{Title: c.Txt("303"), Description: c.Txt("11"), Href: scripturl + "?action=viewmembers;sa=all", IsSelected: sa == "all"},
		AdminTab{Title: c.Txt("mlist_search"), Description: c.Txt("11"), Href: scripturl + "?action=viewmembers;sa=search", IsSelected: sa == "search" || sa == "query"})
	if showApprove {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: phpSprintf(c.Txt("admin_browse_awaiting_approval"), itoa(awaitingApproval)), Description: c.Txt("admin_browse_approve_desc"), Href: scripturl + "?action=viewmembers;sa=browse;type=approve"})
	}
	if showActivate {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: phpSprintf(c.Txt("admin_browse_awaiting_activate"), itoa(awaitingActivation)), Description: c.Txt("admin_browse_activate_desc"), Href: scripturl + "?action=viewmembers;sa=browse;type=activate"})
	}
	tabs.Tabs[len(tabs.Tabs)-1].IsLast = true
	c.AdminTabs = tabs

	subs[sa].fn()
}

// activationNumbers returns is_activated -> count for non-active members.
func (c *Ctx) activationNumbers() map[int]int {
	out := map[int]int{}
	rows, err := c.App.DB.Query(c.App.Q(`SELECT COUNT(*), is_activated FROM {$db_prefix}members WHERE is_activated != 1 GROUP BY is_activated`))
	if err == nil {
		for rows.Next() {
			var n, typ int
			rows.Scan(&n, &typ)
			out[typ] = n
		}
		rows.Close()
	}
	return out
}

// ---- member list ----

type VMColumn struct {
	Key      string
	Label    string
	Href     string
	Selected bool
}

type VMMember struct {
	ID          int
	Username    string
	Name        string
	Email       string
	IP          string
	LastActive  string
	IsActivated bool
	Posts       int
}

type ViewMembersCtx struct {
	Columns       []VMColumn
	Members       []VMMember
	PageIndex     string
	Start         int
	SortBy        string
	SortDirection string
	CanDelete     bool
	ParamsURL     string
}

var memberListColumns = []struct{ key, label string }{
	{"ID_MEMBER", "member_id"}, {"memberName", "35"}, {"realName", "display_name"},
	{"emailAddress", "email_address"}, {"memberIP", "ip_address"}, {"lastLogin", "viewmembers_online"}, {"posts", "26"},
}

var memberSortable = map[string]bool{"ID_MEMBER": true, "memberName": true, "realName": true, "emailAddress": true, "memberIP": true, "lastLogin": true, "posts": true}

func (c *Ctx) ViewMemberlist() {
	a := c.App
	scripturl := a.ScriptURL
	subAction := c.REQUEST.Str("sa")

	// Delete?
	if c.POST.Has("delete_members") && c.POST.Arr("delete") != nil && c.allowedTo("profile_remove_any") {
		c.checkSession("post", "", true)
		var ids []int
		c.POST.Arr("delete").Values(func(k string, v any) {
			s, _ := v.(string)
			ids = append(ids, atoi(s))
		})
		c.deleteMembers(ids)
	}

	// Build the search input (from POST, a quick group filter, or decoded params).
	in := map[string]any{}
	isQuery := subAction == "query"
	if isQuery {
		if c.GET.Has("group") {
			g := itoa(c.GET.Int("group"))
			in["membergroups"] = map[string][]string{"1": {g}, "2": {g}}
		} else if c.GET.Has("pgroup") {
			in["postgroups"] = []string{itoa(c.GET.Int("pgroup"))}
		}
		if c.REQUEST.Has("params") && c.POST.Len() == 0 {
			if dec, err := base64.StdEncoding.DecodeString(c.REQUEST.Str("params")); err == nil {
				var m map[string]any
				if json.Unmarshal(dec, &m) == nil {
					in = m
				}
			}
		} else if len(in) == 0 {
			in = c.memberSearchInputFromPost()
		}
	}

	where, whereArgs := c.buildMemberWhere(in)

	searchParams := ""
	if isQuery {
		if b, err := json.Marshal(in); err == nil {
			searchParams = base64.StdEncoding.EncodeToString(b)
		}
	}

	page := &ViewMembersCtx{}
	c.Page = page
	if isQuery {
		page.ParamsURL = ";sa=query;params=" + searchParams
	}
	c.PageTitle = c.Txt("9")
	c.SubTemplate = templateViewMembers
	page.CanDelete = c.allowedTo("profile_remove_any")

	sort := c.REQUEST.Str("sort")
	if !memberSortable[sort] {
		sort = "memberName"
	}
	page.SortBy = sort
	desc := c.REQUEST.Has("desc")
	page.SortDirection = "down"
	if desc {
		page.SortDirection = "up"
	}

	for _, col := range memberListColumns {
		href := scripturl + "?action=viewmembers" + page.ParamsURL + ";sort=" + col.key + ";start=0"
		if !desc && col.key == sort {
			href += ";desc"
		}
		page.Columns = append(page.Columns, VMColumn{Key: col.key, Label: c.Txt(col.label), Href: href, Selected: sort == col.key})
	}

	// Count.
	numMembers := a.SettingInt("totalMembers")
	if where != "" && where != "1" {
		a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE `+where), whereArgs...).Scan(&numMembers)
	}

	maxMembers := a.SettingInt("defaultMaxMembers")
	if maxMembers == 0 {
		maxMembers = 30
	}
	descParam := ""
	if desc {
		descParam = ";desc"
	}
	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=viewmembers"+page.ParamsURL+";sort="+sort+descParam, c.REQUEST.Int("start"), numMembers, maxMembers, false)
	page.Start = c.REQUEST.Int("start")

	whereSQL := ""
	if isQuery && where != "" {
		whereSQL = "\n\t\tWHERE " + where
	}
	descSQL := ""
	if desc {
		descSQL = " DESC"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, memberName, realName, emailAddress, memberIP, lastLogin, posts, is_activated
		FROM {$db_prefix}members`+whereSQL+`
		ORDER BY `+sort+descSQL+`
		LIMIT ? OFFSET ?`), append(append([]any{}, whereArgs...), maxMembers, page.Start)...)
	if err == nil {
		for rows.Next() {
			var id, posts, isActivated int
			var memberName, realName, email, ip string
			var lastLogin int64
			rows.Scan(&id, &memberName, &realName, &email, &ip, &lastLogin, &posts, &isActivated)
			diff := c.Txt("never")
			if lastLogin != 0 {
				d := c.jeffsdatediff(lastLogin)
				if d == 0 {
					diff = c.Txt("viewmembers_today")
				} else if d == 1 {
					diff = itoa(d) + " " + c.Txt("viewmembers_day_ago")
				} else {
					diff = itoa(d) + " " + c.Txt("viewmembers_days_ago")
				}
			}
			if isActivated%10 != 1 {
				diff = `<i title="` + c.Txt("not_activated") + `">` + diff + `</i>`
			}
			page.Members = append(page.Members, VMMember{
				ID: id, Username: memberName, Name: realName, Email: email, IP: ip,
				LastActive: diff, IsActivated: isActivated%10 == 1, Posts: posts,
			})
		}
		rows.Close()
	}
}

// memberSearchInputFromPost collects the advanced-search fields from POST.
func (c *Ctx) memberSearchInputFromPost() map[string]any {
	in := map[string]any{}
	for _, f := range []string{"mem_id", "age", "posts", "reg_date", "last_online", "membername", "email", "website", "location", "ip", "messenger"} {
		if c.POST.Has(f) {
			in[f] = c.POST.Str(f)
		}
	}
	if t := c.POST.Arr("types"); t != nil {
		m := map[string]string{}
		t.Values(func(k string, v any) { s, _ := v.(string); m[k] = s })
		in["types"] = m
	}
	collectArr := func(name string) []string {
		var out []string
		if arr := c.POST.Arr(name); arr != nil {
			arr.Values(func(k string, v any) { s, _ := v.(string); out = append(out, s) })
		}
		return out
	}
	if v := collectArr("gender"); v != nil {
		in["gender"] = v
	}
	if v := collectArr("activated"); v != nil {
		in["activated"] = v
	}
	if v := collectArr("postgroups"); v != nil {
		in["postgroups"] = v
	}
	if mg := c.POST.Arr("membergroups"); mg != nil {
		out := map[string][]string{}
		for _, sub := range []string{"1", "2"} {
			if a := mg.Arr(sub); a != nil {
				var ids []string
				a.Values(func(k string, v any) { s, _ := v.(string); ids = append(ids, s) })
				out[sub] = ids
			}
		}
		in["membergroups"] = out
	}
	return in
}

// inStr / inArr / inTypes read from the decoded-or-post search input map.
func inStr(in map[string]any, key string) string {
	if v, ok := in[key].(string); ok {
		return v
	}
	return ""
}
func inArr(in map[string]any, key string) []string {
	switch v := in[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
func inTypes(in map[string]any, key string) string {
	switch t := in["types"].(type) {
	case map[string]string:
		return t[key]
	case map[string]any:
		if s, ok := t[key].(string); ok {
			return s
		}
	}
	return ""
}
func inMembergroups(in map[string]any, sub string) []string {
	switch mg := in["membergroups"].(type) {
	case map[string][]string:
		return mg[sub]
	case map[string]any:
		return inArr(map[string]any{"x": mg[sub]}, "x")
	}
	return nil
}

var memberRangeTrans = map[string]string{"--": "<", "-": "<=", "=": "=", "+": ">=", "++": ">"}

type memberParam struct {
	fields []string
	typ    string // int/age/date/checkbox/string
	rng    bool
	values []string // for checkbox
}

var memberParams = map[string]memberParam{
	"mem_id":      {[]string{"ID_MEMBER"}, "int", true, nil},
	"age":         {[]string{"birthdate"}, "age", true, nil},
	"posts":       {[]string{"posts"}, "int", true, nil},
	"reg_date":    {[]string{"dateRegistered"}, "date", true, nil},
	"last_online": {[]string{"lastLogin"}, "date", true, nil},
	"gender":      {[]string{"gender"}, "checkbox", false, []string{"0", "1", "2"}},
	"activated":   {[]string{"IIF(is_activated IN (1, 11), 1, 0)"}, "checkbox", false, []string{"0", "1"}},
	"membername":  {[]string{"memberName", "realName"}, "string", false, nil},
	"email":       {[]string{"emailAddress"}, "string", false, nil},
	"website":     {[]string{"websiteTitle", "websiteUrl"}, "string", false, nil},
	"location":    {[]string{"location"}, "string", false, nil},
	"ip":          {[]string{"memberIP"}, "string", false, nil},
	"messenger":   {[]string{"ICQ", "AIM", "YIM", "MSN"}, "string", false, nil},
}

// memberParamOrder keeps the PHP field order for the query parts.
var memberParamOrder = []string{"mem_id", "age", "posts", "reg_date", "last_online", "gender", "activated", "membername", "email", "website", "location", "ip", "messenger"}

// buildMemberWhere is the ViewMemberlist query builder.
func (c *Ctx) buildMemberWhere(in map[string]any) (string, []any) {
	if len(in) == 0 {
		return "", nil
	}
	var parts []string
	var args []any

	for _, name := range memberParamOrder {
		pi := memberParams[name]
		switch pi.typ {
		case "int", "age", "date":
			val := inStr(in, name)
			if val == "" {
				continue
			}
			if pi.typ == "date" {
				if !reYMD.MatchString(val) {
					continue
				}
			}
			rangeOp := inTypes(in, name)
			if memberRangeTrans[rangeOp] == "" {
				rangeOp = "="
			}
			if pi.typ == "age" {
				now := time.Unix(c.forumTime(false, 0), 0).UTC()
				yr, mon, day := now.Year(), int(now.Month()), now.Day()
				age := atoi(val)
				upper := zeroPad(yr-age, 4) + "-" + zeroPad(mon, 2) + "-" + zeroPad(day, 2)
				lower := zeroPad(yr-age-1, 4) + "-" + zeroPad(mon, 2) + "-" + zeroPad(day, 2)
				if rangeOp == "-" || rangeOp == "--" || rangeOp == "=" {
					b := lower
					if rangeOp == "--" {
						b = upper
					}
					parts = append(parts, pi.fields[0]+" > '"+b+"'")
				}
				if rangeOp == "+" || rangeOp == "++" || rangeOp == "=" {
					b := upper
					if rangeOp == "++" {
						b = lower
					}
					parts = append(parts, pi.fields[0]+" <= '"+b+"'")
					parts = append(parts, pi.fields[0]+" > '0000-12-31'")
				}
			} else if pi.typ == "date" {
				ts := dateToUnix(val)
				if rangeOp == "=" {
					parts = append(parts, pi.fields[0]+" > "+itoa64(ts)+" AND "+pi.fields[0]+" < "+itoa64(ts+86400))
				} else {
					parts = append(parts, pi.fields[0]+" "+memberRangeTrans[rangeOp]+" "+itoa64(ts))
				}
			} else {
				parts = append(parts, pi.fields[0]+" "+memberRangeTrans[rangeOp]+" "+itoa(atoi(val)))
			}
		case "checkbox":
			vals := inArr(in, name)
			if len(vals) == 0 || len(vals) == len(pi.values) {
				continue
			}
			var quoted []string
			for _, v := range vals {
				quoted = append(quoted, "'"+strings.ReplaceAll(v, "'", "")+"'")
			}
			parts = append(parts, pi.fields[0]+" IN ("+strings.Join(quoted, ", ")+")")
		case "string":
			val := inStr(in, name)
			if val == "" {
				continue
			}
			// '*'/'?' wildcards -> SQL %/_.
			like := strings.NewReplacer("%", `\%`, "_", `\_`, "*", "%", "?", "_").Replace(strings.ToLower(val))
			var ors []string
			for _, f := range pi.fields {
				ors = append(ors, f+" LIKE ?")
				args = append(args, "%"+like+"%")
			}
			parts = append(parts, "("+strings.Join(ors, " OR ")+")")
		}
	}

	// Membergroups.
	var mgParts []string
	mg1 := inMembergroups(in, "1")
	mg2 := inMembergroups(in, "2")
	if len(mg1) > 0 {
		mgParts = append(mgParts, "ID_GROUP IN ("+joinStrInts(mg1)+")")
	}
	if len(mg2) > 0 {
		for _, mg := range mg2 {
			mgParts = append(mgParts, "FIND_IN_SET("+itoa(atoi(mg))+", additionalGroups)")
		}
	}
	if len(mgParts) > 0 {
		parts = append(parts, "("+strings.Join(mgParts, " OR ")+")")
	}
	if pg := inArr(in, "postgroups"); len(pg) > 0 {
		parts = append(parts, "ID_POST_GROUP IN ("+joinStrInts(pg)+")")
	}

	if len(parts) == 0 {
		return "1", nil
	}
	return strings.Join(parts, "\n\t\t\tAND "), args
}

// joinStrInts joins string-ints as a sanitized comma list.
func joinStrInts(ss []string) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = itoa(atoi(s))
	}
	return strings.Join(out, ", ")
}

// dateToUnix converts 'YYYY-MM-DD' to a unix timestamp (UTC midnight).
func dateToUnix(s string) int64 {
	t, err := time.Parse("2006-1-2", s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// jeffsdatediff is jeffsdatediff(): whole days since $old, 0 if today.
func (c *Ctx) jeffsdatediff(old int64) int {
	forumNow := c.forumTime(false, 0)
	tm := time.Unix(forumNow, 0).UTC()
	sinceMidnight := int64(tm.Hour()*3600 + tm.Minute()*60 + tm.Second())
	dis := nowUnix() - old
	if dis < sinceMidnight {
		return 0
	}
	dis -= sinceMidnight
	return int((dis + 24*3600 - 1) / (24 * 3600))
}

// ---- search form ----

type MemberGroupOpt struct {
	ID              int
	Name            string
	CanBeAdditional bool
}

type SearchMembersCtx struct {
	Membergroups []MemberGroupOpt
	Postgroups   []MemberGroupOpt
}

func (c *Ctx) loadMemberSearchGroups() ([]MemberGroupOpt, []MemberGroupOpt) {
	a := c.App
	mg := []MemberGroupOpt{{ID: 0, Name: c.Txt("membergroups_members")}}
	var pg []MemberGroupOpt
	rows, err := a.DB.Query(a.Q(`SELECT ID_GROUP, groupName, minPosts FROM {$db_prefix}membergroups WHERE ID_GROUP != 3 ORDER BY minPosts, IIF(ID_GROUP < 4, ID_GROUP, 4), groupName`))
	if err == nil {
		for rows.Next() {
			var id, minPosts int
			var name string
			rows.Scan(&id, &name, &minPosts)
			if minPosts == -1 {
				mg = append(mg, MemberGroupOpt{ID: id, Name: name, CanBeAdditional: true})
			} else {
				pg = append(pg, MemberGroupOpt{ID: id, Name: name})
			}
		}
		rows.Close()
	}
	return mg, pg
}

func (c *Ctx) SearchMembers() {
	page := &SearchMembersCtx{}
	c.Page = page
	page.Membergroups, page.Postgroups = c.loadMemberSearchGroups()
	c.PageTitle = c.Txt("9")
	c.SubTemplate = templateSearchMembers
}

// ---- approval queue ----

type ABFilter struct {
	Type     int
	Amount   int
	Desc     string
	Selected bool
}

type ABMember struct {
	ID             int
	Username       string
	Email          string
	IP             string
	DateRegistered string
}

type AdminBrowseCtx struct {
	BrowseType       string
	CurrentFilter    int
	NumMembers       int
	Columns          []VMColumn
	Members          []ABMember
	PageIndex        string
	Start            int
	SortBy           string
	SortDirection    string
	AvailableFilters []ABFilter
	AllowedActions   [][2]string // {key, desc}
	ShowFilter       bool
}

var browseColumns = []struct{ key, label string }{
	{"ID_MEMBER", "admin_browse_id"}, {"memberName", "admin_browse_username"},
	{"emailAddress", "admin_browse_email"}, {"memberIP", "admin_browse_ip"}, {"dateRegistered", "admin_browse_registered"},
}
var browseSortable = map[string]bool{"ID_MEMBER": true, "memberName": true, "emailAddress": true, "memberIP": true, "dateRegistered": true}

func (c *Ctx) MembersAwaitingActivation() {
	a := c.App
	scripturl := a.ScriptURL
	c.PageTitle = c.Txt("9")

	browseType := c.REQUEST.Str("type")
	if browseType != "approve" && browseType != "activate" {
		if a.SettingInt("registration_method") == 1 {
			browseType = "activate"
		} else {
			browseType = "approve"
		}
	}

	actNums := c.activationNumbers()
	var allowed []int
	if browseType == "approve" {
		allowed = []int{3, 4, 5}
	} else {
		allowed = []int{0, 2}
	}

	page := &AdminBrowseCtx{BrowseType: browseType}
	c.Page = page

	currentFilter := -1
	if c.REQUEST.Has("filter") {
		f := c.REQUEST.Int("filter")
		if inInts(allowed, f) && actNums[f] != 0 {
			currentFilter = f
		}
	}

	for _, typ := range []int{0, 2, 3, 4, 5} {
		amount := actNums[typ]
		if inInts(allowed, typ) && amount > 0 {
			desc := "?"
			if c.TxtHas("admin_browse_filter_type_" + itoa(typ)) {
				desc = c.Txt("admin_browse_filter_type_" + itoa(typ))
			}
			page.AvailableFilters = append(page.AvailableFilters, ABFilter{Type: typ, Amount: amount, Desc: desc, Selected: typ == currentFilter})
		}
	}
	if currentFilter == -1 && len(page.AvailableFilters) > 0 {
		currentFilter = page.AvailableFilters[0].Type
		page.AvailableFilters[0].Selected = true
	}
	page.CurrentFilter = currentFilter
	if currentFilter != 0 && currentFilter != 3 && len(page.AvailableFilters) == 1 {
		page.ShowFilter = true
	}

	// Allowed actions.
	if browseType == "approve" {
		if currentFilter == 4 {
			page.AllowedActions = [][2]string{{"reject", c.Txt("admin_browse_w_approve_deletion")}, {"ok", c.Txt("admin_browse_w_reject")}}
		} else {
			page.AllowedActions = [][2]string{
				{"ok", c.Txt("admin_browse_w_approve")},
				{"okemail", c.Txt("admin_browse_w_approve") + " " + c.Txt("admin_browse_w_email")},
				{"require_activation", c.Txt("admin_browse_w_approve_require_activate")},
				{"reject", c.Txt("admin_browse_w_reject")},
				{"rejectemail", c.Txt("admin_browse_w_reject") + " " + c.Txt("admin_browse_w_email")},
			}
		}
	} else {
		page.AllowedActions = [][2]string{
			{"ok", c.Txt("admin_browse_w_activate")},
			{"okemail", c.Txt("admin_browse_w_activate") + " " + c.Txt("admin_browse_w_email")},
			{"delete", c.Txt("admin_browse_w_delete")},
			{"deleteemail", c.Txt("admin_browse_w_delete") + " " + c.Txt("admin_browse_w_email")},
			{"remind", c.Txt("admin_browse_w_remind") + " " + c.Txt("admin_browse_w_email")},
		}
	}

	sort := c.REQUEST.Str("sort")
	if !browseSortable[sort] {
		sort = "dateRegistered"
	}
	page.SortBy = sort
	desc := c.REQUEST.Has("desc")
	page.SortDirection = "down"
	if desc {
		page.SortDirection = "up"
	}
	for _, col := range browseColumns {
		href := scripturl + "?action=viewmembers;sa=browse;type=" + browseType + ";sort=" + col.key + ";start=0"
		if !desc && col.key == sort {
			href += ";desc"
		}
		page.Columns = append(page.Columns, VMColumn{Key: col.key, Label: c.Txt(col.label), Href: href, Selected: sort == col.key})
	}

	a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}members WHERE is_activated = ?`), currentFilter).Scan(&page.NumMembers)

	maxMembers := a.SettingInt("defaultMaxMembers")
	if maxMembers == 0 {
		maxMembers = 30
	}
	descParam := ""
	if desc {
		descParam = ";desc"
	}
	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=viewmembers;sa=browse;type="+browseType+";sort="+sort+descParam, c.REQUEST.Int("start"), page.NumMembers, maxMembers, false)
	page.Start = c.REQUEST.Int("start")

	descSQL := ""
	if desc {
		descSQL = " DESC"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, memberName, emailAddress, memberIP, dateRegistered
		FROM {$db_prefix}members
		WHERE is_activated = ?
		ORDER BY `+sort+descSQL+`
		LIMIT ? OFFSET ?`), currentFilter, maxMembers, page.Start)
	if err == nil {
		for rows.Next() {
			var id int
			var name, email, ip string
			var dateReg int64
			rows.Scan(&id, &name, &email, &ip, &dateReg)
			page.Members = append(page.Members, ABMember{ID: id, Username: name, Email: email, IP: ip, DateRegistered: c.timeformat(dateReg)})
		}
		rows.Close()
	}

	c.SubTemplate = templateAdminBrowse
}

func (c *Ctx) AdminApprove() {
	a := c.App
	c.loadLanguage("Login")

	browseType := c.REQUEST.Str("type")
	if browseType != "approve" && browseType != "activate" {
		if a.SettingInt("registration_method") == 1 {
			browseType = "activate"
		} else {
			browseType = "approve"
		}
	}
	currentFilter := c.REQUEST.Int("orig_filter")
	backTo := "action=viewmembers;sa=browse;type=" + browseType + ";sort=" + c.REQUEST.Str("sort") + ";filter=" + itoa(currentFilter) + ";start=" + itoa(c.REQUEST.Int("start"))

	c.checkSession("post", "", true)

	// Applying a filter -> redirect.
	if c.REQUEST.Has("filter") && c.REQUEST.Int("filter") != currentFilter {
		c.redirectExit("action=viewmembers;sa=browse;type=" + browseType + ";sort=" + c.REQUEST.Str("sort") + ";filter=" + c.REQUEST.Str("filter") + ";start=" + itoa(c.REQUEST.Int("start")))
	}
	if !c.POST.Has("todoAction") && !c.POST.Has("time_passed") {
		c.redirectExit(backTo)
	}

	var condition string
	var condArgs []any
	if c.POST.Has("time_passed") {
		timeBefore := nowUnix() - 86400*int64(c.POST.Int("time_passed"))
		condition = " AND dateRegistered < " + itoa64(timeBefore)
	} else {
		var members []int
		c.POST.Arr("todoAction").Values(func(k string, v any) {
			s, _ := v.(string)
			members = append(members, atoi(s))
		})
		condition = " AND ID_MEMBER IN (" + joinInts(members) + ")"
	}

	type memberInfo struct {
		id             int
		username, name string
		email, code    string
	}
	var infos []memberInfo
	var memberIDs []int
	rows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER, memberName, realName, emailAddress, validation_code FROM {$db_prefix}members WHERE is_activated = ?`+condition+` ORDER BY lngfile`), append([]any{currentFilter}, condArgs...)...)
	if err == nil {
		for rows.Next() {
			var mi memberInfo
			rows.Scan(&mi.id, &mi.username, &mi.name, &mi.email, &mi.code)
			infos = append(infos, mi)
			memberIDs = append(memberIDs, mi.id)
		}
		rows.Close()
	}
	memberCount := len(infos)
	if memberCount == 0 {
		c.redirectExit(backTo)
	}

	todo := c.POST.Str("todo")
	scripturl := a.ScriptURL
	sendApprovalMail := func(subject, body string) {
		for _, m := range infos {
			c.sendmail([]string{m.email}, subject, body, "")
		}
	}
	_ = scripturl

	switch todo {
	case "ok", "okemail":
		a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET validation_code = '', is_activated = 1 WHERE is_activated = ?`+condition), currentFilter)
		if todo == "okemail" {
			for _, m := range infos {
				c.sendmail([]string{m.email}, c.Txt("register_subject"),
					c.Txt("hello_guest")+" "+m.name+"!\n\n"+c.Txt("admin_approve_accept_desc")+" "+c.Txt("719")+" "+m.username+"\n\n"+c.Txt("701")+"\n"+scripturl+"?action=profile\n\n"+c.Txt("130"), "")
			}
		}
	case "require_activation":
		for _, m := range infos {
			code := generateValidationCode()
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET validation_code = ?, is_activated = 0 WHERE is_activated = ?`+condition+` AND ID_MEMBER = ?`), code, currentFilter, m.id)
			c.sendmail([]string{m.email}, c.Txt("register_subject"),
				c.Txt("hello_guest")+" "+m.name+"!\n\n"+c.Txt("admin_approve_require_activation")+" "+c.Txt("admin_approve_remind_desc2")+"\n"+scripturl+"?action=activate;u="+itoa(m.id)+";code="+code+"\n\n"+c.Txt("130"), "")
		}
	case "reject", "rejectemail":
		c.deleteMembers(memberIDs)
		if todo == "rejectemail" {
			for _, m := range infos {
				c.sendmail([]string{m.email}, c.Txt("admin_approve_reject"), m.name+",\n\n"+c.Txt("admin_approve_reject_desc")+"\n\n"+c.Txt("130"), "")
			}
		}
	case "delete", "deleteemail":
		c.deleteMembers(memberIDs)
		if todo == "deleteemail" {
			for _, m := range infos {
				c.sendmail([]string{m.email}, c.Txt("admin_approve_delete"), m.name+",\n\n"+c.Txt("admin_approve_delete_desc")+"\n\n"+c.Txt("130"), "")
			}
		}
	case "remind":
		for _, m := range infos {
			c.sendmail([]string{m.email}, c.Txt("admin_approve_remind"),
				m.name+",\n\n"+c.Txt("admin_approve_remind_desc")+" "+a.Config.MbName+".\n\n"+c.Txt("admin_approve_remind_desc2")+"\n\n"+scripturl+"?action=activate;u="+itoa(m.id)+";code="+m.code+"\n\n"+c.Txt("130"), "")
		}
	}
	_ = sendApprovalMail

	if currentFilter == 3 || currentFilter == 4 {
		un := a.SettingInt("unapprovedMembers")
		if un > memberCount {
			un -= memberCount
		} else {
			un = 0
		}
		a.UpdateSettings(map[string]string{"unapprovedMembers": itoa(un)})
	}
	c.updateStatsMemberRecount()
	if todo != "delete" && todo != "deleteemail" && todo != "reject" && todo != "rejectemail" && todo != "remind" {
		c.updateStatsPostgroups(0)
	}

	c.redirectExit(backTo)
}
