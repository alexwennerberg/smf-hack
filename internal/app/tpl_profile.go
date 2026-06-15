package app

// Hand-port of Themes/default/Profile.template.php part 1: the profile
// layers, summary, showPosts, editBuddies, trackUser, trackIP and
// template_error_message. (The edit forms are in tpl_profile2.go.)

import "strings"

func init() {
	layerFuncs["profile_above"] = templateProfileAbove
	layerFuncs["profile_below"] = templateProfileBelow
}

// templateProfileAbove is template_profile_above().
func templateProfileAbove(c *Ctx) {
	page, ok := c.Page.(*ProfileCtx)
	if !ok {
		return
	}

	// Assuming there are actually some areas the user can visit...
	if len(page.Areas) > 0 {
		c.O(`
		<table width="100%" border="0" cellpadding="0" cellspacing="0" style="padding-top: 1ex;">
			<tr>
				<td width="180" valign="top">
					<table border="0" cellpadding="4" cellspacing="1" class="bordercolor" width="170">`)

		// Loop through every area, displaying its name as a header.
		for _, section := range page.Areas {
			c.O(`
						<tr>
							<td class="catbg">`, section.Title, `</td>
						</tr>
						<tr class="windowbg2">
							<td class="smalltext">`)

			// For every section of the area display it, and bold it if it's
			// the current area.
			for i, area := range section.Areas {
				if section.Keys[i] == page.MenuSelected {
					c.O(`
								<b>`, area, `</b><br />`)
				} else {
					c.O(`
								`, area, `<br />`)
				}
			}
			c.O(`
								<br />
							</td>
						</tr>`)
		}
		c.O(`
					</table>
				</td>
				<td width="100%" valign="top">`)
	} else {
		// If no areas exist just open up a containing table.
		c.O(`
		<table width="100%" border="0" cellpadding="0" cellspacing="0" style="padding-top: 1ex;">
			<tr>
				<td width="100%" valign="top">`)
	}

	// If an error occurred whilst trying to save previously, give the user a
	// clue!
	if len(page.ModifyError) > 0 {
		c.O(`
					<table width="85%" cellpadding="0" cellspacing="0" border="0" align="center">
						<tr>
							<td>`)
		templateProfileErrorMessage(c, page)
		c.O(`</td>
						</tr>
					</table>`)
	}
}

// templateProfileBelow is template_profile_below().
func templateProfileBelow(c *Ctx) {
	if _, ok := c.Page.(*ProfileCtx); !ok {
		return
	}
	c.O(`
				</td>
			</tr>
		</table>`)
}

