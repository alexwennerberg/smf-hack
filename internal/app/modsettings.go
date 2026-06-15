package app

// Port of Sources/ModSettings.php: the feature/modification settings center
// (?action=featuresettings) with basic / layout / karma tabs. All settings are
// DB-resident and use the ManageServer settings-form infrastructure.

import "strings"

func init() {
	registerAction("featuresettings", (*Ctx).ModifyFeatureSettings)
	registerAction("featuresettings2", (*Ctx).ModifyFeatureSettings2)
}

// ModifyFeatureSettings is ModifyFeatureSettings(): the dispatcher.
func (c *Ctx) ModifyFeatureSettings() {
	c.isAllowedTo("admin_forum")

	c.adminIndex("edit_mods_settings")
	c.loadLanguage("Help")
	c.loadLanguage("ModSettings")

	c.PageTitle = c.Txt("modSettings_title")
	c.SubTemplate = templateShowSettings

	sa := c.REQUEST.Str("sa")
	if sa != "basic" && sa != "layout" && sa != "karma" {
		sa = "basic"
	}

	scripturl := c.App.ScriptURL
	c.AdminTabs = &AdminTabs{
		Title:       c.Txt("modSettings_title"),
		Help:        "modsettings",
		Description: c.Txt("smf3"),
		Tabs: []AdminTab{
			{Href: scripturl + "?action=featuresettings;sa=basic;sesc=" + c.Sc, Title: c.Txt("mods_cat_features"), IsSelected: sa == "basic"},
			{Href: scripturl + "?action=featuresettings;sa=layout;sesc=" + c.Sc, Title: c.Txt("mods_cat_layout"), IsSelected: sa == "layout"},
			{Href: scripturl + "?action=featuresettings;sa=karma;sesc=" + c.Sc, Title: c.Txt("smf293"), IsSelected: sa == "karma", IsLast: true},
		},
	}

	switch sa {
	case "layout":
		c.ModifyLayoutSettings(false)
	case "karma":
		c.ModifyKarmaSettings(false)
	default:
		c.ModifyBasicSettings(false)
	}
}

// ModifyFeatureSettings2 is ModifyFeatureSettings2(): route to the saver.
func (c *Ctx) ModifyFeatureSettings2() {
	c.isAllowedTo("admin_forum")
	c.loadLanguage("ModSettings")
	c.checkSession("post", "", true)

	switch c.REQUEST.Str("sa") {
	case "layout":
		c.ModifyLayoutSettings(true)
	case "karma":
		c.ModifyKarmaSettings(true)
	default:
		c.ModifyBasicSettings(true)
	}
}

// basicSettingsSpecs returns the basic-settings spec list.
func (c *Ctx) basicSettingsSpecs() []settingSpec {
	return []settingSpec{
		{typ: "select", name: "pollMode", data: [][2]string{{"0", c.Txt("smf34")}, {"1", c.Txt("smf32")}, {"2", c.Txt("smf33")}}},
		{sep: true},
		{typ: "check", name: "allow_guestAccess"},
		{typ: "check", name: "userLanguage"},
		{typ: "check", name: "allow_editDisplayName"},
		{typ: "check", name: "allow_hideOnline"},
		{typ: "check", name: "allow_hideEmail"},
		{typ: "check", name: "guest_hideContacts"},
		{typ: "check", name: "titlesEnable"},
		{typ: "check", name: "enable_buddylist"},
		{typ: "text", name: "default_personalText"},
		{typ: "int", name: "max_signatureLength"},
		{sep: true},
		{typ: "text", name: "time_format"},
		{typ: "select", name: "number_format", data: [][2]string{{"1234.00", "1234.00"}, {"1,234.00", "1,234.00"}, {"1.234,00", "1.234,00"}, {"1 234,00", "1 234,00"}, {"1234,00", "1234,00"}}},
		{typ: "float", name: "time_offset"},
		{typ: "int", name: "failed_login_threshold"},
		{typ: "int", name: "lastActive"},
		{typ: "check", name: "trackStats"},
		{typ: "check", name: "hitStats"},
		{typ: "check", name: "enableErrorLogging"},
		{typ: "check", name: "securityDisable"},
		{sep: true},
		{typ: "check", name: "send_validation_onChange"},
		{typ: "check", name: "approveAccountDeletion"},
		{sep: true},
		{typ: "check", name: "allow_disableAnnounce"},
		{typ: "check", name: "disallow_sendBody"},
		{typ: "check", name: "modlog_enabled"},
		{typ: "check", name: "queryless_urls"},
		{sep: true},
		{typ: "int", name: "max_image_width"},
		{typ: "int", name: "max_image_height"},
		{sep: true},
		{typ: "check", name: "enableReportPM"},
	}
}

