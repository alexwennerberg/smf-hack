package app

// Port of Sources/ManageNews.php: the news admin (?action=news) — EditNews
// (the forum news lines), ModifyNewsSettings (xml-news settings + the inline
// edit_news/send_mail permissions), and the newsletter / mass-email flow
// (SelectMailingMembers -> ComposeMailing -> SendMailing). Mail goes out via
// the plain-text sendmail port (HTML/MIME alternative bodies are a Phase 8
// item, as elsewhere); the send_html/parse_html message munging is preserved.

import (
	"math"
	"regexp"
	"strings"
)

func init() {
	registerAction("news", (*Ctx).ManageNews)
}

func (c *Ctx) ManageNews() {
	c.isAllowedTo("edit_news", "send_mail", "admin_forum")
	c.adminIndex("news")

	type subAction struct {
		fn   func()
		perm string
	}
	subs := map[string]subAction{
		"editnews":       {c.EditNews, "edit_news"},
		"mailingmembers": {c.SelectMailingMembers, "send_mail"},
		"mailingcompose": {c.ComposeMailing, "send_mail"},
		"mailingsend":    {c.SendMailing, "send_mail"},
		"settings":       {c.ModifyNewsSettings, "admin_forum"},
	}
	sa := c.REQUEST.Str("sa")
	if _, ok := subs[sa]; !ok {
		if c.allowedTo("edit_news") {
			sa = "editnews"
		} else if c.allowedTo("send_mail") {
			sa = "mailingmembers"
		} else {
			sa = "settings"
		}
	}
	c.isAllowedTo(subs[sa].perm)

	scripturl := c.App.ScriptURL
	tabs := &AdminTabs{Title: c.Txt("news_title"), Help: "edit_news", Description: c.Txt("670")}
	if c.allowedTo("edit_news") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("7"), Description: c.Txt("670"), Href: scripturl + "?action=news", IsSelected: sa == "editnews"})
	}
	if c.allowedTo("send_mail") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("6"), Description: c.Txt("news_mailing_desc"), Href: scripturl + "?action=news;sa=mailingmembers", IsSelected: strings.HasPrefix(sa, "mailing")})
	}
	if c.allowedTo("admin_forum") {
		tabs.Tabs = append(tabs.Tabs, AdminTab{Title: c.Txt("settings"), Description: c.Txt("news_settings_desc"), Href: scripturl + "?action=news;sa=settings", IsSelected: sa == "settings"})
	}
	if len(tabs.Tabs) > 0 {
		tabs.Tabs[len(tabs.Tabs)-1].IsLast = true
	}
	c.AdminTabs = tabs

	subs[sa].fn()
}

// NewsItem is one current news line.
type NewsItem struct {
	ID       int
	Unparsed string
	Parsed   string
}

// EditNewsCtx backs template_edit_news.
type EditNewsCtx struct {
	News []NewsItem
}

var reNewsForm = regexp.MustCompile(`(?i)<([/]?)form[^>]*?[>]*>`)

func (c *Ctx) EditNews() {
	a := c.App

	if c.POST.Has("delete_selection") && c.POST.Arr("remove") != nil {
		c.checkSession("post", "", true)
		remove := map[int]bool{}
		c.POST.Arr("remove").Values(func(k string, v any) {
			s, _ := v.(string)
			remove[atoi(s)] = true
		})
		var kept []string
		for i, line := range strings.Split(a.Setting("news"), "\n") {
			if !remove[i] {
				kept = append(kept, line)
			}
		}
		a.UpdateSettings(map[string]string{"news": strings.Join(kept, "\n")})
		c.logAction("news", nil)
	} else if c.POST.Has("save_items") {
		c.checkSession("post", "", true)
		var items []string
		if arr := c.POST.Arr("news"); arr != nil {
			arr.Values(func(k string, v any) {
				s, _ := v.(string)
				if strings.TrimSpace(s) == "" {
					return
				}
				items = append(items, c.preparsecode(Htmlspecialchars(s), false))
			})
		}
		a.UpdateSettings(map[string]string{"news": strings.Join(items, "\n")})
		c.logAction("news", nil)
	}

	page := &EditNewsCtx{}
	c.Page = page
	for id, line := range strings.Split(a.Setting("news"), "\n") {
		page.News = append(page.News, NewsItem{
			ID:       id,
			Unparsed: c.unPreparsecode(line),
			Parsed:   reNewsForm.ReplaceAllString(c.parseBBC(line, true), `<em class="smalltext">&lt;$1form&gt;</em>`),
		})
	}

	c.SubTemplate = templateEditNews
	c.PageTitle = c.Txt("7")
}