// templateProfileSummary is template_summary().
func templateProfileSummary(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileSummaryCtx)
	member := page.MemberFull
	a := c.App
	scripturl := a.ScriptURL

	// First do the containing table and table header.
	c.O(`
<table border="0" cellpadding="4" cellspacing="1" align="center" class="bordercolor">
	<tr class="titlebg">
		<td width="420" height="26">
			<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;
			`, c.Txt("summary"), ` - `, member.Name, `
		</td>
		<td align="center" width="150">`, c.Txt("232"), `</td>
	</tr>`)

	// Do the left hand column - where all the important user info is
	// displayed.
	c.O(`
	<tr>
		<td class="windowbg" width="420">
			<table border="0" cellspacing="0" cellpadding="2" width="100%">
				<tr>
					<td><b>`, c.Txt("68"), `: </b></td>
					<td>`, member.Name, `</td>
				</tr>`)
	if !a.SettingEmpty("titlesEnable") && member.Title != "" {
		c.O(`
				<tr>
					<td><b>`, c.Txt("title1"), `: </b></td>
					<td>`, member.Title, `</td>
				</tr>`)
	}
	group := member.Group
	if group == "" {
		group = member.PostGroup
	}
	c.O(`
				<tr>
					<td><b>`, c.Txt("86"), `: </b></td>
					<td>`, member.Posts, ` (`, sub.PostsPerDay, ` `, c.Txt("posts_per_day"), `)</td>
				</tr><tr>
					<td><b>`, c.Txt("87"), `: </b></td>
					<td>`, group, `</td>
				</tr>`)

	// If the person looking is allowed, they can check the members IP
	// address and hostname.
	if sub.CanSeeIP {
		c.O(`
				<tr>
					<td width="40%">
						<b>`, c.Txt("512"), `: </b>
					</td><td>
						<a href="`, scripturl, `?action=trackip;searchip=`, member.IP, `" target="_blank">`, member.IP, `</a>
					</td>
				</tr><tr>
					<td width="40%">
						<b>`, c.Txt("hostname"), `: </b>
					</td><td width="55%">
						<div title="`, sub.Hostname, `" style="width: 100%; overflow: hidden; font-style: italic;">`, sub.Hostname, `</div>
					</td>
				</tr>`)
	}

	// If karma enabled show the members karma.
	if a.Setting("karmaMode") == "1" {
		c.O(`
				<tr>
					<td>
						<b>`, a.Setting("karmaLabel"), ` </b>
					</td><td>
						`, member.KarmaGood-member.KarmaBad, `
					</td>
				</tr>`)
	} else if a.Setting("karmaMode") == "2" {
		c.O(`
				<tr>
					<td>
						<b>`, a.Setting("karmaLabel"), ` </b>
					</td><td>
						+`, member.KarmaGood, `/-`, member.KarmaBad, `
					</td>
				</tr>`)
	}
	c.O(`
				<tr>
					<td><b>`, c.Txt("233"), `: </b></td>
					<td>`, member.Registered, `</td>
				</tr><tr>
					<td><b>`, c.Txt("lastLoggedIn"), `: </b></td>
					<td>`, member.LastLogin, `</td>
				</tr>`)

	// Is this member requiring activation and/or banned?
	if sub.ActivateMessage != "" || len(sub.Bans) > 0 {
		c.O(`
				<tr>
					<td colspan="2"><hr size="1" width="100%" class="hrcolor" /></td>
				</tr>`)

		// If the person looking at the summary has permission, and the
		// account isn't activated, give the viewer the ability to do it
		// themselves.
		if sub.ActivateMessage != "" {
			confirmClick := ""
			if sub.ActivateType == 4 {
				confirmClick = `onclick="return confirm('` + c.Txt("profileConfirm") + `');"`
			}
			c.O(`
				<tr>
					<td colspan="2">
						<span style="color: red;">`, sub.ActivateMessage, `</span>&nbsp;(<a href="`+scripturl+`?action=profile2;sa=activateAccount;userID=`+itoa(member.ID)+`;sesc=`+c.Sc+`" `, confirmClick, `>`, sub.ActivateLinkText, `</a>)
					</td>
				</tr>`)
		}

		// If the current member is banned, show a message and possibly a
		// link to the ban.
		if len(sub.Bans) > 0 {
			c.O(`
				<tr>
					<td colspan="2">
						<span style="color: red;">`, c.Txt("user_is_banned"), `</span>&nbsp;[<a href="#" onclick="document.getElementById('ban_info').style.display = document.getElementById('ban_info').style.display == 'none' ? '' : 'none';return false;">`+c.Txt("view_ban")+`</a>]
					</td>
				</tr>
				<tr id="ban_info" style="display: none;">
					<td colspan="2">
						<b>`, c.Txt("user_banned_by_following"), `:</b>`)

			for _, ban := range sub.Bans {
				c.O(`
						<br /><span class="smalltext">`, ban.Explanation, `</span>`)
			}

			c.O(`
					</td>
				</tr>`)
		}
	}

	// Messenger type information.
	c.O(`
				<tr>
					<td colspan="2"><hr size="1" width="100%" class="hrcolor" /></td>
				</tr><tr>
					<td><b>`, c.Txt("513"), `:</b></td>
					<td>`, member.ICQLinkText, `</td>
				</tr><tr>
					<td><b>`, c.Txt("603"), `: </b></td>
					<td>`, member.AIMLinkText, `</td>
				</tr><tr>
					<td><b>`, c.Txt("MSN"), `: </b></td>
					<td>`, member.MSNLinkText, `</td>
				</tr><tr>
					<td><b>`, c.Txt("604"), `: </b></td>
					<td>`, member.YIMLinkText, `</td>
				</tr><tr>
					<td><b>`, c.Txt("69"), `: </b></td>
					<td>`)

	// Only show the email address if it's not hidden.
	if member.EmailPublic {
		c.O(`
						<a href="mailto:`, member.Email, `">`, member.Email, `</a>`)
	} else if !member.HideEmail {
		// ... Or if the one looking at the profile is an admin they can see
		// it anyway.
		c.O(`
						<i><a href="mailto:`, member.Email, `">`, member.Email, `</a></i>`)
	} else {
		c.O(`
						<i>`, c.Txt("722"), `</i>`)
	}

	// Some more information.
	onlineCell := ""
	if sub.CanSendPM {
		onlineCell += `<a href="` + member.OnlineHref + `" title="` + member.OnlineLabel + `">`
	}
	if !c.Theme.Empty("use_image_buttons") {
		onlineCell += `<img src="` + member.OnlineImageHref + `" alt="` + member.OnlineText + `" align="middle" />`
	} else {
		onlineCell += member.OnlineText
	}
	if sub.CanSendPM {
		onlineCell += `</a>`
	}
	if !c.Theme.Empty("use_image_buttons") {
		onlineCell += `<span class="smalltext"> ` + member.OnlineText + `</span>`
	}
	c.O(`
					</td>
				</tr><tr>
					<td><b>`, c.Txt("96"), `: </b></td>
					<td><a href="`, member.WebsiteURL, `" target="_blank">`, member.WebsiteTitle, `</a></td>
				</tr><tr>
					<td><b>`, c.Txt("113"), ` </b></td>
					<td>
						<i>`, onlineCell, `</i>`)

	// Can they add this member as a buddy?
	if sub.CanHaveBuddy && !page.IsOwner {
		buddyTxt := "buddy_add"
		if member.IsBuddy {
			buddyTxt = "buddy_remove"
		}
		c.O(`
						&nbsp;&nbsp;<a href="`, scripturl, `?action=buddy;u=`, member.ID, `;sesc=`, c.Sc, `">[`, c.Txt(buddyTxt), `]</a>`)
	}

	birthdayCake := ""
	if sub.TodayIsBirthday {
		birthdayCake = ` &nbsp; <img src="` + c.Theme.ImagesURL() + `/bdaycake.gif" width="40" alt="" />`
	}
	c.O(`
					</td>
				</tr><tr>
					<td colspan="2"><hr size="1" width="100%" class="hrcolor" /></td>
				</tr><tr>
					<td><b>`, c.Txt("231"), `: </b></td>
					<td>`, member.GenderName, `</td>
				</tr><tr>
					<td><b>`, c.Txt("420"), `:</b></td>
					<td>`, sub.Age+birthdayCake, `</td>
				</tr><tr>
					<td><b>`, c.Txt("227"), `:</b></td>
					<td>`, member.Location, `</td>
				</tr><tr>
					<td><b>`, c.Txt("local_time"), `:</b></td>
					<td>`, member.LocalTime, `</td>
				</tr><tr>`)

	if !a.SettingEmpty("userLanguage") {
		c.O(`
					<td><b>`, c.Txt("smf225"), `:</b></td>
					<td>`, member.Language, `</td>
				</tr><tr>`)
	}

	c.O(`
					<td colspan="2"><hr size="1" width="100%" class="hrcolor" /></td>
				</tr>`)

	// Show the users signature.
	c.O(`
				<tr>
					<td colspan="2" height="25">
						<table width="100%" cellpadding="0" cellspacing="0" border="0" style="table-layout: fixed;">
							<tr>
								<td style="padding-bottom: 0.5ex;"><b>`, c.Txt("85"), `:</b></td>
							</tr><tr>
								<td colspan="2" width="100%" class="smalltext"><div class="signature">`, member.Signature, `</div></td>
							</tr>
						</table>
					</td>
				</tr>
			</table>
		</td>`)

	// Now print the second column where the members avatar/text is shown.
	c.O(`
		<td class="windowbg" valign="middle" align="center" width="150">
			`, member.AvatarImage, `<br /><br />
			`, member.Blurb, `
		</td>
	</tr>`)

	// Finally, if applicable, span the bottom of the table with links to
	// other useful member functions.
	c.O(`
	<tr class="titlebg">
		<td colspan="2">`, c.Txt("597"), `:</td>
	</tr>
	<tr>
		<td class="windowbg2" colspan="2">`)
	if !page.IsOwner && sub.CanSendPM {
		c.O(`
			<a href="`, scripturl, `?action=pm;sa=send;u=`, member.ID, `">`, c.Txt("688"), `.</a><br />
			<br />`)
	}
	c.O(`
			<a href="`, scripturl, `?action=profile;u=`, member.ID, `;sa=showPosts">`, c.Txt("460"), ` `, c.Txt("461"), `.</a><br />
			<a href="`, scripturl, `?action=profile;u=`, member.ID, `;sa=statPanel">`, c.Txt("statPanel_show"), `.</a><br />
			<br />
		</td>
	</tr>
</table>`)
}

