package app

// Port of loadTheme() from Load.php. Only the default theme's templates are
// compiled in (user decision), but theme resolution, $settings/$options
// loading and all the $context setup are kept so behavior and output match.

import (
	"strings"
	"time"
)

func (c *Ctx) loadTheme() {
	c.loadThemeID(0)
}

func (c *Ctx) loadThemeID(idTheme int) {
	a := c.App

	// The theme was specified by parameter / board / request / user...
	switch {
	case idTheme != 0:
		// keep it
	case c.BoardInfo != nil && c.BoardInfo.Theme != 0 && c.BoardInfo.OverrideTheme:
		idTheme = c.BoardInfo.Theme
	case !empty(c.REQUEST.Str("theme")) && (!a.SettingEmpty("theme_allow") || c.allowedTo("admin_forum")):
		idTheme = c.REQUEST.Int("theme")
		c.Session.Set("ID_THEME", idTheme)
	case c.Session.GetInt("ID_THEME") != 0 && (!a.SettingEmpty("theme_allow") || c.allowedTo("admin_forum")):
		idTheme = int(c.Session.GetInt("ID_THEME"))
	case c.User.Theme != 0 && !c.REQUEST.Has("theme") && (!a.SettingEmpty("theme_allow") || c.allowedTo("admin_forum")):
		idTheme = c.User.Theme
	case c.BoardInfo != nil && c.BoardInfo.Theme != 0:
		idTheme = c.BoardInfo.Theme
	default:
		idTheme = a.SettingInt("theme_guests")
	}

	// Verify the ID_THEME... no foul play.
	if a.SettingEmpty("theme_default") && idTheme == 1 && !c.allowedTo("admin_forum") {
		idTheme = a.SettingInt("theme_guests")
	} else if !a.SettingEmpty("knownThemes") && !a.SettingEmpty("theme_allow") && !c.allowedTo("admin_forum") {
		known := false
		for _, t := range strings.Split(a.Setting("knownThemes"), ",") {
			if atoi(t) == idTheme {
				known = true
				break
			}
		}
		if !known {
			idTheme = a.SettingInt("theme_guests")
		}
	}

	member := -1
	if c.User.ID != 0 {
		member = c.User.ID
	}

	// Load variables from the current or default theme, global or this
	// user's ($settings = theme 0-member rows, $options = member rows).
	settings := map[string]string{}
	options := map[string]string{}
	rows, err := a.DB.Query(a.Q(`
		SELECT variable, value, ID_MEMBER, ID_THEME
		FROM {$db_prefix}themes
		WHERE ID_MEMBER IN (-1, 0, ?)
			AND ID_THEME IN (?, 1)`), member, idTheme)
	if err != nil {
		c.fatalDBError(err)
	}
	guestOptions := map[string]string{}
	memberLocked := map[string]bool{} // variables members cannot change
	for _, v := range []string{"actual_theme_url", "actual_images_url", "base_theme_dir", "base_theme_url",
		"default_images_url", "default_theme_dir", "default_theme_url", "default_template", "images_url",
		"number_recent_posts", "smiley_sets_default", "theme_dir", "theme_id", "theme_layers",
		"theme_templates", "theme_url"} {
		memberLocked[v] = true
	}
	type themeRow struct {
		variable, value string
		idMember, theme int
	}
	var all []themeRow
	for rows.Next() {
		var r themeRow
		rows.Scan(&r.variable, &r.value, &r.idMember, &r.theme)
		all = append(all, r)
	}
	rows.Close()
	for _, r := range all {
		if r.idMember != 0 && memberLocked[r.variable] {
			continue
		}
		// If this is the theme_dir/url of the default theme, store it.
		if (r.variable == "theme_dir" || r.variable == "theme_url" || r.variable == "images_url") &&
			r.theme == 1 && r.idMember == 0 {
			settings["default_"+r.variable] = r.value
		}
		switch r.idMember {
		case 0:
			if _, ok := settings[r.variable]; !ok || r.theme != 1 {
				settings[r.variable] = r.value
			}
		case -1:
			if _, ok := guestOptions[r.variable]; !ok || r.theme != 1 {
				guestOptions[r.variable] = r.value
			}
		default:
			if _, ok := options[r.variable]; !ok || r.theme != 1 {
				options[r.variable] = r.value
			}
		}
	}
	// Guest defaults fill member option gaps.
	for k, v := range guestOptions {
		if _, ok := options[k]; !ok {
			options[k] = v
		}
	}

	settings["theme_id"] = itoa(idTheme)
	settings["actual_theme_url"] = settings["theme_url"]
	settings["actual_images_url"] = settings["images_url"]
	settings["actual_theme_dir"] = settings["theme_dir"]

	c.Theme = &ThemeSettings{m: settings}
	c.Options = options

	// Set up the contextual user array + browser detection + layers.
	ua := c.UserAgent
	has := func(s string) bool { return strings.Contains(ua, s) }
	b := Browser{
		IsOpera:     has("Opera"),
		IsOpera6:    has("Opera 6"),
		IsOpera7:    has("Opera 7") || has("Opera/7"),
		IsOpera8:    has("Opera 8") || has("Opera/8"),
		IsIE4:       has("MSIE 4") && !has("WebTV"),
		IsSafari:    has("Safari"),
		IsMacIE:     has("MSIE 5.") && has("Mac"),
		IsWebTV:     has("WebTV"),
		IsKonqueror: has("Konqueror"),
		IsFirefox:   has("Firefox"),
		IsFirefox1:  has("Firefox/1."),
		IsFirefox2:  has("Firefox/2."),
	}
	b.IsGecko = has("Gecko") && !b.IsSafari && !b.IsKonqueror
	notOperaGeckoTV := !b.IsOpera && !b.IsGecko && !b.IsWebTV
	b.IsIE7 = has("MSIE 7") && notOperaGeckoTV
	b.IsIE6 = has("MSIE 6") && notOperaGeckoTV
	b.IsIE55 = has("MSIE 5.5") && notOperaGeckoTV
	b.IsIE5 = has("MSIE 5.0") && notOperaGeckoTV
	b.IsIE = b.IsIE4 || b.IsIE5 || b.IsIE55 || b.IsIE6 || b.IsIE7
	b.NeedsSizeFix = (b.IsIE5 || b.IsIE55 || b.IsIE4 || b.IsOpera6) && !has("Mac")

	// 1.1.x doesn't detect IE8, so render as IE7.
	c.HTMLHeaders += `<meta http-equiv="X-UA-Compatible" content="IE=EmulateIE7" />`

	ci := strings.ToLower(ua)
	b.PossiblyRobot = (!has("Mozilla") && !has("Opera")) ||
		strings.Contains(ci, "googlebot") || strings.Contains(ci, "slurp") || strings.Contains(ci, "crawl")
	action := c.REQUEST.Str("action")
	if action == "login" || action == "login2" || action == "register" || !c.User.IsGuest {
		b.PossiblyRobot = false
	}
	c.Browser = b

	c.MenuSeparator = " "
	if c.Theme.Empty("use_image_buttons") {
		c.MenuSeparator = " | "
	}
	c.CurrentAction = action
	c.CurrentSubaction = c.REQUEST.Str("sa")

	// Determine the current smiley set.
	knownSets := strings.Split(a.Setting("smiley_sets_known"), ",")
	inKnown := false
	for _, s := range knownSets {
		if s == c.User.SmileySet {
			inKnown = true
			break
		}
	}
	if (!inKnown && c.User.SmileySet != "none") || a.SettingEmpty("smiley_sets_enable") {
		if !empty(settings["smiley_sets_default"]) {
			c.User.SmileySet = settings["smiley_sets_default"]
		} else {
			c.User.SmileySet = a.Setting("smiley_sets_default")
		}
	}

	// Set the top level linktree up.
	c.LinkTree = append([]Link{{URL: a.ScriptURL, Name: a.Config.MbName}}, c.LinkTree...)

	// Template layers / language. (XML and 'simple' actions skip the chrome.)
	simpleActions := map[string]bool{"findmember": true, "helpadmin": true, "printpage": true,
		"quotefast": true, "spellcheck": true}
	if c.REQUEST.Has("xml") {
		c.loadLanguage("index")
		c.TemplateLayers = []string{}
	} else if simpleActions[action] {
		c.loadLanguage("index")
		c.TemplateLayers = []string{}
	} else {
		c.TemplateLayers = []string{"main"}
		c.loadLanguage("index")
	}

	// loadSubTemplate('init'): template_init() sets hardcoded theme values.
	applyTemplateInit(c.Theme)

	// Set the character set from the template.
	c.CharacterSet = c.Txt("lang_character_set")
	if !a.SettingEmpty("global_character_set") {
		c.CharacterSet = a.Setting("global_character_set")
	}
	c.RightToLeft = !empty(c.Txt("lang_rtl"))
}

// trackStats / hit counting (trackStats() in Subs.php). Counts are flushed
// per call here — SQLite handles this fine and behavior is identical.
func (c *Ctx) trackStatsHit() {
	a := c.App
	if a.SettingEmpty("trackStats") {
		return
	}
	date := time.Now().Format("2006-01-02")
	res, _ := a.DB.Exec(a.Q(`UPDATE {$db_prefix}log_activity SET hits = hits + 1 WHERE date = ?`), date)
	if res != nil {
		if n, _ := res.RowsAffected(); n == 0 {
			a.DB.Exec(a.Q(`INSERT OR IGNORE INTO {$db_prefix}log_activity (date, hits) VALUES (?, 1)`), date)
		}
	}
}
