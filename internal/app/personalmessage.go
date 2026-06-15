package app

// Port of Sources/PersonalMessage.php: MessageMain, messageIndexBar,
// MessageFolder, prepareMessageContext, MessagePost, messagePostError,
// MessagePost2, MessageActionsApply, MessageKillAllQuery, MessageKillAll,
// MessagePrune, ManageLabels, ReportMessage. (MessageSearch/MessageSearch2
// port with Phase 5's search work; deleteMessages/markMessages live in
// subs_members2.go / subs_pm.go.)

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"
)

func init() {
	registerAction("pm", (*Ctx).MessageMain)
	registerAction("im", (*Ctx).MessageMain) // index.php legacy alias
}

// PMLabel is one message label.
type PMLabel struct {
	ID             int
	Name           string
	Messages       int
	UnreadMessages int
}

// PMArea is one entry in the PM sidebar.
type PMArea struct {
	Key            string
	Link           string
	Href           string
	UnreadMessages *int
	Messages       *int
	IsLimitBar     bool
	IsSpacer       bool
}

// PMAreaGroup is one sidebar group.
type PMAreaGroup struct {
	Title string
	Areas []PMArea
}

// PMMessage is one rendered PM.
type PMMessage struct {
	Alternate     bool
	ID            int
	Member        *MemberCtx
	Subject       string
	Time          string
	Counter       int
	Body          string
	RecipientsTo  []string
	RecipientsBcc []string
	Labels        []*PMLabel
	FullyLabeled  bool
	IsRepliedTo   bool
	IsSelected    bool
	Href          string
	Link          string
}

// PMQuoted is the quoted message on the send form.
type PMQuoted struct {
	ID         int
	MemberName string
	MemberLink string
	Subject    string
	Time       string
	Body       string
}

// PMCtx is the page context for the PersonalMessage templates.
type PMCtx struct {
	MessageLimit    int
	LimitBarText    string
	LimitBar        int
	LimitBarPercent float64
	HasLimitBar     bool
	Labels          map[int]*PMLabel
	LabelOrder      []int
	UsingLabels     bool
	CurrentLabelID  int
	CurrentLabel    string
	Folder          string
	RedirectURL     string
	Areas           []PMAreaGroup
	Area            string

	// folder view:
	FromOrTo       string
	AllowHideEmail bool
	SortBy         string
	SortDirection  string
	ShowDelete     bool
	PageIndex      string
	Start          int
	Messages       []*PMMessage
	CanSendPM      bool

	// send form:
	Reply              bool
	QuotedMessage      *PMQuoted
	Subject            string
	Message            string
	To                 string
	Bcc                string
	PostError          map[string]bool
	ErrorMessages      []string
	CopyToOutbox       bool
	PreviewSubject     string
	PreviewMessage     string
	HasPreview         bool
	VisualVerification bool
	VerificationHref   string
	ShowSpellchecking  bool

	SendLog *pmLog

	// search:
	SPSearch        string
	SPUserspec      string
	SPSearchtype    int
	SPMinage        int
	SPMaxage        int
	SPSubjectOnly   bool
	SPShowComplete  bool
	SimpleSearch    bool
	SearchErrMsgs   []string
	SearchLabels    []*PMSearchLabel
	CheckAll        bool
	SearchParamsEnc string
	SearchResults   []*PMMessage

	// killall / prune / labels / report:
	DeleteAll  bool
	BoxName    string
	Admins     map[int]string
	AdminIDs   []int
	AdminCount int
	PMID       int
}

// MessageMain is MessageMain(): the ?action=pm dispatcher.
func (c *Ctx) MessageMain() {
	a := c.App
	scripturl := a.ScriptURL

	// No guests!
	c.isNotGuest("")

	// You're not supposed to be here at all, if you can't even read PMs.
	c.isAllowedTo("pm_read")

	c.loadLanguage("PersonalMessage")

	page := &PMCtx{Labels: map[int]*PMLabel{}, PostError: map[string]bool{}}
	c.Page = page

	// Load up the members maximum message capacity.
	if !c.User.IsAdmin {
		groups := make([]string, len(c.User.Groups))
		for i, g := range c.User.Groups {
			groups[i] = itoa(g)
		}
		var maxMessage, minMessage int
		a.DB.QueryRow(a.Q(`
			SELECT IFNULL(MAX(maxMessages), 0) AS topLimit, IFNULL(MIN(maxMessages), 0) AS bottomLimit
			FROM {$db_prefix}membergroups
			WHERE ID_GROUP IN (`+strings.Join(groups, ", ")+`)`)).Scan(&maxMessage, &minMessage)

		if minMessage != 0 {
			page.MessageLimit = maxMessage
		}
	}

	// Prepare the context for the capacity bar.
	if page.MessageLimit != 0 {
		bar := float64(c.User.Messages*100) / float64(page.MessageLimit)
		page.HasLimitBar = true
		page.LimitBar = int(bar)
		if page.LimitBar > 100 {
			page.LimitBar = 100
		}
		page.LimitBarText = phpSprintf(c.Txt("pm_currently_using"), itoa(c.User.Messages),
			formatPercent(float64(int(bar*10+0.5))/10))
	}

	// We should probably cache this information for speed.
	labelList := c.User.MessageLabels
	if labelList != "" {
		for k, v := range strings.Split(labelList, ",") {
			page.Labels[k] = &PMLabel{ID: k, Name: strings.TrimSpace(v)}
			page.LabelOrder = append(page.LabelOrder, k)
		}
	}
	page.Labels[-1] = &PMLabel{ID: -1, Name: c.Txt("pm_msg_label_inbox")}
	page.LabelOrder = append(page.LabelOrder, -1)

	rows, err := a.DB.Query(a.Q(`
		SELECT labels, is_read, COUNT(*) AS num
		FROM {$db_prefix}pm_recipients
		WHERE ID_MEMBER = ?
		GROUP BY labels, is_read`), c.User.ID)
	if err == nil {
		for rows.Next() {
			var labels string
			var isRead, num int
			rows.Scan(&labels, &isRead, &num)
			for _, thisLabel := range strings.Split(labels, ",") {
				if l, ok := page.Labels[atoi(thisLabel)]; ok {
					l.Messages += num
					if isRead&1 == 0 {
						l.UnreadMessages += num
					}
				}
			}
		}
		rows.Close()
	}

	// This determines if we have more labels than just the standard inbox.
	page.UsingLabels = len(page.Labels) > 1

	// Some stuff for the labels...
	page.CurrentLabelID = -1
	if c.REQUEST.Has("l") {
		if _, ok := page.Labels[c.REQUEST.Int("l")]; ok {
			page.CurrentLabelID = c.REQUEST.Int("l")
		}
	}
	page.CurrentLabel = page.Labels[page.CurrentLabelID].Name
	page.Folder = "inbox"
	if c.REQUEST.Str("f") == "outbox" {
		page.Folder = "outbox"
	}

	// This is convenient.
	page.RedirectURL = "action=pm;f=" + page.Folder
	if c.GET.Has("start") {
		page.RedirectURL += ";start=" + c.GET.Str("start")
	}
	if c.REQUEST.Has("l") {
		page.RedirectURL += ";l=" + c.REQUEST.Str("l")
	}

	// Build the linktree for all the actions...
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm", Name: c.Txt("144")})

	switch c.REQUEST.Str("sa") {
	case "manlabels":
		c.pmIndexBar(page, "manlabels")
		c.pmManageLabels(page)
	case "outbox":
		c.pmIndexBar(page, "outbox")
		c.pmMessageFolder(page)
	case "pmactions":
		c.pmIndexBar(page, "pmactions")
		c.pmActionsApply(page)
	case "prune":
		c.pmIndexBar(page, "prune")
		c.pmPrune(page)
	case "removeall":
		c.pmIndexBar(page, "removeall")
		c.pmKillAllQuery(page)
	case "removeall2":
		c.pmIndexBar(page, "removeall2")
		c.pmKillAll(page)
	case "report":
		c.pmIndexBar(page, "report")
		c.pmReportMessage(page)
	case "search":
		c.pmIndexBar(page, "search")
		c.pmMessageSearch(page)
	case "search2":
		c.pmIndexBar(page, "search2")
		c.pmMessageSearch2(page)
	case "send":
		c.pmIndexBar(page, "send")
		c.pmMessagePost(page)
	case "send2":
		c.pmIndexBar(page, "send2")
		c.pmMessagePost2(page)
	default:
		c.pmMessageFolder(page)
	}
}

