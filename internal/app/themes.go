package app

// Port of Sources/Themes.php (Phase 4 subset: the user-facing theme picker
// PickTheme and the SetJavaScript option setter behind ?action=jsoption).
// The admin theme-management sub-actions (admin/list/reset/settings/install/
// remove/edit/copy) arrive with Phase 7.

import (
	"sort"
	"strings"
)

func init() {
	registerAction("theme", (*Ctx).ThemesMain)
	registerAction("jsoption", (*Ctx).SetJavaScript)
}

// ThemesMain is ThemesMain(): the ?action=theme dispatcher.
func (c *Ctx) ThemesMain() {
	// Load the important language files...
	c.loadLanguage("Themes")
	c.loadLanguage("Settings")

	// No funny business - guests only.
	c.isNotGuest("")

	// Default the page title to Theme Administration by default.
	c.PageTitle = c.Txt("themeadmin_title")

	sa := c.GET.Str("sa")
	if sa == "pick" {
		c.PickTheme()
		return
	}

	// Admin tabs (the settings subset wires the 'admin' tab; the filesystem-
	// heavy list/reset/edit/install/etc. sub-actions are out of scope here).
	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("themeadmin_title"), Help: "themes", Description: c.Txt("themeadmin_description")}
	tabs.Tabs = append(tabs.Tabs,
		AdminTab{Title: c.Txt("themeadmin_admin_title"), Description: c.Txt("themeadmin_admin_desc"), Href: scripturl + "?action=theme;sesc=" + c.Sc + ";sa=admin", IsSelected: sa == "admin" || sa == ""},
		AdminTab{Title: c.Txt("themeadmin_list_title"), Description: c.Txt("themeadmin_list_desc"), Href: scripturl + "?action=theme;sesc=" + c.Sc + ";sa=list", IsSelected: sa == "list"},
		AdminTab{Title: c.Txt("themeadmin_reset_title"), Description: c.Txt("themeadmin_reset_desc"), Href: scripturl + "?action=theme;sesc=" + c.Sc + ";sa=reset", IsSelected: sa == "reset"},
		AdminTab{Title: c.Txt("themeadmin_edit_title"), Description: c.Txt("themeadmin_edit_desc"), Href: scripturl + "?action=theme;sesc=" + c.Sc + ";sa=edit", IsSelected: sa == "edit", IsLast: true},
	)
	c.AdminTabs = tabs

	switch sa {
	case "admin", "":
		c.ThemeAdmin()
	case "settings":
		// The default theme's global layout settings.
		c.SetThemeSettings()
	case "reset", "options":
		// Per-member option defaults + reset-everyone flows.
		c.SetThemeOptions()
	default:
		// list/install/remove/edit/copy are the filesystem/theme-package parts
		// of Themes.php, out of the settings subset; surface a notice rather
		// than a hard error.
		c.adminIndex("manage_themes")
		c.SubTemplate = func(c *Ctx) {
			c.O(`
	<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
		<tr class="titlebg"><td>`, c.Txt("themeadmin_title"), `</td></tr>
		<tr class="windowbg2"><td style="padding: 2ex;">This theme-management feature is not available in this port (default theme only).</td></tr>
	</table>`)
		}
	}
}

// ThemeListItem is one theme in $context['themes'].
type ThemeListItem struct {
	ID   int
	Name string
}

// ThemeAdminCtx backs template_main (the theme admin settings page).
type ThemeAdminCtx struct {
	Themes       []ThemeListItem
	ThemeGuests  string
	CanCreateNew bool
	NewThemeDir  string
	NewThemeName string
}

