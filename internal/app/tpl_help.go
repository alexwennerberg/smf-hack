package app

// Hand-port of the template_manual_*() content sub-templates from
// Themes/default/Help.template.php (the ?action=help user manual). The menu
// layer (template_manual_above/below) lives in help.go alongside ShowHelp().

import "os"

// templateManualIntro is template_manual_intro().
func templateManualIntro(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`

	<p>`, c.Txt("manual_index_you_have_arrived_part1"), `<a href="http://www.simplemachines.org/">`, c.Txt("manual_index_you_have_arrived_link_site0"), `</a>`, c.Txt("manual_index_you_have_arrived_part2"), `<a href="`, scripturl, `?action=help;page=index#board">`, c.Txt("manual_index_you_have_arrived_link_site0_board"), `</a>`, c.Txt("manual_index_you_have_arrived_part3"), `</p>
	<p>`, c.Txt("manual_index_guest_permit_read_part1"), `<a href="`, scripturl, `?action=help;page=registering">`, c.Txt("manual_index_guest_permit_read_link_registering"), `</a>`, c.Txt("manual_index_guest_permit_read_part2"), `</p>
	<ol>
		<li><a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_index_main_menu"), `</a></li>
		<li><a href="`, scripturl, `?action=help;page=index#board">`, c.Txt("manual_index_sec_board_index"), `</a></li>
		<li><a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_index_sec_msg_index"), `</a></li>
		<li><a href="`, scripturl, `?action=help;page=index#topic">`, c.Txt("manual_index_sec_topic"), `</a></li>
	</ol>
	<h2 id="main">`, c.Txt("manual_index_main_menu"), `</h2>
	<p>`, c.Txt("manual_index_suppossing_guest"), `</p>
	<ul>
		<li>`, c.Txt("manual_index_home_desc_part1"), `<a href="`, scripturl, `?action=help;page=index#board">`, c.Txt("manual_index_home_desc_link_board"), `</a>`, c.Txt("manual_index_home_desc_part2"), `</li>
		<li>`, c.Txt("manual_index_help_desc"), `</li>
		<li>`, c.Txt("manual_index_search_desc_part1"), `<a href="`, scripturl, `?action=help;page=searching">`, c.Txt("manual_index_search_desc_link_searching"), `</a>`, c.Txt("manual_index_search_desc_part2"), `</li>
		<li>`, c.Txt("manual_index_calendar_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#calendar">`, c.Txt("manual_index_calendar_desc_link_posting_calendar"), `</a>`, c.Txt("manual_index_calendar_desc_part2"), `</li>
		<li>`, c.Txt("manual_index_login_desc_part1"), `<a href="`, scripturl, `?action=help;page=loginout">`, c.Txt("manual_index_login_desc_link_loginout"), `</a>`, c.Txt("manual_index_login_desc_part2"), `</li>
		<li>`, c.Txt("manual_index_register_desc_part1"), `<a href="`, scripturl, `?action=help;page=registering">`, c.Txt("manual_index_register_desc_link_registering"), `</a>`, c.Txt("manual_index_register_desc_part2"), `</li>
	</ul>
	<p>`, c.Txt("manual_index_once_registered"), `</p>
	<ul>
		<li>`, c.Txt("manual_index_home_reg"), `</li>
		<li>`, c.Txt("manual_index_help_reg"), `</li>
		<li>`, c.Txt("manual_index_search_reg"), `</li>
		<li>`, c.Txt("manual_index_profile_reg_part1"), `<a href="`, scripturl, `?action=help;page=profile">`, c.Txt("manual_index_profile_reg_link_profile"), `</a>`, c.Txt("manual_index_profile_reg_part2"), `</li>
		<li>`, c.Txt("manual_index_calendar_reg"), `</li>
		<li>`, c.Txt("manual_index_logout_reg_part1"), `<a href="`, scripturl, `?action=help;page=loginout#logout">`, c.Txt("manual_index_logout_reg_link_loginout_logout"), `</a>`, c.Txt("manual_index_logout_reg_part2"), `</li>
	</ul>
	<p>`, c.Txt("manual_index_forum_admins_note_presentation"), `</p>
	<h2 id="board">`, c.Txt("manual_index_sec_board_index"), `</h2>
	<p>`, c.Txt("manual_index_sec_board_index_def"), `</p>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<table width="100%" cellpadding="3" cellspacing="0">
				<tr>
					<td valign="bottom"><span class="nav"><img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_index_forum_name"), `</a></b></span></td>
				</tr>
			</table><script language="JavaScript1.2" type="text/javascript">
//<![CDATA[
			var collapseExpand = false;
			function collapseExpandCategory()
			{
					document.getElementById("collapseArrow").src = smf_images_url + "/" + (collapseExpand ? "collapse.gif" : "expand.gif");
					document.getElementById("collapseArrow").alt = collapseExpand ? "-" : "+";
					document.getElementById("collapseCategory").style.display = collapseExpand ? "" : "none";
					collapseExpand = !collapseExpand;
			}
			function markBoardRead()
			{
					document.getElementById("board-new-or-not").src = smf_images_url + "/" + "off.gif";
					document.getElementById("board-new-or-not").alt = "`, c.Txt("manual_index_no_new"), `";
			}
//]]>
</script>
			<div class="tborder">
				<table border="0" width="100%" cellspacing="1" cellpadding="5">
					<tr>
						<td colspan="4" class="catbg" height="18"><a href="javascript:collapseExpandCategory();"><img src="`, c.Theme.ImagesURL(), `/collapse.gif" alt="-" border="0" id="collapseArrow" name="collapseArrow" /></a>&nbsp; <a href="javascript:collapseExpandCategory();" class="board">`, c.Txt("manual_index_cat_name"), `</a></td>
					</tr>
					<tr id="collapseCategory" class="windowbg2">
						<td class="windowbg" width="6%" align="center" valign="top"><img src="`, c.Theme.ImagesURL(), `/on.gif" id="board-new-or-not" alt="`, c.Txt("manual_index_new_posts"), `" name="board-new-or-not" /></td>
						<td align="left" class="windowbg"><b><a href="`, scripturl, `?action=help;page=index#message" class="board">`, c.Txt("manual_index_board_name"), `</a></b><br />
						`, c.Txt("manual_index_board_desc"), `</td>
						<td class="windowbg" valign="middle" align="center" style="width: 12ex;"><span class="smalltext">`, c.Txt("manual_index_topics_posts"), `</span></td>
						<td class="smalltext" valign="middle" width="22%" class="windowbg">`, c.Txt("manual_index_date_time"), `</td>
					</tr>
				</table>
			</div>`)

	// This changes dependant on theme really...
	markReadButton := []stripButton{
		{URL: "javascript:markBoardRead();", Text: "452"},
	}
	if !c.Theme.Empty("use_tabs") {
		align1 := "left"
		if c.RightToLeft {
			align1 = "right"
		}
		align2 := "right"
		if c.RightToLeft {
			align2 = "left"
		}
		c.O(`
		<table border="0" width="100%" cellspacing="0" cellpadding="5">
			<tr>
				<td align="`, align1, `" class="smalltext">
					<img src="`, c.Theme.ImagesURL(), `/new_some.gif" alt="" align="middle" /> `, c.Txt("manual_index_new_posts"), `
					<img src="`, c.Theme.ImagesURL(), `/new_none.gif" alt="" align="middle" style="margin-left: 4ex;" /> `, c.Txt("manual_index_no_new"), `
				</td>
				<td align="`, align2, `">
					<table cellpadding="0" cellspacing="0" border="0" style="position: relative; top: -5px;">
						<tr>
							`)
		c.templateButtonStrip(markReadButton, "top")
		c.O(`
						</tr>
					</table>
				</td>
			</tr>
		</table>`)
	} else {
		c.O(`
			<br />
			<div class="tborder" style="padding: 3px;">
				<table border="0" width="100%" cellspacing="0" cellpadding="5">
					<tr class="titlebg">
						<td align="left" class="smalltext">`)

		// To back support the classic theme we do a little hack here...
		if _, err := os.Stat(c.Theme.Get("theme_dir") + "/images/" + c.User.Language + "/new_some.gif"); err == nil {
			c.O(`
				<img src="`, c.Theme.ImagesURL(), `/`, c.User.Language, `/new_some.gif" alt="`, c.Txt("manual_index_new_posts"), `" border="0" />&nbsp;&nbsp;<img src="`, c.Theme.ImagesURL(), `/`, c.User.Language, `/new_none.gif" alt="`, c.Txt("manual_index_no_new"), `" border="0" />`)
		} else {
			c.O(`
				<img src="`, c.Theme.ImagesURL(), `/new_some.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_new_posts"), `<img src="`, c.Theme.ImagesURL(), `/new_none.gif" alt="" align="middle" style="margin-left: 4ex;" />&nbsp; `, c.Txt("manual_index_no_new"))
		}

		c.O(`
						</td>
						`)
		c.templateButtonStrip(markReadButton, "top")
		c.O(`
					</tr>
				</table>
			</div><br />`)
	}

	c.O(`
		</div>
	</div><br />
		<ul>
		<li>`, c.Txt("manual_index_f_name"), `</li>
		<li>`, c.Txt("manual_index_cat"), `</li>
		<li>`, c.Txt("manual_index_b_name_part1"), `<a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_index_b_name_link_message"), `</a>`, c.Txt("manual_index_b_name_part2"), `</li>
		<li>`, c.Txt("manual_index_b_desc"), `</li>
		<li>`, c.Txt("manual_index_n_no_n_posts"), `</li>
		<li>`, c.Txt("manual_index_m_read"), `</li>
	</ul>
	<h2 id="message">`, c.Txt("manual_index_sec_msg_index"), `</h2>
	<p>`, c.Txt("manual_index_sec_msg_index_def"), `</p>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<script language="JavaScript1.2" type="text/javascript">
//<![CDATA[
			var currentSort = false;
			function sortLastPost()
			{
					document.getElementById("sort-arrow").src = smf_images_url + "/" + (currentSort ? "sort_down.gif" : "sort_up.gif");
					document.getElementById("sort-arrow").alt = "";
					currentSort = !currentSort;
			}
			function markMessageRead()
			{
					document.getElementById("message-new-or-not").style.display = "none";
			}
//]]>
</script>
			<table width="100%" cellpadding="3" cellspacing="0">
				<tr>
					<td><span class="nav"><img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_index_forum_name"), `</a></b><br />
					<img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_index_cat_name"), `</a></b><br />
					<img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#message" class="nav">`, c.Txt("manual_index_board_name"), `</a></b></span></td>
				</tr>
			</table>`)

	// Create the buttons we need here...
	mindexButtons := []stripButton{
		{Key: "markmread", URL: "javascript:markMessageRead();", Text: "mark_read_short"},
		{Key: "notify", URL: scripturl + "?action=help;page=index#message", Text: "manual_index_notify", Custom: `onclick="return confirm('` + c.Txt("manual_index_ru_sure_notify") + `');"`},
		{Key: "topic", URL: scripturl + "?action=help;page=post#newtopic", Text: "manual_index_start_new"},
		{Key: "poll", URL: scripturl + "?action=help;page=post#newpoll", Text: "manual_index_new_poll"},
	}

	if !c.Theme.Empty("use_tabs") {
		c.O(`
			<table width="100%" cellpadding="0" cellspacing="0" border="0">
				<tr>
					<td class="middletext">`, c.Txt("manual_index_pages"), `: [<b>1</b>]</td>
					<td align="right" style="padding-right: 1ex;">
						<table cellpadding="0" cellspacing="0">
							<tr>
								`)
		c.templateButtonStrip(mindexButtons, "bottom")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	} else {
		c.O(`
			<table width="100%" cellpadding="3" cellspacing="0" border="0" class="tborder" style="margin-bottom: 1ex;">
				<tr>
					<td align="left" class="catbg" width="100%" height="30">
						<table cellpadding="3" cellspacing="0" width="100%">
							<tr>
								<td><b>`, c.Txt("manual_index_pages"), `:</b> [<b>1</b>]</td>
								`)
		c.templateButtonStrip(mindexButtons, "bottom")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	}
	c.O(`
			<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
				<tr class="titlebg">
					<td width="9%" colspan="2"></td>
					<td><a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_index_subject"), `</a></td>
					<td width="14%"><a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_index_started_by"), `</a></td>
					<td width="4%" align="center"><a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_index_replies"), `</a></td>
					<td width="4%" align="center"><a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_index_views"), `</a></td>
					<td width="22%"><a href="javascript:sortLastPost();">`, c.Txt("manual_index_last_post"), ` &nbsp; <img id="sort-arrow" src="`, c.Theme.ImagesURL(), `/sort_down.gif" alt="" border="0" name="sort-arrow" /></a></td>
				</tr>
				<tr>
					<td class="windowbg2" valign="middle" align="center" width="5%"><img src="`, c.Theme.ImagesURL(), `/topic/my_normal_poll.gif" alt="" /></td>
					<td class="windowbg2" valign="middle" align="center" width="4%"><img src="`, c.Theme.ImagesURL(), `/post/xx.gif" alt="" align="middle" /></td>
					<td class="windowbg" valign="middle"><a href="`, scripturl, `?action=help;page=index#topic" class="board">`, c.Txt("manual_index_topic_subject"), `</a> <a href="`, scripturl, `?action=help;page=index#topic"><img id="message-new-or-not" src="`, c.Theme.ImagesURL(), `/`, c.User.Language, `/new.gif" border="0" alt="`, c.Txt("manual_index_new"), `" name="message-new-or-not" /></a></td>
					<td class="windowbg2" valign="middle" width="14%"><a href="`, scripturl, `?action=help;page=profile" class="board">`, c.Txt("manual_index_topic_starter"), `</a></td>
					<td class="windowbg" valign="middle" width="4%" align="center">0</td>
					<td class="windowbg" valign="middle" width="4%" align="center">0</td>
					<td class="windowbg2" valign="middle" width="22%"><span class="smalltext">`, c.Txt("manual_index_last_poster"), `</span></td>
				</tr>
			</table>`)

	if !c.Theme.Empty("use_tabs") {
		c.O(`
			<table width="100%" cellpadding="0" cellspacing="0" border="0">
				<tr>
					<td class="middletext">`, c.Txt("manual_index_pages"), `: [<b>1</b>]</td>
					<td align="right" style="padding-right: 1ex;">
						<table cellpadding="0" cellspacing="0">
							<tr>
								`)
		c.templateButtonStrip(mindexButtons, "top")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	} else {
		c.O(`
			<table width="100%" cellpadding="3" cellspacing="0" border="0" class="tborder" style="margin-top: 1ex;">
				<tr>
					<td align="left" class="catbg" width="100%" height="30">
						<table cellpadding="3" cellspacing="0" width="100%">
							<tr>
								<td><b>`, c.Txt("manual_index_pages"), `:</b> [<b>1</b>]</td>
								`)
		c.templateButtonStrip(mindexButtons, "bottom")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	}
	c.O(`
			<table cellpadding="0" cellspacing="0" width="100%">
				<tr>
					<td class="smalltext" align="left" style="padding-top: 1ex;"><img src="`, c.Theme.ImagesURL(), `/topic/my_normal_post.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_normal_post"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/normal_post.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_normal_topic"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/hot_post.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_hot_post"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/veryhot_post.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_very_hot_post"), `</td>
					<td class="smalltext" align="left" valign="top" style="padding-top: 1ex;"><img src="`, c.Theme.ImagesURL(), `/topic/normal_post_locked.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_locked"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/normal_post_sticky.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_sticky"), `<br />
					<img src="`, c.Theme.ImagesURL(), `/topic/normal_poll.gif" alt="" align="middle" />&nbsp; `, c.Txt("manual_index_poll"), `</td>
					<td class="smalltext" align="right" valign="middle">
						<form action="`, scripturl, `?action=help;page=index" method="get" accept-charset="`, c.CharacterSet, `">
							<label for="jumpto">`, c.Txt("manual_index_jump_to"), `</label>: <select name="jumpto" id="jumpto" onchange="if (this.options[this.selectedIndex].value) window.location.href='`, scripturl, `?action=help;page=index' + this.options[this.selectedIndex].value;">
								<option value="">
									`, c.Txt("manual_index_destination"), `:
								</option>
								<option value="">
									-----------------------------
								</option>
								<option value="#board">
									`, c.Txt("manual_index_cat_name"), `
								</option>
								<option value="">
									-----------------------------
								</option>
								<option value="#message">
									=&gt; `, c.Txt("manual_index_board_name"), `
								</option>
								<option value="#message">
									=&gt; `, c.Txt("manual_index_another_board"), `
								</option>
							</select>&nbsp; <input type="button" onclick="if (this.form.jumpto.options[this.form.jumpto.selectedIndex].value) window.location.href = '`, scripturl, `?action=help;page=index' + this.form.jumpto.options[this.form.jumpto.selectedIndex].value;" value="`, c.Txt("manual_index_go"), `" />
						</form>
					</td>
				</tr>
			</table><br />
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_index_nav_tree"), `</li>
		<li>`, c.Txt("manual_index_page_number"), `</li>
		<li>`, c.Txt("manual_index_mark_read_button"), `</li>
		<li>`, c.Txt("manual_index_notify_button"), `</li>
		<li>`, c.Txt("manual_index_new_topic_poll_button_part1"), `<a href="`, scripturl, `?action=help;page=post">`, c.Txt("manual_index_new_topic_poll_button_link_posting"), `</a>`, c.Txt("manual_index_new_topic_poll_button_part2"), `</li>
		<li>`, c.Txt("manual_index_subject_replies_etc"), `</li>
		<li>`, c.Txt("manual_index_topic_icons"), `</li>
		<li>`, c.Txt("manual_index_post_icons"), `</li>
		<li>`, c.Txt("manual_index_topic_subject_links_part1"), `<a href="`, scripturl, `?action=help;page=index#topic">`, c.Txt("manual_index_topic_subject_links_link_topic"), `</a>`, c.Txt("manual_index_topic_subject_links_part2"), `</li>
		<li>`, c.Txt("manual_index_where_topic_part1"), `<a href="`, scripturl, `?action=help;page=profile">`, c.Txt("manual_index_where_topic_link_profile"), `</a>`, c.Txt("manual_index_where_topic_part2"), `</li>
		<li>`, c.Txt("manual_index_jump_to_menu"), `</li>
	</ul>
	<h2 id="topic">`, c.Txt("manual_index_sec_topic"), `</h2>
	<p>`, c.Txt("manual_index_ref_thread"), `</p>`)

	// The buttons...
	displayButtons := []stripButton{
		{Key: "reply", URL: scripturl + "?action=help;page=post#reply", Text: "manual_index_reply"},
		// 'notify' collides with mindexButtons' key; PHP reuses the cached
		// mindex notify HTML here (page=index#message), so we must not reset.
		{Key: "notify", URL: scripturl + "?action=help;page=post#topic", Text: "manual_index_notify", Custom: `onclick="return confirm('` + c.Txt("manual_index_ru_sure_enable_notify") + `');"`},
		{Key: "markunread", URL: scripturl + "?action=help;page=post#topic", Text: "manual_index_mark_unread"},
		{Key: "sendtopic", URL: scripturl + "?action=help;page=post#topic", Text: "manual_index_send_topic"},
		{Key: "print", URL: scripturl + "?action=help;page=post#topic", Text: "manual_index_print"},
	}

	c.O(`
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<table width="100%" cellpadding="3" cellspacing="0">
				<tr>
					<td valign="bottom"><span class="nav"><img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_index_forum_name"), `</a></b><br />
					<img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_index_cat_name"), `</a></b><br />
					<img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#message" class="nav">`, c.Txt("manual_index_board_name"), `</a></b><br />
					<img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#topic" class="nav">`, c.Txt("manual_index_topic_subject"), `</a></b></span></td>
				</tr>
			</table>`)

	if !c.Theme.Empty("use_tabs") {
		c.O(`
			<table width="100%" cellpadding="0" cellspacing="0" border="0">
				<tr>
					<td class="middletext" valign="bottom" style="padding-bottom: 4px;"><b>`, c.Txt("manual_index_pages"), `:</b> [<b>1</b>]</td>
					<td align="right" style="padding-right: 1ex;">
						<table cellpadding="0" cellspacing="0">
							<tr>
								`)
		c.templateButtonStrip(displayButtons, "bottom")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	} else {
		c.O(`
			<table width="100%" cellpadding="3" cellspacing="0" border="0" class="tborder" style="margin-bottom: 1ex;">
				<tr>
					<td align="left" class="catbg" width="100%" height="35">
						<table cellpadding="3" cellspacing="0" width="100%">
							<tr>
								<td><b>`, c.Txt("manual_index_pages"), `:</b> [<b>1</b>]</td>
								`)
		c.templateButtonStrip(displayButtons, "bottom")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	}
	c.O(`
			<table width="100%" cellpadding="3" cellspacing="0" border="0" class="tborder" style="border-bottom: 0;">
				<tr class="catbg3">
					<td valign="middle" align="left" width="2%" style="padding-left: 6px;"><img src="`, c.Theme.ImagesURL(), `/topic/normal_post.gif" alt="" align="middle" /></td>
					<td width="13%">`, c.Txt("manual_index_author"), `</td>
					<td valign="middle" align="left" width="85%" style="padding-left: 6px;">`, c.Txt("manual_index_topic"), `: `, c.Txt("manual_index_topic_subject"), ` &nbsp;(`, c.Txt("manual_index_read_x_times"), `)</td>
				</tr>
			</table>
			<table cellpadding="0" cellspacing="0" border="0" width="100%" class="bordercolor">
				<tr>
					<td style="padding: 1px;">
						<table cellpadding="3" cellspacing="0" border="0" width="100%">
							<tr>
								<td class="windowbg">
									<table width="100%" cellpadding="5" cellspacing="0" style="table-layout: fixed;">
										<tr>
											<td valign="top" width="15%" rowspan="2" style="overflow: hidden;"><b><a href="`, scripturl, `?action=help;page=profile" class="board" title="`, c.Txt("manual_index_view_author_profile"), `">`, c.Txt("manual_index_author"), `</a></b><br />
											<span class="smalltext">`, c.Txt("manual_index_member_group"), `<br />
											`, c.Txt("manual_index_post_group"), `<br />
											<img src="`, c.Theme.ImagesURL(), `/star.gif" alt="*" border="0" /><br />
											`, c.Txt("manual_index_post_count"), `<br />
											<br />
											<br />
											<br />
											<a href="`, scripturl, `?action=help;page=profile" title="`, c.Txt("manual_index_view_profile"), `"><img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" border="0" alt="`, c.Txt("manual_index_view_profile"), `" /></a> <a href="mailto:author@some.address" title="`, c.Txt("manual_index_email"), `"><img src="`, c.Theme.ImagesURL(), `/email_sm.gif" border="0" alt="`, c.Txt("manual_index_email"), `" /></a> <a href="`, scripturl, `?action=help;page=pm" title="`, c.Txt("manual_index_personal_msg"), `"><img src="`, c.Theme.ImagesURL(), `/im_off.gif" border="0" alt="`, c.Txt("manual_index_personal_msg"), `" /></a></span></td>
											<td valign="top" width="85%" height="100%">
												<table width="100%" border="0">
													<tr>
														<td width="20" align="left" valign="middle"><a href="index.php?topic=2.msg2#msg2"><img src="`, c.Theme.ImagesURL(), `/post/xx.gif" alt="" border="0" /></a></td>
														<td align="left" valign="middle">
															<b><a href="`, scripturl, `?action=help;page=index#topic" class="board">`, c.Txt("manual_index_topic_subject"), `</a></b>
															<div class="smalltext">
																&laquo; `, c.Txt("manual_index_post_date_time"), ` &raquo;
															</div>
														</td>
														<td align="right" valign="bottom" height="20" style="font-size: smaller;"><a href="`, scripturl, `?action=help;page=post#quote">`, c.createButton("quote.gif", "manual_index_reply_quote", "smf240", `align="middle"`), `</a></td>
													</tr>
												</table>
												<hr width="100%" size="1" class="hrcolor" />
												<div style="overflow: auto; width: 100%;">
													`, c.Txt("manual_index_topic_text"), `&nbsp;<img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/smiley.gif" border="0" alt="`, c.Txt("manual_index_smiley"), `" />
												</div>
											</td>
										</tr>
										<tr>
											<td valign="bottom" class="smalltext">
												<table width="100%" border="0" style="table-layout: fixed;">
													<tr>
														<td align="right" valign="bottom" class="smalltext"><a href="`, scripturl, `?action=help;page=index#topic" class="board" style="font-size: x-small;">`, c.Txt("manual_index_report_to_mod"), `</a>&nbsp;&nbsp; <img src="`, c.Theme.ImagesURL(), `/ip.gif" alt="" border="0" />&nbsp; `, c.Txt("manual_index_logged"), `</td>
													</tr>
												</table>
											</td>
										</tr>
									</table>
								</td>
							</tr>
						</table>
					</td>
				</tr>
			</table><a name="lastPost" id="lastPost"></a>`)
	if !c.Theme.Empty("use_tabs") {
		c.O(`
			<table width="100%" cellpadding="0" cellspacing="0" border="0">
				<tr>
					<td class="middletext"><b>`, c.Txt("manual_index_pages"), `:</b> [<b>1</b>]</td>
					<td align="right" style="padding-right: 1ex;">
						<table cellpadding="0" cellspacing="0">
							<tr>
								`)
		c.templateButtonStrip(displayButtons, "top")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	} else {
		c.O(`
			<table width="100%" cellpadding="3" cellspacing="0" border="0" class="tborder" style="margin-top: 1ex;">
				<tr>
					<td align="left" class="catbg" width="100%" height="30">
						<table cellpadding="3" cellspacing="0" width="100%">
							<tr>
								<td><b>`, c.Txt("manual_index_pages"), `:</b> [<b>1</b>]</td>
								`)
		c.templateButtonStrip(displayButtons, "top")
		c.O(`
							</tr>
						</table>
					</td>
				</tr>
			</table>`)
	}
	c.O(`
			<div style="padding-top: 4px; padding-bottom: 4px;"></div>
			<div align="right" style="float: right; margin-bottom: 1ex;">
				<form action="`, scripturl, `?action=help;page=index" method="get" accept-charset="`, c.CharacterSet, `">
					<label for="jump2">`, c.Txt("manual_index_jump_to"), `</label>: <select name="jump2" id="jump2" onchange="if (this.options[this.selectedIndex].value) window.location.href='`, scripturl, `?action=help;page=index' + this.options[this.selectedIndex].value;">
						<option value="">
							`, c.Txt("manual_index_destination"), `:
						</option>
						<option value="">
							-----------------------------
						</option>
						<option value="#board">
							`, c.Txt("manual_index_cat_name"), `
						</option>
						<option value="">
							-----------------------------
						</option>
						<option value="#message">
							=&gt; `, c.Txt("manual_index_board_name"), `
						</option>
						<option value="#message">
							=&gt; `, c.Txt("manual_index_another_board"), `
						</option>
					</select>&nbsp; <input type="button" onclick="if (this.form.jump2.options[this.form.jump2.selectedIndex].value) window.location.href = '`, scripturl, `?action=help;page=index' + this.form.jump2.options[this.form.jump2.selectedIndex].value;" value="`, c.Txt("manual_index_go"), `" />
				</form>
			</div><br />
			<br clear="all" />
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_index_navigation_tree"), `</li>
		<li>`, c.Txt("manual_index_prev_next"), `</li>
		<li>`, c.Txt("manual_index_page_no_link"), `</li>
		<li>`, c.Txt("manual_index_reply_button_part1"), `<a href="`, scripturl, `?action=help;page=post#reply">`, c.Txt("manual_index_reply_button_link_posting_reply"), `</a>`, c.Txt("manual_index_reply_button_part2"), `</li>
		<li>`, c.Txt("manual_index_notify_button_enables"), `</li>
		<li>`, c.Txt("manual_index_mark_unread_button"), `</li>
		<li>`, c.Txt("manual_index_send_topic_button"), `</li>
		<li>`, c.Txt("manual_index_print_button"), `</li>
		<li>`, c.Txt("manual_index_author_name_link_part1"), `<a href="`, scripturl, `?action=help;page=profile">`, c.Txt("manual_index_author_name_link_link_profile"), `</a></li>
		<li>`, c.Txt("manual_index_author_details"), `</li>
		<li>`, c.Txt("manual_index_topic_subject_links_start"), `</li>
		<li>`, c.Txt("manual_index_quote_button_part1"), `<a href="`, scripturl, `?action=help;page=post#quote">`, c.Txt("manual_index_quote_button_link_posting_quote"), `</a>`, c.Txt("manual_index_quote_button_part2"), `</li>
		<li>`, c.Txt("manual_index_modify_delete_part1"), `<a href="`, scripturl, `?action=help;page=post#modify">`, c.Txt("manual_index_modify_delete_link_posting_modify"), `</a>`, c.Txt("manual_index_modify_delete_part2"), `</li>
		<li>`, c.Txt("manual_index_report_to_moderator"), `</li>
		<li>`, c.Txt("manual_index_logged_IP"), `</li>
		<li>`, c.Txt("manual_index_jump_to_menu_provides"), `</li>
	</ul>`)
}

// templateManualLogin is template_manual_login().
func templateManualLogin(c *Ctx) {
	scripturl := c.App.ScriptURL
	c.O(`
		<p>`, c.Txt("manual_loginout_complete_reg_part1"), `<a href="`, scripturl, `?action=help;page=registering">`, c.Txt("manual_loginout_complete_reg_link_registering"), `</a>`, c.Txt("manual_loginout_complete_reg_part2"), `</p>
	<ol>
		<li>
			<a href="`, scripturl, `?action=help;page=loginout#login">`, c.Txt("manual_loginout_sec_login"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=loginout#screen">`, c.Txt("manual_loginout_login_screen"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=loginout#quick">`, c.Txt("manual_loginout_sub_quick_login"), `</a></li>
			</ol>
		</li>
		<li><a href="`, scripturl, `?action=help;page=loginout#logout">`, c.Txt("manual_loginout_logout"), `</a></li>
		<li><a href="`, scripturl, `?action=help;page=loginout#reminder">`, c.Txt("manual_loginout_sec_reminder"), `</a></li>
	</ol>
	<h2 id="login">`, c.Txt("manual_loginout_sec_login"), `</h2>
	<p>`, c.Txt("manual_loginout_login_desc"), `</p>
	<h3 id="screen">`, c.Txt("manual_loginout_login_screen"), `</h3>
	<p>`, c.Txt("manual_loginout_login_screen_desc_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_loginout_login_screen_desc_link_index_main"), `</a>`, c.Txt("manual_loginout_login_screen_desc_part2"), `</p>
	<form action="`, scripturl, `?action=help;page=loginout" method="post" accept-charset="`, c.CharacterSet, `" style="margin-top: 4ex;">
		<table border="0" width="400" cellspacing="0" cellpadding="4" class="tborder" align="center">
			<tr class="titlebg">
				<td colspan="2"><img src="`, c.Theme.ImagesURL(), `/icons/login_sm.gif" alt="" align="top" /> `, c.Txt("manual_loginout_login"), `</td>
			</tr>
			<tr class="windowbg">
				<td width="50%" align="right"><b>`, c.Txt("manual_loginout_username"), `:</b></td>
				<td><input type="text" size="20" value="" /></td>
			</tr>
			<tr class="windowbg">
				<td align="right"><b>`, c.Txt("manual_loginout_password"), `:</b></td>
				<td><input type="password" value="" size="20" /></td>
			</tr>
			<tr class="windowbg">
				<td align="right"><b>`, c.Txt("manual_loginout_how_long"), `:</b></td>
				<td><input name="cookielength" type="text" size="4" maxlength="4" value="60" /></td>
			</tr>
			<tr class="windowbg">
				<td align="right"><b>`, c.Txt("manual_loginout_always"), `:</b></td>
				<td><input type="checkbox" class="check" onclick="this.form.cookielength.disabled = this.checked;" /></td>
			</tr>
			<tr class="windowbg">
				<td align="center" colspan="2"><input type="button" style="margin-top: 2ex;" value="Login" /></td>
			</tr>
			<tr class="windowbg">
				<td align="center" colspan="2" class="smalltext"><a href="`, scripturl, `?action=help;page=loginout#reminder" style="font-size: x-small;" class="board">`, c.Txt("manual_loginout_forgot"), `?</a><br />
				<br /></td>
			</tr>
		</table>
	</form><br />
	<p>`, c.Txt("manual_loginout_login_screen_explanation"), `</p>
	<h3 id="quick">`, c.Txt("manual_loginout_sub_quick_login"), `</h3>
	<p>`, c.Txt("manual_loginout_although_many_forums_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_loginout_although_many_forums_link_index_main"), `</a>`, c.Txt("manual_loginout_although_many_forums_part2"), `</p>
	<table cellspacing="0" cellpadding="0" border="0" align="center" width="400" class="tborder">
		<tr>
			<td style="border: solid 1px;">
				<table width="99%" cellpadding="0" cellspacing="5" border="0">
					<tr>
						<td width="100%" valign="top" class="smalltext" style="font-family: verdana, arial, sans-serif;">
							<form action="`, scripturl, `?action=help;page=loginout" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 3px 1ex 1px 0; text-align:right;">
								<input type="text" size="10" /> <input type="password" size="10" /> <select>
									<option>
										`, c.Txt("manual_loginout_hour"), `
									</option>
									<option>
										`, c.Txt("manual_loginout_day"), `
									</option>
									<option>
										`, c.Txt("manual_loginout_week"), `
									</option>
									<option>
										`, c.Txt("manual_loginout_mo"), `
									</option>
									<option selected="selected">
										`, c.Txt("manual_loginout_forever"), `
									</option>
								</select> <input type="button" value="Login" /><br />
								`, c.Txt("manual_loginout_login_all"), `
							</form>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table><br />
	<p>`, c.Txt("manual_loginout_use_quick_login"), `</p>
	<h2 id="logout">`, c.Txt("manual_loginout_logout"), `</h2>
	<p>`, c.Txt("manual_loginout_logout_desc_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_loginout_logout_desc_link_index_main"), `</a>`, c.Txt("manual_loginout_logout_desc_part2"), `</p>
	<h2 id="reminder">`, c.Txt("manual_loginout_sec_reminder"), `</h2>
	<p>`, c.Txt("manual_loginout_reminder_desc_part1"), `<a href="`, scripturl, `?action=help;page=loginout#screen">`, c.Txt("manual_loginout_reminder_desc_link_screen"), `</a>`, c.Txt("manual_loginout_reminder_desc_part2"), `</p>
	<form action="`, scripturl, `?action=help;page=loginout" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" width="400" cellspacing="0" cellpadding="4" align="center" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("manual_loginout_password_reminder"), `</td>
			</tr>
			<tr class="windowbg">
				<td colspan="2" class="smalltext" style="padding: 2ex;">`, c.Txt("manual_loginout_q_explanation"), `</td>
			</tr>
			<tr class="windowbg2">
				<td width="40%">`, c.Txt("manual_loginout_username_email"), `:</td>
				<td><input type="text" name="user" size="30" /></td>
			</tr>
			<tr class="windowbg2">
				<td colspan="2" align="center"><label for="secret"><input type="checkbox" name="sa" value="secret" id="secret" class="check" /> `, c.Txt("manual_loginout_ask_q"), `.</label></td>
			</tr>
			<tr class="windowbg2">
				<td colspan="2" align="center"><input type="button" value="`, c.Txt("manual_loginout_send"), `" /></td>
			</tr>
		</table>
	</form>
	<p>`, c.Txt("manual_loginout_reminder_explanation"), `</p>`)

}

// templateManualPm is template_manual_pm().
func templateManualPm(c *Ctx) {
	scripturl := c.App.ScriptURL
	c.O(`
		<p>`, c.Txt("manual_pm_community"), `</p>
	<ol>
		<li>
			<a href="`, scripturl, `?action=help;page=pm#pm">`, c.Txt("manual_pm_sec_pm"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=pm#description">`, c.Txt("manual_pm_pm_desc"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=pm#reading">`, c.Txt("manual_pm_reading"), `</a></li>
			</ol>
		</li>
		<li>
			<a href="`, scripturl, `?action=help;page=pm#interface">`, c.Txt("manual_pm_sec_pm2"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=pm#starting">`, c.Txt("manual_pm_start_reply"), `</a></li>
			</ol>
		</li>
	</ol>
	<h2 id="pm">`, c.Txt("manual_pm_sec_pm"), `</h2>
	<h3 id="description">`, c.Txt("manual_pm_pm_desc"), `</h3>
	<p>`, c.Txt("manual_pm_pm_desc_1"), `</p>
	<p>`, c.Txt("manual_pm_pm_desc_2"), `</p>
	<p>`, c.Txt("manual_pm_pm_desc_3"), `</p>
	<h3 id="reading">`, c.Txt("manual_pm_reading"), `</h3>
	<p>`, c.Txt("manual_pm_reading_desc_part1"), `<a href="`, scripturl, `?action=help;page=loginout">`, c.Txt("manual_pm_reading_desc_link_loginout"), `</a>`, c.Txt("manual_pm_reading_desc_part2"), `<a href="`, scripturl, `?action=help;page=pm#interface">`, c.Txt("manual_pm_reading_desc_link_loginout_interface"), `</a>`, c.Txt("manual_pm_reading_desc_part3"), `</p>
	<h2 id="interface">`, c.Txt("manual_pm_sec_pm2"), `</h2>
	<p>`, c.Txt("manual_pm_pm_desc2_part1"), `<a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_pm_pm_desc2_link_index_message"), `</a>`, c.Txt("manual_pm_pm_desc2_part2"), `</p>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<script language="JavaScript1.2" type="text/javascript">
//<![CDATA[
			var currentSort = false;
			function sortLastPM()
			{
					document.getElementById("sort-arrow").src = smf_images_url + "/" + (currentSort ? "sort_up.gif" : "sort_down.gif");
					document.getElementById("sort-arrow").alt = "";
					currentSort = !currentSort;
			}
//]]>
</script>
			<form action="`, scripturl, `?action=help;page=pm" method="post" accept-charset="`, c.CharacterSet, `">
				<table border="0" width="100%" cellspacing="0" cellpadding="3">
					<tr>
						<td valign="bottom"><span class="nav"><img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_pm_forum_name"), `</a></b><br />
						<img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=pm#interface" class="nav">`, c.Txt("manual_pm_personal_msgs"), `</a></b><br />
						<img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=pm#interface" class="nav">`, c.Txt("manual_pm_inbox"), `</a></b></span></td>
					</tr>
				</table>
				<table width="100%" border="0" cellpadding="0" cellspacing="0"><tr>
					<td width="125" valign="top">
						<table border="0" cellpadding="4" cellspacing="1" class="bordercolor" width="100">
							<tr>
								<td class="catbg">`, c.Txt("manual_pm_messages"), `</td>
							</tr>
							<tr class="windowbg">
								<td class="smalltext" style="padding-bottom: 2ex;">
								`, c.Txt("manual_pm_new_msg"), `<br /><br />
								<b>`, c.Txt("manual_pm_inbox"), `</b><br />
								`, c.Txt("manual_pm_outbox"), `<br />
							</td>
						</tr>
					</table>
					<br />
				</td>
				<td valign="top">
					<table cellpadding="0" cellspacing="0" border="0" width="100%" class="bordercolor" align="center">
						<tr>
							<td>
								<table border="0" width="100%" cellspacing="1" class="bordercolor">
									<tr class="titlebg">
										<td>&nbsp;</td>
										<td style="width: 32ex;"><a href="javascript:sortLastPM();">`, c.Txt("manual_pm_date"), `&nbsp; <img id="sort-arrow" src="`, c.Theme.ImagesURL(), `/sort_up.gif" alt="" border="0" name="sort-arrow" /></a></td>
										<td width="46%"><a href="`, scripturl, `?action=help;page=pm#interface">`, c.Txt("manual_pm_subject2"), `</a></td>
										<td><a href="`, scripturl, `?action=help;page=pm#interface">`, c.Txt("manual_pm_from"), `</a></td>
										<td align="center" width="24"><input type="checkbox" onclick="invertAll(this, this.form);" class="check" /></td>
									</tr>
									<tr class="windowbg">
										<td align="center" width="2%"><img src="`, c.Theme.ImagesURL(), `/icons/pm_read.gif" style="margin-right: 4px;" alt="" /></td>
										<td>`, c.Txt("manual_pm_date_and_time"), `</td>
										<td><a href="`, scripturl, `?action=help;page=pm#interface" class="board">`, c.Txt("manual_pm_subject"), `</a></td>
										<td>`, c.Txt("manual_pm_another_member"), `</td>
										<td align="center"><input type="checkbox" class="check" /></td>
									</tr>
									<tr>
										<td class="windowbg" style="padding: 2px;" align="right" colspan="6"></td>
									</tr>
									<tr>
										<td colspan="6" class="catbg" height="25">
											<div style="float: left;"><b>`, c.Txt("manual_pm_pages"), `:</b> [<b>1</b>]</div>
											<div style="float: right;">&nbsp;<input type="button" value="`, c.Txt("manual_pm_delete_selected"), `" /></div>
										</td>
									</tr>
								</table>
							</td>
						</tr>
					</table><br />
				</td>
			</tr></table>
			<br />
			</form>
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_pm_nav_tree"), `</li>
		<li>`, c.Txt("manual_pm_delete_button"), `</li>
		<li>`, c.Txt("manual_pm_outbox_button"), `</li>
		<li>`, c.Txt("manual_pm_new_msg2_part1"), `<a href="`, scripturl, `?action=help;page=post#newtopic">`, c.Txt("manual_pm_new_msg2_link_posting_newtopic"), `</a>`, c.Txt("manual_pm_new_msg2_part2"), `</li>
		<li>`, c.Txt("manual_pm_reload"), `</li>
		<li>`, c.Txt("manual_pm_sort_by"), `</li>
		<li>`, c.Txt("manual_pm_main_subject"), `</li>
		<li>`, c.Txt("manual_pm_page_nos"), `</li>
	</ul>
	<h3 id="starting">`, c.Txt("manual_pm_start_reply"), `</h3>
	<p>`, c.Txt("manual_pm_how_to_start_reply_part1"), `<a href="`, scripturl, `?action=help;page=loginout">`, c.Txt("manual_pm_how_to_start_reply_link_loginout"), `</a>`, c.Txt("manual_pm_how_to_start_reply_part2"), `</p>
	<ul>
		<li>`, c.Txt("manual_pm_msg_link_part1"), `<a href="`, scripturl, `?action=help;page=pm#interface">`, c.Txt("manual_pm_msg_link_link_interface"), `</a>`, c.Txt("manual_pm_msg_link_part2"), `</li>
		<li>`, c.Txt("manual_pm_click_name_part1"), `<a href="`, scripturl, `?action=help;page=profile#info-all">`, c.Txt("manual_pm_click_name_link_profile_info-all"), `</a>`, c.Txt("manual_pm_click_name_part2"), `</li>
		<li>`, c.Txt("manual_pm_click_im_icon"), `</li>
		<li>`, c.Txt("manual_pm_click_pm_icon_part1"), `<a href="`, scripturl, `?action=help;page=profile#info-all">`, c.Txt("manual_pm_click_pm_icon_link_profile_info-all"), `</a>`, c.Txt("manual_pm_click_pm_icon_part2"), `</li>
		<li>`, c.Txt("manual_pm_reply_msg_part1"), `<a href="`, scripturl, `?action=help;page=post#reply">`, c.Txt("manual_pm_reply_msg_link_posting_reply"), `</a>`, c.Txt("manual_pm_reply_msg_part2"), `</li>
	</ul>`)
}

// templateManualProfile is template_manual_profile().
func templateManualProfile(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`
	<p>`, c.Txt("manual_profile_profile_screen"), `</p>
	<p>`, c.Txt("manual_profile_edit_profile_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_profile_edit_profile_link_index_main"), `</a>`, c.Txt("manual_profile_edit_profile_part2"), `</p>
	<ol>
		<li>
			<a href="`, scripturl, `?action=help;page=profile#all">`, c.Txt("manual_profile_available_to_all"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=profile#info-all">`, c.Txt("manual_profile_profile_info"), `</a></li>
			</ol>
		</li>
				<li>
			<a href="`, scripturl, `?action=help;page=profile#owners">`, c.Txt("manual_profile_sec_normal"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=profile#edit-owners">`, c.Txt("manual_profile_modify_profile"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=profile#actions-owners">`, c.Txt("manual_profile_actions"), `</a></li>
			</ol>
		</li>
		<li>
			<a href="`, scripturl, `?action=help;page=profile#admins">`, c.Txt("manual_profile_sec_settings"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=profile#info-admins">`, c.Txt("manual_profile_profile_info"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=profile#edit-admins">`, c.Txt("manual_profile_modify_profile"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=profile#actions-admins">`, c.Txt("manual_profile_actions"), `</a></li>
			</ol>
		</li>
	</ol>
	<h2 id="all">`, c.Txt("manual_profile_available_to_all"), `</h2>
	<h3 id="info-all">`, c.Txt("manual_profile_profile_info"), `</h3>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<table width="100%" border="0" cellpadding="0" cellspacing="0" style="padding-top: 1ex;">
				<tr>
					<td width="100%" valign="top">
						<table border="0" cellpadding="4" cellspacing="1" align="center" class="bordercolor">
							<tr class="titlebg">
								<td align="left" width="420" height="26"><img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" border="0" align="top" />&nbsp; `, c.Txt("manual_profile_username"), `:&nbsp;`, c.Txt("manual_profile_login_name"), `</td>
								<td align="center" width="150">`, c.Txt("manual_profile_pic_text"), `</td>
							</tr>
							<tr>
								<td class="windowbg" width="420" align="left">
									<table border="0" cellspacing="0" cellpadding="2" width="100%">
										<tr>
											<td><b>`, c.Txt("manual_profile_name"), `:</b></td>
											<td>`, c.Txt("manual_profile_screen_name"), `</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_posts"), `:</b></td>
											<td>`, c.Txt("manual_profile_member_posts"), `</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_position"), `:</b></td>
											<td>`, c.Txt("manual_profile_membergroup"), `</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_date_reg"), `:</b></td>
											<td>`, c.Txt("manual_profile_date_time_reg"), `</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_last_active"), `:</b></td>
											<td>`, c.Txt("manual_profile_date_time_active"), `</td>
										</tr>
										<tr>
											<td colspan="2">
												<hr size="1" width="100%" class="hrcolor" />
											</td>
										</tr>
										<tr>
											<td><b>ICQ:</b></td>
											<td></td>
										</tr>
										<tr>
											<td><b>AIM:</b></td>
											<td></td>
										</tr>
										<tr>
											<td><b>MSN:</b></td>
											<td></td>
										</tr>
										<tr>
											<td><b>YIM:</b></td>
											<td></td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_email"), `:</b></td>
											<td><a href="mailto:`, c.Txt("manual_profile_email_user"), `" class="board">`, c.Txt("manual_profile_email_user"), `</a></td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_website"), `:</b></td>
											<td><a href="http://www.simplemachines.org/" target="_blank"></a></td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_status"), `:</b></td>
											<td><i><a href="`, scripturl, `?action=help;page=pm" title="`, c.Txt("manual_profile_pm"), ` (`, c.Txt("manual_profile_online"), `)  "><img src="`, c.Theme.ImagesURL(), `/useron.gif" border="0" align="middle" alt="`, c.Txt("manual_profile_online"), `" /></a> <span class="smalltext">`, c.Txt("manual_profile_online"), `</span></i></td>
										</tr>
										<tr>
											<td colspan="2">
												<hr size="1" width="100%" class="hrcolor" />
											</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_gender"), `:</b></td>
											<td></td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_age"), `:</b></td>
											<td>`, c.Txt("manual_profile_n_a"), `</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_location"), `:</b></td>
											<td></td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_local_time"), `:</b></td>
											<td>`, c.Txt("manual_profile_current_date_time"), `</td>
										</tr>
										<tr>
											<td><b>`, c.Txt("manual_profile_language"), `:</b></td>
											<td></td>
										</tr>
										<tr>
											<td colspan="2">
												<hr size="1" width="100%" class="hrcolor" />
											</td>
										</tr>
										<tr>
											<td colspan="2" height="25">
												<table border="0">
													<tr>
														<td><b>`, c.Txt("manual_profile_sig"), `:</b></td>
													</tr>
													<tr>
														<td colspan="2"></td>
													</tr>
												</table>
											</td>
										</tr>
									</table>
								</td>
								<td class="windowbg" valign="middle" align="center" width="150"><br />
								<br /></td>
							</tr>
							<tr class="titlebg">
								<td colspan="2" align="left">`, c.Txt("manual_profile_other_info"), `:</td>
							</tr>
							<tr>
								<td class="windowbg2" colspan="2" align="left"><a href="`, scripturl, `?action=help;page=profile#all" class="board">`, c.Txt("manual_profile_send_pm"), `</a><br />
								<br />
								<a href="`, scripturl, `?action=help;page=profile#all" class="board">`, c.Txt("manual_profile_show_member_posts"), `</a><br />
								<a href="`, scripturl, `?action=help;page=profile#all" class="board">`, c.Txt("manual_profile_show_member_stats"), `</a><br />
								<br /></td>
							</tr>
						</table>
					</td>
				</tr>
			</table><br />
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_profile_summary_part1"), `<a href="`, scripturl, `?action=help;page=profile#owners">`, c.Txt("manual_profile_summary_link_owners"), `</a>`, c.Txt("manual_profile_summary_part2"), `</li>
		<li>`, c.Txt("manual_profile_hide_email"), `</li>
		<li>`, c.Txt("manual_profile_empty_part1"), `<a href="`, scripturl, `?action=help;page=profile#owners">`, c.Txt("manual_profile_empty_link_owners"), `</a>`, c.Txt("manual_profile_empty_part2"), `</li>
		<li>`, c.Txt("manual_profile_send_member_pm_part1"), `<a href="`, scripturl, `?action=help;page=pm">`, c.Txt("manual_profile_send_member_pm_link_pm"), `</a>`, c.Txt("manual_profile_send_member_pm_part2"), `</li>
		<li>`, c.Txt("manual_profile_show_last_posts"), `</li>
		<li>`, c.Txt("manual_profile_show_member_stats2"), `</li>
	</ul>
	<h2 id="owners">`, c.Txt("manual_profile_sec_normal"), `</h2>
	<p>`, c.Txt("manual_profile_normal_desc"), `</p>
	<h3 id="edit-owners">`, c.Txt("manual_profile_modify_profile"), `</h3>
	<ul>
		<li>`, c.Txt("manual_profile_account_related"), `</li>
		<li>`, c.Txt("manual_profile_forum_profile_info"), `</li>
		<li>`, c.Txt("manual_profile_look_layout"), `</li>
	</ul>
		<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<table width="100%" border="0" cellpadding="0" cellspacing="0" style="padding-top: 1ex;">
				<tr>
					<td width="180" valign="top">
						<table border="0" cellpadding="4" cellspacing="1" class="bordercolor" width="170">
							<tr>
								<td class="catbg">`, c.Txt("manual_profile_profile_info2"), `</td>
							</tr>
							<tr class="windowbg2">
								<td class="smalltext"><a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_summary2"), `</a><br />
								<a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_show_stats"), `</a><br />
								<a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_show_posts"), `</a><br />
								<br /></td>
							</tr>
							<tr>
								<td class="catbg">`, c.Txt("manual_profile_modify_own_profile"), `</td>
							</tr>
							<tr class="windowbg2">
								<td class="smalltext"><a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_acct_settings"), `</a><br />
								<a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_forum_profile"), `</a><br />
								<b><a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_look_and_layout"), `</a></b><br />
								<a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_notify_email"), `</a><br />
								<a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_pm_options1"), `</a><br />
								<br /></td>
							</tr>
							<tr>
								<td class="catbg">`, c.Txt("manual_profile_actions"), `</td>
							</tr>
							<tr class="windowbg2">
								<td class="smalltext"><a href="`, scripturl, `?action=help;page=profile#owners" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_delete_account"), `</a><br />
								<br /></td>
							</tr>
						</table>
					</td>
					<td width="100%" valign="top">
						<form action="`, scripturl, `?action=help;page=profile" method="post" accept-charset="`, c.CharacterSet, `">
							<table border="0" width="85%" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
								<tr class="titlebg">
									<td height="26" align="left">&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" border="0" align="top" />&nbsp; `, c.Txt("manual_profile_edit_profile1"), `</td>
								</tr>
								<tr>
									<td class="windowbg" height="25" align="left"><span class="smalltext"><br />
									`, c.Txt("manual_profile_look_layout_explanation"), `<br />
									<br /></span></td>
								</tr>
								<tr>
									<td class="windowbg2" align="left">
										<table border="0" width="100%" cellpadding="3">
											<tr>
												<td colspan="2" width="40%"><b>`, c.Txt("manual_profile_current_theme"), `:</b>&nbsp;`, c.Txt("manual_profile_board_default"), `&nbsp;<a href="`, scripturl, `?action=help;page=profile#owners" class="board">(`, c.Txt("manual_profile_change"), `)</a></td>
											</tr>
											<tr>
												<td colspan="2">
													<hr width="100%" size="1" class="hrcolor" />
												</td>
											</tr>
											<tr>
												<td width="40%"><b>`, c.Txt("manual_profile_time_format"), `:</b><br />
												<a href="`, scripturl, `/index.php?action=helpadmin;help=time_format" onclick="return reqWin(this.href);" class="help"><img src="`, c.Theme.ImagesURL(), `/helptopics.gif" alt="`, c.Txt("manual_profile_help"), `" border="0" align="left" style="padding-right: 1ex;" /></a> <span class="smalltext">`, c.Txt("manual_profile_caption_date"), `</span></td>
												<td><select style="margin-bottom: 4px;">
													<option selected="selected">
														(`, c.Txt("manual_profile_date_option_select"), `)
													</option>
													<option>
														`, c.Txt("manual_profile_date_option_1"), `
													</option>
													<option>
														`, c.Txt("manual_profile_date_option_2"), `
													</option>
													<option>
														`, c.Txt("manual_profile_date_option_3"), `
													</option>
													<option>
														`, c.Txt("manual_profile_date_option_4"), `
													</option>
													<option>
														`, c.Txt("manual_profile_date_option_5"), `
													</option>
												</select><br />
												<input type="text" value="" size="30" /></td>
											</tr>
											<tr>
												<td width="40%">
													<b>`, c.Txt("manual_profile_time_offset"), `:</b>
													<div class="smalltext">
														`, c.Txt("manual_profile_offset_hours"), `
													</div>
												</td>
												<td class="smalltext"><input type="text" size="5" maxlength="5" value="0" /><br />
												<em>(`, c.Txt("manual_profile_forum_time"), `)</em></td>
											</tr>
											<tr>
												<td colspan="2">
													<hr width="100%" size="1" class="hrcolor" />
												</td>
											</tr>
											<tr>
												<td colspan="2">
													<br />
													<table width="100%" cellspacing="0" cellpadding="3">
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_board_descriptions"), `</td>
														</tr>
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_show_child"), `</td>
														</tr>
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_no_ava"), `</td>
														</tr>
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_no_sig"), `</td>
														</tr>
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_return_to_topic"), `</td>
														</tr>
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_recent_posts"), `</td>
														</tr>
														<tr>
															<td width="28"><input type="checkbox" class="check" /></td>
															<td>`, c.Txt("manual_profile_recent_pms"), `</td>
														</tr>
														<tr>
															<td colspan="2">`, c.Txt("manual_profile_first_day_week"), `
															<select>
																<option selected="selected">
																	`, c.Txt("manual_profile_sun"), `
																</option>
																<option>
																	`, c.Txt("manual_profile_mon"), `
																</option>
															</select></td>
														</tr>
														<tr>
															<td colspan="2">`, c.Txt("manual_profile_quick_reply"), `: <select>
																<option selected="selected">
																	`, c.Txt("manual_profile_not_at_all"), `
																</option>
																<option>
																	`, c.Txt("manual_profile_off_default"), `
																</option>
																<option>
																	`, c.Txt("manual_profile_on_default"), `
																</option>
															</select></td>
														</tr>
														<tr>
															<td colspan="2">`, c.Txt("manual_profile_quick_mod"), `&nbsp; <select>
																<option selected="selected">
																	`, c.Txt("manual_profile_no_quick_mod"), `.
																</option>
																<option>
																	`, c.Txt("manual_profile_check_quick_mod"), `.
																</option>
																<option>
																	`, c.Txt("manual_profile_icon_quick_mod"), `.
																</option>
															</select></td>
														</tr>
													</table>
												</td>
											</tr>
											<tr>
												<td colspan="2">
													<hr width="100%" size="1" class="hrcolor" />
												</td>
											</tr>
											<tr>
												<td align="right" colspan="2"><input type="button" value="`, c.Txt("manual_profile_change_profile"), `" /></td>
											</tr>
										</table><br />
									</td>
								</tr>
							</table>
						</form>
					</td>
				</tr>
			</table><br />
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_profile_notify_email_prefs"), `</li>
		<li>`, c.Txt("manual_profile_pm_options_part1"), `<a href="`, scripturl, `?action=help;page=pm">`, c.Txt("manual_profile_pm_options_link_pm"), `</a>`, c.Txt("manual_profile_pm_options_part2"), `</li>
	</ul>
	<h3 id="actions-owners">`, c.Txt("manual_profile_sub_actions"), `</h3>
	<ul>
		<li>`, c.Txt("manual_profile_confirm_delete_acct"), `</li>
	</ul>
	<h2 id="admins">`, c.Txt("manual_profile_sec_settings"), `</h2>
	<p>`, c.Txt("manual_profile_settings_desc"), `</p>
		<div>
		<div style="width: 180px; float: left; border: none;">
			<table border="0" cellpadding="4" cellspacing="1" class="bordercolor" width="170">
				<tr>
					<td class="catbg">`, c.Txt("manual_profile_profile_info"), `</td>
				</tr>
				<tr class="windowbg2">
					<td class="windowbg2"><b><a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_summary2"), `</a></b><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_show_stats"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_show_posts"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_track_user"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_track_ip"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_show_permissions"), `</a><br />
					<br /></td>
				</tr>
				<tr>
					<td class="catbg">`, c.Txt("manual_profile_sub_modify_profile"), `</td>
				</tr>
				<tr class="windowbg2">
					<td class="windowbg2"><a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_acct_settings"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_forum_profile"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_look_and_layout"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_notify_email"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_pm_options1"), `</a><br />
					<br /></td>
				</tr>
				<tr>
					<td class="catbg">`, c.Txt("manual_profile_actions"), `</td>
				</tr>
				<tr class="windowbg2">
					<td class="windowbg2"><a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_ban_user"), `</a><br />
					<a href="`, scripturl, `?action=help;page=profile#admins" style="font-size: x-small;" class="board">`, c.Txt("manual_profile_delete_account"), `</a><br />
					<br /></td>
				</tr>
			</table>
		</div><br />
		<div style="margin: -1.8em 20px 0 200px;">
			<h3 id="info-admins">`, c.Txt("manual_profile_sub_profile_info"), `</h3>
			<ul>
				<li>`, c.Txt("manual_profile_sub_track_user"), `</li>
				<li>`, c.Txt("manual_profile_sub_track_ip"), `</li>
				<li>`, c.Txt("manual_profile_sub_show_permissions"), `</li>
			</ul>
			<h3 id="edit-admins">`, c.Txt("manual_profile_sub_modify_profile"), `</h3>
			<ul>
				<li>`, c.Txt("manual_profile_sub_acct_settings"), `</li>
				<li>`, c.Txt("manual_profile_sub_forum_profile_info"), `</li>
			</ul>
			<h3 id="actions-admins">`, c.Txt("manual_profile_sub_actions2"), `</h3>
			<ul>
				<li>`, c.Txt("manual_profile_sub_ban_user"), `</li>
				<li>`, c.Txt("manual_profile_sub_delete_acct"), `</li>
			</ul>
		</div>
	</div><br clear="all" />`)

}

// templateManualRegister is template_manual_register().
func templateManualRegister(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`
		<p>`, c.Txt("manual_registering_you_have_arrived_part1"), `<a href="`, scripturl, `?action=help;page=profile">`, c.Txt("manual_registering_you_have_arrived_link_profile"), `</a>`, c.Txt("manual_registering_you_have_arrived_part2"), `<a href="`, scripturl, `?action=help;page=pm">`, c.Txt("manual_registering_you_have_arrived_link_profile_pm"), `</a>`, c.Txt("manual_registering_you_have_arrived_part3"), `</p>
	<ol>
		<li><a href="`, scripturl, `?action=help;page=registering#how-to">`, c.Txt("manual_registering_sec_register"), `</a></li>
		<li><a href="`, scripturl, `?action=help;page=registering#screen">`, c.Txt("manual_registering_sec_reg_screen"), `</a></li>
	</ol>
	<h2 id="how-to">`, c.Txt("manual_registering_sec_register"), `</h2>
	<p>`, c.Txt("manual_registering_register_desc"), `</p>
	<ul>
		<li>`, c.Txt("manual_registering_select_register_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_registering_select_register_link_index_main"), `</a>`, c.Txt("manual_registering_select_register_part2"), `</li>
		<li>`, c.Txt("manual_registering_login_Scr_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_registering_login_Scr_link_index_main"), `</a>`, c.Txt("manual_registering_login_Scr_part2"), `</li>
	</ul>
	<table width="400" cellspacing="0" cellpadding="3" class="tborder" align="center">
		<tr class="titlebg">
			<td>`, c.Txt("manual_registering_warning"), `</td>
		</tr>
		<tr>
			<td class="windowbg" style="padding-top: 2ex; padding-bottom: 2ex;">`, c.Txt("manual_registering_warning_desc_1"), `<br />
			`, c.Txt("manual_registering_warning_desc_2"), `<a href="`, scripturl, `?action=help;page=registering#screen" class="board">`, c.Txt("manual_registering_warning_desc_3"), `</a>`, c.Txt("manual_registering_warning_desc_4"), `</td>
		</tr>
	</table><br />
	<h2 id="screen">`, c.Txt("manual_registering_sec_reg_screen"), `</h2>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<form action="`, scripturl, `?action=help;page=registering" method="post" accept-charset="`, c.CharacterSet, `">
				<table border="0" width="100%" cellpadding="3" cellspacing="0" class="tborder">
					<tr class="titlebg">
						<td>`, c.Txt("manual_registering_required_info"), `</td>
					</tr>
					<tr class="windowbg">
						<td width="100%">
							<table cellpadding="3" cellspacing="0" border="0" width="100%">
								<tr>
									<td width="40%">
										<b>`, c.Txt("manual_registering_choose_username"), `:</b>
										<div class="smalltext">
											`, c.Txt("manual_registering_caption_username"), `
										</div>
									</td>
									<td><input type="text" size="20" maxlength="18" /></td>
								</tr>
								<tr>
									<td width="40%">
										<b>`, c.Txt("manual_registering_email"), `:</b>
										<div class="smalltext">
											`, c.Txt("manual_registering_caption_email"), `
										</div>
									</td>
									<td><input type="text" size="30" /> <input type="checkbox" class="check" /> <label>`, c.Txt("manual_registering_hide_email"), `</label></td>
								</tr>
								<tr>
									<td width="40%"><b>`, c.Txt("manual_registering_choose_pass"), `:</b></td>
									<td><input type="password" size="30" /></td>
								</tr>
								<tr>
									<td width="40%"><b>`, c.Txt("manual_registering_verify_pass"), `:</b></td>
									<td><input type="password" size="30" /></td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
				<table width="100%" align="center" border="0" cellspacing="0" cellpadding="5" class="tborder" style="border-top: 0;">
					<tr>
						<td class="windowbg2" style="padding-top: 8px; padding-bottom: 8px;">`, c.Txt("manual_registering_agreement"), `</td>
					</tr>
					<tr>
						<td align="center" class="windowbg2"><label><input type="checkbox" class="check" /> <b>`, c.Txt("manual_registering_agree"), `</b></label></td>
					</tr>
				</table><br />
				<div align="center">
					<input type="button" value="`, c.Txt("manual_registering_register"), `" />
				</div>
			</form>
		</div>
	</div><br />
	<p>`, c.Txt("manual_registering_reg_screen_requirements_part1"), `<a href="`, scripturl, `?action=help;page=loginout#screen">`, c.Txt("manual_registering_reg_screen_requirements_link_loginout_screen"), `</a>`, c.Txt("manual_registering_reg_screen_requirements_part2"), `</p>
	<ul>
		<li>`, c.Txt("manual_registering_email_activate"), `</li>
		<li>`, c.Txt("manual_registering_admin_approve"), `</li>
	</ul>`)
}

// templateManualSearch is template_manual_search().
func templateManualSearch(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`
	<p>`, c.Txt("manual_searching_you_have_arrived"), `</p>
	<ol>
		<li><a href="`, scripturl, `?action=help;page=searching#starting">`, c.Txt("manual_searching_sec_search"), `</a></li>
		<li>
			<a href="`, scripturl, `?action=help;page=searching#syntax">`, c.Txt("manual_searching_sec_syntax"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=searching#quotes">`, c.Txt("manual_searching_sub_quotes"), `</a></li>
			</ol>
		</li>
		<li>
			<a href="`, scripturl, `?action=help;page=searching#searching">`, c.Txt("manual_searching_sec_simple_adv"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=searching#simple">`, c.Txt("manual_searching_sub_simple"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=searching#advanced">`, c.Txt("manual_searching_sub_adv"), `</a></li>
			</ol>
		</li>
	</ol>
	<h2 id="starting">`, c.Txt("manual_searching_sec_search"), `</h2>
	<p>`, c.Txt("manual_searching_search_desc_part1"), `<a href="`, scripturl, `?action=help;page=index#main">`, c.Txt("manual_searching_search_desc_link_index_main"), `</a>`, c.Txt("manual_searching_search_desc_part2"), `</p>
	<h2 id="syntax">`, c.Txt("manual_searching_sec_syntax"), `</h2>
	<p>`, c.Txt("manual_searching_syntax_desc"), `</p>
	<h3 id="quotes">`, c.Txt("manual_searching_sub_quotes"), `</h3>
	<p>`, c.Txt("manual_searching_quotes_desc"), `</p>
	<h2 id="searching">`, c.Txt("manual_searching_sec_simple_adv"), `</h2>
	<h3 id="simple">`, c.Txt("manual_searching_sub_simple"), `</h3>
	<p>`, c.Txt("manual_searching_simple_desc"), `</p>
	<h3 id="advanced">`, c.Txt("manual_searching_sub_adv"), `</h3>
	<p>`, c.Txt("manual_searching_adv_desc"), `</p>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<form action="`, scripturl, `?action=help;page=searching" method="post" accept-charset="`, c.CharacterSet, `">
				<table width="80%" border="0" cellspacing="0" cellpadding="3" align="center">
					<tr>
						<td><span class="nav"><img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_searching_forum_name"), `</a></b><br />
						<img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=searching#advanced" class="nav">`, c.Txt("manual_searching_search"), `</a></b></span></td>
					</tr>
				</table>
				<table width="80%" border="0" cellspacing="0" cellpadding="4" align="center" class="tborder">
					<tr class="titlebg">
						<td>`, c.Txt("manual_searching_search_param"), `</td>
					</tr>
					<tr>
						<td class="windowbg">
							<table>
								<tr>
									<td><b>`, c.Txt("manual_searching_search_for"), `:</b></td>
									<td>&nbsp;</td>
									<td><b>`, c.Txt("manual_searching_by_user"), `:</b></td>
								</tr>
								<tr>
									<td><input type="text" size="40" /></td>
									<td><select>
										<option selected="selected">
											`, c.Txt("manual_searching_match_all"), `
										</option>
										<option>
											`, c.Txt("manual_searching_match_any"), `
										</option>
									</select>&nbsp;&nbsp;&nbsp;</td>
									<td><input type="text" value="*" size="40" />&nbsp;</td>
								</tr>
								<tr>
									<td colspan="3">&nbsp;</td>
								</tr>
																<tr>
									<td colspan="2"><b>`, c.Txt("manual_searching_options"), `:</b></td>
									<td><b>`, c.Txt("manual_searching_msg_age"), `:</b></td>
								</tr>
								<tr>
									<td colspan="2"><input type="checkbox" class="check" /> <label>`, c.Txt("manual_searching_show_results"), `</label><br />
									<input type="checkbox" class="check" /> <label>`, c.Txt("manual_searching_subject_only"), `</label><br /></td>
									<td>`, c.Txt("manual_searching_between"), `<input type="text" value="0" size="5" maxlength="5" />`, c.Txt("manual_searching_and"), `<input type="text" value="9999" size="5" maxlength="5" />`, c.Txt("manual_searching_days"), `.</td>
								</tr>
								<tr>
									<td colspan="3" style="padding-top: 2ex;"><b>`, c.Txt("manual_searching_search_order"), `:</b></td>
								</tr>
								<tr>
									<td colspan="3"><select>
										<option selected="selected">
											`, c.Txt("manual_searching_relevant_first"), `
										</option>
										<option>
											`, c.Txt("manual_searching_big_first"), `
										</option>
										<option>
											`, c.Txt("manual_searching_small_first"), `
										</option>
										<option>
											`, c.Txt("manual_searching_recent_first"), `
										</option>
										<option>
											`, c.Txt("manual_searching_oldest_first"), `
										</option>
									</select></td>
								</tr>
							</table><br />
							<b>`, c.Txt("manual_searching_choose"), `:</b><br />
							<br />
							<table width="80%" border="0" cellpadding="1" cellspacing="0">
								<tr>
									<td width="50%"><span style="text-decoration: underline;">`, c.Txt("manual_searching_cat"), `</span></td>
									<td width="50%"><input type="checkbox" id="brd2" name="brd[2]" value="2" checked="checked" class="check" /> <label for="brd2">`, c.Txt("manual_searching_another_board"), `</label></td>
								</tr>
								<tr>
									<td width="50%"><input type="checkbox" id="brd1" name="brd[1]" value="1" checked="checked" class="check" /> <label for="brd1">`, c.Txt("manual_searching_board_name"), `</label></td>
								</tr>
							</table><br />
							<input type="checkbox" name="all" id="check_all" value="" checked="checked" onclick="invertAll(this, this.form, 'brd');" class="check" /><i><label for="check_all">`, c.Txt("manual_searching_check_all"), `</label></i><br />
							<br />
							<table border="0" cellpadding="2" cellspacing="0" align="left">
								<tr>
									<td valign="bottom"><input type="button" value="`, c.Txt("manual_searching_search"), `" /></td>
								</tr>
							</table>
						</td>
					</tr>
				</table><br />
			</form>
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_searching_nav_tree"), `</li>
		<li>`, c.Txt("manual_searching_three_options_part1"), `<a href="`, scripturl, `?action=help;page=searching#syntax">`, c.Txt("manual_searching_three_options_link_syntax"), `</a>`, c.Txt("manual_searching_three_options_part2"), `</li>
		<li>`, c.Txt("manual_searching_wildcard"), `</li>
		<li>`, c.Txt("manual_searching_results_as_messages"), `</li>
		<li>`, c.Txt("manual_searching_message_age"), `</li>
		<li>`, c.Txt("manual_searching_which_board"), `</li>
		<li>`, c.Txt("manual_searching_search_button"), `</li>
	</ul>`)
}
