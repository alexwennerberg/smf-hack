package app

// Hand-port of Themes/default/Themes.template.php. Phase 4 ships only
// template_pick (the user-facing theme picker); the admin theme-management
// templates arrive with Phase 7.

// templateThemePick is template_pick().
func templateThemePick(c *Ctx) {
	page := c.Page.(*ThemePickCtx)
	scripturl := c.App.ScriptURL

	// Just go through each theme and show its information - thumbnail, etc.
	for _, theme := range page.AvailableThemes {
		bg := "windowbg2"
		if theme.Selected {
			bg = "windowbg"
		}
		userWord := c.Txt("theme_users")
		if theme.NumUsers == 1 {
			userWord = c.Txt("theme_user")
		}
		c.O(`
	<table align="center" width="85%" cellpadding="3" cellspacing="0" border="0" class="tborder">
		<tr class="`, bg, `">
			<td rowspan="2" width="126" height="120"><img src="`, theme.ThumbnailHref, `" alt="" /></td>
			<td valign="top" style="padding-top: 5px;">
				<div style="font-size: larger; padding-bottom: 6px;"><b><a href="`, scripturl, `?action=theme;sa=pick;u=`, page.CurrentMember, `;th=`, theme.ID, `;sesc=`, c.Sc, `">`, theme.Name, `</a></b></div>
				`, theme.Description, `
			</td>
		</tr>
		<tr class="`, bg, `">
			<td valign="bottom" align="right" style="padding: 6px; padding-top: 0;">
				<div style="float: left;" class="smalltext"><i>`, theme.NumUsers, ` `, userWord, `</i></div>
				<a href="`, scripturl, `?action=theme;sa=pick;u=`, page.CurrentMember, `;th=`, theme.ID, `;sesc=`, c.Sc, `">`, c.Txt("theme_set"), `</a> |
				<a href="`, scripturl, `?action=theme;sa=pick;u=`, page.CurrentMember, `;theme=`, theme.ID, `;sesc=`, c.Sc, `">`, c.Txt("theme_preview"), `</a>
			</td>
		</tr>
	</table>
	<br />`)
	}
}

// themeResetCtx backs template_reset_list.
type themeResetCtx struct {
	Themes []themeResetItem
}

