package app

// Ports of Themes/default/ManageCalendar.template.php:
// template_manage_holidays / edit_holiday / modify_settings.

// templateManageHolidays is template_manage_holidays().
func templateManageHolidays(c *Ctx) {
	page := c.Page.(*ModifyHolidaysCtx)
	scripturl := c.App.ScriptURL

	c.O(`
<form action="`, scripturl, `?action=managecalendar;sa=holidays" method="post" accept-charset="`, c.CharacterSet, `">
	<table width="100%" cellspacing="0" cellpadding="4" border="0" class="tborder">
		<tr class="titlebg">
			<td colspan="3">`, c.Txt("current_holidays"), `</td>
		</tr><tr class="catbg3">
			<td colspan="3" height="32">`, c.Txt("139"), `: `, page.PageIndex, `</td>
		</tr><tr class="titlebg">
			<td align="left">`, c.Txt("holidays_title"), `</td>
			<td align="left">`, c.Txt("317"), `</td>
			<td align="center" width="4%"><input type="checkbox" onclick="invertAll(this, this.form);" class="check" /></td>
		</tr>`)

	alternate := false
	for _, h := range page.Holidays {
		rowClass := "windowbg2"
		if alternate {
			rowClass = "windowbg"
		}
		c.O(`
		<tr class="`, rowClass, `">
			<td align="left"><a href="`, scripturl, `?action=managecalendar;sa=editholiday;holiday=`, h.ID, `">`, h.Title, `</a></td>
			<td align="left">`, h.Date, `</td>
			<td align="center" width="4%"><input type="checkbox" name="holiday[`, h.ID, `]" class="check" /></td>
		</tr>`)
		alternate = !alternate
	}

	c.O(`
		<tr class="titlebg">
			<td align="left"><a href="`, scripturl, `?action=managecalendar;sa=editholiday">`, c.Txt("holidays_add"), `</a></td>
			<td colspan="2" align="right">
				<input type="submit" name="delete" style="font-weight: normal;" value="`, c.Txt("quickmod_delete_selected"), `" onclick="if (!confirm('`, c.Txt("holidays_delete_confirm"), `')) return false;" />
				<input type="hidden" name="sc" value="`, c.Sc, `" />
			</td>
		</tr>
	</table>
</form>`)
}

// templateEditHoliday is template_edit_holiday().
func templateEditHoliday(c *Ctx) {
	page := c.Page.(*EditHolidayCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var monthLength = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];

			function generateDays()
			{
				var days = 0, selected = 0;
				var dayElement = document.getElementById("day"), yearElement = document.getElementById("year"), monthElement = document.getElementById("month");

				monthLength[1] = 28;
				if (yearElement.options[yearElement.selectedIndex].value % 4 == 0)
					monthLength[1] = 29;

				selected = dayElement.selectedIndex;
				while (dayElement.options.length)
					dayElement.options[0] = null;

				days = monthLength[monthElement.value - 1];

				for (i = 1; i <= days; i++)
					dayElement.options[dayElement.length] = new Option(i, i);

				if (selected < days)
					dayElement.selectedIndex = selected;
			}
		// ]]></script>`)

	c.O(`
<form action="`, scripturl, `?action=managecalendar;sa=editholiday" method="post" accept-charset="`, c.CharacterSet, `">
	<table width="60%" cellspacing="0" cellpadding="4" border="0" align="center" class="tborder">
		<tr class="titlebg">
			<td colspan="2">`, c.PageTitle, `</td>
		</tr><tr class="windowbg2">
			<td width="25%" align="right">`, c.Txt("holidays_title_label"), `:</td>
			<td><input type="text" name="title" value="`, page.Title, `" size="60" /></td>
		</tr><tr class="windowbg2">
			<td align="right">`, c.Txt("calendar10"), `</td>
			<td>
				<select name="year" id="year" onchange="generateDays();">
					<option value="0000"`, selAttr(page.Year == 0), `>`, c.Txt("every_year"), `</option>`)
	for year := a.SettingInt("cal_minyear"); year <= a.SettingInt("cal_maxyear"); year++ {
		c.O(`
					<option value="`, year, `"`, selAttr(year == page.Year), `>`, year, `</option>`)
	}
	c.O(`
				</select>&nbsp;
				`, c.Txt("calendar9"), `&nbsp;
				<select name="month" id="month" onchange="generateDays();">`)
	for month := 1; month <= 12; month++ {
		c.O(`
					<option value="`, month, `"`, selAttr(month == page.Month), `>`, c.TxtListItem("months", month), `</option>`)
	}
	c.O(`
				</select>&nbsp;
				`, c.Txt("calendar11"), `&nbsp;
				<select name="day" id="day" onchange="generateDays();">`)
	for day := 1; day <= page.LastDay; day++ {
		c.O(`
				<option value="`, day, `"`, selAttr(day == page.Day), `>`, day, `</option>`)
	}
	c.O(`
			</select>
		</td>
		</tr><tr class="windowbg2">
			<td colspan="2" align="center">`)
	if page.IsNew {
		c.O(`
				<input type="submit" value="`, c.Txt("holidays_button_add"), `" />`)
	} else {
		c.O(`
				<input type="submit" name="edit" value="`, c.Txt("holidays_button_edit"), `" />
				<input type="submit" name="delete" value="`, c.Txt("holidays_button_remove"), `" />
				<input type="hidden" name="holiday" value="`, page.ID, `" />`)
	}
	c.O(`
				<input type="hidden" name="sc" value="`, c.Sc, `" />
			</td>
		</tr>
	</table>
</form>`)
}

// selAttr returns ` selected="selected"` when b is true.
func selAttr(b bool) string {
	if b {
		return ` selected="selected"`
	}
	return ""
}

