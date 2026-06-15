package app

// Port of Sources/Search.php PlushSearch1() — the search entry form
// (?action=search) — and PlushSearch2() (?action=search2), the search engine:
// the standard LIKE backend, phrase/weight handling, the buildSearchResults
// prepareSearchContext equivalent, and template_results. The fulltext/custom
// index backends are dropped per the plan (LIKE-only); see PORTING.md Phase 5.

import (
	"encoding/base64"
	"os"
	"regexp"
	"strings"
)

func init() {
	registerAction("search", (*Ctx).PlushSearch1)
	registerAction("search2", (*Ctx).PlushSearch2)
}

// SearchBoardCol is one cell in the two-column board picker: a category
// header, a board checkbox, or an empty filler.
type SearchBoardCol struct {
	Kind       string // "cat", "board", "empty"
	Name       string
	ChildIDs   []int // category select-all
	ID         int   // board
	ChildLevel int
	Selected   bool
}

// SearchCtx is the page context for the search form (template_main).
type SearchCtx struct {
	SearchErrors []string
	SimpleSearch bool
	Search       string
	Userspec     string
	Searchtype   int
	Minage       int
	Maxage       int
	ShowComplete bool
	SubjectOnly  bool
	Topic        int
	TopicLink    string
	TopicID      int
	BoardColumns []*SearchBoardCol
}

// PlushSearch1 is PlushSearch1(): the search form.
func (c *Ctx) PlushSearch1() {
	a := c.App
	scripturl := a.ScriptURL

	c.loadLanguage("Search")
	c.isAllowedTo("search_posts")

	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=search", Name: c.Txt("182")})

	page := &SearchCtx{}
	c.Page = page

	// Decode the parameters preserved across a search2 back-link.
	var brd []int
	haveBrd := false
	advancedSet, advancedVal := false, false
	if c.REQUEST.Has("params") {
		raw := strings.ReplaceAll(c.REQUEST.Str("params"), " ", "+")
		if dec, err := base64.StdEncoding.DecodeString(raw); err == nil {
			for _, data := range strings.Split(string(dec), `|"|`) {
				kv := strings.SplitN(data, `|'|`, 2)
				k := kv[0]
				v := ""
				if len(kv) > 1 {
					v = kv[1]
				}
				switch k {
				case "search":
					page.Search = v
				case "userspec":
					page.Userspec = v
				case "searchtype":
					page.Searchtype = atoi(v)
				case "minage":
					page.Minage = atoi(v)
				case "maxage":
					page.Maxage = atoi(v)
				case "show_complete":
					page.ShowComplete = !empty(v)
				case "subject_only":
					page.SubjectOnly = !empty(v)
				case "topic":
					page.Topic = atoi(v)
				case "advanced":
					advancedSet, advancedVal = true, !empty(v)
				case "brd":
					haveBrd = true
					if v != "" {
						for _, b := range strings.Split(v, ",") {
							brd = append(brd, atoi(b))
						}
					}
				}
			}
		}
	}

	if c.REQUEST.Has("search") {
		page.Search = Htmlspecialchars(unHtmlspecialchars(c.REQUEST.Str("search")))
	}
	if page.Userspec != "" {
		page.Userspec = Htmlspecialchars(page.Userspec)
	}
	if page.Searchtype != 0 {
		page.Searchtype = 2
	}

	// (search_errors come from PlushSearch2's validation, which is deferred.)

	// All the boards this user can see, grouped by category.
	type catEntry struct {
		id     int
		name   string
		boards []*SearchBoardCol
	}
	var cats []*catEntry
	catByID := map[int]*catEntry{}
	rows, err := a.DB.Query(a.Q(`
		SELECT b.ID_CAT, c.name AS catName, b.ID_BOARD, b.name, b.childLevel
		FROM {$db_prefix}boards AS b
			LEFT JOIN {$db_prefix}categories AS c ON (c.ID_CAT = b.ID_CAT)
		WHERE ` + c.User.QuerySeeBoard + `
		ORDER BY c.catOrder, b.boardOrder`))
	if err == nil {
		for rows.Next() {
			var idCat, idBoard, childLevel int
			var catName, name string
			rows.Scan(&idCat, &catName, &idBoard, &name, &childLevel)
			cat, ok := catByID[idCat]
			if !ok {
				cat = &catEntry{id: idCat, name: catName}
				catByID[idCat] = cat
				cats = append(cats, cat)
			}
			selected := (!haveBrd && (a.SettingEmpty("recycle_enable") || idBoard != a.SettingInt("recycle_board"))) ||
				(haveBrd && intIn(brd, idBoard))
			cat.boards = append(cat.boards, &SearchBoardCol{
				Kind: "board", ID: idBoard, Name: name, ChildLevel: childLevel, Selected: selected,
			})
		}
		rows.Close()
	}

	// Flatten: category header followed by its boards.
	var temp []*SearchBoardCol
	for _, cat := range cats {
		var childIDs []int
		for _, b := range cat.boards {
			childIDs = append(childIDs, b.ID)
		}
		temp = append(temp, &SearchBoardCol{Kind: "cat", Name: cat.name, ChildIDs: childIDs})
		temp = append(temp, cat.boards...)
	}

	maxBoards := (len(temp) + 1) / 2
	if maxBoards == 1 {
		maxBoards = 2
	}
	// Alternate into two columns.
	for i := 0; i < maxBoards; i++ {
		page.BoardColumns = append(page.BoardColumns, temp[i])
		if i+maxBoards < len(temp) {
			page.BoardColumns = append(page.BoardColumns, temp[i+maxBoards])
		} else {
			page.BoardColumns = append(page.BoardColumns, &SearchBoardCol{Kind: "empty"})
		}
	}

	// Searching one specific topic?
	if !empty(c.REQUEST.Str("topic")) {
		page.Topic = c.REQUEST.Int("topic")
		page.ShowComplete = true
	}
	if page.Topic != 0 {
		var subject string
		err := a.DB.QueryRow(a.Q(`
			SELECT ms.subject
			FROM ({$db_prefix}topics AS t, {$db_prefix}boards AS b, {$db_prefix}messages AS ms)
			WHERE b.ID_BOARD = t.ID_BOARD
				AND t.ID_TOPIC = ?
				AND ms.ID_MSG = t.ID_FIRST_MSG
				AND `+c.User.QuerySeeBoard+`
			LIMIT 1`), page.Topic).Scan(&subject)
		if err != nil {
			c.fatalLangError("topic_gone", false)
		}
		page.TopicID = page.Topic
		href := scripturl + "?topic=" + itoa(page.Topic) + ".0"
		page.TopicLink = `<a href="` + href + `">` + subject + `</a>`
	}

	// Simple or advanced?
	if advancedSet {
		page.SimpleSearch = !advancedVal
	} else {
		page.SimpleSearch = !a.SettingEmpty("simpleSearch") && !c.REQUEST.Has("advanced")
	}

	c.PageTitle = c.Txt("183")
	c.SubTemplate = templateSearchMain
}