// pmIndexBar is messageIndexBar($area).
func (c *Ctx) pmIndexBar(page *PMCtx, area string) {
	a := c.App
	scripturl := a.ScriptURL

	folders := PMAreaGroup{Title: c.Txt("pm_messages")}
	folders.Areas = append(folders.Areas,
		PMArea{Key: "send", Link: `<a href="` + scripturl + `?action=pm;sa=send">` + c.Txt("321") + `</a>`, Href: scripturl + "?action=pm;sa=send"},
		PMArea{Key: "", IsSpacer: true},
		PMArea{Key: "inbox", Link: `<a href="` + scripturl + `?action=pm">` + c.Txt("316") + `</a>`, Href: scripturl + "?action=pm",
			UnreadMessages: &page.Labels[-1].UnreadMessages, Messages: &page.Labels[-1].Messages},
		PMArea{Key: "outbox", Link: `<a href="` + scripturl + `?action=pm;f=outbox">` + c.Txt("320") + `</a>`, Href: scripturl + "?action=pm;f=outbox"},
	)

	var labels PMAreaGroup
	if page.UsingLabels {
		labels = PMAreaGroup{Title: c.Txt("pm_labels")}
		for _, id := range page.LabelOrder {
			label := page.Labels[id]
			if label.ID == -1 {
				continue
			}
			labels.Areas = append(labels.Areas, PMArea{
				Key:            "label" + itoa(label.ID),
				Link:           `<a href="` + scripturl + `?action=pm;l=` + itoa(label.ID) + `">` + label.Name + `</a>`,
				Href:           scripturl + "?action=pm;l=" + itoa(label.ID),
				UnreadMessages: &label.UnreadMessages,
				Messages:       &label.Messages,
			})
		}
	}

	pref := PMAreaGroup{Title: c.Txt("pm_preferences")}
	pref.Areas = append(pref.Areas,
		PMArea{Key: "search", Link: `<a href="` + scripturl + `?action=pm;sa=search">` + c.Txt("pm_search_bar_title") + `</a>`, Href: scripturl + "?action=pm;sa=search"},
		PMArea{Key: "manlabels", Link: `<a href="` + scripturl + `?action=pm;sa=manlabels">` + c.Txt("pm_manage_labels") + `</a>`, Href: scripturl + "?action=pm;sa=manlabels"},
		PMArea{Key: "prune", Link: `<a href="` + scripturl + `?action=pm;sa=prune">` + c.Txt("pm_prune") + `</a>`, Href: scripturl + "?action=pm;sa=prune"},
	)

	// Do we have a limit on the amount of messages we can keep?
	if page.MessageLimit != 0 {
		bar := float64(c.User.Messages*100) / float64(page.MessageLimit)
		page.HasLimitBar = true
		page.LimitBarPercent = bar
		page.LimitBar = int(bar)
		if page.LimitBar > 100 {
			page.LimitBar = 100
		}
		page.LimitBarText = phpSprintf(c.Txt("pm_currently_using"), itoa(c.User.Messages), formatPercent(math.Floor(bar*10+0.5)/10))

		pref.Areas = append(pref.Areas, PMArea{Key: "limit_bar", IsLimitBar: true})
	}

	page.Areas = append(page.Areas, folders)
	if page.UsingLabels {
		page.Areas = append(page.Areas, labels)
	}
	page.Areas = append(page.Areas, pref)

	// Where we are now.
	page.Area = area

	// obExit will know what to do!
	c.TemplateLayers = append(c.TemplateLayers, "pm")
}

// pmMessageFolder is MessageFolder(): an inbox/outbox listing.
func (c *Ctx) pmMessageFolder(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL

	// Make sure the starting location is valid.
	startParam := c.GET.Str("start")
	startIsNew := false
	if c.GET.Has("start") && startParam != "new" {
		// numeric
	} else if !c.GET.Has("start") && !empty(c.Options["view_newest_pm_first"]) {
		startParam = "0"
	} else {
		startIsNew = true
	}

	// Set up some basic theme stuff.
	page.AllowHideEmail = !a.SettingEmpty("allow_hideEmail")
	page.FromOrTo = "from"
	if page.Folder == "outbox" {
		page.FromOrTo = "to"
	}

	labelQuery := ""
	if page.Folder != "outbox" {
		labelQuery = `
			AND FIND_IN_SET('` + itoa(page.CurrentLabelID) + `', pmr.labels)`
	}

	// Set the index bar correct!
	if page.CurrentLabelID == -1 {
		c.pmIndexBar(page, page.Folder)
	} else {
		c.pmIndexBar(page, "label"+itoa(page.CurrentLabelID))
	}

	// Sorting the folder.
	sortMethods := map[string]string{
		"date":    "pm.ID_PM",
		"name":    "IFNULL(mem.realName, '')",
		"subject": "pm.subject",
	}

	sortExpr := ""
	descending := false
	if expr, ok := sortMethods[c.GET.Str("sort")]; ok {
		page.SortBy = c.GET.Str("sort")
		sortExpr = expr
		descending = c.GET.Has("desc")
	} else {
		page.SortBy = "date"
		sortExpr = "pm.ID_PM"
	}

	if !empty(c.Options["view_newest_pm_first"]) {
		descending = !descending
	}

	page.SortDirection = "up"
	if descending {
		page.SortDirection = "down"
	}

	// Why would you want access to your outbox if you're not allowed to send
	// anything?
	if page.Folder == "outbox" {
		c.isAllowedTo("pm_send")
	}

	// Set the text to resemble the current folder.
	pmbox := c.Txt("316")
	if page.Folder == "outbox" {
		pmbox = c.Txt("320")
	}
	page.BoxName = pmbox

	// Now, build the link tree!
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;f=" + page.Folder, Name: pmbox})

	// Build it further for a label.
	if page.CurrentLabelID != -1 {
		c.LinkTree = append(c.LinkTree, Link{
			URL:  scripturl + "?action=pm;f=" + page.Folder + ";l=" + itoa(page.CurrentLabelID),
			Name: c.Txt("pm_current_label") + ": " + page.CurrentLabel,
		})
	}

	// Mark all messages as read if in the inbox.
	if page.Folder != "outbox" && page.Labels[page.CurrentLabelID].UnreadMessages != 0 {
		c.markMessages(nil, itoa(page.CurrentLabelID), 0, page.Labels)
	}

	// Figure out how many messages there are.
	var maxMessages int
	if page.Folder == "outbox" {
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*)
			FROM {$db_prefix}personal_messages
			WHERE ID_MEMBER_FROM = ?
				AND deletedBySender = 0`), c.User.ID).Scan(&maxMessages)
	} else {
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*)
			FROM {$db_prefix}pm_recipients AS pmr
			WHERE pmr.ID_MEMBER = ` + itoa(c.User.ID) + `
				AND pmr.deleted = 0` + labelQuery)).Scan(&maxMessages)
	}

	// Only show the button if there are messages to delete.
	page.ShowDelete = maxMessages > 0

	// Start on the last page.
	start := atoi(startParam)
	if startIsNew || !regexp.MustCompile(`^\d+$`).MatchString(startParam) || start >= maxMessages {
		if maxMessages > 0 {
			start = (maxMessages - 1) - ((maxMessages - 1) % a.SettingInt("defaultMaxMessages"))
		} else {
			start = 0
		}
	} else if start < 0 {
		start = 0
	}

	// ... but wait - what if we want to start from a specific message?
	if c.GET.Has("pmid") {
		pmid := c.GET.Int("pmid")

		// With only one page of PM's we're gonna want page 1.
		if maxMessages <= a.SettingInt("defaultMaxMessages") {
			start = 0
		} else {
			op := "<"
			if descending {
				op = ">"
			}
			if page.Folder == "outbox" {
				a.DB.QueryRow(a.Q(`
					SELECT COUNT(*)
					FROM {$db_prefix}personal_messages
					WHERE ID_MEMBER_FROM = ?
						AND deletedBySender = 0
						AND ID_PM `+op+` `+itoa(pmid)), c.User.ID).Scan(&start)
			} else {
				a.DB.QueryRow(a.Q(`
					SELECT COUNT(*)
					FROM {$db_prefix}pm_recipients AS pmr
					WHERE pmr.ID_MEMBER = ` + itoa(c.User.ID) + `
						AND pmr.deleted = 0` + labelQuery + `
						AND ID_PM ` + op + ` ` + itoa(pmid))).Scan(&start)
			}

			start = a.SettingInt("defaultMaxMessages") * (start / a.SettingInt("defaultMaxMessages"))
		}
	}

	// Set up the page index.
	labelParam := ""
	if c.REQUEST.Has("l") {
		labelParam = ";l=" + itoa(c.REQUEST.Int("l"))
	}
	descParam := ""
	if c.GET.Has("desc") {
		descParam = ";desc"
	}
	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=pm;f="+page.Folder+labelParam+";sort="+page.SortBy+descParam,
		start, maxMessages, a.SettingInt("defaultMaxMessages"), false)
	page.Start = start

	// Load the messages up...
	from := `{$db_prefix}personal_messages AS pm`
	joinName := ""
	where := ""
	if page.Folder == "outbox" {
		if page.SortBy == "name" {
			joinName = `
			LEFT JOIN {$db_prefix}pm_recipients AS pmr ON (pmr.ID_PM = pm.ID_PM)
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = pmr.ID_MEMBER)`
		}
		where = `pm.ID_MEMBER_FROM = ` + itoa(c.User.ID) + `
			AND pm.deletedBySender = 0`
	} else {
		from += `, {$db_prefix}pm_recipients AS pmr`
		if page.SortBy == "name" {
			joinName = `
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = pm.ID_MEMBER_FROM)`
		}
		where = `pmr.ID_PM = pm.ID_PM
			AND pmr.ID_MEMBER = ` + itoa(c.User.ID) + `
			AND pmr.deleted = 0` + labelQuery
	}
	pmsgCond := ""
	limitClause := `
		LIMIT ` + itoa(a.SettingInt("defaultMaxMessages")) + ` OFFSET ` + itoa(start)
	if !empty(c.GET.Str("pmsg")) {
		pmsgCond = `
			AND pm.ID_PM = ` + itoa(c.GET.Int("pmsg"))
		limitClause = ""
	}
	orderExpr := sortExpr
	if sortExpr == "pm.ID_PM" && page.Folder != "outbox" {
		orderExpr = "pmr.ID_PM"
	}
	orderDir := " ASC"
	if descending {
		orderDir = " DESC"
	}

	var pms []int
	var posters []int
	if page.Folder == "outbox" {
		posters = append(posters, c.User.ID)
	}
	recipientsTo := map[int][]string{}
	recipientsBcc := map[int][]string{}
	rows, err := a.DB.Query(a.Q(`
		SELECT pm.ID_PM, pm.ID_MEMBER_FROM
		FROM ` + from + joinName + `
		WHERE ` + where + pmsgCond + `
		ORDER BY ` + orderExpr + orderDir + limitClause))
	if err == nil {
		seen := map[int]bool{}
		for rows.Next() {
			var idPM, idFrom int
			rows.Scan(&idPM, &idFrom)
			if !seen[idPM] {
				seen[idPM] = true
				pms = append(pms, idPM)
				if idFrom != 0 && page.Folder != "outbox" {
					posters = append(posters, idFrom)
				}
				recipientsTo[idPM] = []string{}
				recipientsBcc[idPM] = []string{}
			}
		}
		rows.Close()
	}

	messageLabels := map[int][]*PMLabel{}
	messageReplied := map[int]bool{}

	type pmRow struct {
		id, fromID    int
		subject, body string
		msgtime       int64
		fromName      string
	}
	var pmRows []pmRow

	if len(pms) > 0 {
		ids := make([]string, len(pms))
		for i, p := range pms {
			ids[i] = itoa(p)
		}
		in := strings.Join(ids, ", ")

		// Get recipients (no bcc-recipients for your inbox, you're not
		// supposed to know :P).
		rows, err := a.DB.Query(a.Q(`
			SELECT pmr.ID_PM, IFNULL(mem_to.ID_MEMBER, 0) AS ID_MEMBER_TO, IFNULL(mem_to.realName, '') AS toName, pmr.bcc, pmr.labels, pmr.is_read
			FROM {$db_prefix}pm_recipients AS pmr
				LEFT JOIN {$db_prefix}members AS mem_to ON (mem_to.ID_MEMBER = pmr.ID_MEMBER)
			WHERE pmr.ID_PM IN (` + in + `)`))
		if err == nil {
			for rows.Next() {
				var idPM, idTo, bcc, isRead int
				var toName, labels string
				rows.Scan(&idPM, &idTo, &toName, &bcc, &labels, &isRead)

				if page.Folder == "outbox" || bcc == 0 {
					display := c.Txt("28")
					if idTo != 0 {
						display = `<a href="` + scripturl + `?action=profile;u=` + itoa(idTo) + `">` + toName + `</a>`
					}
					if bcc == 0 {
						recipientsTo[idPM] = append(recipientsTo[idPM], display)
					} else {
						recipientsBcc[idPM] = append(recipientsBcc[idPM], display)
					}
				}

				if idTo == c.User.ID && page.Folder != "outbox" {
					messageReplied[idPM] = isRead&2 != 0

					if labels != "" {
						for _, v := range strings.Split(labels, ",") {
							if l, ok := page.Labels[atoi(v)]; ok {
								messageLabels[idPM] = append(messageLabels[idPM], l)
							}
						}
					}
				}
			}
			rows.Close()
		}

		// Load any users....
		c.loadMemberData(uniqueInts(posters))

		// Execute the query!
		outboxJoin := ""
		groupBy := ""
		if page.Folder == "outbox" {
			outboxJoin = `
				LEFT JOIN {$db_prefix}pm_recipients AS pmr ON (pmr.ID_PM = pm.ID_PM)`
			groupBy = `
			GROUP BY pm.ID_PM`
		}
		nameJoin := ""
		if page.SortBy == "name" {
			col := "pm.ID_MEMBER_FROM"
			if page.Folder == "outbox" {
				col = "pmr.ID_MEMBER"
			}
			nameJoin = `
				LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = ` + col + `)`
		}
		rows, err = a.DB.Query(a.Q(`
			SELECT pm.ID_PM, pm.subject, pm.ID_MEMBER_FROM, pm.body, pm.msgtime, pm.fromName
			FROM {$db_prefix}personal_messages AS pm` + outboxJoin + nameJoin + `
			WHERE pm.ID_PM IN (` + in + `)` + groupBy + `
			ORDER BY ` + sortExpr + orderDir))
		if err == nil {
			for rows.Next() {
				var r pmRow
				rows.Scan(&r.id, &r.subject, &r.fromID, &r.body, &r.msgtime, &r.fromName)
				pmRows = append(pmRows, r)
			}
			rows.Close()
		}
	}

	// prepareMessageContext, looped.
	tempSelected, _ := c.Session.Get("pm_selected").([]any)
	c.Session.Delete("pm_selected")
	selected := map[int]bool{}
	for _, v := range tempSelected {
		selected[int(toFloat(v))] = true
	}

	counter := start
	for _, r := range pmRows {
		// Use '(no subject)' if none was specified.
		if r.subject == "" {
			r.subject = c.Txt("24")
		}

		// Load the message's information.
		var member *MemberCtx
		if c.loadMemberContext(r.fromID) {
			member = c.memberCtx[r.fromID]
		} else {
			member = &MemberCtx{
				Name:      r.fromName,
				ID:        0,
				Group:     c.Txt("28"),
				Link:      r.fromName,
				HideEmail: true,
				IsGuest:   true,
				Options:   map[string]string{},
			}
		}

		// Censor all the important text...
		r.body = c.censorText(r.body)
		r.subject = c.censorText(r.subject)

		// Run UBBC interpreter on the message.
		r.body = c.parseBBCCached(r.body, true, "pm"+itoa(r.id))

		page.Messages = append(page.Messages, &PMMessage{
			Alternate:     counter%2 != 0,
			ID:            r.id,
			Member:        member,
			Subject:       r.subject,
			Time:          c.timeformat(r.msgtime),
			Counter:       counter,
			Body:          r.body,
			RecipientsTo:  recipientsTo[r.id],
			RecipientsBcc: recipientsBcc[r.id],
			Labels:        messageLabels[r.id],
			FullyLabeled:  len(messageLabels[r.id]) == len(page.Labels),
			IsRepliedTo:   messageReplied[r.id],
			IsSelected:    selected[r.id],
		})
		counter++
	}

	page.CanSendPM = c.allowedTo("pm_send")
	c.SubTemplate = templatePMFolder
	c.PageTitle = c.Txt("143")
}