// templateModifyCalSettings is template_modify_settings().
func templateModifyCalSettings(c *Ctx) {
	page := c.Page.(*CalSettingsCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
	<form action="`, scripturl, `?action=managecalendar;sa=settings" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("calendar_settings"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="right"><label for="cal_enabled">`, c.Txt("setting_cal_enabled"), `</label>:</td>
				<td><input type="checkbox" name="cal_enabled" id="cal_enabled"`, c.checkboxAttr("cal_enabled"), ` class="check" /></td>
			</tr>`)

	if c.CanChangePermissions {
		calPerms := []struct{ label, perm string }{
			{"groups_calendar_view", "calendar_view"},
			{"groups_calendar_post", "calendar_post"},
			{"groups_calendar_edit_own", "calendar_edit_own"},
			{"groups_calendar_edit_any", "calendar_edit_any"},
		}
		// PHP chains the rows: row 1 opens with <tr>; each later row's <tr> joins
		// onto the prior row's </tr> on one line; a final </tr> closes the block.
		for i, p := range calPerms {
			if i == 0 {
				c.O(`
			<tr class="windowbg2">`)
			} else {
				c.O(`
			</tr><tr class="windowbg2">`)
			}
			c.O(`
				<td align="right" valign="top" width="50%">`, c.Txt(p.label), `:</td>
				<td width="50%">`)
			c.themeInlinePermissions(p.perm)
			c.O(`
				</td>`)
		}
		c.O(`
			</tr>`)
	}

	showSelect := func(name, cur string) {
		c.O(`
				<td>
					<select name="`, name, `">
						<option value="never"`, selAttr(cur == "never"), `>`, c.Txt("setting_cal_show_never"), `</option>
						<option value="cal"`, selAttr(cur == "cal"), `>`, c.Txt("setting_cal_show_cal"), `</option>
						<option value="index"`, selAttr(cur == "index"), `>`, c.Txt("setting_cal_show_index"), `</option>
						<option value="all"`, selAttr(cur == "all"), `>`, c.Txt("setting_cal_show_all"), `</option>
					</select>
				</td>`)
	}

	c.O(`
			<tr class="windowbg2">
				<td colspan="2"><hr width="90%" /></td>
			</tr><tr class="windowbg2">
				<td align="right"><label for="cal_daysaslink">`, c.Txt("setting_cal_daysaslink"), `</label>:</td>
				<td><input type="checkbox" name="cal_daysaslink" id="cal_daysaslink"`, c.checkboxAttr("cal_daysaslink"), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td align="right"><label for="cal_showweeknum">`, c.Txt("setting_cal_showweeknum"), `</label>:</td>
				<td><input type="checkbox" name="cal_showweeknum" id="cal_showweeknum"`, c.checkboxAttr("cal_showweeknum"), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr width="90%" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_days_for_index"), `:</td>
				<td><input type="text" name="cal_days_for_index" value="`, a.Setting("cal_days_for_index"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_showholidays"), `:</td>`)
	showSelect("cal_showholidays", page.ShowHolidays)
	c.O(`
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_showbdays"), `:</td>`)
	showSelect("cal_showbdays", page.ShowBdays)
	c.O(`
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_showevents"), `:</td>`)
	showSelect("cal_showevents", page.ShowEvents)
	c.O(`
			</tr><tr class="windowbg2">
				<td colspan="2"><hr width="90%" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_defaultboard"), `:</td>
				<td>
					<select name="cal_defaultboard">`)
	for _, b := range page.CalBoards {
		c.O(`
						<option value="`, b[0], `"`, selAttr(atoi(b[0]) == atoi(a.Setting("cal_defaultboard"))), `>`, b[1], `</option>`)
	}
	c.O(`
					</select>
				</td>
			</tr><tr class="windowbg2">
				<td align="right"><label for="cal_allow_unlinked">`, c.Txt("setting_cal_allow_unlinked"), `</label>:</td>
				<td><input type="checkbox" name="cal_allow_unlinked" id="cal_allow_unlinked"`, c.checkboxAttr("cal_allow_unlinked"), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td align="right"><label for="cal_showInTopic">`, c.Txt("setting_cal_showInTopic"), `</label>:</td>
				<td><input type="checkbox" name="cal_showInTopic" id="cal_showInTopic"`, c.checkboxAttr("cal_showInTopic"), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr width="90%" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_minyear"), `:</td>
				<td><input type="text" name="cal_minyear" value="`, a.Setting("cal_minyear"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_maxyear"), `:</td>
				<td><input type="text" name="cal_maxyear" value="`, a.Setting("cal_maxyear"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr width="90%" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_bdaycolor"), `:</td>
				<td><input type="text" name="cal_bdaycolor" value="`, a.Setting("cal_bdaycolor"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_eventcolor"), `:</td>
				<td><input type="text" name="cal_eventcolor" value="`, a.Setting("cal_eventcolor"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_holidaycolor"), `:</td>
				<td><input type="text" name="cal_holidaycolor" value="`, a.Setting("cal_holidaycolor"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr width="90%" /></td>
			</tr><tr class="windowbg2">
				<td align="right"><label for="cal_allowspan">`, c.Txt("setting_cal_allowspan"), `</label>:</td>
				<td><input type="checkbox" name="cal_allowspan" id="cal_allowspan"`, c.checkboxAttr("cal_allowspan"), ` class="check" /></td>
			</tr><tr class="windowbg2">
				<td align="right">`, c.Txt("setting_cal_maxspan"), `:</td>
				<td><input type="text" name="cal_maxspan" value="`, a.Setting("cal_maxspan"), `" size="40" /></td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" value="`, c.Txt("save_settings"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
