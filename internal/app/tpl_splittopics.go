package app

// Ports of Themes/default/SplitTopics.template.php (template_ask, template_main,
// template_select, template_merge, template_merge_extra_options,
// template_merge_done) plus Xml.template.php template_split.

// templateSplitAsk is template_ask().
func templateSplitAsk(c *Ctx) {
	page := c.Page.(*SplitAskCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<form action="`, scripturl, `?action=splittopics;sa=execute;topic=`, c.Topic, `.0" method="post" accept-charset="`, c.CharacterSet, `">
		<input type="hidden" name="at" value="`, page.MessageID, `" />
		<table border="0" width="400" cellspacing="0" cellpadding="3" align="center" class="tborder">
			<tr class="titlebg">
				<td>`, c.Txt("smf251"), `</td>
			</tr><tr class="windowbg">
				<td align="center" style="padding-top: 2ex; padding-bottom: 1ex;">
					<b><label for="subname">`, c.Txt("smf254"), `</label>:</b> <input type="text" name="subname" id="subname" value="`, page.MessageSubject, `" size="25" /><br />
					<br />
					<input type="radio" name="step2" value="onlythis" checked="checked" class="check" /> `, c.Txt("smf255"), `<br />
					<input type="radio" name="step2" value="afterthis" class="check" /> `, c.Txt("smf256"), `<br />
					<input type="radio" name="step2" value="selective" class="check" /> `, c.Txt("smf257"), `<br />
					<br />
					<input type="submit" value="`, c.Txt("smf251"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateSplitMain is template_main(): the split-done page.
func templateSplitMain(c *Ctx) {
	page := c.Page.(*SplitMainCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<table border="0" width="400" cellspacing="1" class="bordercolor" cellpadding="4" align="center">
			<tr class="titlebg">
				<td>`, c.Txt("smf251"), `</td>
			</tr><tr>
				<td class="windowbg" valign="middle" align="center">
					`, c.Txt("smf259"), `<br /><br />
					<a href="`, scripturl, `?board=`, c.Board, `.0">`, c.Txt("101"), `</a><br />
					<a href="`, scripturl, `?topic=`, page.OldTopic, `.0">`, c.Txt("smf260"), `</a><br />
					<a href="`, scripturl, `?topic=`, page.NewTopic, `.0">`, c.Txt("smf258"), `</a>
				</td>
			</tr>
		</table>`)
}

// templateSplitSelect is template_select(): the selective message picker.
func templateSplitSelect(c *Ctx) {
	page := c.Page.(*SplitSelectCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<form action="`, scripturl, `?action=splittopics;sa=splitSelection;board=`, c.Board, `.0" method="post" accept-charset="`, c.CharacterSet, `"><input type="hidden" name="topic" value="`, c.Topic, `" />
		<table width="100%"><tr><td colspan="2" align="center">
			<input type="hidden" name="subname" value="`, page.NewSubject, `" />
			<input type="submit" value="`, c.Txt("smf251"), `" />
			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</td></tr><tr><td valign="top" width="50%">
			<table id="table_not_selected" border="0" width="98%" cellspacing="1" class="bordercolor" cellpadding="4" align="center">
				<tr class="titlebg">
					<td colspan="2">
						`, c.Txt("smf251"), ` - `, c.Txt("smf257"), `
					</td>
				</tr>
				<tr class="windowbg">
					<td colspan="2" valign="middle">
						`, c.Txt("smf261"), `
					</td>
				</tr>
				<tr class="catbg">
					<td colspan="2" height="18">
						<b>`, c.Txt("139"), `:</b> <span id="pageindex_not_selected">`, page.NotSelected.PageIndex, `</span>
					</td>
				</tr>`)

	for _, message := range page.NotSelected.Messages {
		c.O(`
				<tr class="windowbg" id="not_selected_`, message.ID, `">
					<td class="smalltext">
						`, message.Subject, ` - `, message.Poster, `
						<div class="post">`, message.Body, `</div>
					</td>
					<td valign="middle" align="center" width="5%"><a href="`, scripturl, `?action=splittopics;sa=selectTopics;subname=`, page.TopicSubject, `;topic=`, page.TopicID, `.`, page.NotSelected.Start, `;start2=`, page.Selected.Start, `;move=down;msg=`, message.ID, `" onclick="return select('down', `, message.ID, `);"><img src="`, imagesURL, `/split_select.gif" alt="-&gt;" /></a></td>
				</tr>`)
	}

	c.O(`
			</table>
		</td><td valign="top" width="50%">
			<table id="table_selected" border="0" width="98%" cellspacing="1" class="bordercolor" cellpadding="4" align="center">
				<tr class="titlebg">
					<td colspan="2">
						`, c.Txt("split_selected_posts"), ` (<a href="`, scripturl, `?action=splittopics;sa=selectTopics;subname=`, page.TopicSubject, `;topic=`, page.TopicID, `.`, page.NotSelected.Start, `;start2=`, page.Selected.Start, `;move=reset;msg=0" onclick="return select('reset', 0);">`, c.Txt("split_reset_selection"), `</a>)
					</td>
				</tr>
				<tr class="windowbg">
					<td colspan="2" valign="middle">
						`, c.Txt("split_selected_posts_desc"), `
					</td>
				</tr>
				<tr class="catbg">
					<td colspan="2" height="18">
						<b>`, c.Txt("139"), `:</b> <span id="pageindex_selected">`, page.Selected.PageIndex, `</span>
					</td>
				</tr>`)

	for _, message := range page.Selected.Messages {
		c.O(`
				<tr class="windowbg" id="selected_`, message.ID, `">
					<td width="5%" valign="middle" align="center"><a href="`, scripturl, `?action=splittopics;sa=selectTopics;subname=`, page.TopicSubject, `;topic=`, page.TopicID, `.`, page.NotSelected.Start, `;start2=`, page.Selected.Start, `;move=up;msg=`, message.ID, `" onclick="return select('up', `, message.ID, `);"><img src="`, imagesURL, `/split_deselect.gif" alt="&lt;-" /></a></td>
					<td class="smalltext">
						`, message.Subject, ` - `, message.Poster, `
						<div class="post">`, message.Body, `</div>
					</td>
				</tr>`)
	}

	c.O(`
			</table>
		</td></tr></table>
		</form>
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var start = new Array();
			start[0] = `, page.NotSelected.Start, `;
			start[1] = `, page.Selected.Start, `;

			function select(direction, msg_id)
			{
				if (window.XMLHttpRequest)
				{
					getXMLDocument("`, scripturl, `?action=splittopics;sa=selectTopics;subname=`, page.TopicSubject, `;topic=`, page.TopicID, `." + start[0] + ";start2=" + start[1] + ";move=" + direction + ";msg=" + msg_id + ";xml", onDocReceived);
					return false;
				}
				else
					return true;
			}
			function onDocReceived(XMLDoc)
			{
				var i, j, pageIndex;
				for (i = 0; i < 2; i++)
				{
					pageIndex = XMLDoc.getElementsByTagName("pageIndex")[i];
					setInnerHTML(document.getElementById("pageindex_" + pageIndex.getAttribute("section")), pageIndex.firstChild.nodeValue);
					start[i] = pageIndex.getAttribute("startFrom");
				}
				var numChanges = XMLDoc.getElementsByTagName("change").length, curChange;
				var curSection, curAction, curId, curTable, curRow, curRowIndex, buttonCell, textCell, curData, numRows;
				for (i = 0; i < numChanges; i++)
				{
					curChange = XMLDoc.getElementsByTagName("change")[i];
					curSection = curChange.getAttribute("section");
					curAction = curChange.getAttribute("curAction");
					curId = curChange.getAttribute("id");
					curTable = document.getElementById("table_" + curSection);
					numRows = curTable.rows.length;
					if (curAction == "remove")
						curTable.deleteRow(document.getElementById(curSection + "_" + curId).rowIndex);
					// Insert a message.
					else
					{
						// By default insert the row at the end of the table.
						curRowIndex = -1;
						for (j = curSection == "selected" ? 2 : 3; j < numRows; j++)
						{
							if (parseInt(curTable.rows[j].id.substr(curSection.length + 1)) < curId)
							{
								// This would be a nice place to insert the row.
								curRowIndex = j;
								// We're done for now. Escape the loop.
								j = numRows + 1;
							}
						}
						curRow = curTable.insertRow(curRowIndex);
						curRow.className = "windowbg";
						curRow.id = curSection + "_" + curId;
						if (curSection == "selected")
						{
							buttonCell = curRow.insertCell(-1);
							textCell = curRow.insertCell(-1);
						}
						else
						{
							textCell = curRow.insertCell(-1);
							buttonCell = curRow.insertCell(-1);
						}
						setInnerHTML(buttonCell, "<a href=\"`, scripturl, `?action=splittopics;sa=selectTopics;subname=`, page.TopicSubject, `;topic=`, page.TopicID, `.`, page.NotSelected.Start, `;start2=`, page.Selected.Start, `;move=" + (curSection == "selected" ? "up" : "down") + ";msg=" + curId + "\" onclick=\"return select('" + (curSection == "selected" ? "up" : "down") + "', " + curId + ");\"><img src=\"`, imagesURL, `/split_" + (curSection == "selected" ? "de" : "") + "select.gif\" alt=\"" + (curSection == "selected" ? "&lt;-" : "-&gt;") + "\" border=\"0\" /></a>");
						buttonCell.width = "5%";
						buttonCell.vAlign = "middle";
						buttonCell.align = "center";
						setInnerHTML(textCell, curChange.getElementsByTagName("subject")[0].firstChild.nodeValue + " - " + curChange.getElementsByTagName("poster")[0].firstChild.nodeValue + "<div class=\"post\">" + curChange.getElementsByTagName("body")[0].firstChild.nodeValue + "</div>");
						textCell.className = "smalltext";
						// !!! Should something be here?
						//textCell.alt
					}
				}
			}
		// ]]></script>`)
}

// templateSplitXML is template_split() from Xml.template.php.
func templateSplitXML(c *Ctx) {
	page := c.Page.(*SplitSelectCtx)

	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>
	<pageIndex section="not_selected" startFrom="`, page.NotSelected.Start, `"><![CDATA[`, page.NotSelected.PageIndex, `]]></pageIndex>
	<pageIndex section="selected" startFrom="`, page.Selected.Start, `"><![CDATA[`, page.Selected.PageIndex, `]]></pageIndex>`)

	for _, change := range page.Changes {
		if change.Type == "remove" {
			c.O(`
	<change id="`, change.ID, `" curAction="remove" section="`, change.Section, `" />`)
		} else {
			c.O(`
	<change id="`, change.ID, `" curAction="insert" section="`, change.Section, `">
		<subject><![CDATA[`, change.Insert.Subject, `]]></subject>
		<body><![CDATA[`, change.Insert.Body, `]]></body>
		<poster><![CDATA[`, change.Insert.Poster, `]]></poster>
	</change>`)
		}
	}
	c.O(`
</smf>`)
}

// templateMergeDone is template_merge_done().
func templateMergeDone(c *Ctx) {
	page := c.Page.(*MergeDoneCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<table border="0" width="400" cellspacing="1" class="bordercolor" cellpadding="4" align="center">
			<tr class="titlebg">
				<td>`, c.Txt("smf252"), `</td>
			</tr><tr>
				<td class="windowbg" valign="middle" align="center">
					<br />
					`, c.Txt("smf264"), `<br />
					<br />
					<a href="`, scripturl, `?board=`, page.TargetBoard, `.0">`, c.Txt("101"), `</a><br />
					<a href="`, scripturl, `?topic=`, page.TargetTopic, `.0">`, c.Txt("smf265"), `</a>
				</td>
			</tr>
		</table>`)
}

// templateMerge is template_merge(): choose a topic to merge with.
func templateMerge(c *Ctx) {
	page := c.Page.(*MergeIndexCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=mergetopics;from=`, page.OriginTopic, `;targetboard=`, page.TargetBoard, `;board=`, c.Board, `.0" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" width="540" cellspacing="1" class="bordercolor" cellpadding="4" align="center">
				<tr class="catbg3">
					<td>`, c.Txt("smf252"), `</td>
				</tr>
				<tr>
					<td class="windowbg">`, c.Txt("smf276"), `</td>
				</tr>
				<tr>
					<td colspan="2" class="titlebg">
						<table cellpadding="0" cellspacing="0" border="0"><tr>
							<td><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
						</tr></table>
					</td>
				</tr>
				<tr>
					<td class="windowbg" valign="middle" align="center">
						<table border="0">
							<tr>
								<td align="right"><br /><b>`, c.Txt("smf266"), `:</b> <br /></td>
								<td align="left"><input type="hidden" name="from" value="`, page.OriginTopic, `" /><br />`, page.OriginSubject, `</td>
							</tr><tr>`)

	if len(page.Boards) > 1 {
		c.O(`
								<td align="right"><br /><b>`, c.Txt("smf267"), `:</b></td>
								<td align="left">
									<br />
									<select name="targetboard" onchange="this.form.submit();">`)
		for _, board := range page.Boards {
			selected := ""
			if board.ID == page.TargetBoard {
				selected = ` selected="selected"`
			}
			c.O(`
										<option value="`, board.ID, `"`, selected, `>`, board.Category, ` - `, board.Name, `</option>`)
		}
		c.O(`
									</select> <noscript><input type="submit" value="`, c.Txt("462"), `" /></noscript>
								</td>`)
	}

	c.O(`
							</tr><tr>
								<td align="right" valign="top"><br /><b>`, c.Txt("smf269"), `:</b></td>
								<td align="left" style="white-space: nowrap;">
									<br />
									<table>`)

	mergeButton := c.createButton("merge.gif", "smf252", "", "")
	for _, topic := range page.Topics {
		c.O(`
										<tr>
											<td valign="bottom">
												<a href="`, scripturl, `?action=mergetopics;sa=options;board=`, c.Board, `.0;from=`, page.OriginTopic, `;to=`, topic.ID, `;sesc=`, c.Sc, `">`, mergeButton, `</a>&nbsp;
											</td>
											<td valign="middle" style="white-space: nowrap;">
												<a href="`, scripturl, `?topic=`, topic.ID, `.0" target="_blank">`, topic.Subject, `</a> `, c.Txt("109"), ` `, topic.PosterLink, `
											</td>
										</tr>`)
	}
	c.O(`
									</table>
								</td>
							</tr>
						</table>
					</td>
				</tr>
				<tr>
					<td colspan="2" class="titlebg">
						<table cellpadding="0" cellspacing="0" border="0"><tr>
							<td><b>`, c.Txt("139"), `:</b> `, page.PageIndex, `</td>
						</tr></table>
					</td>
				</tr>
			</table>
		</form>`)
}

// templateMergeExtraOptions is template_merge_extra_options().
func templateMergeExtraOptions(c *Ctx) {
	page := c.Page.(*MergeExtraCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=mergetopics;sa=execute;" method="post" accept-charset="`, c.CharacterSet, `">
			<table border="0" width="100%" cellspacing="1" class="bordercolor" cellpadding="4" align="center">
				<tr class="titlebg">
					<td>`, c.Txt("smf252"), `</td>
				</tr><tr>
					<td class="catbg">`, c.Txt("merge_topic_list"), `</td>
				</tr><tr>
					<td class="windowbg" style="padding: 15px;">
						<table border="0" cellspacing="1" cellpadding="2" width="100%" align="center" class="bordercolor">
							<tr class="titlebg">
								<td>`, c.Txt("merge_check"), `</td>
								<td>`, c.Txt("70"), `</td>
								<td>`, c.Txt("109"), `</td>
								<td>`, c.Txt("111"), `</td>
								<td width="70">`, c.Txt("merge_include_notifications"), `</td>
							</tr>`)

	for _, topic := range page.Topics {
		c.O(`
							<tr>
								<td class="windowbg2" valign="middle">
									<input type="checkbox" class="check" name="topics[]" value="`, topic.ID, `" checked="checked" />
								</td>
								<td class="windowbg2" valign="middle">
									<a href="`, scripturl, `?topic=`, topic.ID, `.0" target="_blank">`, topic.Subject, `</a>
								</td>
								<td class="windowbg2" valign="middle">
									`, topic.Started.Link, `<br />
									<span class="smalltext">`, topic.Started.Time, `</span>
								</td>
								<td class="windowbg2" valign="middle">
									`, topic.Updated.Link, `<br />
									<span class="smalltext">`, topic.Updated.Time, `</span>
								</td>
								<td class="windowbg2" valign="middle">
									<input type="checkbox" class="check" name="notifications[]" value="`, topic.ID, `" checked="checked" />
								</td>
							</tr>`)
	}
	c.O(`
						</table>
						<br />
						<br />`)

	c.O(`
						`, c.Txt("merge_select_subject"), `: <select name="subject" onchange="this.form.customSubject.disabled = this.options[this.selectedIndex].value != 0;">`)
	for _, topic := range page.Topics {
		selected := ""
		if topic.Selected {
			selected = ` selected="selected"`
		}
		c.O(`
							<option value="`, topic.ID, `"`, selected, `>`, topic.Subject, `</option>`)
	}
	c.O(`
							<option value="0">`, c.Txt("merge_custom_subject"), `:</option>
						</select> <input type="text" name="custom_subject" size="60" disabled="disabled" id="customSubject" /><br />
						<br />
						<input type="checkbox" class="check" name="enforce_subject" value="1" /> `, c.Txt("merge_enforce_subject"), `
					</td>
				</tr>`)

	if len(page.Boards) > 1 {
		c.O(`
				<tr>
					<td class="catbg">`, c.Txt("merge_select_target_board"), `</td>
				</tr><tr>
					<td class="windowbg"><table border="0" cellspacing="0" cellpadding="0">`)
		for _, board := range page.Boards {
			checked := ""
			if board.Selected {
				checked = ` checked="checked"`
			}
			c.O(`
						<tr>
							<td>
								<input type="radio" name="board" value="`, board.ID, `"`, checked, ` class="check" /> `, board.Name, `
							</td>
						</tr>`)
		}
		c.O(`
					</table></td>
				</tr>`)
	}

	if len(page.Polls) > 0 {
		c.O(`
				<tr>
					<td class="catbg">`, c.Txt("merge_select_poll"), `</td>
				</tr><tr>
					<td class="windowbg"><table border="0" cellspacing="0" cellpadding="3">`)
		for _, poll := range page.Polls {
			checked := ""
			if poll.Selected {
				checked = ` checked="checked"`
			}
			c.O(`
						<tr>
							<td>
								<input type="radio" name="poll" value="`, poll.ID, `"`, checked, ` class="check" /> `, poll.Question, ` (`, c.Txt("118"), `: <a href="`, scripturl, `?topic=`, poll.TopicID, `.0" target="_blank">`, poll.TopicSubject, `</a>)
							</td>
						</tr>`)
		}
		c.O(`
						<tr>
							<td>
								<input type="radio" name="poll" value="-1" class="check" /> (`, c.Txt("merge_no_poll"), `)
							</td>
						</tr>
					</table></td>
				</tr>`)
	}

	c.O(`
				<tr>
					<td class="windowbg" align="right">
						<input type="submit" value="`, c.Txt("smf252"), `" />
						<input type="hidden" name="sa" value="execute" />
						<input type="hidden" name="sc" value="`, c.Sc, `" />
					</td>
				</tr>
			</table>
		</form>`)
}
