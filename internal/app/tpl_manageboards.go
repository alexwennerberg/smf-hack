package app

// Ports of Themes/default/ManageBoards.template.php.

// templateManageBoardsMain is template_main().
func templateManageBoardsMain(c *Ctx) {
	page := c.Page.(*ManageBoardsCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	a := c.App

	c.O(`
		<table border="0" align="center" cellspacing="1" cellpadding="4" class="bordercolor" width="100%">
			<tr class="titlebg">
				<td width="100%">
					`, c.Txt("boardsEdit"), `
				</td>
			</tr>`)
	if page.MoveBoard != 0 {
		c.O(`
			<tr class="windowbg2" height="30">
				<td>`, page.MoveTitle, ` [<a href="`, scripturl, `?action=manageboards">`, c.Txt("mboards_cancel_moving"), `</a>]</td>
			</tr>`)
	}

	for _, category := range page.Categories {
		c.O(`
			<tr>
				<td class="catbg" height="18">
					<a href="`, scripturl, `?action=manageboards;sa=cat;cat=`, category.ID, `">`, category.Name, `</a> <a href="`, scripturl, `?action=manageboards;sa=cat;cat=`, category.ID, `">`, c.Txt("catModify"), `</a>
				</td>
			</tr>`)

		c.O(`
			<tr>
				<td class="windowbg2" width="100%" valign="top">
					<form action="`, scripturl, `?action=manageboards;sa=newboard;cat=`, category.ID, `" method="post" accept-charset="`, c.CharacterSet, `">
						<table width="100%" border="0" cellpadding="1" cellspacing="0">
							<tr>
								<td style="padding-left: 1ex;" colspan="4"><b>`, c.Txt("mboards_name"), `</b></td>
							</tr>`)

		if category.MoveLink != nil {
			c.O(`
							<tr class="windowbg2">
								<td colspan="4" style="padding-left: 5px;"><a href="`, category.MoveLink.Href, `" title="`, category.MoveLink.Label, `"><img src="`, imagesURL, `/smiley_select_spot.gif" alt="`, category.MoveLink.Label, `" border="0" style="padding: 0px; margin: 0px;" /></a></td>
							</tr>`)
		}

		alternate := false
		for _, board := range category.Boards {
			alternate = !alternate
			altClass := "2"
			if alternate {
				altClass = ""
			}
			moveStyle := ""
			if board.Move {
				moveStyle = "color: red;"
			}
			recycleLink := ""
			if !a.SettingEmpty("recycle_board") && !a.SettingEmpty("recycle_enable") && a.SettingInt("recycle_board") == board.ID {
				recycleLink = `&nbsp;&nbsp;&nbsp;<a href="` + scripturl + `?action=manageboards;sa=settings"><img src="` + imagesURL + `/post/recycled.gif" alt="` + c.Txt("recycle_board") + `" border="0" /></a>`
			}
			permLink := ""
			if !a.SettingEmpty("permission_enable_by_board") && page.CanManagePermissions {
				style := ""
				if !board.LocalPermissions {
					style = ` onclick="return confirm('` + c.Txt("mboards_permissions_confirm") + `');" style="font-style: italic;"`
				}
				permLink = `<a href="` + scripturl + `?action=permissions;sa=switch;to=local;boardid=` + itoa(board.ID) + `;sesc=` + c.Sc + `"` + style + `>` + c.Txt("mboards_permissions") + `</a>`
			}
			c.O(`
							<tr class="windowbg`, altClass, `">
								<td style="padding-left: `, 5+30*board.ChildLevel, `px;`, moveStyle, `">`, board.Name, recycleLink, `</td>
								<td width="10%" align="right">`, permLink, `</td>
								<td width="10%" align="right"><a href="`, scripturl, `?action=manageboards;move=`, board.ID, `">`, c.Txt("mboards_move"), `</a></td>
								<td width="10%" style="padding-right: 1ex;" align="right"><a href="`, scripturl, `?action=manageboards;sa=board;boardid=`, board.ID, `">`, c.Txt("mboards_modify"), `</a></td>
							</tr>`)
			if len(board.MoveLinks) > 0 {
				alternate = !alternate
				altClass = "2"
				if alternate {
					altClass = ""
				}
				c.O(`
							<tr class="windowbg`, altClass, `">
								<td style="padding-left: `, 5+30*board.MoveLinks[0].ChildLevel, `px;" colspan="4">`)
				for _, link := range board.MoveLinks {
					childSuffix := ""
					if link.ChildLevel > 0 {
						childSuffix = "_child"
					}
					c.O(`<a href="`, link.Href, `" style="padding-right: 13px;padding-left: 0px;" title="`, link.Label, `"><img src="`, imagesURL, `/board_select_spot`, childSuffix, `.gif" alt="`, link.Label, `" border="0" style="padding: 0px; margin: 0px;" /></a>`)
				}
				c.O(`
								</td>
							</tr>`)
			}
		}

		c.O(`
							<tr>
								<td colspan="4" align="right"><br /><input type="submit" value="`, c.Txt("mboards_new_board"), `" /></td>
							</tr>
						</table>
						<input type="hidden" name="sc" value="`, c.Sc, `" />
					</form>
				</td>
			</tr>`)
	}
	c.O(`
		</table>`)
}

// templateModifyCategory is template_modify_category().
func templateModifyCategory(c *Ctx) {
	page := c.Page.(*EditCategoryCtx)
	scripturl := c.App.ScriptURL

	titleKey := "catEdit"
	if page.IsNew {
		titleKey = "mboards_new_cat_name"
	}
	c.O(`
<form action="`, scripturl, `?action=manageboards;sa=cat2" method="post" accept-charset="`, c.CharacterSet, `">
	<input type="hidden" name="cat" value="`, page.ID, `" />

	<table border="0" width="500" cellspacing="0" cellpadding="0" class="bordercolor" align="center">
		<tr><td>
			<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
				<tr class="titlebg">
					<td>`, c.Txt(titleKey), `</td>
				</tr><tr>
					<td class="windowbg" valign="top">
						<table border="0" width="100%" cellspacing="0" cellpadding="2">
							<tr>`)

	if len(page.Order) > 1 {
		c.O(`
								<td>
									<b>`, c.Txt("43"), `</b><br />
									<br /><br />
								</td>
								<td valign="top" align="right">
									<select name="cat_order">`)
		for _, order := range page.Order {
			selected := ""
			if order.Selected {
				selected = ` selected="selected"`
			}
			c.O(`
										<option`, selected, ` value="`, order.ID, `">`, order.Name, `</option>`)
		}
		c.O(`
									</select>
								</td>
							</tr><tr>`)
	}

	collapseChecked := ""
	if page.CanCollapse {
		collapseChecked = ` checked="checked"`
	}
	c.O(`
								<td>
									<b>`, c.Txt("44"), `:</b><br />
									`, c.Txt("672"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<input type="text" name="cat_name" value="`, page.EditableName, `" size="30" tabindex="1" />
								</td>
							</tr><tr>
								<td>
									<b>`, c.Txt("collapse_enable"), `</b><br />
									`, c.Txt("collapse_desc"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<input type="checkbox" name="collapse"`, collapseChecked, ` tabindex="2" class="check" />
								</td>
							</tr>`)

	c.O(`
							<tr>
								<td colspan="2" align="right">
									<br />`)
	if page.IsNew {
		c.O(`
									<input type="submit" name="add" value="`, c.Txt("mboards_add_cat_button"), `" onclick="return !isEmptyText(this.form.cat_name);" tabindex="3" />`)
	} else {
		c.O(`
									<input type="submit" name="edit" value="`, c.Txt("17"), `" onclick="return !isEmptyText(this.form.cat_name);" tabindex="3" />
									<input type="submit" name="delete" value="`, c.Txt("mboards_delete_cat"), `" onclick="return confirm('`, c.Txt("catConfirm"), `');" />`)
	}
	c.O(`
								</td>
							</tr>
						</table>
						<input type="hidden" name="sc" value="`, c.Sc, `" />`)
	if page.IsEmpty {
		c.O(`
						<input type="hidden" name="empty" value="1" />`)
	}
	c.O(`
					</td>
				</tr>
			</table>
		</td></tr>
	</table>
</form>`)
}

// templateConfirmCategoryDelete is template_confirm_category_delete().
func templateConfirmCategoryDelete(c *Ctx) {
	page := c.Page.(*EditCategoryCtx)
	scripturl := c.App.ScriptURL

	c.O(`
<form action="`, scripturl, `?action=manageboards;sa=cat2" method="post" accept-charset="`, c.CharacterSet, `">
	<input type="hidden" name="cat" value="`, page.ID, `" />

	<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
		<tr class="titlebg">
			<td>`, c.Txt("mboards_delete_cat"), `</td>
		</tr><tr class="windowbg">
			<td class="windowbg" valign="top">
				`, c.Txt("mboards_delete_cat_contains"), `:
				<ul>`)
	for _, child := range page.Children {
		c.O(`
					<li>`, child, `</li>`)
	}
	c.O(`
				</ul>
			</td>
		</tr>
	</table>
	<br />
	<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
		<tr class="titlebg">
			<td>`, c.Txt("mboards_delete_what_do"), `:</td>
		</tr>
		<tr>
			<td class="windowbg2">`)
	disabled := ""
	if len(page.Order) == 1 {
		disabled = ` disabled="disabled"`
	}
	c.O(`
				<label for="delete_action0"><input type="radio" id="delete_action0" name="delete_action" value="0" class="check" checked="checked" />`, c.Txt("mboards_delete_option1"), `</label><br />
				<label for="delete_action1"><input type="radio" id="delete_action1" name="delete_action" value="1" class="check"`, disabled, ` />`, c.Txt("mboards_delete_option2"), `</label>:
				<select name="cat_to" `, disabled, `>`)
	for _, cat := range page.Order {
		if cat.ID != 0 {
			c.O(`
					<option value="`, cat.ID, `">`, cat.TrueName, `</option>`)
		}
	}
	c.O(`
				</select>
			</td>
		</tr>
		<tr>
			<td align="center" class="windowbg2">
				<input type="submit" name="delete" value="`, c.Txt("mboards_delete_confirm"), `" />
				<input type="submit" name="cancel" value="`, c.Txt("mboards_delete_cancel"), `" />
			</td>
		</tr>
	</table>

	<input type="hidden" name="confirmation" value="1" />
	<input type="hidden" name="sc" value="`, c.Sc, `" />
</form>`)
}

// templateModifyBoard is template_modify_board().
func templateModifyBoard(c *Ctx) {
	page := c.Page.(*EditBoardCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	a := c.App

	titleKey := "boardsEdit"
	if page.IsNew {
		titleKey = "mboards_new_board_name"
	}
	c.O(`
<form action="`, scripturl, `?action=manageboards;sa=board2" method="post" accept-charset="`, c.CharacterSet, `">
	<input type="hidden" name="boardid" value="`, page.ID, `" />

	<table border="0" width="540" cellspacing="0" cellpadding="0" class="bordercolor" align="center">
		<tr><td>
			<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
				<tr class="titlebg">
					<td>`, c.Txt(titleKey), `</td>
				</tr><tr>
					<td class="windowbg" valign="top">
						<table border="0" width="100%" cellspacing="0" cellpadding="2">`)

	c.O(`
							<tr>
								<td>
									<b>`, c.Txt("mboards_category"), `</b><br />
									<br /><br />
								</td>
								<td valign="top" align="right">
									<select name="new_cat" onchange="if (this.form.order) {this.form.order.disabled = this.options[this.selectedIndex].value != 0; this.form.boardOrder.disabled = this.options[this.selectedIndex].value != 0 || this.form.order.options[this.form.order.selectedIndex].value == '';}">`)
	for _, category := range page.Categories {
		selected := ""
		if category.Selected {
			selected = ` selected="selected"`
		}
		c.O(`
										<option`, selected, ` value="`, category.ID, `">`, category.Name, `</option>`)
	}
	c.O(`
									</select>
								</td>
							</tr><tr>`)

	if (page.IsNew && len(page.BoardOrder) > 0) || len(page.BoardOrder) > 1 {
		c.O(`
								<td>
									<b>`, c.Txt("43"), `</b><br />
									<br /><br />
								</td>
								<td valign="top" align="right">`)
		unchangedOpt := ""
		if !page.IsNew {
			unchangedOpt = `<option value="">(` + c.Txt("mboards_unchanged") + `)</option>`
		}
		c.O(`
									<select id="order" name="placement" onchange="this.form.boardOrder.disabled = this.options[this.selectedIndex].value == '';">
										`, unchangedOpt, `
										<option value="before">`, c.Txt("mboards_order_before"), `...</option>
										<option value="child">`, c.Txt("mboards_order_child_of"), `...</option>
										<option value="after">`, c.Txt("mboards_order_after"), `...</option>
									</select>&nbsp;&nbsp;`)
		boardOrderDisabled := `disabled="disabled"`
		if page.IsNew {
			boardOrderDisabled = ""
		}
		c.O(`
									<select id="boardOrder" name="board_order" `, boardOrderDisabled, `>
										`, unchangedOpt)
		for _, order := range page.BoardOrder {
			selected := ""
			if order.Selected {
				selected = ` selected="selected"`
			}
			c.O(`
										<option`, selected, ` value="`, order.ID, `">`, order.Name, `</option>`)
		}
		c.O(`
									</select>
								</td>
							</tr><tr>`)
	}

	c.O(`
								<td>
									<b>`, c.Txt("44"), `:</b><br />
									`, c.Txt("672"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<input type="text" name="board_name" value="`, page.Name, `" size="30" />
								</td>
							</tr><tr>
								<td>
									<b>`, c.Txt("mboards_description"), `</b><br />
									`, c.Txt("mboards_description_desc"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<textarea name="desc" rows="2" cols="29">`, page.Description, `</textarea>
								</td>
							</tr><tr>
								<td valign="top">
									<b>`, c.Txt("mboards_groups"), `</b><br />
									`, c.Txt("mboards_groups_desc"), `<br /><br />
								</td>
								<td valign="top" align="right">`)
	for _, group := range page.Groups {
		postGroupAttr := ""
		if group.IsPostGroup {
			postGroupAttr = ` style="border-bottom: 1px dotted;" title="` + c.Txt("mboards_groups_post_group") + `"`
		}
		checked := ""
		if group.Checked {
			checked = ` checked="checked"`
		}
		c.O(`
									<label for="groups_`, group.ID, `"><span`, postGroupAttr, `>`, group.Name, `</span> <input type="checkbox" name="groups[]" value="`, group.ID, `" id="groups_`, group.ID, `"`, checked, ` /></label><br />`)
	}
	c.O(`
									<i>`, c.Txt("737"), `</i> <input type="checkbox" onclick="invertAll(this, this.form, 'groups[]');" /><br />
									<br />
								</td>
							</tr>`)

	if a.SettingEmpty("permission_enable_by_board") {
		check := func(mode string) string {
			if page.PermissionMode == mode {
				return ` checked="checked"`
			}
			return ""
		}
		c.O(`
							<tr>
								<td valign="top">
									<b>`, c.Txt("mboards_permissions_title"), `</b><br />
									`, c.Txt("mboards_permissions_desc"), `<br />
								</td>
								<td align="right">
									`, c.Txt("permission_mode_normal"), ` <input type="radio" name="permission_mode" value="0" class="check"`, check("normal"), ` /><br />
									`, c.Txt("permission_mode_no_polls"), ` <input type="radio" name="permission_mode" value="2" class="check"`, check("no_polls"), ` /><br />
									`, c.Txt("permission_mode_reply_only"), ` <input type="radio" name="permission_mode" value="3" class="check"`, check("reply_only"), ` /><br />
									`, c.Txt("permission_mode_read_only"), ` <input type="radio" name="permission_mode" value="4" class="check"`, check("read_only"), ` /><br />
									<br />
								</td>
							</tr>`)
	}

	countChecked := ""
	if page.CountPosts {
		countChecked = ` checked="checked"`
	}
	c.O(`
							<tr>
								<td>
									<b>`, c.Txt("mboards_moderators"), `</b><br />
									`, c.Txt("mboards_moderators_desc"), `<br /><br />
								</td>
								<td valign="top" align="right" style="white-space: nowrap;">
									<input type="text" name="moderators" id="moderators" value="`, page.ModeratorList, `" size="30" />
									<a href="`, scripturl, `?action=findmember;input=moderators;quote;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, imagesURL, `/icons/assist.gif" alt="`, c.Txt("find_members"), `" /></a>
								</td>
							</tr><tr>
								<td>
									<b>`, c.Txt("mboards_count_posts"), `</b><br />
									`, c.Txt("mboards_count_posts_desc"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<input type="checkbox" name="count" `, countChecked, ` class="check" />
								</td>
							</tr>`)

	themeDefSel := ""
	if page.Theme == 0 {
		themeDefSel = ` selected="selected"`
	}
	c.O(`
							<tr>
								<td>
									<b>`, c.Txt("mboards_theme"), `</b><br />
									`, c.Txt("mboards_theme_desc"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<select name="boardtheme">
										<option value="0"`, themeDefSel, `>`, c.Txt("mboards_theme_default"), `</option>`)
	for _, theme := range page.Themes {
		selected := ""
		if page.Theme == theme.ID {
			selected = ` selected="selected"`
		}
		c.O(`
										<option value="`, theme.ID, `"`, selected, `>`, theme.Name, `</option>`)
	}
	overrideChecked := ""
	if page.OverrideTheme != 0 {
		overrideChecked = ` checked="checked"`
	}
	c.O(`
									</select>
								</td>
							</tr><tr>
								<td>
									<b>`, c.Txt("mboards_override_theme"), `</b><br />
									`, c.Txt("mboards_override_theme_desc"), `<br /><br />
								</td>
								<td valign="top" align="right">
									<input type="checkbox" name="override_theme"`, overrideChecked, ` class="check" />
								</td>
							</tr>`)

	c.O(`
							<tr>
								<td colspan="2" align="right">
									<br />`)
	if page.IsNew {
		c.O(`
									<input type="hidden" name="cur_cat" value="`, page.Category, `">
									<input type="submit" name="add" value="`, c.Txt("mboards_new_board"), `" onclick="return !isEmptyText(this.form.board_name);" />`)
	} else {
		c.O(`
									<input type="submit" name="edit" value="`, c.Txt("17"), `" onclick="return !isEmptyText(this.form.board_name);" />
									<input type="submit" name="delete" value="`, c.Txt("mboards_delete_board"), `" onclick="return confirm('`, c.Txt("boardConfirm"), `');" />`)
	}
	c.O(`
								</td>
							</tr>
						</table>
						<input type="hidden" name="sc" value="`, c.Sc, `" />`)
	if page.NoChildren {
		c.O(`
						<input type="hidden" name="no_children" value="1" />`)
	}
	c.O(`
					</td>
				</tr>
			</table>
		</td></tr>
	</table>
</form>`)
}

// templateConfirmBoardDelete is template_confirm_board_delete().
func templateConfirmBoardDelete(c *Ctx) {
	page := c.Page.(*EditBoardCtx)
	scripturl := c.App.ScriptURL

	c.O(`
<form action="`, scripturl, `?action=manageboards;sa=board2" method="post" accept-charset="`, c.CharacterSet, `">
	<input type="hidden" name="boardid" value="`, page.ID, `" />

	<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
		<tr class="titlebg">
			<td>`, c.Txt("mboards_delete_board"), `</td>
		</tr><tr class="windowbg">
			<td class="windowbg" valign="top">
				`, c.Txt("mboards_delete_board_contains"), `:
				<ul>`)
	for _, child := range page.Children {
		c.O(`
					<li>`, child, `</li>`)
	}
	c.O(`
				</ul>
			</td>
		</tr>
	</table>
	<br />
	<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
		<tr class="titlebg">
			<td>`, c.Txt("mboards_delete_what_do"), `:</td>
		</tr>
		<tr>
			<td class="windowbg2">`)
	disabled := ""
	if !page.CanMoveChildren {
		disabled = ` disabled="disabled"`
	}
	c.O(`
				<label for="delete_action0"><input type="radio" id="delete_action0" name="delete_action" value="0" class="check" checked="checked" />`, c.Txt("mboards_delete_board_option1"), `</label><br />
				<label for="delete_action1"><input type="radio" id="delete_action1" name="delete_action" value="1" class="check"`, disabled, ` />`, c.Txt("mboards_delete_board_option2"), `</label>:
				<select name="board_to" `, disabled, `>`)
	for _, board := range page.BoardOrder {
		if board.ID != page.ID && !board.IsChild {
			c.O(`
					<option value="`, board.ID, `">`, board.Name, `</option>`)
		}
	}
	c.O(`
				</select>
			</td>
		</tr>
		<tr>
			<td align="center" class="windowbg2">
				<input type="submit" name="delete" value="`, c.Txt("mboards_delete_confirm"), `" />
				<input type="submit" name="cancel" value="`, c.Txt("mboards_delete_cancel"), `" />
			</td>
		</tr>
	</table>

	<input type="hidden" name="confirmation" value="1" />
	<input type="hidden" name="sc" value="`, c.Sc, `" />
</form>`)
}

// templateModifyGeneralSettings is template_modify_general_settings().
func templateModifyGeneralSettings(c *Ctx) {
	page := c.Page.(*BoardSettingsCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=manageboards;sa=settings" method="post" accept-charset="`, c.CharacterSet, `"s>
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("settings"), `</td>
			</tr>`)

	if page.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right" valign="top">`, c.Txt("groups_manage_boards"), `:</td>
				<td width="50%">`)
		c.themeInlinePermissions("manage_boards")
		c.O(`
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr>`)
	}

	countChecked := ""
	if !a.SettingEmpty("countChildPosts") {
		countChecked = ` checked="checked"`
	}
	recycleChecked := ""
	if !a.SettingEmpty("recycle_enable") {
		recycleChecked = ` checked="checked"`
	}
	recycleBoard := "0"
	if !a.SettingEmpty("recycle_board") {
		recycleBoard = a.Setting("recycle_board")
	}
	c.O(`
			<tr class="windowbg2">
				<th width="50%" align="right"><label for="countChildPosts_check">`, c.Txt("countChildPosts"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=countChildPosts" onclick="return reqWin(this.href);">?</a>)</span>:</th>
				<td>
					<input type="checkbox" name="countChildPosts" id="countChildPosts_check"`, countChecked, ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right"><label for="recycle_enable_check">`, c.Txt("recycle_enable"), `</label> <span style="font-weight: normal;">(<a href="`, scripturl, `?action=helpadmin;help=recycle_enable" onclick="return reqWin(this.href);">?</a>)</span>:</th>
				<td>
					<input type="checkbox" name="recycle_enable" id="recycle_enable_check"`, recycleChecked, ` class="check" onclick="document.getElementById('recycle_board_select').disabled = !this.checked;" />
				</td>
			</tr><tr class="windowbg2">
				<th align="right">`, c.Txt("recycle_board"), `:</th>
				<td>
					<input type="hidden" name="recycle_board" value="`, recycleBoard, `" />
					<select name="recycle_board" id="recycle_board_select">
						<option></option>`)
	for _, board := range page.Boards {
		selected := ""
		if board.IsRecycle {
			selected = ` selected="selected"`
		}
		c.O(`
						<option value="`, board.ID, `"`, selected, `>`, board.CategoryName, ` - `, board.Name, `</option>`)
	}
	c.O(`
					</select>
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						document.getElementById("recycle_board_select").disabled = !document.getElementById("recycle_enable_check").checked;
					// ]]></script>
				</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("mboards_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