func intIn(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// searchWordSet is one OR-branch's word breakdown.
type searchWordSet struct {
	allWords     []string
	subjectWords []string
}

// blacklistedSearchWords are too-slow-to-search common words.
var blacklistedSearchWords = map[string]bool{
	"img": true, "url": true, "quote": true, "www": true, "http": true,
	"the": true, "is": true, "it": true, "are": true, "if": true,
}

var searchStripRe = regexp.MustCompile(`[\x0b\x00\xa0\t\r\n (){}\[\]<>!@$%^*.,:+=` + "`" + `~?/\\]+|&(amp|lt|gt|quot);`)
var searchPhraseRe = regexp.MustCompile(`(?:^|\s)(-?)"([^"]+)"(?:$|\s)`)

// SearchMatch is one matched message inside a result topic.
type SearchMatch struct {
	ID                 int
	Attachment         []DisplayAttachment
	Alternate          int
	Member             *MemberCtx
	Icon               string
	IconURL            string
	Subject            string
	SubjectHighlighted string
	Time               string
	Counter            int
	ModifiedTime       string
	ModifiedName       string
	Body               string
	BodyHighlighted    string
	Start              string
}

// SearchPostRef is the first/last-post info on a result topic.
type SearchPostRef struct {
	ID         int
	Time       string
	Subject    string
	Href       string
	Link       string
	Icon       string
	IconURL    string
	MemberID   int
	MemberName string
	MemberHref string
	MemberLink string
}

// SearchTopicResult is one topic in the results (≈ prepareSearchContext output).
type SearchTopicResult struct {
	ID          int
	Relevance   string
	NumMatches  int
	IsSticky    bool
	IsLocked    bool
	IsPoll      bool
	IsHot       bool
	IsVeryHot   bool
	PostedIn    bool
	Views       int
	Replies     int
	CanReply    bool
	CanMarkNot  bool
	Class       string
	FirstPost   SearchPostRef
	LastPost    SearchPostRef
	Board       RecentRef
	Category    RecentRef
	Matches     []*SearchMatch
}

// SearchResultsCtx is the page context for template_results.
type SearchResultsCtx struct {
	Compact      bool
	PageIndex    string
	Params       string
	KeyWords     []string
	CanSendPM    bool
	Topics       []*SearchTopicResult
	NoResults    bool
	// Re-shown adjust-query box fields.
	Search       string
	Searchtype   int
	Userspec     string
	ShowComplete bool
	SubjectOnly  bool
	Minage       int
	Maxage       int
	Sort         string
	Brd          []int
}

// PlushSearch2 is PlushSearch2(): run the standard search and show results.
// Only the index-less "standard" backend is implemented (fulltext/custom were
// dropped per the plan); quick-moderation (display_quick_mod, off by default)
// and the no-pspell "did you mean" are not rendered.
func (c *Ctx) PlushSearch2() {
	a := c.App
	scripturl := a.ScriptURL

	c.loadLanguage("Search")
	c.isAllowedTo("search_posts")

	// Weight factors.
	weightFactors := []string{"frequency", "age", "length", "subject", "first_message", "sticky"}
	weight := map[string]int{}
	weightTotal := 0
	for _, wf := range weightFactors {
		weight[wf] = a.SettingInt("search_weight_" + wf)
		weightTotal += weight[wf]
	}
	if weightTotal == 0 {
		c.fatalLangError("search_invalid_weights", false)
	}

	const recentPercentage = 0.30
	const humungousTopicPosts = 200
	maxResults := a.SettingInt("search_max_results")
	if maxResults == 0 {
		maxResults = 200 * a.SettingInt("search_results_per_page")
	}
	perPage := a.SettingInt("search_results_per_page")

	var searchErrors []string

	// --- Decode params / request into the search parameters. ---
	sp := struct {
		search       string
		advanced     bool
		searchtype   int
		minage       int
		maxage       int
		topic        int
		userspec     string
		showComplete bool
		subjectOnly  bool
		sort         string
		sortDir      string
		brd          []int
		brdSet       bool
	}{}

	paramVals := map[string]string{}
	var paramOrder []string
	if c.REQUEST.Has("params") {
		raw := strings.ReplaceAll(c.REQUEST.Str("params"), " ", "+")
		if dec, err := base64.StdEncoding.DecodeString(raw); err == nil {
			for _, data := range strings.Split(string(dec), `|"|`) {
				kv := strings.SplitN(data, `|'|`, 2)
				v := ""
				if len(kv) > 1 {
					v = kv[1]
				}
				paramVals[kv[0]] = v
			}
		}
	}
	get := func(k string) string {
		if v, ok := paramVals[k]; ok {
			return v
		}
		return ""
	}

	sp.advanced = !empty(paramVals["advanced"]) || (!c.REQUEST.Has("params") && !empty(c.REQUEST.Str("advanced")))
	if !empty(get("searchtype")) || c.REQUEST.Int("searchtype") == 2 {
		sp.searchtype = 2
	}
	if !empty(get("minage")) {
		sp.minage = atoi(get("minage"))
	} else if c.REQUEST.Int("minage") > 0 {
		sp.minage = c.REQUEST.Int("minage")
	}
	if !empty(get("maxage")) {
		sp.maxage = atoi(get("maxage"))
	} else if c.REQUEST.Has("maxage") && c.REQUEST.Int("maxage") != 9999 {
		sp.maxage = c.REQUEST.Int("maxage")
	}
	if !empty(c.REQUEST.Str("topic")) {
		sp.topic = c.REQUEST.Int("topic")
		sp.showComplete = true
	} else if !empty(get("topic")) {
		sp.topic = atoi(get("topic"))
	}
	sp.showComplete = sp.showComplete || !empty(get("show_complete")) || !empty(c.REQUEST.Str("show_complete"))
	sp.subjectOnly = !empty(get("subject_only")) || !empty(c.REQUEST.Str("subject_only"))
	if !empty(get("userspec")) {
		sp.userspec = get("userspec")
	} else if u := c.REQUEST.Str("userspec"); u != "" && u != "*" {
		sp.userspec = u
	}

	// Boards from params.
	if v, ok := paramVals["brd"]; ok {
		sp.brdSet = true
		if v != "" {
			for _, b := range strings.Split(v, ",") {
				sp.brd = append(sp.brd, atoi(b))
			}
		}
	}
	_ = paramOrder

	// minMsgID/maxMsgID from the age window.
	minMsgID, maxMsgID := 0, 0
	if sp.minage != 0 || sp.maxage != 0 {
		selMin := "0"
		if sp.maxage != 0 {
			selMin = "IFNULL(MIN(ID_MSG), -1)"
		}
		selMax := "0"
		if sp.minage != 0 {
			selMax = "IFNULL(MAX(ID_MSG), -1)"
		}
		where := "1"
		if sp.minage != 0 {
			where = "posterTime <= " + itoa64(nowUnix()-86400*int64(sp.minage))
		}
		if sp.maxage != 0 {
			where += " AND posterTime >= " + itoa64(nowUnix()-86400*int64(sp.maxage))
		}
		a.DB.QueryRow(a.Q(`SELECT ` + selMin + `, ` + selMax + ` FROM {$db_prefix}messages WHERE ` + where)).Scan(&minMsgID, &maxMsgID)
		if minMsgID < 0 || maxMsgID < 0 {
			searchErrors = append(searchErrors, "no_messages_in_time_frame")
		}
	}

	// User query (poster filter).
	userQuery := ""
	if sp.userspec != "" {
		us := Htmlspecialchars(sp.userspec)
		us = strings.ReplaceAll(us, "&quot;", `"`)
		us = strings.NewReplacer("%", `\%`, "_", `\_`, "*", "%", "?", "_").Replace(us)
		var possible []string
		for _, m := range regexp.MustCompile(`"([^"]+)"`).FindAllStringSubmatch(us, -1) {
			possible = append(possible, m[1])
		}
		for _, p := range strings.Split(regexp.MustCompile(`"([^"]+)"`).ReplaceAllString(us, ""), ",") {
			possible = append(possible, p)
		}
		var clean []string
		for _, p := range possible {
			if p = strings.TrimSpace(p); p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) > 0 {
			likeName := func(col string) string {
				var parts []string
				for _, p := range clean {
					parts = append(parts, col+" LIKE '"+sqlEsc(p)+"' ESCAPE '\\'")
				}
				return strings.Join(parts, " OR ")
			}
			var ids []int
			rows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE ` + likeName("realName")))
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					ids = append(ids, id)
				}
				rows.Close()
			}
			if len(ids) > 500 {
				userQuery = ""
			} else if len(ids) == 0 {
				userQuery = "m.ID_MEMBER = 0 AND (" + likeName("m.posterName") + ")"
			} else {
				userQuery = "(m.ID_MEMBER IN (" + joinInts(ids) + ") OR (m.ID_MEMBER = 0 AND (" + likeName("m.posterName") + ")))"
			}
		}
	}

	// Board scoping.
	reqBrd := sp.brd
	if c.REQUEST.Has("brd") {
		// (brd[] checkboxes from the form)
		reqBrd = nil
		if arr, ok := c.REQUEST.Get("brd").(*PArray); ok {
			arr.Values(func(_ string, v any) {
				if s, ok := v.(string); ok {
					reqBrd = append(reqBrd, atoi(s))
				}
			})
		} else if s := c.REQUEST.Str("brd"); s != "" {
			for _, b := range strings.Split(s, ",") {
				reqBrd = append(reqBrd, atoi(b))
			}
		}
	}

	var brd []int
	if sp.topic != 0 {
		var idBoard int
		err := a.DB.QueryRow(a.Q(`
			SELECT b.ID_BOARD
			FROM ({$db_prefix}topics AS t, {$db_prefix}boards AS b)
			WHERE b.ID_BOARD = t.ID_BOARD AND t.ID_TOPIC = ? AND `+c.User.QuerySeeBoard+`
			LIMIT 1`), sp.topic).Scan(&idBoard)
		if err != nil {
			c.fatalLangError("topic_gone", false)
		}
		brd = []int{idBoard}
	} else if c.User.IsAdmin && (sp.advanced || len(reqBrd) > 0) {
		brd = reqBrd
	} else {
		cond := ""
		if len(reqBrd) > 0 {
			cond = "\n\t\t\t\tAND b.ID_BOARD IN (" + joinInts(reqBrd) + ")"
		} else if !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") > 0 {
			cond = "\n\t\t\t\tAND b.ID_BOARD != " + itoa(a.SettingInt("recycle_board"))
		}
		rows, err := a.DB.Query(a.Q(`SELECT b.ID_BOARD FROM {$db_prefix}boards AS b WHERE ` + c.User.QuerySeeBoard + cond))
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				brd = append(brd, id)
			}
			rows.Close()
		}
		if len(brd) == 0 {
			searchErrors = append(searchErrors, "no_boards_selected")
		}
	}
	sp.brd = brd

	boardQuery := ""
	if len(brd) != 0 {
		var numBoards int
		a.DB.QueryRow(a.Q(`SELECT COUNT(*) FROM {$db_prefix}boards`)).Scan(&numBoards)
		recycle := a.SettingInt("recycle_board")
		if len(brd) == numBoards {
			boardQuery = ""
		} else if len(brd) == numBoards-1 && recycle != 0 && !intIn(brd, recycle) {
			boardQuery = "!= " + itoa(recycle)
		} else {
			boardQuery = "IN (" + joinInts(brd) + ")"
		}
	}

	sp.showComplete = sp.showComplete || !empty(c.REQUEST.Str("show_complete"))
	compact := !sp.showComplete

	// Sort.
	sortColumns := map[string]bool{"relevance": true, "numReplies": true, "ID_MSG": true}
	if get("sort") == "" && c.REQUEST.Str("sort") != "" {
		parts := strings.SplitN(c.REQUEST.Str("sort"), "|", 2)
		sp.sort = parts[0]
		if len(parts) > 1 {
			sp.sortDir = parts[1]
		}
	} else {
		sp.sort = get("sort")
	}
	if !sortColumns[sp.sort] {
		sp.sort = "relevance"
	}
	if sp.topic != 0 && sp.sort == "numReplies" {
		sp.sort = "ID_MSG"
	}
	if sp.sortDir != "asc" {
		sp.sortDir = "desc"
	}

	minMsg := int((1 - recentPercentage) * float64(a.SettingInt("maxMsgID")))
	recentMsg := a.SettingInt("maxMsgID") - minMsg
	if recentMsg == 0 {
		recentMsg = 1
	}

	// --- Parse the query into words/phrases. ---
	if get("search") != "" {
		sp.search = get("search")
	}
	if c.REQUEST.Has("search") {
		sp.search = unHtmlspecialchars(c.REQUEST.Str("search"))
	}
	if strings.TrimSpace(sp.search) == "" {
		searchErrors = append(searchErrors, "invalid_search_string")
	}

	stripped := searchStripRe.ReplaceAllString(sp.search, " ")
	stripped = unHtmlspecialchars(strings.ToLower(stripped))

	var phrases []string
	var excludedWords []string
	for _, m := range searchPhraseRe.FindAllStringSubmatch(stripped, -1) {
		if m[1] == "-" {
			if w := strings.Trim(m[2], "-_' "); w != "" && !blacklistedSearchWords[w] {
				excludedWords = append(excludedWords, w)
			}
		} else {
			phrases = append(phrases, m[2])
		}
	}
	wordPart := searchPhraseRe.ReplaceAllString(stripped, " ")
	var words []string
	for _, w := range strings.Split(wordPart, " ") {
		if strings.HasPrefix(strings.TrimSpace(w), "-") {
			if ww := strings.Trim(w, "-_' "); ww != "" && !blacklistedSearchWords[ww] {
				excludedWords = append(excludedWords, ww)
			}
			continue
		}
		words = append(words, w)
	}

	var searchArray []string
	seen := map[string]bool{}
	for _, v := range append(append([]string{}, phrases...), words...) {
		v = strings.Trim(v, "-_' ")
		if v == "" || blacklistedSearchWords[v] || seen[v] {
			continue
		}
		seen[v] = true
		searchArray = append(searchArray, v)
		if len(searchArray) >= 10 {
			break
		}
	}
	if len(searchArray) == 0 {
		searchErrors = append(searchErrors, "invalid_search_string")
	}

	// OR-branches.
	var orParts [][]string
	if sp.searchtype == 0 {
		if len(searchArray) > 0 {
			orParts = append(orParts, append([]string{}, searchArray...))
		}
	} else {
		for _, v := range searchArray {
			orParts = append(orParts, []string{v})
		}
	}
	for i := range orParts {
		orParts[i] = append(orParts[i], excludedWords...)
	}

	excludedSubjectSet := map[string]bool{}
	var excludedPhrases []string
	searchWords := make([]searchWordSet, len(orParts))
	for oi, andParts := range orParts {
		for _, word := range andParts {
			isExcluded := strInSlice(excludedWords, word)
			searchWords[oi].allWords = append(searchWords[oi].allWords, word)
			subjectWords := text2words(word, 20)
			if !isExcluded || len(subjectWords) == 1 {
				searchWords[oi].subjectWords = append(searchWords[oi].subjectWords, subjectWords...)
				if isExcluded {
					for _, sw := range subjectWords {
						excludedSubjectSet[sw] = true
					}
				}
			} else {
				excludedPhrases = append(excludedPhrases, word)
			}
		}
		if len(searchWords[oi].subjectWords) > 7 {
			searchWords[oi].subjectWords = searchWords[oi].subjectWords[:7]
		}
	}

	// Highlight words.
	page := &SearchResultsCtx{
		Compact:      compact,
		Search:       Htmlspecialchars(sp.search),
		Searchtype:   sp.searchtype,
		Userspec:     Htmlspecialchars(sp.userspec),
		ShowComplete: sp.showComplete,
		SubjectOnly:  sp.subjectOnly,
		Minage:       sp.minage,
		Maxage:       sp.maxage,
		Sort:         sp.sort,
		Brd:          sp.brd,
		KeyWords:     searchArray,
	}
	c.Page = page

	// Encode params for the page-index / link tree.
	var encParts []string
	addParam := func(k, v string) { encParts = append(encParts, k+`|'|`+v) }
	if sp.search != "" {
		addParam("search", sp.search)
	}
	if sp.advanced {
		addParam("advanced", "1")
	}
	if sp.searchtype != 0 {
		addParam("searchtype", "2")
	}
	if sp.minage != 0 {
		addParam("minage", itoa(sp.minage))
	}
	if sp.maxage != 0 {
		addParam("maxage", itoa(sp.maxage))
	}
	if sp.topic != 0 {
		addParam("topic", itoa(sp.topic))
	}
	if sp.userspec != "" {
		addParam("userspec", sp.userspec)
	}
	if sp.showComplete {
		addParam("show_complete", "1")
	}
	if sp.subjectOnly {
		addParam("subject_only", "1")
	}
	if sp.sort != "relevance" || sp.sortDir != "desc" {
		addParam("sort", sp.sort)
	}
	addParam("brd", joinInts2(sp.brd))
	page.Params = base64.StdEncoding.EncodeToString([]byte(strings.Join(encParts, `|"|`)))

	c.LinkTree = append(c.LinkTree,
		Link{URL: scripturl + "?action=search;params=" + page.Params, Name: c.Txt("182")},
		Link{URL: scripturl + "?action=search2;params=" + page.Params, Name: c.Txt("search_results")})

	// Errors? Back to the form.
	if len(searchErrors) > 0 {
		c.loadLanguage("Errors")
		c.REQUEST.Set("params", page.Params)
		c.PlushSearch1()
		// Surface the errors on the (re-shown) form.
		if fp, ok := c.Page.(*SearchCtx); ok {
			for _, e := range searchErrors {
				fp.SearchErrors = append(fp.SearchErrors, c.Txt("error_"+e))
			}
		}
		return
	}

	// --- Run the search into log_search_results (per-session cache). ---
	idSearch, recompute := c.searchCache(page.Params)

	if recompute {
		c.runStandardSearch(idSearch, searchWords, excludedWords, excludedSubjectSet, excludedPhrases,
			userQuery, boardQuery, sp.topic, sp.subjectOnly, minMsgID, maxMsgID,
			weight, weightTotal, minMsg, recentMsg, humungousTopicPosts, maxResults)
	}

	// --- Retrieve this page of results. ---
	c.buildSearchResults(page, idSearch, sp.topic, sp.sort, sp.sortDir, c.REQUEST.Int("start"), perPage)

	numResults := 0
	if v, ok := c.Session.Get("search_num_results").(float64); ok {
		numResults = int(v)
	}
	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=search2;params="+page.Params, c.REQUEST.Int("start"), numResults, perPage, false)
	page.NoResults = len(page.Topics) == 0
	page.CanSendPM = c.allowedTo("pm_send")

	c.PageTitle = c.Txt("166")
	c.SubTemplate = templateSearchResults
}

func strInSlice(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// sqlEsc escapes a string for inclusion in a single-quoted SQL literal.
func sqlEsc(s string) string { return strings.ReplaceAll(s, "'", "''") }

// sqlLikeEsc escapes LIKE wildcards (and the SQL quote) for a '%word%' pattern.
func sqlLikeEsc(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `_`, `\_`, `%`, `\%`).Replace(s)
	return sqlEsc(s)
}

func itoa64(n int64) string { return i64toa(n) }

// searchCache returns the log_search_results ID for these params, and whether
// the results must be (re)computed. Results are cached per session as long as
// the parameters don't change (so pagination reuses them).
func (c *Ctx) searchCache(params string) (idSearch int, recompute bool) {
	a := c.App
	if cached, _ := c.Session.Get("search_cache_params").(string); cached == params {
		if v, ok := c.Session.Get("search_cache_id").(float64); ok {
			return int(v), false
		}
	}
	ptr := a.SettingInt("search_pointer")
	next := ptr + 1
	if ptr >= 255 {
		next = 0
	}
	a.UpdateSettings(map[string]string{"search_pointer": itoa(next)})
	idSearch = ptr
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_search_results WHERE ID_SEARCH = ?`), idSearch)
	c.Session.Set("search_cache_params", params)
	c.Session.Set("search_cache_id", idSearch)
	return idSearch, true
}

// likeCond builds a "col [NOT] LIKE '%word%'" condition; search_match_words is
// off by default so we use LIKE.
func likeCond(col, word string, not bool) string {
	n := ""
	if not {
		n = " NOT"
	}
	return col + n + " LIKE '%" + sqlLikeEsc(word) + "%' ESCAPE '\\'"
}

// runStandardSearch fills log_search_results for idSearch using the index-less
// standard backend, and records the result count in the session.
func (c *Ctx) runStandardSearch(idSearch int, searchWords []searchWordSet, excludedWords []string,
	excludedSubjectSet map[string]bool, excludedPhrases []string, userQuery, boardQuery string,
	topic int, subjectOnly bool, minMsgID, maxMsgID int, weight map[string]int, weightTotal,
	minMsg, recentMsg, humungous, maxResults int) {

	a := c.App

	excludedSet := map[string]bool{}
	for _, w := range excludedWords {
		excludedSet[w] = true
	}

	commonWhere := func() []string {
		var w []string
		if topic != 0 {
			w = append(w, "t.ID_TOPIC = "+itoa(topic))
		}
		if minMsgID != 0 {
			w = append(w, "t.ID_FIRST_MSG >= "+itoa(minMsgID))
		}
		if maxMsgID != 0 {
			w = append(w, "t.ID_LAST_MSG <= "+itoa(maxMsgID))
		}
		if boardQuery != "" {
			w = append(w, "t.ID_BOARD "+boardQuery)
		}
		return w
	}

	subjectRelevance := "1000 * (" +
		itoa(weight["frequency"]) + " * 1.0 / (t.numReplies + 1) + " +
		itoa(weight["age"]) + " * IIF(t.ID_FIRST_MSG < " + itoa(minMsg) + ", 0, (t.ID_FIRST_MSG - " + itoa(minMsg) + ") * 1.0 / " + itoa(recentMsg) + ") + " +
		itoa(weight["length"]) + " * IIF(t.numReplies < " + itoa(humungous) + ", t.numReplies * 1.0 / " + itoa(humungous) + ", 1) + " +
		itoa(weight["subject"]) + " + " +
		itoa(weight["sticky"]) + " * t.isSticky" +
		") / " + itoa(weightTotal)

	numSubjectResults := 0

	runSubjectQuery := func(words searchWordSet, insertSQL, idCol string) int {
		from := []string{"{$db_prefix}topics AS t"}
		var leftJoin, where []string
		hasM := false
		addM := func() {
			if !hasM {
				from = append(from, "{$db_prefix}messages AS m")
				where = append(where, "m.ID_MSG = t.ID_FIRST_MSG")
				hasM = true
			}
		}
		numTables, prevJoin := 0, 0
		for _, sw := range words.subjectWords {
			numTables++
			if excludedSubjectSet[sw] {
				addM()
				leftJoin = append(leftJoin, "{$db_prefix}log_search_subjects AS subj"+itoa(numTables)+" ON (subj"+itoa(numTables)+".word LIKE '%"+sqlLikeEsc(sw)+"%' ESCAPE '\\' AND subj"+itoa(numTables)+".ID_TOPIC = t.ID_TOPIC)")
				where = append(where, "(subj"+itoa(numTables)+".word IS NULL)")
				where = append(where, likeCond("m.body", sw, true))
			} else {
				from = append(from, "{$db_prefix}log_search_subjects AS subj"+itoa(numTables))
				where = append(where, "subj"+itoa(numTables)+".word LIKE '%"+sqlLikeEsc(sw)+"%' ESCAPE '\\'")
				ref := "t"
				if prevJoin != 0 {
					ref = "subj" + itoa(prevJoin)
				}
				where = append(where, "subj"+itoa(numTables)+".ID_TOPIC = "+ref+".ID_TOPIC")
				prevJoin = numTables
			}
		}
		if userQuery != "" {
			addM()
			where = append(where, userQuery)
		}
		where = append(where, commonWhere()...)
		if len(excludedPhrases) > 0 {
			addM()
			for _, ph := range excludedPhrases {
				where = append(where, likeCond("m.subject", ph, true))
			}
		}
		if len(where) == 0 {
			where = append(where, "1")
		}
		join := ""
		if len(leftJoin) > 0 {
			join = "\n\t\t\t\tLEFT JOIN " + strings.Join(leftJoin, "\n\t\t\t\tLEFT JOIN ")
		}
		limit := ""
		if maxResults != 0 {
			limit = "\n\t\t\tLIMIT " + itoa(maxResults-numSubjectResults)
		}
		q := insertSQL + "\n\t\t\tSELECT " + idCol + "\n\t\t\tFROM (" + strings.Join(from, ", ") + ")" + join +
			"\n\t\t\tWHERE " + strings.Join(where, "\n\t\t\t\tAND ") + limit
		if res, _ := a.DB.Exec(a.Q(q)); res != nil {
			n, _ := res.RowsAffected()
			return int(n)
		}
		return 0
	}

	if subjectOnly {
		for _, words := range searchWords {
			insert := `INSERT OR IGNORE INTO {$db_prefix}log_search_results (ID_SEARCH, ID_TOPIC, relevance, ID_MSG, num_matches)`
			idCol := itoa(idSearch) + ", t.ID_TOPIC, " + subjectRelevance + " AS relevance, t.ID_FIRST_MSG, 1"
			numSubjectResults += runSubjectQuery(words, insert, idCol)
			if maxResults != 0 && numSubjectResults >= maxResults {
				break
			}
		}
		c.Session.Set("search_num_results", numSubjectResults)
		return
	}

	if topic == 0 {
		for _, words := range searchWords {
			if len(words.subjectWords) == 0 {
				continue
			}
			insert := `INSERT OR IGNORE INTO {$db_prefix}log_search_topics (ID_SEARCH, ID_TOPIC)`
			idCol := itoa(idSearch) + ", t.ID_TOPIC"
			numSubjectResults += runSubjectQuery(words, insert, idCol)
			if maxResults != 0 && numSubjectResults >= maxResults {
				break
			}
		}
	}

	var orWhere []string
	for _, words := range searchWords {
		var where []string
		for _, rw := range words.allWords {
			where = append(where, likeCond("m.body", rw, excludedSet[rw]))
			if excludedSet[rw] {
				where = append(where, likeCond("m.subject", rw, true))
			}
		}
		if len(where) > 0 {
			if len(where) > 1 {
				orWhere = append(orWhere, "("+strings.Join(where, " AND ")+")")
			} else {
				orWhere = append(orWhere, where[0])
			}
		}
	}

	mainWhere := []string{"t.ID_TOPIC = m.ID_TOPIC"}
	if len(orWhere) > 0 {
		if len(orWhere) > 1 {
			mainWhere = append(mainWhere, "("+strings.Join(orWhere, " OR ")+")")
		} else {
			mainWhere = append(mainWhere, orWhere[0])
		}
	}
	if userQuery != "" {
		mainWhere = append(mainWhere, userQuery)
	}
	if topic != 0 {
		mainWhere = append(mainWhere, "m.ID_TOPIC = "+itoa(topic))
	}
	if minMsgID != 0 {
		mainWhere = append(mainWhere, "m.ID_MSG >= "+itoa(minMsgID))
	}
	if maxMsgID != 0 {
		mainWhere = append(mainWhere, "m.ID_MSG <= "+itoa(maxMsgID))
	}
	if boardQuery != "" {
		mainWhere = append(mainWhere, "m.ID_BOARD "+boardQuery)
	}

	subjectWeightExpr := "0"
	leftJoin := ""
	if numSubjectResults != 0 {
		subjectWeightExpr = "IIF(lst.ID_TOPIC IS NULL, 0, 1)"
		leftJoin = "\n\t\t\tLEFT JOIN {$db_prefix}log_search_topics AS lst ON (lst.ID_SEARCH = " + itoa(idSearch) + " AND lst.ID_TOPIC = t.ID_TOPIC)"
	}

	relevance := "1000 * (" +
		itoa(weight["frequency"]) + " * COUNT(*) * 1.0 / (t.numReplies + 1) + " +
		itoa(weight["age"]) + " * IIF(MAX(m.ID_MSG) < " + itoa(minMsg) + ", 0, (MAX(m.ID_MSG) - " + itoa(minMsg) + ") * 1.0 / " + itoa(recentMsg) + ") + " +
		itoa(weight["length"]) + " * IIF(t.numReplies < " + itoa(humungous) + ", t.numReplies * 1.0 / " + itoa(humungous) + ", 1) + " +
		itoa(weight["subject"]) + " * " + subjectWeightExpr + " + " +
		itoa(weight["first_message"]) + " * IIF(MIN(m.ID_MSG) = t.ID_FIRST_MSG, 1, 0) + " +
		itoa(weight["sticky"]) + " * t.isSticky" +
		") / " + itoa(weightTotal)

	limit := ""
	if maxResults != 0 {
		limit = "\n\t\tLIMIT " + itoa(maxResults)
	}
	q := `INSERT OR IGNORE INTO {$db_prefix}log_search_results (ID_SEARCH, relevance, ID_TOPIC, ID_MSG, num_matches)
		SELECT ` + itoa(idSearch) + `, ` + relevance + ` AS relevance, t.ID_TOPIC, MAX(m.ID_MSG) AS ID_MSG, COUNT(*) AS num_matches
		FROM ({$db_prefix}topics AS t, {$db_prefix}messages AS m)` + leftJoin + `
		WHERE ` + strings.Join(mainWhere, "\n\t\t\tAND ") + `
		GROUP BY t.ID_TOPIC` + limit
	numResults := 0
	if res, _ := a.DB.Exec(a.Q(q)); res != nil {
		n, _ := res.RowsAffected()
		numResults = int(n)
	}

	if numResults < maxResults && numSubjectResults != 0 {
		q := `INSERT OR IGNORE INTO {$db_prefix}log_search_results (ID_SEARCH, ID_TOPIC, relevance, ID_MSG, num_matches)
			SELECT ` + itoa(idSearch) + `, t.ID_TOPIC, ` + subjectRelevance + ` AS relevance, t.ID_FIRST_MSG, 1
			FROM ({$db_prefix}topics AS t, {$db_prefix}log_search_topics AS lst)
			WHERE lst.ID_SEARCH = ` + itoa(idSearch) + ` AND lst.ID_TOPIC = t.ID_TOPIC
			LIMIT ` + itoa(maxResults-numResults)
		if res, _ := a.DB.Exec(a.Q(q)); res != nil {
			n, _ := res.RowsAffected()
			numResults += int(n)
		}
	}

	c.Session.Set("search_num_results", numResults)
}

// highlightKeywords wraps each keyword (with * as a wildcard) in the body in a
// highlight span, skipping HTML tags so markup isn't corrupted.
func (c *Ctx) highlightKeywords(text string, keywords []string, inTags bool) string {
	if len(keywords) == 0 || text == "" {
		return text
	}
	var alts []string
	for _, kw := range keywords {
		q := regexp.QuoteMeta(kw)
		q = strings.ReplaceAll(q, `\*`, `.+?`)
		alts = append(alts, q)
	}
	pat := "(?i)" + strings.Join(alts, "|")
	if inTags {
		// Match a tag OR a keyword; only wrap the keyword.
		re := regexp.MustCompile(`<[^>]*>?|` + pat)
		return re.ReplaceAllStringFunc(text, func(m string) string {
			if strings.HasPrefix(m, "<") {
				return m
			}
			return `<b class="highlight">` + m + `</b>`
		})
	}
	re := regexp.MustCompile(pat)
	return re.ReplaceAllString(text, `<b class="highlight">$0</b>`)
}

// buildSearchResults reads this page of results from log_search_results and
// renders them into page.Topics.
func (c *Ctx) buildSearchResults(page *SearchResultsCtx, idSearch, topic int, sort, sortDir string, start, perPage int) {
	a := c.App
	scripturl := a.ScriptURL

	idCol := "lsr.ID_TOPIC"
	from := "{$db_prefix}log_search_results AS lsr"
	extraWhere := ""
	if topic != 0 {
		idCol = itoa(topic) + " AS ID_TOPIC"
	}
	if sort == "numReplies" {
		from += ", {$db_prefix}topics AS t"
		extraWhere = "\n\t\t\tAND t.ID_TOPIC = lsr.ID_TOPIC"
	}

	type resRow struct {
		idTopic, numMatches int
		relevance           float64
	}
	var order []int
	byMsg := map[int]*resRow{}
	rows, err := a.DB.Query(a.Q(`
		SELECT ` + idCol + `, lsr.ID_MSG, lsr.relevance, lsr.num_matches
		FROM (` + from + `)
		WHERE lsr.ID_SEARCH = ` + itoa(idSearch) + extraWhere + `
		ORDER BY ` + sort + ` ` + sortDir + `
		LIMIT ` + itoa(perPage) + ` OFFSET ` + itoa(start)))
	if err == nil {
		for rows.Next() {
			var idTopic, idMsg, numMatches int
			var relevance float64
			rows.Scan(&idTopic, &idMsg, &relevance, &numMatches)
			order = append(order, idMsg)
			byMsg[idMsg] = &resRow{idTopic: idTopic, numMatches: numMatches, relevance: relevance}
		}
		rows.Close()
	}
	if len(order) == 0 {
		return
	}

	boardsCanReply := c.boardsAllowedTo("post_reply_any")
	boardsCanNotify := c.boardsAllowedTo("mark_any_notify")
	canReplyOn := func(b int) bool { return intIn(boardsCanReply, 0) || intIn(boardsCanReply, b) }
	canNotifyOn := func(b int) bool {
		return intIn(boardsCanNotify, b) || (intIn(boardsCanNotify, 0) && !c.User.IsGuest)
	}

	// Load posters.
	var posters []int
	prows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}messages WHERE ID_MEMBER != 0 AND ID_MSG IN (` + joinInts(order) + `) LIMIT ` + itoa(len(order))))
	if err == nil {
		for prows.Next() {
			var id int
			prows.Scan(&id)
			posters = append(posters, id)
		}
		prows.Close()
	}
	c.loadMemberData(uniqueInts(posters))

	// Participation.
	participants := map[int]bool{}
	if !a.SettingEmpty("enableParticipation") && !c.User.IsGuest {
		seen := map[int]bool{}
		var topicIDs []int
		for _, m := range byMsg {
			if !seen[m.idTopic] {
				seen[m.idTopic] = true
				topicIDs = append(topicIDs, m.idTopic)
			}
		}
		if len(topicIDs) > 0 {
			rows, err := a.DB.Query(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}messages WHERE ID_TOPIC IN (` + joinInts(topicIDs) + `) AND ID_MEMBER = ? GROUP BY ID_TOPIC`), c.User.ID)
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					participants[id] = true
				}
				rows.Close()
			}
		}
	}

	iconSources := map[string]string{}
	for _, icon := range []string{"xx", "thumbup", "thumbdown", "exclamation", "question", "lamp",
		"smiley", "angry", "cheesy", "grin", "sad", "wink", "moved", "recycled", "wireless"} {
		iconSources[icon] = "images_url"
	}
	iconURL := func(icon string) string {
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
		return c.Theme.Get(iconSources[icon]) + "/post/" + icon + ".gif"
	}

	noSubject := c.Txt("24")

	drows, err := a.DB.Query(a.Q(`
		SELECT
			m.ID_MSG, m.subject, m.posterName, m.posterTime, m.ID_MEMBER,
			m.icon, m.body, m.smileysEnabled, m.modifiedTime, m.modifiedName,
			first_m.ID_MSG AS first_msg, first_m.subject AS first_subject, first_m.icon AS firstIcon, first_m.posterTime AS first_posterTime,
			IFNULL(first_mem.ID_MEMBER, 0) AS first_member_id, IFNULL(first_mem.realName, first_m.posterName) AS first_member_name,
			last_m.ID_MSG AS last_msg, last_m.posterTime AS last_posterTime, IFNULL(last_mem.ID_MEMBER, 0) AS last_member_id,
			IFNULL(last_mem.realName, last_m.posterName) AS last_member_name, last_m.icon AS lastIcon, last_m.subject AS last_subject,
			t.ID_TOPIC, t.isSticky, t.locked, t.ID_POLL, t.numReplies, t.numViews,
			b.ID_BOARD, b.name AS bName, c.ID_CAT, c.name AS cName
		FROM ({$db_prefix}messages AS m, {$db_prefix}topics AS t, {$db_prefix}boards AS b, {$db_prefix}categories AS c, {$db_prefix}messages AS first_m, {$db_prefix}messages AS last_m)
			LEFT JOIN {$db_prefix}members AS first_mem ON (first_mem.ID_MEMBER = first_m.ID_MEMBER)
			LEFT JOIN {$db_prefix}members AS last_mem ON (last_mem.ID_MEMBER = first_m.ID_MEMBER)
		WHERE m.ID_MSG IN (` + joinInts(order) + `)
			AND t.ID_TOPIC = m.ID_TOPIC
			AND b.ID_BOARD = t.ID_BOARD
			AND c.ID_CAT = b.ID_CAT
			AND first_m.ID_MSG = t.ID_FIRST_MSG
			AND last_m.ID_MSG = t.ID_LAST_MSG
		ORDER BY FIND_IN_SET(m.ID_MSG, '` + joinInts2(order) + `')
		LIMIT ` + itoa(len(order))))
	if err != nil {
		return
	}
	counter := start + 1
	for drows.Next() {
		var idMsg, idMember, smileys, firstMsg, firstMemberID, lastMsg, lastMemberID int
		var idTopic, isSticky, locked, idPoll, numReplies, numViews, idBoard, idCat int
		var subject, posterName, icon, body, modifiedName string
		var firstSubject, firstMemberName, lastMemberName, lastSubject, bName, cName, firstIcon, lastIcon string
		var posterTime, modifiedTime, firstPosterTime, lastPosterTime int64
		if err := drows.Scan(&idMsg, &subject, &posterName, &posterTime, &idMember,
			&icon, &body, &smileys, &modifiedTime, &modifiedName,
			&firstMsg, &firstSubject, &firstIcon, &firstPosterTime,
			&firstMemberID, &firstMemberName,
			&lastMsg, &lastPosterTime, &lastMemberID,
			&lastMemberName, &lastIcon, &lastSubject,
			&idTopic, &isSticky, &locked, &idPoll, &numReplies, &numViews,
			&idBoard, &bName, &idCat, &cName); err != nil {
			continue
		}

		rr := byMsg[idMsg]
		if rr == nil {
			rr = &resRow{idTopic: idTopic}
		}

		if subject == "" {
			subject = noSubject
		}
		if firstSubject == "" {
			firstSubject = noSubject
		}
		if lastSubject == "" {
			lastSubject = noSubject
		}

		var member *MemberCtx
		if c.loadMemberContext(idMember) {
			member = c.memberCtx[idMember]
		} else {
			member = &MemberCtx{ID: 0, Name: posterName, Group: c.Txt("28"), Link: posterName, IsGuest: true, Options: map[string]string{}}
		}

		body = c.censorText(body)
		subject = c.censorText(subject)
		firstSubject = c.censorText(firstSubject)
		lastSubject = c.censorText(lastSubject)

		if page.Compact {
			body = strings.NewReplacer("\n", " ", "<br />", "\n").Replace(body)
			body = c.parseBBCCached(body, smileys != 0, itoa(idMsg))
			body = stripTags(strings.ReplaceAll(body, "</div>", "<br />"))
			body = c.searchSnippet(body, page.KeyWords)
		} else {
			body = c.parseBBCCached(body, smileys != 0, itoa(idMsg))
		}

		topicResult := &SearchTopicResult{
			ID:         rr.idTopic,
			Relevance:  phpRound(rr.relevance/10, 1) + "%",
			NumMatches: rr.numMatches,
			IsSticky:   !a.SettingEmpty("enableStickyTopics") && isSticky != 0,
			IsLocked:   locked != 0,
			IsPoll:     a.Setting("pollMode") == "1" && idPoll > 0,
			IsHot:      numReplies >= a.SettingInt("hotTopicPosts"),
			IsVeryHot:  numReplies >= a.SettingInt("hotTopicVeryPosts"),
			PostedIn:   participants[rr.idTopic],
			Views:      numViews,
			Replies:    numReplies,
			CanReply:   canReplyOn(idBoard),
			CanMarkNot: canNotifyOn(idBoard),
			Board:      RecentRef{ID: idBoard, Name: bName, Href: scripturl + "?board=" + itoa(idBoard) + ".0", Link: `<a href="` + scripturl + `?board=` + itoa(idBoard) + `.0">` + bName + `</a>`},
			Category:   RecentRef{ID: idCat, Name: cName, Href: scripturl + "#" + itoa(idCat), Link: `<a href="` + scripturl + `#` + itoa(idCat) + `">` + cName + `</a>`},
		}

		topicResult.FirstPost = SearchPostRef{
			ID: firstMsg, Time: c.timeformat(firstPosterTime), Subject: firstSubject,
			Href: scripturl + "?topic=" + itoa(rr.idTopic) + ".0",
			Link: `<a href="` + scripturl + `?topic=` + itoa(rr.idTopic) + `.0">` + firstSubject + `</a>`,
			Icon: firstIcon, IconURL: iconURL(firstIcon), MemberID: firstMemberID, MemberName: firstMemberName,
		}
		if firstMemberID != 0 {
			topicResult.FirstPost.MemberHref = scripturl + "?action=profile;u=" + itoa(firstMemberID)
			topicResult.FirstPost.MemberLink = `<a href="` + topicResult.FirstPost.MemberHref + `" title="` + c.Txt("92") + ` ` + firstMemberName + `">` + firstMemberName + `</a>`
		} else {
			topicResult.FirstPost.MemberLink = firstMemberName
		}
		lastHref := scripturl + "?topic=" + itoa(rr.idTopic)
		if numReplies == 0 {
			lastHref += ".0"
		} else {
			lastHref += ".msg" + itoa(lastMsg)
		}
		lastHref += "#msg" + itoa(lastMsg)
		topicResult.LastPost = SearchPostRef{
			ID: lastMsg, Time: c.timeformat(lastPosterTime), Subject: lastSubject,
			Href: lastHref, Link: `<a href="` + lastHref + `">` + lastSubject + `</a>`,
			Icon: lastIcon, IconURL: iconURL(lastIcon), MemberID: lastMemberID, MemberName: lastMemberName,
		}
		if lastMemberID != 0 {
			topicResult.LastPost.MemberHref = scripturl + "?action=profile;u=" + itoa(lastMemberID)
			topicResult.LastPost.MemberLink = `<a href="` + topicResult.LastPost.MemberHref + `" title="` + c.Txt("92") + ` ` + lastMemberName + `">` + lastMemberName + `</a>`
		} else {
			topicResult.LastPost.MemberLink = lastMemberName
		}

		topicResult.Class = determineTopicClass(topicResult.IsVeryHot, topicResult.IsHot, topicResult.IsPoll, topicResult.IsLocked, topicResult.IsSticky)
		if topicResult.PostedIn {
			topicResult.Class = "my_" + topicResult.Class
		}

		topicResult.Matches = []*SearchMatch{{
			ID:                 idMsg,
			Alternate:          counter % 2,
			Member:             member,
			Icon:               icon,
			IconURL:            iconURL(icon),
			Subject:            subject,
			SubjectHighlighted: c.highlightKeywords(subject, page.KeyWords, false),
			Time:               c.timeformat(posterTime),
			Counter:            counter,
			ModifiedTime:       c.timeformat(modifiedTime),
			ModifiedName:       modifiedName,
			Body:               body,
			BodyHighlighted:    c.highlightKeywords(body, page.KeyWords, true),
			Start:              "msg" + itoa(idMsg),
		}}
		counter++

		page.Topics = append(page.Topics, topicResult)
	}
	drows.Close()
}

// searchSnippet builds the compact-view body excerpt around keyword matches.
// (A faithful-in-spirit port of prepareSearchContext's snippet regex.)
func (c *Ctx) searchSnippet(body string, keywords []string) string {
	const charLimit = 40
	if len(body) <= charLimit {
		return body
	}
	if len(keywords) == 0 {
		if len(body) > charLimit {
			return substr(body, 0, charLimit) + "<b>...</b>"
		}
		return body
	}
	plain := unHtmlspecialchars(strings.NewReplacer("&nbsp;", " ", "<br />", "\n", "&#91;", "[", "&#93;", "]", "&#58;", ":", "&#64;", "@").Replace(body))

	var alts []string
	for _, kw := range keywords {
		q := regexp.QuoteMeta(kw)
		q = strings.ReplaceAll(q, `\*`, `.+?`)
		alts = append(alts, q)
	}
	re := regexp.MustCompile(`(?is)([^\s]{` + itoa(charLimit) + `}\s|\s.{0,` + itoa(charLimit) + `}?|^)(` + strings.Join(alts, "|") + `)(.{0,` + itoa(charLimit) + `}\s|[^\s]{` + itoa(charLimit) + `})`)
	matches := re.FindAllString(plain, -1)
	if len(matches) == 0 {
		return ""
	}
	var out strings.Builder
	for _, m := range matches {
		m = strings.ReplaceAll(Htmlspecialchars(m), "\n", "<br />")
		out.WriteString("<b>...</b>&nbsp;" + m + "&nbsp;<b>...</b><br />")
	}
	return out.String()
}
