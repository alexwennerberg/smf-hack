package app

// Port of Sources/Admin.php (the admin home) and adminIndex() from Subs.php
// (the admin sidebar/menu + the 'admin' template layer). MySQL-specific tools
// (OptimizeTables, ConvertUtf8/Entities) and the package manager are dropped
// per the plan; the copyright check (an external simplemachines.org request)
// is also dropped. Menu links are emitted byte-for-byte as PHP does even when
// the target action isn't ported yet (packages, detailedversion).

import "runtime"

func init() {
	registerAction("admin", (*Ctx).AdminMain)
}

// AdminArea is one link in an admin sidebar section.
type AdminArea struct {
	Key  string
	Link string
}

// AdminSection is one titled group in the admin sidebar.
type AdminSection struct {
	ID    string
	Title string
	Areas []AdminArea
}

// AdminTab is one tab in an admin page's tab bar.
type AdminTab struct {
	Href        string
	Title       string
	Description string
	IsSelected  bool
	IsLast      bool
}

// AdminTabs is the per-page tab bar ($context['admin_tabs']).
type AdminTabs struct {
	Title       string
	Help        string
	Description string
	Tabs        []AdminTab
}

// AdminVersion is one entry of $context['current_versions'].
type AdminVersion struct {
	Key     string
	Title   string
	Version string
}

// AdminMainCtx backs template_admin and template_credits.
type AdminMainCtx struct {
	Administrators  []string
	MoreAdminsLink  string
	Credits         string
	CanAdmin        bool
	ForumVersion    string
	TimeFormat      string
	CurrentVersions []AdminVersion
	CopyrightExpire int
	CopyrightKey    string
	QuickTasks      []AdminQuickTask
}

// AdminQuickTask is one entry of $context['quick_admin_tasks'].
type AdminQuickTask struct {
	Href        string
	Link        string
	Title       string
	Description string
	IsLast      bool
}

// adminIndex is adminIndex($area): build the admin sidebar, validate the
// admin session, and push the 'admin' template layer.
func (c *Ctx) adminIndex(area string) {
	a := c.App
	scripturl := a.ScriptURL
	sc := c.Sc

	c.loadLanguage("Admin")

	link := func(label, target string) string {
		return `<a href="` + scripturl + target + `">` + label + `</a>`
	}

	// Admin area 'Main'.
	forum := AdminSection{ID: "forum", Title: c.Txt("427"), Areas: []AdminArea{
		{"index", link(c.Txt("208"), "?action=admin")},
		{"credits", link(c.Txt("support_credits_title"), "?action=admin;credits")},
	}}
	if c.allowedTo("edit_news", "send_mail", "admin_forum") {
		forum.Areas = append(forum.Areas, AdminArea{"news", link(c.Txt("news_title"), "?action=news")})
	}
	if c.allowedTo("admin_forum") {
		forum.Areas = append(forum.Areas, AdminArea{"manage_packages", link(c.Txt("package1"), "?action=packages")})
	}
	c.AdminAreas = []AdminSection{forum}

	// Admin area 'Configuration'.
	if c.allowedTo("admin_forum") {
		c.AdminAreas = append(c.AdminAreas, AdminSection{ID: "config", Title: c.Txt("428"), Areas: []AdminArea{
			{"edit_mods_settings", link(c.Txt("modSettings_title"), "?action=featuresettings")},
			{"edit_settings", link(c.Txt("222"), "?action=serversettings;sesc="+sc)},
			{"edit_theme_settings", link(c.Txt("theme_current_settings"), "?action=theme;sa=settings;th="+itoa(c.Theme.Int("theme_id"))+";sesc="+sc)},
			{"manage_themes", link(c.Txt("theme_admin"), "?action=theme;sa=admin;sesc="+sc)},
		}})
	}

	// Admin area 'Forum' (layout controls).
	if c.allowedTo("manage_boards", "admin_forum", "manage_smileys", "manage_attachments", "moderate_forum") {
		layout := AdminSection{ID: "layout", Title: c.Txt("layout_controls")}
		if c.allowedTo("manage_boards") {
			layout.Areas = append(layout.Areas, AdminArea{"manage_boards", link(c.Txt("4"), "?action=manageboards")})
		}
		if c.allowedTo("admin_forum", "moderate_forum") {
			layout.Areas = append(layout.Areas, AdminArea{"posts_and_topics", link(c.Txt("manageposts"), "?action=postsettings")})
		}
		if c.allowedTo("admin_forum") {
			layout.Areas = append(layout.Areas,
				AdminArea{"manage_search", link(c.Txt("manage_search"), "?action=managesearch")})
		}
		if c.allowedTo("manage_smileys") {
			layout.Areas = append(layout.Areas, AdminArea{"manage_smileys", link(c.Txt("smileys_manage"), "?action=smileys")})
		}
		if c.allowedTo("manage_attachments") {
			layout.Areas = append(layout.Areas, AdminArea{"manage_attachments", link(c.Txt("smf201"), "?action=manageattachments")})
		}
		c.AdminAreas = append(c.AdminAreas, layout)
	}

	// Admin area 'Members'.
	if c.allowedTo("moderate_forum", "manage_membergroups", "manage_bans", "manage_permissions", "admin_forum") {
		members := AdminSection{ID: "members", Title: c.Txt("426")}
		if c.allowedTo("moderate_forum") {
			members.Areas = append(members.Areas, AdminArea{"view_members", link(c.Txt("5"), "?action=viewmembers")})
		}
		if c.allowedTo("manage_membergroups") {
			members.Areas = append(members.Areas, AdminArea{"edit_groups", link(c.Txt("8"), "?action=membergroups;")})
		}
		if c.allowedTo("manage_permissions") {
			members.Areas = append(members.Areas, AdminArea{"edit_permissions", link(c.Txt("edit_permissions"), "?action=permissions")})
		}
		if c.allowedTo("admin_forum", "moderate_forum") {
			members.Areas = append(members.Areas, AdminArea{"registration_center", link(c.Txt("registration_center"), "?action=regcenter")})
		}
		if c.allowedTo("manage_bans") {
			members.Areas = append(members.Areas, AdminArea{"ban_members", link(c.Txt("ban_title"), "?action=ban")})
		}
		c.AdminAreas = append(c.AdminAreas, members)
	}

	// Admin area 'Maintenance Controls'.
	if c.allowedTo("admin_forum") {
		maint := AdminSection{ID: "maintenance", Title: c.Txt("501"), Areas: []AdminArea{
			{"maintain_forum", link(c.Txt("maintain_title"), "?action=maintain")},
			{"generate_reports", link(c.Txt("generate_reports"), "?action=reports")},
			{"view_errors", link(c.Txt("errlog1"), "?action=viewErrorLog;desc")},
		}}
		if !a.SettingEmpty("modlog_enabled") {
			maint.Areas = append(maint.Areas, AdminArea{"view_moderation_log", link(c.Txt("modlog_view"), "?action=modlog")})
		}
		c.AdminAreas = append(c.AdminAreas, maint)
	}

	// Make sure the administrator has a valid session.
	c.validateSession()

	// Figure out which section we're in now.
	for _, section := range c.AdminAreas {
		for _, ar := range section.Areas {
			if ar.Key == area {
				c.AdminSection = section.ID
			}
		}
	}
	c.AdminArea = area

	// obExit will know what to do!
	c.TemplateLayers = append(c.TemplateLayers, "admin")
}

