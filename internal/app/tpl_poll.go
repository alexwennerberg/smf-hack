package app

// Hand-port of Themes/default/Poll.template.php template_main().

import "strings"

// templatePollMain is template_main() from Poll.template.php.
func templatePollMain(c *Ctx) {
	page := c.Page.(*EditPollCtx)
	scripturl := c.App.ScriptURL

	noQuestionStyle := ""
	if page.PollError["no_question"] {
		noQuestionStyle = ` style="color: red;"`
	}

	// Some javascript for adding more options.
	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var pollOptionNum = 0;

			function addPollOption()
			{
				if (pollOptionNum == 0)
				{
					for (var i = 0; i < document.forms.postmodify.elements.length; i++)
						if (document.forms.postmodify.elements[i].id.substr(0, 8) == "options-")
							pollOptionNum++;
				}
				pollOptionNum++

				setOuterHTML(document.getElementById("pollMoreOptions"), '<br /><label for="options-' + pollOptionNum + '" `, noQuestionStyle, `>`, c.Txt("smf22"), ` ' + pollOptionNum + '</label>: <input type="text" name="options[' + (pollOptionNum - 1) + ']" id="options-' + (pollOptionNum - 1) + '" value="" size="25" /><span id="pollMoreOptions"></span>');
			}

			function saveEntities()
			{
				document.forms.postmodify.question.value = document.forms.postmodify.question.value.replace(/&#/g, "&#38;#");
				for (i in document.forms.postmodify)
					if (document.forms.postmodify[i].name.indexOf("options") == 0)
						document.forms.postmodify[i].value = document.forms.postmodify[i].value.replace(/&#/g, "&#38;#");
			}
		// ]]></script>`)

	addParam := ";add"
	if page.IsEdit {
		addParam = ""
	}

	// Start the main poll form.
	c.O(`
		<form action="`+scripturl+`?action=editpoll2`, addParam, `;topic=`+itoa(c.Topic)+`.`+itoa(page.Start)+`" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="submitonce(this);saveEntities();" name="postmodify" id="postmodify">
			<table width="75%" align="center" cellpadding="3" cellspacing="0">
				<tr>
					<td valign="bottom" colspan="2">`)
	c.themeLinktree()
	c.O(`</td>
				</tr>
			</table>
			<table border="0"  width="75%" align="center" cellspacing="1" cellpadding="3" class="bordercolor">
				<tr class="titlebg">
					<td>` + c.PageTitle + `</td>
				</tr><tr>
					<td class="windowbg">
						<input type="hidden" name="poll" value="` + itoa(page.PollID) + `" />
						<table border="0" cellpadding="3" width="100%">`)

	if len(page.ErrorMessages) > 0 {
		errTitle := c.Txt("error_while_adding_poll")
		if page.IsEdit {
			errTitle = c.Txt("error_while_editing_poll")
		}
		c.O(`
							<tr>
								<td></td>
								<td align="left">
									<div style="padding: 0px; font-weight: bold;">
										`, errTitle, `:
									</div>
									<div style="color: red; margin: 1ex 0 2ex 3ex;">
										`, strings.Join(page.ErrorMessages, "<br />"), `
									</div>
								</td>
							</tr>`)
	}

	c.O(`
							<tr>
								<td align="right" `, noQuestionStyle, `><b>`+c.Txt("smf21")+`:</b></td>
								<td align="left"><input type="text" name="question" size="40" value="`+page.Question+`" /></td>
							</tr><tr>
								<td></td>
								<td>`)

	pollFewStyle := ""
	if page.PollError["poll_few"] {
		pollFewStyle = ` style="color: red;"`
	}
	for _, choice := range page.Choices {
		c.O(`
									<label for="options-`, choice.ID, `" `, pollFewStyle, `>`, c.Txt("smf22"), ` `, choice.Number, `</label>: <input type="text" name="options[`, choice.ID, `]" id="options-`, choice.ID, `" size="25" value="`, choice.Label, `" />`)

		// Does this option have a vote count yet, or is it new?
		if choice.Votes != -1 {
			c.O(` (`, choice.Votes, ` `, c.Txt("smf42"), `)`)
		}

		if !choice.IsLast {
			c.O(`<br />`)
		}
	}

	c.O(`
									<span id="pollMoreOptions"></span> <a href="javascript:addPollOption(); void(0);">(`, c.Txt("poll_add_option"), `)</a>
								</td>
							</tr><tr>`)

	if page.CanModeratePoll {
		changeVoteChecked := ""
		if page.ChangeVote {
			changeVoteChecked = ` checked="checked"`
		}
		c.O(`
								<td align="right"><b>`, c.Txt("poll_options"), `:</b></td>
								<td class="smalltext"><input type="text" name="poll_max_votes" size="2" value="`, page.MaxVotes, `" /> `, c.Txt("poll_options5"), `</td>
							</tr><tr>
								<td align="right"></td>
								<td class="smalltext">`, c.Txt("poll_options1a"), ` <input type="text" name="poll_expire" size="2" value="`, page.Expiration, `" onchange="this.form.poll_hide[2].disabled = isEmptyText(this) || this.value == 0; if (this.form.poll_hide[2].checked) this.form.poll_hide[1].checked = true;" /> `, c.Txt("poll_options1b"), `</td>
							</tr><tr>
								<td align="right"></td>
								<td class="smalltext"><label for="poll_change_vote"><input type="checkbox" id="poll_change_vote" name="poll_change_vote"`, changeVoteChecked, ` class="check" /> `, c.Txt("poll_options7"), `</label></td>
							</tr><tr>
								<td align="right"></td>`)
	} else {
		c.O(`
								<td align="right" valign="top"><b>`, c.Txt("poll_options"), `:</b></td>`)
	}

	hideChecked := func(v int) string {
		if page.HideResults == v {
			return ` checked="checked"`
		}
		return ""
	}
	expireDisabled := ""
	if empty(page.Expiration) {
		expireDisabled = `disabled="disabled"`
	}
	c.O(`
								<td class="smalltext">
									<input type="radio" name="poll_hide" value="0"`, hideChecked(0), ` class="check" /> `+c.Txt("poll_options2")+`<br />
									<input type="radio" name="poll_hide" value="1"`, hideChecked(1), ` class="check" /> `+c.Txt("poll_options3")+`<br />
									<input type="radio" name="poll_hide" value="2"`, hideChecked(2), expireDisabled, ` class="check" /> `+c.Txt("poll_options4")+`<br />
									<br />
								</td>`)
	// If this is an edit, we can allow them to reset the vote counts.
	if page.IsEdit {
		c.O(`
							</tr><tr>
								<td align="right"><b>` + c.Txt("smf40") + `:</b></td>
								<td class="smalltext"><input type="checkbox" name="resetVoteCount" value="on" class="check" /> ` + c.Txt("smf41") + `</td>`)
	}
	c.O(`
							</tr><tr>
								<td align="center" colspan="2">
									<span class="smalltext"><br />`+c.Txt("smf16")+`</span><br />
									<input type="submit" name="post" value="`+c.Txt("10")+`" onclick="return submitThisOnce(this);" accesskey="s" />
									<input type="submit" name="preview" value="`+c.Txt("507")+`" onclick="return submitThisOnce(this);" accesskey="p" />
								</td>
							</tr><tr>
								<td colspan="2"></td>
							</tr>
						</table>
					</td>
				</tr>
			</table>
			<input type="hidden" name="seqnum" value="`, c.FormSequenceNumber, `" />
			<input type="hidden" name="sc" value="`+c.Sc+`" />
		</form>`)
}
