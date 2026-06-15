package app

// Port of Sources/ManageServer.php: the server-settings center
// (?action=serversettings) plus the shared settings-form infrastructure
// (prepareDBSettingContext / saveDBSettings) reused by ModSettings and the
// other Manage* modules. The 'core' tab edited Settings.php; in this port the
// flat smf.conf holds those values and is edited on disk (not via the web), so
// the core tab renders read-only (save disabled) — a documented deviation.

import (
	"encoding/base64"
	"strconv"
)

func init() {
	registerAction("serversettings", (*Ctx).ModifySettings)
	registerAction("serversettings2", (*Ctx).ModifySettings2)
}

// setting reads a modSettings value, honoring per-request overrides.
func (c *Ctx) setting(name string) string {
	if c.settingOverride != nil {
		if v, ok := c.settingOverride[name]; ok {
			return v
		}
	}
	return c.App.Setting(name)
}

// overrideSetting sets a per-request modSettings value (not persisted).
func (c *Ctx) overrideSetting(name, val string) {
	if c.settingOverride == nil {
		c.settingOverride = map[string]string{}
	}
	c.settingOverride[name] = val
}

// settingSpec describes one editable setting (or a separator/title) before it
// is turned into a SettingVar for the template.
type settingSpec struct {
	sep   bool        // horizontal rule
	title string      // titled section (non-var)
	typ   string      // check/text/int/float/password/select/large_text
	name  string      // modSettings key
	data  [][2]string // select options: {value, label}
	label string      // label override when there's no $txt[name]
	size  int
}

// SettingVar is one entry of $context['config_vars'].
type SettingVar struct {
	Separator bool
	Title     string // titled section
	IsVar     bool
	Label     string
	Help      string
	Type      string
	Size      int
	Data      [][2]string
	Name      string
	Value     string
	Disabled  bool
}

// SettingsCtx backs template_show_settings.
type SettingsCtx struct {
	PostURL         string
	SettingsTitle   string
	SettingsMessage string
	SaveDisabled    bool
	ConfigVars      []SettingVar
}

// settingsPage returns the current *SettingsCtx, creating it if needed.
func (c *Ctx) settingsPage() *SettingsCtx {
	if p, ok := c.Page.(*SettingsCtx); ok {
		return p
	}
	p := &SettingsCtx{}
	c.Page = p
	return p
}

// prepareDBSettingContext is prepareDBSettingContext(): turn a spec list into
// $context['config_vars'] from the current modSettings values.
func (c *Ctx) prepareDBSettingContext(specs []settingSpec) {
	page := c.settingsPage()
	for _, s := range specs {
		if s.sep {
			page.ConfigVars = append(page.ConfigVars, SettingVar{Separator: true})
			continue
		}
		if s.typ == "" {
			page.ConfigVars = append(page.ConfigVars, SettingVar{Title: s.title})
			continue
		}
		label := s.label
		if c.TxtHas(s.name) {
			label = c.Txt(s.name)
		}
		help := ""
		// SMF: 'help' => isset($helptxt[$config_var[1]]) ? $config_var[1] : ''.
		if c.HelpTxtHas(s.name) {
			help = s.name
		}
		size := s.size
		if size == 0 && (s.typ == "int" || s.typ == "float") {
			size = 6
		}
		page.ConfigVars = append(page.ConfigVars, SettingVar{
			IsVar:    true,
			Label:    label,
			Help:     help,
			Type:     s.typ,
			Size:     size,
			Data:     s.data,
			Name:     s.name,
			Value:    Htmlspecialchars(c.setting(s.name)),
			Disabled: false,
		})
	}
}