// pmMessagePost is MessagePost(): the send-a-PM form.
func (c *Ctx) pmMessagePost(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL

	c.isAllowedTo("pm_send")

	// Extract out the spam settings - cause it's neat.
	spam := strings.SplitN(a.Setting("pm_spam_settings")+",,", ",", 4)
	maxRecipients, postsVerification, postsPerHour := atoi(spam[0]), atoi(spam[1]), atoi(spam[2])
	_ = maxRecipients

	// Set the title...
	c.PageTitle = c.Txt("148")

	page.Reply = c.REQUEST.Has("pmsg") || c.REQUEST.Has("quote")

	// Check whether we've gone over the limit of messages we can send per
	// hour.
	if postsPerHour != 0 && !c.allowedTo("admin_forum", "moderate_forum", "send_mail") {
		c.pmCheckPostsPerHour(postsPerHour)
	}

	var formSubject, formMessage string

	// Quoting/Replying to a message?
	if !empty(c.REQUEST.Str("pmsg")) {
		pmsg := c.REQUEST.Int("pmsg")
		quoted := c.pmLoadQuoted(page, pmsg)

		// Add 'Re: ' to it....
		prefix := c.Txt("response_prefix")
		formSubject = quoted.rawSubject
		if page.Reply && strings.TrimSpace(prefix) != "" && !strings.HasPrefix(formSubject, strings.TrimSpace(prefix)) {
			formSubject = prefix + formSubject
		}

		if c.REQUEST.Has("quote") {
			// Remove any nested quotes and <br />...
			formMessage = brToNlRe.ReplaceAllString(quoted.rawBody, "\n")
			if !a.SettingEmpty("removeNestedQuotes") {
				formMessage = nestedQuoteRe.ReplaceAllString(formMessage, "")
				formMessage = strings.TrimPrefix(formMessage, "\n")
				formMessage = strings.ReplaceAll(formMessage, "[/quote]", "")
			}
			if quoted.memberID == 0 {
				formMessage = "[quote author=&quot;" + quoted.realName + "&quot;]\n" + formMessage + "\n[/quote]"
			} else {
				formMessage = "[quote author=" + quoted.realName + " link=action=profile;u=" + itoa(quoted.memberID) + " date=" + i64toa(quoted.msgtime) + "]\n" + formMessage + "\n[/quote]"
			}
		}
	}

	// Sending by ID?  Replying to all?  Fetch the realName(s).
	if c.REQUEST.Has("u") {
		var membersTo []string

		if c.REQUEST.Str("u") == "all" && page.QuotedMessage != nil {
			membersTo = append(membersTo, "&quot;"+page.QuotedMessage.MemberName+"&quot;")

			rows, err := a.DB.Query(a.Q(`
				SELECT IFNULL(mem.realName, '')
				FROM {$db_prefix}pm_recipients AS pmr
					LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = pmr.ID_MEMBER)
				WHERE pmr.ID_PM = ?
					AND pmr.ID_MEMBER != ?
					AND bcc = 0`), c.REQUEST.Int("pmsg"), c.User.ID)
			if err == nil {
				for rows.Next() {
					var name string
					rows.Scan(&name)
					membersTo = append(membersTo, "&quot;"+Htmlspecialchars(name)+"&quot;")
				}
				rows.Close()
			}
		} else {
			var ids []string
			for _, u := range strings.Split(c.REQUEST.Str("u"), ",") {
				ids = append(ids, itoa(atoi(u)))
			}
			rows, err := a.DB.Query(a.Q(`
				SELECT realName
				FROM {$db_prefix}members
				WHERE ID_MEMBER IN (` + strings.Join(ids, ", ") + `)`))
			if err == nil {
				for rows.Next() {
					var name string
					rows.Scan(&name)
					membersTo = append(membersTo, "&quot;"+name+"&quot;")
				}
				rows.Close()
			}
		}

		c.REQUEST.Set("to", strings.Join(membersTo, ", "))
	}

	// Set the defaults...
	page.Subject = formSubject
	if page.Subject == "" {
		page.Subject = c.Txt("24")
	}
	page.Message = strings.NewReplacer(`"`, "&quot;", "<", "&lt;", ">", "&gt;").Replace(formMessage)
	page.To = c.REQUEST.Str("to")
	page.Bcc = c.REQUEST.Str("bcc")
	page.CopyToOutbox = !empty(c.Options["copy_to_outbox"])

	// And build the link tree.
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;sa=send", Name: c.Txt("321")})

	page.VisualVerification = !c.User.IsAdmin && postsVerification != 0 && c.User.Posts < postsVerification
	if page.VisualVerification {
		page.VerificationHref = scripturl + "?action=verificationcode;rand=" + fmt.Sprintf("%x", md5.Sum([]byte(itoa(rand.Int()))))

		// Skip I, J, L, O, Q, S and Z.
		characterRange := "ABCDEFGHKMNPRTUVWXY"
		code := ""
		for i := 0; i < 5; i++ {
			code += string(characterRange[rand.Intn(len(characterRange))])
		}
		c.Session.Set("visual_verification_code", code)
	}

	// Register this form and get a sequence number in $context.
	c.checkSubmitOnce("register")
	c.SubTemplate = templatePMSend
}