func (c *Ctx) ModifyNewsSettings() {
	a := c.App
	c.PageTitle = c.Txt("7") + " - " + c.Txt("settings")
	c.SubTemplate = templateNewsSettings

	if c.POST.Has("save_settings") {
		c.checkSession("post", "", true)
		enable := "0"
		if c.POST.Has("xmlnews_enable") {
			enable = "1"
		}
		a.UpdateSettings(map[string]string{
			"xmlnews_enable": enable,
			"xmlnews_maxlen": itoa(c.POST.Int("xmlnews_maxlen")),
		})
		c.saveInlinePermissions([]string{"edit_news", "send_mail"})
	}
	c.initInlinePermissions([]string{"edit_news", "send_mail"}, []int{-1})
}

// MailGroup is one membergroup row in the email-members group picker.
type MailGroup struct {
	ID          int
	Name        string
	MemberCount int
}

// EmailMembersCtx backs template_email_members.
type EmailMembersCtx struct {
	Groups    []*MailGroup
	CanSendPM bool
}

// SelectMailingMembers is SelectMailingMembers(): the group picker.
func (c *Ctx) SelectMailingMembers() {
	a := c.App
	c.PageTitle = c.Txt("6")

	page := &EmailMembersCtx{}
	c.Page = page
	byID := map[int]*MailGroup{}
	var postGroups, normalGroups []int

	// If we have post groups disabled then we need a "ungrouped members" option.
	if a.SettingEmpty("permission_enable_postgroups") {
		g := &MailGroup{ID: 0, Name: c.Txt("membergroups_members")}
		page.Groups = append(page.Groups, g)
		byID[0] = g
		normalGroups = append(normalGroups, 0)
	}

	// Get all the extra groups as well as Administrator and Global Moderator.
	where := ""
	if a.SettingEmpty("permission_enable_postgroups") {
		where = "\n\t\tWHERE mg.minPosts = -1"
	}
	rows, err := a.DB.Query(a.Q(`
		SELECT mg.ID_GROUP, mg.groupName, mg.minPosts
		FROM {$db_prefix}membergroups AS mg` + where + `
		GROUP BY mg.ID_GROUP
		ORDER BY mg.minPosts, IIF(mg.ID_GROUP < 4, mg.ID_GROUP, 4), mg.groupName`))
	if err == nil {
		for rows.Next() {
			var id, minPosts int
			var name string
			rows.Scan(&id, &name, &minPosts)
			g := &MailGroup{ID: id, Name: name}
			page.Groups = append(page.Groups, g)
			byID[id] = g
			if minPosts == -1 {
				normalGroups = append(normalGroups, id)
			} else {
				postGroups = append(postGroups, id)
			}
		}
		rows.Close()
	}

	add := func(id, n int) {
		if g, ok := byID[id]; ok {
			g.MemberCount += n
		}
	}

	// If we have post groups, count the number of members...
	if len(postGroups) > 0 {
		r, err := a.DB.Query(a.Q(`
			SELECT mem.ID_POST_GROUP AS ID_GROUP, COUNT(*) AS member_count
			FROM {$db_prefix}members AS mem
			WHERE mem.ID_POST_GROUP IN (` + joinInts(postGroups) + `)
			GROUP BY mem.ID_POST_GROUP`))
		if err == nil {
			for r.Next() {
				var id, n int
				r.Scan(&id, &n)
				add(id, n)
			}
			r.Close()
		}
	}

	if len(normalGroups) > 0 {
		// Find people who are members of this group...
		r, err := a.DB.Query(a.Q(`
			SELECT ID_GROUP, COUNT(*) AS member_count
			FROM {$db_prefix}members
			WHERE ID_GROUP IN (` + joinInts(normalGroups) + `)
			GROUP BY ID_GROUP`))
		if err == nil {
			for r.Next() {
				var id, n int
				r.Scan(&id, &n)
				add(id, n)
			}
			r.Close()
		}

		// Also those who have it as an additional membergroup - yuckier...
		r2, err := a.DB.Query(a.Q(`
			SELECT mg.ID_GROUP, COUNT(*) AS member_count
			FROM {$db_prefix}membergroups AS mg, {$db_prefix}members AS mem
			WHERE mg.ID_GROUP IN (` + joinInts(normalGroups) + `)
				AND mem.additionalGroups != ''
				AND mem.ID_GROUP != mg.ID_GROUP
				AND FIND_IN_SET(mg.ID_GROUP, mem.additionalGroups)
			GROUP BY mg.ID_GROUP`))
		if err == nil {
			for r2.Next() {
				var id, n int
				r2.Scan(&id, &n)
				add(id, n)
			}
			r2.Close()
		}
	}

	// Any moderators?
	var numMods int
	a.DB.QueryRow(a.Q(`SELECT COUNT(DISTINCT ID_MEMBER) FROM {$db_prefix}moderators`)).Scan(&numMods)
	add(3, numMods)

	page.CanSendPM = c.allowedTo("pm_send")

	c.SubTemplate = templateEmailMembers
}

