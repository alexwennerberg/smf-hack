package app

// Hand-port of Themes/default/Recent.template.php: template_main (recent
// posts), template_unread and template_replies (the unread listings).

import "strings"

// unreadClass strips the _sticky/_locked suffixes when seperate_sticky_lock is
// on (so the row shows separate lock/sticky icons instead).
func (c *Ctx) unreadClass(class string) string {
	if c.Theme.Empty("seperate_sticky_lock") {
		return class
	}
	if strings.Contains(class, "sticky") {
		if i := strings.LastIndex(class, "_sticky"); i >= 0 {
			class = class[:i]
		}
	}
	if strings.Contains(class, "locked") {
		if i := strings.LastIndex(class, "_locked"); i >= 0 {
			class = class[:i]
		}
	}
	return class
}

// unreadTopicRow renders one topic row, shared by unread/replies.
func (c *Ctx) unreadTopicRow(topic *MITopic) {
	images := c.Theme.ImagesURL()
	class := c.unreadClass(topic.Class)

	bg3 := ""
	if topic.IsSticky && !c.Theme.Empty("seperate_sticky_lock") {
		bg3 = "3"
	}
	lockImg := ""
	if topic.IsLocked && !c.Theme.Empty("seperate_sticky_lock") {
		lockImg = `<img src="` + images + `/icons/quick_lock.gif" align="right" alt="" style="margin: 0;" />`
	}
	stickyImg := ""
	if topic.IsSticky && !c.Theme.Empty("seperate_sticky_lock") {
		stickyImg = `<img src="` + images + `/icons/show_sticky.gif" align="right" alt="" style="margin: 0;" />`
	}

	c.O(`
				<tr>
					<td class="windowbg2" valign="middle" align="center" width="6%">
						<img src="`, images, `/topic/`, class, `.gif" alt="" />
					</td><td class="windowbg2" valign="middle" align="center" width="4%">
						<img src="`, topic.FirstPost.IconURL, `" alt="" align="middle" />
					</td><td class="windowbg`, bg3, `" width="48%" valign="middle">`, lockImg, stickyImg, topic.FirstPost.Link, ` <a href="`, topic.NewHref, `"><img src="`, images, `/`, c.User.Language, `/new.gif" alt="`, c.Txt("302"), `" /></a> <span class="smalltext">`, topic.Pages, ` `, c.Txt("smf88"), ` `, topic.Board.Link, `</span></td>
					<td class="windowbg2" valign="middle" width="14%">
						`, topic.FirstPost.MemberLink, `</td>
					<td class="windowbg" valign="middle" width="4%" align="center">
						`, topic.Replies, `</td>
					<td class="windowbg" valign="middle" width="4%" align="center">
						`, topic.Views, `</td>
					<td class="windowbg2" valign="middle" width="22%">
						<a href="`, topic.LastPost.Href, `"><img src="`, images, `/icons/last_post.gif" alt="`, c.Txt("111"), `" title="`, c.Txt("111"), `" style="float: right;" /></a>
						<span class="smalltext">
							`, topic.LastPost.Time, `<br />
							`, c.Txt("525"), ` `, topic.LastPost.MemberLink, `
						</span>
					</td>
				</tr>`)
}