// pmQuotedRaw carries the un-bbc'd quoted message fields.
type pmQuotedRaw struct {
	rawSubject, rawBody, realName string
	memberID                      int
	msgtime                       int64
}

// pmLoadQuoted loads the quoted/replied-to PM into the page and returns the
// raw fields.
func (c *Ctx) pmLoadQuoted(page *PMCtx, pmsg int) *pmQuotedRaw {
	a := c.App
	scripturl := a.ScriptURL

	from := `{$db_prefix}personal_messages AS pm`
	cond := ""
	if page.Folder == "outbox" {
		cond = `
			AND pm.ID_MEMBER_FROM = ` + itoa(c.User.ID)
	} else {
		from += `, {$db_prefix}pm_recipients AS pmr`
		cond = `
			AND pmr.ID_PM = ` + itoa(pmsg) + `
			AND pmr.ID_MEMBER = ` + itoa(c.User.ID)
	}

	var idPM, memberID int
	var body, subject, memberName, realName string
	var msgtime int64
	err := a.DB.QueryRow(a.Q(`
		SELECT
			pm.ID_PM, pm.body, pm.subject, pm.msgtime, IFNULL(mem.memberName, ''),
			IFNULL(mem.ID_MEMBER, 0) AS ID_MEMBER, IFNULL(mem.realName, pm.fromName) AS realName
		FROM `+from+`
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = pm.ID_MEMBER_FROM)
		WHERE pm.ID_PM = `+itoa(pmsg)+cond+`
		LIMIT 1`)).Scan(&idPM, &body, &subject, &msgtime, &memberName, &memberID, &realName)
	if err != nil {
		c.fatalLangError("pm_not_yours", false)
	}

	// Censor the message.
	subject = c.censorText(subject)
	body = c.censorText(body)

	memberLink := realName
	if memberID != 0 {
		memberLink = `<a href="` + scripturl + `?action=profile;u=` + itoa(memberID) + `">` + realName + `</a>`
	}

	page.QuotedMessage = &PMQuoted{
		ID:         idPM,
		MemberName: realName,
		MemberLink: memberLink,
		Subject:    subject,
		Time:       c.timeformat(msgtime),
		Body:       c.parseBBCCached(body, true, "pm"+itoa(idPM)),
	}

	return &pmQuotedRaw{rawSubject: subject, rawBody: body, realName: realName, memberID: memberID, msgtime: msgtime}
}

// pmCheckPostsPerHour applies the hourly PM flood limit.
func (c *Ctx) pmCheckPostsPerHour(postsPerHour int) {
	a := c.App

	var postCount int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(pr.ID_PM) AS postCount
		FROM {$db_prefix}personal_messages AS pm, {$db_prefix}pm_recipients AS pr
		WHERE pm.ID_MEMBER_FROM = ?
			AND pm.msgtime > ?
			AND pr.ID_PM = pm.ID_PM`), c.User.ID, time.Now().Unix()-3600).Scan(&postCount)

	if postCount != 0 && postCount >= postsPerHour {
		// Excempt moderators.
		var dummy int
		if err := a.DB.QueryRow(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}moderators WHERE ID_MEMBER = ?`),
			c.User.ID).Scan(&dummy); err != nil {
			c.fatalError(phpSprintf(c.Txt("pm_too_many_per_hour"), itoa(postsPerHour)), true)
		}
	}
}

// pmPostError is messagePostError($error_types, $to, $bcc).
func (c *Ctx) pmPostError(page *PMCtx, errorTypes []string, to, bcc string) {
	a := c.App
	scripturl := a.ScriptURL

	c.PageTitle = c.Txt("148")

	// Set everything up like before....
	page.To = to
	page.Bcc = bcc
	page.Subject = Htmlspecialchars(c.REQUEST.Str("subject"))
	page.Message = strings.ReplaceAll(Htmlspecialchars(c.REQUEST.Str("message")), "  ", "&nbsp; ")
	page.CopyToOutbox = !empty(c.REQUEST.Str("outbox"))
	page.Reply = !empty(c.REQUEST.Str("replied_to"))

	if page.Reply {
		c.pmLoadQuoted(page, c.REQUEST.Int("replied_to"))
	}

	// Build the link tree....
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;sa=send", Name: c.Txt("321")})

	// Set each of the errors for the template.
	c.loadLanguage("Errors")
	for _, errorType := range errorTypes {
		page.PostError[errorType] = true
		key := "error_" + errorType
		// There is no compatible language string for the verification code.
		if errorType == "wrong_verification_code" {
			page.ErrorMessages = append(page.ErrorMessages, c.Txt("visual_verification_failed"))
		} else if msg := c.Txt(key); msg != key {
			page.ErrorMessages = append(page.ErrorMessages, msg)
		}
	}

	// Check whether we need to show the code again.
	spam := strings.SplitN(a.Setting("pm_spam_settings")+",,", ",", 4)
	postsVerification := atoi(spam[1])
	page.VisualVerification = !c.User.IsAdmin && postsVerification != 0 && c.User.Posts < postsVerification
	if page.VisualVerification {
		page.VerificationHref = scripturl + "?action=verificationcode;rand=" + fmt.Sprintf("%x", md5.Sum([]byte(itoa(rand.Int()))))
	}

	// No check for the previous submission is needed.
	c.checkSubmitOnce("free")

	// Acquire a new form sequence number.
	c.checkSubmitOnce("register")
	c.SubTemplate = templatePMSend
}

