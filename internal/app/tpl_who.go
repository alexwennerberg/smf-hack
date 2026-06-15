package app

// Hand-port of Themes/default/Who.template.php (template_main).

// templateWhoMain is template_main() from Who.template.php.
func templateWhoMain(c *Ctx) {
	page := c.Page.(*WhoCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()

	// Display the table header and linktree.
	c.O(`
	<div style="padding: 3px;">`)
	c.themeLinktree()
	userAsc := ";asc"
	if page.SortDirection != "down" && page.SortBy == "user" {
		userAsc = ""
	}
	userImg := ""
	if page.SortBy == "user" {
		userImg = `<img src="` + images + `/sort_` + page.SortDirection + `.gif" alt="" />`
	}
	timeAsc := ""
	if page.SortDirection == "down" && page.SortBy == "time" {
		timeAsc = ";asc"
	}
	timeImg := ""
	if page.SortBy == "time" {
		timeImg = `<img src="` + images + `/sort_` + page.SortDirection + `.gif" alt="" />`
	}
	c.O(`</div>
	<table cellpadding="3" cellspacing="0" border="0" width="100%" class="tborder">
		<tr class="titlebg">
			<td width="30%"><a href="`, scripturl, `?action=who;start=`, page.Start, `;sort=user`, userAsc, `">`, c.Txt("who_user"), ` `, userImg, `</a></td>
			<td style="width: 14ex;"><a href="`, scripturl, `?action=who;start=`, page.Start, `;sort=time`, timeAsc, `">`, c.Txt("who_time"), ` `, timeImg, `</a></td>
			<td>`, c.Txt("who_action"), `</td>
		</tr>`)

	alternate := true
	for _, wm := range page.Members {
		member := wm.Member
		alt := "2"
		if !alternate {
			alt = ""
		}
		c.O(`
		<tr class="windowbg`, alt, `">
			<td>`)

		// Guests can't be messaged.
		if !member.IsGuest {
			openA, closeA := "", ""
			if page.CanSendPM {
				openA = `<a href="` + member.OnlineHref + `" title="` + member.OnlineLabel + `">`
				closeA = `</a>`
			}
			onlineBtn := member.OnlineText
			if !c.Theme.Empty("use_image_buttons") {
				onlineBtn = `<img src="` + member.OnlineImageHref + `" alt="` + member.OnlineText + `" align="middle" />`
			}
			c.O(`
				<div style="float: right; width: 14ex;">
					`, openA, onlineBtn, closeA, `
				</div>`)
		}

		hiddenStyle := ""
		if wm.IsHidden {
			hiddenStyle = ` style="font-style: italic;"`
		}
		nameCell := member.Name
		if !member.IsGuest {
			colorStyle := ""
			if wm.Color != "" {
				colorStyle = ` style="color: ` + wm.Color + `"`
			}
			nameCell = `<a href="` + member.Href + `" title="` + c.Txt("92") + ` ` + member.Name + `"` + colorStyle + `>` + member.Name + `</a>`
		}
		c.O(`
				<span`, hiddenStyle, `>`, nameCell, `</span>`)

		if wm.IP != "" {
			c.O(`
				(<a href="`, scripturl, `?action=trackip;searchip=`, wm.IP, `" target="_blank">`, wm.IP, `</a>)`)
		}

		c.O(`
			</td>
			<td nowrap="nowrap">`, wm.Time, `</td>
			<td>`, wm.Action, `</td>
		</tr>`)

		alternate = !alternate
	}

	c.O(`
		<tr class="titlebg">
			<td colspan="3"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
		</tr>
	</table>`)
}