// unreadLegend renders the icon legend box at the bottom (shared).
func (c *Ctx) unreadLegend() {
	images := c.Theme.ImagesURL()
	participation := ""
	if !c.App.SettingEmpty("enableParticipation") {
		participation = `
					<img src="` + images + `/topic/my_normal_post.gif" alt="" align="middle" /> ` + c.Txt("participation_caption") + `<br />`
	}
	stickyLegend := ""
	if c.App.Setting("enableStickyTopics") == "1" {
		stickyIcon := "normal_post_sticky"
		if !c.Theme.Empty("seperate_sticky_lock") {
			stickyIcon = "quick_sticky"
		}
		stickyLegend = `
					<img src="` + images + `/icons/` + stickyIcon + `.gif" alt="" align="middle" /> ` + c.Txt("smf96") + `<br />`
	}
	pollLegend := ""
	if c.App.Setting("pollMode") == "1" {
		pollLegend = `
					<img src="` + images + `/topic/normal_poll.gif" alt="" align="middle" /> ` + c.Txt("smf43")
	}
	c.O(`
	<div class="tborder"><div class="titlebg2">
		<table cellpadding="8" cellspacing="0" width="55%">
			<tr>
				<td align="left" style="padding-top: 2ex;" class="smalltext">`, participation, `
					<img src="`, images, `/topic/normal_post.gif" alt="" align="middle" /> `, c.Txt("457"), `<br />
					<img src="`, images, `/topic/hot_post.gif" alt="" align="middle" /> `, c.Txt("454"), `<br />
					<img src="`, images, `/topic/veryhot_post.gif" alt="" align="middle" /> `, c.Txt("455"), `
				</td>
				<td align="left" valign="top" style="padding-top: 2ex;" class="smalltext">
					<img src="`, images, `/icons/quick_lock.gif" alt="" align="middle" /> `, c.Txt("456"), `<br />`, stickyLegend, pollLegend, `
				</td>
			</tr>
		</table>
	</div></div>`)
}

