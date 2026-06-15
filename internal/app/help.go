package app

// Port of Sources/Help.php: ShowAdminHelp() — the ?action=helpadmin popup that
// the "?" help icons throughout the admin/settings UI link to — and ShowHelp()
// — the ?action=help user manual (template_manual_* in Help.template.php).

import (
	"os"
	"strings"
)

func init() {
	registerAction("help", (*Ctx).ShowHelp)
	registerAction("helpadmin", (*Ctx).ShowAdminHelp)
	layerFuncs["manual_above"] = templateManualAbove
	layerFuncs["manual_below"] = templateManualBelow
}

// manualPage is one entry of $context['all_pages'] (PHP associative array,
// insertion order preserved): page URL key => the manual_index_<txt> suffix.
type manualPage struct{ URL, Txt string }

var manualPages = []manualPage{
	{"index", "intro"},
	{"registering", "register"},
	{"loginout", "login"},
	{"profile", "profile"},
	{"post", "posting"},
	{"pm", "pm"},
	{"searching", "search"},
}

// manualSubTemplates maps the manual_index_<txt> suffix to its sub-template,
// i.e. $context['sub_template'] = 'manual_' . $all_pages[$page].
var manualSubTemplates = map[string]func(*Ctx){
	"intro":    templateManualIntro,
	"register": templateManualRegister,
	"login":    templateManualLogin,
	"profile":  templateManualProfile,
	"posting":  templateManualPosting,
	"pm":       templateManualPm,
	"search":   templateManualSearch,
}

// ShowHelp is ShowHelp(): the user help center (?action=help).
func (c *Ctx) ShowHelp() {
	c.loadLanguage("Manual")

	// Resolve the requested page; default to 'index' when missing/unknown.
	page := c.GET.Str("page")
	txt := ""
	for _, p := range manualPages {
		if p.URL == page {
			txt = p.Txt
			break
		}
	}
	if txt == "" {
		page = "index"
		txt = "intro"
	}

	c.helpCurrentPage = page
	c.SubTemplate = manualSubTemplates[txt]
	c.TemplateLayers = append(c.TemplateLayers, "manual")
	c.PageTitle = c.Txt("manual_smf_user_help") + ": " + c.Txt("manual_index_"+txt)

	// We actually need a special style sheet for help ;)
	cssBase := c.Theme.DefaultThemeURL()
	if _, err := os.Stat(c.Theme.Get("theme_dir") + "/help.css"); err == nil {
		cssBase = c.Theme.ThemeURL()
	}
	c.HTMLHeaders += `
		<link rel="stylesheet" type="text/css" href="` + cssBase + `/help.css" />`
}

// manualMenuItems builds the &bull;-joined page menu shared by the above/below
// layers: the current page in bold, the others as links.
func (c *Ctx) manualMenuItems() string {
	scripturl := c.App.ScriptURL
	items := make([]string, 0, len(manualPages))
	for _, p := range manualPages {
		if p.URL == c.helpCurrentPage {
			items = append(items, `<span class="error" style="font-weight: bold;">`+c.Txt("manual_index_"+p.Txt)+`</span>`)
		} else {
			items = append(items, `<a href="`+scripturl+`?action=help;page=`+p.URL+`">`+c.Txt("manual_index_"+p.Txt)+`</a>`)
		}
	}
	return strings.Join(items, ` &bull; `)
}

// templateManualAbove is template_manual_above().
func templateManualAbove(c *Ctx) {
	c.O(`
	<div class="tborder" style="margin-top: 1ex;">
		<div id="helpmenu" class="titlebg" style="padding: 4px;">`)
	c.O(c.manualMenuItems())
	c.O(`
		</div>
		<div style="padding: 2ex;" id="helpmain" class="windowbg2">`)
}

// templateManualBelow is template_manual_below().
func templateManualBelow(c *Ctx) {
	c.O(`
		</div>
		<div id="helpmenu2" class="titlebg" style="padding: 4px;">`)
	c.O(c.manualMenuItems())
	c.O(`
		</div>
	</div>`)
}

// HelpCtx is the page context for the help popup.
type HelpCtx struct {
	HelpText string
}

// ShowAdminHelp is ShowAdminHelp(): a popup with one help string.
func (c *Ctx) ShowAdminHelp() {
	help := c.GET.Str("help")

	// Load the admin help language file.
	c.loadLanguage("Help")

	// Permission-specific help lives in the ManagePermissions language file.
	if strings.HasPrefix(help, "permissionhelp") {
		c.loadLanguage("ManagePermissions")
	}

	c.PageTitle = c.App.Config.MbName + " - " + c.Txt("119")
	c.TemplateLayers = []string{}

	// $helptxt[$help] ?? $txt[$help] ?? $help — prefer the $helptxt namespace
	// (the long descriptions), fall back to $txt, then the raw help key.
	helpText := help
	if c.HelpTxtHas(help) {
		helpText = c.HelpTxt(help)
	} else if c.TxtHas(help) {
		helpText = c.Txt(help)
	}
	c.Page = &HelpCtx{HelpText: helpText}
	c.SubTemplate = templatePopup
}

// templatePopup is template_popup() from Help.template.php.
func templatePopup(c *Ctx) {
	page := c.Page.(*HelpCtx)

	rtl := ""
	if c.RightToLeft {
		rtl = ` dir="rtl"`
	}
	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"`, rtl, `>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=`, c.CharacterSet, `" />
		<title>`, c.PageTitle, `</title>
		<link rel="stylesheet" type="text/css" href="`, c.Theme.ThemeURL(), `/style.css" />
		<style type="text/css">`)

	// IE 4/5 and Opera 6 just don't do font sizes properly. (they are bigger...)
	if c.Browser.NeedsSizeFix {
		c.O(`
			@import(`, c.Theme.DefaultThemeURL(), `/fonts-compat.css);`)
	}

	c.O(`
		</style>
	</head>
	<body style="margin: 1ex;">
		<div class="popuptext">
			`, page.HelpText, `<br />
			<br />
			<div align="center"><a href="javascript:self.close();">`, c.Txt("1006"), `</a></div>
		</div>
	</body>
</html>`)
}
