package app

// Hand-ports of Themes/default/Register.template.php (template_before,
// template_after, template_coppa, template_coppa_form,
// template_verification_sound) and the activation sub-templates from
// Login.template.php (template_retry_activate, template_resend).
// template_admin_register ports with Phase 7.

// templateRegisterBefore is template_before().
func templateRegisterBefore(c *Ctx) {
	page := c.Page.(*RegisterCtx)
	scripturl := c.App.ScriptURL
	tabindex := 1
	nextTab := func() int { tabindex++; return tabindex - 1 }

	// Make sure they've agreed to the terms and conditions.
	c.O(`
<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
	function verifyAgree()
	{
		if (document.forms.creator.passwrd1.value != document.forms.creator.passwrd2.value)
		{
			alert("`, c.Txt("register_passwords_differ_js"), `");
			return false;
		}`)

	// If they haven't checked the "I agree" box, tell them and don't submit.
	if page.RequireAgreement {
		c.O(`

		if (!document.forms.creator.regagree.checked)
		{
			alert("`, c.Txt("register_agree"), `");
			return false;
		}`)
	}

	// Otherwise, let it through.
	c.O(`

		return true;
	}`)

	if page.RequireAgreement {
		c.O(`
	function checkAgree()
	{
		document.forms.creator.regSubmit.disabled = isEmptyText(document.forms.creator.user) || isEmptyText(document.forms.creator.email) || isEmptyText(document.forms.creator.passwrd1) || !document.forms.creator.regagree.checked;
		setTimeout("checkAgree();", 1000);
	}
	setTimeout("checkAgree();", 1000);`)
	}

	if page.VisualVerification {
		c.O(`
	function refreshImages()
	{
		// Make sure we are using a new rand code.
		var new_url = new String("`, page.VerificationImageHref, `");
		new_url = new_url.substr(0, new_url.indexOf("rand=") + 5);

		// Quick and dirty way of converting decimal to hex
		var hexstr = "0123456789abcdef";
		for(var i=0; i < 32; i++)
			new_url = new_url + hexstr.substr(Math.floor(Math.random() * 16), 1);`)

		if page.UseGraphicLibrary {
			c.O(`
		document.getElementById("verificiation_image").src = new_url;`)
		} else {
			c.O(`
		document.getElementById("verificiation_image_1").src = new_url + ";letter=1";
		document.getElementById("verificiation_image_2").src = new_url + ";letter=2";
		document.getElementById("verificiation_image_3").src = new_url + ";letter=3";
		document.getElementById("verificiation_image_4").src = new_url + ";letter=4";
		document.getElementById("verificiation_image_5").src = new_url + ";letter=5";`)
		}
		c.O(`
	}`)
	}

	c.O(`
// ]]></script>

<form action="`, scripturl, `?action=register2" method="post" accept-charset="`, c.CharacterSet, `" name="creator" id="creator" onsubmit="return verifyAgree();">
	<table border="0" width="100%" cellpadding="3" cellspacing="0" class="tborder">
		<tr class="titlebg">
			<td>`, c.Txt("97"), ` - `, c.Txt("517"), `</td>
		</tr><tr class="windowbg">
			<td width="100%">
				<table cellpadding="3" cellspacing="0" border="0" width="100%">
					<tr>
						<td width="40%">
							<b>`, c.Txt("98"), `:</b>
							<div class="smalltext">`, c.Txt("520"), `</div>
						</td>
						<td>
							<input type="text" name="user" size="20" tabindex="`, nextTab(), `" maxlength="25" />
						</td>
					</tr><tr>
						<td width="40%">
							<b>`, c.Txt("69"), `:</b>
							<div class="smalltext">`, c.Txt("679"), `</div>
						</td>
						<td>
							<input type="text" name="email" size="30" tabindex="`, nextTab(), `" />`)

	// Are they allowed to hide their email?
	if page.AllowHideEmail {
		c.O(`
							<label for="hideEmail"><input type="checkbox" name="hideEmail" id="hideEmail" class="check" /> `, c.Txt("721"), `</label>`)
	}

	c.O(`
						</td>
					</tr><tr>
						<td width="40%">
							<b>`, c.Txt("81"), `:</b>
						</td>
						<td>
							<input type="password" name="passwrd1" size="30" tabindex="`, nextTab(), `" />
						</td>
					</tr><tr>
						<td width="40%">
							<b>`, c.Txt("82"), `:</b>
						</td>
						<td>
							<input type="password" name="passwrd2" size="30" tabindex="`, nextTab(), `" />
						</td>
					</tr>`)

	if page.VisualVerification {
		c.O(`
					<tr valign="top">
						<td width="40%" valign="top">
							<b>`, c.Txt("visual_verification_label"), `:</b>
							<div class="smalltext">`, c.Txt("visual_verification_description"), `</div>
						</td>
						<td>`)
		if page.UseGraphicLibrary {
			c.O(`
							<img src="`, page.VerificationImageHref, `" alt="`, c.Txt("visual_verification_description"), `" id="verificiation_image" /><br />`)
		} else {
			c.O(`
							<img src="`, page.VerificationImageHref, `;letter=1" alt="`, c.Txt("visual_verification_description"), `" id="verificiation_image_1" />
							<img src="`, page.VerificationImageHref, `;letter=2" alt="`, c.Txt("visual_verification_description"), `" id="verificiation_image_2" />
							<img src="`, page.VerificationImageHref, `;letter=3" alt="`, c.Txt("visual_verification_description"), `" id="verificiation_image_3" />
							<img src="`, page.VerificationImageHref, `;letter=4" alt="`, c.Txt("visual_verification_description"), `" id="verificiation_image_4" />
							<img src="`, page.VerificationImageHref, `;letter=5" alt="`, c.Txt("visual_verification_description"), `" id="verificiation_image_5" />`)
		}
		c.O(`
							<input type="text" name="visual_verification_code" size="30" tabindex="`, nextTab(), `" />
							<div class="smalltext">
								<a href="`, page.VerificationImageHref, `;sound" onclick="return reqWin(this.href, 400, 120);">`, c.Txt("visual_verification_sound"), `</a> | <a href="`, scripturl, `?action=register" onclick="refreshImages(); return false;">`, c.Txt("visual_verification_request_new"), `</a>
							</div>
						</td>
					</tr>`)
	}

	// Are there age restrictions in place?
	if page.ShowCoppa {
		c.O(`
					<tr>
						<td colspan="2" align="center" style="padding-top: 1ex;">
							<label for="skip_coppa"><input type="checkbox" name="skip_coppa" id="skip_coppa" tabindex="`, nextTab(), `" class="check" /> <b>`, page.CoppaDesc, `.</b></label>
						</td>
					</tr>`)
	}

	c.O(`
				</table>
			</td>
		</tr>
	</table>`)

	// Require them to agree here?
	if page.RequireAgreement {
		c.O(`
	<table width="100%" align="center" border="0" cellspacing="0" cellpadding="5" class="tborder" style="border-top: 0;">
		<tr>
			<td class="windowbg2" style="padding-top: 8px; padding-bottom: 8px;">
				`, page.Agreement, `
			</td>
		</tr><tr>
			<td align="center" class="windowbg2">
				<label for="regagree"><input type="checkbox" name="regagree" onclick="checkAgree();" id="regagree" class="check" /> <b>`, c.Txt("585"), `</b></label>
			</td>
		</tr>
	</table>`)
	}

	c.O(`
	<br />
	<div align="center">
		<input type="submit" name="regSubmit" value="`, c.Txt("97"), `" />
	</div>
</form>`)

	// Uncheck the agreement thing....
	if page.RequireAgreement {
		c.O(`
<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
	document.forms.creator.regagree.checked = false;
	document.forms.creator.regSubmit.disabled = !document.forms.creator.regagree.checked;
// ]]></script>`)
	}
}

