package app

// Hand-port of Themes/default/Display.template.php (template_main).

import "strings"

// templateDisplayMain is template_main() from Display.template.php.
func templateDisplayMain(c *Ctx) {
	page := c.Page.(*DisplayCtx)
	scripturl := c.App.ScriptURL

	// Show the anchor for the top and for the first message.
	c.O(`
<a name="top"></a>
<a name="msg`, page.FirstMessage, `"></a>`)
	if page.FirstNewMessage {
		c.O(`<a name="new"></a>`)
	}

	// Show the linktree
	c.O(`
<div>`)
	c.themeLinktree()
	c.O(`</div>`)

	// Is this topic also a poll?
	if page.IsPoll {
		poll := page.Poll
		pollGif := "normal_poll"
		if poll.IsLocked {
			pollGif = "normal_poll_locked"
		}
		c.O(`
<table cellpadding="3" cellspacing="0" border="0" width="100%" class="tborder" style="padding-top: 0; margin-bottom: 2ex;">
	<tr>
		<td class="titlebg" colspan="2" valign="middle" style="padding-left: 6px;">
			<img src="`, c.Theme.ImagesURL(), `/topic/`, pollGif, `.gif" alt="" align="bottom" /> `, c.Txt("smf43"), `
		</td>
	</tr>
	<tr>
		<td width="5%" valign="top" class="windowbg"><b>`, c.Txt("smf21"), `:</b></td>
		<td class="windowbg">
			`, poll.Question)
		if poll.ExpireTime != "" {
			expTxt := c.Txt("poll_expires_on")
			if poll.IsExpired {
				expTxt = c.Txt("poll_expired_on")
			}
			c.O(`
					&nbsp;(`, expTxt, `: `, poll.ExpireTime, `)`)
		}

		// Are they not allowed to vote but allowed to view the options?
		if poll.ShowResults || !page.AllowVote {
			c.O(`
			<table>
				<tr>
					<td style="padding-top: 2ex;">
						<table border="0" cellpadding="0" cellspacing="0">`)

			// Show each option with its corresponding percentage bar.
			for _, option := range poll.Options {
				bold := ""
				if option.VotedThis {
					bold = "font-weight: bold;"
				}
				c.O(`
							<tr>
								<td style="padding-right: 2ex;`, bold, `">`, option.Option, `</td>`)
				if page.AllowPollView {
					c.O(`
								<td nowrap="nowrap">`, option.Bar, ` `, option.Votes, ` (`, option.Percent, `%)</td>`)
				}
				c.O(`
							</tr>`)
			}

			c.O(`
						</table>
					</td>
					<td valign="bottom" style="padding-left: 15px;">`)

			// If they are allowed to revote - show them a link!
			if page.AllowChangeVote {
				c.O(`
					<a href="`, scripturl, `?action=vote;topic=`, c.Topic, `.`, page.Start, `;poll=`, poll.ID, `;sesc=`, c.Sc, `">`, c.Txt("poll_change_vote"), `</a><br />`)
			}

			// If we're viewing the results... maybe we want to go back and vote?
			if poll.ShowResults && page.AllowVote {
				c.O(`
						<a href="`, scripturl, `?topic=`, c.Topic, `.`, page.Start, `">`, c.Txt("poll_return_vote"), `</a><br />`)
			}

			// If they're allowed to lock the poll, show a link!
			if poll.Lock {
				lockTxt := c.Txt("smf30")
				if poll.IsLocked {
					lockTxt = c.Txt("smf30b")
				}
				c.O(`
						<a href="`, scripturl, `?action=lockVoting;topic=`, c.Topic, `.`, page.Start, `;sesc=`, c.Sc, `">`, lockTxt, `</a><br />`)
			}

			// If they're allowed to edit the poll... show a link!
			if poll.Edit {
				c.O(`
						<a href="`, scripturl, `?action=editpoll;topic=`, c.Topic, `.`, page.Start, `">`, c.Txt("smf39"), `</a>`)
			}

			c.O(`
					</td>
				</tr>`)
			if page.AllowPollView {
				c.O(`
				<tr>
					<td colspan="2"><b>`, c.Txt("smf24"), `: `, poll.TotalVotes, `</b></td>
				</tr>`)
			}
			c.O(`
			</table><br />`)
		} else {
			// They are allowed to vote! Go to it!
			c.O(`
			<form action="`, scripturl, `?action=vote;topic=`, c.Topic, `.`, page.Start, `;poll=`, poll.ID, `" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0px;">
				<table>
					<tr>
						<td colspan="2">`)

			// Show a warning if they are allowed more than one option.
			if poll.AllowedWarning != "" {
				c.O(`
							`, poll.AllowedWarning, `
						</td>
					</tr><tr>
						<td>`)
			}

			// Show each option with its button - a radio likely.
			for _, option := range poll.Options {
				c.O(`
							`, option.VoteButton, ` `, option.Option, `<br />`)
			}

			c.O(`
						</td>
						<td valign="bottom" style="padding-left: 15px;">`)

			// Allowed to view the results? (without voting!)
			if page.AllowPollView {
				c.O(`
							<a href="`, scripturl, `?topic=`, c.Topic, `.`, page.Start, `;viewResults">`, c.Txt("smf29"), `</a><br />`)
			}

			// Show a link for locking the poll as well...
			if poll.Lock {
				lockTxt := c.Txt("smf30")
				if poll.IsLocked {
					lockTxt = c.Txt("smf30b")
				}
				c.O(`
							<a href="`, scripturl, `?action=lockVoting;topic=`, c.Topic, `.`, page.Start, `;sesc=`, c.Sc, `">`, lockTxt, `</a><br />`)
			}

			// Want to edit it? Click right here......
			if poll.Edit {
				c.O(`
							<a href="`, scripturl, `?action=editpoll;topic=`, c.Topic, `.`, page.Start, `">`, c.Txt("smf39"), `</a>`)
			}

			c.O(`
						</td>
					</tr><tr>
						<td colspan="2"><input type="submit" value="`, c.Txt("smf23"), `" /></td>
					</tr>
				</table>
				<input type="hidden" name="sc" value="`, c.Sc, `" />
			</form>`)
		}

		c.O(`
		</td>
	</tr>
</table>`)
	}

	// (linked calendar events arrive with Phase 5's calendar port.)

	// Build the normal button array.
	buildNormalButtons := func(forceCustomPollFirst bool) []stripButton {
		var buttons []stripButton
		if page.CanReply {
			buttons = append(buttons, stripButton{
				URL:  scripturl + "?action=post;topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";num_replies=" + itoa(page.NumReplies),
				Text: "146",
			})
		}
		if page.CanMarkNotify {
			confirmTxt := c.Txt("notification_enable_topic")
			sa := "on"
			if page.IsMarkedNotify {
				confirmTxt = c.Txt("notification_disable_topic")
				sa = "off"
			}
			buttons = append(buttons, stripButton{
				URL:    scripturl + "?action=notify;sa=" + sa + ";topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";sesc=" + c.Sc,
				Text:   "125",
				Custom: `onclick="return confirm('` + confirmTxt + `');"`,
			})
		}
		// Special case for the custom one.
		markUnread := stripButton{
			URL:  scripturl + "?action=markasread;sa=topic;t=" + itoa(page.MarkUnreadTime) + ";topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";sesc=" + c.Sc,
			Text: "mark_unread",
		}
		addPoll := stripButton{
			URL:  scripturl + "?action=editpoll;add;topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";sesc=" + c.Sc,
			Text: "add_poll",
		}
		if forceCustomPollFirst {
			if page.CanAddPoll {
				buttons = append(buttons, addPoll)
			} else if !c.User.IsGuest && !c.Theme.Empty("show_mark_read") {
				buttons = append(buttons, markUnread)
			}
		} else {
			if !c.User.IsGuest && !c.Theme.Empty("show_mark_read") {
				buttons = append(buttons, markUnread)
			} else if page.CanAddPoll {
				buttons = append(buttons, addPoll)
			}
		}
		if page.CanSendTopic {
			buttons = append(buttons, stripButton{
				URL:  scripturl + "?action=sendtopic;topic=" + itoa(c.Topic) + ".0",
				Text: "707",
			})
		}
		buttons = append(buttons, stripButton{
			URL:    scripturl + "?action=printpage;topic=" + itoa(c.Topic) + ".0",
			Text:   "465",
			Custom: `target="_blank"`,
		})
		return buttons
	}

	topBottom := ""
	if !c.App.SettingEmpty("topbottomEnable") {
		topBottom = c.MenuSeparator + ` &nbsp;&nbsp;<a href="#lastPost"><b>` + c.Txt("topbottom5") + `</b></a>`
	}

	// Show the page index... "Pages: [1]".
	c.O(`
<table width="100%" cellpadding="0" cellspacing="0" border="0">
	<tr>
		<td class="middletext" valign="bottom" style="padding-bottom: 4px;">`, c.Txt("139"), `: `, page.PageIndex, topBottom, `</td>
		<td align="right" style="padding-right: 1ex;">
			<div class="nav" style="margin-bottom: 2px;"> `, page.PreviousNext, `</div>
			<table cellpadding="0" cellspacing="0">
				<tr>
					`)
	c.templateButtonStrip(buildNormalButtons(false), "bottom")
	c.O(`
				</tr>
			</table>
		</td>
	</tr>
</table>`)

	// Show the topic information - icon, subject, etc.
	c.O(`
<table width="100%" cellpadding="3" cellspacing="0" border="0" class="tborder" style="border-bottom: 0;">
		<tr class="catbg3">
				<td valign="middle" width="2%" style="padding-left: 6px;">
						<img src="`, c.Theme.ImagesURL(), `/topic/`, page.Class, `.gif" align="bottom" alt="" />
				</td>
				<td width="13%"> `, c.Txt("29"), `</td>
				<td valign="middle" width="85%" style="padding-left: 6px;" id="top_subject">
						`, c.Txt("118"), `: `, page.Subject, ` &nbsp;(`, c.Txt("641"), ` `, page.NumViews, ` `, c.Txt("642"), `)
				</td>
		</tr>`)

	c.O(`
</table>`)

	c.O(`
<form action="`, scripturl, `?action=quickmod2;topic=`, c.Topic, `.`, page.Start, `" method="post" accept-charset="`, c.CharacterSet, `" name="quickModForm" id="quickModForm" style="margin: 0;" onsubmit="return in_edit_mode == 1 ? modify_save('`, c.Sc, `') : confirm('`, c.Txt("quickmod_confirm"), `');">`)

	// These are some cache image buttons we may want.
	replyButton := c.createButton("quote.gif", "145", "smf240", `align="middle"`)
	modifyButton := c.createButton("modify.gif", "66", "17", `align="middle"`)
	removeButton := c.createButton("delete.gif", "121", "31", `align="middle"`)
	splitButton := c.createButton("split.gif", "smf251", "smf251", `align="middle"`)

	// Time to display all the posts
	c.O(`
<table cellpadding="0" cellspacing="0" border="0" width="100%" class="bordercolor">`)

	// Get all the messages...
	for _, message := range page.Messages {
		member := message.Member
		c.O(`
	<tr><td style="padding: 1px 1px 0 1px;">`)

		// Show the message anchor and a "new" anchor if this message is new.
		if message.ID != page.FirstMessage {
			c.O(`
		<a name="msg`, message.ID, `"></a>`)
			if message.FirstNew {
				c.O(`<a name="new"></a>`)
			}
		}

		bg := "windowbg"
		if message.Alternate != 0 {
			bg = "windowbg2"
		}
		c.O(`
		<table width="100%" cellpadding="3" cellspacing="0" border="0">
			<tr><td class="`, bg, `">`)

		// Show information about the poster of this message.
		c.O(`
				<table width="100%" cellpadding="5" cellspacing="0" style="table-layout: fixed;">
					<tr>
						<td valign="top" width="16%" rowspan="2" style="overflow: hidden;">
							<b>`, member.Link, `</b>
							<div class="smalltext">`)

		// Show the member's custom title, if they have one.
		if member.Title != "" {
			c.O(`
								`, member.Title, `<br />`)
		}

		// Show the member's primary group (like 'Administrator') if they have one.
		if member.Group != "" {
			c.O(`
								`, member.Group, `<br />`)
		}

		// Don't show these things for guests.
		if !member.IsGuest {
			// Show the post group if they have no other group, or the option.
			if (c.Theme.Empty("hide_post_group") || member.Group == "") && member.PostGroup != "" {
				c.O(`
								`, member.PostGroup, `<br />`)
			}
			c.O(`
								`, member.GroupStars, `<br />`)

			// Is karma display enabled?  Total or +/-?
			if c.App.Setting("karmaMode") == "1" {
				c.O(`
								<br />
								`, c.App.Setting("karmaLabel"), ` `, member.KarmaGood-member.KarmaBad, `<br />`)
			} else if c.App.Setting("karmaMode") == "2" {
				c.O(`
								<br />
								`, c.App.Setting("karmaLabel"), ` +`, member.KarmaGood, `/-`, member.KarmaBad, `<br />`)
			}

			// Is this user allowed to modify this member's karma?
			if member.KarmaAllow {
				c.O(`
								<a href="`, scripturl, `?action=modifykarma;sa=applaud;uid=`, member.ID, `;topic=`, c.Topic, `.`, page.Start, `;m=`, message.ID, `;sesc=`, c.Sc, `">`, c.App.Setting("karmaApplaudLabel"), `</a>
								<a href="`, scripturl, `?action=modifykarma;sa=smite;uid=`, member.ID, `;topic=`, c.Topic, `.`, page.Start, `;m=`, message.ID, `;sesc=`, c.Sc, `">`, c.App.Setting("karmaSmiteLabel"), `</a><br />`)
			}

			// Show online and offline buttons?
			if !c.App.SettingEmpty("onlineEnable") {
				if page.CanSendPM {
					c.O(`
								<a href="`, member.OnlineHref, `" title="`, member.OnlineLabel, `">`)
				} else {
					c.O(`
								`)
				}
				if !c.Theme.Empty("use_image_buttons") {
					c.O(`<img src="`, member.OnlineImageHref, `" alt="`, member.OnlineText, `" border="0" style="margin-top: 2px;" />`)
				} else {
					c.O(member.OnlineText)
				}
				if page.CanSendPM {
					c.O(`</a>`)
				}
				if !c.Theme.Empty("use_image_buttons") {
					c.O(`<span class="smalltext"> `, member.OnlineText, `</span>`)
				}
				c.O(`<br /><br />`)
			}

			// Show the member's gender icon?
			if !c.Theme.Empty("show_gender") && member.GenderImage != "" {
				c.O(`
								`, c.Txt("231"), `: `, member.GenderImage, `<br />`)
			}

			// Show how many posts they have made.
			c.O(`
								`, c.Txt("26"), `: `, member.Posts, `<br />
								<br />`)

			// Show avatars, images, etc.?
			if !c.Theme.Empty("show_user_images") && empty(c.Options["show_no_avatars"]) && member.AvatarImage != "" {
				c.O(`
								<div style="overflow: auto; width: 100%;">`, member.AvatarImage, `</div><br />`)
			}

			// Show their personal text?
			if !c.Theme.Empty("show_blurb") && member.Blurb != "" {
				c.O(`
								`, member.Blurb, `<br />
								<br />`)
			}

			// This shows the popular messaging icons.
			c.O(`
								`, member.ICQLink, `
								`, member.MSNLink, `
								`, member.AIMLink, `
								`, member.YIMLink, `<br />`)

			// Show the profile, website, email address, and PM buttons.
			if !c.Theme.Empty("show_profile_buttons") {
				// Don't show the profile button if you're not allowed to view
				// the profile.
				if member.CanViewProfile {
					c.O(`
								<a href="`, member.Href, `">`)
					if !c.Theme.Empty("use_image_buttons") {
						c.O(`<img src="`, c.Theme.ImagesURL(), `/icons/profile_sm.gif" alt="`, c.Txt("27"), `" title="`, c.Txt("27"), `" border="0" />`)
					} else {
						c.O(c.Txt("27"))
					}
					c.O(`</a>`)
				}

				// Don't show an icon if they haven't specified a website.
				if member.WebsiteURL != "" {
					c.O(`
								<a href="`, member.WebsiteURL, `" title="`, member.WebsiteTitle, `" target="_blank">`)
					if !c.Theme.Empty("use_image_buttons") {
						c.O(`<img src="`, c.Theme.ImagesURL(), `/www_sm.gif" alt="`, c.Txt("515"), `" border="0" />`)
					} else {
						c.O(c.Txt("515"))
					}
					c.O(`</a>`)
				}

				// Don't show the email address if they want it hidden.
				if !member.HideEmail {
					c.O(`
								<a href="mailto:`, member.Email, `">`)
					if !c.Theme.Empty("use_image_buttons") {
						c.O(`<img src="`, c.Theme.ImagesURL(), `/email_sm.gif" alt="`, c.Txt("69"), `" title="`, c.Txt("69"), `" border="0" />`)
					} else {
						c.O(c.Txt("69"))
					}
					c.O(`</a>`)
				}

				// Since we know this person isn't a guest, you *can* message
				// them.
				if page.CanSendPM {
					imGif := "im_off"
					if member.IsOnline {
						imGif = "im_on"
					}
					c.O(`
								<a href="`, scripturl, `?action=pm;sa=send;u=`, member.ID, `" title="`, member.OnlineLabel, `">`)
					if !c.Theme.Empty("use_image_buttons") {
						c.O(`<img src="`, c.Theme.ImagesURL(), `/`, imGif, `.gif" alt="`, member.OnlineLabel, `" border="0" />`)
					} else {
						c.O(member.OnlineLabel)
					}
					c.O(`</a>`)
				}
			}
		} else if !member.HideEmail {
			// Otherwise, show the guest's email.
			c.O(`
								<br />
								<br />
								<a href="mailto:`, member.Email, `">`)
			if !c.Theme.Empty("use_image_buttons") {
				c.O(`<img src="`, c.Theme.ImagesURL(), `/email_sm.gif" alt="`, c.Txt("69"), `" title="`, c.Txt("69"), `" border="0" />`)
			} else {
				c.O(c.Txt("69"))
			}
			c.O(`</a>`)
		}

		// Done with the information about the poster... on to the post itself.
		c.O(`
							</div>
						</td>
						<td valign="top" width="85%" height="100%">
							<table width="100%" border="0"><tr>
								<td valign="middle"><a href="`, message.Href, `"><img src="`, message.IconURL, `" alt="" border="0" /></a></td>
								<td valign="middle">
									<div style="font-weight: bold;" id="subject_`, message.ID, `">
										<a href="`, message.Href, `">`, message.Subject, `</a>
									</div>`)

		// If this is the first post, (#0) just say when it was posted.
		counterPart := ""
		if message.Counter != 0 {
			counterPart = c.Txt("146") + " #" + itoa(message.Counter)
		}
		align := "right"
		if c.RightToLeft {
			align = "left"
		}
		c.O(`
									<div class="smalltext">&#171; <b>`, counterPart, ` `, c.Txt("30"), `:</b> `, message.Time, ` &#187;</div></td>
								<td align="`, align, `" valign="bottom" height="20" style="font-size: smaller;">`)

		// Can they reply?
		if page.CanReply && !empty(c.Options["display_quick_reply"]) {
			c.O(`
					<a href="`, scripturl, `?action=post;quote=`, message.ID, `;topic=`, c.Topic, `.`, page.Start, `;num_replies=`, page.NumReplies, `;sesc=`, c.Sc, `" onclick="doQuote(`, message.ID, `, '`, c.Sc, `'); return false;">`, replyButton, `</a>`)
		} else if page.CanReply {
			c.O(`
					<a href="`, scripturl, `?action=post;quote=`, message.ID, `;topic=`, c.Topic, `.`, page.Start, `;num_replies=`, page.NumReplies, `;sesc=`, c.Sc, `">`, replyButton, `</a>`)
		}

		// Can the user modify the contents of this post?
		if message.CanModify {
			c.O(`
					<a href="`, scripturl, `?action=post;msg=`, message.ID, `;topic=`, c.Topic, `.`, page.Start, `;sesc=`, c.Sc, `">`, modifyButton, `</a>`)
		}

		// How about... even... remove it entirely?!
		if message.CanRemove {
			c.O(`
					<a href="`, scripturl, `?action=deletemsg;topic=`, c.Topic, `.`, page.Start, `;msg=`, message.ID, `;sesc=`, c.Sc, `" onclick="return confirm('`, c.Txt("154"), `?');">`, removeButton, `</a>`)
		}

		// What about splitting it off the rest of the topic?
		if page.CanSplit {
			c.O(`
					<a href="`, scripturl, `?action=splittopics;topic=`, c.Topic, `.0;at=`, message.ID, `">`, splitButton, `</a>`)
		}

		// Show the post itself, finally!
		c.O(`
								</td>
							</tr></table>
							<hr width="100%" size="1" class="hrcolor" />
							<div class="post"`)
		if message.CanModify {
			c.O(` id="msg_`, message.ID, `"`)
		}
		c.O(`>`, message.Body, `</div>`)
		if message.CanModify {
			c.O(`
							<img src="`, c.Theme.ImagesURL(), `/icons/modify_inline.gif" alt="" align="right" id="modify_button_`, message.ID, `" style="cursor: pointer; display: none;" onclick="modify_msg('`, message.ID, `', '`, c.Sc, `')" />`)
		}
		c.O(`
						</td>
					</tr>`)

		// Now for the attachments, signature, ip logged, etc...
		c.O(`
					<tr>
						<td valign="bottom" class="smalltext" width="85%">
							<table width="100%" border="0" style="table-layout: fixed;"><tr>
								<td colspan="2" class="smalltext" width="100%">`)

		// Assuming there are attachments...
		if len(message.Attachments) > 0 {
			c.O(`
									<hr width="100%" size="1" class="hrcolor" />
									<div style="overflow: auto; width: 100%;">`)
			for _, attachment := range message.Attachments {
				if attachment.IsImage {
					if attachment.HasThumb {
						c.O(`
									<a href="`, attachment.Href, `;image" id="link_`, attachment.ID, `" onclick="`, attachment.ThumbJS, `"><img src="`, attachment.ThumbHref, `" alt="" id="thumb_`, attachment.ID, `" border="0" /></a><br />`)
					} else {
						c.O(`
									<img src="`, attachment.Href, `;image" alt="" width="`, attachment.Width, `" height="`, attachment.Height, `" border="0" /><br />`)
					}
				}
				sizePart := ""
				if attachment.IsImage {
					sizePart = ", " + itoa(attachment.RealWidth) + "x" + itoa(attachment.RealHeight) + " - " + c.Txt("attach_viewed")
				} else {
					sizePart = " - " + c.Txt("attach_downloaded")
				}
				c.O(`
										<a href="`, attachment.Href, `"><img src="`, c.Theme.ImagesURL(), `/icons/clip.gif" align="middle" alt="*" border="0" />&nbsp;`, attachment.Name, `</a> (`, attachment.Size, sizePart, ` `, attachment.Downloads, ` `, c.Txt("attach_times"), `.)<br />`)
			}

			c.O(`
									</div>`)
		}

		c.O(`
								</td>
							</tr><tr>
								<td valign="bottom" class="smalltext" id="modified_`, message.ID, `">`)

		// Show "« Last Edit: Time by Person »" if this post was edited.
		if !c.Theme.Empty("show_modify") && message.ModifiedName != "" {
			c.O(`
									&#171; <i>`, c.Txt("211"), `: `, message.ModifiedTime, ` `, c.Txt("525"), ` `, message.ModifiedName, `</i> &#187;`)
		}

		align = "right"
		if c.RightToLeft {
			align = "left"
		}
		c.O(`
								</td>
								<td align="`, align, `" valign="bottom" class="smalltext">`)

		// Maybe they want to report this post to the moderator(s)?
		if page.CanReportModerator {
			c.O(`
									<a href="`, scripturl, `?action=reporttm;topic=`, c.Topic, `.`, message.Counter, `;msg=`, message.ID, `">`, c.Txt("rtm1"), `</a> &nbsp;`)
		}
		c.O(`
									<img src="`, c.Theme.ImagesURL(), `/ip.gif" alt="" border="0" />`)

		// Show the IP to this user for this post - because you can moderate?
		if page.CanModerateForum && member.IP != "" {
			c.O(`
									<a href="`, scripturl, `?action=trackip;searchip=`, member.IP, `">`, member.IP, `</a> <a href="`, scripturl, `?action=helpadmin;help=see_admin_ip" onclick="return reqWin(this.href);" class="help">(?)</a>`)
		} else if message.CanSeeIP {
			c.O(`
									<a href="`, scripturl, `?action=helpadmin;help=see_member_ip" onclick="return reqWin(this.href);" class="help">`, member.IP, `</a>`)
		} else if !c.User.IsGuest {
			c.O(`
									<a href="`, scripturl, `?action=helpadmin;help=see_member_ip" onclick="return reqWin(this.href);" class="help">`, c.Txt("511"), `</a>`)
		} else {
			c.O(`
									`, c.Txt("511"))
		}

		c.O(`
								</td>
							</tr></table>`)

		// Show the member's signature?
		if member.Signature != "" && empty(c.Options["show_no_signatures"]) {
			c.O(`
							<hr width="100%" size="1" class="hrcolor" />
							<div class="signature">`, member.Signature, `</div>`)
		}

		c.O(`
						</td>
					</tr>
				</table>
			</td></tr>
		</table>
	</td></tr>`)
	}
	c.O(`
	<tr><td style="padding: 0 0 1px 0;"></td></tr>
</table>
<a name="lastPost"></a>`)

	topBottomTop := ""
	if !c.App.SettingEmpty("topbottomEnable") {
		topBottomTop = c.MenuSeparator + ` &nbsp;&nbsp;<a href="#top"><b>` + c.Txt("topbottom4") + `</b></a>`
	}
	c.O(`
<table width="100%" cellpadding="0" cellspacing="0" border="0">
	<tr>
		<td class="middletext">`, c.Txt("139"), `: `, page.PageIndex, topBottomTop, `</td>
		<td align="right" style="padding-right: 1ex;">
			<table cellpadding="0" cellspacing="0">
				<tr>
					`)
	c.templateButtonStrip(buildNormalButtons(true), "top")
	c.O(`
				</tr>
			</table>
		</td>
	</tr>
</table>`)

	quickReplyCollapsed := "true"
	if c.Options["display_quick_reply"] == "2" {
		quickReplyCollapsed = "false"
	}
	showModify := "0"
	if !c.Theme.Empty("show_modify") {
		showModify = "1"
	}
	c.O(`
<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/xml_topic.js"></script>
<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
	quickReplyCollapsed = `, quickReplyCollapsed, `;

	smf_topic = `, c.Topic, `;
	smf_start = `, page.Start, `;
	smf_show_modify = `, showModify, `;

	// On quick modify, this is what the body will look like.
	var smf_template_body_edit = '<div id="error_box" style="padding: 4px; color: red;"></div><textarea class="editor" name="message" rows="12" style="width: 94%; margin-bottom: 10px;">%body%</textarea><br /><input type="hidden" name="sc" value="`, c.Sc, `" /><input type="hidden" name="topic" value="`, c.Topic, `" /><input type="hidden" name="msg" value="%msg_id%" /><div style="text-align: center;"><input type="submit" name="post" value="`, c.Txt("10"), `" onclick="return modify_save(\'`, c.Sc, `\');" accesskey="s" />&nbsp;&nbsp;<input type="submit" name="cancel" value="`, c.Txt("modify_cancel"), `" onclick="return modify_cancel();" /></div>';

	// And this is the replacement for the subject.
	var smf_template_subject_edit = '<input type="text" name="subject" value="%subject%" size="60" style="width: 99%;"  maxlength="80" />';

	// Restore the message to this after editing.
	var smf_template_body_normal = '%body%';
	var smf_template_subject_normal = '<a href="`, scripturl, `?topic=`, c.Topic, `.msg%msg_id%#msg%msg_id%">%subject%</a>';
	var smf_template_top_subject = "`, c.Txt("118"), `: %subject% &nbsp;(`, c.Txt("641"), ` `, page.NumViews, ` `, c.Txt("642"), `)"

	if (window.XMLHttpRequest)
		showModifyButtons();
// ]]></script>
<table border="0" width="100%" cellpadding="0" cellspacing="0" style="margin-bottom: 1ex;">
		<tr>`)
	if !c.Theme.Empty("linktree_inline") {
		c.O(`
				<td valign="top">`)
		c.themeLinktree()
		c.O(`</td> `)
	}
	align := "right"
	if c.RightToLeft {
		align = "left"
	}
	c.O(`
				<td valign="top" align="`, align, `" class="nav"> `, page.PreviousNext, `</td>
		</tr>
</table>`)

	var modButtons []stripButton
	if page.CanMove {
		modButtons = append(modButtons, stripButton{URL: scripturl + "?action=movetopic;topic=" + itoa(c.Topic) + ".0", Text: "132"})
	}
	if page.CanDelete {
		modButtons = append(modButtons, stripButton{
			URL:    scripturl + "?action=removetopic2;topic=" + itoa(c.Topic) + ".0;sesc=" + c.Sc,
			Text:   "63",
			Custom: `onclick="return confirm('` + c.Txt("162") + `');"`,
		})
	}
	if page.CanLock {
		lockText := "smf279"
		if page.IsLocked {
			lockText = "smf280"
		}
		modButtons = append(modButtons, stripButton{URL: scripturl + "?action=lock;topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";sesc=" + c.Sc, Text: lockText})
	}
	if page.CanSticky {
		stickyText := "smf277"
		if page.IsSticky {
			stickyText = "smf278"
		}
		modButtons = append(modButtons, stripButton{URL: scripturl + "?action=sticky;topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";sesc=" + c.Sc, Text: stickyText})
	}
	if page.CanMerge {
		modButtons = append(modButtons, stripButton{URL: scripturl + "?action=mergetopics;board=" + itoa(c.Board) + ".0;from=" + itoa(c.Topic), Text: "smf252"})
	}
	if page.CanRemovePoll {
		modButtons = append(modButtons, stripButton{
			URL:    scripturl + "?action=removepoll;topic=" + itoa(c.Topic) + "." + itoa(page.Start) + ";sesc=" + c.Sc,
			Text:   "poll_remove",
			Custom: `onclick="return confirm('` + c.Txt("poll_remove_warn") + `');"`,
		})
	}
	if page.CalendarPost {
		modButtons = append(modButtons, stripButton{URL: scripturl + "?action=post;calendar;msg=" + itoa(page.TopicFirstMessage) + ";topic=" + itoa(c.Topic) + ".0;sesc=" + c.Sc, Text: "calendar37"})
	}

	c.O(`
	<table cellpadding="0" cellspacing="0" border="0" style="margin-left: 1ex;">
		<tr>
			`)
	c.templateButtonStrip(modButtons, "bottom")
	c.O(`
		</tr>
	</table>`)

	c.O(`
</form>`)

	align = "right"
	if c.RightToLeft {
		align = "left"
	}
	c.O(`
<div class="tborder"><div class="titlebg2" style="padding: 4px;" align="`, align, `">
	<form action="`, scripturl, `" method="get" accept-charset="`, c.CharacterSet, `" style="padding:0; margin: 0;">
		<span class="smalltext">`, c.Txt("160"), `:</span>
		<select name="jumpto" id="jumpto" onchange="if (this.selectedIndex > 0 &amp;&amp; this.options[this.selectedIndex].value) window.location.href = smf_scripturl + this.options[this.selectedIndex].value.substr(smf_scripturl.indexOf('?') == -1 || this.options[this.selectedIndex].value.substr(0, 1) != '?' ? 0 : 1);">
			<option value="">`, c.Txt("251"), `:</option>`)
	for _, category := range c.JumpTo {
		c.O(`
			<option value="" disabled="disabled">-----------------------------</option>
			<option value="#`, category.ID, `">`, category.Name, `</option>
			<option value="" disabled="disabled">-----------------------------</option>`)
		for _, board := range category.Boards {
			selected := ""
			if board.IsCurrent {
				selected = ` selected="selected"`
			}
			c.O(`
			<option value="?board=`, board.ID, `.0"`, selected, `> `, strings.Repeat("==", board.ChildLevel), `=> `, board.Name, `</option>`)
		}
	}
	c.O(`
		</select>&nbsp;
		<input type="button" value="`, c.Txt("161"), `" onclick="if (this.form.jumpto.options[this.form.jumpto.selectedIndex].value) window.location.href = '`, scripturl, `' + this.form.jumpto.options[this.form.jumpto.selectedIndex].value;" />
	</form>
</div></div>`)

	c.O(`<br />`)

	// (Quick reply [display_quick_reply theme option] ports with Phase 3's
	// posting pipeline — needs checkSubmitOnce/form_sequence_number.)
}