// templateProfileShowPosts is template_showPosts().
func templateProfileShowPosts(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileShowPostsCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<table border="0" width="85%" cellspacing="1" cellpadding="4" class="bordercolor" align="center">
			<tr class="titlebg">
				<td colspan="3" height="26">
					&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;`, c.Txt("showPosts"), `
				</td>
			</tr>`)

	// Only show posts if they have made some!
	if len(sub.Posts) > 0 {
		// Page numbers.
		c.O(`
			<tr class="catbg3">
				<td colspan="3">
					`, c.Txt("139"), `: `, sub.PageIndex, `
				</td>
			</tr>
		</table>`)

		// Button shortcuts
		quoteButton := c.createButton("quote.gif", "145", "smf240", `align="middle"`)
		replyButton := c.createButton("reply_sm.gif", "146", "146", `align="middle"`)
		removeButton := c.createButton("delete.gif", "121", "31", `align="middle"`)
		notifyButton := c.createButton("notify_sm.gif", "131", "125", `align="middle"`)

		// For every post to be displayed, give it its own subtable, and show
		// the important details of the post.
		for _, post := range sub.Posts {
			align := "right"
			if c.RightToLeft {
				align = "left"
			}
			c.O(`
		<table border="0" width="85%" cellspacing="1" cellpadding="0" class="bordercolor" align="center">
			<tr>
				<td width="100%">
					<table border="0" width="100%" cellspacing="0" cellpadding="4" class="bordercolor" align="center">
						<tr class="titlebg2">
							<td style="padding: 0 1ex;">
								`, post.Counter, `
							</td>
							<td width="75%" class="middletext">
								&nbsp;<a href="`, scripturl, `#`, post.CategoryID, `">`, post.CategoryName, `</a> / <a href="`, scripturl, `?board=`, post.BoardID, `.0">`, post.BoardName, `</a> / <a href="`, scripturl, `?topic=`, post.Topic, `.`, post.Start, `#msg`, post.ID, `">`, post.Subject, `</a>
							</td>
							<td class="middletext" align="right" style="padding: 0 1ex; white-space: nowrap;">
								`, c.Txt("30"), `: `, post.Time, `
							</td>
						</tr>
						<tr>
							<td width="100%" height="80" colspan="3" valign="top" class="windowbg2">
								<div class="post">`, post.Body, `</div>
							</td>
						</tr>
						<tr>
							<td colspan="3" class="windowbg2" align="`, align, `"><span class="middletext">`)

			if post.CanDelete {
				c.O(`
					<a href="`, scripturl, `?action=profile;u=`, page.MemID, `;sa=showPosts;start=`, sub.Start, `;delete=`, post.ID, `;sesc=`, c.Sc, `" onclick="return confirm('`, c.Txt("154"), `?');">`, removeButton, `</a>`)
			}
			if post.CanDelete && (post.CanMarkNotify || post.CanReply) {
				c.O(`
								`, c.MenuSeparator)
			}
			if post.CanReply {
				c.O(`
					<a href="`, scripturl, `?action=post;topic=`, post.Topic, `.`, post.Start, `">`, replyButton, `</a>`, c.MenuSeparator, `
					<a href="`, scripturl, `?action=post;topic=`, post.Topic, `.`, post.Start, `;quote=`, post.ID, `;sesc=`, c.Sc, `">`, quoteButton, `</a>`)
			}
			if post.CanReply && post.CanMarkNotify {
				c.O(`
								`, c.MenuSeparator)
			}
			if post.CanMarkNotify {
				c.O(`
					<a href="` + scripturl + `?action=notify;topic=` + itoa(post.Topic) + `.` + post.Start + `">` + notifyButton + `</a>`)
			}

			c.O(`
							</span></td>
						</tr>
					</table>
				</td>
			</tr>
		</table>`)
		}

		// Show more page numbers.
		c.O(`
		<table border="0" width="85%" cellspacing="1" cellpadding="4" class="bordercolor" align="center">
			<tr>
				<td colspan="3" class="catbg3">
					`, c.Txt("139"), `: `, sub.PageIndex, `
				</td>
			</tr>
		</table>`)
	} else {
		// No posts? Just end the table with a informative message.
		c.O(`
			<tr class="windowbg2">
				<td>
					`, c.Txt("170"), `
				</td>
			</tr>
		</table>`)
	}
}

// templateProfileEditBuddies is template_editBuddies().
func templateProfileEditBuddies(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileBuddiesCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<table border="0" width="85%" cellspacing="1" cellpadding="4" class="bordercolor" align="center">
			<tr class="titlebg">
				<td colspan="8" height="26">
					&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;`, c.Txt("editBuddies"), `
				</td>
			</tr>
			<tr class="catbg3">
				<td width="20%">`, c.Txt("68"), `</td>
				<td>`, c.Txt("online8"), `</td>
				<td>`, c.Txt("69"), `</td>
				<td align="center">`, c.Txt("513"), `</td>
				<td align="center">`, c.Txt("603"), `</td>
				<td align="center">`, c.Txt("604"), `</td>
				<td align="center">`, c.Txt("MSN"), `</td>
				<td></td>
			</tr>`)

	// If they don't have any buddies don't list them!
	if len(sub.Buddies) == 0 {
		c.O(`
			<tr class="windowbg">
				<td colspan="8" align="center"><b>`, c.Txt("no_buddies"), `</b></td>
			</tr>`)
	}

	// Now loop through each buddy showing info on each.
	alternate := false
	for _, buddy := range sub.Buddies {
		rowClass := "windowbg2"
		if alternate {
			rowClass = "windowbg"
		}
		emailCell := ""
		if !buddy.HideEmail {
			emailCell = `<a href="mailto:` + buddy.Email + `"><img src="` + c.Theme.ImagesURL() + `/email_sm.gif" alt="` + c.Txt("69") + `" title="` + c.Txt("69") + ` ` + buddy.Name + `" /></a>`
		}
		c.O(`
			<tr class="`, rowClass, `">
				<td>`, buddy.Link, `</td>
				<td align="center"><a href="`, buddy.OnlineHref, `"><img src="`, buddy.OnlineImageHref, `" alt="`, buddy.OnlineLabel, `" title="`, buddy.OnlineLabel, `" /></a></td>
				<td align="center">`, emailCell, `</td>
				<td align="center">`, buddy.ICQLink, `</td>
				<td align="center">`, buddy.AIMLink, `</td>
				<td align="center">`, buddy.YIMLink, `</td>
				<td align="center">`, buddy.MSNLink, `</td>
				<td align="center"><a href="`, scripturl, `?action=profile;u=`, page.MemID, `;sa=editBuddies;remove=`, buddy.ID, `"><img src="`, c.Theme.ImagesURL(), `/icons/delete.gif" alt="`, c.Txt("buddy_remove"), `" title="`, c.Txt("buddy_remove"), `" /></a></td>
			</tr>`)

		alternate = !alternate
	}

	c.O(`
		</table>`)

	// Add a new buddy?
	c.O(`
	<br />
	<form action="`, scripturl, `?action=profile;u=`, page.MemID, `;sa=editBuddies" method="post" accept-charset="`, c.CharacterSet, `">
		<table width="65%" cellpadding="4" cellspacing="0" class="tborder" align="center">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("buddy_add"), `</td>
			</tr>
			<tr class="windowbg">
				<td width="45%">
					<b>`, c.Txt("who_member"), `:</b>
				</td>
				<td width="55%">
					<input type="text" name="new_buddy" id="new_buddy" size="25" />
					<a href="`, scripturl, `?action=findmember;input=new_buddy;quote=1;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, c.Theme.ImagesURL(), `/icons/assist.gif" alt="`, c.Txt("find_members"), `" align="top" /></a>
				</td>
			</tr>
			<tr class="windowbg">
				<td colspan="2" align="right">
					<input type="submit" value="`, c.Txt("buddy_add_button"), `" />
				</td>
			</tr>
		</table>
	</form>`)
}

// templateProfileTrackUser is template_trackUser().
func templateProfileTrackUser(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileTrackCtx)
	scripturl := c.App.ScriptURL

	noneOr := func(list []string) string {
		if len(list) > 0 {
			return strings.Join(list, ", ")
		}
		return "(" + c.Txt("none") + ")"
	}

	// The first table shows IP information about the user.
	c.O(`
		<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
			<table border="0" cellspacing="1" cellpadding="4" align="left" width="100%">
				<tr class="titlebg">
					<td colspan="2">
						<b>`, c.Txt("view_ips_by"), ` `, sub.MemberName, `</b>
					</td>
				</tr>`)

	// The last IP the user used.
	c.O(`
				<tr>
					<td class="windowbg2" align="left" width="200">`, c.Txt("most_recent_ip"), `:</td>
					<td class="windowbg2" align="left">
						<a href="`, scripturl, `?action=trackip;searchip=`, sub.LastIP, `;">`, sub.LastIP, `</a>
					</td>
				</tr>`)

	// Lists of IP addresses used in messages / error messages.
	c.O(`
				<tr>
					<td class="windowbg2" align="left">`, c.Txt("ips_in_messages"), `:</td>
					<td class="windowbg2" align="left">
						`, noneOr(sub.IPs), `
					</td>
				</tr><tr>
					<td class="windowbg2" align="left">`, c.Txt("ips_in_errors"), `:</td>
					<td class="windowbg2" align="left">
						`, noneOr(sub.ErrorIPs), `
					</td>
				</tr>`)

	// List any members that have used the same IP addresses as the current
	// member.
	c.O(`
				<tr>
					<td class="windowbg2" align="left">`, c.Txt("members_in_range"), `:</td>
					<td class="windowbg2" align="left">
						`, noneOr(sub.MembersInRange), `
					</td>
				</tr>
			</table>
		</td></tr></table>
		<br />`)

	// The second table lists all the error messages the user has
	// caused/received.
	c.O(`
		<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
			<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
				<tr class="titlebg">
					<td colspan="4">
						`, c.Txt("errors_by"), ` `, sub.MemberName, `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" colspan="4" style="padding: 2ex;">
						`, c.Txt("errors_desc"), `
					</td>
				</tr><tr class="titlebg">
					<td colspan="4">
						`, c.Txt("139"), `: `, sub.PageIndex, `
					</td>
				</tr><tr class="catbg3">
					<td>`, c.Txt("ip_address"), `</td>
					<td>`, c.Txt("72"), `</td>
					<td>`, c.Txt("317"), `</td>
				</tr>`)

	templateProfileErrorRows(c, sub)

	c.O(`
			</table>
		</td></tr></table>`)
}

// templateProfileErrorRows prints the shared error-message rows.
func templateProfileErrorRows(c *Ctx, sub *ProfileTrackCtx) {
	scripturl := c.App.ScriptURL

	// If there arn't any messages just give a message stating this.
	if len(sub.ErrorMessages) == 0 {
		c.O(`
				<tr><td class="windowbg2" colspan="4"><i>`, c.Txt("no_errors_from_user"), `</i></td></tr>`)
	} else {
		// For every error message print the IP address that caused it, the
		// message displayed and the date it occurred.
		for _, e := range sub.ErrorMessages {
			c.O(`
				<tr>
					<td class="windowbg2">
						<a href="`, scripturl, `?action=trackip;searchip=`, e.IP, `;">`, e.IP, `</a>
					</td>
					<td class="windowbg2">
						`, e.Message, `<br />
						<a href="`, e.URL, `">`, e.URL, `</a>
					</td>
					<td class="windowbg2">`, e.Time, `</td>
				</tr>`)
		}
	}
}

// templateProfileTrackIP is template_trackIP().
func templateProfileTrackIP(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileTrackCtx)
	scripturl := c.App.ScriptURL

	// This function always defaults to the last IP used by a member but can
	// be set to track any IP.
	c.O(`
		<form action="`, scripturl, `?action=trackip" method="post" accept-charset="`, c.CharacterSet, `">`)

	// The first table in the template gives an input box to allow the admin
	// to enter another IP to track.
	c.O(`
			<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
				<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
					<tr class="titlebg">
						<td>`, c.Txt("trackIP"), `</td>
					</tr><tr>
						<td class="windowbg2">
							`, c.Txt("enter_ip"), `:&nbsp;&nbsp;<input type="text" name="searchip" value="`, sub.IP, `" />&nbsp;&nbsp;<input type="submit" value="`, c.Txt("trackIP"), `" />
						</td>
					</tr>
				</table>
			</td></tr></table>
		</form>
		<br />`)

	// The table inbetween the first and second table shows links to the
	// whois server for every region.
	if sub.SingleIP {
		c.O(`
		<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
			<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
				<tr class="titlebg">
					<td colspan="2">
						`, c.Txt("whois_title"), ` `, sub.IP, `
					</td>
				</tr><tr>
					<td class="windowbg2">`)
		for _, server := range sub.WhoisServers {
			bold := ""
			if sub.AutoWhois != nil && sub.AutoWhois.Name == server.Name {
				bold = ` style="font-weight: bold;"`
			}
			c.O(`
						<a href="`, server.URL, `" target="_blank"`, bold, `>`, server.Name, `</a><br />`)
		}
		c.O(`
					</td>
				</tr>
			</table>
		</td></tr></table>
		<br />`)
	}

	// The second table lists all the members who have been logged as using
	// this IP address.
	c.O(`
		<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
			<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
				<tr class="titlebg">
					<td colspan="2">
						`, c.Txt("members_from_ip"), ` `, sub.IP, `
					</td>
				</tr><tr class="catbg3">
					<td>`, c.Txt("ip_address"), `</td>
					<td>`, c.Txt("display_name"), `</td>
				</tr>`)
	if len(sub.IPMembers) == 0 {
		c.O(`
				<tr><td class="windowbg2" colspan="2"><i>`, c.Txt("no_members_from_ip"), `</i></td></tr>`)
	} else {
		// Loop through each of the members and display them.
		for _, entry := range sub.IPMembers {
			c.O(`
				<tr>
					<td class="windowbg2"><a href="`, scripturl, `?action=trackip;searchip=`, entry[0], `;">`, entry[0], `</a></td>
					<td class="windowbg2">`, entry[1], `</td>
				</tr>`)
		}
	}
	c.O(`
			</table>
		</td></tr></table>
		<br />`)

	// The third table in the template displays a list of all the messages
	// sent using this IP (can be quite long).
	c.O(`
		<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
			<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
				<tr class="titlebg">
					<td colspan="4">
						`, c.Txt("messages_from_ip"), ` `, sub.IP, `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" colspan="4" style="padding: 2ex;">
						`, c.Txt("messages_from_ip_desc"), `
					</td>
				</tr><tr class="titlebg">
					<td colspan="4">
						<b>`, c.Txt("139"), `:</b> `, sub.MessagePageIndex, `
					</td>
				</tr><tr class="catbg3">
					<td>`, c.Txt("ip_address"), `</td>
					<td>`, c.Txt("rtm8"), `</td>
					<td>`, c.Txt("319"), `</td>
					<td>`, c.Txt("317"), `</td>
				</tr>`)

	// No message means nothing to do!
	if len(sub.Messages) == 0 {
		c.O(`
				<tr><td class="windowbg2" colspan="4"><i>`, c.Txt("no_messages_from_ip"), `</i></td></tr>`)
	} else {
		// For every message print the IP, member who posts it, subject (with
		// link) and date posted.
		for _, message := range sub.Messages {
			c.O(`
				<tr>
					<td class="windowbg2">
						<a href="`, scripturl, `?action=trackip;searchip=`, message.IP, `">`, message.IP, `</a>
					</td>
					<td class="windowbg2">
						`, message.MemberLink, `
					</td>
					<td class="windowbg2">
						<a href="`, scripturl, `?topic=`, message.Topic, `.msg`, message.ID, `#msg`, message.ID, `">
							`, message.Subject, `
						</a>
					</td>
					<td class="windowbg2">`, message.Time, `</td>
				</tr>`)
		}
	}
	c.O(`
			</table>
		</td></tr></table>
		<br />`)

	// The final table in the template lists all the error messages
	// caused/received by anyone using this IP address.
	c.O(`
		<table cellpadding="0" cellspacing="0" border="0" class="bordercolor" align="center" width="90%"><tr><td>
			<table border="0" cellspacing="1" cellpadding="4" align="center" width="100%">
				<tr class="titlebg">
					<td colspan="4">
						`, c.Txt("errors_from_ip"), ` `, sub.IP, `
					</td>
				</tr><tr class="windowbg">
					<td class="smalltext" colspan="4" style="padding: 2ex;">
						`, c.Txt("errors_from_ip_desc"), `
					</td>
				</tr><tr class="titlebg">
					<td colspan="4">
						`, c.Txt("139"), `: `, sub.ErrorPageIndex, `
					</td>
				</tr><tr class="catbg3">
					<td>`, c.Txt("ip_address"), `</td>
					<td>`, c.Txt("display_name"), `</td>
					<td>`, c.Txt("72"), `</td>
					<td>`, c.Txt("317"), `</td>
				</tr>`)
	if len(sub.ErrorMessages) == 0 {
		c.O(`
				<tr><td class="windowbg2" colspan="4"><i>`, c.Txt("no_errors_from_ip"), `</i></td></tr>`)
	} else {
		// For each error print IP address, member, message received and date
		// caused.
		for _, e := range sub.ErrorMessages {
			c.O(`
				<tr>
					<td class="windowbg2">
						<a href="`, scripturl, `?action=trackip;searchip=`, e.IP, `">`, e.IP, `</a>
					</td>
					<td class="windowbg2">
						`, e.MemberLink, `
					</td>
					<td class="windowbg2">
						`, e.Message, `<br />
						<a href="`, e.URL, `">`, e.URL, `</a>
					</td>
					<td class="windowbg2">`, e.Time, `</td>
				</tr>`)
		}
	}
	c.O(`
			</table>
		</td></tr></table>`)
}

// templateProfileShowPermissions is template_showPermissions().
func templateProfileShowPermissions(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileShowPermsCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<table width="90%" border="0" cellspacing="1" cellpadding="4" align="center" class="bordercolor">
			<tr class="titlebg">
				<td colspan="2" height="26">
					&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;`, c.Txt("showPermissions"), `
					</td>
			</tr>`)
	if sub.HasAllPermissions {
		c.O(`
			<tr class="windowbg2">
				<td colspan="2">`, c.Txt("showPermissions_all"), `</td>
			</tr>`)
	} else {
		if len(sub.NoAccessBoards) > 0 {
			c.O(`
			<tr class="catbg">
				<td align="left" colspan="2">`, c.Txt("showPermissions_restricted_boards"), `</td>
			</tr><tr class="windowbg">
				<td colspan="2" class="smalltext">
					`, c.Txt("showPermissions_restricted_boards_desc"), `:<br />`)
			for _, b := range sub.NoAccessBoards {
				comma := ", "
				if b.IsLast {
					comma = ""
				}
				c.O(`
					<a href="`, scripturl, `?board=`, b.ID, `">`, b.Name, `</a>`, comma)
			}
			c.O(`
				</td>
			</tr>`)
		}

		// General Permissions section.
		c.O(`
			<tr class="catbg">
				<td align="left" colspan="2">`, c.Txt("showPermissions_general"), `</td>
			</tr>`)
		if len(sub.General) > 0 {
			c.O(`
			<tr class="titlebg">
				<td width="50%">`, c.Txt("showPermissions_permission"), `</td>
				<td width="50%"></td>
			</tr>`)

			for _, permission := range sub.General {
				id := permission.ID
				if permission.IsDenied {
					id = "<del>" + permission.ID + "</del>"
				}
				c.O(`
			<tr>
				<td class="windowbg" valign="top">
					`, id, `<br />
					<span class="smalltext">`, permission.Name, `</span>
				</td>
				<td class="windowbg2" valign="top"><span class="smalltext">`)
				if permission.IsDenied {
					c.O(`
					<span style="color: red; font-weight: bold;">`, c.Txt("showPermissions_denied"), `: </span>`, strings.Join(permission.Denied, ", "))
				} else {
					c.O(`
					<span style="font-weight: bold;">`, c.Txt("showPermissions_given"), `: </span>`, strings.Join(permission.Allowed, ", "))
				}
				c.O(`
				</span></td>
			</tr>`)
			}
		} else {
			c.O(`
			<tr class="windowbg2">
				<td colspan="2">`, c.Txt("showPermissions_none_general"), `</td>
			</tr>`)
		}

		// Board permission section.
		globalSel := ""
		if sub.Board == 0 {
			globalSel = ` selected="selected"`
		}
		c.O(`
			<tr class="catbg">
				<td align="left" colspan="2">
					<a name="board_permissions"></a>
					<form action="`+scripturl+`?action=profile;u=`, page.MemID, `;sa=showPermissions#board_permissions" method="post" accept-charset="`, c.CharacterSet, `">
						`, c.Txt("showPermissions_select"), `:
						<select name="board" onchange="if (this.options[this.selectedIndex].value) this.form.submit();">
							<option value="0"`, globalSel, `>`, c.Txt("showPermissions_global"), `</option>`)
		if len(sub.Boards) > 0 {
			c.O(`
							<option value="" disabled="disabled">---------------------------</option>`)
		}

		// Fill the box with any local permission boards.
		for _, board := range sub.Boards {
			sel := ""
			if board.Selected {
				sel = ` selected="selected"`
			}
			c.O(`
							<option value="`, board.ID, `"`, sel, `>`, board.Name, `</option>`)
		}

		c.O(`
						</select>
					</form>
				</td>
			</tr>`)
		if len(sub.BoardPerms) > 0 {
			c.O(`
			<tr class="titlebg">
				<td>`, c.Txt("showPermissions_permission"), `</td>
				<td></td>
			</tr>`)
			for _, permission := range sub.BoardPerms {
				id := permission.ID
				if permission.IsDenied {
					id = "<del>" + permission.ID + "</del>"
				}
				c.O(`
			<tr>
				<td class="windowbg" valign="top">
					`, id, `<br />
					<span class="smalltext">`, permission.Name, `</span>
				</td>
				<td class="windowbg2" valign="top"><span class="smalltext">`)
				if permission.IsDenied {
					c.O(`
					<span style="color: red; font-weight: bold;">`, c.Txt("showPermissions_denied"), `: </span>`, strings.Join(permission.Denied, ", "), `<br />`)
				} else {
					c.O(`
					<span style="font-weight: bold;">`, c.Txt("showPermissions_given"), `: </span>`, strings.Join(permission.Allowed, ", "), `<br />`)
				}
				c.O(`
				</span></td>
			</tr>`)
			}
		} else {
			c.O(`
			<tr class="windowbg2">
				<td colspan="2">`, c.Txt("showPermissions_none_board"), `</td>
			</tr>`)
		}
	}
	c.O(`
		</table><br />`)
}