// templateRegisterAfter is template_after().
func templateRegisterAfter(c *Ctx) {
	page := c.Page.(*RegisterCtx)
	c.O(`
		<br />
		<table border="0" width="80%" cellpadding="3" cellspacing="0" class="tborder" align="center">
			<tr class="titlebg">
				<td>`, c.PageTitle, `</td>
			</tr><tr class="windowbg">
				<td align="left">`, page.Description, `<br /><br /></td>
			</tr>
		</table>
		<br />`)
}

// templateCoppa is template_coppa().
func templateCoppa(c *Ctx) {
	page := c.Page.(*RegisterCtx)
	scripturl := c.App.ScriptURL

	// Formulate a nice complicated message!
	c.O(`
		<br />
		<table width="60%" cellpadding="4" cellspacing="0" border="0" class="tborder" align="center">
			<tr class="titlebg">
				<td>`, c.PageTitle, `</td>
			</tr><tr class="windowbg">
				<td align="left">`, page.CoppaBody, `<br /></td>
			</tr><tr class="windowbg">
				<td align="center">
					<a href="`, scripturl, `?action=coppa;form;member=`, page.CoppaMemberID, `" target="_blank">`, c.Txt("coppa_form_link_popup"), `</a>&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;<a href="`, scripturl, `?action=coppa;form;dl;member=`, page.CoppaMemberID, `">`, c.Txt("coppa_form_link_download"), `</a><br /><br />
				</td>
			</tr><tr class="windowbg">
				<td align="left">`)
	if page.CoppaManyOptions {
		c.O(c.Txt("coppa_send_to_two_options"))
	} else {
		c.O(c.Txt("coppa_send_to_one_option"))
	}
	c.O(`</td>
			</tr>`)

	// Can they send by post?
	if page.CoppaPost != "" {
		c.O(`
			<tr class="windowbg">
				<td align="left"><b>1) `, c.Txt("coppa_send_by_post"), `</b></td>
			</tr><tr class="windowbg">
				<td align="left" style="padding-bottom: 1ex;">
					<div style="padding: 4px; width: 32ex; background-color: white; color: black; margin-left: 5ex; border: 1px solid black;">
						`, page.CoppaPost, `
					</div>
				</td>
			</tr>`)
	}

	// Can they send by fax??
	if page.CoppaFax != "" {
		num := "1"
		if page.CoppaPost != "" {
			num = "2"
		}
		c.O(`
			<tr class="windowbg">
				<td align="left"><b>`, num, `) `, c.Txt("coppa_send_by_fax"), `</b></td>
			</tr><tr class="windowbg">
				<td align="left" style="padding-bottom: 1ex;">
					<div style="padding: 4px; width: 32ex; background-color: white; color: black; margin-left: 5ex; border: 1px solid black;">
						`, page.CoppaFax, `
					</div>
				</td>
			</tr>`)
	}

	// Offer an alternative Phone Number?
	if page.CoppaPhone != "" {
		c.O(`
			<tr class="windowbg" style="padding-bottom: 1ex;">
				<td align="left">`, page.CoppaPhone, `</td>
			</tr>`)
	}
	c.O(`
		</table>
		<br />`)
}

