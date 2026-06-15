package app

// Ports of Themes/default/ManageAttachments.template.php: the manage_files
// layer (above/below) plus template_attachments / avatars / browse /
// maintenance / attachment_repair.

// templateManageFilesAbove is template_manage_files_above().
func templateManageFilesAbove(c *Ctx) {
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	sel := c.attachSelected
	description := c.Txt("smf202")

	c.O(`
	<table border="0" cellspacing="0" cellpadding="4" align="center" width="100%" class="tborder">
		<tr class="titlebg">`)

	if !c.Theme.Empty("use_tabs") {
		c.O(`
			<td><a href="`, scripturl, `?action=helpadmin;help=manage_files" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("smf201"), `</td>
		</tr>
		<tr class="windowbg">
			<td class="smalltext" style="padding: 2ex;">
				`, description, `
			</td>
		</tr>
	</table>
	<table cellpadding="0" cellspacing="0" border="0" style="margin-left: 10px;">
		<tr>
			<td class="maintab_first">&nbsp;</td>`)
		tab := func(key, label, href string) {
			active := sel == key
			if active {
				c.O(`
			<td class="maintab_active_first">&nbsp;</td>`)
			}
			cls := "maintab_back"
			if active {
				cls = "maintab_active_back"
			}
			c.O(`
			<td class="`, cls, `"><a href="`, scripturl, href, `">`, label, `</a></td>`)
			if active {
				c.O(`
			<td class="maintab_active_last">&nbsp;</td>`)
			}
		}
		tab("attachment_settings", c.Txt("attachment_manager_settings"), "?action=manageattachments")
		tab("avatar_settings", c.Txt("attachment_manager_avatar_settings"), "?action=manageattachments;sa=avatars")
		tab("browse", c.Txt("attachment_manager_browse"), "?action=manageattachments;sa=browse")
		tab("maintenance", c.Txt("attachment_manager_maintenance"), "?action=manageattachments;sa=maintenance")
		c.O(`
			<td class="maintab_last">&nbsp;</td>
		</tr>
	</table><br />`)
	} else {
		c.O(`
			<td><a href="`, scripturl, `?action=helpadmin;help=manage_files" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" border="0" align="top" /></a> `, c.Txt("smf201"), `</td>
		</tr>
		<tr class="catbg">
			<td align="left">`)
		oldTab := func(key, label, href string, last bool) {
			marker := ""
			if sel == key {
				marker = `<a href="` + scripturl + href + `"><img src="` + imagesURL + `/selected.gif" alt="&gt;" border="0" /></a> `
			}
			sep := " | "
			if last {
				sep = ""
			}
			c.O(`
				`, marker, `<a href="`, scripturl, href, `">`, label, `</a>`, sep)
		}
		oldTab("attachment_settings", c.Txt("attachment_manager_settings"), "?action=manageattachments", false)
		oldTab("avatar_settings", c.Txt("attachment_manager_avatar_settings"), "?action=manageattachments;sa=avatars", false)
		oldTab("browse", c.Txt("attachment_manager_browse"), "?action=manageattachments;sa=browse", false)
		oldTab("maintenance", c.Txt("attachment_manager_maintenance"), "?action=manageattachments;sa=maintenance", true)
		c.O(`
			</td>
		</tr>
		<tr class="windowbg">
			<td class="smalltext" style="padding: 2ex;">
				`, description, `
			</td>
		</tr>
	</table>
	<br />`)
	}
}

// templateManageFilesBelow is template_manage_files_below() (empty).
func templateManageFilesBelow(c *Ctx) {}

// templateAttachments is template_attachments().
func templateAttachments(c *Ctx) {
	page := c.Page.(*attachSettingsPage)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	a := c.App

	enable := a.SettingInt("attachmentEnable")
	c.O(`
<form action="`, scripturl, `?action=manageattachments" method="post" accept-charset="`, c.CharacterSet, `">
	<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
		<tr class="titlebg">
			<td colspan="2"><a href="`, scripturl, `?action=helpadmin;help=attachmentEnable" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="Help" /></a> `, c.Txt("attachment_manager_settings"), `</td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentEnable">`, c.Txt("attachment_mode"), `:</label></td>
			<td>
				<select name="attachmentEnable" id="attachmentEnable">
					<option value="0"`, selAttr(enable == 0), `>`, c.Txt("attachment_mode_deactivate"), `</option>
					<option value="1"`, selAttr(enable == 1), `>`, c.Txt("attachment_mode_enable_all"), `</option>
					<option value="2"`, selAttr(enable == 2), `>`, c.Txt("attachment_mode_disable_new"), `</option>
				</select>
			</td>
		</tr><tr class="windowbg2">
			<td colspan="2"><hr /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentCheckExtensions">`, c.Txt("attachmentCheckExtensions"), `:</label></td>
			<td><input type="checkbox" name="attachmentCheckExtensions" id="attachmentCheckExtensions" value="1" class="check"`, checkedIf(!a.SettingEmpty("attachmentCheckExtensions")), ` /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentExtensions">`, c.Txt("attachmentExtensions"), `</label>:</td>
			<td><input type="text" name="attachmentExtensions" id="attachmentExtensions" value="`, a.Setting("attachmentExtensions"), `" size="40" /></td>
		</tr><tr class="windowbg2">
			<td colspan="2"><hr /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentUploadDir"`, redBoldIf(!page.ValidUploadDir), `>`, c.Txt("attachmentUploadDir"), `</label>:</td>
			<td><input type="text" name="attachmentUploadDir" id="attachmentUploadDir" value="`, a.Setting("attachmentUploadDir"), `" size="40" /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentDirSizeLimit">`, c.Txt("attachmentDirSizeLimit"), `</label>:</td>
			<td><input type="text" name="attachmentDirSizeLimit" id="attachmentDirSizeLimit" value="`, a.Setting("attachmentDirSizeLimit"), `" size="6" /> `, c.Txt("smf211"), `</td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentPostLimit">`, c.Txt("attachmentPostLimit"), `</label>:</td>
			<td><input type="text" name="attachmentPostLimit" id="attachmentPostLimit" value="`, a.Setting("attachmentPostLimit"), `" size="6" /> `, c.Txt("smf211"), `</td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentSizeLimit">`, c.Txt("attachmentSizeLimit"), `</label>:</td>
			<td><input type="text" name="attachmentSizeLimit" id="attachmentSizeLimit" value="`, a.Setting("attachmentSizeLimit"), `" size="6" /> `, c.Txt("smf211"), `</td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentNumPerPostLimit">`, c.Txt("attachmentNumPerPostLimit"), `</label>:</td>
			<td><input type="text" name="attachmentNumPerPostLimit" id="attachmentNumPerPostLimit" value="`, a.Setting("attachmentNumPerPostLimit"), `" size="6" /></td>
		</tr><tr class="windowbg2">
			<td colspan="2"><hr /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentShowImages">`, c.Txt("attachmentShowImages"), `:</label></td>
			<td><input type="checkbox" name="attachmentShowImages" id="attachmentShowImages" value="1" class="check"`, checkedIf(!a.SettingEmpty("attachmentShowImages")), ` /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentThumbnails">`, c.Txt("attachmentThumbnails"), `</label>:</td>
			<td><input type="checkbox" name="attachmentThumbnails" id="attachmentThumbnails" value="1" class="check"`, checkedIf(!a.SettingEmpty("attachmentThumbnails")), ` /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentThumbWidth">`, c.Txt("attachmentThumbWidth"), `</label>:</td>
			<td><input type="text" name="attachmentThumbWidth" id="attachmentThumbWidth" value="`, a.SettingInt("attachmentThumbWidth"), `" size="6" /></td>
		</tr><tr class="windowbg2">
			<td width="50%" align="right"><label for="attachmentThumbHeight">`, c.Txt("attachmentThumbHeight"), `</label>:</td>
			<td><input type="text" name="attachmentThumbHeight" id="attachmentThumbHeight" value="`, a.SettingInt("attachmentThumbHeight"), `" size="6" /></td>
		</tr><tr class="windowbg2">
			<td colspan="2" align="center">
				<input type="submit" name="attachmentSettings" value="`, c.Txt("attachment_manager_save"), `" />
				<input type="hidden" name="sa" value="attachments" />
				<input type="hidden" name="sc" value="`, c.Sc, `" />
			</td>
		</tr>
	</table>
</form>`)
}

func redBoldIf(b bool) string {
	if b {
		return ` style="color: red; font-weight: bold;"`
	}
	return ""
}

// templateAvatars is template_avatars().
func templateAvatars(c *Ctx) {
	page := c.Page.(*avatarSettingsPage)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=manageattachments" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2"><a href="`, scripturl, `?action=helpadmin;help=avatar_allow_server_stored" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="Help" /></a> `, c.Txt("avatar_server_stored"), `</td>`)
	if !page.GDInstalled {
		c.O(`
			</tr><tr class="windowbg2">
				<td colspan="2" align="center" style="color: red; padding: 2em;">`, c.Txt("avatar_gd_warning"), `</td>`)
	}
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" valign="top" align="right"><label for="profile_server_avatar">`, c.Txt("avatar_server_stored_groups"), `</label>:</td>
				<td>`)
		c.themeInlinePermissions("profile_server_avatar")
		c.O(`
				</td>
			</tr>`)
	}
	c.O(`
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_directory"`, redBoldIf(!page.ValidAvatarDir), `>`, c.Txt("avatar_directory"), `</label>:</td>
				<td><input type="text" name="avatar_directory" id="avatar_directory" value="`, a.Setting("avatar_directory"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_url">`, c.Txt("avatar_url"), `</label>:</td>
				<td><input type="text" name="avatar_url" id="avatar_url" value="`, a.Setting("avatar_url"), `" size="40" /></td>
			</tr>
			<tr>
				<td colspan="2" class="titlebg"><a href="`, scripturl, `?action=helpadmin;help=avatar_allow_external_url" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="Help" /></a> `, c.Txt("avatar_external"), `</td>
			</tr>`)
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" valign="top" align="right"><label for="external_url_groups">`, c.Txt("avatar_external_url_groups"), `</label>:</td>
				<td>`)
		c.themeInlinePermissions("profile_remote_avatar")
		c.O(`
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_download_external">`, c.Txt("avatar_download_external"), ` <a href="`, scripturl, `?action=helpadmin;help=avatar_download_external" onclick="return reqWin(this.href);" class="help">(?)</a>:</label></td>
				<td><input type="checkbox" name="avatar_download_external" id="avatar_download_external" value="1" class="check"`, checkedIf(!a.SettingEmpty("avatar_download_external")), ` onchange="updateStatus()" /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_max_width_external">`, c.Txt("avatar_max_width_external"), `</label>:<div class="smalltext" style="font-weight: bold;">`, c.Txt("avatar_dimension_note"), `</div></td>
				<td>
					<input type="text" name="avatar_max_width_external" id="avatar_max_width_external" value="`, a.Setting("avatar_max_width_external"), `" size="6" />
				</td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_max_height_external">`, c.Txt("avatar_max_height_external"), `</label>:<div class="smalltext" style="font-weight: bold;">`, c.Txt("avatar_dimension_note"), `</div></td>
				<td>
					<input type="text" name="avatar_max_height_external" id="avatar_max_height_external" value="`, a.Setting("avatar_max_height_external"), `" size="6" />
				</td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_action_too_large">`, c.Txt("avatar_action_too_large"), `</label></td>
				<td>
					<select name="avatar_action_too_large" id="avatar_action_too_large">
						<option value="option_refuse"`, selAttr(a.Setting("avatar_action_too_large") == "option_refuse"), `>`, c.Txt("option_refuse"), `</option>
						<option value="option_html_resize"`, selAttr(a.Setting("avatar_action_too_large") == "option_html_resize"), `>`, c.Txt("option_html_resize"), `</option>
						<option value="option_js_resize"`, selAttr(a.Setting("avatar_action_too_large") == "option_js_resize"), `>`, c.Txt("option_js_resize"), `</option>
						<option value="option_download_and_resize"`, selAttr(a.Setting("avatar_action_too_large") == "option_download_and_resize"), `>`, c.Txt("option_download_and_resize"), `</option>
					</select>
				</td>
			</tr><tr>
				<td colspan="2" class="titlebg"><a href="`, scripturl, `?action=helpadmin;help=avatar_allow_upload" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="Help" /></a> `, c.Txt("avatar_upload"), `</td>`)
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" valign="top" align="right"><label for="profile_upload_avatar">`, c.Txt("avatar_upload_groups"), `</label>:</td>
				<td>`)
		c.themeInlinePermissions("profile_upload_avatar")
		c.O(`
				</td>
			</tr>`)
	}
	gdRed := ""
	if !page.GDInstalled {
		gdRed = "color: red;"
	}
	c.O(`
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_max_width_upload">`, c.Txt("avatar_max_width_upload"), `</label>:<div class="smalltext" style="font-weight: bold;">`, c.Txt("avatar_dimension_note"), `</div></td>
				<td><input type="text" name="avatar_max_width_upload" id="avatar_max_width_upload" value="`, a.Setting("avatar_max_width_upload"), `" size="6" /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_max_height_upload">`, c.Txt("avatar_max_height_upload"), `</label>:<div class="smalltext" style="font-weight: bold;">`, c.Txt("avatar_dimension_note"), `</div></td>
				<td><input type="text" name="avatar_max_height_upload" id="avatar_max_height_upload" value="`, a.Setting("avatar_max_height_upload"), `" size="6" /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_resize_upload">`, c.Txt("avatar_resize_upload"), `:</label><div class="smalltext" style="font-weight: bold;`, gdRed, `">`, c.Txt("avatar_resize_upload_note"), `</div></td>
				<td><input type="checkbox" name="avatar_resize_upload" id="avatar_resize_upload" value="1" class="check"`, checkedIf(!a.SettingEmpty("avatar_resize_upload")), ` /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="avatar_download_png">`, c.Txt("avatar_download_png"), ` <a href="`, scripturl, `?action=helpadmin;help=avatar_download_png" onclick="return reqWin(this.href);" class="help">(?)</a>:</label></td>
				<td><input type="checkbox" name="avatar_download_png" id="avatar_download_png" value="1" class="check"`, checkedIf(!a.SettingEmpty("avatar_download_png")), ` /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="custom_avatar_enabled">`, c.Txt("custom_avatar_enabled"), `</label></td>
				<td>
					<select name="custom_avatar_enabled" id="custom_avatar_enabled" onchange="updateStatus()">
						<option value="0"`, selAttr(a.SettingEmpty("custom_avatar_enabled")), `>`, c.Txt("option_attachment_dir"), `</option>
						<option value="1"`, selAttr(!a.SettingEmpty("custom_avatar_enabled")), `>`, c.Txt("option_specified_dir"), `</option>
					</select>
				</td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right">
					<label for="custom_avatar_dir"`, redBoldIf(!page.ValidCustomAvatarDir), `>`, c.Txt("custom_avatar_dir"), `</label>:<br />
					<span class="smalltext">`, c.Txt("custom_avatar_dir_desc"), `</span>
				</td>
				<td><input type="text" name="custom_avatar_dir" id="custom_avatar_dir" value="`, a.Setting("custom_avatar_dir"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="custom_avatar_url">`, c.Txt("custom_avatar_url"), `</label>:</td>
				<td><input type="text" name="custom_avatar_url" id="custom_avatar_url" value="`, a.Setting("custom_avatar_url"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2" align="center">
					<input type="submit" name="avatarSettings" value="`, c.Txt("attachment_manager_save"), `" />
					<input type="hidden" name="sa" value="avatars" />
					<input type="hidden" name="sc" value="`, c.Sc, `" />
				</td>
			</tr>
		</table>
	</form>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function updateStatus()
		{
			document.getElementById("avatar_max_width_external").disabled = document.getElementById("avatar_download_external").checked;
			document.getElementById("avatar_max_height_external").disabled = document.getElementById("avatar_download_external").checked;
			document.getElementById("avatar_action_too_large").disabled = document.getElementById("avatar_download_external").checked;
			document.getElementById("custom_avatar_dir").disabled = document.getElementById("custom_avatar_enabled").value == 0;
			document.getElementById("custom_avatar_url").disabled = document.getElementById("custom_avatar_enabled").value == 0;

		}
		window.onload = updateStatus;
	// ]]></script>
	`)
}

// templateBrowse is template_browse().
func templateBrowse(c *Ctx) {
	page := c.Page.(*browsePage)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	descPart := ""
	if page.SortDirection == "down" {
		descPart = ";desc"
	}

	c.O(`
	<form action="`, scripturl, `?action=manageattachments;sort=`, page.SortBy, descPart, `;sa=remove" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("confirm_delete_attachments"), `');">
	<table border="0" align="center" cellspacing="1" cellpadding="4" class="bordercolor" width="100%">
		<tr class="titlebg">
			<td colspan="5">`, c.Txt("attachment_manager_browse_files"), `</td>
		</tr>`)

	subtab := func(bt, label string) {
		active := page.BrowseType == bt
		btPart := ""
		if bt != "attachments" {
			btPart = ";" + bt
		}
		if active {
			c.O(`
			<td class="maintab_active_first">&nbsp;</td>`)
		}
		cls := "maintab_back"
		if active {
			cls = "maintab_active_back"
		}
		c.O(`
			<td class="`, cls, `"><a href="`, scripturl, `?action=manageattachments;sa=browse`, btPart, `;sort=`, page.SortBy, descPart, `">`, label, `</a></td>`)
		if active {
			c.O(`
			<td class="maintab_active_last">&nbsp;</td>`)
		}
	}

	if !c.Theme.Empty("use_tabs") {
		c.O(`
	</table>
	<table cellpadding="0" cellspacing="0" border="0" style="margin-bottom: 1ex; margin-left: 10px;">
		<tr>
			<td class="maintab_first">&nbsp;</td>`)
		subtab("attachments", c.Txt("attachment_manager_attachments"))
		subtab("avatars", c.Txt("attachment_manager_avatars"))
		subtab("thumbs", c.Txt("attachment_manager_thumbs"))
		c.O(`
			<td class="maintab_last">&nbsp;</td>
		</tr>
	</table>
	<table border="0" align="center" cellspacing="1" cellpadding="4" class="bordercolor" width="100%">
		<tr class="titlebg">`)
	} else {
		old := func(bt, label string) {
			btPart := ""
			if bt != "attachments" {
				btPart = ";" + bt
			}
			marker := ""
			if page.BrowseType == bt {
				marker = `<img src="` + imagesURL + `/selected.gif" alt="&gt;" border="0" /> `
			}
			c.O(`
				<a href="`, scripturl, `?action=manageattachments;sa=browse`, btPart, `;sort=`, page.SortBy, descPart, `">`, marker, label, `</a>`)
		}
		c.O(`
		<tr class="catbg">
			<td colspan="5">`)
		old("attachments", c.Txt("attachment_manager_attachments"))
		c.O(`&nbsp;|&nbsp;`)
		old("avatars", c.Txt("attachment_manager_avatars"))
		c.O(`&nbsp;|&nbsp;`)
		old("thumbs", c.Txt("attachment_manager_thumbs"))
		c.O(`
			</td>
		</tr><tr class="titlebg">`)
	}

	memberLabel := c.Txt("279")
	dateLabel := c.Txt("317")
	if page.BrowseType == "avatars" {
		memberLabel = c.Txt("attachment_manager_member")
		dateLabel = c.Txt("attachment_manager_last_active")
	}
	sortImg := func(col string) string {
		if page.SortBy == col {
			return ` <img src="` + imagesURL + `/sort_` + page.SortDirection + `.gif" alt="" />`
		}
		return ""
	}
	flip := func(col string) string {
		if page.SortBy == col && page.SortDirection == "up" {
			return ";desc"
		}
		return ""
	}
	c.O(`
			<td nowrap="nowrap"><a href="`, scripturl, `?action=manageattachments;sa=browse;`, page.BrowseType, `;sort=name`, flip("name"), `">`, c.Txt("smf213"), sortImg("name"), `</a></td>
			<td nowrap="nowrap"><a href="`, scripturl, `?action=manageattachments;sa=browse;`, page.BrowseType, `;sort=size`, flip("size"), `">`, c.Txt("smf214"), sortImg("size"), `</a></td>
			<td nowrap="nowrap"><a href="`, scripturl, `?action=manageattachments;sa=browse;`, page.BrowseType, `;sort=member`, flip("member"), `">`, memberLabel, sortImg("member"), `</a></td>
			<td nowrap="nowrap"><a href="`, scripturl, `?action=manageattachments;sa=browse;`, page.BrowseType, `;sort=date`, flip("date"), `">`, dateLabel, sortImg("date"), `</a></td>
			<td nowrap="nowrap" align="center"><input type="checkbox" onclick="invertAll(this, this.form);" class="check" /></td>
		</tr>`)

	alternate := false
	for _, post := range page.Posts {
		rowClass := "windowbg2"
		if alternate {
			rowClass = "windowbg"
		}
		dims := ""
		if post.Width != 0 && post.Height != 0 {
			dims = ` <span class="smalltext">` + itoa(post.Width) + "x" + itoa(post.Height) + `</span>`
		}
		timeCol := post.Time
		if page.BrowseType != "avatars" {
			timeCol += "<br />" + c.Txt("smf88") + " " + post.TopicLink
		}
		c.O(`
		<tr class="`, rowClass, `">
			<td>`, post.AttachmentLink, dims, `</td>
			<td align="right">`, post.Size, c.Txt("smf211"), `</td>
			<td>`, post.PosterLink, `</td>
			<td class="smalltext">`, timeCol, `</td>
			<td align="center"><input type="checkbox" name="remove[`, post.AttachmentID, `]" class="check" /></td>
		</tr>`)
		alternate = !alternate
	}
	rowClass := "windowbg2"
	if alternate {
		rowClass = "windowbg"
	}
	c.O(`
		<tr class="`, rowClass, `">
			<td align="right" colspan="5">
				<input type="submit" name="remove_submit" value="`, c.Txt("smf138"), `" />
				<input type="hidden" name="sc" value="`, c.Sc, `" />
				<input type="hidden" name="type" value="`, page.BrowseType, `" />
				<input type="hidden" name="start" value="`, page.Start, `" />
			</td>
		</tr>
		<tr class="catbg">
			<td align="left" colspan="5" style="padding: 5px;"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
		</tr>
	</table>
	</form>`)
}

