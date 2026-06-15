package app

// Ports of Themes/default/ManageSearch.template.php:
// template_modify_settings / modify_weights / select_search_method.

// templateSearchSettings is template_modify_settings().
func templateSearchSettings(c *Ctx) {
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=managesearch;sa=settings" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("settings"), `</td>
			</tr>`)
	if c.CanChangePermissions {
		c.O(`
			<tr class="windowbg2">
				<th width="50%" align="right" valign="top"><label for="search_posts_groups">`, c.Txt("groups_search_posts"), `:</label></th>
				<td>`)
		c.themeInlinePermissions("search_posts")
		c.O(`
				</td>
			</tr>`)
	}
	c.O(`
			<tr class="windowbg2">
				<th width="50%" align="right"><label for="simpleSearch_check">`, c.Txt("simpleSearch"), `</label> (<a href="`, scripturl, `?action=helpadmin;help=simpleSearch" onclick="return reqWin(this.href);">?</a>):</th>
				<td><input type="checkbox" name="simpleSearch" id="simpleSearch_check"`, c.checkboxAttr("simpleSearch"), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<th align="right"><label for="search_results_per_page_input">`, c.Txt("search_results_per_page"), `:</label></th>
				<td><input type="text" name="search_results_per_page" id="search_results_per_page_input" value="`, a.Setting("search_results_per_page"), `" size="10" /></td>
			</tr><tr class="windowbg2">
				<th align="right">
					<label for="search_max_results_input">`, c.Txt("search_max_results"), `:</label>
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("search_max_results_disable"), `</div>
				</th>
				<td valign="top"><input type="text" name="search_max_results" id="search_max_results_input" value="`, c.settingOrZero("search_max_results"), `" size="10" /></td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save" value="`, c.Txt("search_settings_save"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateSearchWeights is template_modify_weights().
func templateSearchWeights(c *Ctx) {
	page := c.Page.(*SearchWeightsCtx)
	scripturl := c.App.ScriptURL

	row := func(n int, factor string) {
		c.O(`
			<tr class="windowbg2">
				<td align="right">`, c.Txt(factor), ` (<a href="`, scripturl, `?action=helpadmin;help=`, factor, `" onclick="return reqWin(this.href);">?</a>):</td>
				<td><input type="text" name="`, factor, `" id="weight`, n, `_val" value="`, c.settingOrZero(factor), `" onchange="calculateNewValues()" size="3" /></td>
				<td id="weight`, n, `">`, page.Percent[factor], `%</td>
			</tr>`)
	}

	c.O(`
	<form action="`, scripturl, `?action=managesearch;sa=weights" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="3">`, c.Txt("search_weights"), `</td>
			</tr>`)
	row(1, "search_weight_frequency")
	row(2, "search_weight_age")
	row(3, "search_weight_length")
	row(4, "search_weight_subject")
	row(5, "search_weight_first_message")
	row(6, "search_weight_sticky")
	c.O(`
			<tr class="windowbg2">
				<td align="right"><b>`, c.Txt("search_weights_total"), `</b></td>
				<td id="weighttotal" style="font-weight: bold;">`, page.Total, `</td>
				<td style="font-weight: bold;">100%</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="3">
					<input type="submit" name="save" value="`, c.Txt("search_weights_save"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function calculateNewValues()
		{
			var total = 0;
			for (var i = 1; i <= 6; i++)
			{
				total += parseInt(document.getElementById('weight' + i + '_val').value);
			}
			setInnerHTML(document.getElementById('weighttotal'), total);
			for (var i = 1; i <= 6; i++)
			{
				setInnerHTML(document.getElementById('weight' + i), (Math.round(1000 * parseInt(document.getElementById('weight' + i + '_val').value) / total) / 10) + '%');
			}
		}
	// ]]></script>`)
}

