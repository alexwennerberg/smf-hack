package app

// Ports of Themes/default/ManageMembergroups.template.php.

// templateMembergroupsMain is template_main().
func templateMembergroupsMain(c *Ctx) {
	page := c.Page.(*MGIndexCtx)
	scripturl := c.App.ScriptURL

	nameCell := func(g MGIndexGroup, extraHelp string) string {
		inner := g.Name
		if g.CanSearch && g.Link != "" {
			inner = g.Link
		}
		if g.Color != "" {
			inner = `<span style="color: ` + g.Color + `">` + inner + `</span>`
		}
		return inner + extraHelp
	}

	c.O(`
		<div class="tborder">
			<form action="`, scripturl, `?action=membergroups;sa=add;generalgroup" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
				<table width="100%" cellpadding="2" cellspacing="1" border="0">
					<tr class="titlebg"><td colspan="4" style="padding: 4px;">`, c.Txt("membergroups_regular"), `</td></tr>
					<tr class="catbg3">
						<td width="42%">`, c.Txt("membergroups_name"), `</td>
						<td width="12%" align="center">`, c.Txt("membergroups_stars"), `</td>
						<td width="10%" align="center">`, c.Txt("membergroups_members_top"), `</td>
						<td width="10%" align="center">`, c.Txt("17"), `</td>
					</tr>`)
	for _, g := range page.Regular {
		help := ""
		if g.ID == 1 {
			help = ` (<a href="` + scripturl + `?action=helpadmin;help=membergroup_administrator" onclick="return reqWin(this.href);">?</a>)`
		} else if g.ID == 3 {
			help = ` (<a href="` + scripturl + `?action=helpadmin;help=membergroup_moderator" onclick="return reqWin(this.href);">?</a>)`
		}
		c.O(`
					<tr>
						<td class="windowbg2">`, nameCell(g, help), `</td>
						<td class="windowbg2" align="left">`, g.Stars, `</td>
						<td class="windowbg" align="center">`, g.NumMembers, `</td>
						<td class="windowbg2" align="center"><a href="`, scripturl, `?action=membergroups;sa=edit;group=`, g.ID, `">`, c.Txt("membergroups_modify"), `</a></td>
					</tr>`)
	}
	c.O(`
					<tr class="windowbg">
						<td colspan="4" align="right" style="padding-top: 1ex; padding-bottom: 2ex;">
							<input type="submit" value="`, c.Txt("membergroups_add_group"), `" style="margin: 4px;" />
						</td>
					</tr>
				</table>
				<input type="hidden" name="sc" value="`, c.Sc, `" />
				<input type="hidden" name="postgroup" value="0" />
				<input type="hidden" name="generalgroup" value="1" />
			</form>
		</div><br />
		<div class="tborder">
			<form action="`, scripturl, `?action=membergroups;sa=add" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;">
				<table width="100%" border="0" cellpadding="2" cellspacing="1">
					<tr class="titlebg"><td colspan="5" style="padding: 4px;">`, c.Txt("membergroups_post"), `</td></tr>
					<tr class="catbg3">
						<td width="42%">`, c.Txt("membergroups_name"), `</td>
						<td width="12%" align="center">`, c.Txt("membergroups_stars"), `</td>
						<td width="10%" align="center">`, c.Txt("membergroups_members_top"), `</td>
						<td width="12%" align="center">`, c.Txt("membergroups_min_posts"), `</td>
						<td width="10%" align="center">`, c.Txt("17"), `</td>
					</tr>`)
	for _, g := range page.Post {
		c.O(`
					<tr>
						<td class="windowbg2">`, nameCell(g, ""), `</td>
						<td class="windowbg2" align="left">`, g.Stars, `</td>
						<td class="windowbg" align="center">`, g.NumMembers, `</td>
						<td class="windowbg" align="center">`, g.MinPosts, `</td>
						<td class="windowbg2" align="center"><a href="`, scripturl, `?action=membergroups;sa=edit;group=`, g.ID, `">`, c.Txt("membergroups_modify"), `</a></td>
					</tr>`)
	}
	c.O(`
					<tr class="windowbg">
						<td colspan="5" align="right" style="padding-top: 1ex; padding-bottom: 2ex;">
							<input type="submit" value="`, c.Txt("membergroups_add_group"), `" style="margin: 4px;" />
						</td>
					</tr>
				</table>
				<input type="hidden" name="sc" value="`, c.Sc, `" />
				<input type="hidden" name="postgroup" value="1" />
			</form>
		</div>`)
}

