package app

// Port of Sources/Themes.php SetThemeSettings() and SetThemeOptions() — the
// two theme-manager sub-actions that genuinely configure the DEFAULT theme's
// output (its layout settings and the per-member option defaults), as opposed
// to the filesystem/package sub-actions (list/install/remove/edit/copy) that
// are out of scope for this default-theme-only port.
//
// The field definitions in Themes/default/Settings.template.php (template_options
// and template_settings) become the Go data built by buildThemeOptions /
// buildThemeSettings; the set_options / set_settings / reset_list sub-templates
// (Themes.template.php) are hand-ported below.

import "strings"

// themeFieldOption is one <option> in a 'list' theme setting/option.
type themeFieldOption struct {
	Value string
	Label string
}

// themeField is one row in $context['settings'] / $context['options'].
type themeField struct {
	ID      string
	Label   string
	Desc    string
	Type    string // checkbox | list | text | number
	Options []themeFieldOption
	Default bool
	Value   string
}

// buildThemeOptions ports template_options(): the per-member theme option
// defaults. Each entry is default=true (a "default option"). Types are already
// resolved the way SetThemeOptions would (bool→checkbox, options→list).
func (c *Ctx) buildThemeOptions() []*themeField {
	day := func(i int) string { return c.TxtListItem("days", i) }
	return []*themeField{
		{ID: "show_board_desc", Label: c.Txt("732"), Type: "checkbox", Default: true},
		{ID: "show_children", Label: c.Txt("show_children"), Type: "checkbox", Default: true},
		{ID: "show_no_avatars", Label: c.Txt("show_no_avatars"), Type: "checkbox", Default: true},
		{ID: "show_no_signatures", Label: c.Txt("show_no_signatures"), Type: "checkbox", Default: true},
		{ID: "show_no_censored", Label: c.Txt("show_no_censored"), Type: "checkbox", Default: true},
		{ID: "return_to_post", Label: c.Txt("return_to_post"), Type: "checkbox", Default: true},
		{ID: "no_new_reply_warning", Label: c.Txt("no_new_reply_warning"), Type: "checkbox", Default: true},
		{ID: "view_newest_first", Label: c.Txt("recent_posts_at_top"), Type: "checkbox", Default: true},
		{ID: "view_newest_pm_first", Label: c.Txt("recent_pms_at_top"), Type: "checkbox", Default: true},
		{ID: "popup_messages", Label: c.Txt("popup_messages"), Type: "checkbox", Default: true},
		{ID: "copy_to_outbox", Label: c.Txt("copy_to_outbox"), Type: "checkbox", Default: true},
		{ID: "auto_notify", Label: c.Txt("auto_notify"), Type: "checkbox", Default: true},
		{ID: "calendar_start_day", Label: c.Txt("calendar_start_day"), Type: "list", Default: true,
			Options: []themeFieldOption{{"0", day(0)}, {"1", day(1)}, {"6", day(6)}}},
		{ID: "display_quick_reply", Label: c.Txt("display_quick_reply"), Type: "list", Default: true,
			Options: []themeFieldOption{{"0", c.Txt("display_quick_reply1")}, {"1", c.Txt("display_quick_reply2")}, {"2", c.Txt("display_quick_reply3")}}},
		{ID: "display_quick_mod", Label: c.Txt("display_quick_mod"), Type: "list", Default: true,
			Options: []themeFieldOption{{"0", c.Txt("display_quick_mod_none")}, {"1", c.Txt("display_quick_mod_check")}, {"2", c.Txt("display_quick_mod_image")}}},
	}
}

