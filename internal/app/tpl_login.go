package app

// Hand-port of template_login() from Themes/default/Login.template.php.
// (template_kick_guest lives in tpl_errors.go since Phase 1 needed it.)

func templateLogin(c *Ctx) {
	page, _ := c.Page.(*LoginCtx)
	if page == nil {
		page = &LoginCtx{}
	}
	scripturl := c.App.ScriptURL

	c.O(`
		<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/sha1.js"></script>

		<form action="`, scripturl, `?action=login2" method="post" accept-charset="`, c.CharacterSet, `" name="frmLogin" id="frmLogin" style="margin-top: 4ex;"`)
	if !c.DisableLoginHash {
		c.O(` onsubmit="hashLoginPassword(this, '`, c.Sc, `');"`)
	}
	c.O(`>
			<table border="0" width="400" cellspacing="0" cellpadding="4" class="tborder" align="center">
				<tr class="titlebg">
					<td colspan="2">
						<img src="`, c.Theme.ImagesURL(), `/icons/login_sm.gif" alt="" align="top" /> `, c.Txt("34"), `
					</td>`)

	// Did they make a mistake last time?
	if page.HasError {
		c.O(`
				</tr><tr class="windowbg">
				<td align="center" colspan="2" style="padding: 1ex;">
						<b style="color: red;">`, page.LoginError, `</b>
					</td>`)
	}

	// Or perhaps there's some special description for this time?
	if page.Description != "" {
		c.O(`
				</tr><tr class="windowbg">
					<td align="center" colspan="2">
						<b>`, page.Description, `</b><br />
						<br />
					</td>`)
	}

	// Now just get the basic information - username, password, etc.
	cookieTime := c.App.Setting("cookieTime")
	c.O(`
				</tr><tr class="windowbg">
					<td width="50%" align="right"><b>`, c.Txt("35"), `:</b></td>
					<td><input type="text" name="user" size="20" value="`, page.DefaultUsername, `" /></td>
				</tr><tr class="windowbg">
					<td align="right"><b>`, c.Txt("36"), `:</b></td>
					<td><input type="password" name="passwrd" value="`, page.DefaultPassword, `" size="20" /></td>
				</tr><tr class="windowbg">
					<td align="right"><b>`, c.Txt("497"), `:</b></td>
					<td><input type="text" name="cookielength" size="4" maxlength="4" value="`, cookieTime, `"`)
	if page.NeverExpire {
		c.O(` disabled="disabled"`)
	}
	c.O(` /></td>
				</tr><tr class="windowbg">
					<td align="right"><b>`, c.Txt("508"), `:</b></td>
					<td><input type="checkbox" name="cookieneverexp"`)
	if page.NeverExpire {
		c.O(` checked="checked"`)
	}
	c.O(` class="check" onclick="this.form.cookielength.disabled = this.checked;" /></td>
				</tr><tr class="windowbg">`)

	// If they have deleted their account, give them a chance to change their
	// mind.
	if page.ShowUndelete {
		c.O(`
					<td align="right"><b style="color: red;">`, c.Txt("undelete_account"), `:</b></td>
					<td><input type="checkbox" name="undelete" class="check" /></td>
				</tr><tr class="windowbg">`)
	}
	c.O(`
					<td align="center" colspan="2"><input type="submit" value="`, c.Txt("34"), `" style="margin-top: 2ex;" /></td>
				</tr><tr class="windowbg">
					<td align="center" colspan="2" class="smalltext"><a href="`, scripturl, `?action=reminder">`, c.Txt("315"), `</a><br /><br /></td>
				</tr>
			</table>

			<input type="hidden" name="hash_passwrd" value="" />
		</form>`)

	// Focus on the correct input - username or password.
	focus := "user"
	if page.DefaultUsername != "" {
		focus = "passwrd"
	}
	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			document.forms.frmLogin.`, focus, `.focus();
		// ]]></script>`)
}