// templateMembergroupNew is template_new_group().
func templateMembergroupNew(c *Ctx) {
	page := c.Page.(*MGNewCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
		<form action="`, scripturl, `?action=membergroups;sa=add" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="90%" cellpadding="4" cellspacing="0" border="0" class="tborder" align="center">
				<tr class="titlebg">
					<td colspan="2" align="center">`, c.Txt("membergroups_new_group"), `</td>
				</tr><tr class="windowbg2">
					<th align="right" width="50%"><label for="group_name_input">`, c.Txt("membergroups_group_name"), `:</label></th>
					<td><input type="text" name="group_name" id="group_name_input" size="30" /></td>
				</tr>`)
	if page.UndefinedGroup {
		c.O(`
				<tr class="windowbg2">
					<th align="right"><label for="postgroup_based_check">`, c.Txt("membergroups_edit_post_group"), `:</label></th>
					<td>
						<input type="hidden" name="postgroup_based" value="0" />
						<input type="checkbox" name="postgroup_based" id="postgroup_based_check" value="1" onclick="updateStatus();" class="check" />
					</td>
				</tr>`)
	}
	if page.PostGroup || page.UndefinedGroup {
		c.O(`
				<tr class="windowbg2">
					<th align="right">`, c.Txt("membergroups_min_posts"), `:</th>
					<td>
						<input type="text" name="min_posts" id="min_posts_input" size="5" />
					</td>
				</tr>`)
	}
	if !page.PostGroup || !a.SettingEmpty("permission_enable_postgroups") {
		c.O(`
				<tr class="windowbg2">
					<th align="right" valign="top" style="padding-top: 1em;">
						<label for="permission_base">`, c.Txt("membergroups_permissions"), `:</label>
						<div class="smalltext" style="font-weight: normal;">`, c.Txt("membergroups_can_edit_later"), `</div>
					</th>
					<td>
						<fieldset id="permission_base">
							<legend>`, c.Txt("membergroups_select_permission_type"), `</legend>
							<input type="radio" name="perm_type" id="perm_type_predefined" value="predefined" checked="checked" class="check" />
							<label for="perm_type_predefined">`, c.Txt("membergroups_new_as_type"), `:</label>
							<select name="level" id="level_select" onclick="document.getElementById('perm_type_predefined').checked = true;">
								<option value="restrict">`, c.Txt("permitgroups_restrict"), `</option>
								<option value="standard" selected="selected">`, c.Txt("permitgroups_standard"), `</option>
								<option value="moderator">`, c.Txt("permitgroups_moderator"), `</option>
								<option value="maintenance">`, c.Txt("permitgroups_maintenance"), `</option>
							</select><br />

							<input type="radio" name="perm_type" id="perm_type_copy" value="copy" class="check" />
							<label for="perm_type_copy">`, c.Txt("membergroups_new_as_copy"), `:</label>
							<select name="copyperm" id="copyperm_select" onclick="document.getElementById('perm_type_copy').checked = true;">
								<option value="-1">`, c.Txt("membergroups_guests"), `</option>
								<option value="0">`, c.Txt("membergroups_members"), `</option>`)
		for _, g := range page.Groups {
			c.O(`
								<option value="`, g[0], `">`, g[1], `</option>`)
		}
		c.O(`
							</select>
						</fieldset>
					</td>
				</tr>`)
	}
	postGroupNote := ""
	if page.PostGroup {
		postGroupNote = `<div class="smalltext" style="font-weight: normal">` + c.Txt("membergroups_new_board_post_groups") + `</div>`
	}
	c.O(`
				<tr class="windowbg2">
					<th align="right" valign="top" style="padding-top: 1em;">
						`, c.Txt("membergroups_new_board"), `:`, postGroupNote, `
					</th>
					<td>
						<fieldset id="visible_boards">
							<legend>`, c.Txt("membergroups_new_board_desc"), `</legend>`)
	for _, b := range page.Boards {
		checked := ""
		if b.Selected {
			checked = ` checked="checked" disabled="disabled"`
		}
		c.O(`
							<div style="margin-left: `, b.ChildLevel, `em;"><input type="checkbox" name="boardaccess[]" id="boardaccess_`, b.ID, `" value="`, b.ID, `"`, checked, ` class="check" /> <label for="boardaccess_`, b.ID, `">`, b.Name, `</label></div>`)
	}
	c.O(`<br />
							<input type="checkbox" id="checkall_check" class="check" onclick="invertAll(this, this.form, 'boardaccess');" /> <label for="checkall_check"><i>`, c.Txt("737"), `</i></label>
						</fieldset>
					</td>
				</tr><tr class="windowbg2">
					<td colspan="2" align="right"><br /><input type="submit" value="`, c.Txt("membergroups_add_group"), `" /></td>
				</tr>
			</table>`)
	if page.UndefinedGroup {
		c.O(`
			<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
				function updateStatus()
				{
					var postgroupBased = document.getElementById('postgroup_based_check').checked;
					document.getElementById('min_posts_input').disabled = !postgroupBased;`)
		if a.SettingEmpty("permission_enable_postgroups") {
			c.O(`
					document.getElementById('perm_type_predefined').disabled = postgroupBased;
					document.getElementById('perm_type_copy').disabled = postgroupBased;
					document.getElementById('level_select').disabled = postgroupBased;
					document.getElementById('copyperm_select').disabled = postgroupBased;`)
		}
		c.O(`
				}
				updateStatus();
			// ]]></script>`)
	}
	c.O(`
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateMembergroupEdit is template_edit_group().
func templateMembergroupEdit(c *Ctx) {
	page := c.Page.(*MGEditCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<form action="`, scripturl, `?action=membergroups;sa=edit;group=`, page.ID, `" method="post" accept-charset="`, c.CharacterSet, `" name="groupForm" id="groupForm">
			<table width="95%" border="0" cellspacing="0" cellpadding="3" class="tborder" align="center">
				<tr class="titlebg">
					<td colspan="2" align="center">`, c.Txt("membergroups_edit_group"), ` - `, page.Name, `</td>
				</tr>
				<tr class="windowbg2">
					<th align="right" width="50%"><label for="group_name_input">`, c.Txt("membergroups_edit_name"), `:</label></th>
					<td><input type="text" name="group_name" id="group_name_input" value="`, page.EditableName, `" size="30" /></td>
				</tr>`)
	if page.AllowPostGroup {
		pgChecked := ""
		minPostsVal := ""
		if page.IsPostGroup {
			pgChecked = ` checked="checked"`
			minPostsVal = ` value="` + itoa(page.MinPosts) + `"`
		}
		c.O(`
				<tr class="windowbg2">
					<th align="right"><label for="post_group_check">`, c.Txt("membergroups_edit_post_group"), `:</label></th>
					<td><input type="checkbox" name="post_group" id="post_group_check" value="1"`, pgChecked, ` onclick="swapPostGroup(this.checked);" class="check" /></td>
				</tr>
				<tr class="windowbg2">
					<th align="right" id="min_posts_text"><label for="min_posts_input">`, c.Txt("membergroups_min_posts"), `:</label></th>
					<td><input type="text" name="min_posts" id="min_posts_input"`, minPostsVal, ` size="6" /></td>
				</tr>`)
	}
	starPrev := "blank.gif"
	if page.StarImage != "" {
		starPrev = page.StarImage
	}
	maxMsgDisabled := ""
	maxMsgVal := itoa(page.MaxMessages)
	if page.ID == 1 {
		maxMsgDisabled = `disabled="disabled"`
		maxMsgVal = "0"
	}
	c.O(`
				<tr class="windowbg2">
					<th align="right"><label for="online_color_input">`, c.Txt("membergroups_online_color"), `:</label></th>
					<td><input type="text" name="online_color" id="online_color_input" value="`, page.Color, `" size="20" /></td>
				</tr>
				<tr class="windowbg2">
					<th align="right"><label for="star_count_input">`, c.Txt("membergroups_star_count"), `:</label></th>
					<td style="padding-bottom: 0;"><input type="text" name="star_count" id="star_count_input" value="`, page.StarCount, `" size="4" onkeyup="if (this.value.length > 2) this.value = 99;" onkeydown="this.onkeyup();" onchange="if (this.value != 0) this.form.star_image.onchange();" /></td>
				</tr>
				<tr class="windowbg2">
					<th align="right" style="padding-top: 1em;">
						<label for="star_image_input">`, c.Txt("membergroups_star_image"), `:</label>
						<div class="smalltext" style="font-weight: normal;">`, c.Txt("membergroups_star_image_note"), `</div>
					</th>
					<td>
						`, c.Txt("membergroups_images_url"), `
						<input type="text" name="star_image" id="star_image_input" value="`, page.StarImage, `" onchange="if (this.value &amp;&amp; this.form.star_count.value == 0) this.form.star_count.value = 1; else if (!this.value) this.form.star_count.value = 0; document.getElementById('star_preview').src = smf_images_url + '/' + (this.value &amp;&amp; this.form.star_count.value > 0 ? this.value.replace(/\$language/g, '`, c.User.Language, `') : 'blank.gif');" size="20" />
						<img id="star_preview" src="`, imagesURL, `/`, starPrev, `" alt="*" />
					</td>
				</tr>
				<tr class="windowbg2">
					<th align="right" style="padding-top: 1em;">
						<label for="max_messages_input">`, c.Txt("membergroups_max_messages"), `:</label>
						<div class="smalltext" style="font-weight: normal">`, c.Txt("membergroups_max_messages_note"), `</div>
					</th>
					<td>
						<input type="text" name="max_messages" id="max_messages_input" value="`, maxMsgVal, `" size="6" `, maxMsgDisabled, `/>
					</td>
				</tr>`)
	if len(page.Boards) > 0 {
		boardNote := ""
		if page.IsPostGroup {
			boardNote = `<div class="smalltext" style="font-weight: normal">` + c.Txt("membergroups_new_board_post_groups") + `</div>`
		}
		c.O(`
				<tr class="windowbg2">
					<th align="right" valign="top">
						`, c.Txt("membergroups_new_board"), `:`, boardNote, `
					</th>
					<td valign="top">
						<fieldset id="visible_boards">
							<legend><a href="javascript:void(0);" onclick="document.getElementById('visible_boards').style.display = 'none';document.getElementById('visible_boards_link').style.display = 'block'; return false;">`, c.Txt("membergroups_new_board_desc"), `</a></legend>`)
		for _, b := range page.Boards {
			checked := ""
			if b.Selected {
				checked = ` checked="checked"`
			}
			c.O(`
							<div style="margin-left: `, b.ChildLevel, `em;"><input type="checkbox" name="boardaccess[]" id="boardaccess_`, b.ID, `" value="`, b.ID, `"`, checked, ` class="check" /> <label for="boardaccess_`, b.ID, `">`, b.Name, `</label></div>`)
		}
		c.O(`<br />
							<input type="checkbox" id="checkall_check" class="check" onclick="invertAll(this, this.form, 'boardaccess');" /> <label for="checkall_check"><i>`, c.Txt("737"), `</i></label>
						</fieldset>
						<a href="javascript:void(0);" onclick="document.getElementById('visible_boards').style.display = 'block'; document.getElementById('visible_boards_link').style.display = 'none'; return false;" id="visible_boards_link" style="display: none;">[ `, c.Txt("membergroups_select_visible_boards"), ` ]</a>
						<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
							document.getElementById("visible_boards_link").style.display = "";
							document.getElementById("visible_boards").style.display = "none";
						// ]]></script>
					</td>
				</tr>`)
	}
	deleteBtn := ""
	if page.AllowDelete {
		deleteBtn = `
						<input type="submit" name="delete" value="` + c.Txt("membergroups_delete") + `" onclick="return confirm('` + c.Txt("membergroups_confirm_delete") + `');" />`
	}
	c.O(`
				<tr class="windowbg2">
					<td colspan="2" align="right" style="padding-top: 1ex;">
						<input type="submit" name="submit" value="`, c.Txt("membergroups_edit_save"), `" />`, deleteBtn, `
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
	if page.AllowPostGroup {
		isPost := "false"
		if page.IsPostGroup {
			isPost = "true"
		}
		c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function swapPostGroup(isChecked)
			{
				var min_posts_text = document.getElementById('min_posts_text');
				document.forms.groupForm.min_posts.disabled = !isChecked;
				min_posts_text.style.color = isChecked ? "" : "#888888";
			}
			swapPostGroup(`, isPost, `);
		// ]]></script>`)
	}
}

