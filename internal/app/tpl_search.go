package app

// Hand-port of Themes/default/Search.template.php template_main() (the search
// form). template_results ports with PlushSearch2.

import "strings"

// templateSearchMain is template_main() from Search.template.php.
func templateSearchMain(c *Ctx) {
	page := c.Page.(*SearchCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()
	a := c.App

	c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function selectBoards(ids)
		{
			var toggle = true;

			for (i = 0; i < ids.length; i++)
				toggle = toggle & document.forms.searchform["brd" + ids[i]].checked;

			for (i = 0; i < ids.length; i++)
				document.forms.searchform["brd" + ids[i]].checked = !toggle;
		}

		function expandCollapseBoards()
		{
			var current = document.getElementById("searchBoardsExpand").style.display != "none";

			document.getElementById("searchBoardsExpand").style.display = current ? "none" : "";
			document.getElementById("exandBoardsIcon").src = smf_images_url + (current ? "/expand.gif" : "/collapse.gif");
		}
	// ]]></script>
	<form action="`, scripturl, `?action=search2" method="post" accept-charset="`, c.CharacterSet, `" name="searchform" id="searchform">
		<table cellpadding="3" cellspacing="0" border="0">
			<tr>
				<td>`)
	c.themeLinktree()
	titleImg := ""
	if !c.Theme.Empty("use_buttons") {
		titleImg = `<img src="` + images + `/buttons/search.gif" align="right" style="margin-right: 4px;" alt="" />`
	}
	c.O(`</td>
			</tr>
		</table>

		<table border="0" cellspacing="0" cellpadding="6" align="center" class="tborder">
			<tr class="titlebg">
				<td>`, titleImg, c.Txt("183"), `</td>
			</tr>`)

	if len(page.SearchErrors) > 0 {
		c.O(`
			<tr>
				<td class="windowbg">
					<div style="color: red; margin: 1ex 0 2ex 3ex;">
						`, strings.Join(page.SearchErrors, "<br />"), `
					</div>
				</td>
			</tr>`)
	}

	c.O(`
			<tr>
				<td class="windowbg">`)

	searchVal := ""
	if page.Search != "" {
		searchVal = ` value="` + page.Search + `"`
	}

	if page.SimpleSearch {
		c.O(`
					<b>`, c.Txt("582"), `:</b><br />
					<table border="0" cellpadding="5" cellspacing="0"><tr>
						<td><input type="text" name="search"`, searchVal, ` size="40" /></td>
						<td>&nbsp;<input type="submit" name="submit" value="`, c.Txt("182"), `" /></td>
					</tr>`)
		if a.SettingEmpty("search_simple_fulltext") {
			c.O(`
					<tr>
						<td align="right" class="smalltext">`, c.Txt("search_example"), `</td>
						<td></td>
					</tr>`)
		}
		c.O(`
					</table><br /><br />
					<a href="`, scripturl, `?action=search;advanced" onclick="this.href += ';search=' + escape(document.searchform.search.value);">`, c.Txt("smf298"), `</a>
					<input type="hidden" name="advanced" value="0" />`)
	} else {
		typeSel1, typeSel2 := ` selected="selected"`, ""
		if page.Searchtype != 0 {
			typeSel1, typeSel2 = "", ` selected="selected"`
		}
		userspec := page.Userspec
		if userspec == "" {
			userspec = "*"
		}
		minage := "0"
		if page.Minage != 0 {
			minage = itoa(page.Minage)
		}
		maxage := "9999"
		if page.Maxage != 0 {
			maxage = itoa(page.Maxage)
		}
		showComplete, subjectOnly := "", ""
		if page.ShowComplete {
			showComplete = ` checked="checked"`
		}
		if page.SubjectOnly {
			subjectOnly = ` checked="checked"`
		}
		c.O(`
					<input type="hidden" name="advanced" value="1" />
					<table cellpadding="1" cellspacing="3" border="0">
						<tr>
							<td>
								<b>`, c.Txt("582"), `:</b>
							</td><td>
							</td><td>
									<b>`, c.Txt("583"), `:</b>
							</td>
						</tr><tr>
							<td>
								<input type="text" name="search"`, searchVal, ` size="40" />
								<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
									if (typeof(window.addEventListener) == "undefined")
									{
										if (window.attachEvent)
										{
											window.addEventListener = function (sEvent, funcHandler, bCapture)
											{
												window.attachEvent("on" + sEvent, funcHandler);
											}
										}
										else
										{
											window.addEventListener = function (sEvent, funcHandler, bCapture) 
											{
												window["on" + sEvent] = funcHandler;
											}
										}
									}
									function initSearch()
									{
										if (document.forms.searchform.search.value.indexOf("%u") != -1)
											document.forms.searchform.search.value = unescape(document.forms.searchform.search.value);
									}
									window.addEventListener("load", initSearch, false);
								// ]]></script>
							</td><td style="padding-right: 2ex;">
								<select name="searchtype">
									<option value="1"`, typeSel1, `>`, c.Txt("343"), `</option>
									<option value="2"`, typeSel2, `>`, c.Txt("344"), `</option>
								</select>
							</td><td>
								<input type="text" name="userspec" value="`, userspec, `" size="40" />
							</td>
						</tr>`)
		if a.SettingEmpty("search_simple_fulltext") {
			c.O(`
						<tr>
							<td align="right" class="smalltext" style="padding: 0px;">`, c.Txt("search_example"), `</td>
							<td colspan="2"></td>
						</tr>`)
		}
		c.O(`
						<tr>
							<td colspan="3"><br />
								<div style="text-align: left; width: 45%; float: right; margin-right: 2ex;">
									<div class="small_header" style="margin-bottom: 2px;"><b>`, c.Txt("search_post_age"), `: </b></div><br />
									`, c.Txt("search_between"), ` <input type="text" name="minage" value="`, minage, `" size="5" maxlength="5" />&nbsp;`, c.Txt("search_and"), `&nbsp;<input type="text" name="maxage" value="`, maxage, `" size="5" maxlength="5" /> `, c.Txt("579"), `.
								</div>
								<div style="width: 45%;">
									<div class="small_header" style="margin-bottom: 2px;"><b>`, c.Txt("search_options"), `:</b></div>
									<label for="show_complete"><input type="checkbox" name="show_complete" id="show_complete" value="1"`, showComplete, ` class="check" /> `, c.Txt("search_show_complete_messages"), `</label><br />
									<label for="subject_only"><input type="checkbox" name="subject_only" id="subject_only" value="1"`, subjectOnly, ` class="check" /> `, c.Txt("search_subject_only"), `</label>
								</div>
							</td>
						</tr><tr>
							<td style="padding-top: 2ex;" colspan="2"><b>`, c.Txt("search_order"), `:</b></td>
							<td></td>
						</tr><tr>
							<td colspan="2">
								<select name="sort">
									<option value="relevance|desc">`, c.Txt("search_orderby_relevant_first"), `</option>
									<option value="numReplies|desc">`, c.Txt("search_orderby_large_first"), `</option>
									<option value="numReplies|asc">`, c.Txt("search_orderby_small_first"), `</option>
									<option value="ID_MSG|desc">`, c.Txt("search_orderby_recent_first"), `</option>
									<option value="ID_MSG|asc">`, c.Txt("search_orderby_old_first"), `</option>
								</select>
							</td>
							<td></td>
						</tr>
					</table><br />`)

		if page.Topic != 0 {
			c.O(`
					`, c.Txt("search_specific_topic"), ` &quot;`, page.TopicLink, `&quot;.<br />
					<input type="hidden" name="topic" value="`, page.TopicID, `" />`)
		} else {
			c.O(`	
					<fieldset class="windowbg2" style="padding: 10px; margin-left: 5px; margin-right: 5px;">
						<a href="javascript:void(0);" onclick="expandCollapseBoards(); return false;"><img src="`, images, `/expand.gif" id="exandBoardsIcon" alt="" /></a> <a href="javascript:void(0);" onclick="expandCollapseBoards(); return false;"><b>`, c.Txt("189"), `</b></a><br />

						<table id="searchBoardsExpand" width="100%" border="0" cellpadding="1" cellspacing="0" align="center" style="margin-top: 1ex; display: none;">`)
			alternate := true
			for _, board := range page.BoardColumns {
				if alternate {
					c.O(`
							<tr>`)
				}
				c.O(`
								<td width="50%">`)
				if board.Kind == "board" {
					sel := ""
					if board.Selected {
						sel = ` checked="checked"`
					}
					c.O(`
									<label for="brd`, board.ID, `" style="margin-left: `, board.ChildLevel, `ex;"><input type="checkbox" id="brd`, board.ID, `" name="brd[`, board.ID, `]" value="`, board.ID, `"`, sel, ` class="check" />`, board.Name, `</label>`)
				} else if board.Kind == "cat" {
					c.O(`
									<a href="javascript:void(0);" onclick="selectBoards([`, joinIntsSep(board.ChildIDs, ", "), `]); return false;" style="text-decoration: underline;">`, board.Name, `</a>`)
				}
				c.O(`
								</td>`)
				if !alternate {
					c.O(`
							</tr>`)
				}
				alternate = !alternate
			}
			c.O(`
						</table><br />
						<input type="checkbox" name="all" id="check_all" value="" checked="checked" onclick="invertAll(this, this.form, 'brd');" class="check" /><i> <label for="check_all">`, c.Txt("737"), `</label></i><br />
					</fieldset> `)
		}

		c.O(`
					<br />
					<div style="padding: 2px;"><input type="submit" name="submit" value="`, c.Txt("182"), `" /></div>`)
	}

	c.O(`
				</td>
			</tr>
		</table>
	</form>`)
}

// templateSearchResults is template_results() from Search.template.php.
// Quick-moderation (display_quick_mod, off by default) is not rendered.
func templateSearchResults(c *Ctx) {
	page := c.Page.(*SearchResultsCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()

	// The adjust-query box (shown when there are no results).
	if page.NoResults {
		searchVal := ""
		if page.Search != "" {
			searchVal = ` value="` + page.Search + `"`
		}
		c.O(`
	<div class="tborder">
		<table width="100%" cellpadding="4" cellspacing="0" border="0" class="bordercolor" style="margin-bottom: 2ex;">
			<tr class="titlebg">
				<td>`, c.Txt("search_adjust_query"), `</td>
			</tr>
			<tr>
				<td class="windowbg">
					<form action="`, scripturl, `?action=search2" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
						<b>`, c.Txt("582"), `:</b><br />
						<input type="text" name="search"`, searchVal, ` size="40" />
						<input type="submit" name="submit" value="`, c.Txt("search_adjust_submit"), `" />

						<input type="hidden" name="searchtype" value="`, page.Searchtype, `" />
						<input type="hidden" name="userspec" value="`, page.Userspec, `" />
						<input type="hidden" name="show_complete" value="`, boolDigit(page.ShowComplete), `" />
						<input type="hidden" name="subject_only" value="`, boolDigit(page.SubjectOnly), `" />
						<input type="hidden" name="minage" value="`, minageStr(page.Minage), `" />
						<input type="hidden" name="maxage" value="`, maxageStr(page.Maxage), `" />
						<input type="hidden" name="sort" value="`, sortOrRelevance(page.Sort), `" />`)
		for _, b := range page.Brd {
			c.O(`
						<input type="hidden" name="brd[`, b, `]" value="`, b, `" />`)
		}
		c.O(`
					</form>
				</td>
			</tr>
		</table>
	</div>`)
	}

	if page.Compact {
		c.O(`
	<div style="padding: 3px;">`)
		c.themeLinktree()
		c.O(`</div>
	<div class="middletext">`, c.Txt("139"), `: `, page.PageIndex, `</div>
		<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
			<tr class="titlebg">`)
		if !page.NoResults {
			c.O(`
				<td width="4%"></td>
				<td width="4%"></td>
				<td width="56%">`, c.Txt("70"), `</td>
				<td width="6%" align="center">`, c.Txt("search_relevance"), `</td>
				<td width="12%">`, c.Txt("109"), `</td>
				<td width="18%" align="center">`, c.Txt("search_date_posted"), `</td>`)
		} else {
			c.O(`
				<td width="100%" colspan="5">`, c.Txt("search_no_results"), `</td>`)
		}
		c.O(`
			</tr>`)

		for _, topic := range page.Topics {
			class := c.unreadClass(topic.Class)
			bg3 := ""
			lockImg, stickyOpen, stickyClose := "", "", ""
			if topic.IsSticky && !c.Theme.Empty("seperate_sticky_lock") {
				bg3 = "3"
			}
			if topic.IsLocked && !c.Theme.Empty("seperate_sticky_lock") {
				lockImg = `<img src="` + images + `/icons/quick_lock.gif" align="right" alt="" style="margin: 0;" />`
			}
			if topic.IsSticky && !c.Theme.Empty("seperate_sticky_lock") {
				stickyOpen = `<img src="` + images + `/icons/show_sticky.gif" align="right" alt="" style="margin: 0;" /><b>`
			}
			if topic.IsSticky {
				stickyClose = "</b>"
			}
			c.O(`
			<tr>
				<td class="windowbg2" valign="top" align="center" width="4%">
					<img src="`, images, `/topic/`, class, `.gif" alt="" /></td>
				<td class="windowbg2" valign="top" align="center" width="4%">
					<img src="`, topic.FirstPost.IconURL, `" alt="" align="middle" /></td>
				<td class="windowbg`, bg3, `" valign="middle">
					`, lockImg, `
					`, stickyOpen, topic.FirstPost.Link, stickyClose, `
				<div class="smalltext"><i>`, c.Txt("smf88"), ` `, topic.Board.Link, `</i></div>`)
			for _, m := range topic.Matches {
				c.O(`<br />
					<div class="quoteheader" style="margin-left: 20px;"><a href="`, scripturl, `?topic=`, topic.ID, `.msg`, m.ID, `#msg`, m.ID, `">`, m.SubjectHighlighted, `</a> `, c.Txt("525"), ` `, m.Member.Link, `</div>`)
				if m.BodyHighlighted != "" {
					c.O(`
					<div class="quote" style="margin-left: 20px;">`, m.BodyHighlighted, `</div>`)
				}
			}
			c.O(`
				</td>
				<td class="windowbg2" valign="top" width="6%" align="center">
					`, topic.Relevance, `
				</td><td class="windowbg" valign="top" width="12%">
					`, topic.FirstPost.MemberLink, `
				</td><td class="windowbg" valign="top" width="18%" align="center">
					`, topic.FirstPost.Time, `
				</td>
			</tr>`)
		}
		c.O(`
		</table>
		<div class="middletext">`, c.Txt("139"), `: `, page.PageIndex, `</div>`)
		searchJumpTo(c)
	} else {
		// Full ("show complete") view.
		c.O(`
		<div style="padding: 3px;">`)
		c.themeLinktree()
		c.O(`</div>
		<div class="middletext">`, c.Txt("139"), `: `, page.PageIndex, `</div>`)
		if page.NoResults {
			c.O(`
		<table border="0" width="100%" cellspacing="0" cellpadding="0" class="bordercolor"><tr><td>
			<table border="0" width="100%" cellpadding="2" cellspacing="1" class="bordercolor"><tr class="windowbg2"><td><br />
				<b>(`, c.Txt("search_no_results"), `)</b><br /><br />
			</td></tr></table>
		</td></tr></table>`)
		}
		quoteBtn := c.createButton("quote.gif", "145", "145", `align="middle"`)
		replyBtn := c.createButton("reply_sm.gif", "146", "146", `align="middle"`)
		notifyBtn := c.createButton("notify_sm.gif", "131", "131", `align="middle"`)
		for _, topic := range page.Topics {
			for _, m := range topic.Matches {
				var buttons []string
				if topic.CanReply {
					buttons = append(buttons,
						`<a href="`+scripturl+`?action=post;topic=`+itoa(topic.ID)+`.`+m.Start+`">`+replyBtn+`</a>`,
						`<a href="`+scripturl+`?action=post;topic=`+itoa(topic.ID)+`.0;quote=`+itoa(m.ID)+`/`+m.Start+`;sesc=`+c.Sc+`">`+quoteBtn+`</a>`)
				}
				if topic.CanMarkNot {
					buttons = append(buttons, `<a href="`+scripturl+`?action=notify;topic=`+itoa(topic.ID)+`.`+m.Start+`">`+notifyBtn+`</a>`)
				}
				c.O(`
			<div class="tborder">
				<table border="0" width="100%" cellspacing="0" cellpadding="0" class="bordercolor">
					<tr>
						<td>
							<table width="100%" cellpadding="4" cellspacing="1" border="0" class="bordercolor">
								<tr class="titlebg">
									<td>
										<div style="float: left; width: 3ex;">&nbsp;`, m.Counter, `&nbsp;</div>
										<div style="float: left;">&nbsp;`, topic.Category.Link, ` / `, topic.Board.Link, ` / <a href="`, scripturl, `?topic=`, topic.ID, `.`, m.Start, `;topicseen#msg`, m.ID, `">`, m.SubjectHighlighted, `</a></div>
										<div align="right">`, c.Txt("30"), `: `, m.Time, `&nbsp;</div>
									</td>
								</tr><tr class="catbg">
									<td>
										<div style="float: left;">`, c.Txt("109"), ` `, topic.FirstPost.MemberLink, `, `, c.Txt("72"), ` `, c.Txt("525"), ` `, m.Member.Link, `</div>
										<div align="right">`, c.Txt("search_relevance"), `: `, topic.Relevance, `</div>
									</td>
								</tr><tr>
									<td width="100%" valign="top" class="windowbg2">
										<div class="post">`, m.BodyHighlighted, `</div>
									</td>
								</tr><tr class="windowbg">
									<td class="middletext" align="right">&nbsp;`, joinStr(buttons, c.MenuSeparator), `</td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
			</div>`)
			}
		}
		c.O(`
			<div class="middletext">`, c.Txt("139"), `: `, page.PageIndex, `</div>`)
	}
}

func boolDigit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
func minageStr(n int) string {
	if n != 0 {
		return itoa(n)
	}
	return "0"
}
func maxageStr(n int) string {
	if n != 0 {
		return itoa(n)
	}
	return "9999"
}
func sortOrRelevance(s string) string {
	if s != "" {
		return s
	}
	return "relevance"
}

// searchJumpTo renders the bottom jump-to dropdown used by the compact results.
func searchJumpTo(c *Ctx) {
	scripturl := c.App.ScriptURL
	if !c.Theme.Empty("linktree_inline") {
		c.O(`
		<div style="padding: 3px;">`)
		c.themeLinktree()
		c.O(`</div>`)
	}
	c.O(`
		<table cellpadding="0" cellspacing="0" width="100%">
			<tr>
				<td class="smalltext" align="right" valign="middle">
					<form action="`, scripturl, `" method="get" accept-charset="`, c.CharacterSet, `">
						<label for="jumpto">`, c.Txt("160"), `:</label>
						<select name="jumpto" id="jumpto" onchange="if (this.selectedIndex > 0 &amp;&amp; this.options[this.selectedIndex].value) window.location.href = smf_scripturl + this.options[this.selectedIndex].value.substr(smf_scripturl.indexOf('?') == -1 || this.options[this.selectedIndex].value.substr(0, 1) != '?' ? 0 : 1);">
							<option value="">`, c.Txt("251"), `:</option>`)
	for _, category := range c.JumpTo {
		c.O(`
							<option value="" disabled="disabled">-----------------------------</option>
							<option value="#`, category.ID, `">`, category.Name, `</option>
							<option value="" disabled="disabled">-----------------------------</option>`)
		for _, board := range category.Boards {
			c.O(`
							<option value="?board=`, board.ID, `.0"> `, repeatStr("==", board.ChildLevel), `=> `, board.Name, `</option>`)
		}
	}
	c.O(`
						</select>
						&nbsp;
						<input type="button" value="`, c.Txt("161"), `" onclick="if (this.form.jumpto.options[this.form.jumpto.selectedIndex].value) window.location.href = '`, scripturl, `' + this.form.jumpto.options[this.form.jumpto.selectedIndex].value;" />
					</form>
				</td>
			</tr>
		</table>`)
}

func repeatStr(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