// pmMessagePost2 is MessagePost2(): send it!
func (c *Ctx) pmMessagePost2(page *PMCtx) {
	a := c.App

	c.isAllowedTo("pm_send")

	// Extract out the spam settings - it saves database space!
	spam := strings.SplitN(a.Setting("pm_spam_settings")+",,", ",", 4)
	maxRecipients, postsVerification, postsPerHour := atoi(spam[0]), atoi(spam[1]), atoi(spam[2])

	// Check whether we've gone over the limit of messages we can send per
	// hour - fatal error if fails!
	if postsPerHour != 0 && !c.allowedTo("admin_forum", "moderate_forum", "send_mail") {
		c.pmCheckPostsPerHour(postsPerHour)
	}

	// Initialize the errors we're about to make.
	var postErrors []string

	// If your session timed out, show an error, but do allow to re-submit.
	if c.checkSession("post", "", false) != "" {
		postErrors = append(postErrors, "session_timeout")
	}

	subject := strings.TrimSpace(c.REQUEST.Str("subject"))
	to := c.POST.Str("to")
	if empty(to) {
		to = c.GET.Str("to")
	}
	bcc := c.POST.Str("bcc")
	if empty(bcc) {
		bcc = c.GET.Str("bcc")
	}

	// Did they make any mistakes?
	if subject == "" {
		postErrors = append(postErrors, "no_subject")
	}
	if !c.REQUEST.Has("message") || c.REQUEST.Str("message") == "" {
		postErrors = append(postErrors, "no_message")
	} else if !a.SettingEmpty("max_messageLength") && entityLen(c.REQUEST.Str("message")) > a.SettingInt("max_messageLength") {
		postErrors = append(postErrors, "long_message")
	}
	if empty(to) && empty(bcc) && empty(c.REQUEST.Str("u")) {
		postErrors = append(postErrors, "no_to")
	}

	// Wrong verification code?
	if !c.User.IsAdmin && postsVerification != 0 && c.User.Posts < postsVerification {
		code, _ := c.Session.Get("visual_verification_code").(string)
		if empty(c.REQUEST.Str("visual_verification_code")) || strings.ToUpper(c.REQUEST.Str("visual_verification_code")) != code {
			postErrors = append(postErrors, "wrong_verification_code")
		}
	}

	// If they did, give a chance to make ammends.
	if len(postErrors) > 0 {
		c.pmPostError(page, postErrors, Htmlspecialchars(to), Htmlspecialchars(bcc))
		return
	}

	// Want to take a second glance before you send?
	if c.REQUEST.Has("preview") {
		// Set everything up to be displayed.
		page.HasPreview = true
		page.PreviewSubject = Htmlspecialchars(subject)
		page.PreviewMessage = c.preparsecode(Htmlspecialchars(c.REQUEST.Str("message")), true)

		// Parse out the BBC if it is enabled.
		page.PreviewMessage = c.parseBBC(page.PreviewMessage, true)

		// Censor, as always.
		page.PreviewSubject = c.censorText(page.PreviewSubject)
		page.PreviewMessage = c.censorText(page.PreviewMessage)

		// Set a descriptive title.
		c.PageTitle = c.Txt("507") + " - " + page.PreviewSubject

		// Pretend they messed up :P.
		c.pmPostError(page, nil, Htmlspecialchars(to), Htmlspecialchars(bcc))
		return
	}

	// Protect from message spamming.
	c.spamProtection("spam")

	// Prevent double submission of this form.
	c.checkSubmitOnce("check")

	// Initialize member ID array.
	recipients := &pmRecipients{}

	// Format the to and bcc members.
	input := map[string][]string{"to": {}, "bcc": {}}

	if empty(c.REQUEST.Str("u")) {
		quoted := regexp.MustCompile(`"([^"]+)"`)
		parse := func(s string) []string {
			var out []string
			seen := map[string]bool{}
			for _, m := range quoted.FindAllStringSubmatch(s, -1) {
				if !seen[m[1]] {
					seen[m[1]] = true
					out = append(out, m[1])
				}
			}
			for _, p := range strings.Split(quoted.ReplaceAllString(s, ""), ",") {
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
			return out
		}

		if !empty(to) {
			input["to"] = parse(to)
		}
		if !empty(bcc) {
			input["bcc"] = parse(bcc)
		}

		for recType, rec := range input {
			var kept []string
			for _, member := range rec {
				if len(strings.TrimSpace(member)) > 0 {
					kept = append(kept, Htmlspecialchars(strings.ToLower(strings.TrimSpace(member))))
				}
			}
			input[recType] = kept
		}

		// Find the requested members - bcc and to.
		foundMembers := c.findMembers(append(append([]string{}, input["to"]...), input["bcc"]...), false, false, 0)

		// Store IDs of the members that were found.
		var foundIDs []int
		for id := range foundMembers {
			foundIDs = append(foundIDs, id)
		}
		sort.Ints(foundIDs)
		for _, id := range foundIDs {
			member := foundMembers[id]
			name := strings.ReplaceAll(member.Name, "&#039;", "'")

			candidates := []string{
				strings.ToLower(member.Username),
				strings.ToLower(name),
				strings.ToLower(member.Email),
			}
			for recType := range input {
				matched := false
				for _, cand := range candidates {
					for _, m := range input[recType] {
						if m == cand {
							matched = true
						}
					}
				}
				if matched {
					if recType == "to" {
						recipients.To = append(recipients.To, member.ID)
					} else {
						recipients.Bcc = append(recipients.Bcc, member.ID)
					}

					// Get rid of this username.
					var kept []string
					for _, m := range input[recType] {
						drop := false
						for _, cand := range candidates {
							if m == cand {
								drop = true
							}
						}
						if !drop {
							kept = append(kept, m)
						}
					}
					input[recType] = kept
				}
			}
		}
	} else {
		var ids []string
		for _, u := range strings.Split(c.REQUEST.Str("u"), ",") {
			ids = append(ids, itoa(atoi(u)))
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER
			FROM {$db_prefix}members
			WHERE ID_MEMBER IN (` + strings.Join(ids, ",") + `)`))
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				recipients.To = append(recipients.To, id)
			}
			rows.Close()
		}
	}

	var sendLog *pmLog

	// Before we send the PM, let's make sure we don't have an abuse of
	// numbers.
	if maxRecipients != 0 && len(recipients.To)+len(recipients.Bcc) > maxRecipients &&
		!c.allowedTo("moderate_forum", "send_mail", "admin_forum") {
		sendLog = &pmLog{Failed: []string{phpSprintf(c.Txt("pm_too_many_recipients"), itoa(maxRecipients))}}
	} else {
		// Do the actual sending of the PM.
		if len(recipients.To) > 0 || len(recipients.Bcc) > 0 {
			sendLog = c.sendpm(recipients, subject, c.REQUEST.Str("message"), !empty(c.REQUEST.Str("outbox")), nil)
		} else {
			sendLog = &pmLog{}
		}
	}

	// Add a log message for all recipients that were not found.
	for recType, rec := range input {
		if len(rec) > 0 {
			errKey := "bad_" + recType
			has := false
			for _, e := range postErrors {
				if e == errKey {
					has = true
				}
			}
			if !has {
				postErrors = append(postErrors, errKey)
			}
		}
		for _, member := range rec {
			sendLog.Failed = append(sendLog.Failed, phpSprintf(c.Txt("pm_error_user_not_found"), member))
		}
	}

	// Mark the message as "replied to".
	if len(sendLog.Sent) > 0 && !empty(c.REQUEST.Str("replied_to")) && c.REQUEST.Str("f") == "inbox" {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}pm_recipients
			SET is_read = is_read | 2
			WHERE ID_PM = ?
				AND ID_MEMBER = ?`), c.REQUEST.Int("replied_to"), c.User.ID)
	}

	// Keep the send report for the template (shown by template_send).
	page.SendLog = sendLog

	// If one or more of the recipient were invalid, go back to the post
	// screen with the failed usernames.
	if len(sendLog.Failed) > 0 {
		toStr := ""
		if len(input["to"]) > 0 {
			toStr = "&quot;" + strings.Join(input["to"], "&quot;, &quot;") + "&quot;"
		}
		bccStr := ""
		if len(input["bcc"]) > 0 {
			bccStr = "&quot;" + strings.Join(input["bcc"], "&quot;, &quot;") + "&quot;"
		}
		c.pmPostError(page, postErrors, toStr, bccStr)
		return
	}

	// Go back to the where they sent from, if possible...
	c.redirectExit(page.RedirectURL)
}

// pmActionsApply is MessageActionsApply(): delete/label the selected PMs.
func (c *Ctx) pmActionsApply(page *PMCtx) {
	a := c.App

	c.checkSession("request", "", true)

	if c.REQUEST.Has("del_selected") {
		c.REQUEST.Set("pm_action", "delete")
	}

	actions := map[int]string{}
	var actionOrder []int
	if !empty(c.REQUEST.Str("pm_action")) && c.REQUEST.Arr("pms") != nil {
		pms := c.REQUEST.Arr("pms")
		for _, k := range pms.Keys() {
			pm := atoi(pms.Str(k))
			if _, ok := actions[pm]; !ok {
				actionOrder = append(actionOrder, pm)
			}
			actions[pm] = c.REQUEST.Str("pm_action")
		}
	} else if arr := c.REQUEST.Arr("pm_actions"); arr != nil {
		for _, k := range arr.Keys() {
			pm := atoi(k)
			if _, ok := actions[pm]; !ok {
				actionOrder = append(actionOrder, pm)
			}
			actions[pm] = arr.Str(k)
		}
	}

	if len(actions) == 0 {
		c.redirectExit(page.RedirectURL)
	}

	var toDelete []int
	toLabel := map[int]int{}
	labelType := map[int]string{}
	var labelOrder []int
	for _, pm := range actionOrder {
		action := actions[pm]
		if action == "delete" {
			toDelete = append(toDelete, pm)
		} else {
			typ := "unk"
			if strings.HasPrefix(action, "add_") {
				typ = "add"
				action = action[4:]
			} else if strings.HasPrefix(action, "rem_") {
				typ = "rem"
				action = action[4:]
			}

			if action == "-1" || action == "0" || atoi(action) > 0 {
				toLabel[pm] = atoi(action)
				labelType[pm] = typ
				labelOrder = append(labelOrder, pm)
			}
		}
	}

	// Deleting, it looks like?
	if len(toDelete) > 0 {
		c.pmDeleteMessages(toDelete, page.Folder, nil)
	}

	// Are we labeling anything?
	if len(toLabel) > 0 && page.Folder == "inbox" {
		updateErrors := 0

		ids := make([]string, 0, len(toLabel))
		for _, pm := range labelOrder {
			ids = append(ids, itoa(pm))
		}

		type labelRow struct {
			idPM   int
			labels string
		}
		var labelRows []labelRow
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_PM, labels
			FROM {$db_prefix}pm_recipients
			WHERE ID_MEMBER = ` + itoa(c.User.ID) + `
				AND ID_PM IN (` + strings.Join(ids, ",") + `)`))
		if err == nil {
			for rows.Next() {
				var r labelRow
				rows.Scan(&r.idPM, &r.labels)
				labelRows = append(labelRows, r)
			}
			rows.Close()
		}

		for _, r := range labelRows {
			labels := []string{"-1"}
			if r.labels != "" {
				labels = strings.Split(strings.TrimSpace(r.labels), ",")
			}

			// Already exists?  Then... unset it!
			target := itoa(toLabel[r.idPM])
			foundAt := -1
			for i, l := range labels {
				if l == target {
					foundAt = i
					break
				}
			}
			if foundAt >= 0 && labelType[r.idPM] != "add" {
				labels = append(labels[:foundAt], labels[foundAt+1:]...)
			} else if labelType[r.idPM] != "rem" {
				labels = append(labels, target)
			}

			seen := map[string]bool{}
			var uniq []string
			for _, l := range labels {
				if !seen[l] {
					seen[l] = true
					uniq = append(uniq, l)
				}
			}
			set := strings.Join(uniq, ",")
			if set == "" {
				set = "-1"
			}

			// Check that this string isn't going to be too large for the
			// database. (PHP's `$set > 60` quirk compares numerically, so it
			// effectively never triggers; preserved as len check like 2.x.)
			if len(set) > 60 {
				updateErrors++
			} else {
				a.DB.Exec(a.Q(`
					UPDATE {$db_prefix}pm_recipients
					SET labels = ?
					WHERE ID_PM = ?
						AND ID_MEMBER = ?`), set, r.idPM, c.User.ID)
			}
		}

		// Any errors?
		if updateErrors != 0 {
			c.fatalError(phpSprintf(c.Txt("labels_too_many"), itoa(updateErrors)), true)
		}
	}

	// Back to the folder.
	var selected []any
	for _, pm := range labelOrder {
		selected = append(selected, pm)
	}
	c.Session.Set("pm_selected", selected)
	anchor := ""
	if len(toLabel) == 1 {
		anchor = "#" + itoa(labelOrder[0])
	}
	c.redirectExit(page.RedirectURL + anchor)
}

// pmKillAllQuery is MessageKillAllQuery().
func (c *Ctx) pmKillAllQuery(page *PMCtx) {
	// Only have to set up the template....
	c.SubTemplate = templatePMAskDelete
	c.PageTitle = c.Txt("412")
	page.DeleteAll = c.REQUEST.Str("f") == "all"

	// And set the folder name...
	boxName := c.Txt("316")
	if page.Folder == "outbox" {
		boxName = c.Txt("320")
	}
	page.BoxName = strings.ReplaceAll(c.Txt("412"), "PMBOX", boxName)
}

// pmKillAll is MessageKillAll(): delete ALL the messages!
func (c *Ctx) pmKillAll(page *PMCtx) {
	c.checkSession("get", "", true)

	// If all then delete all messages the user has.
	if c.REQUEST.Str("f") == "all" {
		c.pmDeleteMessages(nil, "", nil)
	} else if c.REQUEST.Str("f") != "outbox" {
		c.pmDeleteMessages(nil, "inbox", nil)
	} else {
		c.pmDeleteMessages(nil, "outbox", nil)
	}

	// Done... all gone.
	c.redirectExit(page.RedirectURL)
}

// pmPrune is MessagePrune().
func (c *Ctx) pmPrune(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL

	// Actually delete the messages.
	if c.REQUEST.Has("age") {
		c.checkSession("post", "", true)

		// Calculate the time to delete before.
		deleteTime := time.Now().Unix() - 86400*int64(c.REQUEST.Int("age"))

		// Array to store the IDs in.
		var toDelete []int

		// Select all the messages they have sent older than $deleteTime.
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_PM
			FROM {$db_prefix}personal_messages
			WHERE deletedBySender = 0
				AND ID_MEMBER_FROM = ?
				AND msgtime < ?`), c.User.ID, deleteTime)
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				toDelete = append(toDelete, id)
			}
			rows.Close()
		}

		// Select all messages in their inbox older than $deleteTime.
		rows, err = a.DB.Query(a.Q(`
			SELECT pmr.ID_PM
			FROM {$db_prefix}pm_recipients AS pmr, {$db_prefix}personal_messages AS pm
			WHERE pmr.deleted = 0
				AND pmr.ID_MEMBER = ?
				AND pm.ID_PM = pmr.ID_PM
				AND pm.msgtime < ?`), c.User.ID, deleteTime)
		if err == nil {
			for rows.Next() {
				var id int
				rows.Scan(&id)
				toDelete = append(toDelete, id)
			}
			rows.Close()
		}

		// Delete the actual messages.
		c.pmDeleteMessages(toDelete, "", nil)

		// Go back to their inbox.
		c.redirectExit(page.RedirectURL)
	}

	// Build the link tree elements.
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;sa=prune", Name: c.Txt("pm_prune")})

	c.SubTemplate = templatePMPrune
	c.PageTitle = c.Txt("pm_prune")
}

