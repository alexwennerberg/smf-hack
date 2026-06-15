package app

// Hand-port of Themes/default/index.template.php — the page chrome
// (main_above/main_below layers, menu, linktree). Byte-identical output is
// the requirement; every echo maps onto c.O with the same literals.

func init() {
	layerFuncs["main_above"] = templateMainAbove
	layerFuncs["main_below"] = templateMainBelow
}

// applyTemplateInit is template_init(): hardcoded theme settings.
func applyTemplateInit(t *ThemeSettings) {
	t.m["use_default_images"] = "never"
	t.m["doctype"] = "xhtml"
	t.m["theme_version"] = "1.1"
	t.m["use_tabs"] = "1"
	t.m["use_buttons"] = "1"
	t.m["seperate_sticky_lock"] = "1"
}

func boolJS(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// cookieNotEmpty mirrors !empty($_COOKIE[name]).
func (c *Ctx) cookieNotEmpty(name string) bool {
	if ck, err := c.R.Cookie(name); err == nil {
		return !empty(ck.Value)
	}
	return false
}

// templateMainAbove is template_main_above().
func templateMainAbove(c *Ctx) {
	scripturl := c.App.ScriptURL

	// Show right to left and the character set for ease of translating.
	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"`)
	if c.RightToLeft {
		c.O(` dir="rtl"`)
	}
	c.O(`><head>
	<meta http-equiv="Content-Type" content="text/html; charset=`, c.CharacterSet, `" />
	<meta name="description" content="`, c.PageTitle, `" />`)
	if c.RobotNoIndex {
		c.O(`
	<meta name="robots" content="noindex" />`)
	}
	c.O(`
	<meta name="keywords" content="PHP, MySQL, bulletin, board, free, open, source, smf, simple, machines, forum" />
	<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/script.js?fin11"></script>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		var smf_theme_url = "`, c.Theme.ThemeURL(), `";
		var smf_images_url = "`, c.Theme.ImagesURL(), `";
		var smf_scripturl = "`, scripturl, `";
		var smf_iso_case_folding = `, boolJS(false), `;
		var smf_charset = "`, c.CharacterSet, `";
	// ]]></script>
	<title>`, c.PageTitle, `</title>`)

	// The ?fin11 part of this link is just here to make sure browsers don't
	// cache it wrongly.
	c.O(`
	<link rel="stylesheet" type="text/css" href="`, c.Theme.ThemeURL(), `/style.css?fin11" />
	<link rel="stylesheet" type="text/css" href="`, c.Theme.DefaultThemeURL(), `/print.css?fin11" media="print" />`)

	if c.Browser.NeedsSizeFix {
		c.O(`
	<link rel="stylesheet" type="text/css" href="`, c.Theme.DefaultThemeURL(), `/fonts-compat.css" />`)
	}

	// Show all the relative links, such as help, search, contents, and the like.
	c.O(`
	<link rel="help" href="`, scripturl, `?action=help" target="_blank" />
	<link rel="search" href="`, scripturl, `?action=search" />
	<link rel="contents" href="`, scripturl, `" />`)

	// If RSS feeds are enabled, advertise the presence of one.
	if !c.App.SettingEmpty("xmlnews_enable") {
		c.O(`
	<link rel="alternate" type="application/rss+xml" title="`, c.App.Config.MbName, ` - RSS" href="`, scripturl, `?type=rss;action=.xml" />`)
	}

	// If we're viewing a topic, these should be the previous and next topics,
	// respectively.
	if c.Topic != 0 {
		c.O(`
	<link rel="prev" href="`, scripturl, `?topic=`, c.Topic, `.0;prev_next=prev" />
	<link rel="next" href="`, scripturl, `?topic=`, c.Topic, `.0;prev_next=next" />`)
	}

	// If we're in a board, or a topic for that matter, the index will be the
	// board's index.
	if c.Board != 0 {
		c.O(`
	<link rel="index" href="`, scripturl, `?board=`, c.Board, `.0" />`)
	}

	// We'll have to use the cookie to remember the header...
	if c.User.IsGuest {
		c.Options["collapse_header"] = boolToOpt(c.cookieNotEmpty("upshrink"))
		c.Options["collapse_header_ic"] = boolToOpt(c.cookieNotEmpty("upshrinkIC"))
	}
	collapseHeader := !empty(c.Options["collapse_header"])
	collapseHeaderIC := !empty(c.Options["collapse_header_ic"])

	// Output any remaining HTML headers. (from mods, maybe?)
	c.O(c.HTMLHeaders, `

	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		var current_header = `, boolJS(collapseHeader), `;

		function shrinkHeader(mode)
		{`)

	// Guests don't have theme options!!
	if c.User.IsGuest {
		c.O(`
			document.cookie = "upshrink=" + (mode ? 1 : 0);`)
	} else {
		c.O(`
			smf_setThemeOption("collapse_header", mode ? 1 : 0, null, "`, c.Sc, `");`)
	}

	c.O(`
			document.getElementById("upshrink").src = smf_images_url + (mode ? "/upshrink2.gif" : "/upshrink.gif");

			document.getElementById("upshrinkHeader").style.display = mode ? "none" : "";
			document.getElementById("upshrinkHeader2").style.display = mode ? "none" : "";

			current_header = mode;
		}
	// ]]></script>`)

	// the routine for the info center upshrink
	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var current_header_ic = `, boolJS(collapseHeaderIC), `;

			function shrinkHeaderIC(mode)
			{`)

	if c.User.IsGuest {
		c.O(`
				document.cookie = "upshrinkIC=" + (mode ? 1 : 0);`)
	} else {
		c.O(`
				smf_setThemeOption("collapse_header_ic", mode ? 1 : 0, null, "`, c.Sc, `");`)
	}

	c.O(`
				document.getElementById("upshrink_ic").src = smf_images_url + (mode ? "/expand.gif" : "/collapse.gif");

				document.getElementById("upshrinkHeaderIC").style.display = mode ? "none" : "";

				current_header_ic = mode;
			}
		// ]]></script>
</head>
<body>`)

	c.O(`
	<div class="tborder" `)
	if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
		c.O(` style="width: 100%;"`)
	}
	c.O(`>
		<table width="100%" cellpadding="0" cellspacing="0" border="0">
			<tr>
				<td class="catbg" height="32">`)

	if c.Theme.Empty("header_logo_url") {
		c.O(`
					<span style="font-family: Verdana, sans-serif; font-size: 140%; ">`, c.App.Config.MbName, `</span>`)
	} else {
		c.O(`
					<img src="`, c.Theme.Get("header_logo_url"), `" style="margin: 4px;" alt="`, c.App.Config.MbName, `" />`)
	}

	c.O(`
				</td>
				<td align="right" class="catbg">
					<img src="`, c.Theme.ImagesURL(), `/smflogo.gif" style="margin: 2px;" alt="" />
				</td>
			</tr>
		</table>`)

	// display user name
	c.O(`
		<table width="100%" cellpadding="0" cellspacing="0" border="0" >
			<tr>`)

	if !c.User.IsGuest {
		c.O(`
				<td class="titlebg2" height="32">
					<span style="font-size: 130%;"> `, c.Txt("hello_member_ndt"), ` <b>`, c.User.Name, `</b></span>
				</td>`)
	}

	// display the time
	c.O(`
				<td class="titlebg2" height="32" align="right">
					<span class="smalltext">`, c.CurrentTime, `</span>`)

	// this is the upshrink button for the user info section
	upshrinkGif := "upshrink.gif"
	if collapseHeader {
		upshrinkGif = "upshrink2.gif"
	}
	c.O(`
					<a href="#" onclick="shrinkHeader(!current_header); return false;"><img id="upshrink" src="`, c.Theme.ImagesURL(), `/`, upshrinkGif, `" alt="*" title="`, c.Txt("upshrink_description"), `" align="bottom" style="margin: 0 1ex;" /></a>
				</td>
			</tr>
			<tr id="upshrinkHeader"`)
	if collapseHeader {
		c.O(` style="display: none;"`)
	}
	c.O(`>
				<td valign="top" colspan="2">
					<table width="100%" class="bordercolor" cellpadding="8" cellspacing="1" border="0" style="margin-top: 1px;">
						<tr>`)

	if c.UserAvatar.Image != "" {
		c.O(`
							<td class="windowbg" valign="middle">`, c.UserAvatar.Image, `</td>`)
	}

	c.O(`
							<td colspan="2" width="100%" valign="top" class="windowbg2"><span class="middletext">`)

	// If the user is logged in, display stuff like their name, new messages,
	// etc.
	if !c.User.IsGuest {
		c.O(`
								<a href="`, scripturl, `?action=unread">`, c.Txt("unread_since_visit"), `</a> <br />
								<a href="`, scripturl, `?action=unreadreplies">`, c.Txt("show_unread_replies"), `</a><br />`)
	} else {
		// Otherwise they're a guest - send them a lovely greeting...
		c.O(c.WelcomeGuest)
	}

	// Now, onto our second set of info, are they logged in again?
	if !c.User.IsGuest {
		// Is the forum in maintenance mode?
		if c.InMaintenance && c.User.IsAdmin {
			c.O(`
								<b>`, c.Txt("616"), `</b><br />`)
		}

		// Are there any members waiting for approval?
		if c.UnapprovedMembers != 0 {
			if c.UnapprovedMembers == 1 {
				c.O(`
								`, c.Txt("approve_thereis"), ` <a href="`, scripturl, `?action=viewmembers;sa=browse;type=approve">`, c.Txt("approve_member"), `</a> `, c.Txt("approve_members_waiting"), `<br />`)
			} else {
				c.O(`
								`, c.Txt("approve_thereare"), ` <a href="`, scripturl, `?action=viewmembers;sa=browse;type=approve">`, c.UnapprovedMembers, ` `, c.Txt("approve_members"), `</a> `, c.Txt("approve_members_waiting"), `<br />`)
			}
		}

		// Show the total time logged in? $context['user']['total_time_logged_in']
		// is always a {days,hours,minutes} array for a logged-in user, so PHP's
		// !empty() is always true here — render unconditionally (even "0 minutes").
		t := c.TimeLoggedIn
		c.O(`
								`, c.Txt("totalTimeLogged1"))
		// If days is just zero, don't bother to show it.
		if t.Days > 0 {
			c.O(itoa(t.Days), c.Txt("totalTimeLogged2"))
		}
		// Same with hours - only show it if it's above zero.
		if t.Hours > 0 {
			c.O(itoa(t.Hours), c.Txt("totalTimeLogged3"))
		}
		// But, let's always show minutes - Time wasted here: 0 minutes ;).
		c.O(itoa(t.Minutes), c.Txt("totalTimeLogged4"), `<br />`)
		c.O(`				</span>`)
	} else {
		// Otherwise they're a guest - this time ask them to either register
		// or login - lazy bums...
		c.O(`				</span>
								<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/sha1.js"></script>

								<form action="`, scripturl, `?action=login2" method="post" accept-charset="`, c.CharacterSet, `" class="middletext" style="margin: 3px 1ex 1px 0;"`)
		if !c.DisableLoginHash {
			c.O(` onsubmit="hashLoginPassword(this, '`, c.Sc, `');"`)
		}
		c.O(`>
									<input type="text" name="user" size="10" /> <input type="password" name="passwrd" size="10" />
									<select name="cookielength">
										<option value="60">`, c.Txt("smf53"), `</option>
										<option value="1440">`, c.Txt("smf47"), `</option>
										<option value="10080">`, c.Txt("smf48"), `</option>
										<option value="43200">`, c.Txt("smf49"), `</option>
										<option value="-1" selected="selected">`, c.Txt("smf50"), `</option>
									</select>
									<input type="submit" value="`, c.Txt("34"), `" /><br />
									<span class="middletext">`, c.Txt("smf52"), `</span>
									<input type="hidden" name="hash_passwrd" value="" />
								</form>`)
	}

	c.O(`
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>`)

	c.O(`
		<table id="upshrinkHeader2"`)
	if collapseHeader {
		c.O(` style="display: none;"`)
	}
	c.O(` width="100%" cellpadding="4" cellspacing="0" border="0">
			<tr>`)

	// Show a random news item? (or you could pick one from news_lines...)
	if !c.Theme.Empty("enable_news") {
		c.O(`
				<td width="90%" class="titlebg2">
					<span class="smalltext"><b>`, c.Txt("102"), `</b>: `, c.RandomNewsLine, `</span>
				</td>`)
	}
	c.O(`
				<td class="titlebg2" align="right" nowrap="nowrap" valign="top">
					<form action="`, scripturl, `?action=search2" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
						<a href="`, scripturl, `?action=search;advanced"><img src="`, c.Theme.ImagesURL(), `/filter.gif" align="middle" style="margin: 0 1ex;" alt="" /></a>
						<input type="text" name="search" value="" style="width: 190px;" />&nbsp;
						<input type="submit" name="submit" value="`, c.Txt("182"), `" style="width: 11ex;" />
						<input type="hidden" name="advanced" value="0" />`)

	// Search within current topic?
	if c.Topic != 0 {
		c.O(`
						<input type="hidden" name="topic" value="`, c.Topic, `" />`)
	} else if c.Board != 0 {
		// If we're on a certain board, limit it to this board ;).
		c.O(`
						<input type="hidden" name="brd[`, c.Board, `]" value="`, c.Board, `" />`)
	}

	c.O(`
					</form>
				</td>
			</tr>
		</table>
	</div>`)

	// Show the menu here, according to the menu sub template.
	templateMenu(c)

	// The main content should go here.
	c.O(`
	<div id="bodyarea" style="padding: 1ex 0px 2ex 0px;">`)
}

func boolToOpt(b bool) string {
	if b {
		return "1"
	}
	return ""
}

// templateMainBelow is template_main_below().
func templateMainBelow(c *Ctx) {
	c.O(`
	</div>`)

	// Show the "Powered by" and "Valid" logos, as well as the copyright.
	// Remember, the copyright must be somewhere!
	c.O(`

	<div id="footerarea" style="text-align: center; padding-bottom: 1ex;`)
	if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
		c.O(` width: 100%;`)
	}
	first, last := "right", "left"
	if c.RightToLeft {
		first, last = "left", "right"
	}
	c.O(`">
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function smfFooterHighlight(element, value)
			{
				element.src = smf_images_url + "/" + (value ? "h_" : "") + element.id + ".gif";
			}
		// ]]></script>
		<table cellspacing="0" cellpadding="3" border="0" align="center" width="100%">
			<tr>
				<td width="28%" valign="middle" align="`, first, `">
					<a href="http://www.mysql.com/" target="_blank"><img id="powered-mysql" src="`, c.Theme.ImagesURL(), `/powered-mysql.gif" alt="`, c.Txt("powered_by_mysql"), `" width="54" height="20" style="margin: 5px 16px;" onmouseover="smfFooterHighlight(this, true);" onmouseout="smfFooterHighlight(this, false);" /></a>
					<a href="http://www.php.net/" target="_blank"><img id="powered-php" src="`, c.Theme.ImagesURL(), `/powered-php.gif" alt="`, c.Txt("powered_by_php"), `" width="54" height="20" style="margin: 5px 16px;" onmouseover="smfFooterHighlight(this, true);" onmouseout="smfFooterHighlight(this, false);" /></a>
				</td>
				<td valign="middle" align="center" style="white-space: nowrap;">
					`)
	c.themeCopyright()
	c.O(`
				</td>
				<td width="28%" valign="middle" align="`, last, `">
					<a href="http://validator.w3.org/check/referer" target="_blank"><img id="valid-xhtml10" src="`, c.Theme.ImagesURL(), `/valid-xhtml10.gif" alt="`, c.Txt("valid_xhtml"), `" width="54" height="20" style="margin: 5px 16px;" onmouseover="smfFooterHighlight(this, true);" onmouseout="smfFooterHighlight(this, false);" /></a>
					<a href="http://jigsaw.w3.org/css-validator/check/referer" target="_blank"><img id="valid-css" src="`, c.Theme.ImagesURL(), `/valid-css.gif" alt="`, c.Txt("valid_css"), `" width="54" height="20" style="margin: 5px 16px;" onmouseover="smfFooterHighlight(this, true);" onmouseout="smfFooterHighlight(this, false);" /></a>
				</td>
			</tr>
		</table>`)

	// Show the load time?
	if c.ShowLoadTime {
		c.O(`
		<span class="smalltext">`, c.Txt("smf301"), c.LoadTime, c.Txt("smf302"), itoa(c.LoadQueries), c.Txt("smf302b"), `</span>`)
	}

	// This is an interesting bug in Internet Explorer AND Safari. Rather
	// annoying, it makes overflows just not tall enough.
	if (c.Browser.IsIE && !c.Browser.IsIE4) || c.Browser.IsMacIE || c.Browser.IsSafari || c.Browser.IsFirefox {
		// The purpose of this code is to fix the height of overflow: auto div
		// blocks, because IE can't figure it out for itself.
		c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[`)

		if c.Browser.IsSafari {
			c.O(`
			window.addEventListener("load", smf_codeFix, false);

			function smf_codeFix()
			{
				var codeFix = document.getElementsByTagName ? document.getElementsByTagName("div") : document.all.tags("div");

				for (var i = 0; i < codeFix.length; i++)
				{
					if ((codeFix[i].className == "code" || codeFix[i].className == "post" || codeFix[i].className == "signature") && codeFix[i].offsetHeight < 20)
						codeFix[i].style.height = (codeFix[i].offsetHeight + 20) + "px";
				}
			}`)
		} else if c.Browser.IsFirefox {
			c.O(`
			window.addEventListener("load", smf_codeFix, false);
			function smf_codeFix()
			{
				var codeFix = document.getElementsByTagName ? document.getElementsByTagName("div") : document.all.tags("div");

				for (var i = 0; i < codeFix.length; i++)
				{
					if (codeFix[i].className == "code" && (codeFix[i].scrollWidth > codeFix[i].clientWidth || codeFix[i].clientWidth == 0))
						codeFix[i].style.overflow = "scroll";
				}
			}`)
		} else {
			c.O(`
			var window_oldOnload = window.onload;
			window.onload = smf_codeFix;

			function smf_codeFix()
			{
				var codeFix = document.getElementsByTagName ? document.getElementsByTagName("div") : document.all.tags("div");

				for (var i = codeFix.length - 1; i > 0; i--)
				{
					if (codeFix[i].currentStyle.overflow == "auto" && (codeFix[i].currentStyle.height == "" || codeFix[i].currentStyle.height == "auto") && (codeFix[i].scrollWidth > codeFix[i].clientWidth || codeFix[i].clientWidth == 0) && (codeFix[i].offsetHeight != 0 || codeFix[i].className == "code"))
						codeFix[i].style.height = (codeFix[i].offsetHeight + 36) + "px";
				}

				if (window_oldOnload)
				{
					window_oldOnload();
					window_oldOnload = null;
				}
			}`)
		}

		c.O(`
		// ]]></script>`)
	}

	c.O(`
	</div>`)

	// The following will be used to let the user know that some AJAX process
	// is running
	c.O(`
	<div id="ajax_in_progress" style="display: none;`)
	if c.Browser.IsIE && !c.Browser.IsIE7 {
		c.O(`position: absolute;`)
	}
	c.O(`">`, c.Txt("ajax_in_progress"), `</div>
</body></html>`)
}

// themeLinktree is theme_linktree().
func (c *Ctx) themeLinktree() {
	c.O(`<div class="nav" style="font-size: smaller; margin-bottom: 2ex; margin-top: 2ex;">`)

	// Each tree item has a URL and name. Some may have extra_before and
	// extra_after.
	for i, tree := range c.LinkTree {
		// Show something before the link?
		if tree.ExtraBefore != "" {
			c.O(tree.ExtraBefore)
		}

		// Show the link, including a URL if it should have one.
		if !c.Theme.Empty("linktree_link") && tree.URL != "" {
			c.O(`<b>`, `<a href="`, tree.URL, `" class="nav">`, tree.Name, `</a>`, `</b>`)
		} else {
			c.O(`<b>`, tree.Name, `</b>`)
		}

		// Show something after the link...?
		if tree.ExtraAfter != "" {
			c.O(tree.ExtraAfter)
		}

		// Don't show a separator for the last one.
		if i != len(c.LinkTree)-1 {
			c.O(`&nbsp;>&nbsp;`)
		}
	}

	c.O(`</div>`)
}

// menuTab emits one [button] of the top menu (the repeated pattern in
// template_menu()).
func (c *Ctx) menuTab(active bool, first, last, href, label string) {
	if active || c.Browser.IsIE4 {
		c.O(`<td class="maintab_active_`, first, `">&nbsp;</td>`)
	}
	back := "back"
	if active {
		back = "active_back"
	}
	c.O(`
				<td valign="top" class="maintab_`, back, `">
					<a href="`, href, `">`, label, `</a>
				</td>`)
	if active {
		c.O(`<td class="maintab_active_`, last, `">&nbsp;</td>`)
	}
}

// templateMenu is template_menu().
func templateMenu(c *Ctx) {
	scripturl := c.App.ScriptURL

	// Work out where we currently are.
	currentAction := "home"
	adminActions := map[string]bool{"admin": true, "ban": true, "boardrecount": true, "cleanperms": true,
		"detailedversion": true, "dumpdb": true, "featuresettings": true, "featuresettings2": true,
		"findmember": true, "maintain": true, "manageattachments": true, "manageboards": true,
		"managecalendar": true, "managesearch": true, "membergroups": true, "modlog": true, "news": true,
		"optimizetables": true, "packageget": true, "packages": true, "permissions": true,
		"pgdownload": true, "postsettings": true, "regcenter": true, "repairboards": true,
		"reports": true, "serversettings": true, "serversettings2": true, "smileys": true,
		"viewErrorLog": true, "viewmembers": true}
	if adminActions[c.CurrentAction] {
		currentAction = "admin"
	}
	switch c.CurrentAction {
	case "search", "admin", "calendar", "profile", "mlist", "register", "login", "help", "pm":
		currentAction = c.CurrentAction
	}
	if c.CurrentAction == "search2" {
		currentAction = "search"
	}
	if c.CurrentAction == "theme" {
		if c.REQUEST.Str("sa") == "pick" {
			currentAction = "profile"
		} else {
			currentAction = "admin"
		}
	}

	// Are we using right-to-left orientation?
	first, last := "first", "last"
	if c.RightToLeft {
		first, last = "last", "first"
	}

	// Show the start of the tab section.
	c.O(`
			<table cellpadding="0" cellspacing="0" border="0" style="margin-left: 10px;">
				<tr>
					<td class="maintab_`, first, `">&nbsp;</td>`)

	// Show the [home] button.
	c.menuTab(currentAction == "home", first, last, scripturl, c.Txt("103"))

	// Show the [help] button.
	c.menuTab(currentAction == "help", first, last, scripturl+"?action=help", c.Txt("119"))

	// How about the [search] button?
	if c.AllowSearch {
		c.menuTab(currentAction == "search", first, last, scripturl+"?action=search", c.Txt("182"))
	}

	// Is the user allowed to administrate at all? ([admin])
	if c.AllowAdmin {
		c.menuTab(currentAction == "admin", first, last, scripturl+"?action=admin", c.Txt("2"))
	}

	// Edit Profile... [profile]
	if c.AllowEditProfile {
		c.menuTab(currentAction == "profile", first, last, scripturl+"?action=profile", c.Txt("79"))
	}

	// Go to PM center... [pm]
	if !c.User.IsGuest && c.AllowPM {
		label := c.Txt("pm_short") + ` `
		if c.User.UnreadMessages > 0 {
			label += `[<strong>` + itoa(c.User.UnreadMessages) + `</strong>]`
		}
		c.menuTab(currentAction == "pm", first, last, scripturl+"?action=pm", label)
	}

	// The [calendar]!
	if c.AllowCalendar {
		c.menuTab(currentAction == "calendar", first, last, scripturl+"?action=calendar", c.Txt("calendar24"))
	}

	// the [member] list button
	if c.AllowMemberlist {
		c.menuTab(currentAction == "mlist", first, last, scripturl+"?action=mlist", c.Txt("331"))
	}

	// If the user is a guest, show [login] button.
	if c.User.IsGuest {
		c.menuTab(currentAction == "login", first, last, scripturl+"?action=login", c.Txt("34"))
		// If the user is a guest, also show [register] button.
		c.menuTab(currentAction == "register", first, last, scripturl+"?action=register", c.Txt("97"))
	} else {
		// Otherwise, they might want to [logout]...
		c.menuTab(currentAction == "logout", first, last, scripturl+"?action=logout;sesc="+c.Sc, c.Txt("108"))
	}

	// The end of tab section.
	c.O(`
				<td class="maintab_`, last, `">&nbsp;</td>
			</tr>
		</table>`)
}
