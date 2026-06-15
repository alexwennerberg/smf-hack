package app

// Hand-port of Themes/default/PersonalMessage.template.php: the pm_above /
// pm_below layers and the folder, send, ask_delete, prune, labels and
// report_message(_complete) sub-templates. (template_search /
// template_search_results port with Phase 5's PM search work.)

import "strings"

func init() {
	layerFuncs["pm_above"] = templatePMAbove
	layerFuncs["pm_below"] = templatePMBelow
}

// templatePMAbove is template_pm_above(): the PM sidebar.
func templatePMAbove(c *Ctx) {
	page := c.Page.(*PMCtx)

	c.O(`
		<div style="padding: 3px;">`)
	c.themeLinktree()
	c.O(`</div>`)

	c.O(`
			<table width="100%" border="0" cellpadding="0" cellspacing="0"><tr>
				<td width="125" valign="top">
					<table border="0" cellpadding="4" cellspacing="1" class="bordercolor" width="100">`)

	// Loop through every main area - giving a nice section heading.
	for _, section := range page.Areas {
		c.O(`
						<tr>
							<td class="catbg">`, section.Title, `</td>
						</tr>
						<tr class="windowbg2">
							<td class="smalltext" style="padding-bottom: 2ex;">`)

		// Each sub area.
		for _, area := range section.Areas {
			if area.IsSpacer {
				c.O(`<br />`)
			} else if area.IsLimitBar {
				// !!! Hardcoded colors = bad.
				color := "#468008"
				if page.LimitBarPercent > 85 {
					color = "#A53D05"
				} else if page.LimitBarPercent > 40 {
					color = "#EEA800"
				}
				redStyle := ""
				if page.LimitBarPercent > 90 {
					redStyle = ` style="color: red;"`
				}
				c.O(`
								<br /><br />
								<div align="center">
									<b>`, c.Txt("pm_capacity"), `</b>
									<div align="left" style="border: 1px solid black; height: 7px; width: 100px;">
										<div style="border: 0; background-color: `, color, `; height: 7px; width: `, page.LimitBar, `px;"></div>
									</div>
									<span`, redStyle, `>`, page.LimitBarText, `</span>
								</div>
								<br />`)
			} else {
				unread := ""
				if area.UnreadMessages != nil && *area.UnreadMessages != 0 {
					unread = " (<b>" + itoa(*area.UnreadMessages) + "</b>)"
				}
				if area.Key == page.Area {
					c.O(`
								<b>`, area.Link, unread, `</b><br />`)
				} else {
					c.O(`
								`, area.Link, unread, `<br />`)
				}
			}
		}
		c.O(`
							</td>
						</tr>`)
	}
	c.O(`
					</table>
					<br />
				</td>
				<td valign="top">`)
}

// templatePMBelow is template_pm_below().
func templatePMBelow(c *Ctx) {
	c.O(`
				</td>
			</tr></table>`)
}

