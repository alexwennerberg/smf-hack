package app

// Port of Sources/Memberlist.php: Memberlist, MLAll, MLSearch,
// printMemberListRows.

import (
	"math"
	"net/url"
	"regexp"
	"strings"
)

func init() {
	registerAction("mlist", (*Ctx).Memberlist)
}

// MLColumn is one sortable column header.
type MLColumn struct {
	Key      string
	Label    string
	Width    string
	Colspan  string
	Href     string
	Link     string
	Selected bool
}

// MLSortLink is one of the all/search tabs.
type MLSortLink struct {
	Label    string
	Action   string
	Selected bool
}

// MLMember wraps a MemberCtx with list-specific fields.
type MLMember struct {
	*MemberCtx
	PostPercent    int
	RegisteredDate string
	SortLetter     string
}

// MemberlistCtx is the page context for the Memberlist templates.
type MemberlistCtx struct {
	ListingBy      string
	SortLinks      []MLSortLink
	NumMembers     int
	Columns        []*MLColumn
	CanSendPM      bool
	LetterLinks    string
	SortBy         string
	SortDirection  string
	PageIndex      string
	Start          int
	End            int
	Members        []*MLMember
	HasOldSearch   bool
	OldSearch      string
	OldSearchValue string
}

// Memberlist is Memberlist(): show a listing of the registered members.
func (c *Ctx) Memberlist() {
	a := c.App
	scripturl := a.ScriptURL

	// Make sure they can view the memberlist.
	c.isAllowedTo("view_mlist")

	page := &MemberlistCtx{}
	c.Page = page

	page.ListingBy = "all"
	if !empty(c.GET.Str("sa")) {
		page.ListingBy = c.GET.Str("sa")
	}

	// Set up the sort links.
	page.SortLinks = []MLSortLink{
		{Label: c.Txt("303"), Action: "all", Selected: page.ListingBy == "all"},
		{Label: c.Txt("mlist_search"), Action: "search", Selected: page.ListingBy == "search"},
	}

	page.NumMembers = a.SettingInt("totalMembers")

	// Set up the columns...
	page.Columns = []*MLColumn{
		{Key: "isOnline", Label: c.Txt("online8"), Width: "20"},
		{Key: "realName", Label: c.Txt("35")},
		{Key: "emailAddress", Label: c.Txt("307"), Width: "25"},
		{Key: "websiteUrl", Label: c.Txt("96"), Width: "25"},
		{Key: "ICQ", Label: c.Txt("513"), Width: "25"},
		{Key: "AIM", Label: c.Txt("603"), Width: "25"},
		{Key: "YIM", Label: c.Txt("604"), Width: "25"},
		{Key: "MSN", Label: c.Txt("MSN"), Width: "25"},
		{Key: "ID_GROUP", Label: c.Txt("87")},
		{Key: "registered", Label: c.Txt("233")},
		{Key: "posts", Label: c.Txt("21"), Width: "115", Colspan: "2"},
	}

	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=mlist", Name: c.Txt("332")})

	page.CanSendPM = c.allowedTo("pm_send")

	// Jump to the sub action.
	if page.ListingBy == "search" {
		c.mlSearch(page)
	} else {
		c.mlAll(page)
	}
}

