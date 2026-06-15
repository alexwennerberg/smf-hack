package app

// Hand-port of Themes/default/Errors.template.php (template_fatal_error;
// template_error_log arrives with the admin phase). Byte-for-byte output:
// every echo maps to a c.O call with identical literals.

// templateFatalError is template_fatal_error().
func templateFatalError(c *Ctx) {
	e, _ := c.Page.(*ErrorCtx)
	if e == nil {
		e = &ErrorCtx{}
	}

	c.O(`
<div>
	<table border="0" width="80%" cellspacing="0" align="center" cellpadding="4" class="tborder">
		<tr class="titlebg">
			<td>`, e.Title, `</td>
		</tr>
		<tr class="windowbg">
			<td style="padding: 3ex;">
				`, e.Message, `
			</td>
		</tr>
	</table>
</div>`)

	// Show a back button (using javascript.)
	c.O(`
<div align="center" style="margin-top: 2ex;"><a href="javascript:history.go(-1)">`, c.Txt("250"), `</a></div>`)
}

// templateKickGuest is template_kick_guest() from Login.template.php (the
// rest of the Login template ports in Phase 2, but isNotGuest() needs this
// screen now).
func templateKickGuest(c *Ctx) {
	// This isn't that much... just like normal login but with a message at
	// the top.
	c.O(`
		<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/sha1.js"></script>

		<form action="`, c.App.ScriptURL, `?action=login2" method="post" accept-charset="`, c.CharacterSet, `" name="frmLogin" id="frmLogin"`)
	if !c.DisableLoginHash {
		c.O(` onsubmit="hashLoginPassword(this, '`, c.Sc, `');"`)
	}
	c.O(`>
			<table border="0" cellspacing="0" cellpadding="3" class="tborder" align="center">
				<tr class="catbg">
					<td>`, c.Txt("633"), `</td>
				</tr><tr>`)

	// Show the message or default message.
	kick := c.KickMessage
	if empty(kick) {
		kick = c.Txt("634")
	}
	c.O(`
					<td class="windowbg" style="padding-top: 2ex; padding-bottom: 2ex;">
						`, kick, `<br />
						`, c.Txt("635"), ` <a href="`, c.App.ScriptURL, `?action=register">`, c.Txt("636"), `</a> `, c.Txt("637"), `
					</td>`)

	// And now the login information.
	c.O(`
				</tr><tr class="titlebg">
					<td><img src="`, c.Theme.ImagesURL(), `/icons/login_sm.gif" alt="" align="top" /> `, c.Txt("34"), `</td>
				</tr><tr>
					<td class="windowbg">
						<table border="0" cellpadding="3" cellspacing="0" align="center">
							<tr>
								<td align="right"><b>`, c.Txt("35"), `:</b></td>
								<td><input type="text" name="user" size="20" /></td>
							</tr><tr>
								<td align="right"><b>`, c.Txt("36"), `:</b></td>
								<td><input type="password" name="passwrd" size="20" /></td>
							</tr><tr>
								<td align="right"><b>`, c.Txt("497"), `:</b></td>
								<td><input type="text" name="cookielength" size="4" maxlength="4" value="`, c.App.Setting("cookieTime"), `" /></td>
							</tr><tr>
								<td align="right"><b>`, c.Txt("508"), `:</b></td>
								<td><input type="checkbox" name="cookieneverexp" class="check" onclick="this.form.cookielength.disabled = this.checked;" /></td>
							</tr><tr>
								<td align="center" colspan="2"><input type="submit" value="`, c.Txt("34"), `" style="margin-top: 2ex;" /></td>
							</tr><tr>
								<td align="center" colspan="2" class="smalltext"><a href="`, c.App.ScriptURL, `?action=reminder">`, c.Txt("315"), `</a><br /><br /></td>
							</tr>
						</table>
					</td>
				</tr>
			</table>

			<input type="hidden" name="hash_passwrd" value="" />
		</form>`)

	// Do the focus thing...
	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			document.forms.frmLogin.user.focus();
		// ]]></script>`)
}

