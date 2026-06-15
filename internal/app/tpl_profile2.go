package app

// Hand-port of Themes/default/Profile.template.php part 2: the edit forms —
// template_account, template_forumProfile, template_theme,
// template_notification, template_pmprefs, template_deleteAccount,
// template_profile_save and template_error_message.

import "strings"

// templateProfileErrorMessage is template_error_message().
func templateProfileErrorMessage(c *Ctx, page *ProfileCtx) {
	c.O(`
		<div class="windowbg" style="margin: 1ex; padding: 1ex 2ex; border: 1px dashed red; color: red;">
			<span style="text-decoration: underline;">`, c.Txt("profile_errors_occurred"), `:</span>
			<ul>`)

	// Cycle through each error and display an error message.
	for _, e := range page.ModifyErrorList {
		c.O(`
				<li>`, c.Txt("profile_error_"+e), `.</li>`)
	}

	c.O(`
			</ul>
		</div>`)
}

// templateProfileSave is template_profile_save().
func templateProfileSave(c *Ctx, page *ProfileCtx) {
	c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr><tr>`)

	// Only show the password box if it's actually needed.
	if page.IsOwner && page.RequirePass {
		passColor := ""
		if page.ModifyError["bad_password"] || page.ModifyError["no_password"] {
			passColor = ` style="color: red;"`
		}
		c.O(`
								<td width="40%">
									<b`, passColor, `>`, c.Txt("smf241"), `: </b>
									<div class="smalltext">`, c.Txt("smf244"), `</div>
								</td>
								<td>
									<input type="password" name="oldpasswrd" size="20" style="margin-right: 4ex;" />`)
	} else {
		c.O(`
								<td align="right" colspan="2">`)
	}

	c.O(`
									<input type="submit" value="`, c.Txt("88"), `" />
									<input type="hidden" name="sc" value="`, c.Sc, `" />
									<input type="hidden" name="userID" value="`, page.MemID, `" />
									<input type="hidden" name="sa" value="`, page.MenuSelected, `" />
								</td>
							</tr>`)
}

// templateProfileAccount is template_account().
func templateProfileAccount(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileAccountCtx)
	scripturl := c.App.ScriptURL

	// Javascript for checking if password has been entered / taking admin
	// powers away from themselves.
	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function checkProfileSubmit()
			{`)

	// If this part requires a password, make sure to give a warning.
	if page.IsOwner && page.RequirePass {
		c.O(`
				// Did you forget to type your password?
				if (document.forms.creator.oldpasswrd.value == "")
				{
					alert("`, c.Txt("smf244"), `");
					return false;
				}`)
	}

	// This part checks if they are removing themselves from administrative
	// power on accident.
	if sub.AllowEditMembergroups && page.IsOwner && page.Member.Group == 1 {
		c.O(`
				if (typeof(document.forms.creator.ID_GROUP) != "undefined" && document.forms.creator.ID_GROUP.value != "1")
					return confirm("`, c.Txt("deadmin_confirm"), `");`)
	}

	c.O(`
				return true;
			}
		// ]]></script>`)

	// The main containing header.
	c.O(`
		<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" name="creator" id="creator" onsubmit="return checkProfileSubmit();">
			<table border="0" width="85%" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
				<tr class="titlebg">
					<td height="26">
						&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;
						`, c.Txt("79"), `
					</td>
				</tr>`)

	// Display Name, language and date user registered.
	c.O(`
				<tr class="windowbg">
					<td class="smalltext" height="25" style="padding: 2ex;">
						`, c.Txt("account_info"), `
					</td>
				</tr>
				<tr>
					<td class="windowbg2" style="padding-bottom: 2ex;">
						<table width="100%" cellpadding="3" cellspacing="0" border="0">`)

	// Only show these settings if you're allowed to edit the account itself.
	if sub.AllowEditAccount {
		if c.User.IsAdmin && sub.AllowEditUsername {
			c.O(`
							<tr>
								<td colspan="2" align="center" style="color: red">`, c.Txt("username_warning"), `</td>
							</tr>
							<tr>
								<td width="40%">
									<b>`, c.Txt("35"), `: </b>
								</td>
								<td>
									<input type="text" name="memberName" size="30" value="`, page.Member.Username, `" />
								</td>
							</tr>`)
		} else {
			changeLink := ""
			if c.User.IsAdmin {
				changeLink = `
									<div class="smalltext">(<a href="` + scripturl + `?action=profile;u=` + itoa(page.MemID) + `;sa=account;changeusername" style="font-style: italic;">` + c.Txt("username_change") + `</a>)</div>`
			}
			c.O(`
							<tr>
								<td width="40%">
									<b>`, c.Txt("35"), `: </b>`, changeLink, `
								</td>
								<td>
									`, page.Member.Username, `
								</td>
							</tr>`)
		}

		nameColor := ""
		if page.ModifyError["no_name"] || page.ModifyError["name_taken"] {
			nameColor = ` style="color: red;"`
		}
		nameCell := page.Member.Name
		if sub.AllowEditName {
			nameCell = `<input type="text" name="realName" size="30" value="` + page.Member.Name + `" maxlength="60" />`
		}
		c.O(`
							<tr>
								<td>
									<b`, nameColor, `>`, c.Txt("68"), `: </b>
									<div class="smalltext">`, c.Txt("518"), `</div>
								</td>
								<td>`, nameCell, `</td>
							</tr>`)

		// Allow the administrator to change the date they registered on and
		// their post count.
		if c.User.IsAdmin {
			c.O(`
							<tr>
								<td><b>`, c.Txt("233"), `:</b></td>
								<td><input type="text" name="dateRegistered" size="30" value="`, page.Member.Registered, `" /></td>
							</tr>
							<tr>
								<td><b>`, c.Txt("86"), `: </b></td>
								<td><input type="text" name="posts" size="4" value="`, page.Member.Posts, `" /></td>
							</tr>`)
		}

		// (userLanguage select: single-language port.)
	}

	// Only display member group information/editing with the proper
	// permissions.
	if sub.AllowEditMembergroups {
		c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr><tr>
								<td valign="top">
									<b>`, c.Txt("primary_membergroup"), `: </b>
									<div class="smalltext">(<a href="`, scripturl, `?action=helpadmin;help=moderator_why_missing" onclick="return reqWin(this.href);">`, c.Txt("moderator_why_missing"), `</a>)</div>
								</td>
								<td>
									<select name="ID_GROUP">`)
		// Fill the select box with all primary member groups that can be
		// assigned to a member.
		for _, memberGroup := range sub.MemberGroups {
			sel := ""
			if memberGroup.IsPrimary {
				sel = ` selected="selected"`
			}
			c.O(`
										<option value="`, memberGroup.ID, `"`, sel, `>
											`, memberGroup.Name, `
										</option>`)
		}
		c.O(`
									</select>
								</td>
							</tr><tr>
								<td valign="top"><b>`, c.Txt("additional_membergroups"), `:</b></td>
								<td>
									<div id="additionalGroupsList">
										<input type="hidden" name="additionalGroups[]" value="0" />`)
		// For each membergroup show a checkbox so members can be assigned to
		// more than one group.
		for _, memberGroup := range sub.MemberGroups {
			if !memberGroup.CanBeAdditional {
				continue
			}
			checked := ""
			if memberGroup.IsAdditional {
				checked = ` checked="checked"`
			}
			c.O(`
										<label for="additionalGroups-`, memberGroup.ID, `"><input type="checkbox" name="additionalGroups[]" value="`, memberGroup.ID, `" id="additionalGroups-`, memberGroup.ID, `"`, checked, ` class="check" /> `, memberGroup.Name, `</label><br />`)
		}
		c.O(`
									</div>
									<a href="javascript:void(0);" onclick="document.getElementById('additionalGroupsList').style.display = 'block'; document.getElementById('additionalGroupsLink').style.display = 'none'; return false;" id="additionalGroupsLink" style="display: none;">`, c.Txt("additional_membergroups_show"), `</a>
									<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
										document.getElementById("additionalGroupsList").style.display = "none";
										document.getElementById("additionalGroupsLink").style.display = "";
									// ]]></script>
								</td>
							</tr>`)
	}

	// Show this part if you're not only here for assigning membergroups.
	if sub.AllowEditAccount {
		// Show email address box.
		emailColor := ""
		if page.ModifyError["bad_email"] || page.ModifyError["no_email"] || page.ModifyError["email_taken"] {
			emailColor = ` style="color: red;"`
		}
		c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr><tr>
								<td width="40%"><b`, emailColor, `>`, c.Txt("69"), `: </b><div class="smalltext">`, c.Txt("679"), `</div></td>
								<td><input type="text" name="emailAddress" size="30" value="`, page.Member.Email, `" /></td>
							</tr>`)

		// If the user is allowed to hide their email address from the public
		// give them the option to here.
		if sub.AllowHideEmail {
			checked := ""
			if page.Member.HideEmail != 0 {
				checked = ` checked="checked"`
			}
			c.O(`
							<tr>
								<td width="40%"><b>`, c.Txt("721"), `</b></td>
								<td><input type="hidden" name="hideEmail" value="0" /><input type="checkbox" name="hideEmail"`, checked, ` value="1" class="check" /></td>
							</tr>`)
		}

		// Option to show online status - if they are allowed to.
		if sub.AllowHideOnline {
			checked := ""
			if page.Member.ShowOnline != 0 {
				checked = ` checked="checked"`
			}
			c.O(`
							<tr>
								<td width="40%"><b>`, c.Txt("show_online"), `</b></td>
								<td><input type="hidden" name="showOnline" value="0" /><input type="checkbox" name="showOnline"`, checked, ` value="1" class="check" /></td>
							</tr>`)
		}

		// Show boxes so that the user may change his or her password.
		passColor := ""
		if page.ModifyError["bad_new_password"] {
			passColor = ` style="color: red;"`
		}
		c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr><tr>
								<td width="40%"><b`, passColor, `>`, c.Txt("81"), `: </b><div class="smalltext">`, c.Txt("596"), `</div></td>
								<td><input type="password" name="passwrd1" size="20" /></td>
							</tr><tr>
								<td width="40%"><b>`, c.Txt("82"), `: </b></td>
								<td><input type="password" name="passwrd2" size="20" /></td>
							</tr>`)

		// This section allows the user to enter secret question/answer so
		// they can reset a forgotten password.
		c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr><tr>
								<td width="40%"><b>`, c.Txt("pswd1"), `:</b><div class="smalltext">`, c.Txt("secret_desc"), `</div></td>
								<td><input type="text" name="secretQuestion" size="50" value="`, page.Member.SecretQuestion, `" /></td>
							</tr><tr>
								<td width="40%"><b>`, c.Txt("pswd2"), `:</b><div class="smalltext">`, c.Txt("secret_desc2"), `</div></td>
								<td><input type="text" name="secretAnswer" size="20" /><span class="smalltext" style="margin-left: 4ex;"><a href="`, scripturl, `?action=helpadmin;help=secret_why_blank" onclick="return reqWin(this.href);">`, c.Txt("secret_why_blank"), `</a></span></td>
							</tr>`)
	}
	// Show the standard "Save Settings" profile button.
	templateProfileSave(c, page)

	c.O(`
						</table>
					</td>
				</tr>
			</table>
		</form>`)
}