// templateProfileStatPanel is template_statPanel().
func templateProfileStatPanel(c *Ctx) {
	page := c.Page.(*ProfileCtx)
	sub := page.Sub.(*ProfileStatPanelCtx)

	c.O(`
		<table border="0" width="85%" cellspacing="1" cellpadding="4" class="bordercolor" align="center">
			<tr class="titlebg">
				<td colspan="4" height="26">&nbsp;<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="" align="top" />&nbsp;`, c.Txt("statPanel_generalStats"), ` - `, page.Member.Name, `</td>
			</tr>`)

	// First, show a few text statistics such as post/topic count.
	c.O(`
		<tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, c.Theme.ImagesURL(), `/stats_info.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" valign="top" colspan="3">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">
						<tr>
							<td nowrap="nowrap">`, c.Txt("statPanel_total_time_online"), `:</td>
							<td align="right">`, sub.TimeLoggedIn, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("statPanel_total_posts"), `:</td>
							<td align="right">`, sub.NumPosts, ` `, c.Txt("statPanel_posts"), `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("statPanel_total_topics"), `:</td>
							<td align="right">`, sub.NumTopics, ` `, c.Txt("statPanel_topics"), `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("statPanel_users_polls"), `:</td>
							<td align="right">`, sub.NumPolls, ` `, c.Txt("statPanel_polls"), `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("statPanel_users_votes"), `:</td>
							<td align="right">`, sub.NumVotes, ` `, c.Txt("statPanel_votes"), `</td>
						</tr>
					</table>
				</td>
			</tr>`)

	// This next section draws a graph showing what times of day they post
	// the most.
	c.O(`
			<tr class="titlebg">
				<td colspan="4" width="100%">`, c.Txt("statPanel_activityTime"), `</td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, c.Theme.ImagesURL(), `/stats_views.gif" width="20" height="20" alt="" /></td>
				<td colspan="3" class="windowbg2" width="100%" valign="top">
					<table border="0" cellpadding="0" cellspacing="0" width="100%">`)

	// If they haven't post at all, don't draw the graph.
	if len(sub.PostsByTime) == 0 {
		c.O(`
						<tr>
							<td width="100%" valign="top" align="center">`, c.Txt("statPanel_noPosts"), `</td>
						</tr>`)
	} else {
		c.O(`
						<tr>
							<td width="2%" valign="bottom"></td>`)

		// Loops through each hour drawing the bar to the correct height.
		for _, timeOfDay := range sub.PostsByTime {
			c.O(`
							<td width="4%" valign="bottom" align="center"><img src="`, c.Theme.ImagesURL(), `/bar.gif" width="12" height="`, timeOfDay.PostsPercent, `" alt="" /></td>`)
		}
		c.O(`
							<td width="2%" valign="bottom"></td>
						</tr><tr>
							<td width="2%" valign="bottom"></td>`)
		// The labels.
		for _, timeOfDay := range sub.PostsByTime {
			borderRight := "1px"
			if timeOfDay.Hour == 23 {
				borderRight = "0px"
			}
			c.O(`
							<td width="4%" valign="bottom" align="center" style="border-color: black; border-style: solid; border-width: 1px `, borderRight, ` 0px 0px">`, timeOfDay.Hour, `</td>`)
		}
		c.O(`
							<td width="2%" valign="bottom"></td>
						</tr><tr>
							<td width="100%" colspan="26" align="center"><b>`, c.Txt("statPanel_timeOfDay"), `</b></td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
			</tr>`)

	// The final section is two columns with the most popular boards by posts
	// and activity.
	align := "right"
	if c.RightToLeft {
		align = "left"
	}
	c.O(`
			<tr class="titlebg">
				<td colspan="2" width="50%">`, c.Txt("statPanel_topBoards"), `</td>
				<td colspan="2" width="50%">`, c.Txt("statPanel_topBoardsActivity"), `</td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, c.Theme.ImagesURL(), `/stats_replies.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	if len(sub.PopularBoards) == 0 {
		c.O(`
						<tr>
							<td width="100%" valign="top" align="center">`, c.Txt("statPanel_noPosts"), `</td>
						</tr>`)
	} else {
		// Draw a bar for every board.
		for _, board := range sub.PopularBoards {
			bar := "&nbsp;"
			if board.Posts > 0 {
				bar = `<img src="` + c.Theme.ImagesURL() + `/bar.gif" width="` + board.PostsPercent + `" height="15" alt="" />`
			}
			c.O(`
						<tr>
							<td width="60%" valign="top">`, board.Link, `</td>
							<td width="20%" valign="top">`, bar, `</td>
							<td width="20%" align="`, align, `" valign="top">`, board.Posts, `</td>
						</tr>`)
		}
	}
	c.O(`
					</table>
				</td>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, c.Theme.ImagesURL(), `/stats_replies.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="100%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	if len(sub.BoardActivity) == 0 {
		c.O(`
						<tr>
							<td width="100%" valign="top" align="center">`, c.Txt("statPanel_noPosts"), `</td>
						</tr>`)
	} else {
		// Draw a bar for every board.
		for _, activity := range sub.BoardActivity {
			percent := formatPercentMySQL(activity.Percent)
			bar := "&nbsp;"
			if activity.Percent > 0 {
				bar = `<img src="` + c.Theme.ImagesURL() + `/bar.gif" width="` + percent + `" height="15" alt="" />`
			}
			c.O(`
						<tr>
							<td width="60%" valign="top">`, activity.Link, `</td>
							<td width="20%" valign="top">`, bar, `</td>
							<td width="20%" align="`, align, `" valign="top">`, percent, `%</td>
						</tr>`)
		}
	}
	c.O(`
					</table>
				</td>
			</tr>
		</table>`)
}