// saveDBSettings is saveDBSettings(): persist posted values per their spec.
func (c *Ctx) saveDBSettings(specs []settingSpec) {
	a := c.App
	setArray := map[string]string{}
	for _, s := range specs {
		if s.sep || s.typ == "" || !c.POST.Has(s.name) {
			continue
		}
		switch s.typ {
		case "check":
			if !empty(c.POST.Str(s.name)) {
				setArray[s.name] = "1"
			} else {
				setArray[s.name] = "0"
			}
		case "select":
			val := c.POST.Str(s.name)
			for _, opt := range s.data {
				if opt[0] == val {
					setArray[s.name] = val
					break
				}
			}
		case "int":
			setArray[s.name] = itoa(atoi(c.POST.Str(s.name)))
		case "float":
			f, _ := strconv.ParseFloat(c.POST.Str(s.name), 64)
			setArray[s.name] = strconv.FormatFloat(f, 'f', -1, 64)
		case "text", "large_text":
			setArray[s.name] = c.POST.Str(s.name)
		case "password":
			if arr := c.POST.Arr(s.name); arr != nil && arr.Str("0") == arr.Str("1") {
				setArray[s.name] = arr.Str("0")
			}
		}
	}
	a.UpdateSettings(setArray)
}

// ModifySettings is ModifySettings(): the server-settings center dispatcher.
func (c *Ctx) ModifySettings() {
	c.isAllowedTo("admin_forum")
	c.checkSession("get", "", true)

	c.adminIndex("edit_settings")
	c.PageTitle = c.Txt("222")
	c.SubTemplate = templateShowSettings

	sa := c.REQUEST.Str("sa")
	if sa != "core" && sa != "other" && sa != "cache" {
		sa = "core"
	}

	scripturl := c.App.ScriptURL
	c.AdminTabs = &AdminTabs{
		Title:       c.Txt("222"),
		Help:        "serversettings",
		Description: c.Txt("347"),
		Tabs: []AdminTab{
			{Href: scripturl + "?action=serversettings;sa=core;sesc=" + c.Sc, Title: c.Txt("core_configuration"), IsSelected: sa == "core"},
			{Href: scripturl + "?action=serversettings;sa=other;sesc=" + c.Sc, Title: c.Txt("other_configuration"), IsSelected: sa == "other"},
			{Href: scripturl + "?action=serversettings;sa=cache;sesc=" + c.Sc, Title: c.Txt("caching_settings"), IsSelected: sa == "cache", IsLast: true},
		},
	}

	switch sa {
	case "other":
		c.ModifyOtherSettings(false)
	case "cache":
		c.ModifyCacheSettings(false)
	default:
		c.ModifyCoreSettings()
	}
}

// ModifySettings2 is ModifySettings2(): route to the right save function.
func (c *Ctx) ModifySettings2() {
	c.isAllowedTo("admin_forum")
	c.checkSession("post", "", true)

	sa := c.REQUEST.Str("sa")
	switch sa {
	case "other":
		c.ModifyOtherSettings(true)
	case "cache":
		c.ModifyCacheSettings(true)
	default:
		// Core settings live in smf.conf (edited on disk) — nothing to save.
		c.redirectExit("action=serversettings;sa=core;sesc=" + c.Sc)
	}
}

// ModifyCoreSettings is ModifyCoreSettings(): the Settings.php editor. In this
// port those values live in smf.conf and are edited on disk, so this renders
// read-only with an explanatory message.
func (c *Ctx) ModifyCoreSettings() {
	a := c.App
	page := c.settingsPage()

	page.PostURL = a.ScriptURL + "?action=serversettings2;sa=core"
	page.SettingsTitle = c.Txt("core_configuration")
	page.SaveDisabled = true
	page.SettingsMessage = `<div align="center"><b>` + c.Txt("settings_not_writable") + `</b></div><br />`

	cfg := a.Config
	cv := func(name, typ, value string, size int) {
		page.ConfigVars = append(page.ConfigVars, SettingVar{
			IsVar: true, Type: typ, Name: name, Size: size, Disabled: true,
			Label: c.Txt(coreLabelKey(name)), Value: Htmlspecialchars(value),
		})
	}
	maint := "0"
	if cfg.Maintenance != 0 {
		maint = itoa(cfg.Maintenance)
	}
	cv("db_name", "text", cfg.DBPath, 0)
	cv("db_prefix", "text", cfg.DBPrefix, 0)
	page.ConfigVars = append(page.ConfigVars, SettingVar{Separator: true})
	page.ConfigVars = append(page.ConfigVars, SettingVar{IsVar: true, Type: "check", Name: "maintenance", Label: c.Txt("348"), Value: maint, Disabled: true})
	cv("mtitle", "text", cfg.MTitle, 36)
	cv("mmessage", "text", cfg.MMessage, 36)
	page.ConfigVars = append(page.ConfigVars, SettingVar{Separator: true})
	cv("mbname", "text", cfg.MbName, 30)
	cv("webmaster_email", "text", cfg.WebmasterEmail, 30)
	cv("cookiename", "text", cfg.CookieName, 20)
	cv("language", "text", cfg.Language, 0)
	page.ConfigVars = append(page.ConfigVars, SettingVar{Separator: true})
	cv("boardurl", "text", cfg.BoardURL, 36)
}