// templateProfileForumProfile is template_forumProfile().
func templateProfileForumProfile(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileForumCtx)
	a := c.App
	scripturl := a.ScriptURL
	av := &page.Member.Avatar

	// The main containing header.
	c.O(`
		<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" name="creator" id="creator" enctype="multipart/form-data">
			<table border="0" width="85%" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
				<tr class="titlebg">
					<td height="26">
						&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;
						`, c.Txt("79"), `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" height="25" style="padding: 2ex;">
						`, c.Txt("forumProfile_info"), `
					</td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 2ex;">
						<table border="0" width="100%" cellpadding="5" cellspacing="0">`)

	// This is the avatar selection table that is only displayed if avatars
	// are enabled!
	if av.AllowServerStored || av.AllowUpload || av.AllowExternal {
		// If users are allowed to choose avatars stored on the server show
		// selection boxes to choice them from.
		if av.AllowServerStored {
			checked := ""
			if av.Choice == "server_stored" {
				checked = ` checked="checked"`
			}
			badColor := ""
			if page.ModifyError["bad_avatar"] {
				badColor = ` style="color: red;"`
			}
			previewSrc := a.Setting("avatar_url") + "/blank.gif"
			if av.AllowExternal && av.Choice == "external" {
				previewSrc = av.External
			}
			c.O(`
							<tr>
								<td width="40%" valign="top" style="padding: 0 2px;">
									<table width="100%" cellpadding="5" cellspacing="0" border="0" style="height: 25ex;"><tr>
										<td valign="top" width="20" class="windowbg"><input type="radio" name="avatar_choice" id="avatar_choice_server_stored" value="server_stored"`, checked, ` class="check" /></td>
										<td valign="top" style="padding-left: 1ex;">
											<b`, badColor, `><label for="avatar_choice_server_stored">`, c.Txt("229"), `:</label></b>
											<div style="margin: 2ex;"><img name="avatar" id="avatar" src="`, previewSrc, `" alt="Do Nothing" /></div>
										</td>
									</tr></table>
								</td>
								<td>
									<table width="100%" cellpadding="0" cellspacing="0" border="0"><tr>
										<td style="width: 20ex;">
											<select name="cat" id="cat" size="10" onchange="changeSel('');" onfocus="selectRadioByName(document.forms.creator.avatar_choice, 'server_stored');">`)
			// This lists all the file catergories.
			for _, avatar := range sub.Avatars {
				value := avatar.Filename
				if avatar.IsDir {
					value += "/"
				}
				sel := ""
				if avatar.Checked {
					sel = ` selected="selected"`
				}
				c.O(`
												<option value="`, value, `"`, sel, `>`, avatar.Name, `</option>`)
			}
			c.O(`
											</select>
										</td>
										<td>
											<select name="file" id="file" size="10" style="display: none;" onchange="showAvatar()" onfocus="selectRadioByName(document.forms.creator.avatar_choice, 'server_stored');" disabled="disabled"><option></option></select>
										</td>
									</tr></table>
								</td>
							</tr>`)
		}

		// If the user can link to an off server avatar, show them a box to
		// input the address.
		if av.AllowExternal {
			checked := ""
			if av.Choice == "external" {
				checked = ` checked="checked"`
			}
			c.O(`
							<tr>
								<td valign="top" style="padding: 0 2px;">
									<table width="100%" cellpadding="5" cellspacing="0" border="0"><tr>
										<td valign="top" width="20" class="windowbg"><input type="radio" name="avatar_choice" id="avatar_choice_external" value="external"`, checked, ` class="check" /></td>
										<td valign="top" style="padding-left: 1ex;"><b><label for="avatar_choice_external">`, c.Txt("475"), `:</label></b><div class="smalltext">`, c.Txt("474"), `</div></td>
									</tr></table>
								</td>
								<td valign="top">
									<input type="text" name="userpicpersonal" size="45" value="`, av.External, `" onfocus="selectRadioByName(document.forms.creator.avatar_choice, 'external');" onchange="if (typeof(previewExternalAvatar) != 'undefined') previewExternalAvatar(this.value);" />
								</td>
							</tr>`)
		}

		// If the user is able to upload avatars to the server show them an
		// upload box.
		if av.AllowUpload {
			checked := ""
			if av.Choice == "upload" {
				checked = ` checked="checked"`
			}
			current := ""
			if av.IDAttach > 0 {
				current = `<img src="` + av.Href + `" /><input type="hidden" name="ID_ATTACH" value="` + itoa(av.IDAttach) + `" /><br /><br />`
			}
			c.O(`
							<tr>
								<td valign="top" style="padding: 0 2px;">
									<table width="100%" cellpadding="5" cellspacing="0" border="0"><tr>
										<td valign="top" width="20" class="windowbg"><input type="radio" name="avatar_choice" id="avatar_choice_upload" value="upload"`, checked, ` class="check" /></td>
										<td valign="top" style="padding-left: 1ex;"><b><label for="avatar_choice_upload">`, c.Txt("avatar_will_upload"), `:</label></b></td>
									</tr></table>
								</td>
								<td valign="top">
									`, current, `
									<input type="file" size="48" name="attachment" value="" onfocus="selectRadioByName(document.forms.creator.avatar_choice, 'upload');" />
								</td>
							</tr>`)
		}
	}

	// Personal text...
	c.O(`
							<tr>
								<td width="40%"><b>`, c.Txt("228"), `: </b></td>
								<td><input type="text" name="personalText" size="50" maxlength="50" value="`, page.Member.Blurb, `" /></td>
							</tr>
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr>`)

	// Gender, birthdate and location.
	maleSel := ""
	if page.Member.GenderName == "m" {
		maleSel = ` selected="selected"`
	}
	femaleSel := ""
	if page.Member.GenderName == "f" {
		femaleSel = ` selected="selected"`
	}
	c.O(`
							<tr>
								<td width="40%">
									<b>`, c.Txt("563"), `:</b>
									<div class="smalltext">`, c.Txt("566"), ` - `, c.Txt("564"), ` - `, c.Txt("565"), `</div>
								</td>
								<td class="smalltext">
									<input type="text" name="bday3" size="4" maxlength="4" value="`, page.Member.BirthYear, `" /> -
									<input type="text" name="bday1" size="2" maxlength="2" value="`, page.Member.BirthMonth, `" /> -
									<input type="text" name="bday2" size="2" maxlength="2" value="`, page.Member.BirthDay, `" />
								</td>
							</tr><tr>
								<td width="40%"><b>`, c.Txt("227"), `: </b></td>
								<td><input type="text" name="location" size="50" value="`, page.Member.Location, `" /></td>
							</tr>
							<tr>
								<td width="40%"><b>`, c.Txt("231"), `: </b></td>
								<td>
									<select name="gender" size="1">
										<option value="0"></option>
										<option value="1"`, maleSel, `>`, c.Txt("238"), `</option>
										<option value="2"`, femaleSel, `>`, c.Txt("239"), `</option>
									</select>
								</td>
							</tr><tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr>`)

	// Input box for custom titles, if they can edit it...
	if !a.SettingEmpty("titlesEnable") && sub.AllowEditTitle {
		c.O(`
							<tr>
								<td width="40%"><b>` + c.Txt("title1") + `: </b></td>
								<td><input type="text" name="usertitle" size="50" value="` + page.Member.Title + `" /></td>
							</tr>`)
	}

	// Show the signature box.
	c.O(`
							<tr>
								<td width="40%" valign="top">
									<b>`, c.Txt("85"), `:</b>
									<div class="smalltext">`, c.Txt("606"), `</div><br />
									<br />`)

	c.O(`
								</td>
								<td>
									<textarea class="editor" onkeyup="calcCharLeft();" name="signature" rows="5" cols="50">`, page.Member.Signature, `</textarea><br />`)

	// If there is a limit at all!
	if sub.MaxSignatureLength != 0 {
		c.O(`
									<span class="smalltext">`, c.Txt("664"), ` <span id="signatureLeft">`, sub.MaxSignatureLength, `</span></span>`)
	}

	// Some javascript used to count how many characters have been used so
	// far in the signature.
	c.O(`
									<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
										function tick()
										{
											if (typeof(document.forms.creator) != "undefined")
											{
												calcCharLeft();
												setTimeout("tick()", 1000);
											}
											else
												setTimeout("tick()", 800);
										}

										function calcCharLeft()
										{
											var maxLength = `, sub.MaxSignatureLength, `;
											var oldSignature = "", currentSignature = document.forms.creator.signature.value;

											if (!document.getElementById("signatureLeft"))
												return;

											if (oldSignature != currentSignature)
											{
												oldSignature = currentSignature;

												if (currentSignature.replace(/\r/, "").length > maxLength)
													document.forms.creator.signature.value = currentSignature.replace(/\r/, "").substring(0, maxLength);
												currentSignature = document.forms.creator.signature.value.replace(/\r/, "");
											}

											setInnerHTML(document.getElementById("signatureLeft"), maxLength - currentSignature.length);
										}

										setTimeout("tick()", 800);
									// ]]></script>
								</td>
							</tr>`)

	// Website details.
	c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr>
							<tr>
								<td width="40%"><b>`, c.Txt("83"), `: </b><div class="smalltext">`, c.Txt("598"), `</div></td>
								<td><input type="text" name="websiteTitle" size="50" value="`, page.Member.WebsiteTitle, `" /></td>
							</tr><tr>
								<td width="40%"><b>`, c.Txt("84"), `: </b><div class="smalltext">`, c.Txt("599"), `</div></td>
								<td><input type="text" name="websiteUrl" size="50" value="`, page.Member.WebsiteURL, `" /></td>
							</tr>`)

	// Show the standard "Save Settings" profile button.
	templateProfileSave(c, page)

	c.O(`
						</table>
					</td>
				</tr>
			</table>`)

	// Avatar-listing javascript.
	if av.AllowServerStored {
		var avatarList []string
		var collect func(entries []*ProfileAvatarEntry, dir string)
		collect = func(entries []*ProfileAvatarEntry, dir string) {
			for _, e := range entries {
				if e.IsDir {
					collect(e.Files, e.Filename)
				} else if dir != "" {
					avatarList = append(avatarList, dir+"/"+e.Filename)
				}
			}
		}
		collect(sub.Avatars, "")

		maxExternalHeight := "0"
		if !a.SettingEmpty("avatar_max_height_external") {
			maxExternalHeight = a.Setting("avatar_max_height_external")
		}
		maxExternalWidth := "0"
		if !a.SettingEmpty("avatar_max_width_external") {
			maxExternalWidth = a.Setting("avatar_max_width_external")
		}
		c.O(`
			<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
				var files = ["`+strings.Join(avatarList, `", "`)+`"];
				var avatar = document.getElementById("avatar");
				var cat = document.getElementById("cat");
				var selavatar = "`+sub.AvatarSelected+`";
				var avatardir = "`+a.Setting("avatar_url")+`/";
				var size = avatar.alt.substr(3, 2) + " " + avatar.alt.substr(0, 2) + String.fromCharCode(117, 98, 116);
				var file = document.getElementById("file");

				if (avatar.src.indexOf("blank.gif") > -1)
					changeSel(selavatar);
				else
					previewExternalAvatar(avatar.src)

				function changeSel(selected)
				{
					if (cat.selectedIndex == -1)
						return;

					if (cat.options[cat.selectedIndex].value.indexOf("/") > 0)
					{
						var i;
						var count = 0;

						file.style.display = "inline";
						file.disabled = false;

						for (i = file.length; i >= 0; i = i - 1)
							file.options[i] = null;

						for (i = 0; i < files.length; i++)
							if (files[i].indexOf(cat.options[cat.selectedIndex].value) == 0)
							{
								var filename = files[i].substr(files[i].indexOf("/") + 1);
								var showFilename = filename.substr(0, filename.lastIndexOf("."));
								showFilename = showFilename.replace(/[_]/g, " ");

								file.options[count] = new Option(showFilename, files[i]);

								if (filename == selected)
								{
									if (file.options.defaultSelected)
										file.options[count].defaultSelected = true;
									else
										file.options[count].selected = true;
								}

								count++;
							}

						if (file.selectedIndex == -1 && file.options[0])
							file.options[0].selected = true;

						showAvatar();
					}
					else
					{
						file.style.display = "none";
						file.disabled = true;
						document.getElementById("avatar").src = avatardir + cat.options[cat.selectedIndex].value;
						document.getElementById("avatar").style.width = "";
						document.getElementById("avatar").style.height = "";
					}
				}

				function showAvatar()
				{
					if (file.selectedIndex == -1)
						return;

					document.getElementById("avatar").src = avatardir + file.options[file.selectedIndex].value;
					document.getElementById("avatar").alt = file.options[file.selectedIndex].text;
					document.getElementById("avatar").alt += file.options[file.selectedIndex].text == size ? "!" : "";
					document.getElementById("avatar").style.width = "";
					document.getElementById("avatar").style.height = "";
				}

				function previewExternalAvatar(src)
				{
					if (!document.getElementById("avatar"))
						return;

					var maxHeight = `, maxExternalHeight, `;
					var maxWidth = `, maxExternalWidth, `;
					var tempImage = new Image();

					tempImage.src = src;
					if (maxWidth != 0 && tempImage.width > maxWidth)
					{
						document.getElementById("avatar").style.height = parseInt((maxWidth * tempImage.height) / tempImage.width) + "px";
						document.getElementById("avatar").style.width = maxWidth + "px";
					}
					else if (maxHeight != 0 && tempImage.height > maxHeight)
					{
						document.getElementById("avatar").style.width = parseInt((maxHeight * tempImage.width) / tempImage.height) + "px";
						document.getElementById("avatar").style.height = maxHeight + "px";
					}
					document.getElementById("avatar").src = src;
				}
			// ]]></script>`)
	}
	c.O(`
		</form>`)
}

