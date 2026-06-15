package app

import "strings"

// Ports of Themes/default/ManagePermissions.template.php:
// template_permission_index / modify_group / general_permission_settings /
// inline_permissions. (template_by_board is deferred — board-level permissions
// are off by default in this port.)

// themeInlinePermissions is theme_inline_permissions($permission).
func (c *Ctx) themeInlinePermissions(permission string) {
	c.CurrentPermission = permission
	templateInlinePermissions(c)
}

// templatePermissionIndex is template_permission_index().
func templatePermissionIndex(c *Ctx) {
	page := c.Page.(*PermIndexCtx)
	scripturl := c.App.ScriptURL
	a := c.App
	enableDeny := !a.SettingEmpty("permission_enable_deny")

	c.O(`
		<form action="`, scripturl, `?action=permissions;sa=quick" method="post" accept-charset="`, c.CharacterSet, `" name="permissionForm" id="permissionForm">
			<table width="100%" border="0" cellpadding="2" cellspacing="1" class="tborder">`)
	c.O(`
				<tr class="catbg3">
					<td valign="middle">`, c.Txt("membergroups_name"), `</td>
					<td width="10%" align="center" valign="middle">`, c.Txt("membergroups_members_top"), `</td>
					<td width="16%" align="center"`)
	if enableDeny {
		c.O(` class="smalltext"`)
	}
	c.O(`>
						`, c.Txt("membergroups_permissions"))
	if enableDeny {
		c.O(`<br />
						<div style="float: left; width: 50%;">`, c.Txt("permissions_allowed"), `</div> `, c.Txt("permissions_denied"))
	}
	c.O(`
					</td>
					<td width="10%" align="center" valign="middle">`, c.Txt("permissions_modify"), `</td>
					<td width="4%" align="center" valign="middle">
						<input type="checkbox" class="check" onclick="invertAll(this, this.form, 'group');" /></td>
				</tr>`)

	for _, group := range page.Groups {
		help := ""
		switch group.ID {
		case -1:
			help = ` (<a href="` + scripturl + `?action=helpadmin;help=membergroup_guests" onclick="return reqWin(this.href);">?</a>)`
		case 0:
			help = ` (<a href="` + scripturl + `?action=helpadmin;help=membergroup_regular_members" onclick="return reqWin(this.href);">?</a>)`
		case 1:
			help = ` (<a href="` + scripturl + `?action=helpadmin;help=membergroup_administrator" onclick="return reqWin(this.href);">?</a>)`
		case 3:
			help = ` (<a href="` + scripturl + `?action=helpadmin;help=membergroup_moderator" onclick="return reqWin(this.href);">?</a>)`
		}
		membersCell := group.NumMembers
		if group.CanSearch && group.Link != "" {
			membersCell = group.Link
		}
		adminStyle := ""
		if group.ID == 1 {
			adminStyle = ` style="font-style: italic;"`
		}
		c.O(`
				<tr>
					<td class="windowbg2">`, group.Name, help, `</td>
					<td class="windowbg" align="center">`, membersCell, `</td>
					<td class="windowbg2" align="center"`, adminStyle, `>`)
		if !enableDeny {
			c.O(`
						`, group.NumAllowed)
		} else {
			deniedCell := group.NumDenied
			if !empty(group.NumDenied) && group.ID != 1 {
				if group.ID == -1 {
					deniedCell = `<span style="font-style: italic;">` + group.NumDenied + `</span>`
				} else {
					deniedCell = `<span style="color: red;">` + group.NumDenied + `</span>`
				}
			}
			c.O(`
						<div style="float: left; width: 50%;">`, group.NumAllowed, `</div> `, deniedCell)
		}
		c.O(`
					</td>`)
		modifyCell := ""
		modifyBox := ""
		if group.AllowModify {
			modifyCell = `<a href="` + scripturl + `?action=permissions;sa=modify;group=` + itoa(group.ID) + `">` + c.Txt("permissions_modify") + `</a>`
			modifyBox = `<input type="checkbox" name="group[]" value="` + itoa(group.ID) + `" class="check" />`
		}
		c.O(`
					<td class="windowbg2" align="center">`, modifyCell, `</td>
					<td class="windowbg" align="center">`, modifyBox, `</td>
				</tr>`)
	}

	c.O(`
				<tr class="windowbg">
					<td colspan="6" style="padding-top: 1ex; padding-bottom: 1ex; text-align: right;">
						<table width="100%" cellspacing="0" cellpadding="3" border="0"><tr><td>
							<div style="margin-bottom: 1ex;"><b>`, c.Txt("permissions_with_selection"), `...</b></div>
							`, c.Txt("permissions_apply_pre_defined"), ` <a href="`, scripturl, `?action=helpadmin;help=permissions_quickgroups" onclick="return reqWin(this.href);">(?)</a>:
							<select name="predefined">
								<option value="">(`, c.Txt("permissions_select_pre_defined"), `)</option>
								<option value="restrict">`, c.Txt("permitgroups_restrict"), `</option>
								<option value="standard">`, c.Txt("permitgroups_standard"), `</option>
								<option value="moderator">`, c.Txt("permitgroups_moderator"), `</option>
								<option value="maintenance">`, c.Txt("permitgroups_maintenance"), `</option>
							</select><br /><br />
							`, c.Txt("permissions_like_group"), `:
							<select name="copy_from">
								<option value="empty">(`, c.Txt("permissions_select_membergroup"), `)</option>`)
	for _, group := range page.Groups {
		if group.ID != 1 {
			c.O(`
								<option value="`, group.ID, `">`, group.Name, `</option>`)
		}
	}
	c.O(`
							</select><br /><br />
							<select name="add_remove">
								<option value="add">`, c.Txt("permissions_add"), `...</option>
								<option value="clear">`, c.Txt("permissions_remove"), `...</option>`)
	if enableDeny {
		c.O(`
								<option value="deny">`, c.Txt("permissions_deny"), `...</option>`)
	}
	c.O(`
							</select>&nbsp;<select name="permissions">
								<option value="">(`, c.Txt("permissions_select_permission"), `)</option>`)
	for _, pt := range page.Permissions {
		for _, col := range [][]permGroupCol{pt.Left, pt.Right} {
			for _, pg := range col {
				c.O(`
								<option value="" disabled="disabled">[`, pg.Name, `]</option>`)
				for _, perm := range pg.Permissions {
					if perm.HasOwnAny {
						c.O(`
								<option value="`, pt.ID, `/`, perm.OwnID, `">&nbsp;&nbsp;&nbsp;`, perm.Name, ` (`, perm.OwnName, `)</option>
								<option value="`, pt.ID, `/`, perm.AnyID, `">&nbsp;&nbsp;&nbsp;`, perm.Name, ` (`, perm.AnyName, `)</option>`)
					} else {
						c.O(`
								<option value="`, pt.ID, `/`, perm.ID, `">&nbsp;&nbsp;&nbsp;`, perm.Name, `</option>`)
					}
				}
			}
		}
	}
	c.O(`
							</select>
						</td><td valign="bottom" width="16%">
							<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
								function checkSubmit()
								{
									if ((document.forms.permissionForm.predefined.value != "" && (document.forms.permissionForm.copy_from.value != "empty" || document.forms.permissionForm.permissions.value != "")) || (document.forms.permissionForm.copy_from.value != "empty" && document.forms.permissionForm.permissions.value != ""))
									{
										alert("`, c.Txt("permissions_only_one_option"), `");
										return false;
									}
									if (document.forms.permissionForm.predefined.value == "" && document.forms.permissionForm.copy_from.value == "" && document.forms.permissionForm.permissions.value == "")
									{
										alert("`, c.Txt("permissions_no_action"), `");
										return false;
									}
									if (document.forms.permissionForm.permissions.value != "" && document.forms.permissionForm.add_remove.value == "deny")
										return confirm("`, c.Txt("permissions_deny_dangerous"), `");

									return true;
								}
							// ]]></script>
							<input type="submit" value="`, c.Txt("permissions_set_permissions"), `" onclick="return checkSubmit();" />
						</td></tr></table>
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateModifyGroup is template_modify_group().
func templateModifyGroup(c *Ctx) {
	page := c.Page.(*PermModifyCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	a := c.App
	enableDeny := !a.SettingEmpty("permission_enable_deny")
	radios := enableDeny && page.GroupID != -1

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			window.smf_usedDeny = false;

			function warnAboutDeny()
			{
				if (window.smf_usedDeny)
					return confirm("`, c.Txt("permissions_deny_dangerous"), `");
				else
					return true;
			}
		// ]]></script>
		<form action="`, scripturl, `?action=permissions;sa=modify2;group=`, page.GroupID, `;boardid=`, page.BoardID, `" method="post" accept-charset="`, c.CharacterSet, `" name="permissionForm" id="permissionForm" onsubmit="return warnAboutDeny();">
			<table width="100%" cellpadding="4" cellspacing="0" border="0" class="tborder">`)
	if radios {
		c.O(`
				<tr class="windowbg">
					<td colspan="2" class="smalltext" style="padding: 2ex;">`, c.Txt("permissions_option_desc"), `</td>
				</tr>`)
	}

	for _, pt := range page.Permissions {
		if !pt.Show {
			continue
		}
		c.O(`
				<tr class="catbg">
					<td colspan="2" align="center">`)
		if pt.ID == "membergroup" {
			c.O(c.Txt("permissions_general"))
		} else {
			c.O(c.Txt("permissions_board"))
		}
		c.O(` - <span style="color: red;">`, page.GroupName, `</span>
					</td>
				</tr>
				<tr class="windowbg2">`)

		for _, col := range [][]permGroupCol{pt.Left, pt.Right} {
			c.O(`
					<td valign="top" width="50%">
						<table width="100%" cellpadding="1" cellspacing="0" border="0">`)
			for _, pg := range col {
				c.O(`
							<tr class="windowbg2">
								<td colspan="2" width="100%" align="left"><div style="border-bottom: 1px solid; padding-bottom: 2px; margin-bottom: 2px;"><b>`, pg.Name, `</b></div></td>`)
				if !radios {
					c.O(`
								<td colspan="3" width="10"><div style="border-bottom: 1px solid; padding-bottom: 2px; margin-bottom: 2px;">&nbsp;</div></td>`)
				} else {
					c.O(`
								<td align="center"><div style="border-bottom: 1px solid; padding-bottom: 2px; margin-bottom: 2px;">`, c.Txt("permissions_option_on"), `</div></td>
								<td align="center"><div style="border-bottom: 1px solid; padding-bottom: 2px; margin-bottom: 2px;">`, c.Txt("permissions_option_off"), `</div></td>
								<td align="center"><div style="border-bottom: 1px solid; padding-bottom: 2px; margin-bottom: 2px; color: red;">`, c.Txt("permissions_option_deny"), `</div></td>`)
				}
				c.O(`
							</tr>`)

				alternate := false
				for _, perm := range pg.Permissions {
					rowClass := "windowbg2"
					if alternate {
						rowClass = "windowbg"
					}
					helpCell := ""
					if perm.ShowHelp {
						helpCell = `<a href="` + scripturl + `?action=helpadmin;help=permissionhelp_` + perm.ID + `" onclick="return reqWin(this.href);" class="help"><img src="` + imagesURL + `/helptopics.gif" alt="` + c.Txt("119") + `" /></a>`
					}
					c.O(`
							<tr class="`, rowClass, `">
								<td valign="top" width="10" style="padding-right: 1ex;">
									`, helpCell, `
								</td>`)
					if perm.HasOwnAny {
						c.permRadioRow(pt.ID, perm.OwnID, perm.OwnSelect, radios, perm.Name, perm.OwnName, rowClass, true)
						c.permRadioRow(pt.ID, perm.AnyID, perm.AnySelect, radios, "", perm.AnyName, rowClass, false)
					} else {
						c.O(`
								<td valign="top" width="100%" align="left" style="padding-bottom: 2px;">`, perm.Name, `</td>`)
						c.permCheckCells(pt.ID, perm.ID, perm.Select, radios, false)
						c.O(`
							</tr>`)
					}
					alternate = !alternate
				}

				c.O(`
							<tr class="windowbg2">
								<td colspan="5" width="100%"><div style="border-top: 1px solid; padding-bottom: 1.5ex; margin-top: 2px;">&nbsp;</div></td>
							</tr>`)
			}
			c.O(`
						</table>
					</td>`)
		}
	}
	c.O(`
				</tr><tr class="windowbg2">
					<td colspan="2" align="right"><input type="submit" value="`, c.Txt("permissions_commit"), `" />&nbsp;</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// permRadioRow emits the own/any sub-row of a has_own_any permission.
func (c *Ctx) permRadioRow(permType, id, selected string, radios bool, mainName, label, rowClass string, isOwn bool) {
	if isOwn {
		c.O(`
								<td colspan="4" width="100%" valign="top" align="left">`, mainName, `</td>
							</tr><tr class="`, rowClass, `">
								<td></td>
								<td width="100%" class="smalltext" align="right">`, label, `:</td>`)
	} else {
		c.O(`
							</tr><tr class="`, rowClass, `">
								<td></td>
								<td width="100%" class="smalltext" align="right" style="padding-bottom: 1.5ex;">`, label, `:</td>`)
	}
	pad := ""
	if !isOwn {
		pad = ` style="padding-bottom: 1.5ex;"`
	}
	if !radios {
		checked := ""
		if selected == "on" {
			checked = ` checked="checked"`
		}
		extraID := ""
		if isOwn {
			extraID = ` id="` + id + `_on"`
		}
		c.O(`
								<td colspan="3"`, pad, `><input type="checkbox" name="perm[`, permType, `][`, id, `]"`, checked, ` value="on"`, extraID, ` class="check" /></td>`)
	} else {
		onChecked, offChecked, denyChecked := "", "", ""
		switch selected {
		case "on":
			onChecked = ` checked="checked"`
		case "denied":
			denyChecked = ` checked="checked"`
		default:
			offChecked = ` checked="checked"`
		}
		onExtra := ""
		denyExtra := ` onclick="window.smf_usedDeny = true;"`
		if isOwn {
			onExtra = ` id="` + id + `_on"`
		} else {
			onExtra = ` onclick="document.forms.permissionForm.` + strings.TrimSuffix(id, "_any") + `_own_on.checked = true;"`
			denyExtra = ` id="` + id + `_deny" onclick="window.smf_usedDeny = true;"`
		}
		c.O(`
								<td valign="top" width="10"`, pad, `><input type="radio" name="perm[`, permType, `][`, id, `]"`, onChecked, ` value="on"`, onExtra, ` class="check" /></td>
								<td valign="top" width="10"`, pad, `><input type="radio" name="perm[`, permType, `][`, id, `]"`, offChecked, ` value="off" class="check" /></td>
								<td valign="top" width="10"`, pad, `><input type="radio" name="perm[`, permType, `][`, id, `]"`, denyChecked, ` value="deny"`, denyExtra, ` class="check" /></td>`)
	}
	if !isOwn {
		c.O(`
							</tr>`)
	}
}

// permCheckCells emits the on/off/deny cells for a simple permission.
func (c *Ctx) permCheckCells(permType, id, selected string, radios, pad bool) {
	if !radios {
		checked := ""
		if selected == "on" {
			checked = ` checked="checked"`
		}
		c.O(`
								<td valign="top" style="padding-bottom: 2px;"><input type="checkbox" name="perm[`, permType, `][`, id, `]"`, checked, ` value="on" class="check" /></td>`)
	} else {
		onChecked, offChecked, denyChecked := "", "", ""
		switch selected {
		case "on":
			onChecked = ` checked="checked"`
		case "denied":
			denyChecked = ` checked="checked"`
		default:
			offChecked = ` checked="checked"`
		}
		c.O(`
								<td valign="top" width="10" style="padding-bottom: 2px;"><input type="radio" name="perm[`, permType, `][`, id, `]"`, onChecked, ` value="on" class="check" /></td>
								<td valign="top" width="10" style="padding-bottom: 2px;"><input type="radio" name="perm[`, permType, `][`, id, `]"`, offChecked, ` value="off" class="check" /></td>
								<td valign="top" width="10" style="padding-bottom: 2px;"><input type="radio" name="perm[`, permType, `][`, id, `]"`, denyChecked, ` value="deny" onclick="window.smf_usedDeny = true;" class="check" /></td>`)
	}
}

// templateGeneralPermissionSettings is template_general_permission_settings().
func templateGeneralPermissionSettings(c *Ctx) {
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=permissions;sa=settings" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("permission_settings_title"), `</td>
			</tr>`)
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right" valign="top">`, c.Txt("groups_manage_permissions"), `:</td>
				<td width="50%">`)
		c.themeInlinePermissions("manage_permissions")
		c.O(`
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr>
`)
	}
	denyDeny := ""
	if !a.SettingEmpty("permission_enable_deny") {
		denyDeny = ` onclick="if (!this.checked) alert('` + c.Txt("permission_disable_deny_warning") + `');"`
	}
	denyPg := ""
	if !a.SettingEmpty("permission_enable_postgroups") {
		denyPg = ` onclick="if (!this.checked) alert('` + c.Txt("permission_disable_postgroups_warning") + `');"`
	}
	denyBoard := ""
	if !a.SettingEmpty("permission_enable_by_board") {
		denyBoard = ` onclick="if (!this.checked) alert('` + c.Txt("permission_disable_by_board_warning") + `');"`
	}
	c.O(`
			<tr class="windowbg2">
				<td width="50%" align="right"><label for="permission_enable_deny_check">`, c.Txt("permission_settings_enable_deny"), `</label> (<a href="`, scripturl, `?action=helpadmin;help=permissions_deny" onclick="return reqWin(this.href);">?</a>):</td>
				<td>
					<input type="checkbox" name="permission_enable_deny" id="permission_enable_deny_check"`, c.checkboxAttr("permission_enable_deny"), ` class="check"`, denyDeny, `/>
				</td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="permission_enable_postgroups_check">`, c.Txt("permission_settings_enable_postgroups"), `</label> (<a href="`, scripturl, `?action=helpadmin;help=permissions_postgroups" onclick="return reqWin(this.href);">?</a>):</td>
				<td>
					<input type="checkbox" name="permission_enable_postgroups" id="permission_enable_postgroups_check"`, c.checkboxAttr("permission_enable_postgroups"), ` class="check"`, denyPg, `/>
				</td>
			</tr><tr class="windowbg2">
				<td width="50%" align="right"><label for="permission_enable_by_board_check">`, c.Txt("permission_settings_enable_by_board"), `</label> (<a href="`, scripturl, `?action=helpadmin;help=permissions_by_board" onclick="return reqWin(this.href);">?</a>):</td>
				<td>
					<input type="checkbox" name="permission_enable_by_board" id="permission_enable_by_board_check"`, c.checkboxAttr("permission_enable_by_board"), ` class="check"`, denyBoard, `/>
				</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("permission_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateInlinePermissions is template_inline_permissions().
func templateInlinePermissions(c *Ctx) {
	a := c.App
	perm := c.CurrentPermission
	groups := c.InlinePermissions[perm]
	enableDeny := !a.SettingEmpty("permission_enable_deny")

	c.O(`
		<fieldset id="`, perm, `_groups">
			<legend><a href="javascript:void(0);" onclick="document.getElementById('`, perm, `_groups').style.display = 'none';document.getElementById('`, perm, `_groups_link').style.display = 'block'; return false;">`, c.Txt("avatar_select_permission"), `</a></legend>`)
	if !enableDeny {
		c.O(`
			<table width="100%" border="0">`)
	} else {
		c.O(`
			<div class="smalltext" style="padding: 2em;">`, c.Txt("permissions_option_desc"), `</div>
			<table width="100%" border="0">
				<tr>
					<th align="center">`, c.Txt("permissions_option_on"), `</th>
					<th align="center">`, c.Txt("permissions_option_off"), `</th>
					<th align="center" style="color: red;">`, c.Txt("permissions_option_deny"), `</th>
					<td></td>
				</tr>`)
	}
	for _, group := range groups {
		c.O(`
				<tr>`)
		if !enableDeny {
			checked := ""
			if group.Status == "on" {
				checked = ` checked="checked"`
			}
			c.O(`
					<td align="center"><input type="checkbox" name="`, perm, `[`, group.ID, `]" value="on"`, checked, ` class="check" /></td>`)
		} else {
			on, off, deny := "", "", ""
			switch group.Status {
			case "on":
				on = ` checked="checked"`
			case "deny":
				deny = ` checked="checked"`
			default:
				off = ` checked="checked"`
			}
			c.O(`
					<td align="center"><input type="radio" name="`, perm, `[`, group.ID, `]" value="on"`, on, ` class="check" /></td>
					<td align="center"><input type="radio" name="`, perm, `[`, group.ID, `]" value="off"`, off, ` class="check" /></td>
					<td align="center"><input type="radio" name="`, perm, `[`, group.ID, `]" value="deny"`, deny, ` class="check" /></td>`)
		}
		postStyle := ""
		if group.IsPostgroup {
			postStyle = ` style="font-style: italic;"`
		}
		c.O(`
					<td`, postStyle, `>`, group.Name, `</td>
				</tr>`)
	}
	c.O(`
			</table>
		</fieldset>

		<a href="javascript:void(0);" onclick="document.getElementById('`, perm, `_groups').style.display = 'block'; document.getElementById('`, perm, `_groups_link').style.display = 'none'; return false;" id="`, perm, `_groups_link" style="display: none;">[ `, c.Txt("avatar_select_permission"), ` ]</a>

		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			document.getElementById("`, perm, `_groups").style.display = "none";
			document.getElementById("`, perm, `_groups_link").style.display = "";
		// ]]></script>`)
}
