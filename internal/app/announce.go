package app

// AnnounceTopic / AnnouncementSelectMembergroup / AnnouncementSend from
// Sources/Post.php, plus template_announce and template_announcement_send
// from Post.template.php.

import (
	"math"
	"strconv"
	"strings"
)

func init() {
	registerAction("announce", (*Ctx).AnnounceTopic)
}

// AnnounceGroup is one selectable membergroup.
type AnnounceGroup struct {
	ID          int
	Name        string
	MemberCount string
}

// AnnounceCtx is the page context for the announce templates.
type AnnounceCtx struct {
	Groups         []AnnounceGroup
	TopicSubject   string
	Move           int
	GoBack         int
	Start          int
	PercentageDone float64
	Membergroups   string
}

// AnnounceTopic is AnnounceTopic().
func (c *Ctx) AnnounceTopic() {
	c.isAllowedTo("announce_topic")

	c.validateSession()

	c.loadLanguage("Post")

	c.PageTitle = c.Txt("announce_topic")

	// Call the function based on the sub-action.
	switch c.REQUEST.Str("sa") {
	case "send":
		c.AnnouncementSend()
	default:
		c.AnnouncementSelectMembergroup()
	}
}

// announceBoardGroups returns $board_info['groups'] + array(1) as ints.
func (c *Ctx) announceBoardGroups() []int {
	groups := []int{}
	if c.BoardInfo != nil {
		groups = append(groups, c.BoardInfo.Groups...)
	}
	groups = append(groups, 1)
	return groups
}

// AnnouncementSelectMembergroup is AnnouncementSelectMembergroup(): allow a
// user to chose the membergroups to send the announcement to.
func (c *Ctx) AnnouncementSelectMembergroup() {
	a := c.App

	groups := c.announceBoardGroups()

	page := &AnnounceCtx{}
	c.Page = page

	hasRegular := false
	for _, g := range groups {
		if g == 0 {
			hasRegular = true
		}
	}
	if hasRegular {
		page.Groups = append(page.Groups, AnnounceGroup{
			ID:          0,
			Name:        c.Txt("announce_regular_members"),
			MemberCount: "n/a",
		})
	}

	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = itoa(g)
	}

	// Get all membergroups that have access to the board the announcement
	// was made on.
	rows, err := a.DB.Query(a.Q(`
		SELECT mg.ID_GROUP, mg.groupName, COUNT(mem.ID_MEMBER) AS num_members
		FROM {$db_prefix}membergroups AS mg
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_GROUP = mg.ID_GROUP OR FIND_IN_SET(mg.ID_GROUP, mem.additionalGroups) OR mg.ID_GROUP = mem.ID_POST_GROUP)
		WHERE mg.ID_GROUP IN (` + strings.Join(ids, ", ") + `)
		GROUP BY mg.ID_GROUP
		ORDER BY mg.minPosts, IIF(mg.ID_GROUP < 4, mg.ID_GROUP, 4), mg.groupName`))
	if err == nil {
		for rows.Next() {
			var id, num int
			var name string
			rows.Scan(&id, &name, &num)
			page.Groups = append(page.Groups, AnnounceGroup{
				ID:          id,
				Name:        name,
				MemberCount: itoa(num),
			})
		}
		rows.Close()
	}

	// Get the subject of the topic we're about to announce.
	a.DB.QueryRow(a.Q(`
		SELECT m.subject
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE t.ID_TOPIC = ?
			AND m.ID_MSG = t.ID_FIRST_MSG`), c.Topic).Scan(&page.TopicSubject)

	if c.REQUEST.Has("move") {
		page.Move = 1
	}
	if c.REQUEST.Has("goback") {
		page.GoBack = 1
	}

	c.SubTemplate = templateAnnounce
}