// templateProfileThemeSettings is template_theme().
func templateProfileThemeSettings(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileThemeCtx)
	a := c.App
	scripturl := a.ScriptURL
	opts := page.Member.Options

	c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		var localTime = new Date();
		var serverTime = new Date("`, sub.CurrentForumTime, `");

		function autoDetectTimeOffset()
		{
			// Get the difference between the two, set it up so that the sign will tell us who is ahead of who
			var diff = Math.round((localTime.getTime() - serverTime.getTime())/3600000);

			// Make sure we are limiting this to one day's difference
			diff %= 24;

			document.forms.creator.timeOffset.value = diff;
		}
	// ]]></script>`)

	// The main containing header.
	c.O(`
		<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" name="creator" id="creator">
			<table border="0" width="85%" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
				<tr class="titlebg">
					<td height="26">
						&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" border="0" align="top" />&nbsp;
						`, c.Txt("79"), `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" height="25" style="padding: 2ex;">
						`, c.Txt("theme_info"), `
					</td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 2ex;">
						<table border="0" width="100%" cellpadding="3">`)

	// Are they allowed to change their theme?
	if !a.SettingEmpty("theme_allow") || c.User.IsAdmin {
		c.O(`
							<tr>
								<td colspan="2" width="40%"><b>`, c.Txt("theme1a"), `:</b> `, page.Member.ThemeName, ` <a href="`, scripturl, `?action=theme;sa=pick;u=`, page.MemID, `;sesc=`, c.Sc, `">`, c.Txt("theme1b"), `</a></td>
							</tr>`)
	}

	// Are multiple smiley sets enabled?
	if !a.SettingEmpty("smiley_sets_enable") {
		smileyDefault := a.Setting("smiley_sets_default")
		if !c.Theme.Empty("smiley_sets_default") {
			smileyDefault = c.Theme.Get("smiley_sets_default")
		}
		c.O(`
							<tr>
								<td colspan="2" width="40%">
									<b>`, c.Txt("smileys_current"), `:</b>
									<select name="smileySet" onchange="document.getElementById('smileypr').src = this.selectedIndex == 0 ? '`, c.Theme.ImagesURL(), `/blank.gif' : '`, a.Setting("smileys_url"), `/' + (this.selectedIndex != 1 ? this.options[this.selectedIndex].value : '`, smileyDefault, `') + '/smiley.gif';">`)
		for _, set := range sub.SmileySets {
			sel := ""
			if set.Selected {
				sel = ` selected="selected"`
			}
			c.O(`
										<option value="`, set.ID, `"`, sel, `>`, set.Name, `</option>`)
		}
		previewSrc := c.Theme.ImagesURL() + "/blank.gif"
		if page.Member.SmileySetID != "none" {
			setID := page.Member.SmileySetID
			if setID == "" {
				setID = smileyDefault
			}
			previewSrc = a.Setting("smileys_url") + "/" + setID + "/smiley.gif"
		}
		c.O(`
									</select> <img id="smileypr" src="`, previewSrc, `" alt=":)" align="top" style="padding-left: 20px;" />
								</td>
							</tr>`)
	}

	if !a.SettingEmpty("theme_allow") || c.User.IsAdmin || !a.SettingEmpty("smiley_sets_enable") {
		c.O(`
							<tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr>`)
	}

	// Allow the user to change the way the time is displayed.
	helpAlign := "left"
	helpPad := "padding-right"
	if c.RightToLeft {
		helpAlign = "right"
		helpPad = "padding-left"
	}
	offsetColor := ""
	if page.ModifyError["bad_offset"] {
		offsetColor = ` style="color: red;"`
	}
	c.O(`
							<tr>
								<td width="40%">
									<b>`, c.Txt("486"), `:</b><br />
									<a href="`, scripturl, `?action=helpadmin;help=time_format" onclick="return reqWin(this.href);" class="help"><img src="`, c.Theme.ImagesURL(), `/helptopics.gif" alt="`, c.Txt("119"), `" align="`, helpAlign, `" style="`, helpPad, `: 1ex;" /></a>
									<span class="smalltext">`, c.Txt("479"), `</span>
								</td>
								<td>
									<select name="easyformat" onchange="document.forms.creator.timeFormat.value = this.options[this.selectedIndex].value;" style="margin-bottom: 4px;">`)
	// Help the user by showing a list of common time formats.
	for _, timeFormat := range sub.EasyTimeformats {
		sel := ""
		if timeFormat.Format == page.Member.TimeFormat {
			sel = ` selected="selected"`
		}
		c.O(`
										<option value="`, timeFormat.Format, `"`, sel, `>`, timeFormat.Title, `</option>`)
	}
	c.O(`
									</select><br />
									<input type="text" name="timeFormat" value="`, page.Member.TimeFormat, `" size="30" />
								</td>
							</tr><tr>
								<td width="40%"><b`, offsetColor, `>`, c.Txt("371"), `:</b><div class="smalltext">`, c.Txt("519"), `</div></td>
								<td class="smalltext"><input type="text" name="timeOffset" size="5" maxlength="5" value="`, page.Member.TimeOffset, `" /> <a href="javascript:void(0);" onclick="autoDetectTimeOffset(); return false;">`, c.Txt("timeoffset_autodetect"), `</a><br />`, c.Txt("741"), `: <i>`, sub.CurrentForumTime, `</i></td>
							</tr><tr>
								<td colspan="2"><hr width="100%" size="1" class="hrcolor" /></td>
							</tr>`)

	checkOpt := func(name, label string) {
		checked := ""
		if !empty(opts[name]) {
			checked = ` checked="checked"`
		}
		c.O(`
										<tr>
											<td colspan="2">
												<input type="hidden" name="default_options[`, name, `]" value="0" />
												<label for="`, name, `"><input type="checkbox" name="default_options[`, name, `]" id="`, name, `" value="1"`, checked, ` class="check" /> `, label, `</label>
											</td>
										</tr>`)
	}

	// The first checkbox row uses slightly different markup in PHP (no
	// wrapping <tr> indentation differences) — they are identical, so reuse.
	c.O(`
							<tr>
								<td colspan="2">
									<table width="100%" cellspacing="0" cellpadding="3">`)
	checkOpt("show_board_desc", c.Txt("732"))
	checkOpt("show_children", c.Txt("show_children"))
	checkOpt("show_no_avatars", c.Txt("show_no_avatars"))
	checkOpt("show_no_signatures", c.Txt("show_no_signatures"))

	if !c.Theme.Empty("allow_no_censored") {
		checkOpt("show_no_censored", c.Txt("show_no_censored"))
	}

	checkOpt("return_to_post", c.Txt("return_to_post"))
	checkOpt("no_new_reply_warning", c.Txt("no_new_reply_warning"))
	checkOpt("view_newest_first", c.Txt("recent_posts_at_top"))
	checkOpt("view_newest_pm_first", c.Txt("recent_pms_at_top"))

	dayOpt := opts["calendar_start_day"]
	daySel := func(v string) string {
		if (empty(dayOpt) && v == "0") || (!empty(dayOpt) && dayOpt == v) {
			return ` selected="selected"`
		}
		return ""
	}
	qrOpt := opts["display_quick_reply"]
	qrSel := func(v string) string {
		if (empty(qrOpt) && v == "0") || (!empty(qrOpt) && qrOpt == v) {
			return ` selected="selected"`
		}
		return ""
	}
	qmOpt := opts["display_quick_mod"]
	qmSel0 := ""
	qmSel1 := ""
	qmSel2 := ""
	if empty(qmOpt) {
		qmSel0 = ` selected="selected"`
	} else if qmOpt == "1" {
		qmSel1 = ` selected="selected"`
	} else {
		qmSel2 = ` selected="selected"`
	}
	c.O(`
										<tr>
											<td colspan="2"><label for="calendar_start_day">`, c.Txt("calendar_start_day"), `:</label>
												<select name="default_options[calendar_start_day]" id="calendar_start_day">
													<option value="0"`, daySel("0"), `>`, c.TxtList("days").Items[0], `</option>
													<option value="1"`, daySel("1"), `>`, c.TxtList("days").Items[1], `</option>
													<option value="6"`, daySel("6"), `>`, c.TxtList("days").Items[6], `</option>
												</select>
											</td>
										</tr><tr>
											<td colspan="2"><label for="display_quick_reply">`, c.Txt("display_quick_reply"), `</label>
												<select name="default_options[display_quick_reply]" id="display_quick_reply">
													<option value="0"`, qrSel("0"), `>`, c.Txt("display_quick_reply1"), `</option>
													<option value="1"`, qrSel("1"), `>`, c.Txt("display_quick_reply2"), `</option>
													<option value="2"`, qrSel("2"), `>`, c.Txt("display_quick_reply3"), `</option>
												</select>
											</td>
										</tr><tr>
											<td colspan="2"><label for="display_quick_mod">`, c.Txt("display_quick_mod"), `</label>
												<select name="default_options[display_quick_mod]" id="display_quick_mod">
													<option value="0"`, qmSel0, `>`, c.Txt("display_quick_mod_none"), `</option>
													<option value="1"`, qmSel1, `>`, c.Txt("display_quick_mod_check"), `</option>
													<option value="2"`, qmSel2, `>`, c.Txt("display_quick_mod_image"), `</option>
												</select>
											</td>
										</tr>
									</table>
								</td>
							</tr>`)

	// Show the standard "Save Settings" profile button.
	templateProfileSave(c, page)

	c.O(`
						</table>
					</td>
				</tr>
			</table>
		</form>`)
}

