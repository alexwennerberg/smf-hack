package app

// Hand-port of Themes/default/Calendar.template.php template_main() (the month
// grid) and template_event_post() (the standalone event-post form).

// templateCalendarMain is template_main() from Calendar.template.php.
func templateCalendarMain(c *Ctx) {
	page := c.Page.(*CalendarCtx)
	scripturl := c.App.ScriptURL
	a := c.App

	c.O(`
		<div style="padding: 3px;">`)
	c.themeLinktree()
	c.O(`</div>

		<table cellspacing="1" cellpadding="2" width="100%" class="bordercolor">
			<tr class="titlebg"><td style="font-size: x-large;" align="center" colspan="7">`, c.TxtListItem("months_titles", page.CurrentMonth), ` `, page.CurrentYear, `</td></tr>
			<tr>`)

	for _, day := range page.WeekDays {
		c.O(`
				<td class="titlebg2" width="14%" align="center">`, c.TxtListItem("days", day), `</td>`)
	}
	c.O(`
			</tr>`)

	for _, week := range page.Weeks {
		c.O(`
			<tr>`)
		for _, day := range week.Days {
			cls := "windowbg"
			if day.IsToday {
				cls = "calendar_today"
			}
			c.O(`
				<td valign="top" style="height: 100px; padding: 2px;" class="`, cls, `">`)

			if day.Day != 0 {
				if !a.SettingEmpty("cal_daysaslink") && page.CanPost {
					c.O(`
					<a href="`, scripturl, `?action=calendar;sa=post;month=`, page.CurrentMonth, `;year=`, page.CurrentYear, `;day=`, day.Day, `;sesc=`, c.Sc, `">`, day.Day, `</a>`)
				} else {
					c.O(`
					`, day.Day)
				}

				if day.IsFirstDay {
					c.O(`<span class="smalltext"> - `, c.Txt("calendar51"), ` `, week.Number, `</span>`)
				}

				if len(day.Holidays) > 0 {
					c.O(`
					<div class="smalltext" style="color: #`, a.Setting("cal_holidaycolor"), `;">`, c.Txt("calendar5"), ` `, joinStr(day.Holidays, ", "), `</div>`)
				}

				if len(day.Birthdays) > 0 {
					c.O(`
					<div class="smalltext">
						<span style="color: #`, a.Setting("cal_bdaycolor"), `;">`, c.Txt("calendar3"), `</span> `)
					for _, m := range day.Birthdays {
						ageStr := ""
						if m.HasAge {
							ageStr = " (" + itoa(m.Age) + ")"
						}
						sep := ", "
						if m.IsLast {
							sep = ""
						}
						c.O(`
						<a href="`, scripturl, `?action=profile;u=`, m.ID, `">`, m.Name, ageStr, `</a>`, sep)
					}
					c.O(`
					</div>`)
				}

				if len(day.Events) > 0 {
					c.O(`
					<div class="smalltext">
						<span style="color: #`, a.Setting("cal_eventcolor"), `;">`, c.Txt("calendar4"), `</span>`)
					for _, ev := range day.Events {
						if ev.CanEdit {
							c.O(`
						<a href="`, ev.ModifyHref, `" style="color: #FF0000;">*</a> `)
						}
						sep := ", "
						if ev.IsLast {
							sep = ""
						}
						c.O(`
						`, ev.Link, sep)
					}
					c.O(`
					</div>`)
				}
			}

			c.O(`
				</td>`)
		}
		c.O(`
			</tr>`)
	}

	c.O(`
		</table>

		<form action="`, scripturl, `?action=calendar" method="post" accept-charset="`, c.CharacterSet, `">
			<table cellspacing="0" cellpadding="3" width="100%" class="tborder" style="border-top: 0;">
				<tr class="titlebg2">
					<td>`)
	if page.HasPrev {
		c.O(`
						<b><a href="`, page.PrevHref, `">&#171; `, c.TxtListItem("months_short", page.PrevMonth), ` `, page.PrevYear, `</a></b>`)
	}
	c.O(`
					</td>
					<td align="center">`)
	if page.CanPost {
		c.O(`
						<a href="`, scripturl, `?action=calendar;sa=post;month=`, page.CurrentMonth, `;year=`, page.CurrentYear, `;sesc=`, c.Sc, `">`, c.createButton("calendarpe.gif", "calendar23", "calendar23", `align="middle"`), `</a>`)
	}
	c.O(`
					</td>
					<td align="center">
						<select name="month">`)
	for n := 1; n <= 12; n++ {
		sel := ""
		if n == page.CurrentMonth {
			sel = ` selected="selected"`
		}
		c.O(`
							<option value="`, n, `"`, sel, `>`, c.TxtListItem("months", n), `</option>`)
	}
	c.O(`
						</select>&nbsp;
						<select name="year">`)
	for year := a.SettingInt("cal_minyear"); year <= a.SettingInt("cal_maxyear"); year++ {
		sel := ""
		if year == page.CurrentYear {
			sel = ` selected="selected"`
		}
		c.O(`
							<option value="`, year, `"`, sel, `>`, year, `</option>`)
	}
	c.O(`
						</select>&nbsp;
						<input type="submit" value="`, c.Txt("305"), `" />
					</td>
					<td align="center">`)
	if page.CanPost {
		c.O(`
						<a href="`, scripturl, `?action=calendar;sa=post;month=`, page.CurrentMonth, `;year=`, page.CurrentYear, `;sesc=`, c.Sc, `">`, c.createButton("calendarpe.gif", "calendar23", "calendar23", `align="middle"`), `</a>`)
	}
	rightAlign := "right"
	if c.RightToLeft {
		rightAlign = "left"
	}
	c.O(`
					</td>
					<td align="`, rightAlign, `">`)
	if page.HasNext {
		c.O(`
						<b><a href="`, page.NextHref, `">`, c.TxtListItem("months_short", page.NextMonth), ` `, page.NextYear, ` &#187;</a></b>`)
	}
	c.O(`
					</td>
				</tr>
			</table>
		</form>`)
}