// ComposeMailingCtx backs template_email_members_compose.
type ComposeMailingCtx struct {
	Addresses      string
	DefaultSubject string
	DefaultMessage string
}

// ComposeMailing is ComposeMailing(): resolve the recipient list, then either
// hand off to the PM composer (sendPM) or show the email compose form.
func (c *Ctx) ComposeMailing() {
	a := c.App

	var list []string
	seen := map[string]bool{}
	addOne := func(s string) {
		if !seen[s] {
			seen[s] = true
			list = append(list, s)
		}
	}
	doPM := c.POST.Has("sendPM") && !empty(c.POST.Str("sendPM"))

	// Opt-out?
	condition := ""
	if !c.POST.Has("email_force") {
		condition = "\n\t\t\t\tAND mem.notifyAnnouncements = 1"
	}

	now := itoa64(c.forumTime(false, 0))

	// Banned members (by ID) who can't login to turn off notification.
	var bannedIDs []int
	if rows, err := a.DB.Query(a.Q(`
		SELECT DISTINCT mem.ID_MEMBER
		FROM {$db_prefix}ban_groups AS bg
		INNER JOIN {$db_prefix}ban_items AS bi ON (bg.ID_BAN_GROUP = bi.ID_BAN_GROUP)
		INNER JOIN {$db_prefix}members AS mem ON (bi.ID_MEMBER = mem.ID_MEMBER)
		WHERE (bg.cannot_access = 1 OR bg.cannot_login = 1)
			AND (bg.expire_time IS NULL OR bg.expire_time > ` + now + `)`)); err == nil {
		for rows.Next() {
			var id int
			rows.Scan(&id)
			bannedIDs = append(bannedIDs, id)
		}
		rows.Close()
	}
	if len(bannedIDs) > 0 {
		condition += "\n\t\t\t\tAND mem.ID_MEMBER NOT IN (" + joinInts(bannedIDs) + ")"
	}

	// Banned email addresses.
	if rows, err := a.DB.Query(a.Q(`
		SELECT DISTINCT bi.email_address
		FROM {$db_prefix}ban_items AS bi
		INNER JOIN {$db_prefix}ban_groups AS bg ON (bg.ID_BAN_GROUP = bi.ID_BAN_GROUP)
		WHERE (bg.cannot_access = 1 OR bg.cannot_login = 1)
			AND (bg.expire_time IS NULL OR bg.expire_time > ` + now + `)
			AND bi.email_address != ''`)); err == nil {
		for rows.Next() {
			var email string
			rows.Scan(&email)
			condition += "\n\t\t\t\tAND mem.emailAddress NOT LIKE '" + email + "'"
		}
		rows.Close()
	}

	identifier := "mem.emailAddress"
	if doPM {
		identifier = "mem.memberName"
	}

	// who[] selections.
	who := map[int]bool{}
	var whoOrder []int
	if arr := c.POST.Arr("who"); arr != nil {
		for _, k := range arr.Keys() {
			id := atoi(arr.Str(k))
			if !who[id] {
				who[id] = true
				whoOrder = append(whoOrder, id)
			}
		}
	}

	collect := func(q string) {
		if rows, err := a.DB.Query(a.Q(q)); err == nil {
			for rows.Next() {
				var s string
				rows.Scan(&s)
				addOne(s)
			}
			rows.Close()
		}
	}

	// Did they select moderators too?
	if who[3] {
		collect(`
			SELECT DISTINCT ` + identifier + ` AS identifier
			FROM {$db_prefix}members AS mem, {$db_prefix}moderators AS mods
			WHERE mem.ID_MEMBER = mods.ID_MEMBER
				AND mem.is_activated = 1` + condition)
		delete(who, 3)
	}

	// How about regular members? (ID_GROUP = 0)
	if who[0] {
		collect(`
			SELECT ` + identifier + ` AS identifier
			FROM {$db_prefix}members AS mem
			WHERE mem.ID_GROUP = 0
				AND mem.is_activated = 1` + condition)
		delete(who, 0)
	}

	// Load all the other groups.
	var groupIDs []int
	for _, id := range whoOrder {
		if who[id] {
			groupIDs = append(groupIDs, id)
		}
	}
	if len(groupIDs) > 0 {
		collect(`
			SELECT ` + identifier + ` AS identifier
			FROM {$db_prefix}members AS mem, {$db_prefix}membergroups AS mg
			WHERE (mg.ID_GROUP = mem.ID_GROUP OR FIND_IN_SET(mg.ID_GROUP, mem.additionalGroups) OR mg.ID_GROUP = mem.ID_POST_GROUP)
				AND mg.ID_GROUP IN (` + joinInts(groupIDs) + `)
				AND mem.is_activated = 1` + condition)
	}

	// Sending as a personal message? PHP calls MessagePost() directly; this
	// port routes through MessageMain (sa=send) with the bcc prefilled — the
	// only difference is the PM sidebar shows (PHP's direct call omits it).
	if doPM {
		c.REQUEST.Set("bcc", strings.Join(list, ","))
		c.REQUEST.Set("sa", "send")
		c.MessageMain()
		return
	}

	c.PageTitle = c.Txt("6")
	page := &ComposeMailingCtx{
		Addresses:      strings.Join(list, "; "),
		DefaultSubject: a.Config.MbName + ": " + c.Txt("70"),
		DefaultMessage: c.Txt("72") + "\n\n" + c.Txt("130") + "\n\n{$board_url}",
	}
	c.Page = page
	c.SubTemplate = templateEmailMembersCompose
}

