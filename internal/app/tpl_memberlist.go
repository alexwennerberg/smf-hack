package app

// Hand-port of Themes/default/Memberlist.template.php (template_main and
// template_search).

import "strings"

// mlSortTabs emits the all/search tab strip (shared by both templates).
func mlSortTabs(c *Ctx, page *MemberlistCtx) {
	scripturl := c.App.ScriptURL

	c.O(`
	<table cellpadding="0" cellspacing="0" border="0" style="margin-left: 10px;">
		<tr>
			<td class="mirrortab_first">&nbsp;</td>`)

	for _, link := range page.SortLinks {
		action := ""
		if link.Action != "" {
			action = ";sa=" + link.Action
		}
		if link.Selected {
			c.O(`
				<td class="mirrortab_active_first">&nbsp;</td>
				<td valign="top" class="mirrortab_active_back">
					<a href="`+scripturl+`?action=mlist`+action+`">`, link.Label, `</a>
				</td>
				<td class="mirrortab_active_last">&nbsp;</td>`)
		} else {
			c.O(`
				<td valign="top" class="mirrortab_back">
					<a href="`+scripturl+`?action=mlist`+action+`">`, link.Label, `</a>
				</td>`)
		}
	}

	c.O(`
			<td class="mirrortab_last">&nbsp;</td>
		</tr>
	</table>`)
}

// mlOldStyleLinks builds the non-tab style link list.
func mlOldStyleLinks(c *Ctx, page *MemberlistCtx) string {
	scripturl := c.App.ScriptURL
	var links []string
	for _, link := range page.SortLinks {
		action := ""
		if link.Action != "" {
			action = ";sa=" + link.Action
		}
		sel := ""
		if link.Selected {
			sel = `<img src="` + c.Theme.ImagesURL() + `/selected.gif" alt="&gt;" /> `
		}
		links = append(links, sel+`<a href="`+scripturl+`?action=mlist`+action+`">`+link.Label+`</a>`)
	}
	return strings.Join(links, " | ")
}

// templateMemberlistMain is template_main() from Memberlist.template.php.
func templateMemberlistMain(c *Ctx) {
	page := c.Page.(*MemberlistCtx)

	// Show the link tree.
	c.O(`
	<div style="padding: 3px;">`)
	c.themeLinktree()
	c.O(`</div>`)

	// shall we use the tabs?
	if !c.Theme.Empty("use_tabs") {
		mlSortTabs(c, page)
	}

	c.O(`
	<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%" class="bordercolor">`)

	// Old style tabs?
	if c.Theme.Empty("use_tabs") {
		c.O(`
		<tr class="titlebg">
			<td colspan="12">`)
		c.O(`
					`, mlOldStyleLinks(c, page), `
			</td>
		</tr>`)
	}
	rowClass := "titlebg"
	if c.Theme.Empty("use_tabs") {
		rowClass = "catbg"
	}
	c.O(`
		<tr>
			<td colspan="12" class="`, rowClass, `">`)

	// Display page numbers and the a-z links for sorting by name if not a
	// result of a search.
	if !page.HasOldSearch {
		c.O(`
				<table width="100%" cellpadding="0" cellspacing="0" border="0">
					<tr>
						<td>`, c.Txt("139"), `: `, page.PageIndex, `</td>
						<td align="right">`, page.LetterLinks+`</td>
					</tr>
				</table>`)
	} else {
		// If this is a result of a search then just show the page numbers.
		c.O(`
				`, c.Txt("139"), `: `, page.PageIndex)
	}

	headClass := "catbg3"
	if c.Theme.Empty("use_tabs") {
		headClass = "titlebg"
	}
	c.O(`
			</td>
		</tr>
		<tr class="`, headClass, `">`)

	// Display each of the column headers of the table.
	for _, column := range page.Columns {
		widthAttr := ""
		if column.Width != "" {
			widthAttr = ` width="` + column.Width + `"`
		}
		colspanAttr := ""
		if column.Colspan != "" {
			colspanAttr = ` colspan="` + column.Colspan + `"`
		}
		if page.HasOldSearch {
			// We're not able (through the template) to sort the search
			// results right now...
			c.O(`
			<td`, widthAttr, colspanAttr, `>
				`, column.Label, `</td>`)
		} else if column.Selected {
			c.O(`
			<td style="width: auto;"` + colspanAttr + ` nowrap="nowrap">
				<a href="` + column.Href + `">` + column.Label + ` <img src="` + c.Theme.ImagesURL() + `/sort_` + page.SortDirection + `.gif" alt="" /></a></td>`)
		} else {
			c.O(`
			<td`, widthAttr, colspanAttr, `>
				`, column.Link, `</td>`)
		}
	}
	c.O(`
		</tr>`)

	// Assuming there are members loop through each one displaying their
	// data.
	if len(page.Members) > 0 {
		for _, member := range page.Members {
			letterID := ""
			if member.SortLetter != "" {
				letterID = ` id="letter` + member.SortLetter + `"`
			}

			onlineCell := ""
			if page.CanSendPM {
				onlineCell += `<a href="` + member.OnlineHref + `" title="` + member.OnlineText + `">`
			}
			if !c.Theme.Empty("use_image_buttons") {
				onlineCell += `<img src="` + member.OnlineImageHref + `" alt="` + member.OnlineText + `" align="middle" />`
			} else {
				onlineCell += member.OnlineLabel
			}
			if page.CanSendPM {
				onlineCell += `</a>`
			}

			emailCell := ""
			if !member.HideEmail {
				emailCell = `<a href="mailto:` + member.Email + `"><img src="` + c.Theme.ImagesURL() + `/email_sm.gif" alt="` + c.Txt("69") + `" title="` + c.Txt("69") + ` ` + member.Name + `" /></a>`
			}

			websiteCell := ""
			if member.WebsiteURL != "" {
				websiteCell = `<a href="` + member.WebsiteURL + `" target="_blank"><img src="` + c.Theme.ImagesURL() + `/www.gif" alt="` + member.WebsiteTitle + `" title="` + member.WebsiteTitle + `" /></a>`
			}

			group := member.Group
			if group == "" {
				group = member.PostGroup
			}

			postBar := ""
			if member.RealPosts > 0 {
				postBar = `<img src="` + c.Theme.ImagesURL() + `/bar.gif" width="` + itoa(member.PostPercent) + `" height="15" alt="" />`
			}

			c.O(`
		<tr style="text-align: center;"`, letterID, `>
			<td class="windowbg2">
				`, onlineCell, `
			</td>
			<td class="windowbg" align="left">`, member.Link, `</td>
			<td class="windowbg2">`, emailCell, `</td>
			<td class="windowbg">`, websiteCell, `</td>
			<td class="windowbg2">`, member.ICQLink, `</td>
			<td class="windowbg2">`, member.AIMLink, `</td>
			<td class="windowbg2">`, member.YIMLink, `</td>
			<td class="windowbg2">`, member.MSNLink, `</td>
			<td class="windowbg" align="left">`, group, `</td>
			<td class="windowbg" align="left">`, member.RegisteredDate, `</td>
			<td class="windowbg2" width="15">`, member.Posts, `</td>
			<td class="windowbg" width="100" align="left">
				`, postBar, `
			</td>
		</tr>`)
		}
	} else {
		// No members?
		c.O(`
		<tr>
			<td colspan="12" class="windowbg">`, c.Txt("170"), `</td>
		</tr>`)
	}

	// Show the page numbers again. (makes 'em easier to find!)
	c.O(`
		<tr>
			<td class="titlebg" colspan="12">`, c.Txt("139"), `: `, page.PageIndex, `</td>
		</tr>
	</table>`)

	// If it is displaying the result of a search show a "search again" link
	// to edit their criteria.
	if page.HasOldSearch {
		c.O(`
			<br />
				<a href="`, c.App.ScriptURL, `?action=mlist;sa=search;search=`, page.OldSearchValue, `">`, c.Txt("mlist_search2"), `</a>`)
	}
}