// templateCoppaForm is template_coppa_form().
func templateCoppaForm(c *Ctx) {
	page := c.Page.(*RegisterCtx)
	ul := `<u>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</u>`

	c.O(`
		<table border="0" width="100%" cellpadding="3" cellspacing="0" class="tborder" align="center">
			<tr>
				<td align="left">`, page.ForumContacts, `</td>
			</tr><tr>
				<td align="right">
					<i>`, c.Txt("coppa_form_address"), `</i>: `, ul, `<br />
					`, ul, `<br />
					`, ul, `<br />
					`, ul, `
				</td>
			</tr><tr>
				<td align="right">
					<i>`, c.Txt("coppa_form_date"), `</i>: `, ul, `
					<br /><br />
				</td>
			</tr><tr>
				<td align="left">
					`, page.CoppaBody, `
				</td>
			</tr>
		</table>
		<br />`)
}

// templateVerificationSound is template_verification_sound().
func templateVerificationSound(c *Ctx) {
	page := c.Page.(*RegisterCtx)

	c.O(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"`)
	if c.RightToLeft {
		c.O(` dir="rtl"`)
	}
	c.O(`>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=`, c.CharacterSet, `" />
		<title>`, c.PageTitle, `</title>
		<link rel="stylesheet" type="text/css" href="`, c.Theme.ThemeURL(), `/style.css" />
		<style type="text/css">`)

	if c.Browser.NeedsSizeFix {
		c.O(`
			@import(`, c.Theme.DefaultThemeURL(), `/fonts-compat.css);`)
	}

	c.O(`
		</style>
	</head>
	<body style="margin: 1ex;">
		<div class="popuptext">`)
	if c.Browser.IsIE {
		c.O(`
			<object classid="clsid:22D6F312-B0F6-11D0-94AB-0080C74C7E95" type="audio/x-wav">
				<param name="AutoStart" value="1" />
				<param name="FileName" value="`, page.VerificationSoundHref, `;format=.wav" />
			</object>`)
	} else {
		c.O(`
			<object type="audio/x-wav" data="`, page.VerificationSoundHref, `;format=.wav">
				<a href="`, page.VerificationSoundHref, `;format=.wav">`, page.VerificationSoundHref, `;format=.wav</a>
			</object>`)
	}
	c.O(`
			<br />
			<a href="`, page.VerificationSoundHref, `;sound">`, c.Txt("visual_verification_sound_again"), `</a><br />
			<a href="javascript:self.close();">`, c.Txt("visual_verification_sound_close"), `</a><br />
			<a href="`, page.VerificationSoundHref, `;format=.wav">`, c.Txt("visual_verification_sound_direct"), `</a>
		</div>
	</body>
</html>`)
}

// templateRetryActivate is template_retry_activate() (Login.template.php).
func templateRetryActivate(c *Ctx) {
	page := c.Page.(*RegisterCtx)
	scripturl := c.App.ScriptURL

	// Just ask them for their code so they can try it again...
	c.O(`
		<br />
		<form action="`, scripturl, `?action=activate;u=`, page.MemberID, `" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" width="600" cellpadding="4" cellspacing="0" class="tborder" align="center">
				<tr class="titlebg">
					<td colspan="2">`, c.PageTitle, `</td>`)

	// You didn't even have an ID?
	if page.MemberID == 0 {
		c.O(`
				</tr><tr class="windowbg">
					<td align="right" width="40%">`, c.Txt("invalid_activation_username"), `:</td>
					<td><input type="text" name="user" size="30" /></td>`)
	}

	c.O(`
				</tr><tr class="windowbg">
					<td align="right" width="40%">`, c.Txt("invalid_activation_retry"), `:</td>
					<td><input type="text" name="code" size="30" /></td>
				</tr><tr class="windowbg">
					<td colspan="2" align="center" style="padding: 1ex;"><input type="submit" value="`, c.Txt("invalid_activation_submit"), `" /></td>
				</tr>
			</table>
		</form>`)
}

// templateResendActivate is template_resend() (Login.template.php).
func templateResendActivate(c *Ctx) {
	page := c.Page.(*RegisterCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<br />
		<form action="`, scripturl, `?action=activate;sa=resend" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" width="600" cellpadding="4" cellspacing="0" class="tborder" align="center">
				<tr class="titlebg">
					<td colspan="2">`, c.PageTitle, `</td>
				</tr><tr class="windowbg">
					<td align="right" width="40%">`, c.Txt("invalid_activation_username"), `:</td>
					<td><input type="text" name="user" size="40" value="`, page.DefaultUsername, `" /></td>
				</tr><tr class="windowbg">
					<td colspan="2" style="padding-top: 3ex; padding-left: 3ex;">`, c.Txt("invalid_activation_new"), `</td>
				</tr><tr class="windowbg">
					<td align="right" width="40%">`, c.Txt("invalid_activation_new_email"), `:</td>
					<td><input type="text" name="new_email" size="40" /></td>
				</tr><tr class="windowbg">
					<td align="right" width="40%">`, c.Txt("invalid_activation_password"), `:</td>
					<td><input type="password" name="passwd" size="30" /></td>
				</tr><tr class="windowbg">`)

	if page.CanActivate {
		c.O(`
					<td colspan="2" style="padding-top: 3ex; padding-left: 3ex;">`, c.Txt("invalid_activation_known"), `</td>
				</tr><tr class="windowbg">
					<td align="right" width="40%">`, c.Txt("invalid_activation_retry"), `:</td>
					<td><input type="text" name="code" size="30" /></td>
				</tr><tr class="windowbg">`)
	}

	c.O(`
					<td colspan="2" align="center" style="padding: 1ex;"><input type="submit" value="`, c.Txt("invalid_activation_resend"), `" /></td>
				</tr>
			</table>
		</form>`)
}
