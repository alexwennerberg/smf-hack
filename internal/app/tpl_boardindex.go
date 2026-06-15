package app

// Hand-port of Themes/default/BoardIndex.template.php (template_main).

import "strings"

// stripButton is one entry for template_button_strip.
type stripButton struct {
	Key    string // button key; matches PHP's $buttons[$key] global cache
	URL    string
	Text   string // $txt key
	Custom string
}

// templateButtonStrip is template_button_strip() from index.template.php.
//
// PHP keeps a request-global $buttons cache keyed by the button key: once a key
// is rendered, the same HTML is reused for every later strip with that key
// unless force_reset is set. This is observable only when one request renders
// the same key with different values (e.g. the help page's mindex vs display
// mocks both using 'notify'). Buttons with an empty Key are never cached.
func (c *Ctx) templateButtonStrip(buttons []stripButton, direction string, forceReset ...bool) {
	reset := len(forceReset) > 0 && forceReset[0]
	if c.buttonCache == nil {
		c.buttonCache = map[string]string{}
	}
	var parts []string
	for _, b := range buttons {
		html := `<a href="` + b.URL + `" ` + b.Custom + `>` + c.Txt(b.Text) + `</a>`
		if b.Key != "" {
			if cached, ok := c.buttonCache[b.Key]; ok && !reset {
				html = cached
			} else {
				c.buttonCache[b.Key] = html
			}
		}
		parts = append(parts, html)
	}
	if len(parts) == 0 {
		c.O(`<td>&nbsp;</td>`)
		return
	}

	prefix := "main"
	if direction != "top" {
		prefix = "mirror"
	}
	first, last := "first", "last"
	if c.RightToLeft {
		first, last = "last", "first"
	}
	c.O(`
		<td class="`, prefix, `tab_`, first, `">&nbsp;</td>
		<td class="`, prefix, `tab_back">`, strings.Join(parts, ` &nbsp;|&nbsp; `), `</td>
		<td class="`, prefix, `tab_`, last, `">&nbsp;</td>`)
}

