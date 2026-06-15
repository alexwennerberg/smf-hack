package app

// Ports of the registration-center templates from Register.template.php:
// template_admin_register / edit_agreement / edit_reserved_words /
// admin_settings.

// templateAdminRegister is template_admin_register().
func templateAdminRegister(c *Ctx) {
	page := c.Page.(*AdminRegisterCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=regcenter" method="post" accept-charset="`, c.CharacterSet, `" name="postForm" id="postForm">
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function onCheckChange()
			{
				if (document.forms.postForm.emailActivate.checked || document.forms.postForm.password.value == '')
				{
					document.forms.postForm.emailPassword.disabled = true;
					document.forms.postForm.emailPassword.checked = true;
				}
				else
					document.forms.postForm.emailPassword.disabled = false;
			}
		// ]]></script>
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="70%" class="tborder">
			<tr class="titlebg">
				<td colspan="2" align="center">`, c.Txt("admin_browse_register_new"), `</td>
			</tr>`)
	if page.RegistrationDone != "" {
		c.O(`
			<tr class="windowbg2">
				<td colspan="2" align="center"><br />
					`, page.RegistrationDone, `
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2" align="center"><hr /></td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<th width="50%" align="right">
					<label for="user_input">`, c.Txt("admin_register_username"), `:</label>
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_register_username_desc"), `</div>
				</th>
				<td width="50%" align="left">
					<input type="text" name="user" id="user_input" size="30" maxlength="25" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="email_input">`, c.Txt("admin_register_email"), `:</label>
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_register_email_desc"), `</div>
				</th>
				<td width="50%" align="left">
					<input type="text" name="email" id="email_input" size="30" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="password_input">`, c.Txt("admin_register_password"), `:</label>
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_register_password_desc"), `</div>
				</th>
				<td width="50%" align="left">
					<input type="password" name="password" id="password_input" size="30" onchange="onCheckChange();" /><br />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="group_select">`, c.Txt("admin_register_group"), `:</label>
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_register_group_desc"), `</div>
				</th>
				<td width="50%" align="left">
					<select name="group" id="group_select">`)
	for _, g := range page.MemberGroups {
		c.O(`
						<option value="`, g[0], `">`, g[1], `</option>`)
	}
	emailActivateChecked := ""
	if a.SettingInt("registration_method") == 1 {
		emailActivateChecked = ` checked="checked"`
	}
	c.O(`
					</select><br />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="emailPassword_check">`, c.Txt("admin_register_email_detail"), `:</label>
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_register_email_detail_desc"), `</div>
				</th>
				<td width="50%" align="left">
					<input type="checkbox" name="emailPassword" id="emailPassword_check" checked="checked" disabled="disabled" class="check" /><br />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="emailActivate_check">`, c.Txt("admin_register_email_activate"), `:</label>
				</th>
				<td width="50%" align="left">
					<input type="checkbox" name="emailActivate" id="emailActivate_check"`, emailActivateChecked, ` onclick="onCheckChange();" class="check" /><br />
				</td>
			</tr><tr class="windowbg2">
				<td width="100%" colspan="2" align="right">
					<input type="submit" name="regSubmit" value="`, c.Txt("97"), `" />
					<input type="hidden" name="sa" value="register" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateEditAgreement is template_edit_agreement().
func templateEditAgreement(c *Ctx) {
	page := c.Page.(*EditAgreementCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=regcenter" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td align="center">`, c.Txt("smf11"), `</td>
			</tr>`)
	if page.Warning != "" {
		c.O(`
			<tr class="windowbg2">
				<td style="color: red; font-weight: bold;" align="center">
					`, page.Warning, `
				</td>
			</tr>`)
	}
	requireChecked := ""
	if page.RequireAgreement {
		requireChecked = ` checked="checked"`
	}
	c.O(`
			<tr class="windowbg2">
				<td align="center" style="padding-bottom: 1ex; padding-top: 2ex;">
					<textarea cols="70" rows="20" name="agreement" style="width: 94%; margin-bottom: 1ex;">`, page.Agreement, `</textarea><br />
					<label for="requireAgreement"><input type="checkbox" name="requireAgreement" id="requireAgreement"`, requireChecked, ` value="1" /> `, c.Txt("584"), `.</label><br />
					<br />
					<input type="submit" value="`, c.Txt("10"), `" />
					<input type="hidden" name="sa" value="agreement" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateEditReservedWords is template_edit_reserved_words().
func templateEditReservedWords(c *Ctx) {
	page := c.Page.(*ReservedCtx)
	scripturl := c.App.ScriptURL

	chk := func(b bool) string {
		if b {
			return `checked="checked"`
		}
		return ""
	}
	c.O(`
		<form action="`, scripturl, `?action=regcenter" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" cellspacing="1" class="bordercolor" align="center" cellpadding="4" width="80%">
				<tr class="titlebg">
					<td align="center">
						`, c.Txt("341"), `
					</td>
				</tr><tr>
					<td class="windowbg2" align="center">
						<div style="width: 80%;">
							<div style="margin-bottom: 2ex;">`, c.Txt("342"), `</div>
							<textarea cols="30" rows="6" name="reserved" style="width: 98%;">`, page.Words, `</textarea><br />

							<div align="left" style="margin-top: 2ex;">
								<label for="matchword"><input type="checkbox" name="matchword" id="matchword" `, chk(page.MatchWord), ` class="check" /> `, c.Txt("726"), `</label><br />
								<label for="matchcase"><input type="checkbox" name="matchcase" id="matchcase" `, chk(page.MatchCase), ` class="check" /> `, c.Txt("727"), `</label><br />
								<label for="matchuser"><input type="checkbox" name="matchuser" id="matchuser" `, chk(page.MatchUser), ` class="check" /> `, c.Txt("728"), `</label><br />
								<label for="matchname"><input type="checkbox" name="matchname" id="matchname" `, chk(page.MatchName), ` class="check" /> `, c.Txt("729"), `</label><br />
							</div>

							<input type="submit" value="`, c.Txt("10"), `" name="save_reserved_names" style="margin: 1ex;" />
						</div>
					</td>
				</tr>
			</table>
			<input type="hidden" name="sa" value="reservednames" />
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateAdminSettings is template_admin_settings().
func templateAdminSettings(c *Ctx) {
	page := c.Page.(*AdminSettingsCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	if page.UseGraphicLibrary {
		c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function refreshImages()
		{
			var imageType = document.getElementById('visual_verification_type_select').value;
			document.getElementById('verificiation_image').src = '`, page.VerificationImageHref, `;type=' + imageType;
		}
	// ]]></script>`)
	}

	regMethod := a.SettingInt("registration_method")
	sel := func(v int) string {
		if regMethod == v {
			return ` selected="selected"`
		}
		return ""
	}
	c.O(`
	<form action="`, scripturl, `?action=regcenter" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%" class="tborder">
			<tr class="titlebg">
				<td align="center">`, c.Txt("settings"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="center">
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						function checkCoppa()
						{
							var coppaDisabled = document.getElementById('coppaAge_input').value == 0;
							document.getElementById('coppaType_select').disabled = coppaDisabled;

							var disableContacts = coppaDisabled || document.getElementById('coppaType_select').options[document.getElementById('coppaType_select').selectedIndex].value != 1;
							document.getElementById('coppaPost_input').disabled = disableContacts;
							document.getElementById('coppaFax_input').disabled = disableContacts;
							document.getElementById('coppaPhone_input').disabled = disableContacts;
						}
					// ]]></script>
					<table border="0" cellspacing="0" cellpadding="4" align="center" width="100%">
						<tr class="windowbg2">
							<th width="50%" align="right">
								<label for="registration_method_select">`, c.Txt("admin_setting_registration_method"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=registration_method" onclick="return reqWin(this.href);">?</a>)</span>:
							</th>
							<td width="50%" align="left">
								<select name="registration_method" id="registration_method_select">
									<option value="0"`, sel(0), `>`, c.Txt("admin_setting_registration_standard"), `</option>
									<option value="1"`, sel(1), `>`, c.Txt("admin_setting_registration_activate"), `</option>
									<option value="2"`, sel(2), `>`, c.Txt("admin_setting_registration_approval"), `</option>
									<option value="3"`, sel(3), `>`, c.Txt("admin_setting_registration_disabled"), `</option>
								</select>
							</td>
						</tr>
						<tr class="windowbg2">
							<th width="50%" align="right">
								<label for="notify_new_registration_check">`, c.Txt("admin_setting_notify_new_registration"), `</label>:
							</th>
							<td width="50%" align="left">
								<input type="checkbox" name="notify_new_registration" id="notify_new_registration_check" `, c.checkboxAttr("notify_new_registration"), ` class="check" />
							</td>
						</tr><tr class="windowbg2">
							<th width="50%" align="right">
								<label for="send_welcomeEmail_check">`, c.Txt("admin_setting_send_welcomeEmail"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=send_welcomeEmail" onclick="return reqWin(this.href);">?</a>)</span>:
							</th>
							<td width="50%" align="left">
								<input type="checkbox" name="send_welcomeEmail" id="send_welcomeEmail_check"`, c.checkboxAttr("send_welcomeEmail"), ` class="check" />
							</td>
						</tr><tr class="windowbg2">
							<th width="50%" align="right">
								<label for="password_strength_select">`, c.Txt("admin_setting_password_strength"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=password_strength" onclick="return reqWin(this.href);">?</a>)</span>:
							</th>
							<td width="50%" align="left">
								<select name="password_strength" id="password_strength_select">`)
	ps := a.SettingInt("password_strength")
	psSel := func(v int) string {
		if ps == v {
			return ` selected="selected"`
		}
		return ""
	}
	c.O(`
									<option value="0"`, psSel(0), `>`, c.Txt("admin_setting_password_strength_low"), `</option>
									<option value="1"`, psSel(1), `>`, c.Txt("admin_setting_password_strength_medium"), `</option>
									<option value="2"`, psSel(2), `>`, c.Txt("admin_setting_password_strength_high"), `</option>
								</select>
							</td>
						</tr><tr class="windowbg2" valign="top">
							<th width="50%" align="right">
								<label for="visual_verification_type_select">
									`, c.Txt("admin_setting_image_verification_type"), `:<br />
									<span class="smalltext" style="font-weight: normal;">
										`, c.Txt("admin_setting_image_verification_type_desc"), `
									</span>
								</label>
							</th>
							<td width="50%" align="left">`)
	onchange := ""
	if page.UseGraphicLibrary {
		onchange = `onchange="refreshImages();"`
	}
	vv := a.SettingInt("disable_visual_verification")
	vvSel := func(v int) string {
		if vv == v {
			return `selected="selected"`
		}
		return ""
	}
	c.O(`
								<select name="visual_verification_type" id="visual_verification_type_select" `, onchange, `>
									<option value="1" `, vvSel(1), `>`, c.Txt("admin_setting_image_verification_off"), `</option>
									<option value="2" `, vvSel(2), `>`, c.Txt("admin_setting_image_verification_vsimple"), `</option>
									<option value="3" `, vvSel(3), `>`, c.Txt("admin_setting_image_verification_simple"), `</option>
									<option value="0" `, vvSel(0), `>`, c.Txt("admin_setting_image_verification_medium"), `</option>
									<option value="4" `, vvSel(4), `>`, c.Txt("admin_setting_image_verification_high"), `</option>
								</select><br />`)
	if page.UseGraphicLibrary {
		c.O(`
								<img src="`, page.VerificationImageHref, `;type=`, vv, `" alt="`, c.Txt("admin_setting_image_verification_sample"), `" id="verificiation_image" /><br />`)
	} else {
		c.O(`
								<span class="smalltext">`, c.Txt("admin_setting_image_verification_nogd"), `</span>`)
	}
	c.O(`
							</td>
						</tr><tr class="windowbg2">
							<td width="100%" colspan="2" align="center">
								<hr />
							</td>
						</tr><tr class="windowbg2" valign="top">
							<th width="50%" align="right">
								<label for="coppaAge_input">`, c.Txt("admin_setting_coppaAge"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=coppaAge" onclick="return reqWin(this.href);">?</a>)</span>:
								<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_setting_coppaAge_desc"), `</div>
							</th>
							<td width="50%" align="left">
								<input type="text" name="coppaAge" id="coppaAge_input" value="`, settingOrBlank(a, "coppaAge"), `" size="3" maxlength="3" onkeyup="checkCoppa();" />
							</td>
						</tr><tr class="windowbg2" valign="top">
							<th width="50%" align="right">
								<label for="coppaType_select">`, c.Txt("admin_setting_coppaType"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=coppaType" onclick="return reqWin(this.href);">?</a>)</span>:
							</th>
							<td width="50%" align="left">
								<select name="coppaType" id="coppaType_select" onchange="checkCoppa();">
									<option value="0"`, coppaTypeSel(a, 0), `>`, c.Txt("admin_setting_coppaType_reject"), `</option>
									<option value="1"`, coppaTypeSel(a, 1), `>`, c.Txt("admin_setting_coppaType_approval"), `</option>
								</select>
							</td>
						</tr><tr class="windowbg2" valign="top">
							<th width="50%" align="right">
								<label for="coppaPost_input">`, c.Txt("admin_setting_coppaPost"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=coppaPost" onclick="return reqWin(this.href);">?</a>)</span>:
								<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_setting_coppaPost_desc"), `</div>
							</th>
							<td width="50%" align="left">
								<textarea name="coppaPost" id="coppaPost_input" rows="4" cols="35">`, page.CoppaPost, `</textarea>
							</td>
						</tr><tr class="windowbg2" valign="top">
							<th width="50%" align="right">
								<label for="coppaFax_input">`, c.Txt("admin_setting_coppaFax"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=coppaPost" onclick="return reqWin(this.href);">?</a>)</span>:
								<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_setting_coppaPost_desc"), `</div>
							</th>
							<td width="50%" align="left">
								<input type="text" name="coppaFax" id="coppaFax_input" value="`, Htmlspecialchars(a.Setting("coppaFax")), `" size="22" maxlength="35" />
							</td>
						</tr><tr class="windowbg2" valign="top">
							<th width="50%" align="right">
								<label for="coppaPhone_input">`, c.Txt("admin_setting_coppaPhone"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=coppaPost" onclick="return reqWin(this.href);">?</a>)</span>:
								<div class="smalltext" style="font-weight: normal;">`, c.Txt("admin_setting_coppaPost_desc"), `</div>
							</th>
							<td width="50%" align="left">
								<input type="text" name="coppaPhone" id="coppaPhone_input" value="`, Htmlspecialchars(a.Setting("coppaPhone")), `" size="22" maxlength="35" />
							</td>
						</tr><tr class="windowbg2">
							<td width="100%" colspan="3" align="right">
								<input type="submit" name="save" value="`, c.Txt("10"), `" />
								<input type="hidden" name="sa" value="settings" />
							</td>
						</tr>
					</table>
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[`)
	if a.SettingEmpty("coppaAge") || a.SettingEmpty("coppaType") {
		c.O(`
						document.getElementById('coppaPost_input').disabled = true;
						document.getElementById('coppaFax_input').disabled = true;
						document.getElementById('coppaPhone_input').disabled = true;`)
	}
	if a.SettingEmpty("coppaAge") {
		c.O(`
						document.getElementById('coppaType_select').disabled = true;`)
	}
	c.O(`
					// ]]></script>
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// settingOrBlank returns the setting value, or "" if empty.
func settingOrBlank(a *App, name string) string {
	if a.SettingEmpty(name) {
		return ""
	}
	return a.Setting(name)
}

// coppaTypeSel returns the selected attr for the coppaType option.
func coppaTypeSel(a *App, v int) string {
	cur := a.SettingInt("coppaType")
	if v == 0 && a.SettingEmpty("coppaType") {
		return ` selected="selected"`
	}
	if v != 0 && cur == v {
		return ` selected="selected"`
	}
	return ""
}