// AnnouncementSend is AnnouncementSend(): send the announcement in chunks.
func (c *Ctx) AnnouncementSend() {
	a := c.App
	scripturl := a.ScriptURL

	c.checkSession("post", "", true)

	chunkSize := 50
	page := &AnnounceCtx{}
	c.Page = page
	page.Start = c.REQUEST.Int("start")
	groups := c.announceBoardGroups()

	var who []int
	if !empty(c.POST.Str("membergroups")) {
		for _, mg := range strings.Split(c.POST.Str("membergroups"), ",") {
			who = append(who, atoi(mg))
		}
	} else if whoArr := c.POST.Arr("who"); whoArr != nil {
		for _, k := range whoArr.Keys() {
			who = append(who, atoi(whoArr.Str(k)))
		}
	}

	// Check whether at least one membergroup was selected.
	if len(who) == 0 {
		c.fatalLangError("no_membergroup_selected", true)
	}

	// Make sure all membergroups are integers and can access the board of
	// the announcement.
	inGroups := func(g int) bool {
		for _, x := range groups {
			if x == g {
				return true
			}
		}
		return false
	}
	for i, mg := range who {
		if !inGroups(mg) {
			who[i] = 0
		}
	}

	// Get the topic subject and censor it.
	var idMsg int
	var message string
	a.DB.QueryRow(a.Q(`
		SELECT m.ID_MSG, m.subject, m.body
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE t.ID_TOPIC = ?
			AND m.ID_MSG = t.ID_FIRST_MSG`), c.Topic).Scan(&idMsg, &page.TopicSubject, &message)

	page.TopicSubject = c.censorText(page.TopicSubject)
	message = c.censorText(message)

	message = strings.TrimSpace(unHtmlspecialchars(stripTags(strings.NewReplacer(
		"<br />", "\n", "</div>", "\n", "</li>", "\n", "&#91;", "[", "&#93;", "]",
	).Replace(c.parseBBCCached(message, false, itoa(idMsg))))))

	whoIDs := make([]string, len(who))
	var finds []string
	for i, g := range who {
		whoIDs[i] = itoa(g)
		finds = append(finds, "FIND_IN_SET("+itoa(g)+", mem.additionalGroups)")
	}

	notifyCond := ""
	if !a.SettingEmpty("allow_disableAnnounce") {
		notifyCond = `
			AND mem.notifyAnnouncements = 1`
	}

	// Select the email addresses for this batch.
	type recipient struct {
		id      int
		email   string
		lngfile string
	}
	var recipients []recipient
	rows, err := a.DB.Query(a.Q(`
		SELECT mem.ID_MEMBER, mem.emailAddress, mem.lngfile
		FROM {$db_prefix}members AS mem
		WHERE mem.ID_MEMBER != ` + itoa(c.User.ID) + notifyCond + `
			AND mem.is_activated = 1
			AND (mem.ID_GROUP IN (` + strings.Join(whoIDs, ", ") + `) OR mem.ID_POST_GROUP IN (` + strings.Join(whoIDs, ", ") + `) OR ` + strings.Join(finds, " OR ") + `)
			AND mem.ID_MEMBER > ` + itoa(page.Start) + `
		ORDER BY mem.ID_MEMBER
		LIMIT ` + itoa(chunkSize)))
	if err == nil {
		for rows.Next() {
			var r recipient
			rows.Scan(&r.id, &r.email, &r.lngfile)
			recipients = append(recipients, r)
		}
		rows.Close()
	}

	// All members have received a mail. Go to the next screen.
	if len(recipients) == 0 {
		if !empty(c.REQUEST.Str("move")) && c.allowedTo("move_any") {
			goback := ""
			if !empty(c.REQUEST.Str("goback")) {
				goback = ";goback"
			}
			c.redirectExit("action=movetopic;topic=" + itoa(c.Topic) + ".0" + goback)
		} else if !empty(c.REQUEST.Str("goback")) {
			c.redirectExit("topic=" + itoa(c.Topic) + ".new;boardseen#new")
		} else {
			c.redirectExit("board=" + itoa(c.Board) + ".0")
		}
	}

	// Loop through all members that'll receive an announcement in this
	// batch. (Single-language port: one batch mail.)
	subject := c.Txt("notifyXAnn2") + ": " + page.TopicSubject
	body := message + "\n\n" + c.Txt("notifyXAnn3") + "\n\n" + scripturl + "?topic=" + itoa(c.Topic) + ".0\n\n" + c.Txt("130")
	var emails []string
	for _, r := range recipients {
		emails = append(emails, r.email)
		page.Start = r.id
	}

	c.sendmail(emails, subject, body, "")

	if a.SettingInt("latestMember") > 0 {
		// PHP round(..., 1).
		page.PercentageDone = math.Round(1000*float64(page.Start)/float64(a.SettingInt("latestMember"))) / 10
	}

	if !empty(c.REQUEST.Str("move")) {
		page.Move = 1
	}
	if !empty(c.REQUEST.Str("goback")) {
		page.GoBack = 1
	}
	page.Membergroups = strings.Join(whoIDs, ",")
	c.SubTemplate = templateAnnouncementSend
}