// AdminMain is Admin(): the administration center home.
func (c *Ctx) AdminMain() {
	a := c.App
	scripturl := a.ScriptURL

	// You have to be able to do at least one of these to see this page.
	c.isAllowedTo("admin_forum", "manage_permissions", "moderate_forum", "manage_membergroups",
		"manage_bans", "send_mail", "edit_news", "manage_boards", "manage_smileys", "manage_attachments")

	credits := c.GET.Has("credits")
	if credits {
		c.adminIndex("credits")
	} else {
		c.adminIndex("index")
	}

	page := &AdminMainCtx{ForumVersion: forumVersion}
	c.Page = page

	// Find all of this forum's administrators.
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_MEMBER, realName
		FROM {$db_prefix}members
		WHERE ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups)
		LIMIT 33`))
	if err == nil {
		for rows.Next() {
			var id int
			var name string
			rows.Scan(&id, &name)
			page.Administrators = append(page.Administrators, `<a href="`+scripturl+`?action=profile;u=`+itoa(id)+`">`+name+`</a>`)
		}
		rows.Close()
	}
	if len(page.Administrators) > 32 {
		page.Administrators = page.Administrators[:32]
		page.MoreAdminsLink = `<a href="` + scripturl + `?action=mlist;sa=search;fields=group;search=administrator">` + c.Txt("more") + `</a>`
	}

	page.Credits = adminCreditsText

	// This makes it easier to get the latest news with your time format.
	page.TimeFormat = urlencode(c.User.TimeFormat)

	// Adapted version info (Go runtime + SQLite; PHP/MySQL/accelerator probing
	// has no equivalent — documented deviation).
	var sqliteVer string
	a.DB.QueryRow(`SELECT sqlite_version()`).Scan(&sqliteVer)
	page.CurrentVersions = []AdminVersion{
		{"php", c.Txt("support_versions_php"), runtime.Version()},
		{"db", c.Txt("support_versions_mysql"), "SQLite " + sqliteVer},
	}

	page.CanAdmin = c.allowedTo("admin_forum")

	// The format of this array is: permission, action, title, description.
	quickTasks := []struct{ perm, action, title, desc string }{
		{"", "admin;credits", "support_credits_title", "support_credits_info"},
		{"admin_forum", "featuresettings", "modSettings_title", "modSettings_info"},
		{"admin_forum", "maintain", "maintain_title", "maintain_info"},
		{"manage_permissions", "permissions", "edit_permissions", "edit_permissions_info"},
		{"admin_forum", "theme;sa=admin;sesc=" + c.Sc, "theme_admin", "theme_admin_info"},
		{"admin_forum", "packages", "package1", "package_info"},
		{"manage_smileys", "smileys", "smileys_manage", "smileys_manage_info"},
		{"moderate_forum", "viewmembers", "5", "member_center_info"},
	}
	for _, task := range quickTasks {
		if task.perm != "" && !c.allowedTo(task.perm) {
			continue
		}
		page.QuickTasks = append(page.QuickTasks, AdminQuickTask{
			Href:        scripturl + "?action=" + task.action,
			Link:        `<a href="` + scripturl + `?action=` + task.action + `">` + c.Txt(task.title) + `</a>`,
			Title:       c.Txt(task.title),
			Description: c.Txt(task.desc),
		})
	}
	n := len(page.QuickTasks)
	if n%2 == 1 {
		page.QuickTasks = append(page.QuickTasks, AdminQuickTask{IsLast: true})
		page.QuickTasks[n-1].IsLast = true
	} else if n != 0 {
		page.QuickTasks[n-1].IsLast = true
		page.QuickTasks[n-2].IsLast = true
	}

	if credits {
		c.SubTemplate = templateCredits
		c.PageTitle = c.Txt("support_credits_title")
	} else {
		c.SubTemplate = templateAdmin
		c.PageTitle = c.Txt("208")
	}
}

// adminCreditsText is $context['credits'] (the static credits blurb).
const adminCreditsText = `
<i>Simple Machines wants to thank everyone who helped make SMF 1.1 what it is today; shaping and directing our project, all through the thick and the thin. It wouldn't have been possible without you.</i><br />
<div style="margin-top: 1ex;"><i>This includes our users and especially Charter Members - thanks for installing and using our software as well as providing valuable feedback, bug reports, and opinions.</i></div>
<div style="margin-top: 2ex;"><b>Project Managers:</b> Amacythe, David Recordon, Joseph Fung, and Jeff Lewis.</div>
<div style="margin-top: 1ex;"><b>Developers:</b> Hendrik Jan &quot;Compuart&quot; Visser, Matt &quot;Grudge&quot; Wolf, Michael &quot;Thantos&quot; Miller, Theodore &quot;Orstio&quot; Hildebrandt, and Unknown W. &quot;[Unknown]&quot; Brackets</div>
<div style="margin-top: 1ex;"><b>Support Specialists:</b> Ben Scott, Michael &quot;Oldiesmann&quot; Eshom, Jan-Olof &quot;Owdy&quot; Eriksson, A&auml;ron van Geffen, Alexandre &quot;Ap2&quot; Patenaude, Andrea Hubacher, Chris Cromer, [darksteel], dtm.exe, Nick &quot;Fizzy&quot; Dyer, Horseman, Huw Ayling-Miller, jerm, Justyne, kegobeer, Kindred, Matthew &quot;Mattitude&quot; Hall, Mediman, Metho, Omar Bazavilvazo, Pitti, redone, Tomer &quot;Lamper&quot; Dean, Tony, and xenovanis.</div>
<div style="margin-top: 1ex;"><b>Mod Developers:</b> snork13, Cristi&aacute;n &quot;Anguz&quot; L&aacute;vaque, Goosemoose, Jack.R.Abbit, James &quot;Cheschire&quot; Yarbro, Jesse &quot;Gobalopper&quot; Reid, Juan &quot;JayBachatero&quot; Hernandez, Kirby, vbgamer45, and winrules.</div>
<div style="margin-top: 1ex;"><b>Documentation Writers:</b> akabugeyes, eldacar, Gary M. &quot;AwwLilMaggie&quot; Gadsdon, Jerry, and Nave.</div>
<div style="margin-top: 1ex;"><b>Language Coordinators:</b> Daniel Diehl and Adam &quot;Bostasp&quot; Southall.</div>
<div style="margin-top: 1ex;"><b>Graphic Designers:</b> Bjoern &quot;Bloc&quot; Kristiansen, Alienine (Adrian), A.M.A, babylonking, BlackouT, Burpee, diplomat, Eren &quot;forsakenlad&quot; Yasarkurt, Hyper Piranha, Killer Possum, Mystica, Nico &quot;aliencowfarm&quot; Boer, Philip &quot;Meriadoc&quot; Renich and Tippmaster.</div>
<div style="margin-top: 1ex;"><b>Site team:</b> dschwab9 and Tim.</div>
<div style="margin-top: 1ex;"><b>Marketing:</b> Douglas &quot;The Bear&quot; Hazard, RickC and Trekkie101.</div>
<div style="margin-top: 1ex;">And for anyone we may have missed, thank you!</div>`