// buildThemeSettings ports template_settings(): the theme layout settings. The
// smiley-set list is supplied (it depends on modSettings/$txt).
func (c *Ctx) buildThemeSettings(smileySets []themeFieldOption) []*themeField {
	return []*themeField{
		{ID: "header_logo_url", Label: c.Txt("header_logo_url"), Desc: c.Txt("header_logo_url_desc"), Type: "text"},
		{ID: "number_recent_posts", Label: c.Txt("number_recent_posts"), Desc: c.Txt("number_recent_posts_desc"), Type: "number"},
		{ID: "display_who_viewing", Label: c.Txt("who_display_viewing"), Type: "list",
			Options: []themeFieldOption{{"0", c.Txt("who_display_viewing_off")}, {"1", c.Txt("who_display_viewing_numbers")}, {"2", c.Txt("who_display_viewing_names")}}},
		{ID: "smiley_sets_default", Label: c.Txt("smileys_default_set_for_theme"), Type: "list", Options: smileySets},
		{ID: "show_modify", Label: c.Txt("383"), Type: "checkbox"},
		{ID: "show_member_bar", Label: c.Txt("510"), Type: "checkbox"},
		{ID: "linktree_link", Label: c.Txt("522"), Type: "checkbox"},
		{ID: "show_profile_buttons", Label: c.Txt("523"), Type: "checkbox"},
		{ID: "show_mark_read", Label: c.Txt("618"), Type: "checkbox"},
		{ID: "linktree_inline", Label: c.Txt("smf105"), Desc: c.Txt("smf106"), Type: "checkbox"},
		{ID: "show_sp1_info", Label: c.Txt("smf200"), Type: "checkbox"},
		{ID: "allow_no_censored", Label: c.Txt("allow_no_censored"), Type: "checkbox"},
		{ID: "show_bbc", Label: c.Txt("740"), Type: "checkbox"},
		{ID: "additional_options_collapsable", Label: c.Txt("additional_options_collapsable"), Type: "checkbox"},
		{ID: "enable_news", Label: c.Txt("379"), Type: "checkbox"},
		{ID: "show_newsfader", Label: c.Txt("387"), Type: "checkbox"},
		{ID: "newsfader_time", Label: c.Txt("739"), Type: "number"},
		{ID: "show_user_images", Label: c.Txt("384"), Type: "checkbox"},
		{ID: "show_blurb", Label: c.Txt("385"), Type: "checkbox"},
		{ID: "show_latest_member", Label: c.Txt("382"), Type: "checkbox"},
		{ID: "use_image_buttons", Label: c.Txt("521"), Type: "checkbox"},
		{ID: "show_gender", Label: c.Txt("386"), Type: "checkbox"},
		{ID: "hide_post_group", Label: c.Txt("hide_post_group"), Desc: c.Txt("hide_post_group_desc"), Type: "checkbox"},
	}
}

// themeGlobalValues reads the global ($settings, i.e. ID_MEMBER = 0) variables
// for a theme — theme th's rows override theme 1's, matching loadTheme's merge.
func (c *Ctx) themeGlobalValues(th int) map[string]string {
	a := c.App
	vals := map[string]string{}
	rows, err := a.DB.Query(a.Q(`
		SELECT variable, value, ID_THEME
		FROM {$db_prefix}themes
		WHERE ID_MEMBER = 0
			AND ID_THEME IN (1, ?)`), th)
	if err == nil {
		for rows.Next() {
			var variable, value string
			var idTheme int
			rows.Scan(&variable, &value, &idTheme)
			if _, ok := vals[variable]; !ok || idTheme != 1 {
				vals[variable] = value
			}
		}
		rows.Close()
	}
	return vals
}

// themeOptionValues reads the default ($options, ID_MEMBER = -1) option values
// for a theme.
func (c *Ctx) themeOptionValues(th int) map[string]string {
	a := c.App
	vals := map[string]string{}
	rows, err := a.DB.Query(a.Q(`
		SELECT variable, value, ID_THEME
		FROM {$db_prefix}themes
		WHERE ID_MEMBER = -1
			AND ID_THEME IN (1, ?)`), th)
	if err == nil {
		for rows.Next() {
			var variable, value string
			var idTheme int
			rows.Scan(&variable, &value, &idTheme)
			if _, ok := vals[variable]; !ok || idTheme != 1 {
				vals[variable] = value
			}
		}
		rows.Close()
	}
	return vals
}