// ModifyBasicSettings is ModifyBasicSettings().
func (c *Ctx) ModifyBasicSettings(save bool) {
	a := c.App
	specs := c.basicSettingsSpecs()

	if save {
		// Fix PM settings.
		a.UpdateSettings(map[string]string{
			"pm_spam_settings": itoa(c.POST.Int("max_pm_recipients")) + "," + itoa(c.POST.Int("pm_posts_verification")) + "," + itoa(c.POST.Int("pm_posts_per_hour")),
		})
		c.saveDBSettings(specs)
		c.writeLog(false)
		c.redirectExit("action=featuresettings;sa=basic")
	}

	// Hack for PM spam settings: split pm_spam_settings into three ints.
	parts := strings.SplitN(a.Setting("pm_spam_settings"), ",", 3)
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	c.overrideSetting("max_pm_recipients", parts[0])
	c.overrideSetting("pm_posts_verification", parts[1])
	c.overrideSetting("pm_posts_per_hour", parts[2])
	specs = append(specs,
		settingSpec{typ: "int", name: "max_pm_recipients"},
		settingSpec{typ: "int", name: "pm_posts_verification"},
		settingSpec{typ: "int", name: "pm_posts_per_hour"})

	page := c.settingsPage()
	page.PostURL = a.ScriptURL + "?action=featuresettings2;save;sa=basic"
	page.SettingsTitle = c.Txt("mods_cat_features")
	c.prepareDBSettingContext(specs)
}

// ModifyLayoutSettings is ModifyLayoutSettings().
func (c *Ctx) ModifyLayoutSettings(save bool) {
	contiguousLabel := c.Txt("smf235") + `<div class="smalltext">` +
		strings.ReplaceAll(`"3" `+c.Txt("smf236")+`: <b>1 ... 4 [5] 6 ... 9</b>`, " ", "&nbsp;") + `<br />` +
		strings.ReplaceAll(`"5" `+c.Txt("smf236")+`: <b>1 ... 3 4 [5] 6 7 ... 9</b>`, " ", "&nbsp;") + `</div>`
	specs := []settingSpec{
		{typ: "check", name: "compactTopicPagesEnable"},
		{typ: "int", name: "compactTopicPagesContiguous", label: contiguousLabel},
		{sep: true},
		{typ: "select", name: "todayMod", data: [][2]string{{"0", c.Txt("smf290")}, {"1", c.Txt("smf291")}, {"2", c.Txt("smf292")}}},
		{typ: "check", name: "topbottomEnable"},
		{typ: "check", name: "onlineEnable"},
		{typ: "check", name: "enableVBStyleLogin"},
		{sep: true},
		{typ: "int", name: "defaultMaxMembers"},
		{sep: true},
		{typ: "check", name: "timeLoadPageEnable"},
		{typ: "check", name: "disableHostnameLookup"},
		{sep: true},
		{typ: "check", name: "who_enabled"},
	}

	if save {
		c.saveDBSettings(specs)
		c.redirectExit("action=featuresettings;sa=layout")
	}

	page := c.settingsPage()
	page.PostURL = c.App.ScriptURL + "?action=featuresettings2;save;sa=layout"
	page.SettingsTitle = c.Txt("mods_cat_layout")
	c.prepareDBSettingContext(specs)
}

// ModifyKarmaSettings is ModifyKarmaSettings().
func (c *Ctx) ModifyKarmaSettings(save bool) {
	karmaModes := strings.Split(c.Txt("smf64"), "|")
	var karmaData [][2]string
	for i, lbl := range karmaModes {
		karmaData = append(karmaData, [2]string{itoa(i), lbl})
	}
	specs := []settingSpec{
		{typ: "select", name: "karmaMode", data: karmaData},
		{sep: true},
		{typ: "int", name: "karmaMinPosts"},
		{typ: "float", name: "karmaWaitTime"},
		{typ: "check", name: "karmaTimeRestrictAdmins"},
		{sep: true},
		{typ: "text", name: "karmaLabel"},
		{typ: "text", name: "karmaApplaudLabel"},
		{typ: "text", name: "karmaSmiteLabel"},
	}

	if save {
		c.saveDBSettings(specs)
		c.redirectExit("action=featuresettings;sa=karma")
	}

	page := c.settingsPage()
	page.PostURL = c.App.ScriptURL + "?action=featuresettings2;save;sa=karma"
	page.SettingsTitle = c.Txt("smf293")
	c.prepareDBSettingContext(specs)
}