// templateErrorLog is template_error_log() from Errors.template.php.
func templateErrorLog(c *Ctx) {
	page := c.Page.(*ErrorLogCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	descParam := ""
	if page.SortDirection == "down" {
		descParam = ";desc"
	}
	filterHref := ""
	if page.HasFilter {
		filterHref = page.Filter.Href
	}

	c.O(`
		<form action="`, scripturl, `?action=viewErrorLog`, descParam, `;start=`, page.Start, filterHref, `" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="if (lastClicked == 'remove_all' &amp;&amp; !confirm('`, c.Txt("sure_about_errorlog_remove"), `')) return false; else return true;">
			<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
				var lastClicked = "";
			// ]]></script>
			<table border="0" cellspacing="1" cellpadding="5" class="bordercolor" align="center">
				<tr class="catbg">
					<td colspan="2"><a href="`, scripturl, `?action=helpadmin;help=error_log" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("errlog1"), `</td>
				</tr><tr class="windowbg">
					<td class="smalltext" colspan="2" style="padding: 2ex;">`, c.Txt("errlog2"), `</td>
				</tr>`)

	if page.HasFilter {
		c.O(`
				<tr>
					<td colspan="2" class="windowbg2">
						<b>`, c.Txt("applying_filter"), `:</b> `, page.Filter.Entity, ` `, page.Filter.ValueHTML, ` (<a href="`, scripturl, `?action=viewErrorLog`, descParam, `">`, c.Txt("clear_filter"), `</a>)
					</td>
				</tr>`)
	}
	c.O(`
				<tr>
					<td colspan="2" class="titlebg">
						<table width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
							<td>`, c.Txt("139"), `: `, page.PageIndex, `</td>
						</tr></table>
					</td>
				</tr>`)

	delAllValue := c.Txt("smf219")
	if page.HasFilter {
		delAllValue = c.Txt("remove_filtered_results")
	}

	if len(page.Errors) > 0 {
		c.O(`
				<tr>
					<td colspan="2" align="left" class="windowbg2">
						<div style="float: right;"><input type="submit" value="`, c.Txt("remove_selection"), `" onclick="lastClicked = 'remove_selection';" /> <input type="submit" name="delall" value="`, delAllValue, `" onclick="lastClicked = 'remove_all';" /></div>
						<label for="check_all1"><input type="checkbox" id="check_all1" onclick="invertAll(this, this.form, 'delete[]'); this.form.check_all2.checked = this.checked;" class="check" /> <b>`, c.Txt("737"), `</b></label>
					</td>
				</tr>`)
	}

	for _, error := range page.Errors {
		revDesc := ";desc"
		if page.SortDirection == "down" {
			revDesc = ""
		}
		c.O(`
				<tr>
					<td width="15" align="center" class="windowbg2">
						<input type="checkbox" name="delete[]" value="`, error.ID, `" class="check" />
					</td><td class="windowbg2" width="100%"><table width="100%" class="windowbg2" border="0" cellspacing="7" cellpadding="0">
						<tr>
							<td class="windowbg2" width="50%">
								<a href="`, scripturl, `?action=viewErrorLog`, descParam, `;filter=ID_MEMBER;value=`, error.MemberID, `" title="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_member"), `"><img src="`, imagesURL, `/filter.gif" alt="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_member"), `" /></a>
								<b>`, error.MemberLink, `</b>
							</td><td class="windowbg2" width="50%" align="left">
								<a href="`, scripturl, `?action=viewErrorLog`, revDesc, filterHref, `" title="`, c.Txt("reverse_direction"), `"><img src="`, imagesURL, `/sort_`, page.SortDirection, `.gif" alt="" /></a>
								<b>`, error.Time, `</b>
							</td>
						</tr><tr>
							<td class="windowbg2" width="50%">
								<a href="`, scripturl, `?action=viewErrorLog`, descParam, `;filter=ip;value=`, error.MemberIP, `" title="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_ip"), `"><img src="`, imagesURL, `/filter.gif" alt="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_ip"), `" /></a>
								<b><a href="`, scripturl, `?action=trackip;searchip=`, error.MemberIP, `">`, error.MemberIP, `</a></b>&nbsp;&nbsp;
							</td><td class="windowbg2" width="50%">`)
		if error.MemberSession != "" {
			c.O(`
								<a href="`, scripturl, `?action=viewErrorLog`, descParam, `;filter=session;value=`, error.MemberSession, `" title="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_session"), `"><img src="`, imagesURL, `/filter.gif" alt="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_session"), `" /></a>
								`, error.MemberSession)
		}
		c.O(`
							</td>
						</tr><tr>
							<td class="windowbg2" colspan="2"><div style="overflow: hidden; width: 100%; white-space: nowrap;">
								<a href="`, scripturl, `?action=viewErrorLog`, descParam, `;filter=url;value=`, error.URLHref, `" title="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_url"), `"><img src="`, imagesURL, `/filter.gif" alt="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_url"), `" /></a>
								<a href="`, error.URLHTML, `">`, error.URLHTML, `</a>
								</div></td>
						</tr><tr>
							<td class="windowbg2" colspan="2">
								<div style="float: left;"><a href="`, scripturl, `?action=viewErrorLog`, descParam, `;filter=message;value=`, error.MessageHref, `" title="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_message"), `"><img src="`, imagesURL, `/filter.gif" alt="`, c.Txt("apply_filter"), `: `, c.Txt("filter_only_message"), `" /></a></div>
								<div style="float: left; margin-left: 1ex;">`, error.MessageHTML, `</div>
							</td>
						</tr>
					</table></td>
				</tr>`)
	}

	if len(page.Errors) > 0 {
		c.O(`
				<tr>
					<td colspan="2" class="windowbg2">
						<div style="float: right;"><input type="submit" value="`, c.Txt("remove_selection"), `" onclick="lastClicked = 'remove_selection';" /> <input type="submit" name="delall" value="`, delAllValue, `" onclick="lastClicked = 'remove_all';" /></div>
						<label for="check_all2"><input type="checkbox" id="check_all2" onclick="invertAll(this, this.form, 'delete[]'); this.form.check_all1.checked = this.checked;" class="check" /> <b>`, c.Txt("737"), `</b></label>
					</td>
				</tr>`)
	} else {
		c.O(`
				<tr>
					<td colspan="2" class="windowbg2">`, c.Txt("151"), `</td>
				</tr>`)
	}

	c.O(`
				<tr>
					<td colspan="2" class="titlebg">
						<table width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
							<td>`, c.Txt("139"), `: `, page.PageIndex, `</td>
						</tr></table>
					</td>
				</tr>
			</table><br />`)
	if page.SortDirection == "down" {
		c.O(`
			<input type="hidden" name="desc" value="1" />`)
	}
	c.O(`
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}