// templatePMFolder is template_folder(): an inbox/outbox listing.
func templatePMFolder(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()

	lbl := ""
	if page.CurrentLabelID != -1 {
		lbl = ";l=" + itoa(page.CurrentLabelID)
	}

	sortImg := func(col string) string {
		if page.SortBy == col {
			return ` <img src="` + images + `/sort_` + page.SortDirection + `.gif" alt="" />`
		}
		return ""
	}
	descLink := func(col string) string {
		if page.SortBy == col && page.SortDirection == "up" {
			return ";desc"
		}
		return ""
	}
	nameHdr := c.Txt("318")
	if page.FromOrTo != "from" {
		nameHdr = c.Txt("324")
	}

	c.O(`
<form action="`, scripturl, `?action=pm;sa=pmactions;f=`, page.Folder, `;start=`, page.Start, lbl, `" method="post" accept-charset="`, c.CharacterSet, `" name="pmFolder">
	<table border="0" width="100%" cellpadding="2" cellspacing="1" class="bordercolor">
		<tr class="titlebg">
			<td align="center" width="2%">&nbsp;</td>
			<td style="width: 32ex;"><a href="`, scripturl, `?action=pm;f=`, page.Folder, `;start=`, page.Start, `;sort=date`, descLink("date"), `;`, lbl, `">`, c.Txt("317"), sortImg("date"), `</a></td>
			<td width="46%"><a href="`, scripturl, `?action=pm;f=`, page.Folder, `;start=`, page.Start, `;sort=subject`, descLink("subject"), `;`, lbl, `">`, c.Txt("319"), sortImg("subject"), `</a></td>
			<td><a href="`, scripturl, `?action=pm;f=`, page.Folder, `;start=`, page.Start, `;sort=name`, descLink("name"), `;`, lbl, `">`, nameHdr, sortImg("name"), `</a></td>
			<td align="center" width="24"><input type="checkbox" onclick="invertAll(this, this.form);" class="check" /></td>
		</tr>`)
	if !page.ShowDelete {
		c.O(`
		<tr>
			<td class="windowbg" colspan="5">`, c.Txt("151"), `</td>
		</tr>`)
	}

	for _, message := range page.Messages {
		bg := "windowbg"
		if message.Alternate {
			bg = "windowbg2"
		}
		repliedImg := `<img src="` + images + `/icons/pm_read.gif" style="margin-right: 4px;" alt="` + c.Txt("pm_read") + `" />`
		if message.IsRepliedTo {
			repliedImg = `<img src="` + images + `/icons/pm_replied.gif" style="margin-right: 4px;" alt="` + c.Txt("pm_replied") + `" />`
		}
		nameCell := message.Member.Link
		if page.FromOrTo != "from" {
			nameCell = ""
			if len(message.RecipientsTo) > 0 {
				nameCell = strings.Join(message.RecipientsTo, ", ")
			}
		}
		selected := ""
		if message.IsSelected {
			selected = ` checked="checked"`
		}
		c.O(`
		<tr class="`, bg, `">
			<td align="center" width="2%">`, repliedImg, `</td>
			<td>`, message.Time, `</td>
			<td><a href="#msg`, message.ID, `">`, message.Subject, `</a></td>
			<td>`, nameCell, `</td>
			<td align="center"><input type="checkbox" name="pms[]" id="deletelisting`, message.ID, `" value="`, message.ID, `"`, selected, ` onclick="document.getElementById('deletedisplay`, message.ID, `').checked = this.checked;" class="check" /></td>
		</tr>`)
	}

	sizeFix := ""
	if c.Browser.NeedsSizeFix && !c.Browser.IsIE6 {
		sizeFix = "width: 100%;"
	}
	c.O(`
	</table>
	<div class="bordercolor" style="padding: 1px; `, sizeFix, `">
		<table width="100%" cellpadding="2" cellspacing="0" border="0"><tr class="catbg" valign="middle">
			<td>
				<div style="float: left;">`, c.Txt("139"), `: `, page.PageIndex, `</div>
				<div style="float: right;">&nbsp;`)

	if page.ShowDelete {
		if page.UsingLabels && page.Folder != "outbox" {
			c.O(`
				<select name="pm_action" onchange="if (this.options[this.selectedIndex].value) this.form.submit();" onfocus="loadLabelChoices();">
					<option value="">`, c.Txt("pm_sel_label_title"), `:</option>
					<option value="" disabled="disabled">---------------</option>`)

			c.O(`
									<option value="" disabled="disabled">`, c.Txt("pm_msg_label_apply"), `:</option>`)
			for _, id := range page.LabelOrder {
				label := page.Labels[id]
				if label.ID != page.CurrentLabelID {
					c.O(`
					<option value="add_`, label.ID, `">&nbsp;`, label.Name, `</option>`)
				}
			}
			c.O(`
					<option value="" disabled="disabled">`, c.Txt("pm_msg_label_remove"), `:</option>`)
			for _, id := range page.LabelOrder {
				label := page.Labels[id]
				c.O(`
					<option value="rem_`, label.ID, `">&nbsp;`, label.Name, `</option>`)
			}
			c.O(`
				</select>
				<noscript>
					<input type="submit" value="`, c.Txt("pm_apply"), `" />
				</noscript>`)
		}

		c.O(`
				<input type="submit" name="del_selected" value="`, c.Txt("quickmod_delete_selected"), `" style="font-weight: normal;" onclick="if (!confirm('`, c.Txt("smf249"), `')) return false;" />`)
	}

	c.O(`
				<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
					var allLabels = {};
					function loadLabelChoices()
					{
						var listing = document.forms.pmFolder.elements;
						var theSelect = document.forms.pmFolder.pm_action;
						var add, remove, toAdd = {length: 0}, toRemove = {length: 0};

						if (theSelect.childNodes.length == 0)
							return;`)

	// This is done this way for internationalization reasons.
	c.O(`
						if (typeof(allLabels[-1]) == "undefined")
						{
							for (var o = 0; o < theSelect.options.length; o++)
								if (theSelect.options[o].value.substr(0, 4) == "rem_")
									allLabels[theSelect.options[o].value.substr(4)] = theSelect.options[o].text;
						}

						for (var i = 0; i < listing.length; i++)
						{
							if (listing[i].name != "pms[]" || !listing[i].checked)
								continue;

							var alreadyThere = [], x;
							for (x in currentLabels[listing[i].value])
							{
								if (typeof(toRemove[x]) == "undefined")
								{
									toRemove[x] = allLabels[x];
									toRemove.length++;
								}
								alreadyThere[x] = allLabels[x];
							}

							for (x in allLabels)
							{
								if (typeof(alreadyThere[x]) == "undefined")
								{
									toAdd[x] = allLabels[x];
									toAdd.length++;
								}
							}
						}

						while (theSelect.options.length > 2)
							theSelect.options[2] = null;

						if (toAdd.length != 0)
						{
							theSelect.options[theSelect.options.length] = new Option("`, c.Txt("pm_msg_label_apply"), `", "");
							setInnerHTML(theSelect.options[theSelect.options.length - 1], "`, c.Txt("pm_msg_label_apply"), `");
							theSelect.options[theSelect.options.length - 1].disabled = true;

							for (i in toAdd)
							{
								if (i != "length")
									theSelect.options[theSelect.options.length] = new Option(toAdd[i], "add_" + i);
							}
						}

						if (toRemove.length != 0)
						{
							theSelect.options[theSelect.options.length] = new Option("`, c.Txt("pm_msg_label_remove"), `", "");
							setInnerHTML(theSelect.options[theSelect.options.length - 1], "`, c.Txt("pm_msg_label_remove"), `");
							theSelect.options[theSelect.options.length - 1].disabled = true;

							for (i in toRemove)
							{
								if (i != "length")
									theSelect.options[theSelect.options.length] = new Option(toRemove[i], "rem_" + i);
							}
						}
					}
				// ]]></script>`)

	c.O(`
				</div>
			</td>
		</tr></table>
	</div><br />

	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		var currentLabels = {};
	// ]]></script>`)

	if len(page.Messages) > 0 {
		c.O(`
		<table cellpadding="4" cellspacing="0" border="0" width="100%" class="bordercolor">
			<tr class="titlebg">
				<td width="16%">&nbsp;`, c.Txt("29"), `</td>
				<td>`, c.Txt("118"), `</td>
			</tr>
		</table>
		<table cellpadding="0" cellspacing="0" border="0" width="100%" class="bordercolor">`)

		// Cache some handy buttons.
		quoteButton := c.createButton("quote.gif", "145", "smf240", `align="middle"`)
		replyButton := c.createButton("im_reply.gif", "146", "146", `align="middle"`)
		replyAllButton := c.createButton("im_reply_all.gif", "reply_to_all", "reply_to_all", `align="middle"`)
		forwardButton := c.createButton("quote.gif", "145", "145", `align="middle"`)
		deleteButton := c.createButton("delete.gif", "154", "31", `align="middle"`)

		for _, message := range page.Messages {
			member := message.Member
			windowcss := "windowbg"
			if message.Alternate {
				windowcss = "windowbg2"
			}

			c.O(`
		<tr><td style="padding: 1px 1px 0 1px;">
			<a name="msg`, message.ID, `"></a>
			<table width="100%" cellpadding="3" cellspacing="0" border="0">
				<tr><td colspan="2" class="`, windowcss, `">
					<table width="100%" cellpadding="4" cellspacing="1" style="table-layout: fixed;">
						<tr>
							<td valign="top" width="16%" rowspan="2" style="overflow: hidden;">
								<b>`, member.Link, `</b>
								<div class="smalltext">`)
			if member.Title != "" {
				c.O(`
									`, member.Title, `<br />`)
			}
			if member.Group != "" {
				c.O(`
									`, member.Group, `<br />`)
			}

			if !member.IsGuest {
				// Show the post group if and only if they have no other group or the option is on, and they are in a post group.
				if (c.Theme.Empty("hide_post_group") || member.Group == "") && member.PostGroup != "" {
					c.O(`
									`, member.PostGroup, `<br />`)
				}
				c.O(`
									`, member.GroupStars, `<br />`)

				// Show online and offline buttons?
				if !c.App.SettingEmpty("onlineEnable") {
					openA, closeA := "", ""
					if page.CanSendPM {
						openA = `<a href="` + member.OnlineHref + `" title="` + member.OnlineLabel + `">`
						closeA = `</a>`
					}
					img := member.OnlineText
					span := ""
					if !c.Theme.Empty("use_image_buttons") {
						img = `<img src="` + member.OnlineImageHref + `" style="margin-top: 4px;" alt="` + member.OnlineText + `" />`
						span = `<span class="smalltext"> ` + member.OnlineText + `</span>`
					}
					c.O(`
									`, openA, img, closeA, span, `<br /><br />`)
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
				if !c.Theme.Empty("show_user_images") && empty(c.Options["show_no_avatars"]) {
					c.O(`
									`, member.AvatarImage, `<br />`)
				}

				// Show their personal text?
				if !c.Theme.Empty("show_blurb") && member.Blurb != "" {
					c.O(`
									`, member.Blurb, `<br />
									<br />`)
				}
				// Show the profile, website, email address, and personal message buttons.
				if !c.Theme.Empty("show_profile_buttons") {
					profileBtn := c.Txt("27")
					if !c.Theme.Empty("use_image_buttons") {
						profileBtn = `<img src="` + images + `/icons/profile_sm.gif" alt="` + c.Txt("27") + `" title="` + c.Txt("27") + `" />`
					}
					c.O(`
									<a href="`, member.Href, `">`, profileBtn, `</a>`)
					if member.WebsiteURL != "" {
						wwwBtn := c.Txt("515")
						if !c.Theme.Empty("use_image_buttons") {
							wwwBtn = `<img src="` + images + `/www_sm.gif" alt="` + c.Txt("515") + `" title="` + member.WebsiteTitle + `" />`
						}
						c.O(`
									<a href="`, member.WebsiteURL, `" target="_blank">`, wwwBtn, `</a>`)
					}
					if !member.HideEmail {
						emailBtn := c.Txt("69")
						if !c.Theme.Empty("use_image_buttons") {
							emailBtn = `<img src="` + images + `/email_sm.gif" alt="` + c.Txt("69") + `" title="` + c.Txt("69") + `" />`
						}
						c.O(`
									<a href="mailto:`, member.Email, `">`, emailBtn, `</a>`)
					}
					if !c.User.IsGuest && page.CanSendPM {
						imGif := "off"
						if member.IsOnline {
							imGif = "on"
						}
						pmBtn := member.OnlineLabel
						if !c.Theme.Empty("use_image_buttons") {
							pmBtn = `<img src="` + images + `/im_` + imGif + `.gif" alt="` + member.OnlineLabel + `" />`
						}
						c.O(`
									<a href="`, scripturl, `?action=pm;sa=send;u=`, member.ID, `" title="`, member.OnlineLabel, `">`, pmBtn, `</a>`)
					}
				}
			} else if !member.HideEmail {
				emailBtn := c.Txt("69")
				if !c.Theme.Empty("use_image_buttons") {
					emailBtn = `<img src="` + images + `/email_sm.gif" alt="` + c.Txt("69") + `" title="` + c.Txt("69") + `" />`
				}
				c.O(`
									<br />
									<br />
									<a href="mailto:`, member.Email, `">`, emailBtn, `</a>`)
			}
			c.O(`
								</div>
							</td>
							<td class="`, windowcss, `" valign="top" width="85%" height="100%">
								<table width="100%" border="0"><tr>
									<td align="left" valign="middle">
										<b>`, message.Subject, `</b>`)

			// Show who the message was sent to.
			c.O(`
										<div class="smalltext">&#171; <b> `, c.Txt("sent_to"), `:</b> `)

			// People it was sent directly to....
			if len(message.RecipientsTo) > 0 {
				c.O(strings.Join(message.RecipientsTo, ", "))
			} else if page.Folder != "outbox" {
				// Otherwise, we're just going to say "some people"...
				c.O(`(`, c.Txt("pm_undisclosed_recipients"), `)`)
			}

			c.O(` <b> `, c.Txt("30"), `:</b> `, message.Time, ` &#187;</div>`)

			// If we're in the outbox, show who it was sent to besides the "To:" people.
			if len(message.RecipientsBcc) > 0 {
				c.O(`
										<div class="smalltext">&#171; <b> `, c.Txt("1502"), `:</b> `, strings.Join(message.RecipientsBcc, ", "), ` &#187;</div>`)
			}

			if message.IsRepliedTo {
				c.O(`
										<div class="smalltext">&#171; `, c.Txt("pm_is_replied_to"), ` &#187;</div>`)
			}

			c.O(`
									</td>
									<td align="right" valign="bottom" height="20" nowrap="nowrap" style="font-size: smaller;">`)

			// Show reply buttons if you have the permission to send PMs.
			if page.CanSendPM {
				// You can't really reply if the member is gone.
				if !member.IsGuest {
					replyU := itoa(member.ID)
					if page.Folder == "outbox" {
						replyU = ""
					}
					// Were than more than one recipient you can reply to? (Only in the "button style", or text)
					if len(message.RecipientsTo) > 1 && (!c.Theme.Empty("use_buttons") || c.Theme.Empty("use_image_buttons")) {
						c.O(`
										<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;quote;u=all">`, replyAllButton, `</a>`, c.MenuSeparator)
					}
					c.O(`
										<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;quote;u=`, replyU, `">`, quoteButton, `</a>`, c.MenuSeparator, `
										<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;u=`, member.ID, `">`, replyButton, `</a> `, c.MenuSeparator)
				} else {
					// This is for "forwarding" - even if the member is gone.
					c.O(`
										<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;quote">`, forwardButton, `</a>`, c.MenuSeparator)
				}
			}
			c.O(`
										<a href="`, scripturl, `?action=pm;sa=pmactions;pm_actions[`, message.ID, `]=delete;f=`, page.Folder, `;start=`, page.Start, `;`, lbl, `;sesc=`, c.Sc, `" onclick="return confirm('`, addslashes(c.Txt("154")), `?');">`, deleteButton, `</a>
										<input style="vertical-align: middle;" type="checkbox" name="pms[]" id="deletedisplay`, message.ID, `" value="`, message.ID, `" class="check" onclick="document.getElementById('deletelisting`, message.ID, `').checked = this.checked;" />
									</td>
								</tr></table>
								<hr width="100%" size="1" class="hrcolor" />
								<div class="personalmessage">`, message.Body, `</div>
							</td>
						</tr>
						<tr class="`, windowcss, `">
							<td valign="bottom" class="smalltext" width="85%">
								`)
			if !c.App.SettingEmpty("enableReportPM") && page.Folder != "outbox" {
				c.O(`<div align="right"><a href="`, scripturl, `?action=pm;sa=report;l=`, page.CurrentLabelID, `;pmsg=`, message.ID, `" class="smalltext">`, c.Txt("pm_report_to_admin"), `</a></div>`)
			}

			// Show the member's signature?
			if member.Signature != "" && empty(c.Options["show_no_signatures"]) {
				c.O(`
								<hr width="100%" size="1" class="hrcolor" />
								<div class="signature">`, member.Signature, `</div>`)
			}

			c.O(`
							</td>
						</tr>`)

			// Add an extra line at the bottom if we have labels enabled.
			if page.Folder != "outbox" && page.UsingLabels {
				c.O(`
						<tr class="`, windowcss, `">
							<td valign="bottom" colspan="2" width="100%" align="right">`)
				// Add the label drop down box.
				if page.UsingLabels {
					c.O(`
								<select name="pm_actions[`, message.ID, `]" onchange="if (this.options[this.selectedIndex].value) form.submit();">
									<option value="">`, c.Txt("pm_msg_label_title"), `:</option>
									<option value="" disabled="disabled">---------------</option>`)
					// Build a set of the labels already on this message.
					onMessage := map[int]bool{}
					for _, l := range message.Labels {
						onMessage[l.ID] = true
					}
					// Are there any labels which can be added to this?
					if !message.FullyLabeled {
						c.O(`
									<option value="" disabled="disabled">`, c.Txt("pm_msg_label_apply"), `:</option>`)
						for _, id := range page.LabelOrder {
							label := page.Labels[id]
							if !onMessage[label.ID] {
								c.O(`
										<option value="`, label.ID, `">&nbsp;`, label.Name, `</option>`)
							}
						}
					}
					// ... and are there any that can be removed?
					if len(message.Labels) > 0 && (len(message.Labels) > 1 || !onMessage[-1]) {
						c.O(`
									<option value="" disabled="disabled">`, c.Txt("pm_msg_label_remove"), `:</option>`)
						for _, label := range message.Labels {
							c.O(`
										<option value="`, label.ID, `">&nbsp;`, label.Name, `</option>`)
						}
					}
					c.O(`
								</select>
								<noscript>
								<input type="submit" value="`, c.Txt("pm_apply"), `" />
								</noscript>`)
				}
				c.O(`
							</td>
						</tr>`)
			}

			c.O(`
					</table>
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						currentLabels[`, message.ID, `] = {`)

			if len(message.Labels) > 0 {
				first := true
				for _, label := range message.Labels {
					comma := ","
					if first {
						comma = ""
					}
					c.O(comma, `
								"`, label.ID, `": "`, label.Name, `"`)
					first = false
				}
			}

			c.O(`
						};
					// ]]></script>
				</td></tr>
			</table>
		</td></tr>`)
		}

		c.O(`
			<tr><td style="padding: 0 0 1px 0;"></td></tr>
	</table>

	<div class="tborder" style="padding: 1px; margin-top: 1ex;">
		<table cellpadding="3" cellspacing="0" border="0" width="100%">
			<tr class="catbg" valign="middle">
				<td height="25">
					<div style="float: left;">`, c.Txt("139"), `: `, page.PageIndex, `</div>
					<div style="float: right;"><input type="submit" name="del_selected" value="`, c.Txt("quickmod_delete_selected"), `" style="font-weight: normal;" onclick="if (!confirm('`, c.Txt("smf249"), `')) return false;" /></div>
				</td>
			</tr>
		</table>
	</div>`)
	}

	c.O(`
	<input type="hidden" name="sc" value="`, c.Sc, `" />
</form>`)
}

// templatePMSend is template_send(): the send-a-PM form.
func templatePMSend(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()
	tabindex := 1
	nextTab := func() int { tabindex++; return tabindex - 1 }

	if page.ShowSpellchecking {
		c.O(`
		<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/spellcheck.js"></script>`)
	}

	// Show which messages were sent successfully and which failed.
	if page.SendLog != nil && (len(page.SendLog.Sent) > 0 || len(page.SendLog.Failed) > 0) {
		c.O(`
		<br />
		<table border="0" width="80%" cellspacing="1" cellpadding="3" class="bordercolor" align="center">
			<tr class="titlebg">
				<td>`, c.Txt("pm_send_report"), `</td>
			</tr>
			<tr>
				<td class="windowbg">`)
		for _, logEntry := range page.SendLog.Sent {
			c.O(`<span style="color: green">`, logEntry, `</span><br />`)
		}
		for _, logEntry := range page.SendLog.Failed {
			c.O(`<span style="color: red">`, logEntry, `</span><br />`)
		}
		c.O(`
				</td>
			</tr>
		</table><br />`)
	}

	// Show the preview of the personal message.
	if page.HasPreview {
		c.O(`
		<br />
		<table border="0" width="80%" cellspacing="1" cellpadding="3" class="bordercolor" align="center">
			<tr class="titlebg">
				<td>`, page.PreviewSubject, `</td>
			</tr>
			<tr>
				<td class="windowbg">
					`, page.PreviewMessage, `
				</td>
			</tr>
		</table><br />`)
	}

	// Main message editing box.
	c.O(`
		<table border="0" width="80%" align="center" cellpadding="3" cellspacing="1" class="bordercolor">
			<tr class="titlebg">
				<td><img src="`, images, `/icons/im_newmsg.gif" alt="`, c.Txt("321"), `" title="`, c.Txt("321"), `" />&nbsp;`, c.Txt("321"), `</td>
			</tr><tr>
				<td class="windowbg">
					<form action="`, scripturl, `?action=pm;sa=send2" method="post" accept-charset="`, c.CharacterSet, `" name="postmodify" id="postmodify" onsubmit="submitonce(this);saveEntities();">
						<table border="0" cellpadding="3" width="100%">`)

	// If there were errors for sending the PM, show them.
	if len(page.ErrorMessages) > 0 {
		c.O(`
							<tr>
								<td></td>
								<td align="left">
									<b>`, c.Txt("error_while_submitting"), `</b>
									<div style="color: red; margin: 1ex 0 2ex 3ex;">
										`, strings.Join(page.ErrorMessages, "<br />"), `
									</div>
								</td>
							</tr>`)
	}

	// To and bcc. Include a button to search for members.
	toErr := ""
	if page.PostError["no_to"] || page.PostError["bad_to"] {
		toErr = ` style="color: red;"`
	}
	bccErr := ""
	if page.PostError["bad_bcc"] {
		bccErr = ` style="color: red;"`
	}
	c.O(`
							<tr>
								<td align="right"><b`, toErr, `>`, c.Txt("150"), `:</b></td>
								<td class="smalltext">
									<input type="text" name="to" id="to" value="`, page.To, `" tabindex="`, nextTab(), `" size="40" />&nbsp;
									<a href="`, scripturl, `?action=findmember;input=to;quote=1;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, images, `/icons/assist.gif" alt="`, c.Txt("find_members"), `" /></a> <a href="`, scripturl, `?action=findmember;input=to;quote=1;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);">`, c.Txt("find_members"), `</a>
								</td>
							</tr><tr>
								<td align="right"><b`, bccErr, `>`, c.Txt("1502"), `:</b></td>
								<td class="smalltext">
									<input type="text" name="bcc" id="bcc" value="`, page.Bcc, `" tabindex="`, nextTab(), `" size="40" />&nbsp;
									<a href="`, scripturl, `?action=findmember;input=bcc;quote=1;sesc=`, c.Sc, `" onclick="return reqWin(this.href, 350, 400);"><img src="`, images, `/icons/assist.gif" alt="`, c.Txt("find_members"), `" /></a> `, c.Txt("748"), `
								</td>
							</tr>`)
	// Subject of personal message.
	subjErr := ""
	if page.PostError["no_subject"] {
		subjErr = ` style="color: red;"`
	}
	c.O(`
							<tr>
								<td align="right"><b`, subjErr, `>`, c.Txt("70"), `:</b></td>
								<td><input type="text" name="subject" value="`, page.Subject, `" tabindex="`, nextTab(), `" size="40" maxlength="50" /></td>
							</tr>`)

	if page.VisualVerification {
		c.O(`
							<tr>
								<td align="right" valign="top">
									<b>`, c.Txt("pm_visual_verification_label"), `:</b>
								</td>
								<td>`)
		// No-GD path: per-letter images (use_graphic_library is false).
		c.O(`
									<img src="`, page.VerificationHref, `;letter=1" alt="`, c.Txt("pm_visual_verification_desc"), `" />
									<img src="`, page.VerificationHref, `;letter=2" alt="`, c.Txt("pm_visual_verification_desc"), `" />
									<img src="`, page.VerificationHref, `;letter=3" alt="`, c.Txt("pm_visual_verification_desc"), `" />
									<img src="`, page.VerificationHref, `;letter=4" alt="`, c.Txt("pm_visual_verification_desc"), `" />
									<img src="`, page.VerificationHref, `;letter=5" alt="`, c.Txt("pm_visual_verification_desc"), `" /><br />`)
		c.O(`
									<a href="`, page.VerificationHref, `;sound" onclick="return reqWin(this.href, 400, 120);">`, c.Txt("pm_visual_verification_listen"), `</a><br /><br />
									<input type="text" name="visual_verification_code" size="30" tabindex="`, nextTab(), `" />
									<div class="smalltext">`, c.Txt("pm_visual_verification_desc"), `</div>
								</td>
							</tr>`)
	}

	// Show BBC buttons, smileys and textbox.
	c.themePostbox(page.Message, postboxOpts{postError: page.PostError, nextTab: nextTab})

	// Send, Preview, spellcheck buttons.
	c.O(`
							<tr>
								<td align="right" colspan="2">
									<input type="submit" value="`, c.Txt("148"), `" tabindex="`, nextTab(), `" onclick="return submitThisOnce(this);" accesskey="s" />
									<input type="submit" name="preview" value="`, c.Txt("507"), `" tabindex="`, nextTab(), `" onclick="return submitThisOnce(this);" accesskey="p" />`)
	if page.ShowSpellchecking {
		c.O(`
									<input type="button" value="`, c.Txt("spell_check"), `" tabindex="`, nextTab(), `" onclick="spellCheck('postmodify', 'message');" />`)
	}
	outboxChecked := ""
	if page.CopyToOutbox {
		outboxChecked = ` checked="checked"`
	}
	repliedTo := 0
	if page.QuotedMessage != nil {
		repliedTo = page.QuotedMessage.ID
	}
	c.O(`
								</td>
							</tr>
							<tr>
								<td></td>
								<td align="left">
									<label for="outbox"><input type="checkbox" name="outbox" id="outbox" value="1" tabindex="`, nextTab(), `"`, outboxChecked, ` class="check" /> `, c.Txt("pm_save_outbox"), `</label>
								</td>
							</tr>
						</table>
						<input type="hidden" name="sc" value="`, c.Sc, `" />
						<input type="hidden" name="seqnum" value="`, c.FormSequenceNumber, `" />
						<input type="hidden" name="replied_to" value="`, repliedTo, `" />
						<input type="hidden" name="f" value="`, page.Folder, `" />
						<input type="hidden" name="l" value="`, page.CurrentLabelID, `" />
					</form>
				</td>
			</tr>
		</table>`)

	// Some hidden information is needed in order to make the spell checking work.
	if page.ShowSpellchecking {
		c.O(`
		<form name="spell_form" id="spell_form" method="post" accept-charset="`, c.CharacterSet, `" target="spellWindow" action="`, scripturl, `?action=spellcheck"><input type="hidden" name="spellstring" value="" /></form>`)
	}

	// Show the message you're replying to.
	if page.Reply && page.QuotedMessage != nil {
		c.O(`
		<br />
		<br />
		<table width="100%" border="0" cellspacing="1" cellpadding="4" class="bordercolor">
			<tr>
				<td colspan="2" class="windowbg"><b>`, c.Txt("319"), `: `, page.QuotedMessage.Subject, `</b></td>
			</tr>
			<tr>
				<td class="windowbg2">
					<table width="100%" border="0" cellspacing="0" cellpadding="0">
						<tr>
							<td class="windowbg2">`, c.Txt("318"), `: `, page.QuotedMessage.MemberName, `</td>
							<td class="windowbg2" align="right">`, c.Txt("30"), `: `, page.QuotedMessage.Time, `</td>
						</tr>
					</table>
				</td>
			</tr>
			<tr>
				<td colspan="2" class="windowbg">`, page.QuotedMessage.Body, `</td>
			</tr>
		</table>`)
	}

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function autocompleter(element)
			{
				if (typeof(element) != "object")
					element = document.getElementById(element);

				this.element = element;
				this.key = null;
				this.request = null;
				this.source = null;
				this.lastSearch = "";
				this.oldValue = "";
				this.cache = [];

				this.change = function (ev, force)
				{
					if (window.event)
						this.key = window.event.keyCode + 0;
					else
						this.key = ev.keyCode + 0;
					if (this.key == 27)
						return true;
					if (this.key == 34 || this.key == 8 || this.key == 13 || (this.key >= 37 && this.key <= 40))
						force = false;

					if (isEmptyText(this.element))
						return true;

					if (this.request != null && typeof(this.request) == "object")
						this.request.abort();

					var element = this.element, search = this.element.value.replace(/^("[^"]+",[ ]*)+/, "").replace(/^([^,]+,[ ]*)+/, "");
					this.oldValue = this.element.value.substr(0, this.element.value.length - search.length);
					if (search.substr(0, 1) == '"')
						search = search.substr(1);

					if (search == "" || search.substr(search.length - 1) == '"')
						return true;

					if (this.lastSearch == search)
					{
						if (force)
							this.select(this.cache[0]);

						return true;
					}
					else if (search.substr(0, this.lastSearch.length) == this.lastSearch && this.cache.length != 100)
					{
						// Instead of hitting the server again, just narrow down the results...
						var newcache = [], j = 0;
						for (var k = 0; k < this.cache.length; k++)
						{
							if (this.cache[k].substr(0, search.length) == search)
								newcache[j++] = this.cache[k];
						}

						if (newcache.length != 0)
						{
							this.lastSearch = search;
							this.cache = newcache;

							if (force)
								this.select(newcache[0]);

							return true;
						}
					}

					this.request = new XMLHttpRequest();
					this.request.onreadystatechange = function ()
					{
						element.autocompleter.handler(force);
					}

					this.request.open("GET", this.source + escape(textToEntities(search).replace(/&#(\d+);/g, "%#$1%")).replace(/%26/g, "%25%23038%25") + ";" + (new Date().getTime()), true);
					this.request.send(null);

					return true;
				}
				this.keyup = function (ev)
				{
					this.change(ev, true);

					return true;
				}
				this.keydown = function ()
				{
					if (this.request != null && typeof(this.request) == "object")
						this.request.abort();
				}
				this.handler = function (force)
				{
					if (this.request.readyState != 4)
						return true;

					var response = this.request.responseText.split("\n");
					this.lastSearch = this.element.value;
					this.cache = response;

					if (response.length < 2)
						return true;

					if (force)
						this.select(response[0]);

					return true;
				}
				this.select = function (value)
				{
					if (value == "")
						return;

					var i = this.element.value.length + (this.element.value.substr(this.oldValue.length, 1) == '"' ? 0 : 1);
					this.element.value = this.oldValue + '"' + value + '"';

					if (typeof(this.element.createTextRange) != "undefined")
					{
						var d = this.element.createTextRange();
						d.moveStart("character", i);
						d.select();
					}
					else if (this.element.setSelectionRange)
					{
						this.element.focus();
						this.element.setSelectionRange(i, this.element.value.length);
					}
				}

				this.element.autocompleter = this;
				this.element.setAttribute("autocomplete", "off");

				this.element.onchange = function (ev)
				{
					this.autocompleter.change(ev);
				}
				this.element.onkeyup = function (ev)
				{
					this.autocompleter.keyup(ev);
				}
				this.element.onkeydown = function (ev)
				{
					this.autocompleter.keydown(ev);
				}
			}

			if (window.XMLHttpRequest)
			{
				var toComplete = new autocompleter("to"), bccComplete = new autocompleter("bcc");
				toComplete.source = "`, scripturl, `?action=requestmembers;sesc=`, c.Sc, `;search=";
				bccComplete.source = "`, scripturl, `?action=requestmembers;sesc=`, c.Sc, `;search=";
			}

			function saveEntities()
			{
				var textFields = ["subject", "message"];
				for (i in textFields)
					if (document.forms.postmodify.elements[textFields[i]])
						document.forms.postmodify[textFields[i]].value = document.forms.postmodify[textFields[i]].value.replace(/&#/g, "&#38;#");
			}
		// ]]></script>`)
}