// templateMaintenance is template_maintenance().
func templateMaintenance(c *Ctx) {
	page := c.Page.(*maintenancePage)
	scripturl := c.App.ScriptURL

	space := c.Txt("smf215")
	if page.HasSpaceLimit {
		space = page.AttachmentSpace + " " + c.Txt("smf211")
	}
	c.O(`
	<table width="100%" cellpadding="4" cellspacing="0" align="center" border="0" class="tborder">
		<tr>
			<td class="titlebg">`, c.Txt("smf203"), `</td>
		</tr><tr>
			<td class="windowbg2" width="100%" valign="top" style="padding-bottom: 2ex;">
				<table border="0" cellspacing="0" cellpadding="3">
					<tr>
						<td>`, c.Txt("smf204"), `:</td><td>`, page.NumAttachments, `</td>
					</tr><tr>
						<td>`, c.Txt("attachment_manager_total_avatars"), `:</td><td>`, page.NumAvatars, `</td>
					</tr><tr>
						<td>`, c.Txt("smf205"), `:</td><td>`, page.AttachmentTotalSize, ` `, c.Txt("smf211"), ` <a href="`, scripturl, `?action=manageattachments;sa=repair;sesc=`, c.Sc, `">[`, c.Txt("attachment_manager_repair"), `]</a></td>
					</tr><tr>
						<td>`, c.Txt("smf206"), `:</td><td>`, space, `</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
	<br />
	<table width="100%" cellpadding="4" cellspacing="0" align="center" border="0" class="tborder">
		<tr>
			<td class="titlebg">`, c.Txt("smf207"), `</td>
		</tr><tr>
			<td class="windowbg2" width="100%" valign="top">
				<form action="`, scripturl, `?action=manageattachments" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("confirm_delete_attachments"), `');" style="margin: 0 0 2ex 0;">
					`, c.Txt("72"), `: <input type="text" name="notice" value="`, c.Txt("smf216"), `" size="40" /><br />
					`, c.Txt("smf209"), ` <input type="text" name="age" value="25" size="4" /> `, c.Txt("579"), ` <input type="submit" name="submit" value="`, c.Txt("31"), `" />
					<input type="hidden" name="type" value="attachments" />
					<input type="hidden" name="sc" value="`, c.Sc, `" />
					<input type="hidden" name="sa" value="byAge" />
				</form>
				<form action="`, scripturl, `?action=manageattachments" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("confirm_delete_attachments"), `');" style="margin: 0 0 2ex 0;">
					`, c.Txt("72"), `: <input type="text" name="notice" value="`, c.Txt("smf216"), `" size="40" /><br />
					`, c.Txt("smf210"), ` <input type="text" name="size" id="size" value="100" size="4" /> `, c.Txt("smf211"), ` <input type="submit" name="submit" value="`, c.Txt("31"), `" />
					<input type="hidden" name="type" value="attachments" />
					<input type="hidden" name="sc" value="`, c.Sc, `" />
					<input type="hidden" name="sa" value="bySize" />
				</form>
				<form action="`, scripturl, `?action=manageattachments" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("confirm_delete_attachments"), `');" style="margin: 0 0 2ex 0;">
					`, c.Txt("attachment_manager_avatars_older"), ` <input type="text" name="age" value="45" size="4" /> `, c.Txt("579"), ` <input type="submit" name="submit" value="`, c.Txt("31"), `" />
					<input type="hidden" name="type" value="avatars" />
					<input type="hidden" name="sc" value="`, c.Sc, `" />
					<input type="hidden" name="sa" value="byAge" />
				</form>
			</td>
		</tr>
	</table>`)
}

// templateAttachmentRepair is template_attachment_repair().
func templateAttachmentRepair(c *Ctx) {
	page := c.Page.(*repairPage)
	scripturl := c.App.ScriptURL

	if page.Completed {
		c.O(`
	<table width="100%" cellpadding="4" cellspacing="0" align="center" border="0" class="tborder">
		<tr>
			<td class="titlebg">`, c.Txt("repair_attachments_complete"), `</td>
		</tr><tr>
			<td class="windowbg2" width="100%">
				`, c.Txt("repair_attachments_complete_desc"), `
			</td>
		</tr>
	</table>`)
		return
	}
	if !page.ErrorsFound {
		c.O(`
	<table width="100%" cellpadding="4" cellspacing="0" align="center" border="0" class="tborder">
		<tr>
			<td class="titlebg">`, c.Txt("repair_attachments_complete"), `</td>
		</tr><tr>
			<td class="windowbg2" width="100%">
				`, c.Txt("repair_attachments_no_errors"), `
			</td>
		</tr>
	</table>`)
		return
	}
	c.O(`
	<form action="`, scripturl, `?action=manageattachments;sa=repair;fixErrors=1;step=0;substep=0;sesc=`, c.Sc, `" method="post" accept-charset="`, c.CharacterSet, `">
	<table width="100%" cellpadding="4" cellspacing="0" align="center" border="0" class="tborder">
		<tr>
			<td class="titlebg">`, c.Txt("repair_attachments"), `</td>
		</tr><tr>
			<td class="windowbg2">
				`, c.Txt("repair_attachments_error_desc"), `
			</td>
		</tr>`)
	for _, e := range page.RepairErrors {
		if e.Count == 0 {
			continue
		}
		c.O(`
		<tr class="windowbg2">
			<td>
				<input type="checkbox" name="to_fix[]" id="`, e.Key, `" value="`, e.Key, `" />
				<label for="`, e.Key, `">`, phpSprintf(c.Txt("attach_repair_"+e.Key), e.Count), `</label>
			</td>
		</tr>`)
	}
	c.O(`
		<tr>
			<td align="center" class="windowbg2">
				<input type="submit" value="`, c.Txt("repair_attachments_continue"), `" />
				<input type="submit" name="cancel" value="`, c.Txt("repair_attachments_cancel"), `" />
			</td>
		</tr>
	</table>
	</form>`)
}
