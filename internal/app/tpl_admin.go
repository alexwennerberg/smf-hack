package app

// Ports of Themes/default/Admin.template.php: the 'admin' layer
// (template_admin_above/below — the sidebar + optional tab bar) and the admin
// home (template_admin) and support/credits (template_credits) sub-templates.

import "strings"

func init() {
	layerFuncs["admin_above"] = templateAdminAbove
	layerFuncs["admin_below"] = templateAdminBelow
}

// templateAdminAbove is template_admin_above(): the sidebar and tab bar.
func templateAdminAbove(c *Ctx) {
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<table width="100%" cellpadding="0" cellspacing="0" border="0" style="padding-top: 1ex;"><tr>
			<td width="150" valign="top" style="width: 23ex; padding-right: 10px; padding-bottom: 10px;">
				<table width="100%" cellpadding="4" cellspacing="1" border="0" class="bordercolor">`)

	for _, section := range c.AdminAreas {
		c.O(`
					<tr>
						<td class="catbg">`, section.Title, `</td>
					</tr>
					<tr class="windowbg2">
						<td class="smalltext" style="line-height: 1.3; padding-bottom: 3ex;">`)
		for _, area := range section.Areas {
			if area.Key == c.AdminArea {
				c.O(`
							<b>`, area.Link, `</b><br />`)
			} else {
				c.O(`
							`, area.Link, `<br />`)
			}
		}
		c.O(`
						</td>
					</tr>`)
	}

	c.O(`
				</table>
			</td>
			<td valign="top">`)

	if c.AdminTabs == nil {
		return
	}
	tabs := c.AdminTabs

	useTabs := !c.Theme.Empty("use_tabs")
	marginStyle := `style="margin-bottom: 2ex;"`
	if useTabs {
		marginStyle = ""
	}
	c.O(`
				<table border="0" cellspacing="0" cellpadding="4" align="center" width="100%" class="tborder" `, marginStyle, `>
					<tr class="titlebg">
						<td>`)
	if tabs.Help != "" {
		c.O(`
							<a href="`, scripturl, `?action=helpadmin;help=`, tabs.Help, `" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `)
	}
	c.O(`
							`, tabs.Title, `
						</td>
					</tr>
					<tr class="windowbg">`)

	// Find the selected tab's description.
	selDesc := tabs.Description
	for _, tab := range tabs.Tabs {
		if tab.IsSelected && tab.Description != "" {
			selDesc = tab.Description
		}
	}

	if useTabs {
		c.O(`
						<td class="smalltext" style="padding: 2ex;">`, selDesc, `</td>
					</tr>
				</table>`)
		c.O(`
				<table cellpadding="0" cellspacing="0" border="0" style="margin-left: 10px;">
					<tr>
						<td class="maintab_first">&nbsp;</td>`)
		for _, tab := range tabs.Tabs {
			if tab.IsSelected {
				c.O(`
						<td class="maintab_active_first">&nbsp;</td>
						<td valign="top" class="maintab_active_back">
							<a href="`, tab.Href, `">`, tab.Title, `</a>
						</td>
						<td class="maintab_active_last">&nbsp;</td>`)
			} else {
				c.O(`
						<td valign="top" class="maintab_back">
							<a href="`, tab.Href, `">`, tab.Title, `</a>
						</td>`)
			}
		}
		c.O(`
						<td class="maintab_last">&nbsp;</td>
					</tr>
				</table><br />`)
	} else {
		c.O(`
						<td align="left"><b>`)
		for _, tab := range tabs.Tabs {
			if tab.IsSelected {
				c.O(`
							<img src="`, imagesURL, `/selected.gif" alt="*" /> <b><a href="`, tab.Href, `">`, tab.Title, `</a></b>`)
			} else {
				c.O(`
							<a href="`, tab.Href, `">`, tab.Title, `</a>`)
			}
			if !tab.IsLast {
				c.O(` | `)
			}
		}
		c.O(`
						</b></td>
					</tr>
					<tr class="windowbg">
						<td class="smalltext" style="padding: 2ex;">`, selDesc, `</td>
					</tr>
				</table>`)
	}
}

// templateAdminBelow is template_admin_below(): close the layer table.
func templateAdminBelow(c *Ctx) {
	c.O(`
			</td>
		</tr>
	</table>`)
}

// templateAdmin is template_admin(): the admin center home.
func templateAdmin(c *Ctx) {
	page := c.Page.(*AdminMainCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<table width="100%" cellpadding="3" cellspacing="1" border="0" class="bordercolor">
			<tr class="titlebg">
				<td align="center" colspan="2" class="largetext">`, c.Txt("208"), `</td>
			</tr><tr>
				<td class="windowbg" valign="top" style="padding: 7px;">
					<b>`, c.Txt("hello_guest"), ` `, c.User.Name, `!</b>
					<div style="font-size: 0.85em; padding-top: 1ex;">`, c.Txt("644"), `</div>
				</td>
			</tr>
		</table>`)

	c.O(`
	<div id="update_section" style="display: none;">
		<table width="100%" cellpadding="4" cellspacing="1" border="0" class="bordercolor" style="margin-top: 1.5ex;" id="update_table">
			<tr class="titlebg">
				<td id="update_title">`, c.Txt("update_available"), `</td>
			</tr><tr>
				<td class="windowbg" valign="top" style="padding: 0;">
					<div id="update_message" style="font-size: 0.85em; padding: 4px;">`, c.Txt("update_message"), `</div>
				</td>
			</tr>
		</table>
	</div>`)

	c.O(`
		<table width="100%" cellpadding="0" cellspacing="0" border="0" style="margin-top: 1.5ex;"><tr>`)

	c.O(`
			<td valign="top">
				<table width="100%" cellpadding="5" cellspacing="1" border="0" class="bordercolor">
					<tr>
						<td class="catbg">
							<a href="`, scripturl, `?action=helpadmin;help=live_news" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("smf217"), `
						</td>
					</tr><tr>
						<td class="windowbg2" valign="top" style="height: 18ex; padding: 0;">
							<div id="smfAnnouncements" style="height: 18ex; overflow: auto; padding-right: 1ex;"><div style="margin: 4px; font-size: 0.85em;">`, c.Txt("lfyi"), `</div></div>
						</td>
					</tr>
				</table>
			</td>
			<td style="width: 1ex;">&nbsp;</td>`)

	detailLink := ""
	if page.CanAdmin {
		detailLink = `<a href="` + scripturl + `?action=detailedversion">` + c.Txt("dvc_more") + `</a>`
	}
	c.O(`
			<td valign="top" style="width: 40%;">
				<table width="100%" cellpadding="5" cellspacing="1" border="0" class="bordercolor" id="supportVersionsTable">
					<tr>
						<td class="catbg"><a href="`, scripturl, `?action=admin;credits">`, c.Txt("support_title"), `</a></td>
					</tr><tr>
						<td class="windowbg2" valign="top" style="height: 18ex;">
							<b>`, c.Txt("support_versions"), `:</b><br />
							`, c.Txt("support_versions_forum"), `:
							<i id="yourVersion" style="white-space: nowrap;">`, page.ForumVersion, `</i><br />
							`, c.Txt("support_versions_current"), `:
							<i id="smfVersion" style="white-space: nowrap;">??</i><br />
							`, detailLink, `<br />`)

	c.O(`
							<br />
							<b>`, c.Txt("684"), `:</b>
							`, strings.Join(page.Administrators, ", "))
	if page.MoreAdminsLink != "" {
		c.O(`
							 (`, page.MoreAdminsLink, `)`)
	}
	c.O(`
						</td>
					</tr>
				</table>
			</td>
		</tr></table>`)

	c.O(`
		<table width="100%" cellpadding="5" cellspacing="0" border="0" class="tborder" style="margin-top: 1.5ex;">
			<tr valign="top" class="windowbg2">`)

	row := false
	for _, task := range page.QuickTasks {
		c.O(`
				<td style="padding-bottom: 2ex;" width="50%">
					<div style="font-weight: bold; font-size: 1.1em;">`, task.Link, `</div>
					`, task.Description, `
				</td>`)
		if row && !task.IsLast {
			c.O(`
			</tr>
			<tr valign="top" class="windowbg2">`)
		}
		row = !row
	}

	c.O(`
			</tr>
		</table>`)

	if c.App.SettingEmpty("disable_smf_js") {
		c.O(`
		<script language="JavaScript" type="text/javascript" src="http://www.simplemachines.org/smf/current-version.js?version=`, page.ForumVersion, `"></script>
		<script language="JavaScript" type="text/javascript" src="http://www.simplemachines.org/smf/latest-news.js?language=`, c.User.Language, `&amp;format=`, page.TimeFormat, `"></script>`)
	}

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function smfSetAnnouncements()
			{
				if (typeof(window.smfAnnouncements) == "undefined" || typeof(window.smfAnnouncements.length) == "undefined")
					return;

				var str = "<div style=\"margin: 4px; font-size: 0.85em;\">";

				for (var i = 0; i < window.smfAnnouncements.length; i++)
				{
					str += "\n	<div style=\"padding-bottom: 2px;\"><a hre" + "f=\"" + window.smfAnnouncements[i].href + "\">" + window.smfAnnouncements[i].subject + "</a> `, c.Txt("30"), ` " + window.smfAnnouncements[i].time + "</div>";
					str += "\n	<div style=\"padding-left: 2ex; margin-bottom: 1.5ex; border-top: 1px dashed;\">"
					str += "\n		" + window.smfAnnouncements[i].message;
					str += "\n	</div>";
				}

				setInnerHTML(document.getElementById("smfAnnouncements"), str + "</div>");
			}

			function smfAnnouncementsFixHeight()
			{
				if (document.getElementById("supportVersionsTable").offsetHeight)
					document.getElementById("smfAnnouncements").style.height = (document.getElementById("supportVersionsTable").offsetHeight - 10) + "px";
			}

			function smfCurrentVersion()
			{
				var smfVer, yourVer;

				if (typeof(window.smfVersion) != "string")
					return;

				smfVer = document.getElementById("smfVersion");
				yourVer = document.getElementById("yourVersion");

				setInnerHTML(smfVer, window.smfVersion);

				var currentVersion = getInnerHTML(yourVer);
				if (currentVersion != window.smfVersion)
					setInnerHTML(yourVer, "<span style=\"color: red;\">" + currentVersion + "</span>");
			}

			// Sort out the update window
			function smfUpdateAvailable()
			{
				var updateBody;

				// Nothing to declare?
				if (typeof(window.smfUpdatePackage) == "undefined")
					return;

				updateBody = document.getElementById("update_message");

				// Are we setting a custom message?
				if (typeof(window.smfUpdateNotice) != "undefined")
					setInnerHTML(updateBody, window.smfUpdateNotice);

				// Parse in the package download URL if it exists in the string.
				document.getElementById("update-link").href = "`, scripturl, `?action=pgdownload;auto;package=" + window.smfUpdatePackage + ";sesc=`, c.Sc, `";

				// If we decide to override life into "red" mode, do it.
				if (typeof(window.smfUpdateCritical) != "undefined")
				{
					document.getElementById("update_table").style.backgroundColor = "#aa2222";
					document.getElementById("update_title").style.backgroundColor = "#dd2222";
					document.getElementById("update_title").style.color = "white";
					document.getElementById("update_message").style.backgroundColor = "#eebbbb";
					document.getElementById("update_message").style.color = "black";
				}
				// And we can override the title if we really want.
				if (typeof(window.smfUpdateTitle) != "undefined")
					setInnerHTML(document.getElementById("update_title"), window.smfUpdateTitle);

				// Finally, make the box visible.
				document.getElementById("update_section").style.display = "";
			}`)

	c.O(`

			var oldonload;
			if (typeof(window.onload) != "undefined")
				oldonload = window.onload;

			window.onload = function ()
			{
				smfSetAnnouncements();
				smfCurrentVersion();
				smfUpdateAvailable();`)

	if c.Browser.IsIE && !c.Browser.IsIE4 {
		c.O(`
				if (typeof(smf_codeFix) != "undefined")
					window.detachEvent("onload", smf_codeFix);
				window.attachEvent("onload",
					function ()
					{
						with (document.all.supportVersionsTable)
							style.height = parentNode.offsetHeight;
					}
				);
				if (typeof(smf_codeFix) != "undefined")
					window.attachEvent("onload", smf_codeFix);`)
	}

	c.O(`

				if (oldonload)
					oldonload();
			}
		// ]]></script>`)
}