// SendMailingCtx backs template_email_members_send.
type SendMailingCtx struct {
	Emails         string
	Subject        string
	Message        string
	SendHTML       string
	ParseHTML      string
	Start          int
	PercentageDone float64
}

// SendMailing is SendMailing(): the batched mass-email send (60 at a time).
func (c *Ctx) SendMailing() {
	a := c.App
	scripturl := a.ScriptURL

	c.checkSession("post", "", true)

	const numAtOnce = 60

	// Get all the receivers (de-duplicated, trimmed, non-empty).
	var cleanOrder []string
	cleanSeen := map[string]bool{}
	for _, raw := range strings.Split(c.POST.Str("emails"), ";") {
		m := strings.TrimSpace(raw)
		if m != "" && !cleanSeen[m] {
			cleanSeen[m] = true
			cleanOrder = append(cleanOrder, m)
		}
	}

	page := &SendMailingCtx{
		Emails:    strings.Join(cleanOrder, ";"),
		Subject:   Htmlspecialchars(c.POST.Str("subject")),
		Message:   Htmlspecialchars(c.POST.Str("message")),
		SendHTML:  "0",
		ParseHTML: "0",
		Start:     c.REQUEST.Int("start"),
	}
	c.Page = page
	sendHTML := c.POST.Has("send_html") && !empty(c.POST.Str("send_html"))
	parseHTML := c.POST.Has("parse_html") && !empty(c.POST.Str("parse_html"))
	if sendHTML {
		page.SendHTML = "1"
	}
	if parseHTML {
		page.ParseHTML = "1"
	}

	// This batch's recipients.
	var sendList []string
	sendSet := map[string]bool{}
	i := 0
	for _, email := range cleanOrder {
		i++
		if i <= page.Start {
			continue
		}
		if i > page.Start+numAtOnce {
			break
		}
		if !sendSet[email] {
			sendSet[email] = true
			sendList = append(sendList, email)
		}
	}

	page.Start += numAtOnce
	if len(cleanOrder) > 0 {
		// PHP round(x, 2).
		pd := float64(page.Start) * 100 / float64(len(cleanOrder))
		page.PercentageDone = math.Round(pd*100) / 100
	}

	// Prepare the message for HTML.
	message := c.POST.Str("message")
	if sendHTML && parseHTML {
		message = strings.NewReplacer("\n", "<br />\n", "  ", "&nbsp; ").Replace(message)
	}

	// Replace in all the standard things.
	curTime := c.timeformatNoToday(c.forumTime(false, 0))
	latestMember := a.Setting("latestMember")
	latestName := a.Setting("latestRealName")
	var boardURLrepl, latestLink string
	if sendHTML {
		boardURLrepl = `<a href="` + scripturl + `">` + scripturl + `</a>`
		latestLink = `<a href="` + scripturl + `?action=profile;u=` + latestMember + `">` + latestName + `</a>`
	} else {
		boardURLrepl = scripturl
		latestLink = latestName
	}
	message = strings.NewReplacer(
		"{$board_url}", boardURLrepl,
		"{$current_time}", curTime,
		"{$latest_member.link}", latestLink,
		"{$latest_member.id}", latestMember,
		"{$latest_member.name}", latestName,
	).Replace(message)
	subject := strings.NewReplacer(
		"{$board_url}", scripturl,
		"{$current_time}", curTime,
		"{$latest_member.link}", latestName,
		"{$latest_member.id}", latestMember,
		"{$latest_member.name}", latestName,
	).Replace(c.POST.Str("subject"))

	// Prevent spam filters from tagging this as spam.
	if sendHTML && !reHTMLTag.MatchString(message) {
		if !reBodyTag.MatchString(message) {
			message = "<html><head><title>" + subject + "</title></head>\n<body>" + message + "</body></html>"
		} else {
			message = "<html>" + message + "</html>"
		}
	}

	memberVars := func(email, link, id, name string) *strings.Replacer {
		return strings.NewReplacer(
			"{$member.email}", email,
			"{$member.link}", link,
			"{$member.id}", id,
			"{$member.name}", name,
		)
	}

	// Members first (personalised), removing them from the leftover list.
	if len(sendList) > 0 {
		placeholders := make([]string, len(sendList))
		args := make([]any, len(sendList))
		for i, e := range sendList {
			placeholders[i] = "?"
			args[i] = e
		}
		if rows, err := a.DB.Query(a.Q(`
			SELECT realName, memberName, ID_MEMBER, emailAddress
			FROM {$db_prefix}members
			WHERE emailAddress IN (`+strings.Join(placeholders, ", ")+`)
				AND is_activated = 1`), args...); err == nil {
			for rows.Next() {
				var realName, memberName, email string
				var id int
				rows.Scan(&realName, &memberName, &id, &email)
				delete(sendSet, email)

				link := scripturl + "?action=profile;u=" + itoa(id)
				if sendHTML {
					link = `<a href="` + scripturl + `?action=profile;u=` + itoa(id) + `">` + realName + `</a>`
				}
				mv := memberVars(email, link, itoa(id), realName)
				c.sendmail([]string{email}, mv.Replace(subject), mv.Replace(message), "")
			}
			rows.Close()
		}
	}

	// Send to people who weren't members.
	for _, email := range sendList {
		if !sendSet[email] {
			continue
		}
		link := email
		if sendHTML {
			link = `<a href="mailto:` + email + `">` + email + `</a>`
		}
		mv := memberVars(email, link, "??", email)
		c.sendmail([]string{email}, mv.Replace(subject), mv.Replace(message), "")
	}

	// Still more to do?
	if len(cleanOrder) > page.Start {
		c.PageTitle = c.Txt("6")
		c.SubTemplate = templateEmailMembersSend
		return
	}

	c.redirectExit("action=admin")
}

var (
	reHTMLTag = regexp.MustCompile(`(?i)<html`)
	reBodyTag = regexp.MustCompile(`(?i)<body`)
)
