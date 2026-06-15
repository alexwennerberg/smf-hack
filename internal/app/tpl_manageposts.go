package app

// Ports of the ManagePosts templates from Admin.template.php:
// template_edit_censored / edit_post_settings / edit_bbc_settings /
// edit_topic_settings. The settings templates read modSettings directly.

// checkboxAttr returns ` checked="checked"` when the setting is non-empty.
func (c *Ctx) checkboxAttr(name string) string {
	if !c.App.SettingEmpty(name) {
		return ` checked="checked"`
	}
	return ""
}

// settingOrZero returns the setting value, or "0" if empty.
func (c *Ctx) settingOrZero(name string) string {
	if c.App.SettingEmpty(name) {
		return "0"
	}
	return c.App.Setting(name)
}

// templateEditCensored is template_edit_censored().
func templateEditCensored(c *Ctx) {
	page := c.Page.(*CensorCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=postsettings;sa=censor" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td colspan="2">`, c.Txt("135"), `</td>
				</tr><tr class="windowbg2">
					<td align="center">
						<table width="100%">
							<tr>
								<td colspan="2" align="center">
									`, c.Txt("136"), `<br />`)

	for _, w := range page.Words {
		c.O(`
									<div style="margin-top: 1ex;"><input type="text" name="censor_vulgar[]" value="`, w[0], `" size="20" /> => <input type="text" name="censor_proper[]" value="`, w[1], `" size="20" /></div>`)
	}

	c.O(`
									<noscript>
										<div style="margin-top: 1ex;"><input type="text" name="censor_vulgar[]" size="20" /> => <input type="text" name="censor_proper[]" size="20" /></div>
									</noscript>
									<div id="moreCensoredWords"></div><div style="margin-top: 1ex; display: none;" id="moreCensoredWords_link"><a href="#;" onclick="addNewWord(); return false;">`, c.Txt("censor_clickadd"), `</a></div>
									<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
										document.getElementById("moreCensoredWords_link").style.display = "";

										function addNewWord()
										{
											setOuterHTML(document.getElementById("moreCensoredWords"), '<div style="margin-top: 1ex;"><input type="text" name="censor_vulgar[]" size="20" /> => <input type="text" name="censor_proper[]" size="20" /></div><div id="moreCensoredWords"></div>');
										}
									// ]]></script><br />
								</td>
							</tr><tr>
								<td colspan="2"><hr /></td>
							</tr><tr>
								<th width="50%" align="right"><label for="censorWholeWord_check">`, c.Txt("smf231"), `:</label></th>
								<td align="left"><input type="checkbox" name="censorWholeWord" value="1" id="censorWholeWord_check"`, c.checkboxAttr("censorWholeWord"), ` class="check" /></td>
							</tr><tr>
								<th align="right"><label for="censorIgnoreCase_check">`, c.Txt("censor_case"), `:</label></th>
								<td align="left">
									<input type="checkbox" name="censorIgnoreCase" value="1" id="censorIgnoreCase_check"`, c.checkboxAttr("censorIgnoreCase"), ` class="check" />
								</td>
							</tr><tr>
								<td colspan="2" align="right">
									<input type="submit" name="save_censor" value="`, c.Txt("10"), `" />
								</td>
							</tr>
						</table>
					</td>
				</tr>
			</table>

			<br />`)

	c.O(`
			<table width="600" cellpadding="4" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>`, c.Txt("censor_test"), `</td>
				</tr><tr class="windowbg2">
					<td align="center">
						<input type="text" name="censortest" value="`, page.CensorTest, `" />
						<input type="submit" value="`, c.Txt("censor_test_save"), `" />
					</td>
				</tr>
			</table>

			<input type="hidden" name="sc" value="`, c.Sc, `" />
		</form>`)
}

// settingsRow renders one (?)-helped checkbox or text setting row used by the
// post/topic settings templates.
func (c *Ctx) helpLink(help string) string {
	return ` <span style="font-weight: normal;">(<a href="` + c.App.ScriptURL + `?action=helpadmin;help=` + help + `" onclick="return reqWin(this.href);">?</a>)</span>`
}

// templateEditPostSettings is template_edit_post_settings().
func templateEditPostSettings(c *Ctx) {
	page := c.Page.(*PostSettingsCtx)

	c.O(`
	<form action="`, c.App.ScriptURL, `?action=postsettings;sa=posts" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("manageposts_settings"), `</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right"><label for="removeNestedQuotes_check">`, c.Txt("removeNestedQuotes"), `</label>`, c.helpLink("removeNestedQuotes"), `:</th>
				<td>
					<input type="checkbox" name="removeNestedQuotes" id="removeNestedQuotes_check"`, c.checkboxAttr("removeNestedQuotes"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="enableEmbeddedFlash_check">`, c.Txt("enableEmbeddedFlash"), `</label>`, c.helpLink("enableEmbeddedFlash"), `:
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("enableEmbeddedFlash_warning"), `</div>
				</th>
				<td valign="top">
					<input type="checkbox" name="enableEmbeddedFlash" id="enableEmbeddedFlash_check"`, c.checkboxAttr("enableEmbeddedFlash"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="enableSpellChecking_check">`, c.Txt("enableSpellChecking"), `</label>`, c.helpLink("enableSpellChecking"), `:
					<div class="smalltext" style="font-weight: normal;`)
	if !page.SpellcheckInstalled {
		c.O(` color: red;`)
	}
	c.O(`">`, c.Txt("enableSpellChecking_warning"), `</div>
				</th>
				<td valign="top">
					<input type="checkbox" name="enableSpellChecking" id="enableSpellChecking_check"`, c.checkboxAttr("enableSpellChecking"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="max_messageLength_input">`, c.Txt("max_messageLength"), `</label>:
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("max_messageLength_zero"), `</div>
				</th>
				<td valign="top">
					<input type="text" name="max_messageLength" id="max_messageLength_input" value="`, c.settingOrZero("max_messageLength"), `" size="5" /> `, c.Txt("manageposts_characters"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="fixLongWords_input">`, c.Txt("fixLongWords"), `</label>`, c.helpLink("fixLongWords"), `:
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("fixLongWords_zero"), `</div>
				</th>
				<td valign="top">
					<input type="text" name="fixLongWords" id="fixLongWords_input" value="`, c.settingOrZero("fixLongWords"), `" size="5" /> `, c.Txt("manageposts_characters"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="topicSummaryPosts_input">`, c.Txt("topicSummaryPosts"), `</label>`, c.helpLink("topicSummaryPosts"), `:
				</th>
				<td valign="top">
					<input type="text" name="topicSummaryPosts" id="topicSummaryPosts_input" value="`, c.settingOrZero("topicSummaryPosts"), `" size="5" /> `, c.Txt("manageposts_posts"), `
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="spamWaitTime_input">`, c.Txt("spamWaitTime"), `</label>`, c.helpLink("spamWaitTime"), `:
				</th>
				<td valign="top">
					<input type="text" name="spamWaitTime" id="spamWaitTime_input" value="`, c.settingOrZero("spamWaitTime"), `" size="5" /> `, c.Txt("manageposts_seconds"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="edit_wait_time_input">`, c.Txt("edit_wait_time"), `</label>`, c.helpLink("edit_wait_time"), `:
				</th>
				<td valign="top">
					<input type="text" name="edit_wait_time" id="edit_wait_time_input" value="`, c.settingOrZero("edit_wait_time"), `" size="5" /> `, c.Txt("manageposts_seconds"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="edit_disable_time_input">`, c.Txt("edit_disable_time"), `</label>`, c.helpLink("edit_disable_time"), `:
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("edit_disable_time_zero"), `</div>
				</th>
				<td valign="top">
					<input type="text" name="edit_disable_time" id="edit_disable_time_input" value="`, c.settingOrZero("edit_disable_time"), `" size="5" /> `, c.Txt("manageposts_minutes"), `
				</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("manageposts_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}

// templateEditBBCSettings is template_edit_bbc_settings().
func templateEditBBCSettings(c *Ctx) {
	page := c.Page.(*BBCSettingsCtx)
	scripturl := c.App.ScriptURL

	c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		function toggleBBCDisabled(disable)
		{
			for (var i = 0; i < document.forms.bbcForm.length; i++)
			{
				if (typeof(document.forms.bbcForm[i].name) == "undefined" || (document.forms.bbcForm[i].name.substr(0, 11) != "enabledTags"))
					continue;

				document.forms.bbcForm[i].disabled = disable;
			}
			document.getElementById("select_all").disabled = disable;
		}
	// ]]></script>

	<form action="`, scripturl, `?action=postsettings;sa=bbc" method="post" accept-charset="`, c.CharacterSet, `" name="bbcForm" id="bbcForm" onsubmit="toggleBBCDisabled(false);">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("manageposts_bbc_settings_title"), `</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right"><label for="enableBBC_check">`, c.Txt("enableBBC"), `</label>`, c.helpLink("enableBBC"), `:</th>
				<td>
					<input type="checkbox" name="enableBBC" id="enableBBC_check"`, c.checkboxAttr("enableBBC"), ` onchange="toggleBBCDisabled(!this.checked);" class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right"><label for="enablePostHTML_check">`, c.Txt("enablePostHTML"), `</label>`, c.helpLink("enablePostHTML"), `:</th>
				<td>
					<input type="checkbox" name="enablePostHTML" id="enablePostHTML_check"`, c.checkboxAttr("enablePostHTML"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right"><label for="autoLinkUrls_check">`, c.Txt("autoLinkUrls"), `</label>:</th>
				<td>
					<input type="checkbox" name="autoLinkUrls" id="autoLinkUrls_check"`, c.checkboxAttr("autoLinkUrls"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right" valign="top"><label for="enabledBBCTags">`, c.Txt("bbcTagsToUse"), `</label>:</th>
				<td>
					<fieldset id="enabledBBCTags">
						<legend>`, c.Txt("bbcTagsToUse_select"), `</legend>
						<table width="100%"><tr>`)
	for _, bbcColumn := range page.Columns {
		c.O(`
							<td valign="top">`)
		for _, bbcTag := range bbcColumn {
			checked := ""
			if bbcTag.IsEnabled {
				checked = ` checked="checked"`
			}
			helpSuffix := ""
			if bbcTag.ShowHelp {
				helpSuffix = ` (<a href="` + scripturl + `?action=helpadmin;help=tag_` + bbcTag.Tag + `" onclick="return reqWin(this.href);">?</a>)`
			}
			c.O(`
								<input type="checkbox" name="enabledTags[]" id="tag_`, bbcTag.Tag, `" value="`, bbcTag.Tag, `"`, checked, ` class="check" /> <label for="tag_`, bbcTag.Tag, `">`, bbcTag.Tag, `</label>`, helpSuffix, `<br />`)
		}
		c.O(`
							</td>`)
	}
	allChecked := ""
	if page.AllSelected {
		allChecked = ` checked="checked"`
	}
	c.O(`
						</tr></table><br />
						<input type="checkbox" id="select_all" onclick="invertAll(this, this.form, 'enabledTags');"`, allChecked, ` class="check" /> <label for="select_all"><i>`, c.Txt("bbcTagsToUse_select_all"), `</i></label>
					</fieldset>
				</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("manageposts_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)

	if c.App.SettingEmpty("enableBBC") {
		c.O(`
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		toggleBBCDisabled(true);
	// ]]></script>`)
	}
}

// templateEditTopicSettings is template_edit_topic_settings().
func templateEditTopicSettings(c *Ctx) {
	c.O(`
	<form action="`, c.App.ScriptURL, `?action=postsettings;sa=topics" method="post" accept-charset="`, c.CharacterSet, `">
		<table border="0" cellspacing="0" cellpadding="4" align="center" width="80%" class="tborder">
			<tr class="titlebg">
				<td colspan="2">`, c.Txt("manageposts_topic_settings"), `</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="enableStickyTopics_check">`, c.Txt("enableStickyTopics"), `</label>`, c.helpLink("enableStickyTopics"), `:
				</th>
				<td valign="top">
					<input type="checkbox" name="enableStickyTopics" id="enableStickyTopics_check"`, c.checkboxAttr("enableStickyTopics"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="enableParticipation_check">`, c.Txt("enableParticipation"), `</label>`, c.helpLink("enableParticipation"), `:
				</th>
				<td valign="top">
					<input type="checkbox" name="enableParticipation" id="enableParticipation_check"`, c.checkboxAttr("enableParticipation"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="oldTopicDays_input">`, c.Txt("oldTopicDays"), `</label>`, c.helpLink("oldTopicDays"), `:
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("oldTopicDays_zero"), `</div>
				</th>
				<td valign="top">
					<input type="text" name="oldTopicDays" id="oldTopicDays_input" value="`, c.settingOrZero("oldTopicDays"), `" size="5" /> `, c.Txt("manageposts_days"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="defaultMaxTopics_input">`, c.Txt("defaultMaxTopics"), `</label>:
				</th>
				<td valign="top">
					<input type="text" name="defaultMaxTopics" id="defaultMaxTopics_input" value="`, c.settingOrZero("defaultMaxTopics"), `" size="5" /> `, c.Txt("manageposts_topics"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="defaultMaxMessages_input">`, c.Txt("defaultMaxMessages"), `</label>:
				</th>
				<td valign="top">
					<input type="text" name="defaultMaxMessages" id="defaultMaxMessages_input" value="`, c.settingOrZero("defaultMaxMessages"), `" size="5" /> `, c.Txt("manageposts_posts"), `
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="hotTopicPosts_input">`, c.Txt("hotTopicPosts"), `</label>`, c.helpLink("hotTopicPosts"), `:
				</th>
				<td valign="top">
					<input type="text" name="hotTopicPosts" id="hotTopicPosts_input" value="`, c.settingOrZero("hotTopicPosts"), `" size="5" /> `, c.Txt("manageposts_posts"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="hotTopicVeryPosts_input">`, c.Txt("hotTopicVeryPosts"), `</label>`, c.helpLink("hotTopicPosts"), `:
				</th>
				<td valign="top">
					<input type="text" name="hotTopicVeryPosts" id="hotTopicVeryPosts_input" value="`, c.settingOrZero("hotTopicVeryPosts"), `" size="5" /> `, c.Txt("manageposts_posts"), `
				</td>
			</tr><tr class="windowbg2">
				<td colspan="2"><hr /></td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="enableAllMessages_input">`, c.Txt("enableAllMessages"), `</label>`, c.helpLink("enableAllMessages"), `:
					<div class="smalltext" style="font-weight: normal;">`, c.Txt("enableAllMessages_zero"), `</div>
				</th>
				<td valign="top">
					<input type="text" name="enableAllMessages" id="enableAllMessages_input" value="`, c.settingOrZero("enableAllMessages"), `" size="5" /> `, c.Txt("manageposts_posts"), `
				</td>
			</tr><tr class="windowbg2">
				<th width="50%" align="right">
					<label for="enablePreviousNext_check">`, c.Txt("enablePreviousNext"), `</label>`, c.helpLink("enablePreviousNext"), `:
				</th>
				<td valign="top">
					<input type="checkbox" name="enablePreviousNext" id="enablePreviousNext_check"`, c.checkboxAttr("enablePreviousNext"), ` class="check" />
				</td>
			</tr><tr class="windowbg2">
				<td align="right" colspan="2">
					<input type="submit" name="save_settings" value="`, c.Txt("manageposts_settings_submit"), `" />
				</td>
			</tr>
		</table>
		<input type="hidden" name="sc" value="`, c.Sc, `" />
	</form>`)
}