// pmManageLabels is ManageLabels().
func (c *Ctx) pmManageLabels(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL

	// Build the link tree elements...
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;sa=manlabels", Name: c.Txt("pm_manage_labels")})

	c.PageTitle = c.Txt("pm_manage_labels")
	c.SubTemplate = templatePMLabels

	theLabels := map[int]string{}
	var labelIDs []int
	for _, id := range page.LabelOrder {
		if id != -1 {
			theLabels[id] = page.Labels[id].Name
			labelIDs = append(labelIDs, id)
		}
	}

	if c.GET.Has("sesc") {
		c.checkSession("request", "", true)

		// This will be for updating messages.
		messageChanges := map[int]bool{}
		newLabels := map[int]int{}

		if c.POST.Has("add") {
			// Adding a new label?
			label := strings.ReplaceAll(Htmlspecialchars(strings.TrimSpace(c.POST.Str("label"))), ",", "&#044;")
			if entityLen(label) > 30 {
				label = entitySubstr(label, 0, 30)
			}
			if label != "" {
				next := 0
				for _, id := range labelIDs {
					if id >= next {
						next = id + 1
					}
				}
				theLabels[next] = label
				labelIDs = append(labelIDs, next)
			}
		} else if c.POST.Has("delete") && c.POST.Arr("delete_label") != nil {
			// Deleting an existing label?
			deleteSet := c.POST.Arr("delete_label")
			i := 0
			var keptIDs []int
			for _, id := range labelIDs {
				if deleteSet.Has(itoa(id)) {
					delete(theLabels, id)
					messageChanges[id] = true
				} else {
					newLabels[id] = i
					i++
					keptIDs = append(keptIDs, id)
				}
			}
			labelIDs = keptIDs
		} else if c.POST.Has("save") && c.POST.Arr("label_name") != nil {
			// The hardest one to deal with... changes.
			names := c.POST.Arr("label_name")
			i := 0
			var keptIDs []int
			for _, id := range labelIDs {
				if names.Has(itoa(id)) {
					name := strings.TrimSpace(strings.ReplaceAll(Htmlspecialchars(names.Str(itoa(id))), ",", "&#044;"))
					if entityLen(name) > 30 {
						name = entitySubstr(name, 0, 30)
					}
					if name != "" {
						theLabels[id] = name
						newLabels[id] = i
						i++
						keptIDs = append(keptIDs, id)
					} else {
						delete(theLabels, id)
						messageChanges[id] = true
					}
				} else {
					newLabels[id] = i
					i++
					keptIDs = append(keptIDs, id)
				}
			}
			labelIDs = keptIDs
		}

		// Save the label status.
		var ordered []string
		for _, id := range labelIDs {
			ordered = append(ordered, theLabels[id])
		}
		a.updateMemberDataMap(c.User.ID, map[string]any{"messageLabels": strings.Join(ordered, ",")})

		// Update all the messages currently with any label changes in them!
		if len(messageChanges) > 0 {
			searchSet := map[int]bool{}
			maxChanged := 0
			for id := range messageChanges {
				searchSet[id] = true
				if id > maxChanged {
					maxChanged = id
				}
			}
			maxNew := 0
			for id := range newLabels {
				if id > maxNew {
					maxNew = id
				}
			}
			for i := maxChanged + 1; i <= maxNew; i++ {
				searchSet[i] = true
			}

			var finds []string
			for id := range searchSet {
				finds = append(finds, "FIND_IN_SET('"+itoa(id)+"', labels)")
			}

			type recipRow struct {
				idPM   int
				labels string
			}
			var recips []recipRow
			rows, err := a.DB.Query(a.Q(`
				SELECT ID_PM, labels
				FROM {$db_prefix}pm_recipients
				WHERE ` + strings.Join(finds, " OR ") + `
					AND ID_MEMBER = ` + itoa(c.User.ID)))
			if err == nil {
				for rows.Next() {
					var r recipRow
					rows.Scan(&r.idPM, &r.labels)
					recips = append(recips, r)
				}
				rows.Close()
			}

			for _, r := range recips {
				// Do the long task of updating them...
				var toChange []string
				for _, value := range strings.Split(r.labels, ",") {
					v := atoi(value)
					if searchSet[v] {
						if newIdx, ok := newLabels[v]; ok {
							toChange = append(toChange, itoa(newIdx))
						}
					} else {
						toChange = append(toChange, value)
					}
				}

				if len(toChange) == 0 {
					toChange = []string{"-1"}
				}

				seen := map[string]bool{}
				var uniq []string
				for _, v := range toChange {
					if !seen[v] {
						seen[v] = true
						uniq = append(uniq, v)
					}
				}

				// Update the message.
				a.DB.Exec(a.Q(`
					UPDATE {$db_prefix}pm_recipients
					SET labels = ?
					WHERE ID_PM = ?
						AND ID_MEMBER = ?`), strings.Join(uniq, ","), r.idPM, c.User.ID)
			}
		}

		// To make the changes appear right away, redirect.
		c.redirectExit("action=pm;sa=manlabels")
	}
}

