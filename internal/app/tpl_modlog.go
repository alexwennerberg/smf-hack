package app

// templateModlog is template_main() from Modlog.template.php.
func templateModlog(c *Ctx) {
	page := c.Page.(*ModlogCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<form action="`, scripturl, `?action=modlog" method="post" accept-charset="`, c.CharacterSet, `">
			<input type="hidden" name="order" value="`, page.Order, `" />
			<input type="hidden" name="dir" value="`, page.Dir, `" />
			<input type="hidden" name="start" value="`, page.Start, `" />
			<div class="tborder">
				<table border="0" cellspacing="1" cellpadding="4" width="100%">
					<tr class="titlebg">
						<td>
							<div style="float: left;"><a href="`, scripturl, `?action=helpadmin;help=modlog" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("modlog_moderation_log"), `</div>
							<div align="right">`)

	if page.SearchParams == "" {
		c.O(c.Txt("modlog_total_entries"))
	} else {
		c.O(c.Txt("modlog_search_result"))
	}
	c.O(`: `, page.EntryCount, `</div>
						</td>
					</tr>
					<tr class="windowbg">
						<td class="smalltext" style="padding: 2ex;">`, c.Txt("modlog_moderation_log_desc"), `</td>
					</tr>`)

	// Only display page numbers if not a result of a search.
	if page.PageIndex != "" {
		c.O(`
					<tr class="catbg">
						<td>`, c.Txt("139"), `: `, page.PageIndex, `</td>
					</tr>`)
	}

	c.O(`
				</table>
				<table border="0" cellspacing="1" cellpadding="4" width="100%">
					<tr class="titlebg">
						<td width="10" align="center"><input type="checkbox" name="all" class="check" onclick="invertAll(this, this.form);" /></td>`)

	for _, column := range page.Columns {
		c.O(`
						<td><a href="`, column.Href, `">`)
		if column.Selected {
			c.O(`<b>`, column.Label, `</b> <img src="`, imagesURL, `/sort_`, page.SortDirection, `.gif" alt="" />`)
		} else {
			c.O(column.Label)
		}
		c.O(`</a></td>`)
	}

	c.O(`
					</tr>`)

	for _, entry := range page.Entries {
		disabled := ""
		if !entry.Editable {
			disabled = ` disabled="disabled"`
		}
		c.O(`
					<tr class="windowbg2">
						<td rowspan="2" class="windowbg" align="center"><input type="checkbox" class="check" name="delete[]" value="`, entry.ID, `"`, disabled, ` /></td>
						<td>`, entry.Action, `</td>
						<td>`, entry.Time, `</td>
						<td>`, entry.ModeratorLink, `</td>
						<td>`, entry.Position, `</td>
						<td>`, entry.IP, `</td>
					</tr>
					<tr>
						<td colspan="5" class="windowbg">`)

		for _, ex := range entry.Extra {
			c.O(`
							<i>`, ex.Key, `</i>: `, ex.Value)
		}
		c.O(`
						</td>
					</tr>`)
	}

	if len(page.Entries) == 0 {
		c.O(`
					<tr>
						<td class="windowbg2" align="center" colspan="7">
							<b>`, c.Txt("modlog_no_entries_found"), `</b>
						</td>
					</tr>`)
	}

	c.O(`
				</table>
				<table border="0" cellspacing="1" cellpadding="4" width="100%">
					<tr class="titlebg">
						<td align="right" valign="bottom">
							<div style="float: left;">
								`, c.Txt("modlog_search"), ` (`, c.Txt("modlog_by"), `: `, page.SearchLabel, `):
								<input type="text" name="search" size="18" value="`, page.SearchString, `" /> <input type="submit" value="`, c.Txt("modlog_go"), `" />
							</div>

							<input type="submit" name="remove" value="`, c.Txt("modlog_remove"), `" />
							<input type="submit" name="removeall" value="`, c.Txt("modlog_removeall"), `" />
						</td>
					</tr>`)

	if page.PageIndex != "" {
		c.O(`
					<tr class="catbg">
						<td>`, c.Txt("139"), `: `, page.PageIndex, `</td>
					</tr>`)
	}

	c.O(`
				</table>
			</div>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}