// implodeThemeVal mirrors PHP's "is_array($val) ? implode(',', $val) : $val".
func implodeThemeVal(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case *PArray:
		var parts []string
		t.Values(func(_ string, item any) {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		})
		return strings.Join(parts, ",")
	}
	return ""
}

// truncTheme replicates MySQL's SUBSTRING(x, 1, n) silent truncation.
func truncTheme(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// SetThemeSettings is SetThemeSettings() (?action=theme;sa=settings): the
// theme's global layout settings (stored as ID_MEMBER = 0 rows).
func (c *Ctx) SetThemeSettings() {
	a := c.App

	th := c.GET.Int("th")
	if !c.GET.Has("th") && c.GET.Has("id") {
		th = c.GET.Int("id")
	}
	if th == 0 {
		c.ThemeAdmin()
		return
	}

	// Select the best fitting tab (the list tab).
	c.selectThemeTab("list")

	c.loadLanguage("Admin")
	c.isAllowedTo("admin_forum")

	// Just for navigation, show some nice bar on the left.
	if c.Theme.Int("theme_id") == th {
		c.adminIndex("edit_theme_settings")
	} else {
		c.adminIndex("manage_themes")
	}

	// Submitting!
	if c.POST.Has("submit") {
		c.checkSession("post", "", true)

		c.saveThemeFields(c.POST.Arr("options"), 0, th)
		c.saveThemeFields(c.POST.Arr("default_options"), 0, 1)

		c.redirectExit("action=theme;sa=settings;th=" + itoa(th) + ";sesc=" + c.Sc)
		return
	}

	c.checkSession("get", "", true)

	// Fetch the smiley sets...
	sets := strings.Split("none,"+a.Setting("smiley_sets_known"), ",")
	setNames := strings.Split(c.Txt("smileys_none")+"\n"+a.Setting("smiley_sets_names"), "\n")
	smileySets := []themeFieldOption{{Value: "", Label: c.Txt("smileys_no_default")}}
	for i, set := range sets {
		name := ""
		if i < len(setNames) {
			name = setNames[i]
		}
		smileySets = append(smileySets, themeFieldOption{Value: set, Label: Htmlspecialchars(name)})
	}

	// The current global values, htmlspecialchars'd (except the url/dir vars).
	vals := c.themeGlobalValues(th)
	vals["theme_id"] = itoa(th)
	vals["actual_theme_url"] = vals["theme_url"]
	vals["actual_images_url"] = vals["images_url"]
	vals["actual_theme_dir"] = vals["theme_dir"]
	for k, v := range vals {
		if k != "theme_url" && k != "theme_dir" && k != "images_url" {
			vals[k] = Htmlspecialchars(v)
		}
	}

	fields := c.buildThemeSettings(smileySets)
	for _, f := range fields {
		f.Value = vals[f.ID]
	}

	c.PageTitle = c.Txt("theme4")
	c.Page = &themeSetForm{ThemeID: th, Name: vals["name"],
		ActualThemeURL: vals["actual_theme_url"], ActualImagesURL: vals["actual_images_url"],
		ActualThemeDir: vals["actual_theme_dir"], Fields: fields}
	c.SubTemplate = templateThemeSetSettings
}

// SetThemeOptions is SetThemeOptions() (?action=theme;sa=reset / sa=options):
// the per-member option defaults and the reset-everyone flows.
func (c *Ctx) SetThemeOptions() {
	a := c.App

	th := c.GET.Int("th")
	if !c.GET.Has("th") && c.GET.Has("id") {
		th = c.GET.Int("id")
	}

	c.isAllowedTo("admin_forum")
	c.adminIndex("manage_themes")

	// No theme chosen yet: show the reset list.
	if th == 0 {
		c.checkSession("get", "", true)
		c.themeResetList()
		return
	}

	// Submit (set the defaults)?
	if c.POST.Has("submit") && empty(c.POST.Str("who")) {
		c.checkSession("post", "", true)

		oldSettings := c.saveThemeFields(c.POST.Arr("options"), -1, th)
		oldSettings = append(oldSettings, c.saveThemeFields(c.POST.Arr("default_options"), -1, 1)...)

		// Are there options in non-default themes set that should be cleared?
		if len(oldSettings) > 0 {
			a.DB.Exec(a.Q(`
				DELETE FROM {$db_prefix}themes
				WHERE ID_THEME != 1
					AND ID_MEMBER = -1
					AND variable IN (`+placeholders(len(oldSettings))+`)`), toAnySlice(oldSettings)...)
		}

		c.redirectExit("action=theme;sesc=" + c.Sc + ";sa=reset")
		return
	} else if c.POST.Has("submit") && c.POST.Str("who") == "1" {
		c.checkSession("post", "", true)

		c.applyOptionsToMembers(c.POST.Arr("default_options"), c.POST.Arr("default_options_master"), 1, true)
		c.applyOptionsToMembers(c.POST.Arr("options"), c.POST.Arr("options_master"), th, false)

		c.redirectExit("action=theme;sesc=" + c.Sc + ";sa=reset")
		return
	} else if !empty(c.GET.Str("who")) && c.GET.Str("who") == "2" {
		c.checkSession("get", "", true)

		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}themes
			WHERE ID_MEMBER > 0
				AND ID_THEME = ?`), th)

		c.redirectExit("action=theme;sesc=" + c.Sc + ";sa=reset")
		return
	}

	c.checkSession("get", "", true)

	c.loadLanguage("Profile")

	options := c.buildThemeOptions()

	reset := !empty(c.REQUEST.Str("who"))
	var optVals map[string]string
	if !reset {
		optVals = c.themeOptionValues(th)
	} else {
		optVals = map[string]string{}
	}
	for _, f := range options {
		f.Value = optVals[f.ID]
	}

	// The form header needs the target theme's id + name.
	name := ""
	a.DB.QueryRow(a.Q(`
		SELECT value
		FROM {$db_prefix}themes
		WHERE ID_MEMBER = 0
			AND ID_THEME = ?
			AND variable = 'name'
		LIMIT 1`), th).Scan(&name)

	c.PageTitle = c.Txt("theme4")
	c.Page = &themeOptForm{ThemeID: th, Name: name, Reset: reset, Fields: options}
	c.SubTemplate = templateThemeSetOptions
}

// saveThemeFields REPLACEs each posted field as an (idMember, idTheme, var, val)
// theme row and returns the list of variable names that were set.
func (c *Ctx) saveThemeFields(opts *PArray, idMember, idTheme int) []string {
	a := c.App
	var names []string
	if opts == nil {
		return names
	}
	opts.Values(func(opt string, val any) {
		names = append(names, opt)
		a.DB.Exec(a.Q(`
			INSERT OR REPLACE INTO {$db_prefix}themes
				(ID_MEMBER, ID_THEME, variable, value)
			VALUES (?, ?, ?, ?)`),
			idMember, idTheme, truncTheme(opt, 255), truncTheme(implodeThemeVal(val), 65534))
	})
	return names
}

// applyOptionsToMembers ports the who==1 (reset all members) per-option master
// loop: master 1 copies the value to every member, master 2 deletes it.
func (c *Ctx) applyOptionsToMembers(opts, master *PArray, idTheme int, isDefault bool) {
	a := c.App
	if opts == nil {
		return
	}
	var oldSettings []string
	opts.Values(func(opt string, val any) {
		m := "0"
		if master != nil {
			m = master.Str(opt)
		}
		switch m {
		case "1":
			a.DB.Exec(a.Q(`
				INSERT OR REPLACE INTO {$db_prefix}themes
					(ID_MEMBER, ID_THEME, variable, value)
				SELECT ID_MEMBER, ?, ?, ?
				FROM {$db_prefix}members`),
				idTheme, truncTheme(opt, 255), truncTheme(implodeThemeVal(val), 65534))
			if isDefault {
				oldSettings = append(oldSettings, opt)
			}
		case "2":
			if isDefault {
				a.DB.Exec(a.Q(`
					DELETE FROM {$db_prefix}themes
					WHERE variable = ?
						AND ID_MEMBER > 0`), opt)
			} else {
				a.DB.Exec(a.Q(`
					DELETE FROM {$db_prefix}themes
					WHERE variable = ?
						AND ID_MEMBER > 0
						AND ID_THEME = ?`), opt, idTheme)
			}
		}
	})

	// Delete options from other themes (default-options path only).
	if isDefault && len(oldSettings) > 0 {
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}themes
			WHERE ID_THEME != 1
				AND ID_MEMBER > 0
				AND variable IN (`+placeholders(len(oldSettings))+`)`), toAnySlice(oldSettings)...)
	}
}

