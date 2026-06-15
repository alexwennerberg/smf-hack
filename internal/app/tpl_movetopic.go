package app

import "strings"

// templateMoveTopic is template_main() from MoveTopic.template.php: the
// destination-board picker plus optional rename and redirect-reason fields.
func templateMoveTopic(c *Ctx) {
	page := c.Page.(*MoveTopicCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=movetopic2;topic=`, c.Topic, `.0" method="post" accept-charset="`, c.CharacterSet, `" onsubmit="submitonce(this);">
		<table border="0" width="400" cellspacing="0" cellpadding="4" align="center" class="tborder">
			<tr class="titlebg">
				<td>`, c.Txt("132"), `</td>
			</tr><tr>
				<td class="windowbg" valign="middle" align="center" style="padding-bottom: 1ex; padding-top: 2ex;">
					<b>`, c.Txt("133"), `:</b> <select name="toboard">`)

	// Show dashes (-) before the board name if it's a child.
	for _, board := range page.Boards {
		selected := ""
		if board.Selected {
			selected = ` selected="selected"`
		}
		c.O(`
						<option value="`, board.ID, `"`, selected, `>`, board.Category, ` `, strings.Repeat("-", 1+board.ChildLevel), ` `, board.Name, `</option>`)
	}

	c.O(`
					</select><br />
					<br />
					<label for="reset_subject"><input type="checkbox" name="reset_subject" id="reset_subject" onclick="document.getElementById('subjectArea').style.display = this.checked ? 'block' : 'none';" class="check" /> `, c.Txt("moveTopic2"), `.</label><br />
					<div id="subjectArea" style="display: none; margin-top: 1ex; margin-bottom: 2ex;">
						`, c.Txt("moveTopic3"), `: <input type="text" name="custom_subject" size="30" value="`, page.Subject, `" /><br />
						<label for="enforce_subject"><input type="checkbox" name="enforce_subject" id="enforce_subject" class="check" /> `, c.Txt("moveTopic4"), `.</label>
					</div>`)

	// Disable the reason textarea when the postRedirect checkbox is unchecked...
	c.O(`
					<label for="postRedirect"><input type="checkbox" name="postRedirect" id="postRedirect" checked="checked" onclick="document.getElementById('reasonArea').style.display = this.checked ? 'block' : 'none';" class="check" /> `, c.Txt("moveTopic1"), `.</label><br />
					<div id="reasonArea" style="margin-top: 1ex;">
						`, c.Txt("smf57"), `<br />
						<textarea name="reason" rows="3" cols="40">`, c.Txt("movetopic_default"), `</textarea><br />
					</div>
					<br />
					<input type="submit" value="`, c.Txt("132"), `" onclick="return submitThisOnce(this);" accesskey="s" />
				</td>
			</tr>
		</table>`)

	if page.BackToTopic {
		c.O(`
		<input type="hidden" name="goback" value="1" />`)
	}

	c.O(`
		<input type="hidden" name="sc" value="`, c.Sc, `" />
		<input type="hidden" name="seqnum" value="`, c.FormSequenceNumber, `" />
	</form>`)
}