// templateProfileNotification is template_notification().
func templateProfileNotification(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileNotifyCtx)
	a := c.App
	scripturl := a.ScriptURL
	opts := page.Member.Options

	// The main containing header.
	c.O(`
			<table border="0" width="85%" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
				<tr class="titlebg">
					<td height="26">
						&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;
						`, c.Txt("79"), `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" height="25" style="padding: 2ex;">
						`, c.Txt("notification_info"), `
					</td>
				</tr><tr>
					<td class="windowbg2" width="100%">
						<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">`)

	// Allow notification on announcements to be disabled?
	if !a.SettingEmpty("allow_disableAnnounce") {
		checked := ""
		if page.Member.NotifyAnnouncement != 0 {
			checked = ` checked="checked"`
		}
		c.O(`
							<input type="hidden" name="notifyAnnouncements" value="0" />
							<label for="notifyAnnouncements"><input type="checkbox" id="notifyAnnouncements" name="notifyAnnouncements"`, checked, ` class="check" /> `, c.Txt("notifyXAnn4"), `</label><br />`)
	}

	// More notification options.
	onceChecked := ""
	if page.Member.NotifyOnce != 0 {
		onceChecked = ` checked="checked"`
	}
	autoChecked := ""
	if !empty(opts["auto_notify"]) {
		autoChecked = ` checked="checked"`
	}
	c.O(`
							<input type="hidden" name="notifyOnce" value="0" />
							<label for="notifyOnce"><input type="checkbox" id="notifyOnce" name="notifyOnce"`, onceChecked, ` class="check" /> `, c.Txt("notifyXOnce1"), `</label><br />

							<input type="hidden" name="default_options[auto_notify]" value="0" />
							<label for="auto_notify"><input type="checkbox" id="auto_notify" name="default_options[auto_notify]" value="1"`, autoChecked, ` class="check" /> `, c.Txt("auto_notify"), `</label><br />`)

	if a.SettingEmpty("disallow_sendBody") {
		bodyChecked := ""
		if page.Member.NotifySendBody != 0 {
			bodyChecked = ` checked="checked"`
		}
		c.O(`
							<input type="hidden" name="notifySendBody" value="0" />
							<label for="notifySendBody"><input type="checkbox" id="notifySendBody" name="notifySendBody"`, bodyChecked, ` class="check" /> `, c.Txt("notify_send_body"), `</label><br />`)
	}

	typeSel := func(v int) string {
		if page.Member.NotifyTypes == v {
			return ` selected="selected"`
		}
		return ""
	}
	align := "right"
	if c.RightToLeft {
		align = "left"
	}
	c.O(`
							<br />
							<label for="notifyTypes">`, c.Txt("notify_send_types"), `:</label>
							<select name="notifyTypes" id="notifyTypes">
								<option value="1"`, typeSel(1), `>`, c.Txt("notify_send_type_everything"), `</option>
								<option value="2"`, typeSel(2), `>`, c.Txt("notify_send_type_everything_own"), `</option>
								<option value="3"`, typeSel(3), `>`, c.Txt("notify_send_type_only_replies"), `</option>
								<option value="4"`, typeSel(4), `>`, c.Txt("notify_send_type_nothing"), `</option>
							</select><br />

							<div align="`, align, `">
								<input type="submit" style="margin: 0 1ex 1ex 1ex;" value="`, c.Txt("notifyX1"), `" />
								<input type="hidden" name="sc" value="`, c.Sc, `" />
								<input type="hidden" name="userID" value="`, page.MemID, `" />
								<input type="hidden" name="sa" value="`, page.MenuSelected, `" />
							</div>
						</form>
					</td>
				</tr>
			</table>
			<br />
			<table border="0" width="85%" cellspacing="0" cellpadding="0" align="center" class="bordercolor"><tr><td>
				<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
					<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
						<tr><td class="catbg" width="100%">`, c.Txt("notifications_topics"), `</td></tr>
					</table>
					<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
						<tr>
							<td class="windowbg" width="20" valign="middle" align="center" rowspan="`, len(sub.TopicNotifications)+3, `">
								<img src="`, c.Theme.ImagesURL(), `/icons/notify_sm.gif" width="20" height="20" alt="" />
							</td>`)
	if len(sub.TopicNotifications) > 0 {
		c.O(`
							<td class="titlebg" width="71%">` + c.Txt("70") + `</td>
							<td class="titlebg" width="24%">` + c.Txt("109") + `</td>
							<td class="titlebg" width="5%"><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></td>
						</tr>`)
		for _, topic := range sub.TopicNotifications {
			c.O(`
						<tr>
							<td class="windowbg" valign="middle" width="48%">
								`, topic.Link)

			if topic.New {
				c.O(` <a href="`, topic.NewHref, `"><img src="`+c.Theme.ImagesURL()+`/`+c.User.Language+`/new.gif" alt="`, c.Txt("302"), `" /></a>`)
			}

			c.O(`<br />
								<span class="smalltext"><i>`+c.Txt("smf88")+` `+topic.BoardLink+`</i></span>
							</td>
							<td class="windowbg2" valign="middle" width="14%">`+topic.PosterLink+`</td>
							<td class="windowbg2" valign="middle" width="5%">
								<input type="checkbox" name="notify_topics[]" value="`, topic.ID, `" class="check" />
							</td>
						</tr>`)
		}

		c.O(`
						<tr class="catbg">
							<td colspan="3">
								<b>`, c.Txt("139"), `:</b> `, sub.PageIndex, `
							</td>
						</tr>
						<tr>
							<td colspan="3" class="windowbg2" align="right">
								<input type="submit" name="edit_notify_topics" value="`, c.Txt("notifications_update"), `" />
							</td>
						</tr>`)
	} else {
		c.O(`
							<td width="100%" colspan="3" class="windowbg2">
								`, c.Txt("notifications_topics_none"), `<br />
								<br />`, c.Txt("notifications_topics_howto"), `<br />
								<br />
							</td>
						</tr>`)
	}
	c.O(`
					</table>
					<input type="hidden" name="sc" value="`, c.Sc, `" />
					<input type="hidden" name="userID" value="`, page.MemID, `" />
					<input type="hidden" name="sa" value="`, page.MenuSelected, `" />
				</form>
			</td></tr></table><br />
			<table border="0" width="85%" cellspacing="0" cellpadding="0" align="center" class="bordercolor"><tr><td>
				<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
					<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
						<tr><td class="catbg" width="100%">`, c.Txt("notifications_boards"), `</td></tr>
					</table>
					<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
						<tr>
							<td class="windowbg" width="20" valign="middle" align="center" rowspan="`, len(sub.BoardNotifications)+2, `">
								<img src="`, c.Theme.ImagesURL(), `/icons/notify_sm.gif" width="20" height="20" alt="" />
							</td>`)

	if len(sub.BoardNotifications) > 0 {
		c.O(`
							<td class="titlebg" width="95%">` + c.Txt("smf82") + `</td>
							<td class="titlebg" width="5%"><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></td>
						</tr>`)
		for _, board := range sub.BoardNotifications {
			c.O(`
						<tr>
							<td class="windowbg" valign="middle" width="48%">`, board.Link)

			if board.New {
				c.O(` <a href="`, board.Href, `"><img src="`+c.Theme.ImagesURL()+`/`+c.User.Language+`/new.gif" alt="`, c.Txt("302"), `" /></a>`)
			}

			c.O(`</td>
							<td class="windowbg2" valign="middle" width="5%">
								<input type="checkbox" name="notify_boards[]" value="`, board.ID, `" />
							</td>
						</tr>`)
		}

		c.O(`
						<tr>
							<td colspan="2" class="windowbg2" align="right">
								<input type="submit" name="edit_notify_boards" value="`, c.Txt("notifications_update"), `" />
							</td>
						</tr>`)
	} else {
		c.O(`
							<td width="100%" colspan="2" class="windowbg2">
								`, c.Txt("notifications_boards_none"), `<br />
								<br />`, c.Txt("notifications_boards_howto"), `<br />
								<br />
							</td>
						</tr>`)
	}
	c.O(`
					</table>
					<input type="hidden" name="sc" value="`, c.Sc, `" />
					<input type="hidden" name="userID" value="`, page.MemID, `" />
					<input type="hidden" name="sa" value="`, page.MenuSelected, `" />
				</form>
			</td></tr></table><br />`)
}