// templateCredits is template_credits(): the support/credits page.
func templateCredits(c *Ctx) {
	page := c.Page.(*AdminMainCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	detailLink := ""
	if page.CanAdmin {
		detailLink = ` <a href="` + scripturl + `?action=detailedversion">` + c.Txt("dvc_more") + `</a>`
	}
	c.O(`
		<table width="100%" cellpadding="5" cellspacing="0" border="0" class="tborder">
			<tr class="titlebg">
				<td>`, c.Txt("support_title"), `</td>
			</tr><tr>
				<td class="windowbg2">
					<b>`, c.Txt("support_versions"), `:</b><br />
					`, c.Txt("support_versions_forum"), `:
					<i id="yourVersion" style="white-space: nowrap;">`, page.ForumVersion, `</i>`, detailLink, `<br />
					`, c.Txt("support_versions_current"), `:
					<i id="smfVersion" style="white-space: nowrap;">??</i><br />`)

	for _, ver := range page.CurrentVersions {
		c.O(`
					`, ver.Title, `:
					<i>`, ver.Version, `</i><br />`)
	}

	c.O(`

				</td>
			</tr>
		</table>`)

	c.O(`
		<table width="100%" cellpadding="5" cellspacing="0" border="0" class="tborder" style="margin-top: 2ex;">
			<tr class="titlebg">
				<td><a href="`, scripturl, `?action=helpadmin;help=latest_support" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("support_latest"), `</td>
			</tr><tr>
				<td class="windowbg2">
					<div id="latestSupport">`, c.Txt("support_latest_fetch"), `</div>
				</td>
			</tr>
		</table>`)

	c.O(`
		<table width="100%" cellpadding="5" cellspacing="0" border="0" class="tborder" style="margin-top: 2ex;">
			<tr class="titlebg">
				<td>`, c.Txt("571"), `</td>
			</tr><tr>
				<td class="windowbg2"><span style="font-size: 0.85em;" id="credits">`, page.Credits, `</span></td>
			</tr>
		</table>`)

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var smfSupportVersions = {};

			smfSupportVersions.forum = "`, page.ForumVersion, `";`)
	for _, ver := range page.CurrentVersions {
		c.O(`
			smfSupportVersions.`, ver.Key, ` = "`, ver.Version, `";`)
	}
	c.O(`
		// ]]></script>`)

	if c.App.SettingEmpty("disable_smf_js") {
		c.O(`
		<script language="JavaScript" type="text/javascript" src="http://www.simplemachines.org/smf/current-version.js?version=`, page.ForumVersion, `"></script>
		<script language="JavaScript" type="text/javascript" src="http://www.simplemachines.org/smf/latest-news.js?language=`, c.User.Language, `&amp;format=`, page.TimeFormat, `"></script>
		<script language="JavaScript" type="text/javascript" src="http://www.simplemachines.org/smf/latest-support.js?language=`, c.User.Language, `"></script>`)
	}

	c.O(`
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			function smfSetLatestSupport()
			{
				if (window.smfLatestSupport)
					setInnerHTML(document.getElementById("latestSupport"), window.smfLatestSupport);
			}

			function smfCurrentVersion()
			{
				var smfVer, yourVer;

				if (!window.smfVersion)
					return;

				smfVer = document.getElementById("smfVersion");
				yourVer = document.getElementById("yourVersion");

				setInnerHTML(smfVer, window.smfVersion);

				var currentVersion = getInnerHTML(yourVer);
				if (currentVersion != window.smfVersion)
					setInnerHTML(yourVer, "<span style=\"color: red;\">" + currentVersion + "</span>");
			}`)

	c.O(`

			var oldonload;
			if (typeof(window.onload) != "undefined")
				oldonload = window.onload;

			window.onload = function ()
			{
				smfSetLatestSupport();
				smfCurrentVersion()

				if (oldonload)
					oldonload();
			}
		// ]]></script>`)
}

// templateMaintain is template_maintain() from Admin.template.php.
func templateMaintain(c *Ctx) {
	page := c.Page.(*MaintainCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
		<table width="100%" cellpadding="4" cellspacing="1" border="0" class="bordercolor">
			<tr class="titlebg">
				<td><a href="`, scripturl, `?action=helpadmin;help=maintenance_general" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("maintain_title"), ` - `, c.Txt("maintain_general"), `</td>
			</tr>
			<tr>
				<td class="windowbg2" style="line-height: 1.3; padding-bottom: 2ex;">
					<a href="`, scripturl, `?action=optimizetables">`, c.Txt("maintain_optimize"), `</a><br />
					<a href="`, scripturl, `?action=detailedversion">`, c.Txt("maintain_version"), `</a><br />
					<a href="`, scripturl, `?action=repairboards">`, c.Txt("maintain_errors"), `</a><br />
					<a href="`, scripturl, `?action=boardrecount">`, c.Txt("maintain_recount"), `</a><br />
					<a href="`, scripturl, `?action=maintain;sa=logs">`, c.Txt("maintain_logs"), `</a><br />`)
	if page.ConvertUTF8 {
		c.O(`
					<a href="`, scripturl, `?action=convertutf8">`, c.Txt("utf8_title"), `</a><br />`)
	}
	if page.ConvertEntity {
		c.O(`
					<a href="`, scripturl, `?action=convertentities">`, c.Txt("entity_convert_title"), `</a><br />`)
	}
	c.O(`
				</td>
			</tr>`)

	c.O(`
			<tr class="titlebg">
				<td><a href="`, scripturl, `?action=helpadmin;help=maintenance_backup" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("maintain_title"), ` - `, c.Txt("maintain_backup"), `</td>
			</tr>
			<tr>
				<td class="windowbg2" style="padding-bottom: 1ex;">
					<form action="`, scripturl, `" method="get" accept-charset="`, c.CharacterSet, `" onsubmit="return this.struct.checked || this.data.checked;">
						<label for="struct"><input type="checkbox" name="struct" id="struct" onclick="this.form.submitDump.disabled = !this.form.struct.checked &amp;&amp; !this.form.data.checked;" class="check" /> `, c.Txt("maintain_backup_struct"), `</label><br />
						<label for="data"><input type="checkbox" name="data" id="data" onclick="this.form.submitDump.disabled = !this.form.struct.checked &amp;&amp; !this.form.data.checked;" checked="checked" class="check" /> `, c.Txt("maintain_backup_data"), `</label><br />
						<br />
						<label for="compress"><input type="checkbox" name="compress" id="compress" value="gzip" checked="checked" class="check" /> `, c.Txt("maintain_backup_gz"), `</label>
						<div align="right" style="margin: 1ex;"><input type="submit" id="submitDump" value="`, c.Txt("maintain_backup_save"), `" /></div>
						<input type="hidden" name="action" value="dumpdb" />
						<input type="hidden" name="sesc" value="`, c.Sc, `" />
					</form>
				</td>
			</tr>`)

	c.O(`
			<tr class="titlebg">
				<td><a href="`, scripturl, `?action=helpadmin;help=maintenance_rot" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" align="top" /></a> `, c.Txt("maintain_title"), ` - `, c.Txt("maintain_old"), `</td>
			</tr>
			<tr>
				<td class="windowbg2">
					<a name="rotLink"></a>`)

	c.O(`
					<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
						var rotSwap = false;
						function swapRot()
						{
							rotSwap = !rotSwap;

							document.getElementById("rotIcon").src = smf_images_url + (rotSwap ? "/collapse.gif" : "/expand.gif");
							setInnerHTML(document.getElementById("rotText"), rotSwap ? "`, c.Txt("maintain_old_choose"), `" : "`, c.Txt("maintain_old_all"), `");
							document.getElementById("rotPanel").style.display = (rotSwap ? "block" : "none");

							for (var i = 0; i < document.forms.rotForm.length; i++)
							{
								if (document.forms.rotForm.elements[i].type.toLowerCase() == "checkbox" && document.forms.rotForm.elements[i].id != "delete_old_not_sticky")
									document.forms.rotForm.elements[i].checked = !rotSwap;
							}
						}
					// ]]></script>`)

	c.O(`
					<form action="`, scripturl, `?action=removeoldtopics2" method="post" accept-charset="`, c.CharacterSet, `" name="rotForm" id="rotForm">
						`, c.Txt("maintain_old_since_days1"), `<input type="text" name="maxdays" value="30" size="3" />`, c.Txt("maintain_old_since_days2"), `<br />
						<div style="padding-left: 3ex;">
							<label for="delete_type_nothing"><input type="radio" name="delete_type" id="delete_type_nothing" value="nothing" class="check" checked="checked" /> `, c.Txt("maintain_old_nothing_else"), `</label><br />
							<label for="delete_type_moved"><input type="radio" name="delete_type" id="delete_type_moved" value="moved" class="check" /> `, c.Txt("maintain_old_are_moved"), `</label><br />
							<label for="delete_type_locked"><input type="radio" name="delete_type" id="delete_type_locked" value="locked" class="check" /> `, c.Txt("maintain_old_are_locked"), `</label><br />
						</div>`)

	if !c.App.SettingEmpty("enableStickyTopics") {
		c.O(`
						<div style="padding-left: 3ex; padding-top: 1ex;">
							<label for="delete_old_not_sticky"><input type="checkbox" name="delete_old_not_sticky" id="delete_old_not_sticky" class="check" checked="checked" /> `, c.Txt("maintain_old_are_not_stickied"), `</label><br />
						</div>`)
	}

	c.O(`
						<br />
						<a href="#rotLink" onclick="swapRot();"><img src="`, imagesURL, `/expand.gif" alt="+" id="rotIcon" /></a> <a href="#rotLink" onclick="swapRot();"><span id="rotText" style="font-weight: bold;">`, c.Txt("maintain_old_all"), `</span></a>
						<div style="display: none;" id="rotPanel">
							<table width="100%" cellpadding="3" cellspacing="0" border="0">
								<tr>
									<td valign="top">`)

	middle := len(page.Categories) / 2
	for i, category := range page.Categories {
		c.O(`
										<span style="text-decoration: underline;">`, category.Name, `</span><br />`)
		for _, board := range category.Boards {
			c.O(`
										<label for="boards_`, board.ID, `"><input type="checkbox" name="boards[`, board.ID, `]" id="boards_`, board.ID, `" checked="checked" class="check" /> `, strings.Repeat("&nbsp; ", board.ChildLevel), board.Name, `</label><br />`)
		}
		c.O(`
										<br />`)
		if i+1 == middle {
			c.O(`
									</td>
									<td valign="top">`)
		}
	}

	c.O(`
								</td>
								</tr>
							</table>
						</div>

						<div align="right" style="margin: 1ex;"><input type="submit" value="`, c.Txt("maintain_old_remove"), `" onclick="return confirm('`, c.Txt("maintain_old_confirm"), `');" /></div>
						<input type="hidden" name="sc" value="`, c.Sc, `" />
					</form>
				</td>
			</tr>
		</table>`)

	if page.Finished {
		c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		setTimeout("alert(\"`, c.Txt("maintain_done"), `\")", 120);
	// ]]></script>`)
	}
}

// templateNotDone is template_not_done() from Admin.template.php: the
// auto-refreshing progress page for chunked operations.
func templateNotDone(c *Ctx) {
	page := c.Page.(*NotDoneCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<div class="tborder">
		<div class="titlebg" style="padding: 4px;">`, c.Txt("not_done_title"), `</div>
		<div class="windowbg" style="padding: 4px;">
			`, c.Txt("not_done_reason"))

	if page.ContinuePercent != 0 {
		pad := "1pt"
		if c.Browser.IsSafari || c.Browser.IsKonqueror {
			pad = "2pt"
		}
		c.O(`
			<div style="padding-left: 20%; padding-right: 20%; margin-top: 1ex;">
				<div style="font-size: 8pt; height: 12pt; border: 1px solid black; background-color: white; padding: 1px; position: relative;">
					<div style="padding-top: `, pad, `; width: 100%; z-index: 2; color: black; position: absolute; text-align: center; font-weight: bold;">`, page.ContinuePercent, `%</div>
					<div style="width: `, page.ContinuePercent, `%; height: 12pt; z-index: 1; background-color: red;">&nbsp;</div>
				</div>
			</div>`)
	}

	c.O(`
			<form action="`, scripturl, page.ContinueGetData, `" method="post" accept-charset="`, c.CharacterSet, `" style="margin: 0;" name="autoSubmit" id="autoSubmit">
				<div style="margin: 1ex; text-align: right;"><input type="submit" name="cont" value="`, c.Txt("not_done_continue"), `" /></div>
				`, page.ContinuePostData, `
			</form>
		</div>
	</div>
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		var countdown = `, page.ContinueCountdown, `;
		doAutoSubmit();

		function doAutoSubmit()
		{
			if (countdown == 0)
				document.forms.autoSubmit.submit();
			else if (countdown == -1)
				return;

			document.forms.autoSubmit.cont.value = "`, c.Txt("not_done_continue"), ` (" + countdown + ")";
			countdown--;

			setTimeout("doAutoSubmit();", 1000);
		}
	// ]]></script>`)
}

// templateShowSettings is template_show_settings() from Admin.template.php:
// the generic settings form used by ManageServer/ModSettings and the other
// Manage* modules.
func templateShowSettings(c *Ctx) {
	page := c.Page.(*SettingsCtx)
	scripturl := c.App.ScriptURL
	imagesURL := c.Theme.ImagesURL()

	c.O(`
	<form action="`, page.PostURL, `" method="post" accept-charset="`, c.CharacterSet, `">
		<table width="80%" border="0" cellspacing="0" cellpadding="0" class="tborder" align="center">
			<tr><td>
				<table border="0" cellspacing="0" cellpadding="4" width="100%">
					<tr class="titlebg">
						<td colspan="3">`, page.SettingsTitle, `</td>
					</tr>`)

	if page.SettingsMessage != "" {
		c.O(`
					<tr>
						<td class="windowbg2" colspan="3">`, page.SettingsMessage, `</td>
					</tr>`)
	}

	for _, cv := range page.ConfigVars {
		c.O(`
					<tr class="windowbg2">`)

		if cv.IsVar {
			if cv.Help != "" {
				c.O(`
						<td class="windowbg2" valign="top" width="16"><a href="`, scripturl, `?action=helpadmin;help=`, cv.Help, `" onclick="return reqWin(this.href);" class="help"><img src="`, imagesURL, `/helptopics.gif" alt="`, c.Txt("119"), `" border="0" align="top" /></a></td>`)
			} else {
				c.O(`
						<td class="windowbg2"></td>`)
			}

			disabledStyle := ""
			if cv.Disabled {
				disabledStyle = ` style="color: #777777;"`
			}
			passwordConfirm := ""
			if cv.Type == "password" {
				passwordConfirm = `<br /><i>` + c.Txt("admin_confirm_password") + `</i>`
			}
			c.O(`
						<td valign="top" `, disabledStyle, `><label for="`, cv.Name, `">`, cv.Label, passwordConfirm, `</label></td>
						<td class="windowbg2" width="50%">`)

			disabledAttr := ""
			if cv.Disabled {
				disabledAttr = ` disabled="disabled"`
			}
			sizeAttr := ""
			if cv.Size != 0 {
				sizeAttr = ` size="` + itoa(cv.Size) + `"`
			}

			switch cv.Type {
			case "check":
				checked := ""
				if cv.Value != "" && cv.Value != "0" {
					checked = ` checked="checked"`
				}
				c.O(`
							<input type="hidden" name="`, cv.Name, `" value="0" /><input type="checkbox"`, disabledAttr, ` name="`, cv.Name, `" id="`, cv.Name, `" `, checked, ` class="check" />`)
			case "password":
				c.O(`
							<input type="password"`, disabledAttr, ` name="`, cv.Name, `[0]"`, sizeAttr, ` value="*#fakepass#*" onfocus="this.value = ''; this.form.`, cv.Name, `.disabled = false;" /><br />
							<input type="password" disabled="disabled" id="`, cv.Name, `" name="`, cv.Name, `[1]"`, sizeAttr, ` />`)
			case "select":
				c.O(`
							<select name="`, cv.Name, `"`, disabledAttr, `>`)
				for _, opt := range cv.Data {
					selected := ""
					if opt[0] == cv.Value {
						selected = ` selected="selected"`
					}
					c.O(`
								<option value="`, opt[0], `"`, selected, `>`, opt[1], `</option>`)
				}
				c.O(`
							</select>`)
			case "large_text":
				rows := cv.Size
				if rows == 0 {
					rows = 4
				}
				c.O(`
							<textarea rows="`, rows, `" cols="30" `, disabledAttr, ` name="`, cv.Name, `">`, cv.Value, `</textarea>`)
			default:
				c.O(`
							<input type="text"`, disabledAttr, ` name="`, cv.Name, `" value="`, cv.Value, `"`, sizeAttr, ` />`)
			}

			c.O(`
						</td>`)
		} else if cv.Separator {
			c.O(`
							<td colspan="3" class="windowbg2"><hr size="1" width="100%" class="hrcolor" /></td>`)
		} else {
			c.O(`
							<td colspan="3" class="windowbg2" align="center"><b>`, cv.Title, `</b></td>`)
		}
		c.O(`
					</tr>`)
	}

	saveDisabled := ""
	if page.SaveDisabled {
		saveDisabled = ` disabled="disabled"`
	}
	c.O(`
					</tr><tr>
						<td class="windowbg2" colspan="3" align="center" valign="middle"><input type="submit" value="`, c.Txt("10"), `"`, saveDisabled, ` /></td>
					</tr>
				</table>
			</td></tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