// pmReportMessage is ReportMessage().
func (c *Ctx) pmReportMessage(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL

	// Check that this feature is even enabled!
	if a.SettingEmpty("enableReportPM") || empty(c.REQUEST.Str("pmsg")) {
		c.fatalLangError("1", false)
	}

	page.PMID = c.REQUEST.Int("pmsg")
	c.PageTitle = c.Txt("pm_report_title")

	// If we're here, just send the user to the template, with a few useful
	// context bits.
	if !c.REQUEST.Has("report") {
		c.SubTemplate = templatePMReport

		// Now, get all the administrators.
		page.Admins = map[int]string{}
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_MEMBER, realName
			FROM {$db_prefix}members
			WHERE ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups)
			ORDER BY realName`))
		if err == nil {
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				page.Admins[id] = name
				page.AdminIDs = append(page.AdminIDs, id)
			}
			rows.Close()
		}

		// How many admins in total?
		page.AdminCount = len(page.Admins)
	} else {
		// Otherwise, let's get down to the sending stuff. First, pull out
		// the message contents, and verify it actually went to them!
		var subject, body, senderName string
		var msgtime int64
		var memberFromID int
		err := a.DB.QueryRow(a.Q(`
			SELECT pm.subject, pm.body, pm.msgtime, pm.ID_MEMBER_FROM, IFNULL(m.realName, pm.fromName) AS senderName
			FROM {$db_prefix}personal_messages AS pm, {$db_prefix}pm_recipients AS pmr
				LEFT JOIN {$db_prefix}members AS m ON (m.ID_MEMBER = pm.ID_MEMBER_FROM)
			WHERE pm.ID_PM = ?
				AND pmr.ID_PM = pm.ID_PM
				AND pmr.ID_MEMBER = ?
				AND pmr.deleted = 0
			LIMIT 1`), page.PMID, c.User.ID).Scan(&subject, &body, &msgtime, &memberFromID, &senderName)
		// Can only be a hacker here!
		if err != nil {
			c.fatalLangError("1", false)
		}

		// Remove the line breaks...
		body = brToNlRe.ReplaceAllString(body, "\n")

		// Get any other recipients of the email.
		var otherRecipients []string
		hiddenRecipients := 0
		rows, err := a.DB.Query(a.Q(`
			SELECT IFNULL(mem_to.ID_MEMBER, 0) AS ID_MEMBER_TO, IFNULL(mem_to.realName, '') AS toName, pmr.bcc
			FROM {$db_prefix}pm_recipients AS pmr
				LEFT JOIN {$db_prefix}members AS mem_to ON (mem_to.ID_MEMBER = pmr.ID_MEMBER)
			WHERE pmr.ID_PM = ?
				AND pmr.ID_MEMBER != ?`), page.PMID, c.User.ID)
		if err == nil {
			for rows.Next() {
				var idTo, bcc int
				var toName string
				rows.Scan(&idTo, &toName, &bcc)
				// If it's hidden still don't reveal their names - privacy
				// after all ;)
				if bcc != 0 {
					hiddenRecipients++
				} else {
					otherRecipients = append(otherRecipients, "[url="+scripturl+"?action=profile;u="+itoa(idTo)+"]"+toName+"[/url]")
				}
			}
			rows.Close()
		}

		if hiddenRecipients != 0 {
			otherRecipients = append(otherRecipients, phpSprintf(c.Txt("pm_report_pm_hidden"), itoa(hiddenRecipients)))
		}

		// Now let's get out and loop through the admins.
		adminCond := ""
		if !empty(c.REQUEST.Str("ID_ADMIN")) {
			adminCond = "AND ID_MEMBER = " + itoa(c.REQUEST.Int("ID_ADMIN"))
		}
		var adminIDs []int
		rows, err = a.DB.Query(a.Q(`
			SELECT ID_MEMBER, realName, lngfile
			FROM {$db_prefix}members
			WHERE (ID_GROUP = 1 OR FIND_IN_SET(1, additionalGroups))
				` + adminCond + `
			ORDER BY lngfile`))
		if err == nil {
			for rows.Next() {
				var id int
				var name, lngfile string
				rows.Scan(&id, &name, &lngfile)
				adminIDs = append(adminIDs, id)
			}
			rows.Close()
		}

		// Maybe we shouldn't advertise this?
		if len(adminIDs) == 0 {
			c.fatalLangError("1", false)
		}

		memberFromName := unHtmlspecialchars(senderName)

		// Make the body. (Single-language port: one message for all admins.)
		reportBody := strings.NewReplacer("{REPORTER}", unHtmlspecialchars(c.User.Name), "{SENDER}", memberFromName).
			Replace(c.Txt("pm_report_pm_user_sent"))
		reportBody += "\n[b]" + c.REQUEST.Str("reason") + "[/b]\n\n"
		if len(otherRecipients) > 0 {
			reportBody += c.Txt("pm_report_pm_other_recipients") + " " + strings.Join(otherRecipients, ", ") + "\n\n"
		}
		quoteAttr := "&quot;" + memberFromName + "&quot;"
		if memberFromID != 0 {
			quoteAttr = memberFromName + " link=action=profile;u=" + itoa(memberFromID) + " date=" + i64toa(msgtime)
		}
		reportBody += c.Txt("pm_report_pm_unedited_below") + "\n[quote author=" + quoteAttr + "]\n" + unHtmlspecialchars(body) + "[/quote]"

		reportSubject := subject
		if !strings.Contains(subject, c.Txt("pm_report_pm_subject")) {
			reportSubject = c.Txt("pm_report_pm_subject") + subject
		}

		c.sendpm(&pmRecipients{To: adminIDs}, reportSubject, reportBody, false, nil)

		// Leave them with a template.
		c.SubTemplate = templatePMReportDone
	}
}

// PMSearchLabel is one row in the PM search's label checkbox list.
type PMSearchLabel struct {
	ID      int
	Name    string
	Checked bool
}

// pmMessageSearch is MessageSearch(): the PM search form.
func (c *Ctx) pmMessageSearch(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL

	if c.REQUEST.Has("search") {
		page.SPSearch = Htmlspecialchars(unHtmlspecialchars(c.REQUEST.Str("search")))
	}

	// Build the list of labels that can be searched.
	var searchedLabels []int
	for _, id := range page.LabelOrder {
		label := page.Labels[id]
		page.SearchLabels = append(page.SearchLabels, &PMSearchLabel{
			ID: label.ID, Name: label.Name, Checked: true,
		})
	}
	page.CheckAll = len(searchedLabels) == 0 || len(page.SearchLabels) == len(searchedLabels)

	page.SimpleSearch = !a.SettingEmpty("simpleSearch") && !c.REQUEST.Has("advanced")
	c.PageTitle = c.Txt("pm_search_title")
	c.SubTemplate = templatePMSearch
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;sa=search", Name: c.Txt("pm_search_bar_title")})
}

// pmMessageSearch2 is MessageSearch2(): run the PM search. Forced to the inbox
// (outbox search is unsupported in SMF 1.1, as noted there).
func (c *Ctx) pmMessageSearch2(page *PMCtx) {
	a := c.App
	scripturl := a.ScriptURL
	me := c.User.ID

	page.Folder = "inbox"
	page.CanSendPM = c.allowedTo("pm_send")

	const maxMembersToSearch = 500

	// Decode params.
	pv := map[string]string{}
	if c.REQUEST.Has("params") {
		if dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(c.REQUEST.Str("params"), " ", "+")); err == nil {
			for _, data := range strings.Split(string(dec), `|"|`) {
				kv := strings.SplitN(data, `|'|`, 2)
				v := ""
				if len(kv) > 1 {
					v = kv[1]
				}
				pv[kv[0]] = v
			}
		}
	}
	pget := func(k string) string { return pv[k] }

	page.Start = c.GET.Int("start")

	advanced := !empty(pget("advanced")) || !empty(c.REQUEST.Str("advanced"))
	if !empty(pget("searchtype")) || c.REQUEST.Int("searchtype") == 2 {
		page.SPSearchtype = 2
	}
	if !empty(pget("minage")) {
		page.SPMinage = atoi(pget("minage"))
	} else if c.REQUEST.Int("minage") > 0 {
		page.SPMinage = c.REQUEST.Int("minage")
	}
	if !empty(pget("maxage")) {
		page.SPMaxage = atoi(pget("maxage"))
	} else if c.REQUEST.Has("maxage") && c.REQUEST.Int("maxage") != 9999 {
		page.SPMaxage = c.REQUEST.Int("maxage")
	}
	page.SPSubjectOnly = !empty(pget("subject_only")) || !empty(c.REQUEST.Str("subject_only"))
	page.SPShowComplete = !empty(pget("show_complete")) || !empty(c.REQUEST.Str("show_complete"))
	if !empty(pget("userspec")) {
		page.SPUserspec = pget("userspec")
	} else if u := c.REQUEST.Str("userspec"); u != "" && u != "*" {
		page.SPUserspec = u
	}

	// userQuery (poster filter).
	userQuery := ""
	if page.SPUserspec != "" {
		us := strings.ReplaceAll(Htmlspecialchars(page.SPUserspec), "&quot;", `"`)
		us = strings.NewReplacer("%", `\%`, "_", `\_`, "*", "%", "?", "_").Replace(us)
		var possible []string
		for _, m := range regexp.MustCompile(`"([^"]+)"`).FindAllStringSubmatch(us, -1) {
			possible = append(possible, m[1])
		}
		for _, p := range strings.Split(regexp.MustCompile(`"([^"]+)"`).ReplaceAllString(us, ""), ",") {
			possible = append(possible, p)
		}
		var clean []string
		for _, p := range possible {
			if p = strings.TrimSpace(p); p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) > 0 {
			likeName := func(col string) string {
				var parts []string
				for _, p := range clean {
					parts = append(parts, col+" LIKE '"+sqlEsc(p)+"' ESCAPE '\\'")
				}
				return strings.Join(parts, " OR ")
			}
			var ids []int
			rows, err := a.DB.Query(a.Q(`SELECT ID_MEMBER FROM {$db_prefix}members WHERE ` + likeName("realName")))
			if err == nil {
				for rows.Next() {
					var id int
					rows.Scan(&id)
					ids = append(ids, id)
				}
				rows.Close()
			}
			if len(ids) > maxMembersToSearch {
				userQuery = ""
			} else if len(ids) == 0 {
				userQuery = "AND pm.ID_MEMBER_FROM = 0 AND (" + likeName("pm.fromName") + ")"
			} else {
				userQuery = "AND (pm.ID_MEMBER_FROM IN (" + joinInts(ids) + ") OR (pm.ID_MEMBER_FROM = 0 AND (" + likeName("pm.fromName") + ")))"
			}
		}
	}

	// Sort.
	sort := pget("sort")
	sortDir := pget("sort_dir")
	if sort == "" && c.REQUEST.Str("sort") != "" {
		parts := strings.SplitN(c.REQUEST.Str("sort"), "|", 2)
		sort = parts[0]
		if len(parts) > 1 {
			sortDir = parts[1]
		}
	}
	if sort != "ID_PM" {
		sort = "ID_PM"
	}
	if sortDir != "asc" {
		sortDir = "desc"
	}

	// Label filtering (inbox + advanced + using labels).
	labelQuery := ""
	if advanced && page.UsingLabels {
		var searchLabels []int
		if v, ok := pv["labels"]; ok && v != "" {
			for _, s := range strings.Split(v, ",") {
				searchLabels = append(searchLabels, atoi(s))
			}
		} else if arr, ok := c.REQUEST.Get("searchlabel").(*PArray); ok {
			arr.Values(func(_ string, val any) {
				if s, ok := val.(string); ok {
					searchLabels = append(searchLabels, atoi(s))
				}
			})
		}
		var lstr []string
		for _, l := range searchLabels {
			lstr = append(lstr, itoa(l))
		}
		pv["labels"] = strings.Join(lstr, ",")
		if len(searchLabels) == 0 {
			page.SearchErrMsgs = append(page.SearchErrMsgs, "no_labels_selected")
		} else if len(searchLabels) != len(page.Labels) {
			var conds []string
			for _, l := range searchLabels {
				conds = append(conds, "FIND_IN_SET('"+itoa(l)+"', pmr.labels)")
			}
			labelQuery = "\n\t\t\tAND (" + strings.Join(conds, " OR ") + ")"
		}
	}

	// The search string.
	if pget("search") != "" {
		page.SPSearch = pget("search")
	} else if c.REQUEST.Has("search") {
		page.SPSearch = c.REQUEST.Str("search")
	}
	if strings.TrimSpace(page.SPSearch) == "" {
		page.SearchErrMsgs = append(page.SearchErrMsgs, "invalid_search_string")
	}

	// Phrases + words, excludes.
	var searchArray, excludedWords []string
	for _, m := range searchPhraseRe.FindAllStringSubmatch(page.SPSearch, -1) {
		if m[1] == "-" {
			if w := strings.ToLower(strings.TrimSpace(m[2])); w != "" {
				excludedWords = append(excludedWords, w)
			}
		} else {
			searchArray = append(searchArray, m[2])
		}
	}
	for _, w := range strings.Split(searchPhraseRe.ReplaceAllString(page.SPSearch, " "), " ") {
		if strings.HasPrefix(strings.TrimSpace(w), "-") {
			if ww := strings.ToLower(strings.TrimSpace(w))[1:]; ww != "" {
				excludedWords = append(excludedWords, ww)
			}
			continue
		}
		searchArray = append(searchArray, w)
	}
	var cleanArray []string
	seen := map[string]bool{}
	for _, v := range searchArray {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		v = Htmlspecialchars(v)
		if !seen[v] {
			seen[v] = true
			cleanArray = append(cleanArray, v)
		}
	}
	searchArray = cleanArray
	if len(searchArray) == 0 {
		page.SearchErrMsgs = append(page.SearchErrMsgs, "invalid_search_string")
	}

	excludedSet := map[string]bool{}
	for _, w := range excludedWords {
		excludedSet[w] = true
	}
	allWords := append(append([]string{}, searchArray...), excludedWords...)

	// Encode params for pagination.
	var enc []string
	putP := func(k, v string) { enc = append(enc, k+`|'|`+v) }
	if page.SPSearch != "" {
		putP("search", page.SPSearch)
	}
	if advanced {
		putP("advanced", "1")
	}
	if page.SPSearchtype != 0 {
		putP("searchtype", "2")
	}
	if page.SPMinage != 0 {
		putP("minage", itoa(page.SPMinage))
	}
	if page.SPMaxage != 0 {
		putP("maxage", itoa(page.SPMaxage))
	}
	if page.SPSubjectOnly {
		putP("subject_only", "1")
	}
	if page.SPShowComplete {
		putP("show_complete", "1")
	}
	if page.SPUserspec != "" {
		putP("userspec", page.SPUserspec)
	}
	if v, ok := pv["labels"]; ok && v != "" {
		putP("labels", v)
	}
	page.SearchParamsEnc = base64.StdEncoding.EncodeToString([]byte(strings.Join(enc, `|"|`)))

	// Re-escape for the form.
	page.SPSearch = Htmlspecialchars(page.SPSearch)
	page.SPUserspec = Htmlspecialchars(page.SPUserspec)

	// Errors? Back to the form.
	if len(page.SearchErrMsgs) > 0 {
		c.loadLanguage("Errors")
		msgs := page.SearchErrMsgs
		page.SearchErrMsgs = nil
		for _, e := range msgs {
			page.SearchErrMsgs = append(page.SearchErrMsgs, c.Txt("error_"+e))
		}
		c.pmMessageSearch(page)
		return
	}

	// The body/subject query.
	var andParts []string
	for _, w := range allWords {
		if w == "" {
			continue
		}
		if page.SPSubjectOnly {
			andParts = append(andParts, likeCond("pm.subject", w, excludedSet[w]))
		} else {
			if excludedSet[w] {
				andParts = append(andParts, "(pm.subject NOT LIKE '%"+sqlLikeEsc(w)+"%' ESCAPE '\\' AND pm.body NOT LIKE '%"+sqlLikeEsc(w)+"%' ESCAPE '\\')")
			} else {
				andParts = append(andParts, "(pm.subject LIKE '%"+sqlLikeEsc(w)+"%' ESCAPE '\\' OR pm.body LIKE '%"+sqlLikeEsc(w)+"%' ESCAPE '\\')")
			}
		}
	}
	joiner := " AND "
	if page.SPSearchtype == 2 {
		joiner = " OR "
	}
	searchQuery := "1"
	if len(andParts) > 0 {
		searchQuery = strings.Join(andParts, joiner)
	}

	folderCond := "\n\t\t\tAND pmr.ID_MEMBER = " + itoa(me) + "\n\t\t\tAND pmr.deleted = 0"

	// Qualify the sort column (both joined tables have ID_PM).
	sortExpr := "pm." + sort

	var numResults int
	a.DB.QueryRow(a.Q(`
		SELECT COUNT(*)
		FROM ({$db_prefix}pm_recipients AS pmr, {$db_prefix}personal_messages AS pm)
		WHERE pm.ID_PM = pmr.ID_PM` + folderCond + `
			` + userQuery + labelQuery + `
			AND (` + searchQuery + `)`)).Scan(&numResults)

	var foundMessages, posters []int
	rows, err := a.DB.Query(a.Q(`
		SELECT pm.ID_PM, pm.ID_MEMBER_FROM
		FROM ({$db_prefix}pm_recipients AS pmr, {$db_prefix}personal_messages AS pm)
		WHERE pm.ID_PM = pmr.ID_PM` + folderCond + `
			` + userQuery + labelQuery + `
			AND (` + searchQuery + `)
		ORDER BY ` + sortExpr + ` ` + sortDir + `
		LIMIT ` + itoa(a.SettingInt("search_results_per_page")) + ` OFFSET ` + itoa(page.Start)))
	if err == nil {
		for rows.Next() {
			var idPM, idFrom int
			rows.Scan(&idPM, &idFrom)
			foundMessages = append(foundMessages, idPM)
			posters = append(posters, idFrom)
		}
		rows.Close()
	}
	c.loadMemberData(uniqueInts(posters))

	page.PageIndex, _ = c.constructPageIndex(scripturl+"?action=pm;sa=search2;params="+page.SearchParamsEnc, page.Start, numResults, a.SettingInt("search_results_per_page"), false)

	if len(foundMessages) > 0 {
		in := joinInts(foundMessages)

		// Recipients + labels + replied.
		recipientsTo := map[int][]string{}
		messageLabels := map[int][]*PMLabel{}
		messageReplied := map[int]bool{}
		firstLabel := map[int]int{}
		rrows, err := a.DB.Query(a.Q(`
			SELECT pmr.ID_PM, IFNULL(mem_to.ID_MEMBER, 0) AS ID_MEMBER_TO, IFNULL(mem_to.realName, '') AS toName, pmr.bcc, pmr.labels, pmr.is_read
			FROM {$db_prefix}pm_recipients AS pmr
				LEFT JOIN {$db_prefix}members AS mem_to ON (mem_to.ID_MEMBER = pmr.ID_MEMBER)
			WHERE pmr.ID_PM IN (` + in + `)`))
		if err == nil {
			for rrows.Next() {
				var idPM, idTo, bcc, isRead int
				var toName, labels string
				rrows.Scan(&idPM, &idTo, &toName, &bcc, &labels, &isRead)
				if bcc == 0 {
					disp := c.Txt("28")
					if idTo != 0 {
						disp = `<a href="` + scripturl + `?action=profile;u=` + itoa(idTo) + `">` + toName + `</a>`
					}
					recipientsTo[idPM] = append(recipientsTo[idPM], disp)
				}
				if idTo == me {
					messageReplied[idPM] = isRead&2 != 0
					if labels != "" {
						for _, v := range strings.Split(labels, ",") {
							if l, ok := page.Labels[atoi(v)]; ok {
								messageLabels[idPM] = append(messageLabels[idPM], l)
							}
							if _, ok := firstLabel[idPM]; !ok && v != "-1" {
								firstLabel[idPM] = atoi(v)
							}
						}
					}
				}
			}
			rrows.Close()
		}

		// The messages.
		type pmRow struct {
			id, fromID    int
			subject, body string
			msgtime       int64
			fromName      string
		}
		var pmRows []pmRow
		mrows, err := a.DB.Query(a.Q(`
			SELECT pm.ID_PM, pm.subject, pm.ID_MEMBER_FROM, pm.body, pm.msgtime, pm.fromName
			FROM {$db_prefix}personal_messages AS pm
			WHERE pm.ID_PM IN (` + in + `)
			ORDER BY ` + sort + ` ` + sortDir + `
			LIMIT ` + itoa(len(foundMessages))))
		if err == nil {
			for mrows.Next() {
				var r pmRow
				mrows.Scan(&r.id, &r.subject, &r.fromID, &r.body, &r.msgtime, &r.fromName)
				pmRows = append(pmRows, r)
			}
			mrows.Close()
		}

		counter := 0
		for _, r := range pmRows {
			if r.subject == "" {
				r.subject = c.Txt("24")
			}
			var member *MemberCtx
			if c.loadMemberContext(r.fromID) {
				member = c.memberCtx[r.fromID]
			} else {
				member = &MemberCtx{Name: r.fromName, ID: 0, Group: c.Txt("28"), Link: r.fromName, HideEmail: true, IsGuest: true, Options: map[string]string{}}
			}
			r.body = c.censorText(r.body)
			r.subject = c.censorText(r.subject)
			r.body = c.parseBBCCached(r.body, true, "pm"+itoa(r.id))

			labelParam := ""
			if l, ok := firstLabel[r.id]; ok {
				labelParam = ";l=" + itoa(l)
			}
			href := scripturl + "?action=pm;f=" + page.Folder + labelParam + ";pmid=" + itoa(r.id) + "#msg" + itoa(r.id)
			counter++
			page.SearchResults = append(page.SearchResults, &PMMessage{
				ID:           r.id,
				Member:       member,
				Subject:      r.subject,
				Body:         r.body,
				Time:         c.timeformat(r.msgtime),
				RecipientsTo: recipientsTo[r.id],
				Labels:       messageLabels[r.id],
				FullyLabeled: len(messageLabels[r.id]) == len(page.Labels),
				IsRepliedTo:  messageReplied[r.id],
				Href:         href,
				Link:         `<a href="` + href + `">` + r.subject + `</a>`,
				Counter:      counter,
			})
		}
	}

	c.PageTitle = c.Txt("pm_search_title")
	c.SubTemplate = templatePMSearchResults
	page.Area = "search"
	c.LinkTree = append(c.LinkTree, Link{URL: scripturl + "?action=pm;sa=search", Name: c.Txt("pm_search_bar_title")})
}