// templatePMAskDelete is template_ask_delete().
func templatePMAskDelete(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL

	lbl := ""
	if page.CurrentLabelID != -1 {
		lbl = ";l=" + itoa(page.CurrentLabelID)
	}
	title := c.Txt("412")
	if page.DeleteAll {
		title = c.Txt("411")
	}

	c.O(`
			<table border="0" width="80%" cellpadding="4" cellspacing="1" class="bordercolor" align="center">
				<tr class="titlebg">
					<td>`, title, `</td>
				</tr>
				<tr>
					<td class="windowbg">
						`, c.Txt("413"), `<br />
						<br />
						<b><a href="`, scripturl, `?action=pm;sa=removeall2;f=`, page.Folder, `;`, lbl, `;sesc=`, c.Sc, `">`, c.Txt("163"), `</a> - <a href="javascript:history.go(-1);">`, c.Txt("164"), `</a></b>
					</td>
				</tr>
			</table>`)
}

// templatePMPrune is template_prune().
func templatePMPrune(c *Ctx) {
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=pm;sa=prune" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="return confirm('`, c.Txt("pm_prune_warning"), `');">
		<table width="60%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
			<tr class="catbg">
				<td>`, c.Txt("pm_prune"), `</td>
			</tr>
			<tr class="windowbg">
				<td>`, c.Txt("pm_prune_desc1"), ` <input type="text" name="age" size="3" value="14" /> `, c.Txt("pm_prune_desc2"), `</td>
			</tr>
			<tr class="windowbg">
				<td align="right"><input type="submit" value="`, c.Txt("smf138"), `" /></td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templatePMLabels is template_labels().
func templatePMLabels(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=pm;sa=manlabels;sesc=`, c.Sc, `" method="post" accept-charset="`, c.CharacterSet, `">
		<table width="60%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("pm_manage_labels"), `</td>
			</tr>
			<tr class="windowbg2">
				<td colspan="2" style="padding: 1ex;"><span class="smalltext">`, c.Txt("pm_labels_desc"), `</span></td>
			</tr>
			<tr class="catbg3">
				<td colspan="2">
					<div style="float: right; width: 4%; text-align: center;"><input type="checkbox" class="check" onclick="invertAll(this, this.form);" /></div>
					`, c.Txt("pm_label_name"), `
				</td>
			</tr>`)
	// Count the real (non-inbox) labels.
	realLabels := 0
	for _, id := range page.LabelOrder {
		if id != -1 {
			realLabels++
		}
	}
	if realLabels == 0 {
		c.O(`
			<tr class="windowbg2">
				<td colspan="2" align="center">`, c.Txt("pm_labels_no_exist"), `</td>
			</tr>`)
	} else {
		alternate := true
		for _, id := range page.LabelOrder {
			label := page.Labels[id]
			if label.ID != -1 {
				css := "windowbg"
				if alternate {
					css = "windowbg2"
				}
				c.O(`
				<tr class="`, css, `">
					<td>
						<input type="text" name="label_name[`, label.ID, `]" value="`, label.Name, `" size="30" maxlength="30" />
					</td>
					<td width="4%" align="center"><input type="checkbox" class="check" name="delete_label[`, label.ID, `]" /></td>
				</tr>`)
				alternate = !alternate
			}
		}

		c.O(`
			<tr class="catbg3">
				<td align="right" colspan="2">
					<input type="submit" name="save" value="`, c.Txt("10"), `" style="font-weight: normal;" />
					<input type="submit" name="delete" value="`, c.Txt("quickmod_delete_selected"), `" style="font-weight: normal;" onclick="return confirm('`, c.Txt("pm_labels_delete"), `');" />
				</td>
			</tr>`)
	}
	c.O(`
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>
	<form action="`, scripturl, `?action=pm;sa=manlabels;sesc=`, c.Sc, `" method="post" accept-charset="`, c.CharacterSet, `" style="margin-top: 1ex;">
		<table width="60%" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
			<tr class="titlebg">
				<td colspan="2" align="left">
					`, c.Txt("pm_label_add_new"), `
				</td>
			</tr>
			<tr class="windowbg2">
				<td align="right" width="40%">
					<b>`, c.Txt("pm_label_name"), `:</b>
				</td>
				<td align="left">
					<input type="text" name="label" value="" size="30" maxlength="20" />
				</td>
			</tr>
			<tr class="catbg3">
				<td colspan="2" align="right">
					<input type="submit" name="add" value="`, c.Txt("pm_label_add_new"), `" style="font-weight: normal;" />
				</td>
			</tr>
		</table>
	</form>`)
}

// templatePMReport is template_report_message().
func templatePMReport(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=pm;sa=report;l=`, page.CurrentLabelID, `" method="post" accept-charset="`, c.CharacterSet, `">
		<input type="hidden" name="pmsg" value="`, page.PMID, `" />
		<table border="0" width="80%" cellspacing="0" class="tborder" align="center" cellpadding="4">
			<tr class="titlebg">
				<td>`, c.Txt("pm_report_title"), `</td>
			</tr>
			<tr class="windowbg2">
				<td align="left">
					<span class="smalltext">`, c.Txt("pm_report_desc"), `</span>
				</td>
			</tr>`)

	// If there is more than one admin on the forum, allow the user to choose the one they want to direct to.
	// !!! Why?
	if page.AdminCount > 1 {
		c.O(`
			<tr class="windowbg">
				<td align="left">
					<b>`, c.Txt("pm_report_admins"), `:</b>
					<select name="ID_ADMIN">
						<option value="0">`, c.Txt("pm_report_all_admins"), `</option>`)
		for _, id := range page.AdminIDs {
			c.O(`
						<option value="`, id, `">`, page.Admins[id], `</option>`)
		}
		c.O(`
					</select>
				</td>
			</tr>`)
	}

	c.O(`
			<tr class="windowbg">
				<td align="left">
					<b>`, c.Txt("pm_report_reason"), `:</b>
				</td>
			</tr>
			<tr class="windowbg">
				<td align="center">
					<textarea name="reason" rows="4" cols="70" style="width: 80%;"></textarea>
				</td>
			</tr>
			<tr class="windowbg">
				<td align="center">
					<input type="submit" name="report" value="`, c.Txt("pm_report_message"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templatePMReportDone is template_report_message_complete().
func templatePMReportDone(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL

	c.O(`
			<table border="0" width="80%" cellspacing="0" class="tborder" align="center" cellpadding="4">
				<tr class="titlebg">
					<td>`, c.Txt("pm_report_title"), `</td>
				</tr>
				<tr class="windowbg">
					<td align="left">
						`, c.Txt("pm_report_done"), `
					</td>
				</tr>
				<tr class="windowbg">
					<td align="center">
						<br /><a href="`, scripturl, `?action=pm;l=`, page.CurrentLabelID, `">`, c.Txt("pm_report_return"), `</a>
					</td>
				</tr>
			</table>`)
}

// templatePMSearch is template_search() from PersonalMessage.template.php.
func templatePMSearch(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function expandCollapseLabels()
			{
				var current = document.getElementById("searchLabelsExpand").style.display != "none";

				document.getElementById("searchLabelsExpand").style.display = current ? "none" : "";
				document.getElementById("expandLabelsIcon").src = smf_images_url + (current ? "/expand.gif" : "/collapse.gif");
			}
		// ]]></script>
	<form action="`, scripturl, `?action=pm;sa=search2" method="post" accept-charset="`, c.CharacterSet, `" name="pmSearchForm">
		<table border="0" width="75%" align="center" cellpadding="3" cellspacing="0" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("pm_search_title"), `</td>
			</tr>`)

	if len(page.SearchErrMsgs) > 0 {
		c.O(`
			<tr>
				<td class="windowbg">
					<div style="color: red; margin: 1ex 0 2ex 3ex;">
						`, strings.Join(page.SearchErrMsgs, "<br />"), `
					</div>
				</td>
			</tr>`)
	}

	c.O(`
			<tr>
				<td class="windowbg">`)

	searchVal := ""
	if page.SPSearch != "" {
		searchVal = ` value="` + page.SPSearch + `"`
	}

	if page.SimpleSearch {
		c.O(`
					<b>`, c.Txt("pm_search_text"), `:</b><br />
					<input type="text" name="search"`, searchVal, ` size="40" />&nbsp;
					<input type="submit" name="submit" value="`, c.Txt("pm_search_go"), `" /><br />
					<a href="`, scripturl, `?action=pm;sa=search;advanced" onclick="this.href += ';search=' + escape(document.forms.pmSearchForm.search.value);">`, c.Txt("pm_search_advanced"), `</a>
					<input type="hidden" name="advanced" value="0" />`)
	} else {
		typeSel1, typeSel2 := ` selected="selected"`, ""
		if page.SPSearchtype != 0 {
			typeSel1, typeSel2 = "", ` selected="selected"`
		}
		userspec := page.SPUserspec
		if userspec == "" {
			userspec = "*"
		}
		minage := "0"
		if page.SPMinage != 0 {
			minage = itoa(page.SPMinage)
		}
		maxage := "9999"
		if page.SPMaxage != 0 {
			maxage = itoa(page.SPMaxage)
		}
		showComplete, subjectOnly := "", ""
		if page.SPShowComplete {
			showComplete = ` checked="checked"`
		}
		if page.SPSubjectOnly {
			subjectOnly = ` checked="checked"`
		}
		c.O(`
					<input type="hidden" name="advanced" value="1" />
					<table cellpadding="1" cellspacing="3" border="0">
						<tr>
							<td>
								<b>`, c.Txt("pm_search_text"), `:</b>
							</td>
							<td></td>
							<td>
								<b>`, c.Txt("pm_search_user"), `:</b>
							</td>
						</tr><tr>
							<td>
								<input type="text" name="search"`, searchVal, ` size="40" />
							</td><td style="padding-right: 2ex;">
								<select name="searchtype">
									<option value="1"`, typeSel1, `>`, c.Txt("pm_search_match_all"), `</option>
									<option value="2"`, typeSel2, `>`, c.Txt("pm_search_match_any"), `</option>
								</select>
							</td><td>
								<input type="text" name="userspec" value="`, userspec, `" size="40" />
							</td>
						</tr><tr>
							<td style="padding-top: 2ex;" colspan="2"><b>`, c.Txt("pm_search_options"), `:</b></td>
							<td style="padding-top: 2ex;"><b>`, c.Txt("pm_search_post_age"), `: </b></td>
						</tr><tr>
							<td colspan="2">
								<label for="show_complete"><input type="checkbox" name="show_complete" id="show_complete" value="1"`, showComplete, ` class="check" /> `, c.Txt("pm_search_show_complete"), `</label><br />
								<label for="subject_only"><input type="checkbox" name="subject_only" id="subject_only" value="1"`, subjectOnly, ` class="check" /> `, c.Txt("pm_search_subject_only"), `</label>
							</td>
							<td>
								`, c.Txt("pm_search_between"), ` <input type="text" name="minage" value="`, minage, `" size="5" maxlength="5" />&nbsp;`, c.Txt("pm_search_between_and"), `&nbsp;<input type="text" name="maxage" value="`, maxage, `" size="5" maxlength="5" /> `, c.Txt("pm_search_between_days"), `.
							</td>
						</tr><tr>
							<td style="padding-top: 2ex;" colspan="2"><b>`, c.Txt("pm_search_order"), `:</b></td>
							<td></td>
						</tr><tr>
							<td colspan="2">
								<select name="sort">
		<!--- <option value="relevance|desc">`, c.Txt("pm_search_orderby_relevant_first"), `</option> --->
									<option value="ID_PM|desc">`, c.Txt("pm_search_orderby_recent_first"), `</option>
									<option value="ID_PM|asc">`, c.Txt("pm_search_orderby_old_first"), `</option>
								</select>
							</td>
							<td></td>
						</tr>`)

		if page.UsingLabels {
			checkAllStyle := ""
			if page.CheckAll {
				checkAllStyle = `style="display: none;"`
			}
			c.O(`
						<tr>
							<td colspan="4">
					<a href="javascript:void(0);" onclick="expandCollapseLabels(); return false;"><img src="`, images, `/expand.gif" id="expandLabelsIcon" alt="" /></a> <a href="javascript:void(0);" onclick="expandCollapseLabels(); return false;"><b>`, c.Txt("pm_search_choose_label"), `</b></a><br />

					<table id="searchLabelsExpand" width="90%" border="0" cellpadding="1" cellspacing="0" align="center" `, checkAllStyle, `>`)
			alternate := true
			for _, label := range page.SearchLabels {
				if alternate {
					c.O(`
						<tr>`)
				}
				checked := ""
				if label.Checked {
					checked = `checked="checked"`
				}
				c.O(`
							<td width="50%">
								<label for="searchlabel_`, label.ID, `"><input type="checkbox" id="searchlabel_`, label.ID, `" name="searchlabel[`, label.ID, `]" value="`, label.ID, `" `, checked, ` class="check" />
								`, label.Name, `</label>
							</td>`)
				if !alternate {
					c.O(`
						</tr>`)
				}
				alternate = !alternate
			}
			if !alternate {
				c.O(`
						<td width="50%"></td>
					</tr>`)
			}
			checkAllChecked := ""
			if page.CheckAll {
				checkAllChecked = `checked="checked"`
			}
			c.O(`
					</table>

					<br />
					<input type="checkbox" name="all" id="check_all" value="" `, checkAllChecked, ` onclick="invertAll(this, this.form, 'searchlabel');" class="check" /><i> <label for="check_all">`, c.Txt("737"), `</label></i><br />
							</td>
						</tr>`)
		}

		c.O(`
					</table>
					<br />

					<div style="padding: 2px;"><input type="submit" name="submit" value="`, c.Txt("pm_search_go"), `" /></div>`)
	}

	c.O(`
				</td>
			</tr>
		</table>
	</form>`)
}

// templatePMSearchResults is template_search_results() from
// PersonalMessage.template.php.
func templatePMSearchResults(c *Ctx) {
	page := c.Page.(*PMCtx)
	scripturl := c.App.ScriptURL

	if page.SPShowComplete {
		c.O(`
		<table border="0" width="98%" align="center" cellpadding="3" cellspacing="1" class="bordercolor">
			<tr class="titlebg">
				<td colspan="3">`, c.Txt("pm_search_results"), `</td>
			</tr>
			<tr class="catbg" height="30">
				<td colspan="3"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
			</tr>
		</table><br />`)
	} else {
		c.O(`
		<table border="0" width="98%" align="center" cellpadding="3" cellspacing="1" class="bordercolor">
			<tr class="titlebg">
				<td colspan="3">`, c.Txt("pm_search_results"), `</td>
			</tr>
			<tr class="catbg">
				<td colspan="3"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
			</tr>
			<tr class="titlebg">
				<td width="30%">`, c.Txt("317"), `</td>
				<td width="50%">`, c.Txt("319"), `</td>
				<td width="20%">`, c.Txt("318"), `</td>
			</tr>`)
	}

	alternate := true
	for _, message := range page.SearchResults {
		if page.SPShowComplete {
			c.O(`
		<table width="98%" align="center" cellpadding="3" cellspacing="1" border="0" class="bordercolor">
			<tr class="titlebg">
				<td align="left">
					<div style="float: left;">
					`, message.Counter, `&nbsp;&nbsp;<a href="`, message.Href, `">`, message.Subject, `</a>
					</div>
					<div style="float: right;">
						`, c.Txt("176"), `: `, message.Time, `
					</div>
				</td>
			</tr>
			<tr class="catbg">
				<td>`, c.Txt("318"), `: `, message.Member.Link, `, `, c.Txt("324"), `: `)
			if len(message.RecipientsTo) > 0 {
				c.O(strings.Join(message.RecipientsTo, ", "))
			} else if page.Folder != "outbox" {
				c.O(`(`, c.Txt("pm_undisclosed_recipients"), `)`)
			}
			c.O(`
				</td>
			</tr>
			<tr class="windowbg2" valign="top">
				<td>`, message.Body, `</td>
			</tr>
			<tr class="windowbg">
				<td align="right" class="middletext">`)
			if page.CanSendPM {
				quoteButton := c.createButton("quote.gif", "145", "145", `align="middle"`)
				replyButton := c.createButton("im_reply.gif", "146", "146", `align="middle"`)
				lbl := ""
				if page.CurrentLabelID != -1 {
					lbl = ";l=" + itoa(page.CurrentLabelID)
				}
				if !message.Member.IsGuest {
					replyU := itoa(message.Member.ID)
					if page.Folder == "outbox" {
						replyU = ""
					}
					c.O(`
							<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;quote;u=`, replyU, `">`, quoteButton, `</a>`, c.MenuSeparator, `
							<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;u=`, message.Member.ID, `">`, replyButton, `</a> `, c.MenuSeparator)
				} else {
					c.O(`
							<a href="`, scripturl, `?action=pm;sa=send;f=`, page.Folder, lbl, `;pmsg=`, message.ID, `;quote">`, quoteButton, `</a>`, c.MenuSeparator)
				}
			}
			c.O(`
				</td>
			</tr>
		</table><br />`)
		} else {
			bg := "windowbg2"
			if alternate {
				bg = "windowbg"
			}
			c.O(`
			<tr class="`, bg, `" valign="top">
				<td>`, message.Time, `</td>
				<td>`, message.Link, `</td>
				<td>`, message.Member.Link, `</td>
			</tr>`)
		}
		alternate = !alternate
	}

	if page.SPShowComplete {
		if len(page.SearchResults) == 0 {
			c.O(`
		<table width="98%" align="center" cellpadding="3" cellspacing="0" border="0" class="tborder" style="border-width: 0 1px 1px 1px;">
			<tr class="windowbg">
				<td>`, c.Txt("pm_search_none_found"), `</td>
			</tr>
		</table><br />`)
		}
		c.O(`
		<table width="98%" align="center" cellpadding="3" cellspacing="0" border="0" class="tborder" style="border-width: 0 1px 1px 1px;">
			<tr class="catbg" height="30">
				<td colspan="3"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
			</tr>
		</table>`)
	} else {
		if len(page.SearchResults) == 0 {
			c.O(`
			<tr class="windowbg2">
				<td colspan="3">`, c.Txt("pm_search_none_found"), `</td>
			</tr>`)
		}
		c.O(`
			<tr class="catbg">
				<td colspan="3"><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
			</tr>
		</table>`)
	}
}