// coreLabelKey maps a core setting name to its $txt label key.
func coreLabelKey(name string) string {
	switch name {
	case "db_name":
		return "smf8"
	case "db_prefix":
		return "smf54"
	case "mtitle":
		return "maintenance1"
	case "mmessage":
		return "maintenance2"
	case "mbname":
		return "350"
	case "webmaster_email":
		return "355"
	case "cookiename":
		return "352"
	case "language":
		return "default_language"
	case "boardurl":
		return "351"
	}
	return name
}

// ModifyOtherSettings is ModifyOtherSettings(): SMTP/cookies/db/session/output.
func (c *Ctx) ModifyOtherSettings(save bool) {
	c.loadLanguage("ModSettings")
	c.loadLanguage("Help")

	specs := []settingSpec{
		{typ: "select", name: "mail_type", data: [][2]string{{"0", c.Txt("mail_type_default")}, {"1", "SMTP"}}},
		{typ: "text", name: "smtp_host"},
		{typ: "text", name: "smtp_port"},
		{typ: "text", name: "smtp_username"},
		{typ: "password", name: "smtp_password"},
		{sep: true},
		{typ: "int", name: "cookieTime"},
		{typ: "check", name: "localCookies"},
		{typ: "check", name: "globalCookies"},
		{sep: true},
		{typ: "int", name: "autoOptDatabase"},
		{typ: "int", name: "autoOptMaxOnline"},
		{typ: "check", name: "autoFixDatabase"},
		{sep: true},
		{typ: "check", name: "enableCompressedOutput"},
		{typ: "check", name: "databaseSession_enable"},
		{typ: "check", name: "databaseSession_loose"},
		{typ: "int", name: "databaseSession_lifetime"},
	}

	if save {
		// Make the SMTP password a little harder to see in a backup etc.
		if arr := c.POST.Arr("smtp_password"); arr != nil && !empty(arr.Str("1")) {
			arr.Set("0", base64.StdEncoding.EncodeToString([]byte(arr.Str("0"))))
			arr.Set("1", base64.StdEncoding.EncodeToString([]byte(arr.Str("1"))))
		}
		c.saveDBSettings(specs)
		c.redirectExit("action=serversettings;sa=other;sesc=" + c.Sc)
	}

	page := c.settingsPage()
	page.PostURL = c.App.ScriptURL + "?action=serversettings2;save;sa=other"
	page.SettingsTitle = c.Txt("other_configuration")
	c.prepareDBSettingContext(specs)
}

// ModifyCacheSettings is ModifyCacheSettings(): cache level + memcached host.
func (c *Ctx) ModifyCacheSettings(save bool) {
	c.loadLanguage("ModSettings")
	c.loadLanguage("Help")

	specs := []settingSpec{
		{typ: "select", name: "cache_enable", data: [][2]string{{"0", c.Txt("cache_off")}, {"1", c.Txt("cache_level1")}, {"2", c.Txt("cache_level2")}, {"3", c.Txt("cache_level3")}}},
		{typ: "text", name: "cache_memcached"},
	}

	if save {
		c.saveDBSettings(specs)
		c.redirectExit("action=serversettings;sa=cache;sesc=" + c.Sc)
	}

	page := c.settingsPage()
	page.PostURL = c.App.ScriptURL + "?action=serversettings2;save;sa=cache"
	page.SettingsTitle = c.Txt("caching_settings")
	// No PHP cache accelerators in this port.
	page.SettingsMessage = phpSprintf(c.Txt("caching_information"), c.Txt("detected_no_caching"))
	c.prepareDBSettingContext(specs)
}