// mlAll is MLAll(): list all members, page by page. (The 2k-member realName
// index cache is not ported; SQLite sorts the list directly.)
func (c *Ctx) mlAll(page *MemberlistCtx) {
	a := c.App
	scripturl := a.ScriptURL

	// Without cache we need an extra query to get the amount of members.
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM {$db_prefix}members
		WHERE is_activated = 1`)).Scan(&page.NumMembers)

	// Set defaults for sort (realName) and start. (0)
	sortKey := c.REQUEST.Str("sort")
	valid := false
	for _, col := range page.Columns {
		if col.Key == sortKey {
			valid = true
		}
	}
	if !valid {
		sortKey = "realName"
	}

	start := 0
	startStr := c.REQUEST.Str("start")
	if regexp.MustCompile(`^\d+$`).MatchString(startStr) {
		start = atoi(startStr)
	} else {
		// A letter link: count members sorting before that letter.
		lower := strings.ToLower(startStr)
		if lower == "" || lower[0] == '\'' || lower[0] == '\\' || lower[0] == '/' {
			c.fatalError("Hacker?", false)
		}
		letter := string(lower[0])
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*)
			FROM {$db_prefix}members
			WHERE LOWER(SUBSTR(realName, 1, 1)) < ?
				AND is_activated = 1`), letter).Scan(&start)
	}

	for i := 97; i < 123; i++ {
		ch := string(rune(i))
		page.LetterLinks += `<a href="` + scripturl + `?action=mlist;sa=all;start=` + ch + `#letter` + ch + `">` + strings.ToUpper(ch) + `</a> `
	}

	// Sort out the column information.
	hasDesc := c.REQUEST.Has("desc")
	for _, col := range page.Columns {
		col.Href = scripturl + "?action=mlist;sort=" + col.Key + ";start=0"
		if !hasDesc && col.Key == sortKey {
			col.Href += ";desc"
		}
		col.Link = `<a href="` + col.Href + `">` + col.Label + `</a>`
		col.Selected = sortKey == col.Key
	}

	page.SortBy = sortKey
	page.SortDirection = "down"
	if hasDesc {
		page.SortDirection = "up"
	}

	// Construct the page index.
	descParam := ""
	if hasDesc {
		descParam = ";desc"
	}
	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=mlist;sort="+sortKey+descParam,
		start, page.NumMembers, a.SettingInt("defaultMaxMembers"), false)

	// Send the data to the template.
	page.Start = start + 1
	page.End = start + a.SettingInt("defaultMaxMembers")
	if page.End > page.NumMembers {
		page.End = page.NumMembers
	}

	c.PageTitle = c.Txt("308") + " " + itoa(page.Start) + " " + c.Txt("311") + " " + itoa(page.End)
	c.LinkTree = append(c.LinkTree, Link{
		URL:        scripturl + "?action=mlist;sort=" + sortKey + ";start=" + itoa(start),
		Name:       c.PageTitle,
		ExtraAfter: " (" + c.Txt("309") + " " + itoa(page.NumMembers) + " " + c.Txt("310") + ")",
	})

	// List out the different sorting methods...
	onlineHidden := ""
	if !c.allowedTo("moderate_forum") {
		onlineHidden = " OR NOT mem.showOnline"
	}
	emailDown := "mem.emailAddress ASC"
	emailUp := "mem.emailAddress DESC"
	if !c.allowedTo("moderate_forum") && !a.SettingEmpty("allow_hideEmail") {
		emailDown = "mem.hideEmail ASC, mem.emailAddress ASC"
		emailUp = "mem.hideEmail DESC, mem.emailAddress DESC"
	}
	sortMethods := map[string]map[string]string{
		"isOnline": {
			"down": "((lo.logTime IS NULL)" + onlineHidden + ") ASC, realName ASC",
			"up":   "((lo.logTime IS NULL)" + onlineHidden + ") DESC, realName DESC",
		},
		"realName": {
			"down": "mem.realName ASC",
			"up":   "mem.realName DESC",
		},
		"emailAddress": {
			"down": emailDown,
			"up":   emailUp,
		},
		"websiteUrl": {
			"down": "LENGTH(mem.websiteURL) > 0 DESC, (mem.websiteURL IS NULL) ASC, mem.websiteURL ASC",
			"up":   "LENGTH(mem.websiteURL) > 0 ASC, (mem.websiteURL IS NULL) DESC, mem.websiteURL DESC",
		},
		"ICQ": {
			"down": "LENGTH(mem.ICQ) > 0 DESC, (mem.ICQ IS NULL) OR mem.ICQ = 0 ASC, mem.ICQ ASC",
			"up":   "LENGTH(mem.ICQ) > 0 ASC, (mem.ICQ IS NULL) OR mem.ICQ = 0 DESC, mem.ICQ DESC",
		},
		"AIM": {
			"down": "LENGTH(mem.AIM) > 0 DESC, (mem.AIM IS NULL) ASC, mem.AIM ASC",
			"up":   "LENGTH(mem.AIM) > 0 ASC, (mem.AIM IS NULL) DESC, mem.AIM DESC",
		},
		"YIM": {
			"down": "LENGTH(mem.YIM) > 0 DESC, (mem.YIM IS NULL) ASC, mem.YIM ASC",
			"up":   "LENGTH(mem.YIM) > 0 ASC, (mem.YIM IS NULL) DESC, mem.YIM DESC",
		},
		"MSN": {
			"down": "LENGTH(mem.MSN) > 0 DESC, (mem.MSN IS NULL) ASC, mem.MSN ASC",
			"up":   "LENGTH(mem.MSN) > 0 ASC, (mem.MSN IS NULL) DESC, mem.MSN DESC",
		},
		"registered": {
			"down": "mem.dateRegistered ASC",
			"up":   "mem.dateRegistered DESC",
		},
		"ID_GROUP": {
			"down": "(mg.groupName IS NULL) ASC, mg.groupName ASC",
			"up":   "(mg.groupName IS NULL) DESC, mg.groupName DESC",
		},
		"posts": {
			"down": "mem.posts DESC",
			"up":   "mem.posts ASC",
		},
	}

	joinOnline := ""
	if sortKey == "isOnline" {
		joinOnline = `
			LEFT JOIN {$db_prefix}log_online AS lo ON (lo.ID_MEMBER = mem.ID_MEMBER)`
	}
	joinGroup := ""
	if sortKey == "ID_GROUP" {
		joinGroup = `
			LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))`
	}

	// Select the members from the database.
	var members []int
	rows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER
		FROM {$db_prefix}members AS mem` + joinOnline + joinGroup + `
		WHERE mem.is_activated = 1
		ORDER BY ` + sortMethods[sortKey][page.SortDirection] + `
		LIMIT ` + itoa(a.SettingInt("defaultMaxMembers")) + ` OFFSET ` + itoa(start)))
	if err == nil {
		for rows.Next() {
			var m int
			rows.Scan(&m)
			members = append(members, m)
		}
		rows.Close()
	}
	c.printMemberListRows(page, members)

	// Add anchors at the start of each letter.
	if sortKey == "realName" {
		lastLetter := ""
		for _, m := range page.Members {
			thisLetter := ""
			if m.Name != "" {
				thisLetter = strings.ToLower(entitySubstr(m.Name, 0, 1))
			}
			if thisLetter != lastLetter && len(thisLetter) == 1 && thisLetter[0] >= 'a' && thisLetter[0] <= 'z' {
				m.SortLetter = Htmlspecialchars(thisLetter)
				lastLetter = thisLetter
			}
		}
	}
}

// mlSearch is MLSearch(): search for members, or show the search box.
func (c *Ctx) mlSearch(page *MemberlistCtx) {
	a := c.App
	scripturl := a.ScriptURL

	c.PageTitle = c.Txt("mlist_search")

	// They're searching..
	if c.REQUEST.Has("search") && c.REQUEST.Has("fields") {
		search := strings.TrimSpace(c.REQUEST.Str("search"))
		var fieldList []string
		if c.GET.Has("fields") {
			fieldList = strings.Split(c.GET.Str("fields"), ",")
		} else if arr := c.POST.Arr("fields"); arr != nil {
			for _, k := range arr.Keys() {
				fieldList = append(fieldList, arr.Str(k))
			}
		}

		page.HasOldSearch = true
		page.OldSearch = c.REQUEST.Str("search")
		page.OldSearchValue = url.QueryEscape(c.REQUEST.Str("search"))

		// No fields?  Use default...
		if len(fieldList) == 0 {
			fieldList = []string{"name"}
		}
		has := func(f string) bool {
			for _, x := range fieldList {
				if x == f {
					return true
				}
			}
			return false
		}

		// Search for a name?
		var fields []string
		if has("name") {
			fields = []string{"memberName", "realName"}
		}
		// Search for messengers...
		if has("messenger") && (!c.User.IsGuest || a.SettingEmpty("guest_hideContacts")) {
			fields = append(fields, "MSN", "AIM", "ICQ", "YIM")
		}
		// Search for websites.
		if has("website") {
			fields = append(fields, "websiteTitle", "websiteUrl")
		}
		// Search for groups.
		if has("group") {
			fields = append(fields, "IFNULL(groupName, '')")
		}
		// Search for an email address?
		condition := ""
		if has("email") {
			if c.allowedTo("moderate_forum") {
				fields = append(fields, "emailAddress")
			} else {
				fields = append(fields, "(hideEmail = 0 AND emailAddress")
				condition = ")"
			}
		}

		query := "= ''"
		if search != "" {
			esc := strings.NewReplacer("'", "''", "_", `\_`, "%", `\%`, "*", "%").Replace(search)
			query = "LIKE '%" + esc + `%' ESCAPE '\'`
		}

		where := strings.Join(fields, " "+query+" OR ") + " " + query + condition

		numResults := 0
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*)
			FROM {$db_prefix}members AS mem
				LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))
			WHERE ` + where + `
				AND is_activated = 1`)).Scan(&numResults)

		page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=mlist;sa=search;search="+search+";fields="+strings.Join(fieldList, ","),
			c.Start, numResults, a.SettingInt("defaultMaxMembers"), false)
		page.NumMembers = numResults

		// Find the members from the database.
		var members []int
		rows, err := a.DB.Query(a.Q(`
			SELECT mem.ID_MEMBER
			FROM {$db_prefix}members AS mem
				LEFT JOIN {$db_prefix}log_online AS lo ON (lo.ID_MEMBER = mem.ID_MEMBER)
				LEFT JOIN {$db_prefix}membergroups AS mg ON (mg.ID_GROUP = IIF(mem.ID_GROUP = 0, mem.ID_POST_GROUP, mem.ID_GROUP))
			WHERE ` + where + `
				AND is_activated = 1
			LIMIT ` + itoa(a.SettingInt("defaultMaxMembers")) + ` OFFSET ` + itoa(c.Start)))
		if err == nil {
			for rows.Next() {
				var m int
				rows.Scan(&m)
				members = append(members, m)
			}
			rows.Close()
		}
		c.printMemberListRows(page, members)
	} else {
		c.SubTemplate = templateMemberlistSearch
		if c.REQUEST.Has("search") {
			page.OldSearch = Htmlspecialchars(c.REQUEST.Str("search"))
		}
	}

	c.LinkTree = append(c.LinkTree, Link{
		URL:  scripturl + "?action=mlist;sa=search",
		Name: c.PageTitle,
	})
}

// printMemberListRows is printMemberListRows($request).
func (c *Ctx) printMemberListRows(page *MemberlistCtx, members []int) {
	a := c.App

	// Get the most posts.
	mostPosts := 0
	a.DB.QueryRow(a.Q(`SELECT IFNULL(MAX(posts), 0) FROM {$db_prefix}members`)).Scan(&mostPosts)

	// Avoid division by zero...
	if mostPosts == 0 {
		mostPosts = 1
	}

	// Load all the members for display.
	c.loadMemberData(members)

	for _, member := range members {
		if !c.loadMemberContext(member) {
			continue
		}

		m := c.memberCtx[member]
		page.Members = append(page.Members, &MLMember{
			MemberCtx:      m,
			PostPercent:    int(math.Round(float64(m.RealPosts*100) / float64(mostPosts))),
			RegisteredDate: c.strftimeLocal("%Y-%m-%d", m.RegisteredTS),
		})
	}

	if c.SubTemplate == nil {
		c.SubTemplate = templateMemberlistMain
	}
}
