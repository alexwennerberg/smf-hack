package app

// Hand-port of Themes/default/Stats.template.php (template_main) and the
// stats XML sub-template from Themes/default/Xml.template.php (template_stats).

// templateStatsMain is template_main() from Stats.template.php.
func templateStatsMain(c *Ctx) {
	page := c.Page.(*StatsCtx)
	scripturl := c.App.ScriptURL
	images := c.Theme.ImagesURL()
	hitStats := !c.App.SettingEmpty("hitStats")

	bar := func(value, percent int) string {
		if value > 0 {
			return `<img src="` + images + `/bar.gif" width="` + itoa(percent) + `" height="15" alt="" />`
		}
		return "&nbsp;"
	}

	c.O(`
		<table width="100%" cellpadding="3" cellspacing="0">
			<tr>
				<td>`)
	c.themeLinktree()
	c.O(`</td>
			</tr>
		</table>
		<table border="0" width="100%" cellspacing="1" cellpadding="4" class="bordercolor">
			<tr class="titlebg">
				<td align="center" colspan="4">`, c.PageTitle, `</td>
			</tr>
			<tr>
				<td class="catbg" colspan="4"><b>`, c.Txt("smf_stats_2"), `</b></td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_info.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">
						<tr>
							<td nowrap="nowrap">`, c.Txt("488"), `:</td>
							<td align="right">`)
	if page.ShowMemberList {
		c.O(`<a href="`, scripturl, `?action=mlist">`, page.NumMembers, `</a>`)
	} else {
		c.O(page.NumMembers)
	}
	c.O(`</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("489"), `:</td>
							<td align="right">`, page.NumPosts, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("490"), `:</td>
							<td align="right">`, page.NumTopics, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("658"), `:</td>
							<td align="right">`, page.NumCategories, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("users_online"), `:</td>
							<td align="right">`, page.UsersOnline, `</td>
						</tr><tr>
							<td nowrap="nowrap" valign="top">`, c.Txt("888"), `:</td>
							<td align="right">`, page.MostOnlineNum, ` - `, page.MostOnlineDate, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("users_online_today"), `:</td>
							<td align="right">`, page.OnlineToday, `</td>`)
	if hitStats {
		c.O(`
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("num_hits"), `:</td>
							<td align="right">`, page.NumHits, `</td>`)
	}
	c.O(`
						</tr>
					</table>
				</td>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_info.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">
						<tr>
							<td nowrap="nowrap">`, c.Txt("average_members"), `:</td>
							<td align="right">`, page.AverageMembers, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("average_posts"), `:</td>
							<td align="right">`, page.AveragePosts, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("average_topics"), `:</td>
							<td align="right">`, page.AverageTopics, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("665"), `:</td>
							<td align="right">`, page.NumBoards, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("656"), `:</td>
							<td align="right">`, c.CommonStats.LatestMemberLink, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("average_online"), `:</td>
							<td align="right">`, page.AverageOnline, `</td>
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("gender_ratio"), `:</td>
							<td align="right">`, page.GenderRatio, `</td>`)
	if hitStats {
		c.O(`
						</tr><tr>
							<td nowrap="nowrap">`, c.Txt("average_hits"), `:</td>
							<td align="right">`, page.AverageHits, `</td>`)
	}
	c.O(`
						</tr>
					</table>
				</td>
			</tr><tr>
				<td class="catbg" colspan="2" width="50%"><b>`, c.Txt("smf_stats_3"), `</b></td>
				<td class="catbg" colspan="2" width="50%"><b>`, c.Txt("smf_stats_4"), `</b></td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_posters.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	for _, poster := range page.TopPosters {
		c.O(`
						<tr>
							<td width="60%" valign="top">`, poster.Link, `</td>
							<td width="20%" align="left" valign="top">`, bar(poster.NumPosts, poster.PostPercent), `</td>
							<td width="20%" align="right" valign="top">`, poster.NumPosts, `</td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_board.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	for _, board := range page.TopBoards {
		c.O(`
						<tr>
							<td width="60%" valign="top">`, board.Link, `</td>
							<td width="20%" align="left" valign="top">`, bar(board.NumPosts, board.PostPercent), `</td>
							<td width="20%" align="right" valign="top">`, board.NumPosts, `</td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
			</tr><tr>
				<td class="catbg" colspan="2" width="50%"><b>`, c.Txt("smf_stats_11"), `</b></td>
				<td class="catbg" colspan="2" width="50%"><b>`, c.Txt("smf_stats_12"), `</b></td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_replies.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	for _, topic := range page.TopTopicsReplies {
		c.O(`
						<tr>
							<td width="60%" valign="top">`, topic.Link, `</td>
							<td width="20%" align="left" valign="top">`, bar(topic.NumReplies, topic.PostPercent), `</td>
							<td width="20%" align="right" valign="top">`, topic.NumReplies, `</td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_views.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	for _, topic := range page.TopTopicsViews {
		c.O(`
						<tr>
							<td width="60%" valign="top">`, topic.Link, `</td>
							<td width="20%" align="left" valign="top">`, bar(topic.NumViews, topic.PostPercent), `</td>
							<td width="20%" align="right" valign="top">`, topic.NumViews, `</td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
			</tr><tr>
				<td class="catbg" colspan="2" width="50%"><b>`, c.Txt("smf_stats_15"), `</b></td>
				<td class="catbg" colspan="2" width="50%"><b>`, c.Txt("smf_stats_16"), `</b></td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_replies.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	for _, poster := range page.TopStarters {
		c.O(`
						<tr>
							<td width="60%" valign="top">`, poster.Link, `</td>
							<td width="20%" align="left" valign="top">`, bar(poster.NumTopics, poster.PostPercent), `</td>
							<td width="20%" align="right" valign="top">`, poster.NumTopics, `</td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
				<td class="windowbg" width="20" valign="middle" align="center" nowrap="nowrap"><img src="`, images, `/stats_views.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" width="50%" valign="top">
					<table border="0" cellpadding="1" cellspacing="0" width="100%">`)
	for _, poster := range page.TopTimeOnline {
		// SMF guards on the raw time_online (seconds) > 0; NumPosts holds the raw
		// seconds here. The formatted TimeOnline ("0m") is always non-empty.
		c.O(`
						<tr>
							<td width="60%" valign="top">`, poster.Link, `</td>
							<td width="20%" align="left" valign="top">`, bar(poster.NumPosts, poster.TimePercent), `</td>
							<td width="20%" align="right" valign="top" nowrap="nowrap">`, poster.TimeOnline, `</td>
						</tr>`)
	}
	c.O(`
					</table>
				</td>
			</tr><tr>
				<td class="catbg" colspan="4"><b>`, c.Txt("smf_stats_5"), `</b></td>
			</tr><tr>
				<td class="windowbg" width="20" valign="middle" align="center"><img src="`, images, `/stats_history.gif" width="20" height="20" alt="" /></td>
				<td class="windowbg2" colspan="4">`)

	if len(page.Monthly) != 0 {
		c.O(`
					<table border="0" width="100%" cellspacing="1" cellpadding="4" class="tborder" style="margin-bottom: 1ex;" id="stats">
						<tr class="titlebg" valign="middle" align="center">
							<td width="25%">`, c.Txt("smf_stats_13"), `</td>
							<td width="15%">`, c.Txt("smf_stats_7"), `</td>
							<td width="15%">`, c.Txt("smf_stats_8"), `</td>
							<td width="15%">`, c.Txt("smf_stats_9"), `</td>
							<td width="15%">`, c.Txt("smf_stats_14"), `</td>`)
		if hitStats {
			c.O(`
							<td>`, c.Txt("smf_stats_10"), `</td>`)
		}
		c.O(`
						</tr>`)

		for _, month := range page.Monthly {
			expandImg := "expand.gif"
			if month.Expanded {
				expandImg = "collapse.gif"
			}
			c.O(`
						<tr class="windowbg2" valign="middle" id="tr_`, month.ID, `">
							<th align="left" width="25%">
								<a name="`, month.ID, `" id="link_`, month.ID, `" href="`, month.Href, `" onclick="return doingExpandCollapse || expand_collapse('`, month.ID, `', `, month.NumDays, `);"><img src="`, images, `/`, expandImg, `" alt="" id="img_`, month.ID, `" /> `, month.Month, ` `, month.Year, `</a>
							</th>
							<th align="center" width="15%">`, month.NewTopics, `</th>
							<th align="center" width="15%">`, month.NewPosts, `</th>
							<th align="center" width="15%">`, month.NewMembers, `</th>
							<th align="center" width="15%">`, month.MostMembersOnline, `</th>`)
			if hitStats {
				c.O(`
							<th align="center">`, month.Hits, `</th>`)
			}
			c.O(`
						</tr>`)

			if month.Expanded {
				for _, day := range month.Days {
					c.O(`
						<tr class="windowbg2" valign="middle" align="left">
							<td align="left" style="padding-left: 3ex;">`, day.Year, `-`, day.Month, `-`, day.Day, `</td>
							<td align="center">`, day.NewTopics, `</td>
							<td align="center">`, day.NewPosts, `</td>
							<td align="center">`, day.NewMembers, `</td>
							<td align="center">`, day.MostMembersOnline, `</td>`)
					if hitStats {
						c.O(`
							<td align="center">`, day.Hits, `</td>`)
					}
					c.O(`
						</tr>`)
				}
			}
		}
		c.O(`
					</table>`)
	}

	c.O(`
				</td>
			</tr>
		</table>
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var doingExpandCollapse = false;

			function expand_collapse(curId, numDays)
			{
				if (window.XMLHttpRequest)
				{
					if (document.getElementById("img_" + curId).src.indexOf("expand") > 0)
					{
						if (typeof window.ajax_indicator == "function")
							ajax_indicator(true);
						getXMLDocument(smf_scripturl + "?action=stats;expand=" + curId + ";xml", onDocReceived);
						doingExpandCollapse = true;
					}
					else
					{
						var myTable = document.getElementById("stats"), i;
						var start = document.getElementById("tr_" + curId).rowIndex + 1;
						for (i = 0; i < numDays; i++)
							myTable.deleteRow(start);
						// Adjust the image and link.
						document.getElementById("img_" + curId).src = smf_images_url + "/expand.gif";
						document.getElementById("link_" + curId).href = smf_scripturl + "?action=stats;expand=" + curId + "#" + curId;
						// Modify the session variables.
						getXMLDocument(smf_scripturl + "?action=stats;collapse=" + curId + ";xml");
					}
					return false;
				}
				else
					return true;
			}
			function onDocReceived(XMLDoc)
			{
				var numMonths = XMLDoc.getElementsByTagName("month").length, i, j, k, numDays, curDay, start;
				var myTable = document.getElementById("stats"), curId, myRow, myCell, myData;
				var dataCells = [
					"date",
					"new_topics",
					"new_posts",
					"new_members",
					"most_members_online"
				];

				if (numMonths > 0 && XMLDoc.getElementsByTagName("month")[0].getElementsByTagName("day").length > 0 && XMLDoc.getElementsByTagName("month")[0].getElementsByTagName("day")[0].getAttribute("hits") != null)
					dataCells[5] = "hits";

				for (i = 0; i < numMonths; i++)
				{
					numDays = XMLDoc.getElementsByTagName("month")[i].getElementsByTagName("day").length;
					curId = XMLDoc.getElementsByTagName("month")[i].getAttribute("id");
					start = document.getElementById("tr_" + curId).rowIndex + 1;
					for (j = 0; j < numDays; j++)
					{
						curDay = XMLDoc.getElementsByTagName("month")[i].getElementsByTagName("day")[j];
						myRow = myTable.insertRow(start + j);
						myRow.className = "windowbg2";

						for (k in dataCells)
						{
							myCell = myRow.insertCell(-1);
							if (dataCells[k] == "date")
								myCell.style.paddingLeft = "3ex";
							else
								myCell.style.textAlign = "center";
							myData = document.createTextNode(curDay.getAttribute(dataCells[k]));
							myCell.appendChild(myData);
						}
					}
					// Adjust the arrow to point downwards.
					document.getElementById("img_" + curId).src = smf_images_url + "/collapse.gif";
					// Adjust the link to collapse instead of expand
					document.getElementById("link_" + curId).href = smf_scripturl + "?action=stats;collapse=" + curId + "#" + curId;
				}

				doingExpandCollapse = false;
				if (typeof window.ajax_indicator == "function")
					ajax_indicator(false);
			}
		// ]]></script>`)
}

// templateStatsXML is template_stats() from Xml.template.php (the expand AJAX).
func templateStatsXML(c *Ctx) {
	page := c.Page.(*StatsCtx)
	hitStats := !c.App.SettingEmpty("hitStats")

	c.O(`<?xml version="1.0" encoding="`, c.CharacterSet, `"?>
<smf>`)
	for _, month := range page.Monthly {
		c.O(`
	<month id="`, month.DateYear, month.DateMonth, `">`)
		for _, day := range month.Days {
			c.O(`
		<day date="`, day.Year, `-`, day.Month, `-`, day.Day, `" new_topics="`, day.NewTopics, `" new_posts="`, day.NewPosts, `" new_members="`, day.NewMembers, `" most_members_online="`, day.MostMembersOnline, `"`)
			if hitStats {
				c.O(` hits="`, day.Hits, `"`)
			}
			c.O(` />`)
		}
		c.O(`
	</month>`)
	}
	c.O(`
</smf>`)
}