// joinStr is a tiny strings.Join wrapper to keep the template terse.
func joinStr(xs []string, sep string) string {
	out := ""
	for i, s := range xs {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// templateEventPost is template_event_post() from Calendar.template.php — the
// standalone calendar event post/edit/delete form.
func templateEventPost(c *Ctx) {
	page := c.Page.(*EventPostCtx)
	event := page.Event
	a := c.App
	scripturl := a.ScriptURL

	sel := func(cond bool) string {
		if cond {
			return ` selected="selected"`
		}
		return ""
	}

	// Start the javascript for drop down boxes...
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

			function toggleLinked(form)
			{
				form.board.disabled = !form.link_to_board.checked;
			}


			function saveEntities()
			{
				document.forms.postevent.evtitle.value = document.forms.postevent.evtitle.value.replace(/&#/g, "&#38;#");
			}
		// ]]></script>

		<form action="`, scripturl, `?action=calendar;sa=post" method="post" name="postevent" accept-charset="`, c.CharacterSet, `" onsubmit="submitonce(this);saveEntities();" style="margin: 0;">
			<table width="55%" align="center" cellpadding="0" cellspacing="3">
				<tr>
					<td valign="bottom" colspan="2">
						`)
	c.themeLinktree()
	c.O(`
					</td>
				</tr>
			</table>`)

	if event.New {
		c.O(`
			<input type="hidden" name="eventid" value="`, event.ID, `" />`)
	}

	// Start the main table.
	c.O(`
			<table border="0" width="55%" align="center" cellspacing="1" cellpadding="3" class="bordercolor">
				<tr class="titlebg">
					<td>`, c.PageTitle, `</td>
				</tr>
				<tr>
					<td class="windowbg">
						<table border="0" cellpadding="3" width="100%">`)

	if len(page.ErrorMessages) > 0 {
		serious := ""
		if page.ErrorType == "serious" {
			serious = "<b>" + c.Txt("error_while_submitting") + "</b>"
		}
		c.O(`
							<tr>
								<td></td>
								<td>
									`, serious, `
									<div style="color: red; margin: 1ex 0 2ex 3ex;">
										`, joinStr(page.ErrorMessages, "<br />"), `
									</div>
								</td>
							</tr>`)
	}

	noEvent := ""
	if page.PostError["no_event"] {
		noEvent = ` style="color: red;"`
	}
	c.O(`
							<tr>
								<td align="right">
									<b`, noEvent, `>`, c.Txt("calendar12"), `</b>
								</td>
								<td class="smalltext">
									<input type="text" name="evtitle" maxlength="30" size="30" value="`, event.Title, `" style="width: 90%;" />
								</td>
							</tr><tr>
								<td></td>
								<td class="smalltext">
									<input type="hidden" name="calendar" value="1" />`, c.Txt("calendar10"), `&nbsp;
									<select name="year" id="year" onchange="generateDays();">`)

	// Show a list of all the years we allow...
	for year := a.SettingInt("cal_minyear"); year <= a.SettingInt("cal_maxyear"); year++ {
		c.O(`
										<option value="`, year, `"`, sel(year == event.Year), `>`, year, `</option>`)
	}

	c.O(`
									</select>&nbsp;
									`, c.Txt("calendar9"), `&nbsp;
									<select name="month" id="month" onchange="generateDays();">`)

	// There are 12 months per year - ensure that they all get listed.
	for month := 1; month <= 12; month++ {
		c.O(`
										<option value="`, month, `"`, sel(month == event.Month), `>`, c.TxtListItem("months", month), `</option>`)
	}

	c.O(`
									</select>&nbsp;
									`, c.Txt("calendar11"), `&nbsp;
									<select name="day" id="day">`)

	// This prints out all the days in the current month - this changes dynamically as we switch months.
	for day := 1; day <= event.LastDay; day++ {
		c.O(`
										<option value="`, day, `"`, sel(day == event.Day), `>`, day, `</option>`)
	}

	c.O(`
									</select>
								</td>
							</tr>`)

	// If events can span more than one day then allow the user to select how long it should last.
	if !a.SettingEmpty("cal_allowspan") {
		c.O(`
							<tr>
								<td align="right"><b>`, c.Txt("calendar54"), `</b></td>
								<td class="smalltext">
									<select name="span">`)

		for days := 1; days <= a.SettingInt("cal_maxspan"); days++ {
			c.O(`
										<option value="`, days, `"`, sel(event.Span == days), `>`, days, `</option>`)
		}

		c.O(`
									</select>
								</td>
							</tr>`)
	}

	// If this is a new event let the user specify which board they want the linked post to be put into.
	if event.New {
		c.O(`
							<tr>
								<td align="right"><b>`, c.Txt("calendar_link_event"), `</b></td>
								<td class="smalltext">
									<input type="checkbox" class="check" name="link_to_board" checked="checked" onclick="toggleLinked(this.form);" />
								</td>
							</tr>
							<tr>
								<td align="right"><b>`, c.Txt("calendar13"), `</b></td>
								<td class="smalltext">
									<select id="board" name="board" onchange="this.form.submit();">`)

		for _, board := range event.Boards {
			c.O(`
										<option value="`, board.ID, `"`, sel(board.ID == event.Board), `>`, board.CatName, ` - `, board.Prefix, board.Name, `</option>`)
		}

		c.O(`
									</select>
								</td>
							</tr>`)
	}

	submitLabel := c.Txt("105")
	if !event.New {
		submitLabel = c.Txt("10")
	}
	c.O(`
							<tr align="center">
								<td colspan="2">
									<input type="submit" value="`, submitLabel, `" />`)
	// Delete button?
	if !event.New {
		c.O(`
									<input type="submit" name="deleteevent" value="`, c.Txt("calendar22"), `" onclick="return confirm('`, c.Txt("calendar_confirm_delete"), `');" />`)
	}

	c.O(`
									<input type="hidden" name="sc" value="`, c.Sc, `" />
									<input type="hidden" name="eventid" value="`, event.ID, `" />
								</td>
							</tr>`)

	c.O(`
						</table>
					</td>
				</tr>
			</table>
		</form>`)
}
