package app

// Port of Sources/ManageSearch.php: the search admin (?action=managesearch) —
// settings, relevance weights, and the search-method page. Per the plan the
// port is LIKE-only: MySQL fulltext and the custom word index are dropped, so
// the method page always offers only the standard (no-index) method and the
// create/remove-index sub-actions are no-ops.

import (
	"math"
	"strconv"
)

func init() {
	registerAction("managesearch", (*Ctx).ManageSearch)
}

func (c *Ctx) ManageSearch() {
	c.isAllowedTo("admin_forum")
	c.adminIndex("manage_search")
	c.loadLanguage("Search")

	sa := c.REQUEST.Str("sa")
	valid := map[string]bool{"settings": true, "weights": true, "method": true, "createfulltext": true, "removecustom": true, "removefulltext": true, "createmsgindex": true}
	if !valid[sa] {
		sa = "settings"
	}

	scripturl := c.App.ScriptURL
	methodSelected := sa == "method" || sa == "createfulltext" || sa == "removecustom" || sa == "removefulltext" || sa == "createmsgindex"
	c.AdminTabs = &AdminTabs{
		Title: c.Txt("manage_search"), Help: "search", Description: c.Txt("search_settings_desc"),
		Tabs: []AdminTab{
			{Title: c.Txt("search_weights"), Description: c.Txt("search_weights_desc"), Href: scripturl + "?action=managesearch;sa=weights", IsSelected: sa == "weights"},
			{Title: c.Txt("search_method"), Description: c.Txt("search_method_desc"), Href: scripturl + "?action=managesearch;sa=method", IsSelected: methodSelected},
			{Title: c.Txt("settings"), Description: c.Txt("search_settings_desc"), Href: scripturl + "?action=managesearch;sa=settings", IsSelected: sa == "settings", IsLast: true},
		},
	}

	switch sa {
	case "weights":
		c.EditWeights()
	case "method", "createfulltext", "removecustom", "removefulltext":
		c.EditSearchMethod()
	case "createmsgindex":
		// Custom index dropped — back to the method page.
		c.redirectExit("action=managesearch;sa=method")
	default:
		c.EditSearchSettings()
	}
}

func (c *Ctx) EditSearchSettings() {
	a := c.App
	c.PageTitle = c.Txt("search_settings_title")
	c.SubTemplate = templateSearchSettings

	if c.POST.Has("save") {
		c.checkSession("post", "", true)
		simple := "0"
		if c.POST.Has("simpleSearch") {
			simple = "1"
		}
		a.UpdateSettings(map[string]string{
			"simpleSearch":            simple,
			"search_results_per_page": itoa(c.POST.Int("search_results_per_page")),
			"search_max_results":      itoa(c.POST.Int("search_max_results")),
		})
		c.saveInlinePermissions([]string{"search_posts"})
	}

	c.initInlinePermissions([]string{"search_posts"}, nil)
}

// SearchWeightsCtx backs template_modify_weights.
type SearchWeightsCtx struct {
	Total   int
	Percent map[string]string
}

var searchWeightFactors = []string{"search_weight_frequency", "search_weight_age", "search_weight_length", "search_weight_subject", "search_weight_first_message", "search_weight_sticky"}

func (c *Ctx) EditWeights() {
	a := c.App
	c.PageTitle = c.Txt("search_weights_title")
	c.SubTemplate = templateSearchWeights

	if c.POST.Has("save") {
		c.checkSession("post", "", true)
		changes := map[string]string{}
		for _, f := range searchWeightFactors {
			changes[f] = itoa(c.POST.Int(f))
		}
		a.UpdateSettings(changes)
	}

	page := &SearchWeightsCtx{Percent: map[string]string{}}
	c.Page = page
	for _, f := range searchWeightFactors {
		page.Total += a.SettingInt(f)
	}
	for _, f := range searchWeightFactors {
		pct := 0.0
		if page.Total != 0 {
			pct = 100 * float64(a.SettingInt(f)) / float64(page.Total)
		}
		page.Percent[f] = round1(pct)
	}
}

// round1 formats a float rounded to one decimal place (PHP round($x, 1)).
func round1(x float64) string {
	return strconv.FormatFloat(math.Round(x*10)/10, 'f', -1, 64)
}

// SearchMethodCtx backs template_select_search_method.
type SearchMethodCtx struct {
	DataLength         string
	IndexLength        string
	FulltextLength     string
	CustomIndexLength  string
	HasFulltextIndex   bool
	CannotCreateFull   bool
	CustomIndex        bool
	PartialCustomIndex bool
	DoubleIndex        bool
}

func (c *Ctx) EditSearchMethod() {
	a := c.App
	c.PageTitle = c.Txt("search_method_title")
	c.SubTemplate = templateSelectSearchMethod

	if c.POST.Has("save") {
		c.checkSession("post", "", true)
		// fulltext/custom unavailable -> search_index is always standard ('').
		force := "0"
		if c.POST.Has("search_force_index") {
			force = "1"
		}
		match := "0"
		if c.POST.Has("search_match_words") {
			match = "1"
		}
		a.UpdateSettings(map[string]string{
			"search_index":       "",
			"search_force_index": force,
			"search_match_words": match,
		})
	}
	// createfulltext/removefulltext/removecustom: no-ops (SQLite has no fulltext
	// and the custom word index is dropped) — just render the method page.

	c.Page = &SearchMethodCtx{
		DataLength: "0", IndexLength: "0", FulltextLength: "0", CustomIndexLength: "0",
		CannotCreateFull: true, // no MySQL fulltext in this port
	}
}