// templateRecentMain is template_main() from Recent.template.php.
func templateRecentMain(c *Ctx) {
	page := c.Page.(*RecentCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<div style="padding: 3px;">`)
	c.themeLinktree()
	c.O(`</div>
		<div class="middletext" style="margin-bottom: 1ex;">`, c.Txt("139"), `: `, page.PageIndex, `</div>`)

	for _, post := range page.Posts {
		// Create the per-post button strip (delete, reply, quote, notify).
		var buttons []stripButton
		if post.CanDelete {
			buttons = append(buttons, stripButton{
				Text:   "31",
				Custom: `onclick="return confirm('` + c.Txt("154") + `?');"`,
				URL:    scripturl + "?action=deletemsg;msg=" + itoa(post.ID) + ";topic=" + itoa(post.Topic) + ";recent;sesc=" + c.Sc,
			})
		}
		if post.CanReply {
			buttons = append(buttons,
				stripButton{Text: "146", URL: scripturl + "?action=post;topic=" + itoa(post.Topic) + "." + itoa(post.Start)},
				stripButton{Text: "145", URL: scripturl + "?action=post;topic=" + itoa(post.Topic) + "." + itoa(post.Start) + ";quote=" + itoa(post.ID) + ";sesc=" + c.Sc})
		}
		if post.CanMarkNotify {
			buttons = append(buttons, stripButton{Text: "131", URL: scripturl + "?action=notify;topic=" + itoa(post.Topic) + "." + itoa(post.Start)})
		}

		c.O(`
		<table width="100%" cellpadding="4" cellspacing="1" border="0" class="bordercolor">
				<tr class="titlebg2">
						<td class="middletext">
								<div style="float: left; width: 3ex;">&nbsp;`, post.Counter, `&nbsp;</div>
								<div style="float: left;">&nbsp;`, post.Category.Link, ` / `, post.Board.Link, ` / <b>`, post.Link, `</b></div>
								<div align="right">&nbsp;`, c.Txt("30"), `: `, post.Time, `&nbsp;</div>
						</td>
				</tr>
				<tr>
						<td class="catbg" colspan="3">
							<span class="middletext"> `, c.Txt("109"), ` `, post.FirstPoster.Link, ` - `, c.Txt("22"), ` `, c.Txt("525"), ` `, post.Poster.Link, ` </span>
						</td>
				</tr>
				<tr>
						<td class="windowbg2" colspan="3" valign="top" height="80">
								<div class="post">`, post.Message, `</div>
						</td>
				</tr>`)

		// Are we using tabs?
		if !c.Theme.Empty("use_tabs") {
			c.O(`
			</table>`)
			if len(buttons) != 0 {
				c.O(`
			<table cellpadding="0" cellspacing="0" align="right" style="margin-right: 2ex;">
				<tr>
					`)
				c.templateButtonStrip(buttons, "top")
				c.O(`
				</tr>
			</table><br />`)
			}
		} else {
			if len(buttons) != 0 {
				c.O(`
				<tr>
					<td class="catbg" colspan="3" align="right">
						<table><tr>
						`)
				c.templateButtonStrip(buttons, "top")
				c.O(`
						</tr></table>
					</td>
				</tr>`)
			}
			c.O(`
			</table>`)
		}

		c.O(`
			<br />`)
	}
	c.O(`
			<div class="middletext">`, c.Txt("139"), `: `, page.PageIndex, `</div>`)
}

// templateUnread is template_unread() from Recent.template.php.
func templateUnread(c *Ctx) {
	page := c.Page.(*UnreadCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()
	allParam := ""
	if page.ShowingAllTopics {
		allParam = ";all"
	}

	sortImg := func(col string) string {
		if page.SortBy == col {
			return ` <img src="` + images + `/sort_` + page.SortDirection + `.gif" alt="" border="0" />`
		}
		return ""
	}
	desc := func(col string) string {
		if page.SortBy == col && page.SortDirection == "up" {
			return ";desc"
		}
		return ""
	}
	sortURL := func(col string) string {
		return scripturl + "?action=unread" + allParam + page.QuerystringBoardLimits + ";sort=" + col + desc(col)
	}

	c.O(`
	<table width="100%" border="0" cellspacing="0" cellpadding="3">
		<tr>
			<td>`)
	c.themeLinktree()
	c.O(`</td>
		</tr>
	</table>

	<table border="0" width="100%" cellpadding="0" cellspacing="0">
		<tr>
			<td class="middletext" valign="middle">`, c.Txt("139"), `: `, page.PageIndex, `</td>`)

	// Mark-read button (set up once, used top and bottom).
	markText := "mark_read_short"
	markURL := scripturl + "?action=markasread;sa=board" + page.QuerystringBoardLimits + ";sesc=" + c.Sc
	if page.NoBoardLimits {
		markText = "452"
		markURL = scripturl + "?action=markasread;sa=all;sesc=" + c.Sc
	}
	markBtn := []stripButton{{Text: markText, URL: markURL}}

	if !c.Theme.Empty("show_mark_read") {
		c.O(`
			<td align="right" style="padding-right: 1ex;">
				<table border="0" cellpadding="0" cellspacing="0">
					<tr>
						`)
		c.templateButtonStrip(markBtn, "bottom")
		c.O(`
					</tr>
				</table>
			</td>`)
	}
	c.O(`
		</tr>
	</table>

	<table border="0" width="100%" cellspacing="0" cellpadding="0" class="bordercolor">
		<tr><td>
			<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
				<tr class="titlebg">`)
	if len(page.Topics) != 0 {
		c.O(`
					<td width="10%" colspan="2">&nbsp;</td>
					<td>
						<a href="`, sortURL("subject"), `">`, c.Txt("70"), sortImg("subject"), `</a>
					</td><td width="14%">
						<a href="`, sortURL("starter"), `">`, c.Txt("109"), sortImg("starter"), `</a>
					</td><td width="4%" align="center">
						<a href="`, sortURL("replies"), `">`, c.Txt("110"), sortImg("replies"), `</a>
					</td><td width="4%" align="center">
						<a href="`, sortURL("views"), `">`, c.Txt("301"), sortImg("views"), `</a>
					</td><td width="24%">
						<a href="`, sortURL("last_post"), `">`, c.Txt("111"), sortImg("last_post"), `</a>
					</td>`)
	} else {
		none := c.Txt("unread_topics_visit_none")
		if page.ShowingAllTopics {
			none = c.Txt("151")
		}
		c.O(`
					<td width="100%" colspan="7">`, none, `</td>`)
	}
	c.O(`
				</tr>`)

	for _, topic := range page.Topics {
		c.unreadTopicRow(topic)
	}

	if len(page.Topics) != 0 && !page.ShowingAllTopics {
		c.O(`
				<tr class="titlebg">
					<td colspan="7" align="right" class="middletext"><a href="`, scripturl, `?action=unread;all`, page.QuerystringBoardLimits, `">`, c.Txt("unread_topics_all"), `</a></td>
				</tr>`)
	}

	c.O(`
			</table>
		</td></tr>
	</table>

	<table border="0" width="100%" cellpadding="0" cellspacing="0">
		<tr>
			<td class="middletext" valign="middle">`, c.Txt("139"), `: `, page.PageIndex, `</td>`)
	if !c.Theme.Empty("show_mark_read") {
		c.O(`
			<td align="right" style="padding-right: 1ex;">
				<table border="0" cellpadding="0" cellspacing="0">
					<tr>
						`)
		c.templateButtonStrip(markBtn, "top")
		c.O(`
					</tr>
				</table>
			</td>`)
	}
	c.O(`
		</tr>
	</table><br />`)

	c.unreadLegend()
}

// templateReplies is template_replies() from Recent.template.php.
func templateReplies(c *Ctx) {
	page := c.Page.(*UnreadCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()

	sortImg := func(col string) string {
		if page.SortBy == col {
			return ` <img src="` + images + `/sort_` + page.SortDirection + `.gif" alt="" border="0" />`
		}
		return ""
	}
	desc := func(col string) string {
		if page.SortBy == col && page.SortDirection == "up" {
			return ";desc"
		}
		return ""
	}
	sortURL := func(col string) string {
		return scripturl + "?action=unreadreplies" + page.QuerystringBoardLimits + ";sort=" + col + desc(col)
	}

	hasMark := page.TopicsToMark != "" && !c.Theme.Empty("show_mark_read")
	markBtn := []stripButton{{Text: "452", URL: scripturl + "?action=markasread;sa=unreadreplies;topics=" + page.TopicsToMark + ";sesc=" + c.Sc}}

	c.O(`
	<table width="100%" border="0" cellspacing="0" cellpadding="3" align="center">
		<tr>
			<td>`)
	c.themeLinktree()
	c.O(`</td>
		</tr>
	</table>

	<table border="0" width="100%" cellpadding="0" cellspacing="0">
		<tr>
			<td class="middletext" valign="middle">`, c.Txt("139"), `: `, page.PageIndex, `</td>`)
	if hasMark {
		c.O(`
			<td align="right" style="padding-right: 1ex;">
				<table border="0" cellpadding="0" cellspacing="0">
					<tr>
						`)
		c.templateButtonStrip(markBtn, "bottom")
		c.O(`
					</tr>
				</table>
			</td>`)
	}
	c.O(`
		</tr>
	</table>

	<table border="0" width="100%" cellspacing="0" cellpadding="0" class="bordercolor">
		<tr><td>
			<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
				<tr class="titlebg">`)
	if len(page.Topics) != 0 {
		c.O(`
					<td width="10%" colspan="2">&nbsp;</td>
					<td><a href="`, sortURL("subject"), `">`, c.Txt("70"), sortImg("subject"), `</a></td>
					<td width="14%"><a href="`, sortURL("starter"), `">`, c.Txt("109"), sortImg("starter"), `</a></td>
					<td width="4%" align="center"><a href="`, sortURL("replies"), `">`, c.Txt("110"), sortImg("replies"), `</a></td>
					<td width="4%" align="center"><a href="`, sortURL("views"), `">`, c.Txt("301"), sortImg("views"), `</a></td>
					<td width="24%"><a href="`, sortURL("last_post"), `">`, c.Txt("111"), sortImg("last_post"), `</a></td>`)
	} else {
		c.O(`
					<td width="100%" colspan="7">`, c.Txt("151"), `</td>`)
	}
	c.O(`
				</tr>`)

	for _, topic := range page.Topics {
		c.unreadTopicRow(topic)
	}

	c.O(`
			</table>
		</td></tr>
	</table>

	<table border="0" width="100%" cellpadding="0" cellspacing="0">
		<tr>
			<td class="middletext" valign="middle">`, c.Txt("139"), `: `, page.PageIndex, `</td>`)
	if hasMark {
		c.O(`
			<td align="right" style="padding-right: 1ex;">
				<table border="0" cellpadding="0" cellspacing="0">
					<tr>
						`)
		c.templateButtonStrip(markBtn, "top")
		c.O(`
					</tr>
				</table>
			</td>`)
	}
	c.O(`
		</tr>
	</table><br />
`)

	c.unreadLegend()
}
