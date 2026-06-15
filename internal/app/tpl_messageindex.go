package app

// Hand-port of Themes/default/MessageIndex.template.php (template_main).
// Quick-moderation blocks (display_quick_mod theme option) port with
// Phase 6; the option is off by default.

import "strings"

// templateMessageIndexMain is template_main() from MessageIndex.template.php.
func templateMessageIndexMain(c *Ctx) {
	page := c.Page.(*MessageIndexCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<div style="margin-bottom: 2px;"><a name="top"></a>`)
	c.themeLinktree()
	c.O(`</div>`)

	if len(page.Boards) > 0 && (!empty(c.Options["show_children"]) || page.Start == 0) {
		sizeFix := ""
		if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
			sizeFix = " width: 100%;"
		}
		c.O(`
	<div class="tborder" style="margin-bottom: 3ex; `, sizeFix, `">
		<table border="0" width="100%" cellspacing="1" cellpadding="5" class="bordercolor">
			<tr>
				<td colspan="4" class="catbg">`, c.Txt("parent_boards"), `</td>
			</tr>`)

		for _, board := range page.Boards {
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

			if len(board.Moderators) > 0 {
				modTxt := c.Txt("299")
				if len(board.Moderators) == 1 {
					modTxt = c.Txt("298")
				}
				c.O(`
					<div style="padding-top: 1px;"><small><i>`, modTxt, `: `, strings.Join(board.LinkModerators, ", "), `</i></small></div>`)
			}

			// Show some basic information about the number of posts, etc.
			c.O(`
				</td>
				<td class="windowbg" valign="middle" align="center" style="width: 12ex;"><small>
					`, board.Posts, ` `, c.Txt("21"), ` <br />
					`, board.Topics, ` `, c.Txt("330"), `</small>
				</td>
				<td class="windowbg2" valign="middle" width="22%"><small>`)

			if board.LastPost != nil && board.LastPost.ID != 0 {
				c.O(`
					<b>`, c.Txt("22"), `</b> `, c.Txt("525"), ` `, board.LastPost.MemberLink, `<br />
					`, c.Txt("smf88"), ` `, board.LastPost.Link, `<br />
					`, c.Txt("30"), ` `, board.LastPost.Time)
			}

			c.O(`</small>
				</td>
			</tr>`)

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
					<small><b>`, c.Txt("parent_boards"), `</b>: `, strings.Join(children, ", "), `</small>
				</td>
			</tr>`)
			}
		}

		c.O(`
		</table>
	</div>`)
	}

	if !empty(c.Options["show_board_desc"]) && page.Description != "" {
		c.O(`
		<table width="100%" cellpadding="6" cellspacing="1" border="0" class="tborder" style="padding: 0; margin-bottom: 2ex;">
			<tr>
				<td class="titlebg2" width="100%" height="24" style="border-top: 0;">
					<small>`, page.Description, `</small>
				</td>
			</tr>
		</table>`)
	}

	// Create the button set...
	normalButtons := func() []stripButton {
		var buttons []stripButton
		// They can only mark read if they are logged in and it's enabled!
		if !c.User.IsGuest && !c.Theme.Empty("show_mark_read") {
			buttons = append(buttons, stripButton{
				URL:  scripturl + "?action=markasread;sa=board;board=" + itoa(c.Board) + ".0;sesc=" + c.Sc,
				Text: "mark_read_short",
			})
		}
		if page.CanMarkNotify {
			confirmTxt := c.Txt("notification_enable_board")
			sa := "on"
			if page.IsMarkedNotify {
				confirmTxt = c.Txt("notification_disable_board")
				sa = "off"
			}
			buttons = append(buttons, stripButton{
				URL:    scripturl + "?action=notifyboard;sa=" + sa + ";board=" + itoa(c.Board) + "." + itoa(page.Start) + ";sesc=" + c.Sc,
				Text:   "125",
				Custom: `onclick="return confirm('` + confirmTxt + `');"`,
			})
		}
		if page.CanPostNew {
			buttons = append(buttons, stripButton{
				URL:  scripturl + "?action=post;board=" + itoa(c.Board) + ".0",
				Text: "smf258",
			})
		}
		if page.CanPostPoll {
			buttons = append(buttons, stripButton{
				URL:  scripturl + "?action=post;board=" + itoa(c.Board) + ".0;poll",
				Text: "smf20",
			})
		}
		return buttons
	}()

	if !page.NoTopicListing {
		topBottom := ""
		if !c.App.SettingEmpty("topbottomEnable") {
			topBottom = c.MenuSeparator + `&nbsp;&nbsp;<a href="#bot"><b>` + c.Txt("topbottom5") + `</b></a>`
		}
		c.O(`
		<table width="100%" cellpadding="0" cellspacing="0" border="0">
			<tr>
				<td class="middletext">`, c.Txt("139"), `: `, page.PageIndex, topBottom, `</td>
				<td align="right" style="padding-right: 1ex;">
					<table cellpadding="0" cellspacing="0">
						<tr>
							`)
		c.templateButtonStrip(normalButtons, "bottom")
		c.O(`
						</tr>
					</table>
				</td>
			</tr>
		</table>`)

		sizeFix := ""
		if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
			sizeFix = `style="width: 100%;"`
		}
		c.O(`
			<div class="tborder" `, sizeFix, `>
				<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
					<tr>`)

		// Are there actually any topics to show?
		if len(page.Topics) > 0 {
			sortLink := func(key, label, width, align string) {
				desc := ""
				if page.SortBy == key && page.SortDirection == "up" {
					desc = ";desc"
				}
				sortImg := ""
				if page.SortBy == key {
					sortImg = ` <img src="` + c.Theme.ImagesURL() + `/sort_` + page.SortDirection + `.gif" alt="" />`
				}
				attrs := ""
				if width != "" {
					attrs += ` width="` + width + `"`
				}
				if align != "" {
					attrs += ` align="` + align + `"`
				}
				c.O(`

						<td class="catbg3"`, attrs, `><a href="`, scripturl, `?board=`, c.Board, `.`, page.Start, `;sort=`, key, desc, `">`, label, sortImg, `</a></td>`)
			}
			c.O(`
						<td width="9%" colspan="2" class="catbg3"></td>`)
			sortLink("subject", c.Txt("70"), "", "")
			sortLink("starter", c.Txt("109"), "11%", "")
			sortLink("replies", c.Txt("110"), "4%", "center")
			sortLink("views", c.Txt("301"), "4%", "center")
			sortLink("last_post", c.Txt("111"), "22%", "")
		} else {
			// No topics.... just say, "sorry bub".
			c.O(`
						<td class="catbg3" width="100%" colspan="7"><b>`, c.Txt("151"), `</b></td>`)
		}

		c.O(`
					</tr>`)

		for _, topic := range page.Topics {
			// Do we want to seperate the sticky and lock status out?
			class := topic.Class
			if !c.Theme.Empty("seperate_sticky_lock") {
				if i := strings.LastIndex(class, "_sticky"); i >= 0 && strings.Contains(class, "sticky") {
					class = class[:i]
				}
				if i := strings.LastIndex(class, "_locked"); i >= 0 && strings.Contains(class, "locked") {
					class = class[:i]
				}
			}

			c.O(`
					<tr>
						<td class="windowbg2" valign="middle" align="center" width="5%">
							<img src="`, c.Theme.ImagesURL(), `/topic/`, class, `.gif" alt="" />
						</td>
						<td class="windowbg2" valign="middle" align="center" width="4%">
							<img src="`, topic.FirstPost.IconURL, `" alt="" />
						</td>
						<td class="windowbg`)
			if !c.Theme.Empty("seperate_sticky_lock") && topic.IsSticky {
				c.O(`3`)
			}
			c.O(`" valign="middle" `, ``, `>`)

			if !c.Theme.Empty("seperate_sticky_lock") {
				c.O(`
							`)
				if topic.IsLocked {
					c.O(`<img src="`, c.Theme.ImagesURL(), `/icons/quick_lock.gif" align="right" alt="" id="lockicon`, topic.FirstPost.ID, `" style="margin: 0;" />`)
				}
				c.O(`
							`)
				if topic.IsSticky {
					c.O(`<img src="`, c.Theme.ImagesURL(), `/icons/show_sticky.gif" align="right" alt="" id="stickyicon`, topic.FirstPost.ID, `" style="margin: 0;" />`)
				}
			}

			c.O(`
							`)
			if topic.IsSticky {
				c.O(`<b>`)
			}
			c.O(`<span id="msg_`, topic.FirstPost.ID, `">`, topic.FirstPost.Link, `</span>`)
			if topic.IsSticky {
				c.O(`</b>`)
			}

			// Is this topic new? (assuming they are logged in!)
			if topic.New && !c.User.IsGuest {
				c.O(`
							<a href="`, topic.NewHref, `" id="newicon`, topic.FirstPost.ID, `"><img src="`, c.Theme.ImagesURL(), `/`, c.User.Language, `/new.gif" alt="`, c.Txt("302"), `" /></a>`)
			}

			c.O(`
							<small id="pages`, topic.FirstPost.ID, `">`, topic.Pages, `</small>
						</td>
						<td class="windowbg2" valign="middle" width="14%">
							`, topic.FirstPost.MemberLink, `
						</td>
						<td class="windowbg`)
			if topic.IsSticky {
				c.O(`3`)
			}
			c.O(`" valign="middle" width="4%" align="center">
							`, topic.Replies, `
						</td>
						<td class="windowbg`)
			if topic.IsSticky {
				c.O(`3`)
			}
			c.O(`" valign="middle" width="4%" align="center">
							`, topic.Views, `
						</td>
						<td class="windowbg2" valign="middle" width="22%">
							<a href="`, topic.LastPost.Href, `"><img src="`, c.Theme.ImagesURL(), `/icons/last_post.gif" alt="`, c.Txt("111"), `" title="`, c.Txt("111"), `" style="float: right;" /></a>
							<span class="smalltext">
								`, topic.LastPost.Time, `<br />
								`, c.Txt("525"), ` `, topic.LastPost.MemberLink, `
							</span>
						</td>`)
			c.O(`
					</tr>`)
		}

		c.O(`
				</table>
			</div>
			<a name="bot"></a>`)

		topBottomTop := ""
		if !c.App.SettingEmpty("topbottomEnable") {
			topBottomTop = c.MenuSeparator + `&nbsp;&nbsp;<a href="#top"><b>` + c.Txt("topbottom4") + `</b></a>`
		}
		c.O(`
	<table width="100%" cellpadding="0" cellspacing="0" border="0">
		<tr>
			<td class="middletext">`, c.Txt("139"), `: `, page.PageIndex, topBottomTop, `</td>
			<td align="right" style="padding-right: 1ex;">
				<table cellpadding="0" cellspacing="0">
					<tr>
						`)
		c.templateButtonStrip(normalButtons, "top")
		c.O(`
					</tr>
				</table>
			</td>
		</tr>
	</table>`)
	}

	// Show breadcrumbs at the bottom too?
	c.O(`
	<div>`)
	c.themeLinktree()
	c.O(`<br /></div>`)

	c.O(`
	<div class="tborder">
		<table cellpadding="8" cellspacing="0" width="100%" class="titlebg2">
			<tr>`)

	if !page.NoTopicListing {
		participation := ""
		if !c.App.SettingEmpty("enableParticipation") {
			participation = `
					<img src="` + c.Theme.ImagesURL() + `/topic/my_normal_post.gif" alt="" align="middle" /> ` + c.Txt("participation_caption") + `<br />`
		}
		sticky := ""
		if c.App.Setting("enableStickyTopics") == "1" {
			sticky = `
					<img src="` + c.Theme.ImagesURL() + `/icons/quick_sticky.gif" alt="" align="middle" /> ` + c.Txt("smf96") + `<br />`
		}
		poll := ""
		if c.App.Setting("pollMode") == "1" {
			poll = `
					<img src="` + c.Theme.ImagesURL() + `/topic/normal_poll.gif" alt="" align="middle" /> ` + c.Txt("smf43")
		}
		c.O(`
				<td style="padding-top: 2ex;" class="smalltext">`, participation, `
					<img src="`, c.Theme.ImagesURL(), `/topic/normal_post.gif" alt="" align="middle" /> `, c.Txt("457"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/hot_post.gif" alt="" align="middle" /> `, c.Txt("454"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/veryhot_post.gif" alt="" align="middle" /> `, c.Txt("455"), `
				</td>
				<td valign="top" style="padding-top: 2ex;" class="smalltext">
					<img src="`, c.Theme.ImagesURL(), `/icons/quick_lock.gif" alt="" align="middle" /> `, c.Txt("456"), `<br />`, sticky, poll, `
				</td>`)
	}

	align := "right"
	if c.RightToLeft {
		align = "left"
	}
	c.O(`
				<td align="`, align, `" valign="middle">
					<form action="`, scripturl, `" method="get" accept-charset="`, c.CharacterSet, `" name="jumptoForm">
						<span class="smalltext"><label for="jumpto">`, c.Txt("160"), `</label>:</span>
					<select name="jumpto" id="jumpto" onchange="if (this.selectedIndex > 0 &amp;&amp; this.options[this.selectedIndex].value) window.location.href = smf_scripturl + this.options[this.selectedIndex].value.substr(smf_scripturl.indexOf('?') == -1 || this.options[this.selectedIndex].value.substr(0, 1) != '?' ? 0 : 1);">
								<option value="">`, c.Txt("251"), `:</option>`)

	// Show each category - they all have an id, name, and the boards in them.
	for _, category := range c.JumpTo {
		c.O(`
								<option value="" disabled="disabled">-----------------------------</option>
								<option value="#`, category.ID, `">`, category.Name, `</option>
								<option value="" disabled="disabled">-----------------------------</option>`)

		for _, board := range category.Boards {
			selected := ""
			if board.IsCurrent {
				selected = ` selected="selected"`
			}
			c.O(`
								<option value="?board=`, board.ID, `.0"`, selected, `> `, strings.Repeat("==", board.ChildLevel), `=> `, board.Name, `</option>`)
		}
	}

	c.O(`
						</select>&nbsp;
					<input type="button" value="`, c.Txt("161"), `" onclick="if (this.form.jumpto.options[this.form.jumpto.selectedIndex].value) window.location.href = '`, scripturl, `' + this.form.jumpto.options[this.form.jumpto.selectedIndex].value;" />
					</form>
				</td>
			</tr>
		</table>
	</div>`)

	// Javascript for inline editing.
	c.O(`
<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/xml_board.js"></script>
<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[

	// Hide certain bits during topic edit.
	hide_prefixes.push("lockicon", "stickyicon", "pages", "newicon");

	// Use it to detect when we've stopped editing.
	document.onclick = modify_topic_click;

	var mouse_on_div;
	function modify_topic_click()
	{
		if (in_edit_mode == 1 && mouse_on_div == 0)
			modify_topic_save("`, c.Sc, `");
	}

	function modify_topic_keypress(oEvent)
	{
		if (typeof(oEvent.keyCode) != "undefined" && oEvent.keyCode == 13)
		{
			modify_topic_save("`, c.Sc, `");
			if (typeof(oEvent.preventDefault) == "undefined")
				oEvent.returnValue = false;
			else
				oEvent.preventDefault();
		}
	}

	// For templating, shown when an inline edit is made.
	function modify_topic_show_edit(subject)
	{
		// Just template the subject.
		setInnerHTML(cur_subject_div, '<input type="text" name="subject" value="' + subject + '" size="60" style="width: 99%;"  maxlength="80" onkeypress="modify_topic_keypress(event)" /><input type="hidden" name="topic" value="' + cur_topic_id + '" /><input type="hidden" name="msg" value="' + cur_msg_id.substr(4) + '" />');
	}

	// And the reverse for hiding it.
	function modify_topic_hide_edit(subject)
	{
		// Re-template the subject!
		setInnerHTML(cur_subject_div, '<a href="`, scripturl, `?topic=' + cur_topic_id + '.0">' + subject + '</a>');
	}

// ]]></script>`)
}
