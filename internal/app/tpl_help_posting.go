package app

// templateManualPosting is template_manual_posting().
func templateManualPosting(c *Ctx) {
	scripturl := c.App.ScriptURL
	c.O(`
	<p>`, c.Txt("manual_posting_forum_about_part1"), `<a href="`, scripturl, `?action=help;page=post#bbcref">`, c.Txt("manual_posting_forum_about_link_bbcref"), `</a>`, c.Txt("manual_posting_forum_about_part2"), `<a href="`, scripturl, `?action=help;page=post#smileysref">`, c.Txt("manual_posting_forum_about_link_bbcref_smileysref"), `</a>`, c.Txt("manual_posting_forum_about_part3"), `</p>
	<p>`, c.Txt("manual_posting_please_note"), `</p>
	<ol>
		<li>
			<a href="`, scripturl, `?action=help;page=post#basics">`, c.Txt("manual_posting_sec_posting_basics"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=post#newtopic">`, c.Txt("manual_posting_starting_topic"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#newpoll">`, c.Txt("manual_posting_start_poll"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#reply">`, c.Txt("manual_posting_replying"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#quote">`, c.Txt("manual_posting_quote_post"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#modify">`, c.Txt("manual_posting_modify_delete"), `</a></li>
			</ol>
		</li>
		<li>
			<a href="`, scripturl, `?action=help;page=post#standard">`, c.Txt("manual_posting_sec_posting_options"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=post#messageicon">`, c.Txt("manual_posting_sub_message_icon"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#bbc">`, c.Txt("manual_posting_sub_bbc"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#smileys">`, c.Txt("manual_posting_sub_smileys"), `</a></li>
			</ol>
		</li>
		<li><a href="`, scripturl, `?action=help;page=post#tags">`, c.Txt("manual_posting_sec_tags"), `</a></li>
		<li>
			<a href="`, scripturl, `?action=help;page=post#additional">`, c.Txt("manual_posting_sec_additional_options"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=post#notify">`, c.Txt("manual_posting_notify"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#return">`, c.Txt("manual_posting_return"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#nosmileys">`, c.Txt("manual_posting_no_smiley"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#attachments">`, c.Txt("manual_posting_sub_attach"), `</a></li>
			</ol>
		</li>
		<li>
			<a href="`, scripturl, `?action=help;page=post#references">`, c.Txt("manual_posting_sec_references"), `</a>
			<ol class="la">
				<li><a href="`, scripturl, `?action=help;page=post#bbcref">`, c.Txt("manual_posting_sub_SMF_bbc"), `</a></li>
				<li><a href="`, scripturl, `?action=help;page=post#smileysref">`, c.Txt("manual_posting_sub_help_smileys"), `</a></li>
			</ol>
		</li>
	</ol>
	<h2 id="basics">`, c.Txt("manual_posting_sec_posting_basics"), `</h2>
	<h3 id="newtopic">`, c.Txt("manual_posting_starting_topic"), `</h3>
	<p>`, c.Txt("manual_posting_starting_topic_desc_part1"), `<a href="`, scripturl, `?action=help;page=index#message">`, c.Txt("manual_posting_starting_topic_desc_link_index_message"), `</a>`, c.Txt("manual_posting_starting_topic_desc_part2"), `<a href="`, scripturl, `?action=help;page=post#standard">`, c.Txt("manual_posting_starting_topic_desc_link_index_message_standard"), `</a>`, c.Txt("manual_posting_starting_topic_desc_part3"), `</p>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<form action="`, scripturl, `?action=help;page=post" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
				<table width="100%" align="center" cellpadding="0" cellspacing="3">
					<tr>
						<td valign="bottom" colspan="2"><span class="nav"><img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_posting_forum_name"), `</a></b><br />
						<img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#board" class="nav">`, c.Txt("manual_posting_cat_name"), `</a></b><br />
						<img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><a href="`, scripturl, `?action=help;page=index#message" class="nav">`, c.Txt("manual_posting_board_name"), `</a></b><br />
						<img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_main.gif" alt="| " border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/linktree_side.gif" alt="|-" border="0" /> <img src="`, c.Theme.ImagesURL(), `/icons/folder_open.gif" alt="+" border="0" />&nbsp; <b><i>`, c.Txt("manual_posting_start_topic"), `</i></b></span></td>
					</tr>
				</table>
				<table border="0" width="100%" align="center" cellspacing="1" cellpadding="3" class="bordercolor">
					<tr class="titlebg">
						<td>`, c.Txt("manual_posting_start_topic"), `</td>
					</tr>
					<tr>
						<td class="windowbg">
							<table border="0" cellpadding="3" width="100%">
								<tr class="windowbg">
									<td colspan="2" align="center"><a href="`, scripturl, `?action=help;page=post#standard">`, c.Txt("manual_posting_std_options"), `&nbsp;`, c.Txt("manual_posting_omit_clarity"), `</a></td>
								</tr>
								<tr>
									<td align="right"><b>`, c.Txt("manual_posting_subject"), `:</b></td>
									<td><input type="text" name="subject" size="80" maxlength="80" tabindex="1" /></td>
								</tr>
								<tr>
									<td valign="top" align="right"></td>
									<td>
									<textarea class="editor" name="message" rows="12" cols="60" onselect="storeCaret(this);" onclick="storeCaret(this);" onkeyup="storeCaret(this);" onchange="storeCaret(this);" tabindex="2">
</textarea></td>
								</tr>
								<tr class="windowbg">
									<td colspan="2" align="center"><a href="`, scripturl, `?action=help;page=post#additional">`, c.Txt("manual_posting_sec_additional_options"), `&nbsp;`, c.Txt("manual_posting_omit_clarity"), `</a></td>
								</tr>
								<tr>
									<td align="center" colspan="2"><span class="smalltext"><br />
									`, c.Txt("manual_posting_shortcuts"), `</span><br />
									<input type="button" accesskey="s" tabindex="3" value="`, c.Txt("manual_posting_posts"), `" /> <input type="button" accesskey="p" tabindex="4" value="`, c.Txt("manual_posting_preview"), `" /></td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
			</form><br />
		</div>
	</div><br />
	<ul>
		<li>`, c.Txt("manual_posting_nav_tree"), `</li>
		<li>`, c.Txt("manual_posting_spell_check"), `</li>
	</ul>
	<h3 id="newpoll">`, c.Txt("manual_posting_start_poll"), `</h3>
	<p>`, c.Txt("manual_posting_poll_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#newtopic">`, c.Txt("manual_posting_poll_desc_link_newtopic"), `</a>`, c.Txt("manual_posting_poll_desc_part2"), `</p>
	<p>`, c.Txt("manual_posting_poll_options"), `</p>
	<p>`, c.Txt("manual_posting_poll_note"), `</p>
	<h3 id="reply">`, c.Txt("manual_posting_replying"), `</h3>
	<p>`, c.Txt("manual_posting_replying_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#newtopic">`, c.Txt("manual_posting_replying_desc_link_newtopic"), `</a>`, c.Txt("manual_posting_replying_desc_part2"), `</p>
	<p>`, c.Txt("manual_posting_quick_reply_part1"), `<a href="`, scripturl, `?action=help;page=post#bbc">`, c.Txt("manual_posting_quick_reply_link_bbc"), `</a>`, c.Txt("manual_posting_quick_reply_part2"), `<a href="`, scripturl, `?action=help;page=post#smileys">`, c.Txt("manual_posting_quick_reply_link_bbc_smileys"), `</a>`, c.Txt("manual_posting_quick_reply_part3"), `</p>
	<h3 id="quote">`, c.Txt("manual_posting_quote_post"), `</h3>
	<p>`, c.Txt("manual_posting_quote_desc"), `</p>
	<ul>
		<li>`, c.Txt("manual_posting_quote_both_part1"), `<a href="`, scripturl, `?action=help;page=post#bbc">`, c.Txt("manual_posting_quote_both_link_bbc"), `</a>`, c.Txt("manual_posting_quote_both_part2"), `</li>
		<li>`, c.Txt("manual_posting_quote_independant_part1"), `<a href="`, scripturl, `?action=help;page=post#bbcref">`, c.Txt("manual_posting_quote_independant_link_bbcref"), `</a>`, c.Txt("manual_posting_quote_independant_part2"), `</li>
	</ul>
	<h3 id="modify">`, c.Txt("manual_posting_modify_delete"), `</h3>
	<p>`, c.Txt("manual_posting_modify_desc"), `</p>
	<p>`, c.Txt("manual_posting_delete_desc"), `</p>
	<h2 id="standard">`, c.Txt("manual_posting_sec_posting_options"), `</h2>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<br />
			<script language="JavaScript1.2" type="text/javascript">
//<![CDATA[
			function showimage()
			{
					document.images.icons.src = "`, c.Theme.ImagesURL(), `/post/" + document.forms.postmodify.icon.options[document.forms.postmodify.icon.selectedIndex].value + ".gif";
					document.images.icons.src ="`, c.Theme.ImagesURL(), `/post/" + document.forms.postmodify.icon.options[document.forms.postmodify.icon.selectedIndex].value + ".gif";
			}
			var currentSwap = false;
			function swapOptions()
			{
					document.getElementById("postMoreExpand").src = smf_images_url + "/" + (currentSwap ? "collapse.gif" : "expand.gif");
					document.getElementById("postMoreExpand").alt = currentSwap ? "-" : "+";
					document.getElementById("postMoreOptions").style.display = currentSwap ? "" : "none";
					if (document.getElementById("postAttachment"))
								document.getElementById("postAttachment").style.display = currentSwap ? "" : "none";
					if (document.getElementById("postAttachment2"))
								document.getElementById("postAttachment2").style.display = currentSwap ? "" : "none";
					currentSwap = !currentSwap;
			}
//]]>
</script>
			<form action="`, scripturl, `?action=help;page=post" method="post" accept-charset="`, c.CharacterSet, `" name="postmodify" style="margin: 0;" id="postmodify">
				<table border="0" width="100%" align="center" cellspacing="1" cellpadding="3" class="bordercolor">
					<tr>
						<td class="windowbg">
							<table border="0" cellpadding="3" width="100%">
								<tr>
									<td align="right"><b>`, c.Txt("manual_posting_msg_icon"), `:</b></td>
									<td><select name="icon" id="icon" onchange="showimage();">
										<option value="xx" selected="selected">
											`, c.Txt("manual_posting_standard_icon"), `
										</option>
										<option value="thumbup">
											`, c.Txt("manual_posting_thumb_up_icon"), `
										</option>
										<option value="thumbdown">
											`, c.Txt("manual_posting_thumb_down_icon"), `
										</option>
										<option value="exclamation">
											`, c.Txt("manual_posting_exc_pt_icon"), `
										</option>
										<option value="question">
											`, c.Txt("manual_posting_q_mark_icon"), `
										</option>
										<option value="lamp">
											`, c.Txt("manual_posting_lamp_icon"), `
										</option>
										<option value="smiley">
											`, c.Txt("manual_posting_smiley_icon"), `
										</option>
										<option value="angry">
											`, c.Txt("manual_posting_angry_icon"), `
										</option>
										<option value="cheesy">
											`, c.Txt("manual_posting_cheesy_icon"), `
										</option>
										<option value="grin">
											`, c.Txt("manual_posting_grin_icon"), `
										</option>
										<option value="sad">
											`, c.Txt("manual_posting_sad_icon"), `
										</option>
										<option value="wink">
											`, c.Txt("manual_posting_wink_icon"), `
										</option>
									</select> <img src="`, c.Theme.ImagesURL(), `/post/xx.gif" name="icons" border="0" hspace="15" alt="" id="icons" /></td>
								</tr>
																<tr>
									<td align="right"></td>
									<td valign="middle">
										<script language="JavaScript" type="text/javascript">
//<![CDATA[
										function bbc_highlight(something, mode)
										{
													something.style.backgroundImage = "url(" + smf_images_url + (mode ? "/bbc/bbc_hoverbg.gif)" : "/bbc/bbc_bg.gif)");
										}
//]]>
</script>
										<a href="javascript:void(0);" onclick="surroundText('[b]', '[/b]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/bold.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_bold_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[i]', '[/i]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/italicize.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_italicize_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[u]', '[/u]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/underline.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_underline_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[s]', '[/s]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/strike.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_strike_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[glow=red,2,300]', '[/glow]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/glow.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_glow_example"), `" /></a>
										<a href="javascript:void(0);" onclick="surroundText('[shadow=red,left]', '[/shadow]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/shadow.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_shadow_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[move]', '[/move]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/move.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_move_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[pre]', '[/pre]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/pre.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_pre_example"), `" /></a>
										<a href="javascript:void(0);" onclick="surroundText('[left]', '[/left]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/left.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_left_example"), `" /></a>
										<a href="javascript:void(0);" onclick="surroundText('[center]', '[/center]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/center.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_center_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[right]', '[/right]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/right.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_right_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[hr]', '', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/hr.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_hr_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[size=10pt]', '[/size]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/size.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_size_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[font=Verdana]', '[/font]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/face.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_face_example"), `" /></a>
										<select onchange="surroundText('[color='+this.options[this.selectedIndex].value+']', '[/color]', document.forms.postmodify.message); this.selectedIndex = 0;" style="margin-bottom: 1ex; margin-left: 2ex;">
											<option value="" selected="selected">
												`, c.Txt("manual_posting_Change_Color"), `
											</option>
											<option value="Black">
												`, c.Txt("manual_posting_color_black"), `
											</option>
											<option value="Red">
												`, c.Txt("manual_posting_color_red"), `
											</option>
											<option value="Yellow">
												`, c.Txt("manual_posting_color_yellow"), `
											</option>
											<option value="Pink">
												`, c.Txt("manual_posting_color_pink"), `
											</option>
											<option value="Green">
												`, c.Txt("manual_posting_color_green"), `
											</option>
											<option value="Orange">
												`, c.Txt("manual_posting_color_orange"), `
											</option>
											<option value="Purple">
												`, c.Txt("manual_posting_color_purple"), `
											</option>
											<option value="Blue">
												`, c.Txt("manual_posting_color_blue"), `
											</option>
											<option value="Beige">
												`, c.Txt("manual_posting_color_beige"), `
											</option>
											<option value="Brown">
												`, c.Txt("manual_posting_color_brown"), `
											</option>
											<option value="Teal">
												`, c.Txt("manual_posting_color_teal"), `
											</option>
											<option value="Navy">
												`, c.Txt("manual_posting_color_navy"), `
											</option>
											<option value="Maroon">
												`, c.Txt("manual_posting_color_maroon"), `
											</option>
											<option value="LimeGreen">
												`, c.Txt("manual_posting_color_lime"), `
											</option>
										</select><br />
										<a href="javascript:void(0);" onclick="surroundText('[flash=200,200]', '[/flash]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/flash.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_flash_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[img]', '[/img]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/img.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_img_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[url]', '[/url]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/url.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_url_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[email]', '[/email]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/email.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_email_example"), `" /></a>
										<a href="javascript:void(0);" onclick="surroundText('[ftp]', '[/ftp]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/ftp.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_ftp_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[table]', '[/table]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/table.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_table_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[tr]', '[/tr]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/tr.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_tr_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[td]', '[/td]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/td.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_td_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[sup]', '[/sup]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/sup.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_sup_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[sub]', '[/sub]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/sub.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_sub_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[tt]', '[/tt]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/tele.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_tele_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" />
										<a href="javascript:void(0);" onclick="surroundText('[code]', '[/code]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/code.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_code_example"), `" /></a><a href="javascript:void(0);" onclick="surroundText('[quote]', '[/quote]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/quote.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_quote_example"), `" /></a><img src="`, c.Theme.ImagesURL(), `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" /><a href="javascript:void(0);" onclick="surroundText('[list][li]', '[/li][li][/li][/list]', document.forms.postmodify.message);"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/list.gif" align="bottom" width="23" height="22" border="0" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" alt="`, c.Txt("manual_posting_list_example"), `" /></a>
									</td>
								</tr>
								<tr>
									<td align="right"></td>
									<td valign="middle">
										<a href="javascript:void(0);" onclick="replaceText(' :)', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/smiley.gif" align="bottom" alt="`, c.Txt("manual_posting_smiley_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' ;)', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/wink.gif" align="bottom" alt="`, c.Txt("manual_posting_wink_code"), `" border="0" /></a>
										<a href="javascript:void(0);" onclick="replaceText(' :D', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/cheesy.gif" align="bottom" alt="`, c.Txt("manual_posting_cheesy_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' ;D', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/grin.gif" align="bottom" alt="`, c.Txt("manual_posting_grin_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' &gt;:(', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/angry.gif" align="bottom" alt="`, c.Txt("manual_posting_angry_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :(', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/sad.gif" align="bottom" alt="`, c.Txt("manual_posting_sad_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :o', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/shocked.gif" align="bottom" alt="`, c.Txt("manual_posting_shocked_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' 8)', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/cool.gif" align="bottom" alt="`, c.Txt("manual_posting_cool_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' ???', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/huh.gif" align="bottom" alt="`, c.Txt("manual_posting_huh_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' ::)', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/rolleyes.gif" align="bottom" alt="`, c.Txt("manual_posting_rolleyes_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :P', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/tongue.gif" align="bottom" alt="`, c.Txt("manual_posting_tongue_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :-[', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/embarrassed.gif" align="bottom" alt="`, c.Txt("manual_posting_embarrassed_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :-X', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/lipsrsealed.gif" align="bottom" alt="`, c.Txt("manual_posting_lipsrsealed_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :-\\', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/undecided.gif" align="bottom" alt="`, c.Txt("manual_posting_undecided_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :-*', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/kiss.gif" align="bottom" alt="`, c.Txt("manual_posting_kiss_code"), `" border="0" /></a> <a href="javascript:void(0);" onclick="replaceText(' :\'(', document.forms.postmodify.message);"><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/cry.gif" align="bottom" alt="`, c.Txt("manual_posting_cry_code"), `" border="0" /></a><br />
									</td>
								</tr>
								<tr>
									<td valign="top" align="right"></td>
									<td>
									<textarea class="editor" name="message" rows="12" cols="60" onselect="storeCaret(this);" onclick="storeCaret(this);" onkeyup="storeCaret(this);" onchange="storeCaret(this);" tabindex="2">
</textarea></td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
			</form><br />
		</div>
	</div><br />
	<h3 id="messageicon">`, c.Txt("manual_posting_sub_message_icon"), `</h3>
	<p>`, c.Txt("manual_posting_msg_icon_dropdown"), `</p>
	<h3 id="bbc">`, c.Txt("manual_posting_sub_bbc"), `</h3>
	<p>`, c.Txt("manual_posting_bbc_desc"), `</p>
	<p>`, c.Txt("manual_posting_bbc_ref_part1"), `<a href="`, scripturl, `?action=help;page=post#bbcref">`, c.Txt("manual_posting_bbc_ref_link_bbcref"), `</a>`, c.Txt("manual_posting_bbc_ref_part2"), `</p>
	<h3 id="smileys">`, c.Txt("manual_posting_sub_smileys"), `</h3>
	<p>`, c.Txt("manual_posting_smiley_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#nosmileys">`, c.Txt("manual_posting_smiley_desc_link_nosmileys"), `</a>`, c.Txt("manual_posting_smiley_desc_part2"), `</p>
	<p>`, c.Txt("manual_posting_smiley_ref_part1"), `<a href="`, scripturl, `?action=help;page=post#smileysref">`, c.Txt("manual_posting_smiley_ref_link_smileysref"), `</a>`, c.Txt("manual_posting_smiley_ref_part2"), `</p>
	<h2 id="tags">`, c.Txt("manual_posting_sec_tags"), `</h2>
	<p>`, c.Txt("manual_posting_tags_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#bbcref">`, c.Txt("manual_posting_tags_desc_link_bbcref"), `</a>`, c.Txt("manual_posting_tags_desc_part2"), `</p>
	<p>`, c.Txt("manual_posting_note_tags"), `</p>
	<h2 id="additional">`, c.Txt("manual_posting_sec_additional_options"), `</h2>
	<p>`, c.Txt("manual_posting_sec_additional_options_desc"), `</p>
	<div style="border: solid 1px;">
		<div style="padding: 2px 30px;">
			<br />
			<script language="JavaScript1.2" type="text/javascript">
//<![CDATA[
			var currentSwap = false;
			function swapOptions()
			{
						document.getElementById("postMoreExpand").src = smf_images_url + "/" + (currentSwap ? "collapse.gif" : "expand.gif");
						document.getElementById("postMoreExpand").alt = currentSwap ? "-" : "+";
						document.getElementById("postMoreOptions").style.display = currentSwap ? "" : "none";
						if (document.getElementById("postAttachment"))
								document.getElementById("postAttachment").style.display = currentSwap ? "" : "none";
						if (document.getElementById("postAttachment2"))
								document.getElementById("postAttachment2").style.display = currentSwap ? "" : "none";
						currentSwap = !currentSwap;
			}
//]]>
</script>
			<form action="`, scripturl, `?action=help;page=post" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
				<table border="0" width="100%" align="center" cellspacing="1" cellpadding="3" class="bordercolor">
					<tr>
						<td class="windowbg">
							<table border="0" cellpadding="3" width="100%">
								<tr>
									<td colspan="2" style="padding-left: 5ex;"><a href="javascript:swapOptions();"><img src="`, c.Theme.ImagesURL(), `/expand.gif" alt="+" border="0" id="postMoreExpand" name="postMoreExpand" /></a> <a href="javascript:swapOptions();" class="board"><b>`, c.Txt("manual_posting_sec_additional_options"), `...</b></a></td>
								</tr>
								<tr>
									<td></td>
									<td>
										<div id="postMoreOptions">
											<table width="80%" cellpadding="0" cellspacing="0" border="0">
												<tr>
													<td class="smalltext"><input type="checkbox" class="check" />&nbsp;`, c.Txt("manual_posting_notify"), `</td>
												</tr>
												<tr>
													<td class="smalltext"><input type="checkbox" class="check" />&nbsp;`, c.Txt("manual_posting_return"), `</td>
												</tr>
												<tr>
													<td class="smalltext"><input type="checkbox" class="check" />&nbsp;`, c.Txt("manual_posting_no_smiley"), `</td>
												</tr>
											</table>
										</div>
									</td>
								</tr>
								<tr id="post`, c.Txt("manual_posting_attach"), `ment2">
									<td align="right" valign="top"><b>`, c.Txt("manual_posting_attach"), `:</b></td>
									<td class="smalltext"><input type="file" size="48" name="attachment[]" /><br />
									<input type="file" size="48" name="attachment[]" /><br />
									`, c.Txt("manual_posting_allowed_types"), `<br />
									`, c.Txt("manual_posting_max_size"), `</td>
								</tr>
								<tr>
									<td align="center" colspan="2">
										<script language="JavaScript" type="text/javascript">
//<![CDATA[
										swapOptions();
//]]>
</script> <span class="smalltext"><br />
										`, c.Txt("manual_posting_shortcuts"), `</span><br />
										<input type="button" accesskey="s" tabindex="3" value="`, c.Txt("manual_posting_posts"), `" /> <input type="button" accesskey="p" tabindex="4" value="`, c.Txt("manual_posting_preview"), `" />
									</td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
			</form><br />
		</div>
	</div><br />
	<h3 id="notify">`, c.Txt("manual_posting_sub_notify"), `</h3>
	<p>`, c.Txt("manual_posting_notify_desc"), `</p>
	<h3 id="return">`, c.Txt("manual_posting_sub_return"), `</h3>
	<p>`, c.Txt("manual_posting_return_desc"), `</p>
	<h3 id="nosmileys">`, c.Txt("manual_posting_sub_no_smiley"), `</h3>
	<p>`, c.Txt("manual_posting_no_smiley_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#smileysref">`, c.Txt("manual_posting_no_smiley_desc_link_smileysref"), `</a>`, c.Txt("manual_posting_no_smiley_desc_part2"), `</p>
	<h3 id="attachments">`, c.Txt("manual_posting_sub_attach"), `</h3>
	<p>`, c.Txt("manual_posting_attach_desc_part1"), `<a href="`, scripturl, `?action=help;page=post#modify">`, c.Txt("manual_posting_attach_desc_link_modify"), `</a>`, c.Txt("manual_posting_attach_desc_part2"), `</p>
	<ul>
		<li>`, c.Txt("manual_posting_attach_desc2"), `</li>
		<li>`, c.Txt("manual_posting_most_forums_attach"), `</li>
	</ul>
	<h2 id="references">`, c.Txt("manual_posting_sec_references"), `</h2>
	<h3 id="bbcref">`, c.Txt("manual_posting_sub_SMF_bbc"), `</h3>
	<p>`, c.Txt("manual_posting_sub_smf_bbc_desc"), `</p>
	<table id="reference1" cellspacing="4" cellpadding="2">
		<tr>
			<th>`, c.Txt("manual_posting_header_name"), `</th>
			<th>`, c.Txt("manual_posting_header_button"), `</th>
			<th>`, c.Txt("manual_posting_header_code"), `</th>
			<th>`, c.Txt("manual_posting_header_output"), `</th>
			<th>`, c.Txt("manual_posting_header_comments"), `</th>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_bold"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/bold.gif" alt="`, c.Txt("manual_posting_bbc_bold"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_bold_code"), `</td>
			<td><b>`, c.Txt("manual_posting_bold_output"), `</b></td>
			<td>`, c.Txt("manual_posting_bold_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_italic"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/italicize.gif" alt="`, c.Txt("manual_posting_bbc_italic"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_italic_code"), `</td>
			<td><i>`, c.Txt("manual_posting_italic_output"), `</i></td>
			<td>`, c.Txt("manual_posting_italic_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_underline"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/underline.gif" alt="`, c.Txt("manual_posting_bbc_underline"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_underline_code"), `</td>
			<td><u>`, c.Txt("manual_posting_underline_output"), `</u></td>
			<td>`, c.Txt("manual_posting_underline_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_strike"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/strike.gif" alt="`, c.Txt("manual_posting_bbc_strike"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_strike_code"), `</td>
			<td><s>`, c.Txt("manual_posting_strike_output"), `</s></td>
			<td>`, c.Txt("manual_posting_strike_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_glow"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/glow.gif" alt="`, c.Txt("manual_posting_bbc_glow"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_glow_code"), `</td>
			<td>
				<div style="filter: Glow(color=red, strength=2); width: 30px;">
					`, c.Txt("manual_posting_glow_output"), `
				</div>
			</td>
			<td>`, c.Txt("manual_posting_glow_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_shadow"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/shadow.gif" alt="`, c.Txt("manual_posting_bbc_shadow"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_shadow_code"), `</td>
			<td>
				<div style="filter: Shadow(color=red, direction=240); width: 30px;">
					`, c.Txt("manual_posting_shadow_output"), `
				</div>
			</td>
			<td>`, c.Txt("manual_posting_shadow_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_move"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/move.gif" alt="`, c.Txt("manual_posting_bbc_move"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_move_code"), `</td>
			<td><marquee>`, c.Txt("manual_posting_move_output"), `</marquee></td>
			<td>`, c.Txt("manual_posting_move_comment"), `</td>
		</tr>
				<tr>
			<td>`, c.Txt("manual_posting_bbc_pre"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/pre.gif" alt="`, c.Txt("manual_posting_bbc_pre"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>[pre]Simple<br />
			&nbsp;&nbsp;Machines<br />
			&nbsp;&nbsp;&nbsp;&nbsp;Forum[/pre]</td>
			<td>
				<pre>
Simple
  Machines
						Forum
</pre>
			</td>
			<td>`, c.Txt("manual_posting_pre_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_left"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/left.gif" alt="`, c.Txt("manual_posting_bbc_left"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_left_code"), `</td>
			<td>
				<p align="left">`, c.Txt("manual_posting_left_output"), `</p>
			</td>
			<td>`, c.Txt("manual_posting_left_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_centered"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/center.gif" alt="`, c.Txt("manual_posting_bbc_centered"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_centered_code"), `</td>
			<td>
				<center>
					`, c.Txt("manual_posting_centered_output"), `
				</center>
			</td>
			<td>`, c.Txt("manual_posting_centered_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_right"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/right.gif" alt="`, c.Txt("manual_posting_bbc_right"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_right_code"), `</td>
			<td>
				<p align="right">`, c.Txt("manual_posting_right_output"), `</p>
			</td>
			<td>`, c.Txt("manual_posting_right_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_rtl"), `</td>
			<td>*</td>
			<td>`, c.Txt("manual_posting_rtl_code"), `</td>
			<td>
				<div dir="rtl">
					`, c.Txt("manual_posting_rtl_output"), `
				</div>
			</td>
			<td>`, c.Txt("manual_posting_rtl_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_ltr"), `</td>
			<td>*</td>
			<td>`, c.Txt("manual_posting_ltr_code"), `</td>
			<td>
				<div dir="ltr">
					`, c.Txt("manual_posting_ltr_output"), `
				</div>
			</td>
			<td>`, c.Txt("manual_posting_ltr_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_hr"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/hr.gif" alt="`, c.Txt("manual_posting_bbc_hr"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_hr_code"), `</td>
			<td>
				<hr />
			</td>
			<td>`, c.Txt("manual_posting_hr_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_size"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/size.gif" alt="`, c.Txt("manual_posting_bbc_size"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_size_code"), `</td>
			<td><span style="font-size: 10pt;">`, c.Txt("manual_posting_size_output"), `</span></td>
			<td>`, c.Txt("manual_posting_size_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_font"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/face.gif" alt="`, c.Txt("manual_posting_bbc_font"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_font_code"), `</td>
			<td><span style="font-family: Verdana;">`, c.Txt("manual_posting_font_output"), `</span></td>
			<td>`, c.Txt("manual_posting_font_comment"), `</td>
		</tr>
				<tr>
			<td>`, c.Txt("manual_posting_bbc_color"), `</td>
			<td><select>
				<option value="" selected="selected">
					`, c.Txt("manual_posting_Change_Color"), `
				</option>
				<option value="Black">
					`, c.Txt("manual_posting_color_black"), `
				</option>
				<option value="Red">
					`, c.Txt("manual_posting_color_red"), `
				</option>
				<option value="Yellow">
					`, c.Txt("manual_posting_color_yellow"), `
				</option>
				<option value="Pink">
					`, c.Txt("manual_posting_color_pink"), `
				</option>
				<option value="Green">
					`, c.Txt("manual_posting_color_green"), `
				</option>
				<option value="Orange">
					`, c.Txt("manual_posting_color_orange"), `
				</option>
				<option value="Purple">
					`, c.Txt("manual_posting_color_purple"), `
				</option>
				<option value="Blue">
					`, c.Txt("manual_posting_color_blue"), `
				</option>
				<option value="Beige">
					`, c.Txt("manual_posting_color_beige"), `
				</option>
				<option value="Brown">
					`, c.Txt("manual_posting_color_brown"), `
				</option>
				<option value="Teal">
					`, c.Txt("manual_posting_color_teal"), `
				</option>
				<option value="Navy">
					`, c.Txt("manual_posting_color_navy"), `
				</option>
				<option value="Maroon">
					`, c.Txt("manual_posting_color_maroon"), `
				</option>
				<option value="LimeGreen">
					`, c.Txt("manual_posting_color_lime"), `
				</option>
			</select></td>
			<td>`, c.Txt("manual_posting_color_code"), `</td>
			<td><span style="color: red;">`, c.Txt("manual_posting_color_output"), `</span></td>
			<td>`, c.Txt("manual_posting_color_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_flash"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/flash.gif" alt="`, c.Txt("manual_posting_bbc_flash"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_flash_code"), `</td>
			<td><a href="http://somesite/somefile.swf" class="board" target="_blank">`, c.Txt("manual_posting_flash_output"), `</a></td>
			<td>`, c.Txt("manual_posting_flash_comment"), `</td>
		</tr>
		<tr>
			<td rowspan="2">`, c.Txt("manual_posting_bbc_img"), `</td>
			<td rowspan="2"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/img.gif" alt="`, c.Txt("manual_posting_bbc_img"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_img_top_code"), `</td>
			<td><img src="`, c.Theme.ImagesURL(), `/on.gif" alt="" /></td>
			<td rowspan="2">`, c.Txt("manual_posting_img_top_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_img_bottom_code"), `</td>
			<td><img src="`, c.Theme.ImagesURL(), `/on.gif" width="48" height="48" alt="" /></td>
		</tr>
		<tr>
			<td rowspan="2">`, c.Txt("manual_posting_bbc_url"), `</td>
			<td rowspan="2"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/url.gif" alt="`, c.Txt("manual_posting_bbc_url"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_url_code"), `</td>
			<td><a href="http://somesite" class="board" target="_blank">`, c.Txt("manual_posting_url_output"), `</a></td>
			<td rowspan="2">`, c.Txt("manual_posting_url_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_url_bottom_code"), `</td>
			<td><a href="http://somesite" class="board" target="_blank">`, c.Txt("manual_posting_url_bottom_output"), `</a></td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_email"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/email.gif" alt="`, c.Txt("manual_posting_bbc_email"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_email_code"), `</td>
			<td><a href="mailto:someone@somesite" class="board">`, c.Txt("manual_posting_email_output"), `</a></td>
			<td>`, c.Txt("manual_posting_email_comment"), `</td>
		</tr>
		<tr>
			<td rowspan="2">`, c.Txt("manual_posting_bbc_ftp"), `</td>
			<td rowspan="2"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/ftp.gif" alt="`, c.Txt("manual_posting_bbc_ftp"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_ftp_code"), `</td>
			<td><a href="ftp://somesite/somefile" class="board" target="_blank">`, c.Txt("manual_posting_ftp_output"), `</a></td>
			<td rowspan="2">`, c.Txt("manual_posting_ftp_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_ftp_bottom_code"), `</td>
			<td><a href="ftp://somesite/somefile" class="board" target="_blank">`, c.Txt("manual_posting_ftp_bottom_output"), `</a></td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_table"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/table.gif" alt="`, c.Txt("manual_posting_bbc_table"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_table_code"), `</td>
			<td>*</td>
			<td>`, c.Txt("manual_posting_table_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_row"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/tr.gif" alt="`, c.Txt("manual_posting_bbc_row"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_row_code"), `</td>
			<td>*</td>
			<td>`, c.Txt("manual_posting_row_comment"), `</td>
		</tr>
				<tr>
			<td rowspan="2">`, c.Txt("manual_posting_bbc_column"), `</td>
			<td rowspan="2"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/td.gif" alt="`, c.Txt("manual_posting_bbc_column"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_column_code"), `</td>
			<td>
				<table>
					<tr>
						<td valign="top">`, c.Txt("manual_posting_column_output"), `</td>
					</tr>
				</table>
			</td>
			<td rowspan="2">`, c.Txt("manual_posting_column_comment"), `</td>
		</tr>
		<tr>
			<td>[table][tr][td]SMF[/td]<br />
			[td]Bulletin[/td][/tr]<br />
			[tr][td]Board[/td]<br />
			[td]Code[/td][/tr][/table]</td>
			<td>
				<table>
					<tr>
						<td valign="top">SMF</td>
						<td valign="top">Bulletin</td>
					</tr>
					<tr>
						<td valign="top">Board</td>
						<td valign="top">Code</td>
					</tr>
				</table>
			</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_sup"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/sup.gif" alt="`, c.Txt("manual_posting_bbc_sup"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_sup_code"), `</td>
			<td><sup>`, c.Txt("manual_posting_sup_output"), `</sup></td>
			<td>`, c.Txt("manual_posting_sup_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_sub"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/sub.gif" alt="`, c.Txt("manual_posting_bbc_sub"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_sub_code"), `</td>
			<td><sub>`, c.Txt("manual_posting_sub_output"), `</sub></td>
			<td>`, c.Txt("manual_posting_sub_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_tt"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/tele.gif" alt="`, c.Txt("manual_posting_bbc_tt"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_tt_code"), `</td>
			<td><tt>`, c.Txt("manual_posting_tt_output"), `</tt></td>
			<td>`, c.Txt("manual_posting_tt_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_code"), `</td>
			<td><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/code.gif" alt="`, c.Txt("manual_posting_bbc_code"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_code_code"), `</td>
			<td>
				<div class="codeheader">
					Code:
				</div>
				<div class="code">
					<font color="#000000"><font color="#0000BB">&lt;?php phpinfo</font><font color="#007700">();</font> <font color="#0000BB">?&gt;</font></font>
				</div>
			</td>
			<td>`, c.Txt("manual_posting_code_comment"), `</td>
		</tr>
		<tr>
			<td rowspan="2">`, c.Txt("manual_posting_bbc_quote"), `</td>
			<td rowspan="2"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/quote.gif" alt="`, c.Txt("manual_posting_bbc_quote"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_quote_code"), `</td>
			<td>
				<div class="`, c.Txt("manual_posting_quote_output"), `header">
					Quote
				</div>
				<div class="quote">
					`, c.Txt("manual_posting_quote_output"), `
				</div>
			</td>
			<td rowspan="2">`, c.Txt("manual_posting_quote_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_quote_buttom_code"), `</td>
			<td>
				<div class="`, c.Txt("manual_posting_quote_buttom_output"), `header">
					Quote from: author
				</div>
				<div class="quote">
					`, c.Txt("manual_posting_quote_buttom_output"), `
				</div>
			</td>
		</tr>
		<tr>
			<td rowspan="2">`, c.Txt("manual_posting_bbc_list"), `</td>
			<td rowspan="2"><img onmouseover="bbc_highlight(this, true);" onmouseout="bbc_highlight(this, false);" src="`, c.Theme.ImagesURL(), `/bbc/list.gif" alt="`, c.Txt("manual_posting_bbc_list"), `" style="background-image: url(`, c.Theme.ImagesURL(), `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></td>
			<td>`, c.Txt("manual_posting_list_code"), `</td>
			<td>`, c.Txt("manual_posting_list_output"), `</td>
			<td rowspan="2">`, c.Txt("manual_posting_list_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_list_buttom_code"), `</td>
			<td>`, c.Txt("manual_posting_list_buttom_output"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_abbr"), `</td>
			<td>*</td>
			<td>`, c.Txt("manual_posting_abbr_code"), `</td>
			<td><abbr title="exempli gratia">`, c.Txt("manual_posting_abbr_output"), `</abbr></td>
			<td>`, c.Txt("manual_posting_abbr_comment"), `</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_bbc_acro"), `</td>
			<td>*</td>
			<td>`, c.Txt("manual_posting_acro_code"), `</td>
			<td><acronym title="Simple Machines Forum">`, c.Txt("manual_posting_acro_output"), `</acronym></td>
			<td>`, c.Txt("manual_posting_acro_comment"), `</td>
		</tr>
	</table><br />
	<h3 id="smileysref">`, c.Txt("manual_posting_sub_help_smileys"), `</h3>
	<p>`, c.Txt("manual_posting_smileys_help_desc"), `</p>
	<table id="reference2" cellspacing="4" cellpadding="2">
		<tr>
			<th>`, c.Txt("manual_posting_smileys_help_name"), `</th>
			<th>`, c.Txt("manual_posting_smileys_help_img"), `</th>
			<th>`, c.Txt("manual_posting_smileys_help_code"), `</th>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_smiley_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/smiley.gif" alt="" /></td>
			<td>:)</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_wink_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/wink.gif" alt="" /></td>
			<td>;)</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_cheesy_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/cheesy.gif" alt="" /></td>
			<td>:D</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_grin_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/grin.gif" alt="" /></td>
			<td>;D</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_angry_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/angry.gif" alt="" /></td>
			<td>&gt;:(</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_sad_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/sad.gif" alt="" /></td>
			<td>:(</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_shocked_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/shocked.gif" alt="" /></td>
			<td>:o</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_cool_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/cool.gif" alt="" /></td>
			<td>8)</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_huh_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/huh.gif" alt="" /></td>
			<td>???</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_rolleyes_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/rolleyes.gif" alt="" /></td>
			<td>::)</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_tongue_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/tongue.gif" alt="" /></td>
			<td>:P</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_embarrassed_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/embarrassed.gif" alt="" /></td>
			<td>:-[</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_lipsrsealed_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/lipsrsealed.gif" alt="" /></td>
			<td>:-X</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_undecided_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/undecided.gif" alt="" /></td>
			<td>:-\</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_kiss_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/kiss.gif" alt="" /></td>
			<td>:-*</td>
		</tr>
		<tr>
			<td>`, c.Txt("manual_posting_cry_help_name"), `</td>
			<td><img src="`, c.App.Setting("smileys_url"), `/`, c.User.SmileySet, `/cry.gif" alt="" /></td>
			<td>:'(</td>
		</tr>
	</table><br />
	<p>`, c.Txt("manual_posting_smiley_parse"), `</p>`)
}
