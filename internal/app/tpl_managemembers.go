package app

// Ports of Themes/default/ManageMembers.template.php:
// template_view_members / template_search_members / template_admin_browse.

// templateViewMembers is template_view_members().
func templateViewMembers(c *Ctx) {
	page := c.Page.(*ViewMembersCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
	<form action="`, scripturl, `?action=viewmembers`, page.ParamsURL, `" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor" align="center">
			<tr class="catbg">
				<td align="left" colspan="8">
					<b>`, c.Txt("139"), `:</b> `, page.PageIndex, `
				</td>
			</tr>
			<tr class="titlebg">`)
	for _, col := range page.Columns {
		c.O(`
				<td valign="top">
					<a href="`, col.Href, `">`)
		if col.Selected {
			c.O(col.Label, ` <img src="`, imagesURL, `/sort_`, page.SortDirection, `.gif" alt="" />`)
		} else {
			c.O(col.Label)
		}
		c.O(`</a>
				</td>`)
	}
	if page.CanDelete {
		c.O(`
				<td align="center">
					<input type="checkbox" class="check" onclick="invertAll(this, this.form);" />
				</td>`)
	} else {
		c.O(`
				<td></td>`)
	}
	c.O(`
			</tr>`)

	if len(page.Members) == 0 {
		c.O(`
			<tr>
				<td class="windowbg" colspan="8">(`, c.Txt("search_no_results"), `)</td>
			</tr>`)
	} else {
		for _, m := range page.Members {
			href := scripturl + "?action=profile;u=" + itoa(m.ID)
			c.O(`
			<tr>
				<td class="windowbg" width="5%">
					`, m.ID, `
				</td>
				<td class="windowbg2">
					<a href="`, href, `">`, m.Username, `</a>
				</td>
				<td class="windowbg2">
					<a href="`, href, `">`, m.Name, `</a>
				</td>
				<td class="windowbg">
					<a href="mailto:`, m.Email, `">`, m.Email, `</a>
				</td>
				<td class="windowbg2">
					<a href="`, scripturl, `?action=trackip;searchip=`, m.IP, `">`, m.IP, `</a>
				</td>
				<td class="windowbg2">
					`, m.LastActive, `
				</td>
				<td class="windowbg2">
					`, m.Posts, `
				</td>`)
			if page.CanDelete {
				c.O(`
				<td align="center" class="windowbg" width="5%">
					<input type="checkbox" name="delete[]" value="`, m.ID, `" class="check" />
				</td>`)
			} else {
				c.O(`
				<td class="windowbg"></td>`)
			}
			c.O(`
			</tr>`)
		}
		deleteBtn := ""
		if page.CanDelete {
			deleteBtn = `
					<input type="submit" name="delete_members" value="` + c.Txt("608") + `" onclick="return confirm('` + c.Txt("confirm_delete_members") + `');" />`
		}
		descInput := ""
		if page.SortDirection == "up" {
			descInput = `
					<input type="hidden" name="desc" value="1" />`
		}
		c.O(`
			<tr>
				<td class="windowbg2" align="right" colspan="8">`, deleteBtn, `
					<input type="hidden" name="sc" value="`, c.Sc, `" />
					<input type="hidden" name="sort" value="`, page.SortBy, `" />
					<input type="hidden" name="start" value="`, page.Start, `" />`, descInput, `
				</td>
			</tr>`)
	}
	c.O(`
		</table>
	</form>`)
}

// templateSearchMembers is template_search_members().
func templateSearchMembers(c *Ctx) {
	page := c.Page.(*SearchMembersCtx)
	scripturl := c.App.ScriptURL

	rangeSelect := func(name string) string {
		return `
								<select name="types[` + name + `]">
									<option value="--">&lt;</option>
									<option value="-">&lt;=</option>
									<option value="=" selected="selected">=</option>
									<option value="+">&gt;=</option>
									<option value="++">&gt;</option>
								</select>
							`
	}

	c.O(`
	<form action="`, scripturl, `?action=viewmembers" method="post" accept-charset="`, c.CharacterSet, `">
		<input type="hidden" name="sa" value="query" /><div class="tborder">
		<table width="100%" cellpadding="4" cellspacing="0" class="windowbg">
			<tr class="titlebg">
				<td colspan="5">`, c.Txt("search_for"), `:</td>
			</tr>
						<tr>
							<td colspan="5" align="right"><span class="smalltext">(`, c.Txt("wild_cards_allowed"), `)</span></td>
						</tr><tr>
							<th align="right">`, c.Txt("member_id"), `:</th>
							<td align="center">`, rangeSelect("mem_id"), `</td>
							<td align="left"><input type="text" name="mem_id" value="" size="6" /></td>
							<th align="right">`, c.Txt("35"), `:</th>
							<td align="left"><input type="text" name="membername" value="" /> </td>
						</tr><tr>
							<th align="right">`, c.Txt("age"), `:</th>
							<td align="center">`, rangeSelect("age"), `</td>
							<td align="left"><input type="text" name="age" value="" size="6" /></td>
							<th align="right">`, c.Txt("email_address"), `:</th>
							<td align="left"><input type="text" name="email" value="" /></td>
						</tr><tr>
							<th align="right">`, c.Txt("26"), `:</th>
							<td align="center">`, rangeSelect("posts"), `</td>
							<td align="left"><input type="text" name="posts" value="" size="6" /></td>
							<th align="right">`, c.Txt("96"), `:</th>
							<td align="left"><input type="text" name="website" value="" /></td>
						</tr><tr>
							<th align="right">`, c.Txt("233"), `:</th>
							<td align="center">`, rangeSelect("reg_date"), `</td>
							<td align="left"><input type="text" name="reg_date" value="" /> <span class="smalltext">`, c.Txt("date_format"), `</span></td>
							<th align="right">`, c.Txt("227"), `:</th>
							<td align="left"><input type="text" name="location" value="" /></td>
						</tr><tr>
							<th align="right">`, c.Txt("viewmembers_online"), `:</th>
							<td align="center">`, rangeSelect("last_online"), `</td>
							<td align="left"><input type="text" name="last_online" value="" /> <span class="smalltext">`, c.Txt("date_format"), `</span></td>
							<th align="right">`, c.Txt("ip_address"), `:</th>
							<td align="left"><input type="text" name="ip" value="" /></td>
						</tr><tr>
							<th align="right">`, c.Txt("231"), `:</th>
							<td align="left" colspan="2">
								<label for="gender-0"><input type="checkbox" name="gender[]" value="0" id="gender-0" checked="checked" class="check" /> `, c.Txt("undefined_gender"), `</label>&nbsp;&nbsp;
								<label for="gender-1"><input type="checkbox" name="gender[]" value="1" id="gender-1" checked="checked" class="check" /> `, c.Txt("238"), `</label>&nbsp;&nbsp;
								<label for="gender-2"><input type="checkbox" name="gender[]" value="2" id="gender-2" checked="checked" class="check" /> `, c.Txt("239"), `</label>
							</td>
							<th align="right">`, c.Txt("messenger_address"), `:</th>
							<td align="left"><input type="text" name="messenger" value="" /></td>
						</tr><tr>
							<th align="right">`, c.Txt("activation_status"), `:</th>
							<td align="left" colspan="2">
								<label for="activated-0"><input type="checkbox" name="activated[]" value="1" id="activated-0" checked="checked" class="check" /> `, c.Txt("activated"), `</label>&nbsp;&nbsp;
								<label for="activated-1"><input type="checkbox" name="activated[]" value="0" id="activated-1" checked="checked" class="check" /> `, c.Txt("not_activated"), `</label>
							</td>
			</tr>
		</table></div>

		<table width="100%" cellpadding="0" cellspacing="0" class="tborder" style="margin-top: 2ex;">
			<tr class="catbg3">
				<td colspan="2" height="28"><b>`, c.Txt("member_part_of_these_membergroups"), `:</b></td>
			</tr>
			<tr>
				<td class="windowbg" width="50%" valign="top">
					<table width="100%" cellpadding="3" cellspacing="1" border="0" >
						<tr class="titlebg">
							<th>`, c.Txt("membergroups"), `</th>
							<th width="40">`, c.Txt("primary"), `</th>
							<th width="40">`, c.Txt("additional"), `</th>
						</tr>`)
	for _, mg := range page.Membergroups {
		additional := ""
		if mg.CanBeAdditional {
			additional = `<input type="checkbox" name="membergroups[2][]" value="` + itoa(mg.ID) + `" checked="checked" class="check" />`
		}
		c.O(`
						<tr class="windowbg2">
							<td>`, mg.Name, `</td>
							<td align="center">
								<input type="checkbox" name="membergroups[1][]" value="`, mg.ID, `" checked="checked" class="check" />
							</td>
							<td align="center">
								`, additional, `
							</td>
						</tr>`)
	}
	c.O(`
						<tr class="windowbg2">
							<td><em>`, c.Txt("737"), `</em></td>
							<td align="center"><input type="checkbox" onclick="invertAll(this, this.form, 'membergroups[1]');" checked="checked" class="check" /></td>
							<td align="center"><input type="checkbox" onclick="invertAll(this, this.form, 'membergroups[2]');" checked="checked" class="check" /></td>
						</tr>
					</table>
				</td>
				<td class="windowbg" valign="top">

					<table width="100%" cellpadding="3" cellspacing="1" border="0">
						<tr class="titlebg">
							<th colspan="2">`, c.Txt("membergroups_postgroups"), `</th>
						</tr>`)
	for _, pg := range page.Postgroups {
		c.O(`
						<tr class="windowbg2">
							<td>`, pg.Name, `</td>
							<td width="40" align="center">
								<input type="checkbox" name="postgroups[]" value="`, pg.ID, `" checked="checked" class="check" />
							</td>
						</tr>`)
	}
	c.O(`
						<tr class="windowbg2">
							<td><em>`, c.Txt("737"), `</em></td>
							<td align="center"><input type="checkbox" onclick="invertAll(this, this.form, 'postgroups[]');" checked="checked" class="check" /></td>
						</tr>
					</table>
				</td>
			</tr>
		</table>

		<div align="center" style="margin: 2ex;"><input type="submit" value="`, c.Txt("182"), `" /></div>
	</form>`)
}

// templateAdminBrowse is template_admin_browse().
func templateAdminBrowse(c *Ctx) {
	page := c.Page.(*AdminBrowseCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	rejectMsg := c.Txt("admin_browse_w_reject")
	approveMsg := c.Txt("admin_browse_w_approve")
	if page.BrowseType != "approve" {
		approveMsg = c.Txt("admin_browse_w_activate")
	}

	c.O(`
	<form action="`, scripturl, `?action=viewmembers" method="post" accept-charset="`, c.CharacterSet, `" name="postForm" id="postForm">
		<div class="tborder" style="padding: 1px;"><table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
			<tr class="catbg">
				<td colspan="6">
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						function onSelectChange()
						{
							if (document.forms.postForm.todo.value == "")
								return;

							var message = "";`)
	if page.CurrentFilter == 4 {
		c.O(`
							if (document.forms.postForm.todo.value.indexOf("reject") != -1)
								message = "`, c.Txt("admin_browse_w_delete"), `";
							else
								message = "`, rejectMsg, `";`)
	} else {
		c.O(`
							if (document.forms.postForm.todo.value.indexOf("delete") != -1)
								message = "`, c.Txt("admin_browse_w_delete"), `";
							else if (document.forms.postForm.todo.value.indexOf("reject") != -1)
								message = "`, rejectMsg, `";
							else if (document.forms.postForm.todo.value == "remind")
								message = "`, c.Txt("admin_browse_w_remind"), `";
							else
								message = "`, approveMsg, `";`)
	}
	c.O(`
							if (confirm(message + " `, c.Txt("admin_browse_warn"), `"))
								document.forms.postForm.submit();
						}`)
	if page.NumMembers > 20 {
		c.O(`
						function onOutstandingSubmit()
						{
							if (document.forms.postFormOutstanding.todo.value == "")
								return;

							var message = "";
							if (document.forms.postFormOutstanding.todo.value.indexOf("delete") != -1)
								message = "`, c.Txt("admin_browse_w_delete"), `";
							else if (document.forms.postFormOutstanding.todo.value.indexOf("reject") != -1)
								message = "`, rejectMsg, `";
							else if (document.forms.postFormOutstanding.todo.value == "remind")
								message = "`, c.Txt("admin_browse_w_remind"), `";
							else
								message = "`, approveMsg, `";

							if (confirm(message + " `, c.Txt("admin_browse_outstanding_warn"), `"))
								return true;
							else
								return false;
						}`)
	}
	c.O(`
					// ]]></script>

				`, c.Txt("139"), `: `, page.PageIndex, `
				</td>
			</tr>
			<tr class="titlebg">`)
	for _, col := range page.Columns {
		c.O(`
				<td valign="top">
					<a href="`, col.Href, `">`)
		if col.Selected {
			c.O(col.Label, ` <img src="`, imagesURL, `/sort_`, page.SortDirection, `.gif" alt="" />`)
		} else {
			c.O(col.Label)
		}
		c.O(`</a>
				</td>`)
	}
	c.O(`
				<td><input type="checkbox" class="check" onclick="invertAll(this, this.form, 'todo');" /></td>
			</tr>`)

	if len(page.Members) == 0 {
		noMembers := c.Txt("admin_browse_no_members_activate")
		if page.BrowseType == "approve" {
			noMembers = c.Txt("admin_browse_no_members_approval")
		}
		c.O(`
			<tr class="windowbg2">
				<td colspan="6" align="center">`, noMembers, `</td>
			</tr>`)
	} else {
		for _, m := range page.Members {
			c.O(`
			<tr>
				<td class="windowbg2" width="5%">`, m.ID, `</td>
				<td class="windowbg">
					<a href="`, scripturl, `?action=profile;u=`, m.ID, `">`, m.Username, `</a>
				</td>
				<td class="windowbg"><a href="mailto:`, m.Email, `">`, m.Email, `</a></td>
				<td class="windowbg2"><a href="`, scripturl, `?action=trackip;searchip=`, m.IP, `">`, m.IP, `</a></td>
				<td class="windowbg">`, m.DateRegistered, `</td>
				<td class="windowbg" width="5%">
					<input type="checkbox" value="`, m.ID, `" name="todoAction[]" class="check" />
				</td>
			</tr>`)
		}
		c.O(`
			<tr class="windowbg2">
				<td align="right" colspan="6">
					<table width="100%" cellpadding="0" cellspacing="0" border="0">
						<tr>
							<td align="left">`)
		if len(page.AvailableFilters) > 1 {
			c.O(`
								<b>`, c.Txt("admin_browse_filter_by"), `:</b>
								<select name="filter" onchange="this.form.submit();">`)
			for _, f := range page.AvailableFilters {
				sel := ""
				if f.Selected {
					sel = ` selected="selected"`
				}
				userWord := c.Txt("users")
				if f.Amount == 1 {
					userWord = c.Txt("user")
				}
				c.O(`
									<option value="`, f.Type, `"`, sel, `>`, f.Desc, ` - `, f.Amount, ` `, userWord, `</option>`)
			}
			c.O(`
								</select>
								<noscript><input type="submit" value="`, c.Txt("161"), `" name="filter" /></noscript>`)
		}
		if page.ShowFilter && len(page.AvailableFilters) > 0 {
			c.O(`
								<span class="smalltext"><b>`, c.Txt("admin_browse_filter_show"), `:</b> `, page.AvailableFilters[0].Desc, `</span>`)
		}
		descInput := ""
		if page.SortDirection == "up" {
			descInput = `
								<input type="hidden" name="desc" value="1" />`
		}
		c.O(`
							</td>
							<td align="right">
								<select name="todo" onchange="onSelectChange();">
									<option selected="selected" value="">`, c.Txt("admin_browse_with_selected"), `:</option>
									<option value="" disabled="disabled">-----------------------------</option>`)
		for _, act := range page.AllowedActions {
			c.O(`
									<option value="`, act[0], `">`, act[1], `</option>`)
		}
		c.O(`
								</select>
								<noscript><input type="submit" value="`, c.Txt("161"), `" /></noscript>
								<input type="hidden" name="type" value="`, page.BrowseType, `" />
								<input type="hidden" name="sort" value="`, page.SortBy, `" />
								<input type="hidden" name="start" value="`, page.Start, `" />
								<input type="hidden" name="orig_filter" value="`, page.CurrentFilter, `" />
								<input type="hidden" name="sa" value="approve" />`, descInput, `
							</td>
						</tr>
					</table>
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="catbg">
				<td colspan="6">`, c.Txt("139"), `: `, page.PageIndex, `</td>
			</tr>
		</table></div>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)

	if page.NumMembers > 20 {
		activateOpt := ""
		if page.BrowseType == "activate" {
			activateOpt = `
						<option value="ok">` + c.Txt("admin_browse_w_activate") + `</option>`
		}
		requireOpt := `
						<option value="require_activation">` + c.Txt("admin_browse_w_approve_require_activate") + `</option>`
		remindOpt := ""
		if page.BrowseType == "activate" {
			requireOpt = ""
			remindOpt = `
						<option value="remind">` + c.Txt("admin_browse_w_remind") + `</option>`
		}
		descInput := ""
		if page.SortDirection == "up" {
			descInput = `
					<input type="hidden" name="desc" value="1" />`
		}
		c.O(`
	<form action="`, scripturl, `?action=viewmembers" method="post" accept-charset="`, c.CharacterSet, `" name="postFormOutstanding" id="postFormOutstanding" onsubmit="return onOutstandingSubmit();">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="100%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("admin_browse_outstanding"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="left" width="50%">
					`, c.Txt("admin_browse_outstanding_days_1"), `:
				</td>
				<td align="left">
					<input type="text" name="time_passed" value="14" maxlength="4" size="3" /> `, c.Txt("admin_browse_outstanding_days_2"), `.
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="left" width="50%">
					`, c.Txt("admin_browse_outstanding_perform"), `:
				</td>
				<td align="left">
					<select name="todo">`, activateOpt, `
						<option value="okemail">`, approveMsg, ` `, c.Txt("admin_browse_w_email"), `</option>`, requireOpt, `
						<option value="reject">`, c.Txt("admin_browse_w_reject"), `</option>
						<option value="rejectemail">`, c.Txt("admin_browse_w_reject"), ` `, c.Txt("admin_browse_w_email"), `</option>
						<option value="delete">`, c.Txt("admin_browse_w_delete"), `</option>
						<option value="deleteemail">`, c.Txt("admin_browse_w_delete"), ` `, c.Txt("admin_browse_w_email"), `</option>`, remindOpt, `
					</select>
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="center" colspan="2">
					<input type="submit" value="`, c.Txt("admin_browse_outstanding_go"), `" />
					<input type="hidden" name="type" value="`, page.BrowseType, `" />
					<input type="hidden" name="sort" value="`, page.SortBy, `" />
					<input type="hidden" name="start" value="`, page.Start, `" />
					<input type="hidden" name="orig_filter" value="`, page.CurrentFilter, `" />
					<input type="hidden" name="sa" value="approve" />`, descInput, `
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
	}
}