// templateMemberlistSearch is template_search() from
// Memberlist.template.php.
func templateMemberlistSearch(c *Ctx) {
	page := c.Page.(*MemberlistCtx)
	scripturl := c.App.ScriptURL

	// Start the submission form for the search!
	c.O(`
		<form action="`, scripturl, `?action=mlist;sa=search" method="post" accept-charset="`, c.CharacterSet, `">`)

	// Display that link tree...
	c.O(`
		<div style="padding: 3px;">`)
	c.themeLinktree()
	c.O(`</div>`)

	// Display links to view all/search.
	if !c.Theme.Empty("use_tabs") {
		mlSortTabs(c, page)
		c.O(`
		<div class="tborder">`)
	} else {
		c.O(`
		<div class="bordercolor" style="padding: 1px;">
			<div class="titlebg" style="padding: 4px 4px 4px 10px;">`)
		c.O(`
					`, mlOldStyleLinks(c, page), `
			</div>
		</div>
		<div class="bordercolor" style="padding: 1px">`)
	}

	// Display the input boxes for the form.
	c.O(`

		<div class="windowbg" align="center" style="padding-bottom: 1ex;">
			<table width="440" border="0" cellpadding="0" cellspacing="0">
				<tr>
					<td colspan="2" align="left">
						<br />
						<b>`, c.Txt("582"), `:</b> <input type="text" name="search" value="`, page.OldSearch, `" size="35" /> <input type="submit" name="submit" value="`+c.Txt("182")+`" style="margin-left: 20px;" /><br />
						<br />
					</td>
				</tr>
				<tr>
					<td align="left">
								<label for="fields-email"><input type="checkbox" name="fields[]" id="fields-email" value="email" checked="checked" class="check" /> `, c.Txt("mlist_search_email"), `</label><br />
								<label for="fields-messenger"><input type="checkbox" name="fields[]" id="fields-messenger" value="messenger" class="check" /> `, c.Txt("mlist_search_messenger"), `</label><br />
								<label for="fields-group"><input type="checkbox" name="fields[]" id="fields-group" value="group" class="check" /> `, c.Txt("mlist_search_group"), `</label>
					</td>
					<td align="left" valign="top">
								<label for="fields-name"><input type="checkbox" name="fields[]" id="fields-name" value="name" checked="checked" class="check" /> `, c.Txt("mlist_search_name"), `</label><br />
								<label for="fields-website"><input type="checkbox" name="fields[]" id="fields-website" value="website" class="check" /> `, c.Txt("mlist_search_website"), `</label>
					</td>
				</tr>
			</table>
		</div>
	</div>
</form>`)
}