// ThemeAdmin is ThemeAdmin() (?action=theme;sa=admin): the global theme
// settings (allow members to pick / default-on / guests' theme / reset all).
func (c *Ctx) ThemeAdmin() {
	a := c.App
	c.loadLanguage("Admin")
	c.isAllowedTo("admin_forum")
	c.adminIndex("manage_themes")

	if c.POST.Has("submit") {
		c.checkSession("post", "", true)
		opts := c.POST.Arr("options")
		getOpt := func(k string) string {
			if opts != nil {
				return opts.Str(k)
			}
			return ""
		}
		a.UpdateSettings(map[string]string{
			"theme_allow":   getOpt("theme_allow"),
			"theme_default": getOpt("theme_default"),
			"theme_guests":  getOpt("theme_guests"),
		})
		if c.POST.Int("theme_reset") != -1 {
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_THEME = ?`), c.POST.Int("theme_reset"))
		}
		c.redirectExit("action=theme;sesc=" + c.Sc + ";sa=admin")
		return
	}

	c.checkSession("get", "", true)

	page := &ThemeAdminCtx{ThemeGuests: a.Setting("theme_guests")}
	c.Page = page
	rows, err := a.DB.Query(a.Q(`SELECT ID_THEME, value AS name FROM {$db_prefix}themes WHERE variable = 'name' AND ID_MEMBER = 0 ORDER BY ID_THEME`))
	if err == nil {
		for rows.Next() {
			var id int
			var name string
			rows.Scan(&id, &name)
			page.Themes = append(page.Themes, ThemeListItem{ID: id, Name: name})
		}
		rows.Close()
	}
	// Theme creation needs a writable Themes dir; the port ships the default
	// theme only, so this stays off (the install box is hidden).
	page.CanCreateNew = false
	page.NewThemeName = "theme1"

	c.SubTemplate = templateThemeMain
}

// ThemePickCtx is the page context for template_pick.
type ThemePickCtx struct {
	CurrentMember   int
	AvailableThemes []*pickThemeEntry
}

// pickThemeEntry is one row in the theme picker.
type pickThemeEntry struct {
	ID            int
	Selected      bool
	Name          string
	Description   string
	ThumbnailHref string
	NumUsers      int
}

// PickTheme is PickTheme() (?action=theme;sa=pick): let a member (or, for
// admins, anyone) choose their theme.
func (c *Ctx) PickTheme() {
	a := c.App

	c.checkSession("get", "", true)

	c.loadLanguage("Profile")

	c.Session.Set("ID_THEME", 0)

	th := c.GET.Str("th")
	if c.GET.Has("id") {
		th = c.GET.Str("id")
	}

	// Have we made a decision, or are we just browsing?
	if th != "" {
		thID := atoi(th)
		switch {
		case !c.REQUEST.Has("u") || !c.allowedTo("admin_forum"):
			// Save for this user.
			a.updateMemberDataMap(c.User.ID, map[string]any{"ID_THEME": thID})
			c.redirectExit("action=profile;sa=theme")
		case c.REQUEST.Str("u") == "0":
			// For everyone.
			a.DB.Exec(a.Q(`UPDATE {$db_prefix}members SET ID_THEME = ?`), thID)
			c.redirectExit("action=theme;sa=admin;sesc=" + c.Sc)
		case c.REQUEST.Str("u") == "-1":
			// Change the default/guest theme.
			a.UpdateSettings(map[string]string{"theme_guests": itoa(thID)})
			c.redirectExit("action=theme;sa=admin;sesc=" + c.Sc)
		default:
			// Change a specific member's theme.
			a.updateMemberDataMap(c.REQUEST.Int("u"), map[string]any{"ID_THEME": thID})
			c.redirectExit("action=profile;u=" + itoa(c.REQUEST.Int("u")) + ";sa=theme")
		}
	}

	page := &ThemePickCtx{}
	c.Page = page

	// Figure out who the member of the minute is, and what theme they've chosen.
	var currentTheme int
	switch {
	case !c.REQUEST.Has("u") || !c.allowedTo("admin_forum"):
		page.CurrentMember = c.User.ID
		currentTheme = c.User.Theme
	case c.REQUEST.Str("u") == "0":
		page.CurrentMember = 0
		currentTheme = 0
	case c.REQUEST.Str("u") == "-1":
		page.CurrentMember = -1
		currentTheme = a.SettingInt("theme_guests")
	default:
		page.CurrentMember = c.REQUEST.Int("u")
		a.DB.QueryRow(a.Q(`
			SELECT ID_THEME
			FROM {$db_prefix}members
			WHERE ID_MEMBER = ?
			LIMIT 1`), page.CurrentMember).Scan(&currentTheme)
	}

	// Get the theme name and descriptions.
	available := map[int]*pickThemeEntry{}
	themeData := map[int]map[string]string{}
	if !a.SettingEmpty("knownThemes") {
		known := strings.Split(a.Setting("knownThemes"), ",")
		knownList := make([]string, len(known))
		for i, k := range known {
			knownList[i] = itoa(atoi(k))
		}

		// Non-admins can't see un-default'd themes (and never theme 1 directly).
		guestGuard := ""
		if a.SettingEmpty("theme_default") && !c.allowedTo("admin_forum") {
			guestGuard = `
				AND ID_THEME IN (` + strings.Join(knownList, ", ") + `)
				AND ID_THEME != 1`
		}

		rows, err := a.DB.Query(a.Q(`
			SELECT ID_THEME, variable, value
			FROM {$db_prefix}themes
			WHERE variable IN ('name', 'theme_url', 'theme_dir', 'images_url')` + guestGuard + `
				AND ID_THEME != 0
			LIMIT ` + itoa(len(known)*8)))
		if err == nil {
			for rows.Next() {
				var id int
				var variable, value string
				rows.Scan(&id, &variable, &value)
				if _, ok := available[id]; !ok {
					available[id] = &pickThemeEntry{ID: id, Selected: currentTheme == id}
					themeData[id] = map[string]string{}
				}
				themeData[id][variable] = value
			}
			rows.Close()
		}
	}

	// Okay, this is a complicated problem: the default theme is 1, but they
	// aren't allowed to access 1!
	guestTheme := a.SettingInt("theme_guests")
	if _, ok := available[guestTheme]; !ok {
		available[0] = &pickThemeEntry{ID: 0}
		themeData[0] = map[string]string{}
		guestTheme = 0
	}

	// Tally up which theme each member is REALLY using.
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_THEME, COUNT(*) AS theCount
		FROM {$db_prefix}members
		GROUP BY ID_THEME`))
	if err == nil {
		for rows.Next() {
			var idTheme, theCount int
			rows.Scan(&idTheme, &theCount)

			if idTheme == 1 && a.SettingEmpty("theme_default") {
				idTheme = guestTheme
			} else if a.SettingEmpty("theme_allow") {
				idTheme = guestTheme
			}

			if t, ok := available[idTheme]; ok {
				t.NumUsers += theCount
			} else if t, ok := available[guestTheme]; ok {
				t.NumUsers += theCount
			}
		}
		rows.Close()
	}

	// Fill in the name/thumbnail/description for each real theme.
	for id, t := range available {
		// Don't try to load the forum or board default theme's data... it
		// doesn't have any!
		if id == 0 {
			continue
		}
		t.Name = themeData[id]["name"]
		// Only the default theme ships a Settings language file; others fall to
		// the no-Settings branch (thumbnail from images_url, empty description).
		t.ThumbnailHref = themeData[id]["images_url"] + "/thumbnail.gif"
		if id == 1 {
			t.ThumbnailHref = c.txtReplace("theme_thumbnail_href", "images_url", themeData[id]["images_url"])
			t.Description = c.Txt("theme_description")
		}
	}

	// As long as we're not doing the default theme...
	if !c.REQUEST.Has("u") || c.REQUEST.Int("u") >= 0 {
		def := &pickThemeEntry{}
		if guestTheme != 0 {
			if src, ok := available[guestTheme]; ok {
				*def = *src
			}
		}
		def.ID = 0
		def.Name = c.Txt("theme_forum_default")
		def.Selected = currentTheme == 0
		def.Description = c.Txt("theme_global_description")
		available[0] = def
	}

	// ksort: emit the picker in ascending theme-id order.
	ids := make([]int, 0, len(available))
	for id := range available {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		page.AvailableThemes = append(page.AvailableThemes, available[id])
	}

	c.PageTitle = c.Txt("theme_pick")
	c.SubTemplate = templateThemePick
}

