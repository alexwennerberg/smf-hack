package app

// Ports of Themes/default/ManageSmileys.template.php: template_settings,
// editsets, modifyset, editsmileys, modifysmiley, addsmiley, setorder,
// editicons, editicon.

import "strings"

// templateSmileySettings is template_settings().
func templateSmileySettings(c *Ctx) {
	page := c.Page.(*smileySettingsPage)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=smileys;sa=settings" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("settings"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="right">`, c.Txt("smiley_set_select_default"), `: </td>
				<td>
					<select name="default_smiley_set">`)
	for _, set := range page.Sets {
		c.O(`
						<option value="`, set.ID, `"`, selAttr(set.Selected), `>
							`, set.Name, `
						</option>`)
	}
	c.O(`
					</select>
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right" width="50%"><label for="smiley_sets_enable">`, c.Txt("smiley_sets_enable"), `</label>:</td>
				<td><input type="checkbox" name="smiley_sets_enable" id="smiley_sets_enable"`, checkedIf(!a.SettingEmpty("smiley_sets_enable")), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td align="right"><label for="smiley_enable">`, c.Txt("smileys_enable"), `</label>:<div class="smalltext" style="font-weight: bold;">`, c.Txt("smileys_enable_note"), `</div></td>
				<td><input type="checkbox" name="smiley_enable" id="smiley_enable"`, checkedIf(!a.SettingEmpty("smiley_enable")), ` class="check" /></td>

			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("smiley_sets_base_url"), `:</td>
				<td><input type="text" name="smiley_sets_url" value="`, a.Setting("smileys_url"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td align="right"`, redIf(!page.SmileysDirFound), `>`, c.Txt("smiley_sets_base_dir"), `:</td>
				<td><input type="text" name="smiley_sets_dir" value="`, page.SmileysDir, `" size="40" /></td>
			</tr>
			<tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr><tr class="windowbg2">
				<td align="right" width="50%"><label for="messageIcons_enable">`, c.Txt("icons_enable_customized"), `</label>:<div class="smalltext" style="font-weight: bold;">`, c.Txt("icons_enable_customized_note"), `</div></td>
				<td><input type="checkbox" name="messageIcons_enable" id="messageIcons_enable"`, checkedIf(!a.SettingEmpty("messageIcons_enable")), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" value="`, c.Txt("smiley_sets_save"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

func redIf(b bool) string {
	if b {
		return ` style="color: red"`
	}
	return ""
}

// templateEditsets is template_editsets().
func templateEditsets(c *Ctx) {
	page := c.Page.(*editSetsPage)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=smileys;sa=editsets" method="post" accept-charset="`, c.CharacterSet, `">
		<div class="tborder" style="padding: 1px;"><table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
			<tr class="titlebg">
				<td width="20">`, c.Txt("smiley_sets_default"), `</td>
				<td width="15%">`, c.Txt("smiley_sets_name"), `</td>
				<td>`, c.Txt("smiley_sets_url"), `</td>
				<td>`, c.Txt("smiley_set_modify"), `</td>
				<td width="20"></td>
			</tr>`)
	for _, set := range page.Sets {
		star := ""
		if set.Selected {
			star = "<b>*</b>"
		}
		del := ""
		if set.ID != 0 {
			del = `<input type="checkbox" name="smiley_set[` + itoa(set.ID) + `]" value="1" class="check" />`
		}
		c.O(`
			<tr class="windowbg2">
				<td align="center">`, star, `</td>
				<td class="windowbg">`, set.Name, `</td>
				<td class="windowbg">`, page.SmileysURL, `/<b>`, set.Path, `</b>/...</td>
				<td><a href="`, scripturl, `?action=smileys;sa=modifyset;set=`, set.ID, `">`, c.Txt("smiley_set_modify"), `</a></td>
				<td>`, del, `</td>
			</tr>`)
	}
	c.O(`
			<tr class="catbg3">
			<td colspan="5" align="right">
					<input type="submit" name="delete" value="`, c.Txt("smiley_sets_delete"), `" onclick="return confirm('`, c.Txt("smiley_sets_confirm"), `');" />
					<input type="submit" name="add" value="`, c.Txt("smiley_sets_add"), `" />
				</td>
			</tr>
		</table></div>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form><br />
	<table width="100%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
		<tr class="titlebg">
			<td>`, c.Txt("smiley_sets_latest"), `</td>
		</tr>
		<tr class="windowbg2">
			<td id="smileysLatest">`, c.Txt("smiley_sets_latest_fetch"), `</td>
		</tr>
	</table>`)
}

// templateModifyset is template_modifyset().
func templateModifyset(c *Ctx) {
	page := c.Page.(*modifySetPage)
	scripturl := c.App.ScriptURL
	cur := page.Current

	title := c.Txt("smiley_set_modify_existing")
	if cur.IsNew {
		title = c.Txt("smiley_set_new")
	}
	c.O(`
	<form action="`, scripturl, `?action=smileys;sa=editsets" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, title, `</td>
			</tr>`)
	if cur.CanImport > 0 {
		var msg, tail string
		if cur.CanImport == 1 {
			msg = c.Txt("smiley_set_import_single")
			tail = c.Txt("smiley_set_to_import_single")
		} else {
			msg = phpSprintf(c.Txt("smiley_set_import_multiple"), cur.CanImport)
			tail = c.Txt("smiley_set_to_import_multiple")
		}
		c.O(`
			<tr class="windowbg">
				<td align="left" colspan="2">
					<span class="smalltext">`, msg, ` <a href="`, scripturl, `?action=smileys;sa=import;set=`, cur.ID, `;sesc=`, c.Sc, `">`, c.Txt("662"), `</a> `, tail, `</span>
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_sets_name">`, c.Txt("smiley_sets_name"), `</label>: </b></td>
				<td><input type="text" name="smiley_sets_name" value="`, cur.Name, `" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_sets_path">`, c.Txt("smiley_sets_url"), `</label>: </b></td>
				<td>
					`, page.SmileysURL, `/`)
	if cur.FixedDefault {
		c.O(`<b>default</b><input type="hidden" name="smiley_sets_path" value="default" />`)
	} else if len(page.SetDirs) == 0 {
		c.O(`
					<input type="text" name="smiley_sets_path" value="`, cur.Path, `" /> `)
	} else {
		c.O(`
					<select name="smiley_sets_path">`)
		for _, d := range page.SetDirs {
			dis := ""
			if !d.Selectable {
				dis = ` disabled="disabled"`
			}
			c.O(`
						<option value="`, d.ID, `"`, selAttr(d.Current), dis, `>`, d.ID, `</option>`)
		}
		c.O(`
					</select> `)
	}
	c.O(`/..
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_sets_default">`, c.Txt("smiley_set_select_default"), `</label>: </b></td>
				<td><input type="checkbox" name="smiley_sets_default" value="1"`, checkedIf(cur.Selected), ` class="check" /></td>
			</tr>`)
	if cur.IsNew && page.SmileyEnable {
		c.O(`
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_sets_import">`, c.Txt("smiley_set_import_directory"), `</label>: </b></td>
				<td><input type="checkbox" name="smiley_sets_import" value="1" class="check" /></td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" value="`, c.Txt("smiley_sets_save"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
		<input type="hidden" name="set" value="`, cur.ID, `" />
	</form>`)
}

// templateEditsmileys is template_editsmileys().
func templateEditsmileys(c *Ctx) {
	page := c.Page.(*editSmileysPage)
	scripturl := c.App.ScriptURL

	sortLink := func(col, label string) string {
		if page.Sort == col {
			return "<b>" + label + "</b>"
		}
		return `<a href="` + scripturl + `?action=smileys;sa=editsmileys;sort=` + col + `">` + label + `</a>`
	}

	c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function makeChanges(action)
		{
			if (action == '-1')
				return false;
			else if (action == 'delete')
			{
				if (confirm('`, c.Txt("smileys_confirm"), `'))
					document.forms.smileyForm.submit();
			}
			else
				document.forms.smileyForm.submit();
		}
	// ]]></script>
	<form action="`, scripturl, `?action=smileys;sa=editsmileys" method="post" accept-charset="`, c.CharacterSet, `" name="smileyForm" id="smileyForm">
		<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%" class="tborder">
			<tr>
				<td colspan="7" align="right" class="titlebg">
					<select name="set" onchange="changeSet(this.options[this.selectedIndex].value);">`)
	for _, set := range page.Sets {
		c.O(`
						<option value="`, set.Path, `"`, selAttr(page.SelectedSet == set.Path), `>`, set.Name, `</option>`)
	}
	c.O(`
					</select>
				</td>
			</tr><tr class="catbg3">
				<td></td>
				<td>
					`, sortLink("code", c.Txt("smileys_code")), `
				</td><td>
					`, sortLink("filename", c.Txt("smileys_filename")), `
				</td><td>
					`, sortLink("hidden", c.Txt("smileys_location")), `
				</td><td>
					`, sortLink("description", c.Txt("smileys_description")), `
				</td><td>
					`, c.Txt("smileys_modify"), `
				</td>
				<td width="4%"></td>
			</tr>`)
	for _, sm := range page.Smileys {
		notFound := ""
		if len(sm.SetsNotFound) > 0 {
			notFound = `<br />
					<span class="smalltext"><b>` + c.Txt("smileys_not_found_in_set") + `:</b> ` + strings.Join(sm.SetsNotFound, ", ") + `</span>`
		}
		c.O(`
			<tr class="windowbg2">
				<td valign="top">
					<a href="`, scripturl, `?action=smileys;sa=modifysmiley;smiley=`, sm.ID, `"><img src="`, page.SmileysURL, `/`, page.SelectedSet, `/`, sm.Filename, `" alt="`, sm.Description, `" style="padding: 2px;" id="smiley`, sm.ID, `" /><input type="hidden" name="smileys[`, sm.ID, `][filename]" value="`, sm.Filename, `" /></a>
				</td><td valign="top" style="font-family: monospace;">
					`, sm.Code, `
				</td><td valign="top" class="windowbg">
					`, sm.Filename, `
				</td><td valign="top">
					`, sm.Location, `
				</td><td valign="top" class="windowbg">
					`, sm.Description, notFound, `
				</td><td valign="top">
					<a href="`, scripturl, `?action=smileys;sa=modifysmiley;smiley=`, sm.ID, `">`, c.Txt("smileys_modify"), `</a>
				</td><td valign="top" align="center" width="4%">
					<input type="checkbox" name="checked_smileys[]" value="`, sm.ID, `" class="check" />
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg">
				<td colspan="7" align="right">
					<select name="smiley_action" onchange="makeChanges(this.value);">
						<option value="-1">`, c.Txt("smileys_with_selected"), `:</option>
						<option value="-1">--------------</option>
						<option value="hidden">`, c.Txt("smileys_make_hidden"), `</option>
						<option value="post">`, c.Txt("smileys_show_on_post"), `</option>
						<option value="popup">`, c.Txt("smileys_show_on_popup"), `</option>
						<option value="delete">`, c.Txt("smileys_remove"), `</option>
					</select>
					<noscript><input type="submit" name="perform_action" value="`, c.Txt("161"), `" /></noscript>
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function changeSet(newSet)
		{
			var currentImage, i, knownSmileys = [`)
	var ids []string
	for _, sm := range page.Smileys {
		ids = append(ids, itoa(sm.ID))
	}
	c.O(strings.Join(ids, ", "), `];

			for (i = 0; i < knownSmileys.length; i++)
			{
				currentImage = document.getElementById("smiley" + knownSmileys[i]);
				currentImage.src = "`, page.SmileysURL, `/" + newSet + "/" + document.forms.smileyForm["smileys[" + knownSmileys[i] + "][filename]"].value;
			}
		}
	// ]]></script>`)
}

// templateModifysmiley is template_modifysmiley().
func templateModifysmiley(c *Ctx) {
	page := c.Page.(*modifySmileyPage)
	scripturl := c.App.ScriptURL
	cur := page.Current

	c.O(`
	<form action="`, scripturl, `?action=smileys;sa=editsmileys" method="post" accept-charset="`, c.CharacterSet, `" name="smileyForm" id="smileyForm">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("smiley_modify_existing"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b>`, c.Txt("smiley_preview"), `: </b></td>
				<td><img src="`, page.SmileysURL, `/`, page.SelectedSet, `/`, cur.Filename, `" id="preview" alt="" /> (`, c.Txt("smiley_preview_using"), `: <select name="set" onchange="updatePreview();">`)
	for _, set := range page.Sets {
		c.O(`
						<option value="`, set.Path, `"`, selAttr(page.SelectedSet == set.Path), `>`, set.Name, `</option>`)
	}
	c.O(`
					</select>)
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_code">`, c.Txt("smileys_code"), `</label>: </b></td>
				<td><input type="text" name="smiley_code" value="`, cur.Code, `" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_filename">`, c.Txt("smileys_filename"), `</label>: </b></td>
				<td>`)
	if len(page.Filenames) == 0 {
		c.O(`
					<input type="text" name="smiley_filename" value="`, cur.Filename, `" />`)
	} else {
		c.O(`
					<select name="smiley_filename" onchange="updatePreview();">`)
		for _, f := range page.Filenames {
			c.O(`
						<option value="`, f.ID, `"`, selAttr(f.Selected), `>`, f.ID, `</option>`)
		}
		c.O(`
					</select>`)
	}
	c.O(`
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_description">`, c.Txt("smileys_description"), `</label>: </b></td>
				<td><input type="text" name="smiley_description" value="`, cur.Description, `" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="smiley_location">`, c.Txt("smileys_location"), `</label>: </b></td>
				<td>
					<select name="smiley_location">
						<option value="0"`, selAttr(cur.Location == 0), `>
							`, c.Txt("smileys_location_form"), `
						</option>
						<option value="1"`, selAttr(cur.Location == 1), `>
							`, c.Txt("smileys_location_hidden"), `
						</option>
						<option value="2"`, selAttr(cur.Location == 2), `>
							`, c.Txt("smileys_location_popup"), `
						</option>
					</select>
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" value="`, c.Txt("smileys_save"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
		<input type="hidden" name="smiley" value="`, cur.ID, `" />
	</form>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function updatePreview()
		{
			var currentImage = document.getElementById("preview");
			currentImage.src = "`, page.SmileysURL, `/" + document.forms.smileyForm.set.value + "/" + document.forms.smileyForm.smiley_filename.value;
		}
	// ]]></script>`)
}

// templateAddsmiley is template_addsmiley().
func templateAddsmiley(c *Ctx) {
	page := c.Page.(*addSmileyPage)
	scripturl := c.App.ScriptURL

	firstFile := ""
	if len(page.Filenames) > 0 {
		firstFile = page.Filenames[0].ID
	}

	c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function swapUploads()
		{
			document.getElementById("uploadMore").style.display = document.getElementById("uploadSmiley").disabled ? "none" : "";
			document.getElementById("uploadSmiley").disabled = !document.getElementById("uploadSmiley").disabled;
		}

		function selectMethod(element)
		{
			document.getElementById("method-existing").checked = element != "upload";
			document.getElementById("method-upload").checked = element == "upload";
		}
	// ]]></script>

	<form action="`, scripturl, `?action=smileys;sa=addsmiley" method="post" accept-charset="`, c.CharacterSet, `" name="smileyForm" id="smileyForm" enctype="multipart/form-data">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("smileys_add_method"), `:</td>
			</tr>
			<tr class="windowbg">
				<td width="40%"><label for="method-existing"><input type="radio" name="method" id="method-existing" value="existing" checked="checked" class="check" /> `, c.Txt("smileys_add_existing"), `</label></td>
				<td>
					<img src="`, page.SmileysURL, `/`, page.SelectedSet, `/`, firstFile, `" id="preview" alt="" />  (`, c.Txt("smiley_preview_using"), `: <select name="set" onchange="updatePreview();selectMethod('existing');">`)
	for _, set := range page.Sets {
		c.O(`
							<option value="`, set.Path, `"`, selAttr(page.SelectedSet == set.Path), `>`, set.Name, `</option>`)
	}
	c.O(`
						</select>)
				</td>
			</tr>
			<tr class="windowbg" valign="bottom">
				<td style="padding-bottom: 2ex;" align="right" width="40%">
					<b><label for="smiley_filename">`, c.Txt("smileys_filename"), `</label>: </b>
				</td>
				<td style="padding-bottom: 2ex;" width="60%">`)
	if len(page.Filenames) == 0 {
		c.O(`
					<input type="text" name="smiley_filename" value="`, page.Current.Filename, `" onchange="selectMethod('existing');" />`)
	} else {
		c.O(`
						<select name="smiley_filename" onchange="updatePreview();selectMethod('existing');">`)
		for _, f := range page.Filenames {
			c.O(`
							<option value="`, f.ID, `"`, selAttr(f.Selected), `>`, f.ID, `</option>`)
		}
		c.O(`
						</select>
						`)
	}
	c.O(`
				</td>
			</tr>
			<tr class="windowbg2">
				<td colspan="2"><label for="method-upload"><input type="radio" name="method" id="method-upload" value="upload" class="check" /> `, c.Txt("smileys_add_upload"), `</label></td>
			</tr>
			<tr class="windowbg2">
				<td style="padding-bottom: 2ex;" width="40%" align="right"><b>`, c.Txt("smileys_add_upload_choose"), `:</b><div class="smalltext">`, c.Txt("smileys_add_upload_choose_desc"), `</div></td>
				<td style="padding-bottom: 2ex;" width="60%"><input type="file" name="uploadSmiley" id="uploadSmiley" onchange="selectMethod('upload');" /></td>
			</tr>
			<tr class="windowbg2">
				<td width="40%" align="right"><b><label for="sameall">`, c.Txt("smileys_add_upload_all"), `:</label></b></td>
				<td width="60%"><input type="checkbox" name="sameall" id="sameall" checked="checked" class="check" onclick="swapUploads(); selectMethod('upload');" /></td>
			</tr>
			<tr class="windowbg2" id="uploadMore" style="display: none;">
				<td colspan="2">
					<table width="100%" cellpadding="4" cellspacing="0" border="0" class="windowbg2">`)
	for _, set := range page.Sets {
		c.O(`
						<tr class="windowbg2">
							<td width="40%" align="right">`, c.Txt("smileys_add_upload_for1"), ` <b>`, set.Name, `</b> `, c.Txt("smileys_add_upload_for2"), `:</td>
							<td width="60%"><input type="file" name="individual_`, set.Name, `" onchange="selectMethod('upload');" /></td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
			</tr>
		</table>
		<br />
		<table width="80%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("smiley_new"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="right" width="40%"><b><label for="smiley_code">`, c.Txt("smileys_code"), `</label>: </b></td>
				<td width="60%"><input type="text" name="smiley_code" value="" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right" width="40%"><b><label for="smiley_description">`, c.Txt("smileys_description"), `</label>: </b></td>
				<td width="60%"><input type="text" name="smiley_description" value="" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right" width="40%"><b><label for="smiley_location">`, c.Txt("smileys_location"), `</label>: </b></td>
				<td width="60%">
					<select name="smiley_location">
						<option value="0" selected="selected">
							`, c.Txt("smileys_location_form"), `
						</option>
						<option value="1">
							`, c.Txt("smileys_location_hidden"), `
						</option>
						<option value="2">
							`, c.Txt("smileys_location_popup"), `
						</option>
					</select>
				</td>
			</tr>
			<tr class="windowbg">
				<td align="right" colspan="2"><input type="submit" value="`, c.Txt("smileys_save"), `" /></td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function updatePreview()
		{
			var currentImage = document.getElementById("preview");
			currentImage.src = "`, page.SmileysURL, `/" + document.forms.smileyForm.set.value + "/" + document.forms.smileyForm.smiley_filename.value;
		}
	// ]]></script>`)
}

// templateSetorder is template_setorder().
func templateSetorder(c *Ctx) {
	page := c.Page.(*setOrderPage)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	smileysURL := page.SmileysURL
	def := page.SmileySetDefault

	for _, loc := range page.Locations {
		prompt := c.Txt("smileys_move_select_smiley")
		if page.MoveSmiley != 0 {
			prompt = c.Txt("smileys_move_select_destination")
		}
		c.O(`
	<br />
	<form action="`, scripturl, `?action=smileys;sa=editsmileys" method="post" accept-charset="`, c.CharacterSet, `">
	<table border="0" cellspacing="1" cellpadding="4" align="center" width="80%" class="tborder" style="padding: 1px;">
			<tr class="titlebg">
				<td>`, loc.Title, `</td>
			</tr>
			<tr class="windowbg">
				<td class="smalltext">`, loc.Description, `</td>
			</tr>
			<tr class="windowbg2">
				<td>
					<b>`, prompt, `...</b><br />`)
		for _, row := range loc.Rows {
			rowNum := 0
			if len(row) > 0 {
				rowNum = row[0].Row
			}
			if page.MoveSmiley != 0 {
				c.O(`
					<a href="`, scripturl, `?action=smileys;sa=setorder;location=`, loc.ID, `;source=`, page.MoveSmiley, `;row=`, rowNum, `;sesc=`, c.Sc, `"><img src="`, imagesURL, `/smiley_select_spot.gif" alt="`, c.Txt("smileys_move_here"), `" /></a>`)
			}
			for _, sm := range row {
				if page.MoveSmiley == 0 {
					c.O(`<a href="`, scripturl, `?action=smileys;sa=setorder;move=`, sm.ID, `"><img src="`, smileysURL, `/`, def, `/`, sm.Filename, `" style="padding: 2px; border: 0px solid black;" alt="`, sm.Description, `" /></a>`)
				} else {
					border := "0px solid black"
					if sm.Selected {
						border = "2px solid red"
					}
					c.O(`<img src="`, smileysURL, `/`, def, `/`, sm.Filename, `" style="padding: 2px; border: `, border, `;" alt="`, sm.Description, `" /><a href="`, scripturl, `?action=smileys;sa=setorder;location=`, loc.ID, `;source=`, page.MoveSmiley, `;after=`, sm.ID, `;sesc=`, c.Sc, `" title="`, c.Txt("smileys_move_here"), `"><img src="`, imagesURL, `/smiley_select_spot.gif" alt="`, c.Txt("smileys_move_here"), `" /></a>`)
				}
			}
			c.O(`
					<br />`)
		}
		if page.MoveSmiley != 0 {
			c.O(`
					<a href="`, scripturl, `?action=smileys;sa=setorder;location=`, loc.ID, `;source=`, page.MoveSmiley, `;row=`, loc.LastRow, `;sesc=`, c.Sc, `"><img src="`, imagesURL, `/smiley_select_spot.gif" alt="`, c.Txt("smileys_move_here"), `" /></a>`)
		}
		c.O(`
				</td>
			</tr>
		</table>
	</form>`)
	}
}

// templateEditicons is template_editicons().
func templateEditicons(c *Ctx) {
	page := c.Page.(*editIconsPage)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=smileys;sa=editicons" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%" class="tborder">
			<tr class="titlebg">
				<td></td>
				<td>`, c.Txt("smileys_filename"), `</td>
				<td>`, c.Txt("smileys_description"), `</td>
				<td>`, c.Txt("icons_board"), `</td>
				<td>`, c.Txt("smileys_modify"), `</td>
				<td width="4%"></td>
			</tr>`)
	for _, icon := range page.Icons {
		c.O(`
			<tr class="windowbg2">
				<td valign="top">
					<img src="`, icon.ImageURL, `" alt="`, icon.Title, `" style="padding: 2px;" />
				</td><td valign="top">
					`, icon.Filename, `.gif
				</td><td valign="top" class="windowbg">
					`, icon.Title, `
				</td><td valign="top">
					`, icon.Board, `
				</td><td valign="top">
					<a href="`, scripturl, `?action=smileys;sa=editicon;icon=`, icon.ID, `">`, c.Txt("smileys_modify"), `</a>
				</td><td valign="top" align="center" width="4%">
					<input type="checkbox" name="checked_icons[]" value="`, icon.ID, `" class="check" />
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg">
				<td colspan="6" align="right">
					<br />
					<input type="submit" name="add" value="`, c.Txt("icons_add_new"), `" />
					<input type="submit" name="delete" value="`, c.Txt("smileys_remove"), `" onclick="return confirm('`, c.Txt("icons_confirm"), `');" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateEditicon is template_editicon().
func templateEditicon(c *Ctx) {
	page := c.Page.(*editIconPage)
	scripturl := c.App.ScriptURL

	iconID := 0
	if !page.NewIcon {
		iconID = page.Icon.ID
	}
	title := c.Txt("icons_edit_icon")
	if page.NewIcon {
		title = c.Txt("icons_new_icon")
	}
	c.O(`
	<form action="`, scripturl, `?action=smileys;sa=editicon;icon=`, iconID, `" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, title, `</td>
			</tr>`)
	if !page.NewIcon {
		c.O(`
			<tr class="windowbg2">
				<td align="right"><b>`, c.Txt("smiley_preview"), `: </b></td>
				<td><img src="`, page.Icon.ImageURL, `" alt="`, page.Icon.Title, `" /></td>
			</tr>`)
	}
	filenameVal := ""
	if page.Icon.Filename != "" {
		filenameVal = page.Icon.Filename + ".gif"
	}
	c.O(`
			<tr class="windowbg2">
				<td align="right"><b><label for="icon_filename">`, c.Txt("smileys_filename"), `</label>: </b><br /><span class="smalltext">`, c.Txt("icons_filename_all_gif"), `</span></td>
				<td><input type="text" name="icon_filename" value="`, filenameVal, `" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="icon_description">`, c.Txt("smileys_description"), `</label>: </b></td>
				<td><input type="text" name="icon_description" value="`, page.Icon.Title, `" /></td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="icon_board">`, c.Txt("icons_board"), `</label>: </b></td>
				<td>
					<select name="icon_board">
						<option value="0"`, selAttr(page.Icon.BoardID == 0), `>`, c.Txt("icons_edit_icons_all_boards"), `</option>`)
	for _, b := range page.Boards {
		c.O(`
						<option value="`, b.ID, `"`, selAttr(page.Icon.BoardID != 0 && page.Icon.BoardID == b.ID), `>`, b.Name, `</option>`)
	}
	c.O(`
					</select>
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><b><label for="icon_location">`, c.Txt("smileys_location"), `</label>: </b></td>
				<td>
					<select name="icon_location">
						<option value="0"`, selAttr(page.Icon.After == 0), `>`, c.Txt("icons_location_first_icon"), `</option>`)
	for _, data := range page.Icons {
		if page.Icon.ID == 0 || data.ID != page.Icon.ID {
			c.O(`
						<option value="`, data.ID, `"`, selAttr(page.Icon.After != 0 && data.ID == page.Icon.After), `>`, c.Txt("icons_location_after"), `: `, data.Title, `</option>`)
		}
	}
	c.O(`
					</select>
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" value="`, c.Txt("smileys_save"), `" />
				</td>
			</tr>
		</table>`)
	if !page.NewIcon {
		c.O(`
		<input type="hidden" name="icon" value="`, page.Icon.ID, `" />`)
	}
	c.O(`
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