// templateBoardIndexMain is template_main() from BoardIndex.template.php.
func templateBoardIndexMain(c *Ctx) {
	page := c.Page.(*BoardIndexCtx)
	scripturl := c.App.ScriptURL

	// Show some statistics next to the link tree if SP1 info is off.
	c.O(`
	<table width="100%" cellpadding="0" cellspacing="0">
		<tr>
			<td valign="bottom">`)
	c.themeLinktree()
	c.O(`</td>
			<td align="right">`)
	if c.Theme.Empty("show_sp1_info") {
		c.O(`
				`, c.Txt("19"), `: `, c.CommonStats.TotalMembers, ` &nbsp;&#8226;&nbsp; `, c.Txt("95"), `: `, c.CommonStats.TotalPosts, ` &nbsp;&#8226;&nbsp; `, c.Txt("64"), `: `, c.CommonStats.TotalTopics, `
				`)
		if !c.Theme.Empty("show_latest_member") {
			c.O(`<br />`, c.Txt("201"), ` <b>`, c.CommonStats.LatestMemberLink, `</b>`, c.Txt("581"))
		}
	}
	c.O(`
			</td>
		</tr>
	</table>`)

	// Show the news fader?  (assuming there are things to show...)
	if !c.Theme.Empty("show_newsfader") && len(c.FaderNewsLines) > 0 {
		cellspacing := "0"
		if c.Browser.IsIE || c.Browser.IsOpera6 {
			cellspacing = "1"
		}
		faderTime := c.Theme.Get("newsfader_time")
		if empty(faderTime) {
			faderTime = "5000"
		}
		c.O(`
	<table border="0" width="100%" class="tborder" cellspacing="`, cellspacing, `" cellpadding="4" style="margin-bottom: 2ex;">
		<tr>
			<td class="catbg"> &nbsp;`, c.Txt("102"), `</td>
		</tr>
		<tr>
			<td valign="middle" align="center" height="60">`)

		// Prepare all the javascript settings.
		c.O(`
				<div id="smfFadeScroller" style="width: 90%; padding: 2px;"><b>`, c.NewsLines[0], `</b></div>
				<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
					// The fading delay (in ms.)
					var smfFadeDelay = `, faderTime, `;
					// Fade from... what text color? To which background color?
					var smfFadeFrom = {"r": 0, "g": 0, "b": 0}, smfFadeTo = {"r": 255, "g": 255, "b": 255};
					// Surround each item with... anything special?
					var smfFadeBefore = "<b>", smfFadeAfter = "</b>";

					var foreColor, backEl, backColor;

					if (typeof(document.getElementById('smfFadeScroller').currentStyle) != "undefined")
					{
						foreColor = document.getElementById('smfFadeScroller').currentStyle.color.match(/#([\da-f][\da-f])([\da-f][\da-f])([\da-f][\da-f])/);
						smfFadeFrom = {"r": parseInt(foreColor[1]), "g": parseInt(foreColor[2]), "b": parseInt(foreColor[3])};

						backEl = document.getElementById('smfFadeScroller');
						while (backEl.currentStyle.backgroundColor == "transparent" && typeof(backEl.parentNode) != "undefined")
							backEl = backEl.parentNode;

						backColor = backEl.currentStyle.backgroundColor.match(/#([\da-f][\da-f])([\da-f][\da-f])([\da-f][\da-f])/);
						smfFadeTo = {"r": eval("0x" + backColor[1]), "g": eval("0x" + backColor[2]), "b": eval("0x" + backColor[3])};
					}
					else if (typeof(window.opera) == "undefined" && typeof(document.defaultView) != "undefined")
					{
						foreColor = document.defaultView.getComputedStyle(document.getElementById('smfFadeScroller'), null).color.match(/rgb\((\d+), (\d+), (\d+)\)/);
						smfFadeFrom = {"r": parseInt(foreColor[1]), "g": parseInt(foreColor[2]), "b": parseInt(foreColor[3])};

						backEl = document.getElementById('smfFadeScroller');
						while (document.defaultView.getComputedStyle(backEl, null).backgroundColor == "transparent" && typeof(backEl.parentNode) != "undefined" && typeof(backEl.parentNode.tagName) != "undefined")
							backEl = backEl.parentNode;

						backColor = document.defaultView.getComputedStyle(backEl, null).backgroundColor.match(/rgb\((\d+), (\d+), (\d+)\)/);
						smfFadeTo = {"r": parseInt(backColor[1]), "g": parseInt(backColor[2]), "b": parseInt(backColor[3])};
					}

					// List all the lines of the news for display.
					var smfFadeContent = new Array(
						"`, strings.Join(c.FaderNewsLines, `",
						"`), `"
					);
				// ]]></script>
				<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/fader.js"></script>
			</td>
		</tr>
	</table>`)
	}

	first := true
	for _, category := range page.Categories {
		marginTop := "1ex;"
		if first {
			marginTop = "0;"
		}
		sizeFix := ""
		if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
			sizeFix = "width: 100%;"
		}
		catbg := ""
		if category.New {
			catbg = "2"
		}
		c.O(`
	<div class="tborder" style="margin-top: `, marginTop, ``, sizeFix, `">
		<div class="catbg`, catbg, `" style="padding: 5px 5px 5px 10px;">`)
		first = false

		// If this category even can collapse, show a link to collapse it.
		if category.CanCollapse {
			c.O(`
				<a href="`, category.CollapseHref, `">`, category.CollapseImage, `</a>`)
		}

		c.O(`
				`, category.Link, `
		</div>`)

		// Assuming the category hasn't been collapsed...
		if !category.IsCollapsed {
			c.O(`
		<table border="0" width="100%" cellspacing="1" cellpadding="5" class="bordercolor" style="margin-top: 1px;">`)

			for _, board := range category.Boards {
				rowspan := ""
				if len(board.Children) > 0 {
					rowspan = `rowspan="2"`
				}
				c.O(`
			<tr>
				<td `, rowspan, ` class="windowbg" width="6%" align="center" valign="top"><a href="`, scripturl, `?action=unread;board=`, board.ID, `.0">`)

				// If the board is new, show a strong indicator.
				if board.New {
					c.O(`<img src="`, c.Theme.ImagesURL(), `/on.gif" alt="`, c.Txt("333"), `" title="`, c.Txt("333"), `" />`)
				} else if board.ChildrenNew {
					c.O(`<img src="`, c.Theme.ImagesURL(), `/on2.gif" alt="`, c.Txt("333"), `" title="`, c.Txt("333"), `" />`)
				} else {
					c.O(`<img src="`, c.Theme.ImagesURL(), `/off.gif" alt="`, c.Txt("334"), `" title="`, c.Txt("334"), `" />`)
				}

				c.O(`</a>
				</td>
				<td class="windowbg2">
					<b><a href="`, board.Href, `" name="b`, board.ID, `">`, board.Name, `</a></b><br />
						`, board.Description)

				// Show the "Moderators: ".
				if len(board.Moderators) > 0 {
					modTxt := c.Txt("299")
					if len(board.Moderators) == 1 {
						modTxt = c.Txt("298")
					}
					c.O(`
					<div style="padding-top: 1px;" class="smalltext"><i>`, modTxt, `: `, strings.Join(board.LinkModerators, ", "), `</i></div>`)
				}

				// Show some basic information about the number of posts, etc.
				c.O(`
				</td>
				<td class="windowbg" valign="middle" align="center" style="width: 12ex;"><span class="smalltext">
					`, board.Posts, ` `, c.Txt("21"), ` <br />
					`, board.Topics, ` `, c.Txt("330"), `
				</span></td>
				<td class="windowbg2" valign="middle" width="22%">
					<span class="smalltext">`)

				if board.LastPost != nil && board.LastPost.ID != 0 {
					c.O(`
						<b>`, c.Txt("22"), `</b>  `, c.Txt("525"), ` `, board.LastPost.MemberLink, `<br />
						`, c.Txt("smf88"), ` `, board.LastPost.Link, `<br />
						`, c.Txt("30"), ` `, board.LastPost.Time)
				}
				c.O(`
					</span>
				</td>
			</tr>`)

				// Show the "Child Boards: ".
				if len(board.Children) > 0 {
					var children []string
					for _, child := range board.Children {
						title := c.Txt("334")
						if child.New {
							title = c.Txt("333")
						}
						link := `<a href="` + child.Href + `" title="` + title + ` (` + c.Txt("330") + `: ` + itoa(child.Topics) + `, ` + c.Txt("21") + `: ` + itoa(child.Posts) + `)">` + child.Name + `</a>`
						if child.New {
							link = "<b>" + link + "</b>"
						}
						children = append(children, link)
					}

					wbg := ""
					if !c.Theme.Empty("seperate_sticky_lock") {
						wbg = "3"
					}
					c.O(`
			<tr>
				<td colspan="3" class="windowbg`, wbg, `">
					<span class="smalltext"><b>`, c.Txt("parent_boards"), `</b>: `, strings.Join(children, ", "), `</span>
				</td>
			</tr>`)
				}
			}
			c.O(`
		</table>`)
		}
		c.O(`
	</div>`)
	}

	if !c.User.IsGuest {
		al, ar := "left", "right"
		if c.RightToLeft {
			al, ar = "right", "left"
		}
		c.O(`
	<table border="0" width="100%" cellspacing="0" cellpadding="5">
		<tr>
			<td align="`, al, `" class="smalltext">
				<img src="`, c.Theme.ImagesURL(), `/new_some.gif" alt="" align="middle" /> `, c.Txt("333"), `
				<img src="`, c.Theme.ImagesURL(), `/new_none.gif" alt="" align="middle" style="margin-left: 4ex;" /> `, c.Txt("334"), `
			</td>
			<td align="`, ar, `">`)

		// Show the mark all as read button?
		if !c.Theme.Empty("show_mark_read") && len(page.Categories) > 0 {
			c.O(`
				<table cellpadding="0" cellspacing="0" border="0" style="position: relative; top: -5px;">
					<tr>
							 `)
			c.templateButtonStrip([]stripButton{
				{URL: scripturl + "?action=markasread;sa=all;sesc=" + c.Sc, Text: "452"},
			}, "top")
			c.O(`
					</tr>
				</table>`)
		}
		c.O(`
			</td>
		</tr>
	</table>`)
	}

	// Here's where the "Info Center" starts...
	sizeFix := ""
	if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
		sizeFix = `style="width: 100%;"`
	}
	collapseIC := !empty(c.Options["collapse_header_ic"])
	icGif := "collapse.gif"
	if collapseIC {
		icGif = "expand.gif"
	}
	c.O(`<br />
	<div class="tborder" `, sizeFix, `>
		<div class="catbg" style="padding: 6px; vertical-align: middle; text-align: center; ">
			<a href="#" onclick="shrinkHeaderIC(!current_header_ic); return false;"><img id="upshrink_ic" src="`, c.Theme.ImagesURL(), `/`, icGif, `" alt="*" title="`, c.Txt("upshrink_description"), `" style="margin-right: 2ex;" align="right" /></a>
			`, c.Txt("685"), `
		</div>
		<div id="upshrinkHeaderIC"`)
	if collapseIC {
		c.O(` style="display: none;"`)
	}
	c.O(`>
			<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">`)

	// This is the "Recent Posts" bar.
	if !c.Theme.Empty("number_recent_posts") {
		c.O(`
				<tr>
					<td class="titlebg" colspan="2">`, c.Txt("214"), `</td>
				</tr>
				<tr>
					<td class="windowbg" width="20" valign="middle" align="center">
						<a href="`, scripturl, `?action=recent"><img src="`, c.Theme.ImagesURL(), `/post/xx.gif" alt="`, c.Txt("214"), `" /></a>
					</td>
					<td class="windowbg2">`)

		// Only show one post.
		if c.Theme.Int("number_recent_posts") == 1 && page.LatestPost != nil {
			c.O(`
						<b><a href="`, scripturl, `?action=recent">`, c.Txt("214"), `</a></b>
						<div class="smalltext">
								`, c.Txt("234"), ` &quot;`, page.LatestPost.Link, `&quot; `, c.Txt("235"), ` (`, page.LatestPost.Time, `)<br />
						</div>`)
		} else if len(page.LatestPosts) > 0 {
			// Several recent posts.
			c.O(`
						<table cellpadding="0" cellspacing="0" width="100%" border="0">`)
			for _, post := range page.LatestPosts {
				c.O(`
							<tr>
								<td class="middletext" valign="top"><b>`, post.Link, `</b> `, c.Txt("525"), ` `, post.PosterLink, ` (`, post.BoardLink, `)</td>
								<td class="middletext" align="right" valign="top" nowrap="nowrap">`, post.Time, `</td>
							</tr>`)
			}
			c.O(`
						</table>`)
		}
		c.O(`
					</td>
				</tr>`)
	}

	// Show information about events, birthdays, and holidays on the calendar.
	if page.ShowCalendar {
		calHeader := c.Txt("calendar47")
		if page.CalendarOnlyToday {
			calHeader = c.Txt("calendar47b")
		}
		c.O(`
				<tr>
					<td class="titlebg" colspan="2">`, calHeader, `</td>
				</tr><tr>
					<td class="windowbg" width="20" valign="middle" align="center">
						<a href="`, scripturl, `?action=calendar"><img src="`, c.Theme.ImagesURL(), `/icons/calendar.gif" alt="`, c.Txt("calendar24"), `" /></a>
					</td>
					<td class="windowbg2" width="100%">
						<span class="smalltext">`)

		// Holidays like "Christmas", "Chanukah", and "We Love [Unknown] Day" :P.
		if len(page.CalendarHolidays) > 0 {
			c.O(`
							<span style="color: #`, c.App.Setting("cal_holidaycolor"), `;">`, c.Txt("calendar5"), ` `, joinStr(page.CalendarHolidays, ", "), `</span><br />`)
		}

		// People's birthdays. Like mine. And yours, I guess. Kidding.
		if len(page.CalendarBirthdays) > 0 {
			bdayLabel := c.Txt("calendar3b")
			if page.CalendarOnlyToday {
				bdayLabel = c.Txt("calendar3")
			}
			c.O(`
							<span style="color: #`, c.App.Setting("cal_bdaycolor"), `;">`, bdayLabel, `</span> `)
			for _, m := range page.CalendarBirthdays {
				openB, closeB := "", ""
				if m.IsToday {
					openB, closeB = "<b>", "</b>"
				}
				age := ""
				if m.HasAge {
					age = " (" + itoa(m.Age) + ")"
				}
				sep := ", "
				if m.IsLast {
					sep = "<br />"
				}
				c.O(`
							<a href="`, scripturl, `?action=profile;u=`, itoa(m.ID), `">`, openB, m.Name, closeB, age, `</a>`, sep)
			}
		}

		// Events like community get-togethers.
		if len(page.CalendarEvents) > 0 {
			evLabel := c.Txt("calendar4b")
			if page.CalendarOnlyToday {
				evLabel = c.Txt("calendar4")
			}
			c.O(`
							<span style="color: #`, c.App.Setting("cal_eventcolor"), `;">`, evLabel, `</span> `)
			for _, ev := range page.CalendarEvents {
				edit := ""
				if ev.CanEdit {
					edit = `<a href="` + ev.ModifyHref + `" style="color: #FF0000;">*</a> `
				}
				openA, closeA := "", ""
				if ev.Href != "" {
					openA = `<a href="` + ev.Href + `">`
					closeA = `</a>`
				}
				title := ev.Title
				if ev.IsToday {
					title = "<b>" + ev.Title + "</b>"
				}
				sep := ", "
				if ev.IsLast {
					sep = "<br />"
				}
				c.O(`
							`, edit, openA, title, closeA, sep)
			}

			// Show a little help text to help them along ;).
			if page.CalendarCanEdit {
				c.O(`
							(<a href="`, scripturl, `?action=helpadmin;help=calendar_how_edit" onclick="return reqWin(this.href);">`, c.Txt("calendar_how_edit"), `</a>)`)
			}
		}
		c.O(`
						</span>
					</td>
				</tr>`)
	}

	// Show YaBB SP1 style information...
	if !c.Theme.Empty("show_sp1_info") {
		latestPostLink, latestPostTime := "", ""
		if page.LatestPost != nil {
			latestPostLink, latestPostTime = page.LatestPost.Link, page.LatestPost.Time
		}
		c.O(`
				<tr>
					<td class="titlebg" colspan="2">`, c.Txt("645"), `</td>
				</tr>
				<tr>
					<td class="windowbg" width="20" valign="middle" align="center">
						<a href="`, scripturl, `?action=stats"><img src="`, c.Theme.ImagesURL(), `/icons/info.gif" alt="`, c.Txt("645"), `" /></a>
					</td>
					<td class="windowbg2" width="100%">
						<span class="middletext">
							`, c.CommonStats.TotalPosts, ` `, c.Txt("95"), ` `, c.Txt("smf88"), ` `, c.CommonStats.TotalTopics, ` `, c.Txt("64"), ` `, c.Txt("525"), ` `, c.CommonStats.TotalMembers, ` `, c.Txt("19"), `. `, c.Txt("656"), `: <b> `, c.CommonStats.LatestMemberLink, `</b>
							<br /> `, c.Txt("659"), `: <b>&quot;`, latestPostLink, `&quot;</b>  ( `, latestPostTime, ` )<br />
							<a href="`, scripturl, `?action=recent">`, c.Txt("234"), `</a>`)
		if page.ShowStats {
			c.O(`<br />
							<a href="`, scripturl, `?action=stats">`, c.Txt("smf223"), `</a>`)
		}
		c.O(`
						</span>
					</td>
				</tr>`)
	}

	// "Users online" - in order of activity.
	whoOpen, whoClose := "", ""
	if page.ShowWho {
		whoOpen = `<a href="` + scripturl + `?action=who">`
		whoClose = `</a>`
	}
	c.O(`
				<tr>
					<td class="titlebg" colspan="2">`, c.Txt("158"), `</td>
				</tr><tr>
					<td rowspan="2" class="windowbg" width="20" valign="middle" align="center">
						`, whoOpen, `<img src="`, c.Theme.ImagesURL(), `/icons/online.gif" alt="`, c.Txt("158"), `" />`, whoClose, `
					</td>
					<td class="windowbg2" width="100%">`)

	guestsTxt := c.Txt("guests")
	if page.NumGuests == 1 {
		guestsTxt = c.Txt("guest")
	}
	usersTxt := c.Txt("users")
	if page.NumUsersOnline == 1 {
		usersTxt = c.Txt("user")
	}
	c.O(`
						`, whoOpen, itoa(page.NumGuests), ` `, guestsTxt, `, `, itoa(page.NumUsersOnline), ` `, usersTxt)

	// Handle hidden users and buddies.
	if page.NumUsersHidden > 0 || page.ShowBuddies {
		c.O(` (`)

		// Show the number of buddies online?
		if page.ShowBuddies {
			buddiesTxt := c.Txt("buddies")
			if page.NumBuddies == 1 {
				buddiesTxt = c.Txt("buddy")
			}
			c.O(itoa(page.NumBuddies), ` `, buddiesTxt)
		}

		// How about hidden users?
		if page.NumUsersHidden > 0 {
			if page.ShowBuddies {
				c.O(`, `)
			}
			c.O(itoa(page.NumUsersHidden), ` `, c.Txt("hidden"))
		}

		c.O(`)`)
	}

	c.O(whoClose, `
						<div class="smalltext">`)

	// Assuming there ARE users online...
	if len(page.UsersOnline) > 0 {
		c.O(`
							`, c.Txt("140"), `:<br />`, strings.Join(page.ListUsersOnline, ", "))
	}

	c.O(`
							<br />
							`)
	if page.ShowStats && c.Theme.Empty("show_sp1_info") {
		c.O(`<a href="`, scripturl, `?action=stats">`, c.Txt("smf223"), `</a>`)
	}
	c.O(`
						</div>
					</td>
				</tr>
				<tr>
					<td class="windowbg2" width="100%">
						<span class="middletext">
							`, c.Txt("most_online_today"), `: <b>`, c.App.Setting("mostOnlineToday"), `</b>.
							`, c.Txt("most_online_ever"), `: `, c.App.Setting("mostOnline"), ` (`, c.timeformat(int64(c.App.SettingInt("mostDate"))), `)
						</span>
					</td>
				</tr>`)

	// If they are logged in, but SP1 style information is off... show a
	// personal message bar.
	if !c.User.IsGuest && c.Theme.Empty("show_sp1_info") {
		pmOpen, pmClose := "", ""
		if c.AllowPM {
			pmOpen = `<a href="` + scripturl + `?action=pm">`
			pmClose = `</a>`
		}
		msgTxt := c.Txt("153")
		if c.User.Messages == 1 {
			msgTxt = c.Txt("471")
		}
		c.O(`
				<tr>
					<td class="titlebg" colspan="2">`, c.Txt("159"), `</td>
				</tr><tr>
					<td class="windowbg" width="20" valign="middle" align="center">
						`, pmOpen, `<img src="`, c.Theme.ImagesURL(), `/message_sm.gif" alt="`, c.Txt("159"), `" />`, pmClose, `
					</td>
					<td class="windowbg2" valign="top">
						<b><a href="`, scripturl, `?action=pm">`, c.Txt("159"), `</a></b>
						<div class="smalltext">
							`, c.Txt("660"), ` `, c.User.Messages, ` `, msgTxt, `.... `, c.Txt("661"), ` <a href="`, scripturl, `?action=pm">`, c.Txt("662"), `</a> `, c.Txt("663"), `
						</div>
					</td>
				</tr>`)
	}

	// Show the login bar. (it's only true if they are logged out anyway.)
	if page.ShowLoginBar {
		c.O(`
				<tr>
					<td class="titlebg" colspan="2">`, c.Txt("34"), ` <a href="`, scripturl, `?action=reminder" class="smalltext">(`, c.Txt("315"), `)</a></td>
				</tr>
				<tr>
					<td class="windowbg" width="20" align="center">
						<a href="`, scripturl, `?action=login"><img src="`, c.Theme.ImagesURL(), `/icons/login.gif" alt="`, c.Txt("34"), `" /></a>
					</td>
					<td class="windowbg2" valign="middle">
						<form action="`, scripturl, `?action=login2" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
							<table border="0" cellpadding="2" cellspacing="0" align="center" width="100%"><tr>
								<td valign="middle" align="left">
									<label for="user"><b>`, c.Txt("35"), `:</b><br />
									<input type="text" name="user" id="user" size="15" /></label>
								</td>
								<td valign="middle" align="left">
									<label for="passwrd"><b>`, c.Txt("36"), `:</b><br />
									<input type="password" name="passwrd" id="passwrd" size="15" /></label>
								</td>
								<td valign="middle" align="left">
									<label for="cookielength"><b>`, c.Txt("497"), `:</b><br />
									<input type="text" name="cookielength" id="cookielength" size="4" maxlength="4" value="`, c.App.Setting("cookieTime"), `" /></label>
								</td>
								<td valign="middle" align="left">
									<label for="cookieneverexp"><b>`, c.Txt("508"), `:</b><br />
									<input type="checkbox" name="cookieneverexp" id="cookieneverexp" checked="checked" class="check" /></label>
								</td>
								<td valign="middle" align="left">
									<input type="submit" value="`, c.Txt("34"), `" />
								</td>
							</tr></table>
						</form>
					</td>
				</tr>`)
	}

	c.O(`
			</table>
		</div>
	</div>`)
}