// txtReplace returns c.Txt(key) with a single {marker} substituted; used for
// the dynamic theme_thumbnail_href = {images_url}/thumbnail.gif string.
func (c *Ctx) txtReplace(key, marker, value string) string {
	return strings.ReplaceAll(c.Txt(key), "{"+marker+"}", value)
}

// SetJavaScript is SetJavaScript() (?action=jsoption): the 1x1-pixel option
// setter that stores per-member theme options.
func (c *Ctx) SetJavaScript() {
	a := c.App

	// Sorry, guests can't do this.
	if c.User.IsGuest {
		c.exit()
	}

	c.checkSession("get", "", true)

	blankGif := c.Theme.ImagesURL() + "/blank.gif"

	// This good-for-nothing pixel is being used to keep the session alive.
	if empty(c.GET.Str("var")) || !c.GET.Has("val") {
		c.redirectExit(blankGif)
	}

	reservedVars := map[string]bool{
		"actual_theme_url": true, "actual_images_url": true, "base_theme_dir": true,
		"base_theme_url": true, "default_images_url": true, "default_theme_dir": true,
		"default_theme_url": true, "default_template": true, "images_url": true,
		"number_recent_posts": true, "smiley_sets_default": true, "theme_dir": true,
		"theme_id": true, "theme_layers": true, "theme_templates": true, "theme_url": true,
	}

	variable := c.GET.Str("var")

	// Can't change reserved vars.
	if reservedVars[strings.ToLower(variable)] {
		c.redirectExit(blankGif)
	}

	// Use a specific theme?
	themeID := atoi(c.Theme.Get("theme_id"))
	if c.GET.Has("th") {
		themeID = c.GET.Int("th")
	} else if c.GET.Has("id") {
		themeID = c.GET.Int("id")
	}

	// Collapse multi-value (val[]=a&val[]=b) options into a comma list.
	val := c.GET.Str("val")
	if arr, ok := c.GET.Get("val").(*PArray); ok {
		var parts []string
		arr.Values(func(_ string, v any) {
			if s, ok := v.(string); ok {
				parts = append(parts, s)
			}
		})
		val = strings.Join(parts, ",")
	}

	// MySQL silently truncated; mirror it (variable 255, value 65534).
	if len(variable) > 255 {
		variable = variable[:255]
	}
	if len(val) > 65534 {
		val = val[:65534]
	}

	a.DB.Exec(a.Q(`
		INSERT OR REPLACE INTO {$db_prefix}themes
			(ID_THEME, ID_MEMBER, variable, value)
		VALUES (?, ?, ?, ?)`), themeID, c.User.ID, variable, val)

	// Don't output anything...
	c.redirectExit(blankGif)
}