// templateAnnounce is template_announce() from Post.template.php.
func templateAnnounce(c *Ctx) {
	page := c.Page.(*AnnounceCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`, scripturl, `?action=announce;sa=send" method="post" accept-charset="`, c.CharacterSet, `">
			<table width="600" cellpadding="5" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>`, c.Txt("announce_title"), `</td>
				</tr><tr class="windowbg">
					<td class="smalltext" style="padding: 2ex;">`, c.Txt("announce_desc"), `</td>
				</tr><tr>
					<td class="windowbg2">
						`, c.Txt("announce_this_topic"), ` <a href="`, scripturl, `?topic=`, c.Topic, `.0">`, page.TopicSubject, `</a><br />
					</td>
				</tr><tr>
					<td class="windowbg2">`)

	for _, group := range page.Groups {
		c.O(`
						<label for="who_`, group.ID, `"><input type="checkbox" name="who[`, group.ID, `]" id="who_`, group.ID, `" value="`, group.ID, `" checked="checked" class="check" /> `, group.Name, `</label> <i>(`, group.MemberCount, `)</i><br />`)
	}

	c.O(`
						<br />
						<label for="checkall"><input type="checkbox" id="checkall" class="check" onclick="invertAll(this, this.form);" checked="checked" /> <i>`, c.Txt("737"), `</i></label>
					</td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 1ex;" align="center">
						<input type="submit" value="`, c.Txt("105"), `" />
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
			<input type="hidden" name="topic" value="`, c.Topic, `" />
			<input type="hidden" name="move" value="`, page.Move, `" />
			<input type="hidden" name="goback" value="`, page.GoBack, `" />
		</form>`)
}

// templateAnnouncementSend is template_announcement_send() from
// Post.template.php.
func templateAnnouncementSend(c *Ctx) {
	page := c.Page.(*AnnounceCtx)
	scripturl := c.App.ScriptURL

	c.O(`
		<form action="`+scripturl+`?action=announce;sa=send" method="post" accept-charset="`, c.CharacterSet, `" name="autoSubmit" id="autoSubmit">
			<table width="600" cellpadding="5" cellspacing="0" border="0" align="center" class="tborder">
				<tr class="titlebg">
					<td>
						`, c.Txt("announce_sending"), ` <a href="`, scripturl, `?topic=`, c.Topic, `.0" target="_blank">`, page.TopicSubject, `</a>
					</td>
				</tr><tr>
					<td class="windowbg2"><b>`, formatPercent(page.PercentageDone), `% `, c.Txt("announce_done"), `</b></td>
				</tr><tr>
					<td class="windowbg2" style="padding-bottom: 1ex;" align="center">
						<input type="submit" name="b" value="`, c.Txt("announce_continue"), `" />
					</td>
				</tr>
			</table>
			<input type="hidden" name="sc" value="`, c.Sc, `" />
			<input type="hidden" name="topic" value="`, c.Topic, `" />
			<input type="hidden" name="move" value="`, page.Move, `" />
			<input type="hidden" name="goback" value="`, page.GoBack, `" />
			<input type="hidden" name="start" value="`, page.Start, `" />
			<input type="hidden" name="membergroups" value="`, page.Membergroups, `" />
		</form>
		<script language="JavaScript" type="text/javascript"><!-- // --><![CDATA[
			var countdown = 2;
			doAutoSubmit();

			function doAutoSubmit()
			{
				if (countdown == 0)
					document.forms.autoSubmit.submit();
				else if (countdown == -1)
					return;

				document.forms.autoSubmit.b.value = "`, c.Txt("announce_continue"), ` (" + countdown + ")";
				countdown--;

				setTimeout("doAutoSubmit();", 1000);
			}
		// ]]></script>`)
}

// formatPercent prints a float like PHP echo does (no trailing ".0").
func formatPercent(f float64) string {
	if f == math.Trunc(f) {
		return itoa(int(f))
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatPercentMySQL mirrors a MySQL DECIMAL value produced by `a/b*100` with
// the default div_precision_increment (4): a fixed 4-decimal scale with trailing
// zeros kept. SMF's profile board-activity percent comes straight from such an
// expression, so PHP renders e.g. "96.5517" / "50.0000" where a raw float64
// would print full precision.
func formatPercentMySQL(f float64) string {
	return strconv.FormatFloat(f, 'f', 4, 64)
}