// templateProfilePmprefs is template_pmprefs().
func templateProfilePmprefs(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfilePMPrefsCtx)
	a := c.App
	scripturl := a.ScriptURL
	opts := page.Member.Options

	// The main containing header.
	c.O(`
		<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" name="creator" id="creator">
			<table border="0" width="85%" cellspacing="0" cellpadding="4" align="center" class="tborder">
				<tr class="titlebg">
					<td height="26">
						&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;
						`, c.Txt("79"), `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" style="padding: 2ex;">
						`, c.Txt("pmprefs_info"), `
					</td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 2ex;">
						<table border="0" width="100%" cellpadding="3">`)

	// A text box for the user to input usernames of everyone they want to
	// ignore personal messages from.
	c.O(`
							<tr>
								<td valign="top">
									<b>`, c.Txt("325"), `:</b>
									<div class="smalltext">
										`, c.Txt("326"), `<br />
										<br />
										<a href="`, scripturl, `?action=findmember;input=pm_ignore_list;delim=\\n;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, c.Theme.ImagesURL(), `/icons/assist.gif" alt="`, c.Txt("find_members"), `" align="middle" /> `, c.Txt("find_members"), `</a>
									</div>
								</td>
								<td>
									<textarea name="pm_ignore_list" id="pm_ignore_list" rows="10" cols="50">`, sub.IgnoreList, `</textarea>
								</td>
							</tr>`)

	// Extra options available to the user for personal messages.
	copyChecked := ""
	if !empty(opts["copy_to_outbox"]) {
		copyChecked = ` checked="checked"`
	}
	popupChecked := ""
	if !empty(opts["popup_messages"]) {
		popupChecked = ` checked="checked"`
	}
	neverSel := ""
	if sub.SendEmail == 0 {
		neverSel = ` selected="selected"`
	}
	alwaysSel := ""
	if sub.SendEmail != 0 && (sub.SendEmail == 1 || (a.SettingEmpty("enable_buddylist") && sub.SendEmail > 1)) {
		alwaysSel = ` selected="selected"`
	}
	c.O(`
							<tr>
								<td colspan="2">
									<input type="hidden" name="default_options[copy_to_outbox]" value="0" />
									<label for="copy_to_outbox"><input type="checkbox" name="default_options[copy_to_outbox]" id="copy_to_outbox" value="1"`, copyChecked, ` class="check" /> `, c.Txt("copy_to_outbox"), `</label><br />
									<input type="hidden" name="default_options[popup_messages]" value="0" />
									<label for="popup_messages"><input type="checkbox" name="default_options[popup_messages]" id="popup_messages" value="1"`, popupChecked, ` class="check" /> `, c.Txt("popup_messages"), `</label><br />
									<label for="pm_email_notify">`, c.Txt("327"), `</label>
									<select name="pm_email_notify" id="pm_email_notify">
										<option value="0"`, neverSel, `>`, c.Txt("email_notify_never"), `</option>
										<option value="1"`, alwaysSel, `>`, c.Txt("email_notify_always"), `</option>`)

	if !a.SettingEmpty("enable_buddylist") {
		buddySel := ""
		if sub.SendEmail > 1 {
			buddySel = ` selected="selected"`
		}
		c.O(`
										<option value="2"`, buddySel, `>`, c.Txt("email_notify_buddies"), `</option>`)
	}

	c.O(`
									</select><br />
								</td>
							</tr>`)

	// Show the standard "Save Settings" profile button.
	templateProfileSave(c, page)

	c.O(`
						</table>
					</td>
				</tr>
			</table>
		</form>`)
}

// templateProfileDeleteAccount is template_deleteAccount().
func templateProfileDeleteAccount(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileDeleteCtx)
	scripturl := c.App.ScriptURL

	// The main containing header.
	c.O(`
		<form action="`, scripturl, `?action=profile2" method="post" accept-charset="`, c.CharacterSet, `" name="creator" id="creator">
			<table border="0" width="85%" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
				<tr class="titlebg">
					<td height="26">
						&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;
						`, c.Txt("deleteAccount"), `
					</td>
				</tr>`)
	// If deleting another account give them a lovely info box.
	if !page.IsOwner {
		c.O(`
					<tr class="windowbg">
						<td class="smalltext" colspan="2" style="padding-top: 2ex; padding-bottom: 2ex;">
							`, c.Txt("deleteAccount_desc"), `
						</td>
					</tr>`)
	}
	c.O(`
				<tr>
					<td class="windowbg2">
						<table width="100%" cellspacing="0" cellpadding="3"><tr>
							<td align="center" colspan="2">`)

	// If they are deleting their account AND the admin needs to approve it -
	// give them another piece of info ;)
	if sub.NeedsApproval {
		c.O(`
								<div style="color: red; border: 2px dashed red; padding: 4px;">`, c.Txt("deleteAccount_approval"), `</div><br />
							</td>
						</tr><tr>
							<td align="center" colspan="2">`)
	}

	// If the user is deleting their own account warn them first - and
	// require a password!
	if page.IsOwner {
		alignA := "right"
		alignB := "left"
		if c.RightToLeft {
			alignA = "left"
			alignB = "right"
		}
		passColor := ""
		if page.ModifyError["bad_password"] || page.ModifyError["no_password"] {
			passColor = ` style="color: red;"`
		}
		c.O(`
								<span style="color: red;">`, c.Txt("own_profile_confirm"), `</span><br /><br />
							</td>
						</tr><tr>
							<td class="windowbg2" align="`, alignA, `">
								<b`, passColor, `>`, c.Txt("smf241"), `: </b>
							</td>
							<td class="windowbg2" align="`, alignB, `">
								<input type="password" name="oldpasswrd" size="20" />&nbsp;&nbsp;&nbsp;&nbsp;
								<input type="submit" value="`, c.Txt("163"), `" />
								<input type="hidden" name="sc" value="`, c.Sc, `" />
								<input type="hidden" name="userID" value="`, page.MemID, `" />
								<input type="hidden" name="sa" value="`, page.MenuSelected, `" />
							</td>`)
	} else {
		// Otherwise an admin doesn't need to enter a password - but they
		// still get a warning - plus the option to delete lovely posts!
		c.O(`
								<div style="color: red; margin-bottom: 2ex;">`, c.Txt("deleteAccount_warning"), `</div>
							</td>
						</tr>`)

		// Only actually give these options if they are kind of important.
		if sub.CanDeletePosts {
			c.O(`
						<tr>
							<td colspan="2" align="center">
								`, c.Txt("deleteAccount_posts"), `: <select name="remove_type">
									<option value="none">`, c.Txt("deleteAccount_none"), `</option>
									<option value="posts">`, c.Txt("deleteAccount_all_posts"), `</option>
									<option value="topics">`, c.Txt("deleteAccount_topics"), `</option>
								</select>
							</td>
						</tr>`)
		}

		c.O(`
						<tr>
							<td colspan="2" align="center">
								<label for="deleteAccount"><input type="checkbox" name="deleteAccount" id="deleteAccount" value="1" class="check" onclick="if (this.checked) return confirm('`, c.Txt("deleteAccount_confirm"), `');" /> `, c.Txt("deleteAccount_member"), `.</label>
							</td>
						</tr>
						<tr>
							<td colspan="2" class="windowbg2" align="center" style="padding-top: 2ex;">
								<input type="submit" value="`, c.Txt("smf138"), `" />
								<input type="hidden" name="sc" value="`, c.Sc, `" />
								<input type="hidden" name="userID" value="`, page.MemID, `" />
								<input type="hidden" name="sa" value="`, page.MenuSelected, `" />
							</td>`)
	}
	c.O(`
						</tr></table>
					</td>
				</tr>
			</table>
		</form>`)
}