// templateSelectSearchMethod is template_select_search_method().
func templateSelectSearchMethod(c *Ctx) {
	page := c.Page.(*SearchMethodCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	indexNoneChecked := ""
	if a.SettingEmpty("search_index") {
		indexNoneChecked = ` checked="checked"`
	}
	fulltextChecked := ""
	if a.Setting("search_index") == "fulltext" {
		fulltextChecked = ` checked="checked"`
	}
	customChecked := ""
	if a.Setting("search_index") == "custom" {
		customChecked = ` checked="checked"`
	}

	c.O(`
	<form action="`, scripturl, `?action=managesearch;sa=method" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="3">`, c.Txt("search_method"), `</td>
			</tr>`)
	doubleIndex := ""
	if page.DoubleIndex {
		doubleIndex = c.Txt("search_double_index") + `<br />`
	}
	c.O(`
			<tr class="windowbg2">
				<td colspan="3">
					<b>`, c.Txt("search_method_messages_table_space"), `:</b> `, page.DataLength, ` `, c.Txt("search_method_kilobytes"), ` <br />
					<b>`, c.Txt("search_method_messages_index_space"), `:</b> `, page.IndexLength, ` `, c.Txt("search_method_kilobytes"), `<br />`, doubleIndex, `
					<br />
				</td>
			</tr>`)
	c.O(`
			<tr class="windowbg2">
				<th width="47%" align="right">`, c.Txt("search_index"), `:<div class="smalltext" style="font-weight: normal;"><a href="`, scripturl, `?action=helpadmin;help=search_why_use_index" onclick="return reqWin(this.href);">`, c.Txt("search_create_index_why"), `</a></div></th>
				<td width="3%" align="center" valign="top" class="windowbg"><input type="radio" name="search_index" value=""`, indexNoneChecked, ` /></td>
				<td>
					`, c.Txt("search_index_none"), `
				</td>
			</tr><tr class="windowbg2">
				<td></td>
				<td width="3%" align="center" valign="top" class="windowbg"><input type="radio" name="search_index" value="fulltext"`, fulltextChecked, ` onclick="alert('`, c.Txt("search_method_fulltext_warning"), `'); selectRadioByName(this.form.search_index, 'fulltext');" /></td>
				<td>
					`, c.Txt("search_method_fulltext_index"), `<br />
					<span class="smalltext">
						<b>`, c.Txt("search_index_label"), `:</b> `, c.Txt("search_method_fulltext_cannot_create"), `
					</span>
				</td>
			</tr><tr class="windowbg2">
				<td align="right"></td>
				<td width="3%" align="center" valign="top" class="windowbg"><input type="radio" name="search_index" value="custom"`, customChecked, ` onclick="alert('`, c.Txt("search_index_custom_warning"), `'); selectRadioByName(this.form.search_method, '1');" /></td>
				<td>
					`, c.Txt("search_index_custom"), `<br />
					<span class="smalltext">
						<b>`, c.Txt("search_index_label"), `:</b> `, c.Txt("search_method_no_index_exists"), ` [<a href="`, scripturl, `?action=managesearch;sa=createmsgindex">`, c.Txt("search_index_create_custom"), `</a>]
					</span>
				</td>
			</tr><tr class="windowbg2">
				<th align="right"><label for="search_force_index_check">`, c.Txt("search_force_index"), `:</label></th>
				<td colspan="2"><input type="checkbox" name="search_force_index" id="search_force_index_check" value="1"`, c.checkboxAttr("search_force_index"), ` /></td>
			</tr><tr class="windowbg2">
				<th align="right"><label for="search_match_words_check">`, c.Txt("search_match_words"), `:</label></th>
				<td colspan="2"><input type="checkbox" name="search_match_words" id="search_match_words_check" value="1"`, c.checkboxAttr("search_match_words"), ` /></td>
			</tr><tr class="windowbg2">
				<td></td>
				<td align="right" colspan="2">
					<input type="submit" name="save" value="`, c.Txt("search_method_save"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
