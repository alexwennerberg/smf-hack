package app

// Ports of Themes/default/ManageBans.template.php:
// template_main / ban_edit / ban_edit_trigger / browse_triggers / ban_log.

// templateBanMain is template_main().
func templateBanMain(c *Ctx) {
	page := c.Page.(*BanListCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
	<form action="`, scripturl, `?action=ban;sa=list" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("ban_remove_selected_confirm"), `');">
		<table border="0" align="center" cellspacing="1" cellpadding="4" class="bordercolor" width="100%">
			<tr class="catbg3">
				<td colspan="8"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
			</tr><tr class="titlebg">`)

	for _, col := range page.Columns {
		width := ""
		if col.Width != "" {
			width = ` width="` + col.Width + `"`
		}
		if col.Selected {
			c.O(`
				<th`, width, `>
					<a href="`, col.Href, `">`, col.Label, `&nbsp;<img src="`, imagesURL, `/sort_`, page.SortDirection, `.gif" alt="" /></a>
				</th>`)
		} else if col.Sortable {
			c.O(`
				<th`, width, `>
					`, col.Link, `
				</th>`)
		} else {
			c.O(`
				<th`, width, `>
					`, col.Label, `
				</th>`)
		}
	}
	c.O(`
				<th><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></th>
			</tr>`)

	descPart := ""
	if page.SortDirection == "up" {
		descPart = ";desc"
	}
	for _, ban := range page.Bans {
		c.O(`
			<tr>
				<td align="left" valign="top" class="windowbg">`, ban.Name, `</td>
				<td align="left" valign="top" class="windowbg2">`, ban.Notes, `</td>
				<td align="left" valign="top" class="windowbg2">`, ban.Reason, `</td>
				<td align="left" valign="top" class="windowbg2">`, ban.Added, `</td>
				<td align="left" valign="top" class="windowbg">`, ban.Expires, `</td>
				<td align="center" valign="top" class="windowbg">`, ban.NumEntries, `</td>
				<td align="center" valign="top" class="windowbg2">
					&nbsp;<a href="`, scripturl, `?action=ban;sa=edit;sort=`, page.SortBy, descPart, `;bg=`, ban.ID, `">`, c.Txt("17"), `</a>
				</td>
				<td align="center" valign="top" class="windowbg2"><input type="checkbox" name="remove[]" value="`, ban.ID, `" class="check" /></td>
			</tr>`)
	}
	c.O(`
			<tr class="catbg3">
				<td colspan="8" align="left">
					<div style="float: left;">
						<b>`, c.Txt("139"), `:</b> `, page.PageIndex, `
					</div>
					<div style="float: right;">
						<input type="submit" name="removeBans" value="`, c.Txt("ban_remove_selected"), `" />
					</div>
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateBanEdit is template_ban_edit().
func templateBanEdit(c *Ctx) {
	page := c.Page.(*BanEditCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	ban := page.Ban

	title := c.Txt("ban_edit") + ` '` + ban.Name + `'`
	if ban.IsNew {
		title = c.Txt("ban_add_new")
	}
	c.O(`<br />
	<table border="0" align="center" cellspacing="1" cellpadding="4" class="tborder" width="60%">
		<tr class="catbg">
			<td>`, title, `</td>
		</tr><tr class="windowbg2">
			<td align="center">
				<form action="`, scripturl, `?action=ban;sa=edit" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="if (this.ban_name.value == '') {alert('`, c.Txt("ban_name_empty"), `'); return false;} if (this.partial_ban.checked &amp;&amp; !(this.cannot_post.checked || this.cannot_register.checked || this.cannot_login.checked)) {alert('`, c.Txt("ban_restriction_empty"), `'); return false;}">
					<table cellpadding="4">
						<tr>
							<th align="right">`, c.Txt("ban_name"), `:</th>
							<td align="left"><input type="text" name="ban_name" value="`, ban.Name, `" size="50" /></td>
						</tr><tr>
							<th align="right" valign="top">`, c.Txt("ban_expiration"), `:</th>
							<td align="left"><input type="radio" name="expiration" value="never" id="never_expires" onclick="updateStatus();"`, checkedIf(ban.ExpirationState == "never"), ` class="check" />&nbsp;&nbsp;<label for="never_expires">`, c.Txt("never"), `</label><br />
							<input type="radio" name="expiration" value="one_day" id="expires_one_day" onclick="updateStatus();"`, checkedIf(ban.ExpirationState == "still_active_but_we_re_counting_the_days"), ` class="check" />&nbsp;&nbsp;<label for="expires_one_day">`, c.Txt("ban_will_expire_within"), `</label>: <input type="text" name="expire_date" id="expire_date" size="3" value="`, int(ban.ExpirationDays), `" /> `, c.Txt("ban_days"), `<br />
							<input type="radio" name="expiration" value="expired" id="already_expired" onclick="updateStatus();"`, checkedIf(ban.ExpirationState == "expired"), ` class="check" />&nbsp;&nbsp;<label for="already_expired">`, c.Txt("ban_expired"), `</label>
							</td>
						</tr><tr>
							<th align="right" valign="bottom">`, c.Txt("ban_reason"), `:</th>
							<td align="left">
								<div class="smalltext">`, c.Txt("ban_reason_desc"), `</div>
								<input type="text" name="reason" value="`, ban.Reason, `" size="50" />
							</td>
						</tr><tr>
							<th align="right" valign="middle">`, c.Txt("ban_notes"), `:</th>
							<td align="left">
								<div class="smalltext">`, c.Txt("ban_notes_desc"), `</div>
								<textarea name="notes" cols="50" rows="3">`, ban.Notes, `</textarea>
							</td>
						</tr><tr>
							<th align="right" valign="top">`, c.Txt("ban_restriction"), `:</th>
							<td align="left">
								<input type="radio" name="full_ban" id="full_ban" value="1" onclick="updateStatus();"`, checkedIf(ban.CannotAccess), ` class="check" />&nbsp;&nbsp;<label for="full_ban">`, c.Txt("ban_full_ban"), `</label><br />
								<input type="radio" name="full_ban" id="partial_ban" value="0" onclick="updateStatus();"`, checkedIf(!ban.CannotAccess), ` class="check" />&nbsp;&nbsp;<label for="partial_ban">`, c.Txt("ban_partial_ban"), `</label><br />
								&nbsp;&nbsp;&nbsp;<input type="checkbox" name="cannot_post" id="cannot_post" value="1"`, checkedIf(ban.CannotPost), ` class="check" /> <label for="cannot_post">`, c.Txt("ban_cannot_post"), `</label> (<a href="`, scripturl, `?action=helpadmin;help=ban_cannot_post" onclick="return reqWin(this.href);">?</a>)<br />
								&nbsp;&nbsp;&nbsp;<input type="checkbox" name="cannot_register" id="cannot_register" value="1"`, checkedIf(ban.CannotRegister), ` class="check" /> <label for="cannot_register">`, c.Txt("ban_cannot_register"), `</label><br />
								&nbsp;&nbsp;&nbsp;<input type="checkbox" name="cannot_login" id="cannot_login" value="1"`, checkedIf(ban.CannotLogin), ` class="check" /> <label for="cannot_login">`, c.Txt("ban_cannot_login"), `</label><br />
							</td>
						</tr>`)

	if page.Suggestions != nil {
		sug := page.Suggestions
		c.O(`
						<tr>
							<th align="right" valign="top">`, c.Txt("ban_triggers"), `:</th>
							<td>
								<table cellpadding="4">
									<tr>
										<td valign="bottom"><input type="checkbox" name="ban_suggestion[]" id="main_ip_check" value="main_ip" class="check" /></td>
										<td align="left" valign="top">
											`, c.Txt("ban_on_ip"), `:<br />
											<input type="text" name="main_ip" value="`, sug.MainIP, `" size="50" onfocus="document.getElementById('main_ip_check').checked = true;" />
										</td>
									</tr><tr>`)
		if c.App.SettingEmpty("disableHostnameLookup") {
			c.O(`
										<td valign="bottom"><input type="checkbox" name="ban_suggestion[]" id="hostname_check" value="hostname" class="check" /></td>
										<td align="left" valign="top">
											`, c.Txt("ban_on_hostname"), `:<br />
											<input type="text" name="hostname" value="`, sug.Hostname, `" size="50" onfocus="document.getElementById('hostname_check').checked = true;" />
										</td>
									</tr><tr>`)
		}
		c.O(`
										<td valign="bottom"><input type="checkbox" name="ban_suggestion[]" id="email_check" value="email" class="check" /></td>
										<td align="left" valign="top">
											`, c.Txt("ban_on_email"), `:<br />
											<input type="text" name="email" value="`, sug.Email, `" size="50" onfocus="document.getElementById('email_check').checked = true;" />
										</td>
									</tr><tr>
										<td valign="bottom"><input type="checkbox" name="ban_suggestion[]" id="user_check" value="user" class="check" /></td>
										<td align="left" valign="top">
											`, c.Txt("ban_on_username"), `:<br />`)
		if !sug.HasMember {
			c.O(`
											<input type="text" name="user" id="user" value="" size="40" onfocus="document.getElementById('user_check').checked = true;" />&nbsp;<a href="`, scripturl, `?action=findmember;input=user;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, imagesURL, `/icons/assist.gif" alt="`, c.Txt("find_members"), `" /></a>`)
		} else {
			c.O(`
											`, sug.Member.Link, `
											<input type="hidden" name="bannedUser" value="`, sug.Member.ID, `" />`)
		}
		c.O(`
										</td>
									</tr>`)
		if len(sug.MessageIPs) > 0 {
			c.O(`
									<tr>
										<th align="left" colspan="2"><br />`, c.Txt("ips_in_messages"), `:</th>
									</tr>`)
			for _, ip := range sug.MessageIPs {
				c.O(`
									<tr>
										<td><input type="checkbox" name="ban_suggestion[ips][]" value="`, ip, `" class="check" /></td>
										<td align="left">`, ip, `</td>
									</tr>`)
			}
		}
		if len(sug.ErrorIPs) > 0 {
			c.O(`
									<tr>
										<th align="left" colspan="2"><br />`, c.Txt("ips_in_errors"), `:</th>
									</tr>`)
			for _, ip := range sug.ErrorIPs {
				c.O(`
									<tr>
										<td><input type="checkbox" name="ban_suggestion[ips][]" value="`, ip, `" class="check" /></td>
										<td align="left">`, ip, `</td>
									</tr>`)
			}
		}
		c.O(`
								</table>
							</td>
						</tr>`)
	}

	submitName, submitVal := "modify_ban", c.Txt("ban_modify")
	if ban.IsNew {
		submitName, submitVal = "add_ban", c.Txt("ban_add")
	}
	c.O(`
						<tr>
							<td colspan="2" align="right"><input type="submit" name="`, submitName, `" value="`, submitVal, `" /></td>
						</tr>
					</table>`)
	if ban.IsNew {
		c.O(`<br />
					`, c.Txt("ban_add_notes"))
	}
	c.O(`
					<input type="hidden" name="old_expire" value="`, int(ban.ExpirationDays), `" />
					<input type="hidden" name="bg" value="`, ban.ID, `" />
					<input type="hidden" name="sc" value="`, c.Sc, `" />
				</form>
			</td>
		</tr>`)

	if !ban.IsNew && page.Suggestions == nil {
		c.O(`
		<tr>
			<td align="center" style="padding: 0px;">
				<form action="`, scripturl, `?action=ban;sa=edit" method="post" accept-charset="`, c.CharacterSet, `" style="padding: 0px;margin: 0px;" onsubmit="return confirm('`, c.Txt("ban_remove_selected_triggers_confirm"), `');">
					<table cellpadding="4" cellspacing="1" width="100%"><tr class="titlebg">
						<td width="65%" align="left">`, c.Txt("ban_banned_entity"), `</td>
						<td width="15%" align="center">`, c.Txt("ban_hits"), `</td>
						<td width="15%" align="center">`, c.Txt("ban_actions"), `</td>
						<td width="5%" align="center"><input type="checkbox" onclick="invertAll(this, this.form, 'ban_items');" class="check" /></td>
					</tr>`)
		if len(page.BanItems) == 0 {
			c.O(`
					<tr class="windowbg2"><td colspan="4">(`, c.Txt("ban_no_triggers"), `)</td></tr>`)
		} else {
			for _, item := range page.BanItems {
				c.O(`
						<tr class="windowbg2" align="left">
							<td>`)
				switch item.Type {
				case "ip":
					c.O(`<b>`, c.Txt("512"), `:</b>&nbsp;`, item.IP)
				case "hostname":
					c.O(`<b>`, c.Txt("hostname"), `:</b>&nbsp;`, item.Hostname)
				case "email":
					c.O(`<b>`, c.Txt("69"), `:</b>&nbsp;`, item.Email)
				case "user":
					c.O(`<b>`, c.Txt("35"), `:</b>&nbsp;`, item.UserLink)
				}
				c.O(`
						</td>
						<td class="windowbg" align="center">`, item.Hits, `</td>
						<td class="windowbg" align="center"><a href="`, scripturl, `?action=ban;sa=edittrigger;bg=`, ban.ID, `;bi=`, item.ID, `">`, c.Txt("ban_edit_trigger"), `</a></td>
						<td align="center" class="windowbg2"><input type="checkbox" name="ban_items[]" value="`, item.ID, `" class="check" /></td>
					</tr>`)
			}
		}
		c.O(`
					<tr class="catbg3">
						<td colspan="4" align="right">
							<div style="float: left;">
								[<a href="`, scripturl, `?action=ban;sa=edittrigger;bg=`, ban.ID, `"><b>`, c.Txt("ban_add_trigger"), `</b></a>]
							</div>
							<input type="submit" name="remove_selection" value="`, c.Txt("ban_remove_selected_triggers"), `" />
							</div>
						</td>
					</tr>
					</table>
					<input type="hidden" name="bg" value="`, ban.ID, `" />
					<input type="hidden" name="sc" value="`, c.Sc, `" />
				</form>
			</td>
		</tr>`)
	}
	c.O(`
	</table>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function updateStatus()
		{
			document.getElementById("expire_date").disabled = !document.getElementById("expires_one_day").checked;
			document.getElementById("cannot_post").disabled = document.getElementById("full_ban").checked;
			document.getElementById("cannot_register").disabled = document.getElementById("full_ban").checked;
			document.getElementById("cannot_login").disabled = document.getElementById("full_ban").checked;
		}
		window.onload = updateStatus;
	// ]]></script>`)
}

// templateBanEditTrigger is template_ban_edit_trigger().
func templateBanEditTrigger(c *Ctx) {
	page := c.Page.(*BanEditTriggerCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	title := c.Txt("ban_edit_trigger_title")
	if page.IsNew {
		title = c.Txt("ban_add_trigger")
	}
	c.O(`
	<form action="`, scripturl, `?action=ban;sa=edit" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" align="center" cellspacing="0" cellpadding="4" class="tborder" width="60%">
			<tr class="titlebg">
				<td>`, title, `</td>
			</tr>
			<tr class="windowbg">
				<td align="center">
					<table cellpadding="4">
						<tr>
							<td valign="bottom"><input type="radio" name="bantype" value="ip_ban"`, checkedIf(page.IP.Selected), ` /></td>
							<td align="left" valign="top">
								`, c.Txt("ban_on_ip"), `:<br />
								<input type="text" name="ip" value="`, page.IP.Value, `" size="50" onfocus="selectRadioByName(this.form.bantype, 'ip_ban');" />
							</td><td>
							</td>
						</tr><tr>`)
	if c.App.SettingEmpty("disableHostnameLookup") {
		c.O(`
							<td valign="bottom"><input type="radio" name="bantype" value="hostname_ban"`, checkedIf(page.Hostname.Selected), ` /></td>
							<td align="left" valign="top">
								`, c.Txt("ban_on_hostname"), `:<br />
								<input type="text" name="hostname" value="`, page.Hostname.Value, `" size="50" onfocus="selectRadioByName(this.form.bantype, 'hostname_ban');" />
							</td><td>
							</td>
						</tr><tr>`)
	}
	c.O(`
							<td valign="bottom"><input type="radio" name="bantype" value="email_ban"`, checkedIf(page.Email.Selected), ` /></td>
							<td align="left" valign="top">
								`, c.Txt("ban_on_email"), `:<br />
								<input type="text" name="email" value="`, page.Email.Value, `" size="50" onfocus="selectRadioByName(this.form.bantype, 'email_ban');" />
							</td><td>
							</td>
						</tr><tr>
							<td valign="bottom"><input type="radio" name="bantype" value="user_ban"`, checkedIf(page.BannedUser.Selected), ` /></td>
							<td align="left" valign="top">
								`, c.Txt("ban_on_username"), `:<br />
								<input type="text" name="user" id="user" value="`, page.BannedUser.Value, `" size="50" onfocus="selectRadioByName(this.form.bantype, 'user_ban');" />
							</td><td valign="bottom">
								<a href="`, scripturl, `?action=findmember;input=user;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, imagesURL, `/icons/assist.gif" alt="`, c.Txt("find_members"), `" /></a>
							</td>
						</tr><tr>
							<td colspan="3" align="right"><br />
								<input type="submit" name="`, submitTriggerName(page.IsNew), `" value="`, submitTriggerVal(c, page.IsNew), `" />
							</td>
						</tr>
					</table>
				</td>
			</tr>
		</table>
		<input type="hidden" name="bi" value="`, page.ID, `" />
		<input type="hidden" name="bg" value="`, page.Group, `" />
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// checkedIf returns the checked attribute when b is true.
func checkedIf(b bool) string {
	if b {
		return ` checked="checked"`
	}
	return ""
}

func submitTriggerName(isNew bool) string {
	if isNew {
		return "add_new_trigger"
	}
	return "edit_trigger"
}
func submitTriggerVal(c *Ctx, isNew bool) string {
	if isNew {
		return c.Txt("ban_add_trigger_submit")
	}
	return c.Txt("ban_edit_trigger_submit")
}

// templateBanBrowseTriggers is template_browse_triggers().
func templateBanBrowseTriggers(c *Ctx) {
	page := c.Page.(*BanBrowseCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	sel := func(entity string) string {
		if page.SelectedEntity == entity {
			return `<img src="` + imagesURL + `/selected.gif" alt="&gt;" /> `
		}
		return ""
	}

	c.O(`
<form action="`, scripturl, `?action=ban;sa=browse;entity=`, page.SelectedEntity, `" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("ban_remove_selected_triggers_confirm"), `');">
	<table border="0" align="center" cellspacing="1" cellpadding="4" class="tborder" width="100%">
		<tr class="titlebg">
			<td colspan="4">`, c.Txt("ban_trigger_browse"), `</td>
		</tr><tr class="catbg3">
			<td colspan="4">
				<a href="`, scripturl, `?action=ban;sa=browse;entity=ip">`, sel("ip"), c.Txt("512"), `</a>&nbsp;|&nbsp;<a href="`, scripturl, `?action=ban;sa=browse;entity=hostname">`, sel("hostname"), c.Txt("hostname"), `</a>&nbsp;|&nbsp;<a href="`, scripturl, `?action=ban;sa=browse;entity=email">`, sel("email"), c.Txt("69"), `</a>&nbsp;|&nbsp;<a href="`, scripturl, `?action=ban;sa=browse;entity=member">`, sel("member"), c.Txt("35"), `</a>
			</td>
		</tr><tr class="titlebg">
			<th align="left">`, c.Txt("ban_banned_entity"), `</th>
			<th align="left">`, c.Txt("ban_name"), `</th>
			<th>`, c.Txt("ban_hits"), `</th>
			<th align="center"><input type="checkbox" onclick="invertAll(this, this.form);" class="check" /></th>
		</tr>`)
	if len(page.BanItems) == 0 {
		c.O(`
				<tr class="windowbg2"><td colspan="4">(`, c.Txt("ban_no_triggers"), `)</td></tr>`)
	} else {
		for _, item := range page.BanItems {
			c.O(`
		<tr class="windowbg2">
			<td>`, item.Entity, `</td>
			<td>`, item.GroupLink, `</td>
			<td align="center" class="windowbg">`, item.Hits, `</td>
			<td align="center"><input type="checkbox" name="remove[]" value="`, item.ID, `" class="check" /></td>
		</tr>`)
		}
	}
	c.O(`
		<tr class="windowbg">
			<td align="right" colspan="4">
				<input type="submit" name="remove_triggers" value="`, c.Txt("ban_remove_selected_triggers"), `" />
				<input type="hidden" name="sc" value="`, c.Sc, `" />
				<input type="hidden" name="start" value="`, page.Start, `" />
			</td>
		</tr>
		<tr class="catbg3">
			<td align="left" colspan="5" style="padding: 5px;"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
		</tr>
	</table>
</form>`)
}

// templateBanLog is template_ban_log().
func templateBanLog(c *Ctx) {
	page := c.Page.(*BanLogCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	flip := ""
	if page.SortDirection == "up" {
		flip = ";desc"
	}
	sortImg := func(col string) string {
		if page.Sort == col {
			return `&nbsp;<img src="` + imagesURL + `/sort_` + page.SortDirection + `.gif" alt="" />`
		}
		return ""
	}

	c.O(`
	<form action="`, scripturl, `?action=ban;sa=log" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" align="center" cellspacing="1" cellpadding="4" class="bordercolor" width="100%">
			<tr class="catbg3">
				<td colspan="7"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
			</tr><tr class="titlebg">
				<th>
					<a href="`, scripturl, `?action=ban;sa=log;sort=ip`, flip, `;start=`, page.Start, `">`, c.Txt("ban_log_ip"), sortImg("ip"), `</a>
				</th>
				<th>
					<a href="`, scripturl, `?action=ban;sa=log;sort=email`, flip, `;start=`, page.Start, `">`, c.Txt("ban_log_email"), sortImg("email"), `</a>
				</th>
				<th>
					<a href="`, scripturl, `?action=ban;sa=log;sort=name`, flip, `;start=`, page.Start, `">`, c.Txt("ban_log_member"), sortImg("name"), `</a>
				</th>
				<th>
					<a href="`, scripturl, `?action=ban;sa=log;sort=date`, flip, `;start=`, page.Start, `">`, c.Txt("ban_log_date"), sortImg("date"), `</a>
				</th>
				<th><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></th>
			</tr>`)
	if len(page.Entries) == 0 {
		c.O(`
			<tr class="windowbg2">
				<td colspan="5">(`, c.Txt("ban_log_no_entries"), `)</td>
			</tr>`)
	} else {
		for _, log := range page.Entries {
			member := log.MemberLink
			if log.MemberID == 0 {
				member = `<i>` + c.Txt("470") + `</i>`
			}
			c.O(`
			<tr>
				<td class="windowbg"><a href="`, scripturl, `?action=trackip;searchip=`, log.IP, `">`, log.IP, `</a></td>
				<td class="windowbg2">`, log.Email, `</td>
				<td class="windowbg">`, member, `</td>
				<td class="windowbg2">`, log.Date, `</td>
				<td class="windowbg" align="center"><input type="checkbox" name="remove[]" value="`, log.ID, `" class="check" /></td>
			</tr>`)
		}
		c.O(`
			<tr class="windowbg2">
				<td colspan="5" align="right">
					<input type="submit" name="removeAll" value="`, c.Txt("ban_log_remove_all"), `" onclick="return confirm('`, c.Txt("ban_log_remove_all_confirm"), `');" />
					<input type="submit" name="removeSelected" value="`, c.Txt("ban_log_remove_selected"), `" onclick="return confirm('`, c.Txt("ban_log_remove_selected_confirm"), `');" />
				</td>
			</tr>`)
	}
	c.O(`
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
