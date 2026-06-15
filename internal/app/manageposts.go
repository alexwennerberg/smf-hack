package app

// Port of Sources/ManagePosts.php: the Posts and Topics admin
// (?action=postsettings) with posts / bbc / censor / topics sub-tabs. The
// MySQL body-column resize in ModifyPostSettings (mediumtext vs text for the
// fulltext index) is dropped — SQLite TEXT is unbounded (documented).

import "strings"

func init() {
	registerAction("postsettings", (*Ctx).ManagePostSettings)
}

// ManagePostSettings is ManagePostSettings(): the dispatcher.
func (c *Ctx) ManagePostSettings() {
	c.adminIndex("posts_and_topics")

	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"posts":  {c.ModifyPostSettings, "admin_forum"},
		"bbc":    {c.ModifyBBCSettings, "admin_forum"},
		"censor": {c.SetCensor, "moderate_forum"},
		"topics": {c.ModifyTopicSettings, "admin_forum"},
	}

	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		if c.allowedTo("admin_forum") {
			sa = "posts"
		} else {
			sa = "censor"
		}
	}
	c.isAllowedTo(subs[sa].perm)

	c.PageTitle = c.Txt("manageposts_title")
	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("manageposts_title"), Help: "posts_and_topics", Description: c.Txt("manageposts_description")}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs,
			AdminTab{Title: c.Txt("manageposts_settings"), Description: c.Txt("manageposts_settings_description"), Href: scripturl + "?action=postsettings;sa=posts", IsSelected: sa == "posts"},
			AdminTab{Title: c.Txt("manageposts_bbc_settings"), Description: c.Txt("manageposts_bbc_settings_description"), Href: scripturl + "?action=postsettings;sa=bbc", IsSelected: sa == "bbc"})
	}
	if c.allowedTo("moderate_forum") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("135"), Description: c.Txt("141"), Href: scripturl + "?action=postsettings;sa=censor", IsSelected: sa == "censor", IsLast: !c.allowedTo("admin_forum")})
	}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("manageposts_topic_settings"), Description: c.Txt("manageposts_topic_settings_description"), Href: scripturl + "?action=postsettings;sa=topics", IsSelected: sa == "topics", IsLast: true})
	}
	c.AdminTabs = tabs

	subs[sa].fn()
}

// CensorCtx backs template_edit_censored.
type CensorCtx struct {
	Words      [][2]string // {vulgar, proper}
	CensorTest string
}

// SetCensor is SetCensor(): manage the censored word list.
func (c *Ctx) SetCensor() {
	a := c.App

	if c.POST.Has("save_censor") {
		c.checkSession("post", "", true)

		var vulgar, proper []string
		if c.POST.Has("censortext") {
			for _, line := range strings.Split(strings.ReplaceAll(c.POST.Str("censortext"), "\r", ""), "\n") {
				parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
				vulgar = append(vulgar, parts[0])
				if len(parts) > 1 {
					proper = append(proper, parts[1])
				} else {
					proper = append(proper, "")
				}
			}
		} else if cv := c.POST.Arr("censor_vulgar"); cv != nil && c.POST.Arr("censor_proper") != nil {
			cp := c.POST.Arr("censor_proper")
			cv.Values(func(k string, v any) {
				s, _ := v.(string)
				if s == "" {
					return
				}
				vulgar = append(vulgar, s)
				ps, _ := cp.Get(k).(string)
				proper = append(proper, ps)
			})
		}

		whole := "0"
		if c.POST.Has("censorWholeWord") {
			whole = "1"
		}
		ignore := "0"
		if c.POST.Has("censorIgnoreCase") {
			ignore = "1"
		}
		a.UpdateSettings(map[string]string{
			"censor_vulgar":    strings.Join(vulgar, "\n"),
			"censor_proper":    strings.Join(proper, "\n"),
			"censorWholeWord":  whole,
			"censorIgnoreCase": ignore,
		})
	}

	page := &CensorCtx{}
	c.Page = page

	if c.POST.Has("censortest") {
		test := Htmlspecialchars(c.POST.Str("censortest"))
		page.CensorTest = strings.ReplaceAll(c.censorText(test), `"`, "&quot;")
	}

	censorVulgar := strings.Split(a.Setting("censor_vulgar"), "\n")
	censorProper := strings.Split(a.Setting("censor_proper"), "\n")
	for i, vw := range censorVulgar {
		if vw == "" {
			continue
		}
		if strings.TrimSpace(strings.ReplaceAll(vw, "*", " ")) == "" {
			continue
		}
		pw := ""
		if i < len(censorProper) {
			pw = censorProper[i]
		}
		page.Words = append(page.Words, [2]string{Htmlspecialchars(strings.TrimSpace(vw)), Htmlspecialchars(pw)})
	}

	c.SubTemplate = templateEditCensored
	c.PageTitle = c.Txt("135")
}

