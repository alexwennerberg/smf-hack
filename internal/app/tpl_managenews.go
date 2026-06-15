package app

// Ports of Themes/default/ManageNews.template.php: template_edit_news /
// template_news_settings and the newsletter templates (template_email_members /
// _compose / _send).

// templateEditNews is template_edit_news().
func templateEditNews(c *Ctx) {
	page := c.Page.(*EditNewsCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=news;sa=editnews" method="post" accept-charset="`, c.CharacterSet, `" name="postmodify" id="postmodify">
			<table width="85%" cellpadding="3" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<th width="50%"></th>
					<th align="left" width="45%">`, c.Txt("507"), `</th>
					<th align="center" width="5%"><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></th>
				</tr>`)

	for _, n := range page.News {
		c.O(`
				<tr class="windowbg2">
					<td align="center">
						<div style="margin-bottom: 2ex;"><textarea rows="3" cols="65" name="news[]" style="width: 85%;">`, n.Unparsed, `</textarea></div>
					</td><td align="left" valign="top">
						<div style="overflow: auto; width: 100%; height: 10ex;">`, n.Parsed, `</div>
					</td><td align="center">
						<input type="checkbox" name="remove[]" value="`, n.ID, `" class="check" />
					</td>
				</tr>`)
	}

	c.O(`
				<tr class="windowbg2">
					<td align="center">
						<div id="moreNewsItems"></div><div id="moreNewsItems_link" style="display: none;"><a href="javascript:void(0);" onclick="addNewsItem(); return false;">`, c.Txt("editnews_clickadd"), `</a></div>
						<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
							document.getElementById("moreNewsItems_link").style.display = "";

							function addNewsItem()
							{
								setOuterHTML(document.getElementById("moreNewsItems"), '<div style="margin-bottom: 2ex;"><textarea rows="3" cols="65" name="news[]" style="width: 85%;"></textarea></div><div id="moreNewsItems"></div>');
							}
						// ]]></script>
						<noscript>
							<div style="margin-bottom: 2ex;"><textarea rows="3" cols="65" style="width: 85%;" name="news[]"></textarea></div>
						</noscript>
					</td>
					<td colspan="2" valign="bottom" align="right" style="padding: 1ex;">
						<input type="submit" name="save_items" value="`, c.Txt("10"), `" /> <input type="submit" name="delete_selection" value="`, c.Txt("editnews_remove_selected"), `" onclick="return confirm('`, c.Txt("editnews_remove_confirm"), `');" />
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateNewsSettings is template_news_settings().
func templateNewsSettings(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=news;sa=settings" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("settings"), `</td>
			</tr>`)
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right" valign="top">`, c.Txt("groups_edit_news"), `:</td>
				<td width="50%">`)
		c.themeInlinePermissions("edit_news")
		c.O(`
				</td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right" valign="top">`, c.Txt("groups_send_mail"), `:</td>
				<td width="50%">`)
		c.themeInlinePermissions("send_mail")
		c.O(`
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right"><label for="xmlnews_enable_check">`, c.Txt("xmlnews_enable"), `</label> (<a href="`, scripturl, `?action=helpadmin;help=xmlnews_enable" onclick="return reqWin(this.href);">?</a>):</td>
				<td>
					<input type="checkbox" name="xmlnews_enable" id="xmlnews_enable_check"`, c.checkboxAttr("xmlnews_enable"), ` class="check" onclick="document.getElementById('xmlnews_maxlen_input').disabled = !this.checked;" />
				</td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("xmlnews_maxlen"), `</td>
				<td valign="top">
					<input type="hidden" name="xmlnews_maxlen" value="`, c.settingOrZero("xmlnews_maxlen"), `" />
					<input type="text" name="xmlnews_maxlen" id="xmlnews_maxlen_input" value="`, c.settingOrZero("xmlnews_maxlen"), `" />
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						document.getElementById("xmlnews_maxlen_input").disabled = !document.getElementById("xmlnews_enable_check").checked;
					// ]]></script>
				</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("news_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateEmailMembers is template_email_members() — the group picker.
func templateEmailMembers(c *Ctx) {
	page := c.Page.(*EmailMembersCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=news;sa=mailingcompose" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="600" cellpadding="5" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>`, c.Txt("6"), `</td>
				</tr><tr class="windowbg">
					<td class="smalltext" style="padding: 2ex;">`, c.Txt("smf250"), `</td>
				</tr><tr>
					<td class="windowbg2">`)

	for _, group := range page.Groups {
		c.O(`
						<label for="who_`, group.ID, `"><input type="checkbox" name="who[`, group.ID, `]" id="who_`, group.ID, `" value="`, group.ID, `" checked="checked" class="check" /> `, group.Name, `</label> <i>(`, group.MemberCount, `)</i><br />`)
	}

	c.O(`
						<br />
						<label for="checkAllGroups"><input type="checkbox" id="checkAllGroups" checked="checked" onclick="invertAll(this, this.form, 'who');" class="check" /> <i>`, c.Txt("737"), `</i></label><br />

						<hr />
					</td>
				</tr><tr>
					<td class="windowbg2">`)

	if page.CanSendPM {
		c.O(`
					<label for="sendPM"><input type="checkbox" name="sendPM" id="sendPM" value="1" class="check" /> `, c.Txt("email_as_pms"), `</label><br />`)
	}

	c.O(`
						<label for="email_force"><input type="checkbox" name="email_force" id="email_force" value="1" class="check" /> `, c.Txt("email_force"), `</label>
					</td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 1ex;" align="center">
						<input type="submit" value="`, c.Txt("65"), `" />
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateEmailMembersCompose is template_email_members_compose().
func templateEmailMembersCompose(c *Ctx) {
	page := c.Page.(*ComposeMailingCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<form action="`, scripturl, `?action=news;sa=mailingsend" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>
						<a href="`, scripturl, `?action=helpadmin;help=email_members" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("6"), `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" style="padding: 2ex;">`, c.Txt("735"), `</td>
				</tr><tr>
					<td class="windowbg2" align="center">
						<textarea cols="70" rows="7" name="emails" class="editor">`, page.Addresses, `</textarea>
					</td>
				</tr>
			</table>
			<br />
			<table width="600" cellpadding="5" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>`, c.Txt("338"), `</td>
				</tr><tr class="windowbg">
					<td class="smalltext" style="padding: 2ex;">`, c.Txt("email_variables"), `</td>
				</tr><tr>
					<td class="windowbg2">
						<input type="text" name="subject" size="60" value="`, page.DefaultSubject, `" /><br />
						<br />
						<textarea cols="70" rows="9" name="message" class="editor">`, page.DefaultMessage, `</textarea><br />
						<br />
						<label for="send_html"><input type="checkbox" name="send_html" id="send_html" class="check" onclick="this.form.parse_html.disabled = !this.checked;" /> `, c.Txt("email_as_html"), `</label><br />
						<label for="parse_html"><input type="checkbox" name="parse_html" id="parse_html" checked="checked" disabled="disabled" class="check" /> `, c.Txt("email_parsed_html"), `</label><br />
						<br />
						<div align="center"><input type="submit" value="`, c.Txt("sendtopic_send"), `" /></div>
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateEmailMembersSend is template_email_members_send() — the batch
// continuation page (auto-submits to send the next chunk).
func templateEmailMembersSend(c *Ctx) {
	page := c.Page.(*SendMailingCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<form action="`, scripturl, `?action=news;sa=mailingsend" method="post" accept-charset="`, c.CharacterSet, `" name="autoSubmit" id="autoSubmit">
			<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>
						<a href="`, scripturl, `?action=helpadmin;help=email_members" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("6"), `
					</td>
				</tr><tr>
					<td class="windowbg2"><b>`, formatPercent(page.PercentageDone), `% `, c.Txt("email_done"), `</b></td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 1ex;" align="center">
						<input type="submit" name="b" value="`, c.Txt("email_continue"), `" />
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
			<input type="hidden" name="emails" value="`, page.Emails, `" />
			<input type="hidden" name="subject" value="`, page.Subject, `" />
			<input type="hidden" name="message" value="`, page.Message, `" />
			<input type="hidden" name="start" value="`, page.Start, `" />
			<input type="hidden" name="send_html" value="`, page.SendHTML, `" />
			<input type="hidden" name="parse_html" value="`, page.ParseHTML, `" />
		</form>
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var countdown = 2;
			doAutoSubmit();

			function doAutoSubmit()
			{
				if (countdown == 0)
					document.forms.autoSubmit.submit();
				else if (countdown == -1)
					return;

				document.forms.autoSubmit.b.value = "`, c.Txt("email_continue"), ` (" + countdown + ")";
				countdown--;

				setTimeout("doAutoSubmit();", 1000);
			}
		// ]]></script>`)
}