// templateThemeResetList is template_reset_list().
func templateThemeResetList(c *Ctx) {
	page := c.Page.(*themeResetCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<table width="80%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("themeadmin_reset_title"), `</td>
			</tr>
			<tr class="windowbg">
				<td colspan="2" class="smalltext" style="padding: 2ex;">`, c.Txt("themeadmin_reset_tip"), `</td>
			</tr>`)

	// Show each theme.... with X for delete and a link to settings.
	for _, theme := range page.Themes {
		c.O(`
			<tr class="windowbg2">
				<td style="padding-bottom: 1ex;">
					<b><a href="`, scripturl, `?action=theme;th=`, theme.ID, `;sesc=`, c.Sc, `;sa=settings">`, theme.Name, `</a></b><br />
					<div style="padding-left: 5ex; line-height: 3ex;" class="smalltext">
						<a href="`, scripturl, `?action=theme;th=`, theme.ID, `;sesc=`, c.Sc, `;sa=reset">`, c.Txt("themeadmin_reset_defaults"), `</a> (`, theme.NumDefaultOptions, ` `, c.Txt("themeadmin_reset_defaults_current"), `)<br />
						<a href="`, scripturl, `?action=theme;th=`, theme.ID, `;sesc=`, c.Sc, `;sa=reset;who=1">`, c.Txt("themeadmin_reset_members"), `</a><br />
						<a href="`, scripturl, `?action=theme;th=`, theme.ID, `;sesc=`, c.Sc, `;sa=reset;who=2" onclick="return confirm('`, c.Txt("themeadmin_reset_remove_confirm"), `');">`, c.Txt("themeadmin_reset_remove"), `</a> (`, theme.NumMembers, ` `, c.Txt("themeadmin_reset_remove_current"), `)
					</div>
				</td>
			</tr>`)
	}

	c.O(`
		</table>`)
}

// themeOptForm backs template_set_options.
type themeOptForm struct {
	ThemeID int
	Name    string
	Reset   bool
	Fields  []*themeField
}

// templateThemeSetOptions is template_set_options().
func templateThemeSetOptions(c *Ctx) {
	page := c.Page.(*themeOptForm)
	scripturl := c.App.ScriptURL

	resetWho := "0"
	if page.Reset {
		resetWho = "1"
	}
	c.O(`
		<form action="`, scripturl, `?action=theme;th=`, page.ThemeID, `;sa=reset" method="post" accept-charset="`, c.CharacterSet, `">
			<input type="hidden" name="who" value="`, resetWho, `" />

			<table width="80%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td colspan="2">`, c.Txt("theme_options_title"), ` - `, page.Name, `</td>
				</tr>
				<tr class="windowbg">
					<td colspan="2" class="smalltext" style="padding: 2ex;">`, ternStr(page.Reset, c.Txt("themeadmin_reset_options_info"), c.Txt("theme_options_defaults")), `</td>
				</tr>`)

	for _, setting := range page.Fields {
		c.O(`
				<tr class="windowbg2">
					<td colspan="2">`)

		if page.Reset {
			defPrefix := ""
			if setting.Default {
				defPrefix = "default_"
			}
			c.O(`
						<select name="`, defPrefix, `options_master[`, setting.ID, `]" onchange="this.form.options_`, setting.ID, `.disabled = this.selectedIndex != 1;">
							<option value="0" selected="selected">`, c.Txt("themeadmin_reset_options_none"), `</option>
							<option value="1">`, c.Txt("themeadmin_reset_options_change"), `</option>
							<option value="2">`, c.Txt("themeadmin_reset_options_remove"), `</option>
						</select>`)
		}

		defPrefix := ""
		if setting.Default {
			defPrefix = "default_"
		}
		switch setting.Type {
		case "checkbox":
			c.O(`
						<input type="hidden" name="`+defPrefix+`options[`+setting.ID+`]" value="0" />
						<label for="options_`, setting.ID, `"><input type="checkbox" name="`, defPrefix, `options[`, setting.ID, `]" id="options_`, setting.ID, `"`, checkedIf(!empty(setting.Value)), disabledIf(page.Reset), ` value="1" class="check" /> `, setting.Label, `</label>`)
		case "list":
			c.O(`
						&nbsp;<label for="options_`, setting.ID, `">`, setting.Label, `</label>
						<select name="`, defPrefix, `options[`, setting.ID, `]" id="options_`, setting.ID, `"`, disabledIf(page.Reset), `>`)
			for _, opt := range setting.Options {
				c.O(`
							<option value="`, opt.Value, `"`, selAttr(opt.Value == setting.Value), `>`, opt.Label, `</option>`)
			}
			c.O(`
						</select>`)
		default:
			sizeAttr := ""
			if setting.Type == "number" {
				sizeAttr = ` size="5"`
			}
			c.O(`
						&nbsp;<label for="options_`, setting.ID, `">`, setting.Label, `</label>
						<input type="text" name="`, defPrefix, `options[`, setting.ID, `]" id="options_`, setting.ID, `" value="`, setting.Value, `"`, sizeAttr, disabledIf(page.Reset), ` />`)
		}

		if setting.Desc != "" {
			c.O(`
						<div class="smalltext">`, setting.Desc, `</div>`)
		}

		c.O(`
					</td>
				</tr>`)
	}

	c.O(`
				<tr class="windowbg2">
					<td align="center" colspan="2"><br /><input type="submit" name="submit" value="`, c.Txt("10"), `" /></td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// themeSetForm backs template_set_settings.
type themeSetForm struct {
	ThemeID         int
	Name            string
	ActualThemeURL  string
	ActualImagesURL string
	ActualThemeDir  string
	Fields          []*themeField
}

// templateThemeSetSettings is template_set_settings().
func templateThemeSetSettings(c *Ctx) {
	page := c.Page.(*themeSetForm)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<form action="`, scripturl, `?action=theme;sa=settings;th=`, page.ThemeID, `" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" width="80%" cellspacing="0" cellpadding="4" align="center" class="tborder">
				<tr class="titlebg">
					<td colspan="2"><a href="`, scripturl, `?action=helpadmin;help=theme_settings" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("theme4"), ` - `, page.Name, `</td>
				</tr>`)

	// !!! Why can't I edit the default theme popup.
	if page.ThemeID != 1 {
		c.O(`
				<tr class="catbg">
					<td colspan="2"><img src="`, imagesURL, `/icons/config_sm.gif" alt="" align="top" /> `, c.Txt("theme_edit"), `</td>
				</tr>
				<tr class="windowbg2">
					<td colspan="2" style="padding-bottom: 2ex;">
						<a href="`, scripturl, `?action=theme;th=`, page.ThemeID, `;sesc=`, c.Sc, `;sa=edit;filename=index.template.php">`, c.Txt("theme_edit_index"), `</a><br />
						<a href="`, scripturl, `?action=theme;th=`, page.ThemeID, `;sesc=`, c.Sc, `;sa=edit;filename=style.css">`, c.Txt("theme_edit_style"), `</a>
					</td>
				</tr>`)
	}

	c.O(`
				<tr class="catbg">
					<td colspan="2"><img src="`, imagesURL, `/icons/config_sm.gif" alt="" align="top" /> `, c.Txt("theme5"), `</td>
				</tr>
				<tr class="windowbg2">
					<td>`, c.Txt("actual_theme_name"), `</td>
					<td><input type="text" name="options[name]" value="`, page.Name, `" size="32" /></td>
				</tr>
				<tr class="windowbg2">
					<td>`, c.Txt("actual_theme_url"), `</td>
					<td><input type="text" name="options[theme_url]" value="`, page.ActualThemeURL, `" size="50" style="max-width: 100%; width: 50ex;" /></td>
				</tr>
				<tr class="windowbg2">
					<td>`, c.Txt("actual_images_url"), `</td>
					<td><input type="text" name="options[images_url]" value="`, page.ActualImagesURL, `" size="50" style="max-width: 100%; width: 50ex;" /></td>
				</tr>
				<tr class="windowbg2">
					<td style="padding-bottom: 2ex;">`, c.Txt("actual_theme_dir"), `</td>
					<td style="padding-bottom: 2ex;"><input type="text" name="options[theme_dir]" value="`, page.ActualThemeDir, `" size="50" style="max-width: 100%; width: 50ex;" /></td>
				</tr>
				<tr class="catbg">
					<td colspan="2"><img src="`, imagesURL, `/icons/config_sm.gif" alt="" align="top" /> `, c.Txt("theme6"), `</td>
				</tr>`)

	for _, setting := range page.Fields {
		c.O(`
			<tr class="windowbg2">
				<td colspan="2">`)

		defPrefix := ""
		if setting.Default {
			defPrefix = "default_"
		}
		switch setting.Type {
		case "checkbox":
			c.O(`
					<input type="hidden" name="`, defPrefix, `options[`, setting.ID, `]" value="0" />
					<label for="`, setting.ID, `"><input type="checkbox" name="`, defPrefix, `options[`, setting.ID, `]" id="`, setting.ID, `"`, checkedIf(!empty(setting.Value)), ` value="1" class="check" /> `, setting.Label, `</label>`)
		case "list":
			c.O(`
					<label for="`, setting.ID, `">`, setting.Label, `</label>
					<select name="`, defPrefix, `options[`, setting.ID, `]" id="`, setting.ID, `">`)
			for _, opt := range setting.Options {
				c.O(`
						<option value="`, opt.Value, `"`, selAttr(opt.Value == setting.Value), `>`, opt.Label, `</option>`)
			}
			c.O(`
					</select>`)
		default:
			sizeAttr := ` size="40"`
			if setting.Type == "number" {
				sizeAttr = ` size="5"`
			}
			c.O(`
					<label for="`, setting.ID, `">`, setting.Label, `</label>
					<input type="text" name="`, defPrefix, `options[`, setting.ID, `]" id="`, setting.ID, `" value="`, setting.Value, `"`, sizeAttr, ` />`)
		}

		if setting.Desc != "" {
			c.O(`
					<div class="smalltext">`, setting.Desc, `</div>`)
		}

		c.O(`
				</td>
			</tr>`)
	}

	c.O(`
				<tr class="windowbg2">
					<td align="center" colspan="2" style="padding-top: 1ex; padding-bottom: 1ex;"><input type="submit" name="submit" value="`, c.Txt("10"), `" /></td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// templateThemeMain is template_main() — the theme admin settings page.
func templateThemeMain(c *Ctx) {
	page := c.Page.(*ThemeAdminCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()
	a := c.App

	c.O(`
		<form action="`, scripturl, `?action=theme;sa=admin" method="post" accept-charset="`, c.CharacterSet, `">
			<input type="hidden" value="0" name="options[theme_allow]" />
			<input type="hidden" value="0" name="options[theme_default]" />

			<table width="80%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td colspan="3">
						<a href="`, scripturl, `?action=helpadmin;help=themes" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a>
						`, c.Txt("themeadmin_title"), `
					</td>
				</tr>
				<tr class="windowbg">
					<td colspan="3" class="smalltext" style="padding: 2ex;">`, c.Txt("themeadmin_explain"), `</td>
				</tr>
				<tr class="windowbg2">
					<td colspan="3"><label for="options-theme_allow"><input type="checkbox" name="options[theme_allow]" id="options-theme_allow" value="1"`, checkedIf(!a.SettingEmpty("theme_allow")), ` class="check" /> `, c.Txt("theme_allow"), `</label></td>
				</tr>
				<tr class="windowbg2">
					<td colspan="3"><label for="options-theme_default"><input type="checkbox" name="options[theme_default]" id="options-theme_default" value="1"`, checkedIf(!a.SettingEmpty("theme_default")), ` class="check" /> `, c.Txt("theme_default"), `</label></td>
				</tr>
				<tr class="windowbg2">
					<td style="width: 20ex;">`, c.Txt("theme_guests"), `:</td>
					<td style="width: 20ex;" align="right">
						<select name="options[theme_guests]">`)
	for _, th := range page.Themes {
		c.O(`
							<option value="`, th.ID, `"`, selAttr(page.ThemeGuests == itoa(th.ID)), `>`, th.Name, `</option>`)
	}
	c.O(`
						</select>
					</td>
					<td class="smalltext">&nbsp; <a href="`, scripturl, `?action=theme;sa=pick;u=-1;sesc=`, c.Sc, `">`, c.Txt("theme_select"), `</a></td>
				</tr>
				<tr class="windowbg2">
					<td style="width: 20ex;">`, c.Txt("theme_reset"), `:</td>
					<td style="width: 20ex;" align="right">
						<select name="theme_reset">
							<option value="-1" selected="selected">`, c.Txt("theme_nochange"), `</option>
							<option value="0">`, c.Txt("theme_forum_default"), `</option>`)
	for _, th := range page.Themes {
		c.O(`
							<option value="`, th.ID, `">`, th.Name, `</option>`)
	}
	c.O(`
						</select>
					</td>
					<td class="smalltext">&nbsp; <a href="`, scripturl, `?action=theme;sa=pick;u=0;sesc=`, c.Sc, `">`, c.Txt("theme_select"), `</a></td>
				</tr>
				<tr class="windowbg2">
					<td colspan="3" align="center" valign="middle" style="padding-top: 2ex; padding-bottom: 2ex;"><input type="submit" name="submit" value="`, c.Txt("10"), `" /></td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>
		<table width="80%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder" style="margin-bottom: 2ex; margin-top: 2ex;">
			<tr class="titlebg">
				<td><a href="`, scripturl, `?action=helpadmin;help=latest_themes" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("theme_latest"), `</td>
			</tr>
			<tr>
				<td class="windowbg2" id="themeLatest">`, c.Txt("theme_latest_fetch"), `</td>
			</tr>
		</table>`)
	if !page.CanCreateNew {
		c.O(`
		<b>`, c.Txt("theme_install_writable"), `</b><br /><br />`)
	}
}