// PostSettingsCtx backs template_edit_post_settings.
type PostSettingsCtx struct {
	SpellcheckInstalled bool
}

// ModifyPostSettings is ModifyPostSettings().
func (c *Ctx) ModifyPostSettings() {
	a := c.App
	c.SubTemplate = templateEditPostSettings
	c.PageTitle = c.Txt("manageposts_settings")

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)
		// (SQLite TEXT is unbounded; the MySQL body-column resize is dropped.)
		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		a.UpdateSettings(map[string]string{
			"removeNestedQuotes":  bit("removeNestedQuotes"),
			"enableEmbeddedFlash": bit("enableEmbeddedFlash"),
			"enableSpellChecking": bit("enableSpellChecking"),
			"max_messageLength":   itoa(c.POST.Int("max_messageLength")),
			"fixLongWords":        itoa(c.POST.Int("fixLongWords")),
			"topicSummaryPosts":   itoa(c.POST.Int("topicSummaryPosts")),
			"spamWaitTime":        itoa(c.POST.Int("spamWaitTime")),
			"edit_wait_time":      itoa(c.POST.Int("edit_wait_time")),
			"edit_disable_time":   itoa(c.POST.Int("edit_disable_time")),
		})
	}

	// No pspell in this port.
	c.Page = &PostSettingsCtx{SpellcheckInstalled: false}
}

// BBCTag is one tag in the BBC-settings picker.
type BBCTag struct {
	Tag       string
	IsEnabled bool
	ShowHelp  bool
}

// BBCSettingsCtx backs template_edit_bbc_settings.
type BBCSettingsCtx struct {
	Columns     [][]BBCTag
	AllSelected bool
}

// ModifyBBCSettings is ModifyBBCSettings().
func (c *Ctx) ModifyBBCSettings() {
	a := c.App
	c.SubTemplate = templateEditBBCSettings
	c.PageTitle = c.Txt("manageposts_bbc_settings_title")

	// The master tag list (parse_bbc(false)).
	var bbcTags []string
	seen := map[string]bool{}
	for _, code := range c.bbcCodes(nil) {
		if !seen[code.tag] {
			seen[code.tag] = true
			bbcTags = append(bbcTags, code.tag)
		}
	}

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)
		enabled := map[string]bool{}
		if arr := c.POST.Arr("enabledTags"); arr != nil {
			arr.Values(func(k string, v any) {
				s, _ := v.(string)
				enabled[s] = true
			})
		}
		var disabled []string
		for _, tag := range bbcTags {
			if !enabled[tag] {
				disabled = append(disabled, tag)
			}
		}
		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		a.UpdateSettings(map[string]string{
			"enableBBC":      bit("enableBBC"),
			"enablePostHTML": bit("enablePostHTML"),
			"autoLinkUrls":   bit("autoLinkUrls"),
			"disabledBBC":    strings.Join(disabled, ","),
		})
	}

	page := &BBCSettingsCtx{}
	c.Page = page

	var disabledTags []string
	if !a.SettingEmpty("disabledBBC") {
		disabledTags = strings.Split(a.Setting("disabledBBC"), ",")
	}
	const numColumns = 3
	tagsPerColumn := (len(bbcTags) + numColumns - 1) / numColumns
	col := 0
	page.Columns = append(page.Columns, nil)
	for i, tag := range bbcTags {
		if tagsPerColumn > 0 && i%tagsPerColumn == 0 && i != 0 {
			col++
			page.Columns = append(page.Columns, nil)
		}
		page.Columns[col] = append(page.Columns[col], BBCTag{
			Tag:       tag,
			IsEnabled: !inStrings(disabledTags, tag),
			ShowHelp:  c.TxtHas(tag),
		})
	}
	page.AllSelected = len(disabledTags) == 0
}

// ModifyTopicSettings is ModifyTopicSettings().
func (c *Ctx) ModifyTopicSettings() {
	a := c.App
	c.SubTemplate = templateEditTopicSettings
	c.PageTitle = c.Txt("manageposts_topic_settings")

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)
		bit := func(name string) string {
			if c.POST.Has(name) {
				return "1"
			}
			return "0"
		}
		a.UpdateSettings(map[string]string{
			"enableStickyTopics":  bit("enableStickyTopics"),
			"enableParticipation": bit("enableParticipation"),
			"oldTopicDays":        itoa(c.POST.Int("oldTopicDays")),
			"defaultMaxTopics":    itoa(c.POST.Int("defaultMaxTopics")),
			"defaultMaxMessages":  itoa(c.POST.Int("defaultMaxMessages")),
			"hotTopicPosts":       itoa(c.POST.Int("hotTopicPosts")),
			"hotTopicVeryPosts":   itoa(c.POST.Int("hotTopicVeryPosts")),
			"enableAllMessages":   itoa(c.POST.Int("enableAllMessages")),
			"enablePreviousNext":  bit("enablePreviousNext"),
		})
	}
}