// templateMembergroupMembers is template_group_members().
func templateMembergroupMembers(c *Ctx) {
	page := c.Page.(*MGMembersCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	sortImg := func(col string) string {
		if page.SortBy == col {
			return ` <img src="` + imagesURL + `/sort_` + page.SortDirection + `.gif" alt="" />`
		}
		return ""
	}
	sortDesc := func(col string) string {
		if page.SortBy == col && page.SortDirection == "up" {
			return ";desc"
		}
		return ""
	}
	link := func(col, label string) string {
		return `<a href="` + scripturl + `?action=membergroups;sa=members;start=` + itoa(page.Start) + `;sort=` + col + sortDesc(col) + `;group=` + itoa(page.GroupID) + `">` + label + sortImg(col) + `</a>`
	}

	c.O(`
		<form action="`, scripturl, `?action=membergroups;sa=members;group=`, page.GroupID, `" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="90%" cellpadding="4" cellspacing="1" border="0" class="bordercolor" align="center">
				<tr class="titlebg">
					<td colspan="6" align="left">`, page.PageTitle, `</td>
				</tr>
				<tr class="catbg">
					<td colspan="6" align="left">`, c.Txt("139"), `: `, page.PageIndex, `</td>
				</tr>
				<tr class="titlebg">
					<td>`, link("name", c.Txt("68")), `</td>
					<td>`, link("email", c.Txt("69")), `</td>
					<td>`, link("active", c.Txt("attachment_manager_last_active")), `</td>
					<td>`, link("registered", c.Txt("233")), `</td>
					<td`, colspanIf(!page.Assignable), `>`, link("posts", c.Txt("21")), `</td>`)
	if page.Assignable {
		c.O(`
					<td width="4%" align="center"><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></td>`)
	}
	c.O(`
				</tr>`)

	if len(page.Members) == 0 {
		c.O(`
				<tr class="windowbg2">
					<td colspan="6" align="center">`, c.Txt("membergroups_members_no_members"), `</td>
				</tr>`)
	}
	for _, m := range page.Members {
		c.O(`
				<tr class="windowbg2">
					<td>`, m.Name, `</td>
					<td>`, m.Email, `</td>
					<td class="windowbg">`, m.LastOnline, `</td>
					<td class="windowbg">`, m.Registered, `</td>
					<td`, colspanIf(!page.Assignable), `>`, m.Posts, `</td>`)
		if page.Assignable {
			c.O(`
					<td align="center" width="4%"><input type="checkbox" name="rem[]" value="`, m.ID, `" class="check" /></td>`)
		}
		c.O(`
				</tr>`)
	}
	if page.Assignable {
		c.O(`
				<tr class="titlebg">
					<td colspan="6" align="right">
						<input type="submit" name="remove" value="`, c.Txt("membergroups_members_remove"), `!" style="font-weight: normal;" />
					</td>
				</tr>`)
	}
	c.O(`
			</table><br />`)

	if page.Assignable {
		c.O(`
			<table width="90%" cellpadding="4" cellspacing="0" border="0" class="tborder" align="center">
				<tr class="titlebg">
					<td align="left" colspan="2">`, c.Txt("membergroups_members_add_title"), `</td>
				</tr><tr class="windowbg2">
					<td align="right" width="50%"><b>`, c.Txt("membergroups_members_add_desc"), `:</b></td>
					<td align="left">
						<input type="text" name="toAdd" id="toAdd" size="30" />
						<a href="`, scripturl, `?action=findmember;input=toAdd;quote;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, imagesURL, `/icons/assist.gif" alt="`, c.Txt("find_members"), `" /></a>
					</td>
				</tr><tr class="windowbg2">
					<td colspan="2" align="center">
						<input type="submit" name="add" value="`, c.Txt("membergroups_members_add"), `" />
					</td>
				</tr>
			</table>`)
	}
	c.O(`
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

func colspanIf(b bool) string {
	if b {
		return ` colspan="2"`
	}
	return ""
}

// templateMembergroupSettings is template_membergroup_settings().
func templateMembergroupSettings(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=membergroups;sa=settings" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("membergroups_settings"), `</td>
			</tr>`)
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right" valign="top">`, c.Txt("groups_manage_membergroups"), `:</td>
				<td width="50%">`)
		c.themeInlinePermissions("manage_membergroups")
		c.O(`
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("membergroups_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
