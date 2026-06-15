package app

// Hand-port of Themes/default/Post.template.php template_main() and
// template_postbox(), plus theme_postbox() from Sources/Subs-Post.php.

import "strings"

// templatePostMain is template_main().
func templatePostMain(c *Ctx) {
	page := c.Page.(*PostCtx)
	scripturl := c.App.ScriptURL
	tabindex := 1
	nextTab := func() int { tabindex++; return tabindex - 1 }

	if page.ShowSpellchecking {
		c.O(`
		<script language="JavaScript" type="text/javascript" src="`, c.Theme.DefaultThemeURL(), `/spellcheck.js"></script>`)
	}

	// Start the javascript... and boy is there a lot.
	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[`)

	// Start with message icons - and any missing from this theme.
	c.O(`
			var icon_urls = {`)
	for _, icon := range page.Icons {
		comma := ","
		if icon.IsLast {
			comma = ""
		}
		c.O(`
				"`, icon.Value, `": "`, icon.URL, `"`, comma)
	}
	c.O(`
			};`)

	// The actual message icon selector.
	c.O(`
			function showimage()
			{
				document.images.icons.src = icon_urls[document.forms.postmodify.icon.options[document.forms.postmodify.icon.selectedIndex].value];
			}`)
	// The functions used to preview a posts without loading a new page.
	currentBoard := "null"
	if c.Board != 0 {
		currentBoard = itoa(c.Board)
	}
	makePoll := "false"
	if page.MakePoll {
		makePoll = "true"
	}
	firefoxFix := ""
	if c.Browser.IsFirefox {
		firefoxFix = `
					// Firefox doesn't render <marquee> that have been put it using javascript
					if (document.forms.postmodify.elements["message"].value.indexOf("[move]") != -1)
					{
						return submitThisOnce(document.forms.postmodify);
					}`
	}
	c.O(`
			var current_board = `, currentBoard, `;
			var make_poll = `, makePoll, `;
			var txt_preview_title = "`, c.Txt("preview_title"), `";
			var txt_preview_fetch = "`, c.Txt("preview_fetch"), `";
			function previewPost()
			{
				if (window.XMLHttpRequest)
				{`, firefoxFix, `
					// Opera didn't support setRequestHeader() before 8.01.
					if (typeof(window.opera) != "undefined")
					{
						var test = new XMLHttpRequest();
						if (typeof(test.setRequestHeader) != "function")
							return submitThisOnce(document.forms.postmodify);
					}
					// !!! Currently not sending poll options and option checkboxes.
					var i, x = new Array();
					var textFields = ["subject", "message", "icon", "guestname", "email", "question", "topic", "sc"];
					var numericFields = [
						"board", "topic", "num_replies",
						"poll_max_votes", "poll_expire", "poll_change_vote", "poll_hide"
					];
					var checkboxFields = [
						"ns",
					];

					for (i in textFields)
						if (document.forms.postmodify.elements[textFields[i]])
							x[x.length] = textFields[i] + "=" + escape(textToEntities(document.forms.postmodify[textFields[i]].value.replace(/&#/g, "&#38;#"))).replace(/\+/g, "%2B");
					for (i in numericFields)
						if (document.forms.postmodify.elements[numericFields[i]] && typeof(document.forms.postmodify[numericFields[i]].value) != "undefined")
							x[x.length] = numericFields[i] + "=" + parseInt(document.forms.postmodify.elements[numericFields[i]].value);
					for (i in checkboxFields)
						if (document.forms.postmodify.elements[checkboxFields[i]] && document.forms.postmodify.elements[checkboxFields[i]].checked)
							x[x.length] = checkboxFields[i] + "=" + document.forms.postmodify.elements[checkboxFields[i]].value;

					sendXMLDocument(smf_scripturl + "?action=post2" + (current_board ? ";board=" + current_board : "") + (make_poll ? ";poll" : "") + ";preview;xml", x.join("&"), onDocSent);

					document.getElementById("preview_section").style.display = "";
					setInnerHTML(document.getElementById("preview_subject"), txt_preview_title);
					setInnerHTML(document.getElementById("preview_body"), txt_preview_fetch);

					return false;
				}
				else
					return submitThisOnce(document.forms.postmodify);
			}
			function onDocSent(XMLDoc)
			{
				if (!XMLDoc)
				{
					document.forms.postmodify.preview.onclick = new function ()
					{
						return true;
					}
					document.forms.postmodify.preview.click();
				}

				// Show the preview section.
				var i, preview = XMLDoc.getElementsByTagName("smf")[0].getElementsByTagName("preview")[0];
				setInnerHTML(document.getElementById("preview_subject"), preview.getElementsByTagName("subject")[0].firstChild.nodeValue);

				var bodyText = "";
				for (i = 0; i < preview.getElementsByTagName("body")[0].childNodes.length; i++)
					bodyText += preview.getElementsByTagName("body")[0].childNodes[i].nodeValue;

				setInnerHTML(document.getElementById("preview_body"), bodyText);
				document.getElementById("preview_body").className = "post";

				// Show a list of errors (if any).
				var errors = XMLDoc.getElementsByTagName("smf")[0].getElementsByTagName("errors")[0];
				var numErrors = errors.getElementsByTagName("error").length, errorList = new Array();
				for (i = 0; i < numErrors; i++)
					errorList[errorList.length] = errors.getElementsByTagName("error")[i].firstChild.nodeValue;
				document.getElementById("errors").style.display = numErrors == 0 ? "none" : "";
				document.getElementById("error_serious").style.display = errors.getAttribute("serious") == 1 ? "" : "none";
				setInnerHTML(document.getElementById("error_list"), numErrors == 0 ? "" : errorList.join("<br />"));

				// Show a warning if the topic has been locked.
				document.getElementById("lock_warning").style.display = errors.getAttribute("topic_locked") == 1 ? "" : "none";

				// Adjust the color of captions if the given data is erroneous.
				var captions = errors.getElementsByTagName("caption"), numCaptions = errors.getElementsByTagName("caption").length;
				for (i = 0; i < numCaptions; i++)
					if (document.getElementById("caption_" + captions[i].getAttribute("name")))
						document.getElementById("caption_" + captions[i].getAttribute("name")).style.color = captions[i].getAttribute("color");

				if (errors.getElementsByTagName("post_error").length == 1)
					document.forms.postmodify.message.style.border = "1px solid red";
				else if (document.forms.postmodify.message.style.borderColor == "red" || document.forms.postmodify.message.style.borderColor == "red red red red")
				{
					if (typeof(document.forms.postmodify.message.runtimeStyle) == "undefined")
						document.forms.postmodify.message.style.border = null;
					else
						document.forms.postmodify.message.style.borderColor = "";
				}

				// Set the new number of replies.
				if (document.forms.postmodify.elements["num_replies"])
					document.forms.postmodify.num_replies.value = XMLDoc.getElementsByTagName("smf")[0].getElementsByTagName("num_replies")[0].firstChild.nodeValue;

				var newPosts = XMLDoc.getElementsByTagName("smf")[0].getElementsByTagName("new_posts")[0] ? XMLDoc.getElementsByTagName("smf")[0].getElementsByTagName("new_posts")[0].getElementsByTagName("post") : {length: 0};
				var numNewPosts = newPosts.length;
				if (numNewPosts != 0)
				{
					var newTable = '<span id="new_replies"></span><table width="100%" class="windowbg" cellspacing="0" cellpadding="2" align="center" style="table-layout: fixed;">';
					for (i = 0; i < numNewPosts; i++)
						newTable += '<tr class="catbg"><td colspan="2" align="left" class="smalltext"><div style="float: right;">`, c.Txt("280"), `: ' + newPosts[i].getElementsByTagName("time")[0].firstChild.nodeValue + ' <img src="' + smf_images_url + '/`, c.User.Language, `/new.gif" alt="`, c.Txt("preview_new"), `" /></div>`, c.Txt("279"), `: ' + newPosts[i].getElementsByTagName("poster")[0].firstChild.nodeValue + '</td></tr><tr class="windowbg2"><td colspan="2" class="smalltext" id="msg' + newPosts[i].getAttribute("id") + '" width="100%"><div align="right" class="smalltext"><a href="#top" onclick="return insertQuoteFast(\'' + newPosts[i].getAttribute("id") + '\');">`, c.Txt("260"), `</a></div><div class="post">' + newPosts[i].getElementsByTagName("message")[0].firstChild.nodeValue + '</div></td></tr>';
					newTable += '</table>';
					setOuterHTML(document.getElementById("new_replies"), newTable);
				}

				if (typeof(smf_codeFix) != "undefined")
					smf_codeFix();
			}`)

	// A function needed to discern HTML entities from non-western characters.
	c.O(`
			function saveEntities()
			{
				var textFields = ["subject", "message", "guestname", "question"];
				for (i in textFields)
					if (document.forms.postmodify.elements[textFields[i]])
						document.forms.postmodify[textFields[i]].value = document.forms.postmodify[textFields[i]].value.replace(/&#/g, "&#38;#");
				for (var i = document.forms.postmodify.elements.length - 1; i >= 0; i--)
					if (document.forms.postmodify.elements[i].name.indexOf("options") == 0)
						document.forms.postmodify.elements[i].value = document.forms.postmodify.elements[i].value.replace(/&#/g, "&#38;#");
			}`)

	// Code for showing and hiding additional options.
	if !c.Theme.Empty("additional_options_collapsable") {
		c.O(`
			var currentSwap = false;
			function swapOptions()
			{
				document.getElementById("postMoreExpand").src = smf_images_url + "/" + (currentSwap ? "collapse.gif" : "expand.gif");
				document.getElementById("postMoreExpand").alt = currentSwap ? "-" : "+";

				document.getElementById("postMoreOptions").style.display = currentSwap ? "" : "none";

				if (document.getElementById("postAttachment"))
					document.getElementById("postAttachment").style.display = currentSwap ? "" : "none";
				if (document.getElementById("postAttachment2"))
					document.getElementById("postAttachment2").style.display = currentSwap ? "" : "none";

				if (typeof(document.forms.postmodify) != "undefined")
					document.forms.postmodify.additional_options.value = currentSwap ? "1" : "0";

				currentSwap = !currentSwap;
			}`)
	}

	// If this is a poll - use some javascript to ensure the user doesn't
	// create a poll with illegal option combinations.
	if page.MakePoll {
		c.O(`
			function pollOptions()
			{
				var expireTime = document.getElementById("poll_expire");

				if (isEmptyText(expireTime) || expireTime.value == 0)
				{
					document.forms.postmodify.poll_hide[2].disabled = true;
					if (document.forms.postmodify.poll_hide[2].checked)
						document.forms.postmodify.poll_hide[1].checked = true;
				}
				else
					document.forms.postmodify.poll_hide[2].disabled = false;
			}

			var pollOptionNum = 0, pollTabIndex;
			function addPollOption()
			{
				if (pollOptionNum == 0)
				{
					for (var i = 0; i < document.forms.postmodify.elements.length; i++)
						if (document.forms.postmodify.elements[i].id.substr(0, 8) == "options-")
						{
							pollOptionNum++;
							pollTabIndex = document.forms.postmodify.elements[i].tabIndex;
						}
				}
				pollOptionNum++

				setOuterHTML(document.getElementById("pollMoreOptions"), '<br /><label for="options-' + pollOptionNum + '">`, c.Txt("smf22"), ` ' + pollOptionNum + '</label>: <input type="text" name="options[' + pollOptionNum + ']" id="options-' + pollOptionNum + '" value="" size="25" tabindex="' + pollTabIndex + '" /><span id="pollMoreOptions"></span>');
			}`)
	}

	// End of the javascript, start the form and display the link tree.
	boardParam := ""
	if c.Board != 0 {
		boardParam = "board=" + itoa(c.Board)
	}
	c.O(`
		// ]]></script>

		<form action="`, scripturl, `?action=`, page.Destination, `;`, boardParam, `" method="post" accept-charset="`, c.CharacterSet, `" name="postmodify" id="postmodify" onsubmit="submitonce(this);saveEntities();" enctype="multipart/form-data" style="margin: 0;">
			<table width="100%" align="center" cellpadding="0" cellspacing="3">
				<tr>
					<td valign="bottom" colspan="2">
						`)
	c.themeLinktree()
	c.O(`
					</td>
				</tr>
			</table>`)

	// If the user wants to see how their message looks - the preview table is
	// where it's at!
	previewStyle := ` style="display: none;"`
	if page.HasPreview {
		previewStyle = ""
	}
	previewBody := strings.Repeat("<br />", 5)
	if page.PreviewMessage != "" {
		previewBody = page.PreviewMessage
	}
	c.O(`
		<div id="preview_section"`, previewStyle, `>
			<table border="0" width="100%" cellspacing="1" cellpadding="3" class="bordercolor" align="center" style="table-layout: fixed;">
				<tr class="titlebg">
					<td id="preview_subject">`, page.PreviewSubject, `</td>
				</tr>
				<tr>
					<td class="post" width="100%" id="preview_body">
						`, previewBody, `
					</td>
				</tr>
			</table><br />
		</div>`)

	// Start the main table.
	c.O(`
			<table border="0" width="100%" align="center" cellspacing="1" cellpadding="3" class="bordercolor">
				<tr class="titlebg">
					<td>`, c.PageTitle, `</td>
				</tr>
				<tr>
					<td class="windowbg">`, `
						<input type="hidden" name="topic" value="`+itoa(c.Topic)+`" />`, `
						<table border="0" cellpadding="3" width="100%">`)

	// If an error occurred, explain what happened.
	errorsStyle := ` style="display: none"`
	if len(page.ErrorMessages) > 0 {
		errorsStyle = ""
	}
	seriousStyle := ` display: none;`
	if page.ErrorType == "serious" {
		seriousStyle = ""
	}
	c.O(`
							<tr`, errorsStyle, ` id="errors">
								<td></td>
								<td align="left">
									<div style="padding: 0px; font-weight: bold;`, seriousStyle, `" id="error_serious">
										`, c.Txt("error_while_submitting"), `
									</div>
									<div style="color: red; margin: 1ex 0 2ex 3ex;" id="error_list">
										`, strings.Join(page.ErrorMessages, "<br />"), `
									</div>
								</td>
							</tr>`)

	// If it's locked, show a message to warn the replyer.
	lockStyle := ` style="display: none"`
	if page.Locked {
		lockStyle = ""
	}
	c.O(`
							<tr`, lockStyle, ` id="lock_warning">
								<td></td>
								<td align="left">
									`, c.Txt("smf287"), `
								</td>
							</tr>`)

	// Guests have to put in their name and email...
	if page.ShowGuestName {
		nameColor := ""
		if page.PostError["long_name"] || page.PostError["no_name"] || page.PostError["bad_name"] {
			nameColor = "color: red;"
		}
		c.O(`
							<tr>
								<td align="right" style="font-weight: bold;`, nameColor, `" id="caption_guestname">
									`, c.Txt("68"), `:
								</td>
								<td>
									<input type="text" name="guestname" size="25" value="`, page.GuestName, `" tabindex="`, nextTab(), `" />
								</td>
							</tr>`)

		if c.App.SettingEmpty("guest_post_no_email") {
			emailColor := ""
			if page.PostError["no_email"] || page.PostError["bad_email"] {
				emailColor = "color: red;"
			}
			c.O(`
							<tr>
								<td align="right" style="font-weight: bold;`, emailColor, `" id="caption_email">
									`, c.Txt("69"), `:
								</td>
								<td>
									<input type="text" name="email" size="25" value="`, page.GuestEmail, `" tabindex="`, nextTab(), `" />
								</td>
							</tr>`)
		}
	}

	// Now show the subject box for this post.
	subjColor := ""
	if page.PostError["no_subject"] {
		subjColor = "color: red;"
	}
	subjValue := ""
	if page.Subject != "" {
		subjValue = ` value="` + page.Subject + `"`
	}
	c.O(`
							<tr>
								<td align="right" style="font-weight: bold;`, subjColor, `" id="caption_subject">
									`, c.Txt("70"), `:
								</td>
								<td>
									<input type="text" name="subject"`, subjValue, ` tabindex="`, nextTab(), `" size="80" maxlength="80" />
								</td>
							</tr>
							<tr>
								<td align="right">
									<b>`, c.Txt("71"), `:</b>
								</td>
								<td>
									<select name="icon" id="icon" onchange="showimage()">`)

	// Loop through each message icon allowed, adding it to the drop down
	// list.
	for _, icon := range page.Icons {
		sel := ""
		if icon.Value == page.Icon {
			sel = ` selected="selected"`
		}
		c.O(`
										<option value="`, icon.Value, `"`, sel, `>`, icon.Name, `</option>`)
	}

	c.O(`
									</select>
									<img src="`, page.IconURL, `" name="icons" hspace="15" alt="" />
								</td>
							</tr>`)

	// If this is a poll then display all the poll options!
	if page.MakePoll {
		questionColor := ""
		if page.PostError["no_question"] {
			questionColor = "color: red;"
		}
		c.O(`
							<tr>
								<td align="right" style="font-weight: bold;`, questionColor, `" id="caption_question">
									`, c.Txt("smf21"), `:
								</td>
								<td align="left">
									<input type="text" name="question" value="`, page.PollQuestion, `" tabindex="`, nextTab(), `" size="80" />
								</td>
							</tr>
							<tr>
								<td align="right"></td>
								<td>`)

		// Loop through all the choices and print them out.
		for _, choice := range page.Choices {
			c.O(`
									<label for="options-`, choice.ID, `">`, c.Txt("smf22"), ` `, choice.Number, `</label>: <input type="text" name="options[`, choice.ID, `]" id="options-`, choice.ID, `" value="`, choice.Label, `" tabindex="`, nextTab(), `" size="25" />`)

			if !choice.IsLast {
				c.O(`<br />`)
			}
		}

		changeVoteChecked := ""
		if page.PollChangeVote {
			changeVoteChecked = ` checked="checked"`
		}
		hideChecked := func(v int) string {
			if page.PollHide == v {
				return ` checked="checked"`
			}
			return ""
		}
		expireDisabled := ""
		if empty(page.PollExpire) {
			expireDisabled = ` disabled="disabled"`
		}
		c.O(`
									<span id="pollMoreOptions"></span> <a href="javascript:addPollOption(); void(0);">(`, c.Txt("poll_add_option"), `)</a>
								</td>
							</tr>
							<tr>
								<td align="right"><b>`, c.Txt("poll_options"), `:</b></td>
								<td class="smalltext"><input type="text" name="poll_max_votes" size="2" value="`, page.PollMaxVotes, `" /> `, c.Txt("poll_options5"), `</td>
							</tr>
							<tr>
								<td align="right"></td>
								<td class="smalltext">`, c.Txt("poll_options1a"), ` <input type="text" id="poll_expire" name="poll_expire" size="2" value="`, page.PollExpire, `" onchange="pollOptions();" /> `, c.Txt("poll_options1b"), `</td>
							</tr>
							<tr>
								<td align="right"></td>
								<td class="smalltext"><label for="poll_change_vote"><input type="checkbox" id="poll_change_vote" name="poll_change_vote"`, changeVoteChecked, ` class="check" /> `, c.Txt("poll_options7"), `</label></td>
							</tr>
							<tr>
								<td align="right"></td>
								<td class="smalltext">
									<input type="radio" id="poll_hide" name="poll_hide" value="0"`, hideChecked(0), ` class="check" /> `, c.Txt("poll_options2"), `<br />
									<input type="radio" id="poll_hide" name="poll_hide" value="1"`, hideChecked(1), ` class="check" /> `, c.Txt("poll_options3"), `<br />
									<input type="radio" id="poll_hide" name="poll_hide" value="2"`, hideChecked(2), expireDisabled, ` class="check" /> `, c.Txt("poll_options4"), `<br />
									<br />
								</td>
							</tr>`)
	}

	// The below function prints the BBC, smileys and the message itself out.
	c.themePostbox(page.Message, postboxOpts{
		postError: page.PostError,
		nextTab:   nextTab,
	})

	// If this message has been edited in the past - display when it was.
	if page.LastModified != "" {
		c.O(`
									<tr>
										<td valign="top" align="right">
											<b>`, c.Txt("211"), `:</b>
										</td>
										<td>
											`, page.LastModified, `
										</td>
									</tr>`)
	}

	// If the admin has enabled the hiding of the additional options - show a
	// link and image for it.
	if !c.Theme.Empty("additional_options_collapsable") {
		c.O(`
									<tr>
										<td colspan="2" style="padding-left: 5ex;">
											<a href="javascript:swapOptions();"><img src="`, c.Theme.ImagesURL(), `/expand.gif" alt="+" id="postMoreExpand" /></a> <a href="javascript:swapOptions();"><b>`, c.Txt("post_additionalopt"), `</b></a>
										</td>
									</tr>`)
	}

	// Display the check boxes for all the standard options - if they are
	// available to the user!
	notifyBox := ""
	if page.CanNotify {
		checked := ""
		if page.Notify || !empty(c.Options["auto_notify"]) {
			checked = ` checked="checked"`
		}
		notifyBox = `<input type="hidden" name="notify" value="0" /><label for="check_notify"><input type="checkbox" name="notify" id="check_notify"` + checked + ` value="1" class="check" /> ` + c.Txt("smf14") + `</label>`
	}
	lockBox := ""
	if page.CanLock {
		checked := ""
		if page.Locked {
			checked = ` checked="checked"`
		}
		lockBox = `<input type="hidden" name="lock" value="0" /><label for="check_lock"><input type="checkbox" name="lock" id="check_lock"` + checked + ` value="1" class="check" /> ` + c.Txt("smf15") + `</label>`
	}
	backChecked := ""
	if page.BackToTopic || !empty(c.Options["return_to_post"]) {
		backChecked = ` checked="checked"`
	}
	stickyBox := ""
	if page.CanSticky {
		checked := ""
		if page.Sticky {
			checked = ` checked="checked"`
		}
		stickyBox = `<input type="hidden" name="sticky" value="0" /><label for="check_sticky"><input type="checkbox" name="sticky" id="check_sticky"` + checked + ` value="1" class="check" /> ` + c.Txt("sticky_after2") + `</label>`
	}
	smileysChecked := ""
	if !page.UseSmileys {
		smileysChecked = ` checked="checked"`
	}
	moveBox := ""
	if page.CanMove {
		moveBox = `<input type="hidden" name="move" value="0" /><label for="check_move"><input type="checkbox" name="move" id="check_move" value="1" class="check" /> ` + c.Txt("move_after2") + `</label>`
	}
	announceRow := ""
	if page.CanAnnounce && page.IsFirstPost {
		announceRow = `
													<tr>
														<td class="smalltext"><label for="check_announce"><input type="checkbox" name="announce_topic" id="check_announce" value="1" class="check" /> ` + c.Txt("announce_topic") + `</label></td>
														<td class="smalltext"></td>
													</tr>`
	}
	c.O(`
									<tr>
										<td></td>
										<td>
											<div id="postMoreOptions">
												<table width="80%" cellpadding="0" cellspacing="0" border="0">
													<tr>
														<td class="smalltext">`, notifyBox, `</td>
														<td class="smalltext">`, lockBox, `</td>
													</tr>
													<tr>
														<td class="smalltext"><label for="check_back"><input type="checkbox" name="goback" id="check_back"`+backChecked+` value="1" class="check" /> `+c.Txt("back_to_topic")+`</label></td>
														<td class="smalltext">`, stickyBox, `</td>
													</tr>
													<tr>
														<td class="smalltext"><label for="check_smileys"><input type="checkbox" name="ns" id="check_smileys"`, smileysChecked, ` value="NS" class="check" /> `, c.Txt("277"), `</label></td>`, `
														<td class="smalltext">`, moveBox, `</td>
													</tr>`, announceRow, `
												</table>
											</div>
										</td>
									</tr>`)

	// If this post already has attachments on it - give information about
	// them.
	if len(page.CurrentAttachments) > 0 {
		c.O(`
							<tr id="postAttachment">
								<td align="right" valign="top">
									<b>`, c.Txt("smf119b"), `:</b>
								</td>
								<td class="smalltext">
									<input type="hidden" name="attach_del[]" value="0" />
									`, c.Txt("smf130"), `:<br />`)
		for _, attachment := range page.CurrentAttachments {
			checked := ` checked="checked"`
			if attachment.Unchecked {
				checked = ""
			}
			c.O(`
									<input type="checkbox" name="attach_del[]" value="`, attachment.ID, `"`, checked, ` class="check" /> `, attachment.Name, `<br />`)
		}
		c.O(`
									<br />
								</td>
							</tr>`)
	}

	// Is the user allowed to post any additional ones? If so give them the
	// boxes to do it!
	if page.CanPostAttachment {
		c.O(`
							<tr id="postAttachment2">
								<td align="right" valign="top">
									<b>`, c.Txt("smf119"), `:</b>
								</td>
								<td class="smalltext">
									<input type="file" size="48" name="attachment[]" />`)

		// Show more boxes only if they aren't approaching their limit.
		if page.NumAllowedAttachments > 1 {
			c.O(`
									<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
										var allowed_attachments = `, page.NumAllowedAttachments, ` - 1;

										function addAttachment()
										{
											if (allowed_attachments <= 0)
												return alert("`, c.Txt("more_attachments_error"), `");

											setOuterHTML(document.getElementById("moreAttachments"), '<br /><input type="file" size="48" name="attachment[]" /><span id="moreAttachments"></span>');
											allowed_attachments = allowed_attachments - 1;

											return true;
										}
									// ]]></script>
									<span id="moreAttachments"></span> <a href="javascript:addAttachment(); void(0);">(`, c.Txt("more_attachments"), `)</a><br />
									<noscript><input type="file" size="48" name="attachment[]" /><br /></noscript>`)
		} else {
			c.O(`
									<br />`)
		}

		// Show some useful information such as allowed extensions, maximum
		// size and amount of attachments allowed.
		if !c.App.SettingEmpty("attachmentCheckExtensions") {
			c.O(`
									`, c.Txt("smf120"), `: `, page.AllowedExtensions, `<br />`)
		}
		perPost := ""
		if !c.App.SettingEmpty("attachmentNumPerPostLimit") {
			perPost = `, ` + c.Txt("maxAttachPerPost") + `: ` + c.App.Setting("attachmentNumPerPostLimit")
		}
		c.O(`
									`, c.Txt("smf121"), `: `, c.App.Setting("attachmentSizeLimit"), ` `+c.Txt("smf211"), perPost, `
								</td>
							</tr>`)
	}

	// Finally, the submit buttons.
	c.O(`
							<tr>
								<td align="center" colspan="2">
									<span class="smalltext"><br />`, c.Txt("smf16"), `</span><br />
									<input type="submit" name="post" value="`, page.SubmitLabel, `" tabindex="`, nextTab(), `" onclick="return submitThisOnce(this);" accesskey="s" />
									<input type="submit" name="preview" value="`, c.Txt("507"), `" tabindex="`, nextTab(), `" onclick="return event.ctrlKey || previewPost();" accesskey="p" />`)

	// Spell check button if the option is enabled.
	if page.ShowSpellchecking {
		c.O(`
									<input type="button" value="`, c.Txt("spell_check"), `" tabindex="`, nextTab(), `" onclick="spellCheck('postmodify', 'message');" />`)
	}

	c.O(`
								</td>
							</tr>
							<tr>
								<td colspan="2"></td>
							</tr>
						</table>
					</td>
				</tr>
			</table>`)

	// Assuming this isn't a new topic pass across the number of replies when
	// the topic was created.
	if c.Topic != 0 {
		c.O(`
			<input type="hidden" name="num_replies" value="`, page.NumReplies, `" />`)
	}

	additionalOptions := 0
	if page.ShowAdditionalOptions {
		additionalOptions = 1
	}
	c.O(`
			<input type="hidden" name="additional_options" value="`, additionalOptions, `" />
			<input type="hidden" name="sc" value="`, c.Sc, `" />
			<input type="hidden" name="seqnum" value="`, c.FormSequenceNumber, `" />
		</form>`)

	// Now some javascript to hide the additional options on load...
	if !c.Theme.Empty("additional_options_collapsable") && !page.ShowAdditionalOptions {
		c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			swapOptions();
		// ]]></script>`)
	}

	// A hidden form to post data to the spell checking window.
	if page.ShowSpellchecking {
		c.O(`
		<form action="`, scripturl, `?action=spellcheck" method="post" accept-charset="`, c.CharacterSet, `" name="spell_form" id="spell_form" target="spellWindow">
			<input type="hidden" name="spellstring" value="" />
		</form>`)
	}

	// If the user is replying to a topic show the previous posts.
	if len(page.PreviousPosts) > 0 {
		c.O(`
			<br />
			<br />

			<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
				function insertQuoteFast(messageid)
				{
					if (window.XMLHttpRequest)
						getXMLDocument("`, scripturl, `?action=quotefast;quote=" + messageid + ";sesc=`, c.Sc, `;xml", onDocReceived);
					else
						reqWin("`, scripturl, `?action=quotefast;quote=" + messageid + ";sesc=`, c.Sc, `", 240, 90);

					return true;
				}
				function onDocReceived(XMLDoc)
				{
					var text = "";
					for (var i = 0; i < XMLDoc.getElementsByTagName("quote")[0].childNodes.length; i++)
						text += XMLDoc.getElementsByTagName("quote")[0].childNodes[i].nodeValue;

					replaceText(text, document.forms.postmodify.message);
				}
			// ]]></script>

			<table cellspacing="1" cellpadding="0" width="92%" align="center" class="bordercolor">
				<tr>
					<td>
						<table width="100%" class="windowbg" cellspacing="0" cellpadding="2" align="center">
							<tr class="titlebg">
								<td colspan="2">`, c.Txt("468"), `</td>
							</tr>
						</table>
						<span id="new_replies"></span>
						<table width="100%" class="windowbg" cellspacing="0" cellpadding="2" align="center" style="table-layout: fixed;">`)
		for _, post := range page.PreviousPosts {
			newImg := ""
			if post.IsNew {
				newImg = ` <img src="` + c.Theme.ImagesURL() + `/` + c.User.Language + `/new.gif" alt="` + c.Txt("preview_new") + `" />`
			}
			c.O(`
							<tr class="catbg">
								<td colspan="2" align="left" class="smalltext">
									<div style="float: right;">`, c.Txt("280"), `: `, post.Time, newImg, `</div>
									`, c.Txt("279"), `: `, post.Poster, `
								</td>
							</tr><tr class="windowbg2">
								<td colspan="2" class="smalltext" id="msg`, post.ID, `" width="100%">
									<div align="right" class="smalltext"><a href="#top" onclick="return insertQuoteFast(`, post.ID, `);">`, c.Txt("260"), `</a></div>
									<div class="post">`, post.Message, `</div>
								</td>
							</tr>`)
		}
		c.O(`
						</table>
					</td>
				</tr>
			</table>`)
	}
}

// postboxOpts carries the overridable theme_postbox() context values.
type postboxOpts struct {
	form      string // $context['post_form'], default 'postmodify'
	name      string // $context['post_box_name'], default 'message'
	rows      int    // default 12
	columns   int    // default 60
	postError map[string]bool
	nextTab   func() int
}

// pbSmiley is one smiley button.
type pbSmiley struct {
	Code          string
	Filename      string
	Description   string
	JsDescription string
	Last          bool
}

// pbSmileyRow is one row of smiley buttons.
type pbSmileyRow struct {
	Smileys []pbSmiley
	Last    bool
}

// addslashesJS is PHP addslashes(): escape ', " and \.
func addslashesJS(s string) string {
	return strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`).Replace(s)
}

// themePostbox is theme_postbox() from Subs-Post.php: prepare smileys and
// flags, then print the postbox.
func (c *Ctx) themePostbox(msg string, o postboxOpts) {
	a := c.App

	// Load the Post language file.
	c.loadLanguage("Post")

	// Initialize smiley array...
	var postform, popup []pbSmileyRow

	// Load smileys - don't bother to run a query if we're not using the
	// database's ones anyhow.
	if a.SettingEmpty("smiley_enable") && c.User.SmileySet != "none" {
		postform = []pbSmileyRow{{
			Smileys: []pbSmiley{
				{Code: ":)", Filename: "smiley.gif", Description: c.Txt("287")},
				{Code: ";)", Filename: "wink.gif", Description: c.Txt("292")},
				{Code: ":D", Filename: "cheesy.gif", Description: c.Txt("289")},
				{Code: ";D", Filename: "grin.gif", Description: c.Txt("293")},
				{Code: ">:(", Filename: "angry.gif", Description: c.Txt("288")},
				{Code: ":(", Filename: "sad.gif", Description: c.Txt("291")},
				{Code: ":o", Filename: "shocked.gif", Description: c.Txt("294")},
				{Code: "8)", Filename: "cool.gif", Description: c.Txt("295")},
				{Code: "???", Filename: "huh.gif", Description: c.Txt("296")},
				{Code: "::)", Filename: "rolleyes.gif", Description: c.Txt("450")},
				{Code: ":P", Filename: "tongue.gif", Description: c.Txt("451")},
				{Code: ":-[", Filename: "embarrassed.gif", Description: c.Txt("526")},
				{Code: ":-X", Filename: "lipsrsealed.gif", Description: c.Txt("527")},
				{Code: `:-\`, Filename: "undecided.gif", Description: c.Txt("528")},
				{Code: ":-*", Filename: "kiss.gif", Description: c.Txt("529")},
				{Code: ":'(", Filename: "cry.gif", Description: c.Txt("530")},
			},
			Last: true,
		}}
	} else if c.User.SmileySet != "none" {
		rowIndex := map[string]map[int]int{"postform": {}, "popup": {}}
		rows, err := a.DB.Query(a.Q(`
			SELECT code, filename, description, smileyRow, hidden
			FROM {$db_prefix}smileys
			WHERE hidden IN (0, 2)
			ORDER BY smileyRow, smileyOrder`))
		if err == nil {
			for rows.Next() {
				var code, filename, description string
				var smileyRow, hidden int
				rows.Scan(&code, &filename, &description, &smileyRow, &hidden)

				sm := pbSmiley{
					Code:        Htmlspecialchars(code),
					Filename:    Htmlspecialchars(filename),
					Description: Htmlspecialchars(description),
				}
				location := "postform"
				target := &postform
				if hidden != 0 {
					location = "popup"
					target = &popup
				}
				idx, ok := rowIndex[location][smileyRow]
				if !ok {
					*target = append(*target, pbSmileyRow{})
					idx = len(*target) - 1
					rowIndex[location][smileyRow] = idx
				}
				(*target)[idx].Smileys = append((*target)[idx].Smileys, sm)
			}
			rows.Close()
		}
	}

	// Clean house... add slashes to the code for javascript.
	for _, location := range []*[]pbSmileyRow{&postform, &popup} {
		for j := range *location {
			smileys := (*location)[j].Smileys
			for i := range smileys {
				smileys[i].Code = addslashesJS(smileys[i].Code)
				smileys[i].JsDescription = addslashesJS(smileys[i].Description)
			}
			if len(smileys) > 0 {
				smileys[len(smileys)-1].Last = true
			}
		}
		if len(*location) > 0 {
			(*location)[len(*location)-1].Last = true
		}
	}
	smileysURL := a.Setting("smileys_url") + "/" + c.User.SmileySet

	// Allow for things to be overridden.
	if o.columns == 0 {
		o.columns = 60
	}
	if o.rows == 0 {
		o.rows = 12
	}
	if o.name == "" {
		o.name = "message"
	}
	if o.form == "" {
		o.form = "postmodify"
	}

	// Set a flag so the sub template knows what to do...
	showBBC := !a.SettingEmpty("enableBBC") && !c.Theme.Empty("show_bbc")

	// Generate a list of buttons that shouldn't be shown.
	disabledTags := map[string]bool{}
	if !a.SettingEmpty("disabledBBC") {
		for _, tag := range strings.Split(a.Setting("disabledBBC"), ",") {
			disabledTags[strings.TrimSpace(tag)] = true
		}
	}

	// Go!  Supa-sub-template-smash!
	templatePostbox(c, msg, o, showBBC, disabledTags, postform, popup, smileysURL)
}

// pbTag is one BBC button definition (or a divider when Image is empty).
type pbTag struct {
	Image  string
	Code   string
	Before string
	After  string
	Desc   string
}

// templatePostbox is template_postbox() from Post.template.php.
func templatePostbox(c *Ctx, msg string, o postboxOpts, showBBC bool, disabledTags map[string]bool, postform, popup []pbSmileyRow, smileysURL string) {
	imagesURL := c.Theme.ImagesURL()

	// Assuming BBC code is enabled then print the buttons and some javascript
	// to handle it.
	if showBBC {
		c.O(`
			<tr>
				<td align="right"></td>
				<td valign="middle">
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						function bbc_highlight(something, mode)
						{
							something.style.backgroundImage = "url(" + smf_images_url + (mode ? "/bbc/bbc_hoverbg.gif)" : "/bbc/bbc_bg.gif)");
						}
					// ]]></script>`)

		bbcRows := [2][]pbTag{
			{
				{Image: "bold", Code: "b", Before: "[b]", After: "[/b]", Desc: c.Txt("253")},
				{Image: "italicize", Code: "i", Before: "[i]", After: "[/i]", Desc: c.Txt("254")},
				{Image: "underline", Code: "u", Before: "[u]", After: "[/u]", Desc: c.Txt("255")},
				{Image: "strike", Code: "s", Before: "[s]", After: "[/s]", Desc: c.Txt("441")},
				{},
				{Image: "glow", Code: "glow", Before: "[glow=red,2,300]", After: "[/glow]", Desc: c.Txt("442")},
				{Image: "shadow", Code: "shadow", Before: "[shadow=red,left]", After: "[/shadow]", Desc: c.Txt("443")},
				{Image: "move", Code: "move", Before: "[move]", After: "[/move]", Desc: c.Txt("439")},
				{},
				{Image: "pre", Code: "pre", Before: "[pre]", After: "[/pre]", Desc: c.Txt("444")},
				{Image: "left", Code: "left", Before: "[left]", After: "[/left]", Desc: c.Txt("445")},
				{Image: "center", Code: "center", Before: "[center]", After: "[/center]", Desc: c.Txt("256")},
				{Image: "right", Code: "right", Before: "[right]", After: "[/right]", Desc: c.Txt("446")},
				{},
				{Image: "hr", Code: "hr", Before: "[hr]", Desc: c.Txt("531")},
				{},
				{Image: "size", Code: "size", Before: "[size=10pt]", After: "[/size]", Desc: c.Txt("532")},
				{Image: "face", Code: "font", Before: "[font=Verdana]", After: "[/font]", Desc: c.Txt("533")},
			},
			{
				{Image: "flash", Code: "flash", Before: "[flash=200,200]", After: "[/flash]", Desc: c.Txt("433")},
				{Image: "img", Code: "img", Before: "[img]", After: "[/img]", Desc: c.Txt("435")},
				{Image: "url", Code: "url", Before: "[url]", After: "[/url]", Desc: c.Txt("257")},
				{Image: "email", Code: "email", Before: "[email]", After: "[/email]", Desc: c.Txt("258")},
				{Image: "ftp", Code: "ftp", Before: "[ftp]", After: "[/ftp]", Desc: c.Txt("434")},
				{},
				{Image: "table", Code: "table", Before: "[table]", After: "[/table]", Desc: c.Txt("436")},
				{Image: "tr", Code: "td", Before: "[tr]", After: "[/tr]", Desc: c.Txt("449")},
				{Image: "td", Code: "td", Before: "[td]", After: "[/td]", Desc: c.Txt("437")},
				{},
				{Image: "sup", Code: "sup", Before: "[sup]", After: "[/sup]", Desc: c.Txt("447")},
				{Image: "sub", Code: "sub", Before: "[sub]", After: "[/sub]", Desc: c.Txt("448")},
				{Image: "tele", Code: "tt", Before: "[tt]", After: "[/tt]", Desc: c.Txt("440")},
				{},
				{Image: "code", Code: "code", Before: "[code]", After: "[/code]", Desc: c.Txt("259")},
				{Image: "quote", Code: "quote", Before: "[quote]", After: "[/quote]", Desc: c.Txt("260")},
				{},
				{Image: "list", Code: "list", Before: `[list]\n[li]`, After: `[/li]\n[li][/li]\n[/list]`, Desc: c.Txt("261")},
			},
		}

		printRow := func(tags []pbTag) {
			foundButton := false
			for _, tag := range tags {
				if tag.Before != "" {
					// Is this tag disabled?
					if disabledTags[tag.Code] {
						continue
					}

					foundButton = true

					// If there's no after, we're just replacing the entire
					// selection in the post box.
					if tag.After == "" {
						c.O(`<a href="javascript:void(0);" onclick="replaceText('`, tag.Before, `', document.forms.`, o.form, `.`, o.name, `); return false;">`)
					} else {
						// On the other hand, if there is one we are
						// surrounding the selection ;).
						c.O(`<a href="javascript:void(0);" onclick="surroundText('`, tag.Before, `', '`, tag.After, `', document.forms.`, o.form, `.`, o.name, `); return false;">`)
					}

					// Okay... we have the link. Now for the image and the
					// closing </a>!
					c.O(`<img onmouseover="bbc_highlight(this, true);" onmouseout="if (window.bbc_highlight) bbc_highlight(this, false);" src="`, imagesURL, `/bbc/`, tag.Image, `.gif" align="bottom" width="23" height="22" alt="`, tag.Desc, `" title="`, tag.Desc, `" style="background-image: url(`, imagesURL, `/bbc/bbc_bg.gif); margin: 1px 2px 1px 1px;" /></a>`)
				} else if foundButton {
					// I guess it's a divider...
					c.O(`<img src="`, imagesURL, `/bbc/divider.gif" alt="|" style="margin: 0 3px 0 3px;" />`)
					foundButton = false
				}
			}
		}

		printRow(bbcRows[0])

		// Print a drop down list for all the colors we allow!
		if !disabledTags["color"] {
			c.O(` <select onchange="surroundText('[color=' + this.options[this.selectedIndex].value.toLowerCase() + ']', '[/color]', document.forms.`, o.form, `.`, o.name, `); this.selectedIndex = 0; document.forms.`, o.form, `.`, o.name, `.focus(document.forms.`, o.form, `.`, o.name, `.caretPos);" style="margin-bottom: 1ex;">
							<option value="" selected="selected">`, c.Txt("change_color"), `</option>
							<option value="Black">`, c.Txt("262"), `</option>
							<option value="Red">`, c.Txt("263"), `</option>
							<option value="Yellow">`, c.Txt("264"), `</option>
							<option value="Pink">`, c.Txt("265"), `</option>
							<option value="Green">`, c.Txt("266"), `</option>
							<option value="Orange">`, c.Txt("267"), `</option>
							<option value="Purple">`, c.Txt("268"), `</option>
							<option value="Blue">`, c.Txt("269"), `</option>
							<option value="Beige">`, c.Txt("270"), `</option>
							<option value="Brown">`, c.Txt("271"), `</option>
							<option value="Teal">`, c.Txt("272"), `</option>
							<option value="Navy">`, c.Txt("273"), `</option>
							<option value="Maroon">`, c.Txt("274"), `</option>
							<option value="LimeGreen">`, c.Txt("275"), `</option>
						</select>`)
		}
		c.O(`<br />`)

		printRow(bbcRows[1])

		c.O(`
				</td>
			</tr>`)
	}

	// Now start printing all of the smileys.
	if len(postform) > 0 {
		c.O(`
			<tr>
				<td align="right"></td>
				<td valign="middle">`)

		// Show each row of smileys ;).
		for _, smileyRow := range postform {
			for _, smiley := range smileyRow.Smileys {
				c.O(`
					<a href="javascript:void(0);" onclick="replaceText(' `, smiley.Code, `', document.forms.`, o.form, `.`, o.name, `); return false;"><img src="`, smileysURL, `/`, smiley.Filename, `" align="bottom" alt="`, smiley.Description, `" title="`, smiley.Description, `" /></a>`)
			}

			// If this isn't the last row, show a break.
			if !smileyRow.Last {
				c.O(`<br />`)
			}
		}

		// If the smileys popup is to be shown... show it!
		if len(popup) > 0 {
			c.O(`
					<a href="javascript:moreSmileys();">[`, c.Txt("more_smileys"), `]</a>`)
		}

		c.O(`
				</td>
			</tr>`)
	}

	// If there are additional smileys then ensure we provide the javascript
	// for them.
	if len(popup) > 0 {
		c.O(`
			<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
				var smileys = [`)

		for _, smileyRow := range popup {
			c.O(`
					[`)
			for _, smiley := range smileyRow.Smileys {
				c.O(`
						["`, smiley.Code, `","`, smiley.Filename, `","`, smiley.JsDescription, `"]`)
				if !smiley.Last {
					c.O(`,`)
				}
			}

			c.O(`]`)
			if !smileyRow.Last {
				c.O(`,`)
			}
		}

		c.O(`];
				var smileyPopupWindow;

				function moreSmileys()
				{
					var row, i;

					if (smileyPopupWindow)
						smileyPopupWindow.close();

					smileyPopupWindow = window.open("", "add_smileys", "toolbar=no,location=no,status=no,menubar=no,scrollbars=yes,width=480,height=220,resizable=yes");
					smileyPopupWindow.document.write('<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">\n<html>');
					smileyPopupWindow.document.write('\n\t<head>\n\t\t<title>`, c.Txt("more_smileys_title"), `</title>\n\t\t<link rel="stylesheet" type="text/css" href="`, c.Theme.ThemeURL(), `/style.css" />\n\t</head>');
					smileyPopupWindow.document.write('\n\t<body style="margin: 1ex;">\n\t\t<table width="100%" cellpadding="5" cellspacing="0" border="0" class="tborder">\n\t\t\t<tr class="titlebg"><td align="left">`, c.Txt("more_smileys_pick"), `</td></tr>\n\t\t\t<tr class="windowbg"><td align="left">');

					for (row = 0; row < smileys.length; row++)
					{
						for (i = 0; i < smileys[row].length; i++)
						{
							smileys[row][i][2] = smileys[row][i][2].replace(/"/g, '&quot;');
							smileyPopupWindow.document.write('<a href="javascript:void(0);" onclick="window.opener.replaceText(&quot; ' + smileys[row][i][0] + '&quot;, window.opener.document.forms.`, o.form, `.`, o.name, `); window.focus(); return false;"><img src="`, smileysURL, `/' + smileys[row][i][1] + '" alt="' + smileys[row][i][2] + '" title="' + smileys[row][i][2] + '" style="padding: 4px;" border="0" /></a> ');
						}
						smileyPopupWindow.document.write("<br />");
					}

					smileyPopupWindow.document.write('</td></tr>\n\t\t\t<tr><td align="center" class="windowbg"><a href="javascript:window.close();\\">`, c.Txt("more_smileys_close_window"), `</a></td></tr>\n\t\t</table>\n\t</body>\n</html>');
					smileyPopupWindow.document.close();
				}
			// ]]></script>`)
	}

	// Finally the most important bit - the actual text box to write in!
	msgErrStyle := ""
	if o.postError["no_message"] || o.postError["long_message"] {
		msgErrStyle = ` style="border: 1px solid red;"`
	}
	c.O(`
			<tr>
				<td valign="top" align="right"></td>
				<td>
					<textarea class="editor" name="`, o.name, `" rows="`, o.rows, `" cols="`, o.columns, `" onselect="storeCaret(this);" onclick="storeCaret(this);" onkeyup="storeCaret(this);" onchange="storeCaret(this);" tabindex="`, o.nextTab(), `"`, msgErrStyle, `>`, msg, `</textarea>
				</td>
			</tr>`)
}