// themeResetItem is one theme in the reset list.
type themeResetItem struct {
	ID                int
	Name              string
	NumDefaultOptions int
	NumMembers        int
}

// themeResetList renders the reset_list sub-template (theme picker for the
// reset/options flows). The PHP Settings.template.php file-existence cull is
// replaced by "keep the default theme + any theme that has members", since the
// single-binary port doesn't ship other themes' template files on disk.
func (c *Ctx) themeResetList() {
	a := c.App

	type themeAgg struct {
		name              string
		numDefaultOptions int
		numMembers        int
	}
	themes := map[int]*themeAgg{}
	order := []int{}
	get := func(id int) *themeAgg {
		if t, ok := themes[id]; ok {
			return t
		}
		t := &themeAgg{}
		themes[id] = t
		order = append(order, id)
		return t
	}

	rows, err := a.DB.Query(a.Q(`
		SELECT ID_THEME, value
		FROM {$db_prefix}themes
		WHERE variable = 'name'
			AND ID_MEMBER = 0
		ORDER BY ID_THEME`))
	if err == nil {
		for rows.Next() {
			var id int
			var name string
			rows.Scan(&id, &name)
			get(id).name = name
		}
		rows.Close()
	}

	rows, err = a.DB.Query(a.Q(`
		SELECT ID_THEME, COUNT(*) AS value
		FROM {$db_prefix}themes
		WHERE ID_MEMBER = -1
		GROUP BY ID_THEME`))
	if err == nil {
		for rows.Next() {
			var id, v int
			rows.Scan(&id, &v)
			get(id).numDefaultOptions = v
		}
		rows.Close()
	}

	rows, err = a.DB.Query(a.Q(`
		SELECT ID_THEME, COUNT(DISTINCT ID_MEMBER) AS value
		FROM {$db_prefix}themes
		WHERE ID_MEMBER > 0
		GROUP BY ID_THEME`))
	if err == nil {
		for rows.Next() {
			var id, v int
			rows.Scan(&id, &v)
			get(id).numMembers = v
		}
		rows.Close()
	}

	var list []themeResetItem
	for _, id := range order {
		t := themes[id]
		// Keep the default theme and anything that actually has members.
		if id != 1 && t.numMembers == 0 {
			continue
		}
		list = append(list, themeResetItem{ID: id, Name: t.name,
			NumDefaultOptions: t.numDefaultOptions, NumMembers: t.numMembers})
	}

	c.Page = &themeResetCtx{Themes: list}
	c.SubTemplate = templateThemeResetList
}

// selectThemeTab marks the named admin tab as selected (the others were set by
// ThemesMain based on the sa value).
func (c *Ctx) selectThemeTab(name string) {
	if c.AdminTabs == nil {
		return
	}
	want := ";sa=" + name
	for i := range c.AdminTabs.Tabs {
		c.AdminTabs.Tabs[i].IsSelected = strings.HasSuffix(c.AdminTabs.Tabs[i].Href, want)
	}
}

// disabledIf returns the disabled attribute when b is true.
func disabledIf(b bool) string {
	if b {
		return ` disabled="disabled"`
	}
	return ""
}

// ternStr is PHP's "b ? yes : no" for strings.
func ternStr(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

// placeholders returns "?, ?, ..." with n entries.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// toAnySlice boxes a []string for variadic Exec args.
func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
