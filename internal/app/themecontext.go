package app

// Port of setupThemeContext() from Subs.php (line 3062).

import (
	"math/rand"
	"strings"
)

func (c *Ctx) setupThemeContext() {
	a := c.App

	// Get some news...
	newsRaw := strings.ReplaceAll(strings.TrimSpace(a.Setting("news")), "\r", "")
	c.NewsLines = nil
	c.FaderNewsLines = nil
	for i, line := range strings.Split(newsRaw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parsed := c.parseBBCCached(strings.TrimSpace(line), true, "news"+itoa(i))
		c.NewsLines = append(c.NewsLines, parsed)
		// Gotta be special for the javascript.
		fader := strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`, "/", `\/`, "<a href=", `<a hre" + "f=`).Replace(parsed)
		c.FaderNewsLines = append(c.FaderNewsLines, fader)
	}
	if len(c.NewsLines) > 0 {
		c.RandomNewsLine = c.NewsLines[rand.Intn(len(c.NewsLines))]
	}

	if !c.User.IsGuest {
		// Personal message popup...
		c.PopupMessages = c.User.UnreadMessages > int(c.Session.GetInt("unread_messages"))
		c.Session.Set("unread_messages", c.User.UnreadMessages)

		if c.allowedTo("moderate_forum") && a.Setting("registration_method") == "2" {
			c.UnapprovedMembers = a.SettingInt("unapprovedMembers")
		}

		// Figure out the avatar... uploaded?
		c.UserAvatar = UserAvatarCtx{}
		switch {
		case c.User.AvatarURL == "" && c.User.AvatarIDAttach != 0:
			if c.User.AvatarCustomDir {
				c.UserAvatar.Href = a.Setting("custom_avatar_url") + "/" + c.User.AvatarFilename
			} else {
				c.UserAvatar.Href = a.ScriptURL + "?action=dlattach;attach=" + itoa(c.User.AvatarIDAttach) + ";type=avatar"
			}
		case strings.HasPrefix(c.User.AvatarURL, "http://"):
			c.UserAvatar.Href = c.User.AvatarURL
			if tooLarge := a.Setting("avatar_action_too_large"); tooLarge == "option_html_resize" || tooLarge == "option_js_resize" {
				if !a.SettingEmpty("avatar_max_width_external") {
					c.UserAvatar.Width = a.Setting("avatar_max_width_external")
				}
				if !a.SettingEmpty("avatar_max_height_external") {
					c.UserAvatar.Height = a.Setting("avatar_max_height_external")
				}
			}
		case c.User.AvatarURL != "":
			c.UserAvatar.Href = a.Setting("avatar_url") + "/" + Htmlspecialchars(c.User.AvatarURL)
		}
		if c.UserAvatar.Href != "" {
			img := `<img src="` + c.UserAvatar.Href + `"`
			if c.UserAvatar.Width != "" {
				img += ` width="` + c.UserAvatar.Width + `"`
			}
			if c.UserAvatar.Height != "" {
				img += ` height="` + c.UserAvatar.Height + `"`
			}
			img += ` alt="" class="avatar" border="0" />`
			c.UserAvatar.Image = img
		}

		// Figure out how long they've been logged in.
		t := c.User.TotalTimeLoggedIn
		c.TimeLoggedIn.Days = int(t / 86400)
		c.TimeLoggedIn.Hours = int(t % 86400 / 3600)
		c.TimeLoggedIn.Minutes = int(t % 3600 / 60)
	} else {
		c.PopupMessages = false
		c.UserAvatar = UserAvatarCtx{}
		c.TimeLoggedIn = struct{ Days, Hours, Minutes int }{}

		c.WelcomeGuest = c.Txt("welcome_guest")
		if a.Setting("registration_method") == "1" {
			c.WelcomeGuest += c.Txt("welcome_guest_activate")
		}

		// If we've upgraded recently, go easy on the passwords.
		if !a.SettingEmpty("disableHashTime") {
			c.DisableLoginHash = true
		} else if c.Browser.IsIE5 || c.Browser.IsIE55 {
			c.DisableLoginHash = true
		}
	}
	if c.WelcomeGuest == "" {
		c.WelcomeGuest = c.Txt("welcome_guest")
	}

	// Set up the menu privileges.
	c.AllowSearch = c.allowedTo("search_posts")
	c.AllowAdmin = c.allowedTo("admin_forum", "manage_boards", "manage_permissions", "moderate_forum",
		"manage_membergroups", "manage_bans", "send_mail", "edit_news", "manage_attachments", "manage_smileys")
	c.AllowEditProfile = !c.User.IsGuest && c.allowedTo("profile_view_own", "profile_view_any",
		"profile_identity_own", "profile_identity_any", "profile_extra_own", "profile_extra_any",
		"profile_remove_own", "profile_remove_any", "moderate_forum", "manage_membergroups")
	c.AllowMemberlist = c.allowedTo("view_mlist")
	c.AllowCalendar = c.allowedTo("calendar_view") && !a.SettingEmpty("cal_enabled")
	c.AllowPM = c.allowedTo("pm_read")

	c.InMaintenance = c.App.Config.Maintenance != 0
	c.CurrentTime = c.timeformatNoToday(nowUnix())
	c.CurrentAction = c.GET.Str("action")
	c.ShowQuickLogin = !a.SettingEmpty("enableVBStyleLogin") && c.User.IsGuest

	// This is done to make it easier to add to all themes...
	if c.PopupMessages && !empty(c.Options["popup_messages"]) && c.REQUEST.Str("action") != "pm" {
		c.HTMLHeaders += `
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		if (confirm("` + c.Txt("show_personal_messages") + `"))
			window.open("` + a.ScriptURL + `?action=pm");
	// ]]></script>`
	}

	// Resize avatars the fancy, but non-GD requiring way.
	if a.Setting("avatar_action_too_large") == "option_js_resize" &&
		(!a.SettingEmpty("avatar_max_width_external") || !a.SettingEmpty("avatar_max_height_external")) {
		c.HTMLHeaders += `
	<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
		var smf_avatarMaxWidth = ` + itoa(a.SettingInt("avatar_max_width_external")) + `;
		var smf_avatarMaxHeight = ` + itoa(a.SettingInt("avatar_max_height_external")) + `;`
		if !c.Browser.IsIE && !c.Browser.IsMacIE {
			c.HTMLHeaders += `
	window.addEventListener("load", smf_avatarResize, false);`
		} else {
			c.HTMLHeaders += `
	var window_oldAvatarOnload = window.onload;
	window.onload = smf_avatarResize;`
		}
		c.HTMLHeaders += `
	// ]]></script>`
	}

	// This looks weird, but it's because BoardIndex.php references the
	// variable.
	c.CommonStats = CommonStats{
		TotalPosts:       c.commaFormat(a.SettingInt("totalMessages")),
		TotalTopics:      c.commaFormat(a.SettingInt("totalTopics")),
		TotalMembers:     c.commaFormat(a.SettingInt("totalMembers")),
		LatestMemberID:   a.SettingInt("latestMember"),
		LatestMemberName: a.Setting("latestRealName"),
	}
	c.CommonStats.LatestMemberHref = a.ScriptURL + "?action=profile;u=" + itoa(c.CommonStats.LatestMemberID)
	c.CommonStats.LatestMemberLink = `<a href="` + c.CommonStats.LatestMemberHref + `">` + c.CommonStats.LatestMemberName + `</a>`
}
