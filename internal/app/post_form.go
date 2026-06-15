package app

// Post() and getTopic() from Sources/Post.php — the post/reply/edit form.

import (
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	registerAction("post", (*Ctx).Post)
}

// PostIcon is one selectable message icon.
type PostIcon struct {
	Value    string
	Name     string
	URL      string
	IsLast   bool
	Selected bool
}

// PostChoice is one poll choice on the form.
type PostChoice struct {
	ID     int
	Number int
	Label  string
	IsLast bool
}

// PostPrevPost is one entry of the topic summary.
type PostPrevPost struct {
	Poster  string
	Message string
	Time    string
	ID      int
	IsNew   bool
}

// PostAttachment is one current attachment on the form.
type PostAttachment struct {
	Name      string
	ID        string
	Unchecked bool
}

// PostCtx is the page context for tpl_post.go.
type PostCtx struct {
	ShowSpellchecking     bool
	MakePoll              bool
	Notify                bool
	Sticky                bool
	Locked                bool
	CanLock               bool
	CanSticky             bool
	CanNotify             bool
	CanMove               bool
	CanAnnounce           bool
	NumReplies            int
	PollQuestion          string
	PollMaxVotes          string
	PollHide              int
	PollExpire            string
	PollChangeVote        bool
	Choices               []PostChoice
	PostError             map[string]bool
	ErrorMessages         []string
	ErrorType             string
	PreviewMessage        string
	PreviewSubject        string
	HasPreview            bool
	UseSmileys            bool
	Icon                  string
	IconURL               string
	Icons                 []PostIcon
	Destination           string
	SubmitLabel           string
	LastModified          string
	GuestName             string
	GuestEmail            string
	ShowGuestName         bool
	Subject               string
	Message               string
	AllowedExtensions     string
	CurrentAttachments    []PostAttachment
	CanPostAttachment     bool
	NumAllowedAttachments int
	BackToTopic           bool
	ShowAdditionalOptions bool
	IsNewTopic            bool
	IsNewPost             bool
	IsFirstPost           bool
	PreviousPosts         []PostPrevPost
	ResponsePrefix        string
}

var (
	brToNlRe      = regexp.MustCompile(`(?i)<br(?: /)?>`)
	nestedQuoteRe = regexp.MustCompile(`(?is)\n?\[quote.*?\].+?\[/quote\]\n?`)
)

func i64toa(n int64) string { return strconv.FormatInt(n, 10) }

// postErrSet keeps post_error keys in insertion order (PHP array semantics).
type postErrSet struct {
	keys []string
	m    map[string]bool
	msg  map[string]string // resolved message overrides (new_reply/new_replies)
}

func newPostErrSet() *postErrSet {
	return &postErrSet{m: map[string]bool{}, msg: map[string]string{}}
}

func (p *postErrSet) add(key string) {
	if !p.m[key] {
		p.m[key] = true
		p.keys = append(p.keys, key)
	}
}

func (p *postErrSet) empty() bool { return len(p.keys) == 0 }

// Post is Post(): show the posting form.
func (c *Ctx) Post() {
	a := c.App
	scripturl := a.ScriptURL

	c.loadLanguage("Post")

	page := &PostCtx{}
	c.Page = page

	hasPoll := c.REQUEST.Has("poll")
	msgID := c.REQUEST.Int("msg")
	hasMsg := c.REQUEST.Has("msg")

	// You can't reply with a poll... hacker.
	if hasPoll && c.Topic != 0 && !hasMsg {
		hasPoll = false
	}

	// You must be posting to *some* board.
	if c.Board == 0 {
		c.fatalLangError("smf232", false)
	}

	// No message is complete without a topic.
	if c.Topic == 0 && !empty(c.REQUEST.Str("msg")) {
		var t int
		if err := a.DB.QueryRow(a.Q(`SELECT ID_TOPIC FROM {$db_prefix}messages WHERE ID_MSG = ?`), msgID).Scan(&t); err != nil {
			hasMsg = false
			msgID = 0
			c.REQUEST.Delete("msg")
			c.POST.Delete("msg")
			c.GET.Delete("msg")
		} else {
			c.Topic = t
		}
	}

	// Check if it's locked.  It isn't locked if no topic is specified.
	locked := 0
	var sticky, pollID, posterID, firstMsgID int
	var firstSubject string
	var lastPostTime int64
	if c.Topic != 0 {
		var notify int
		err := a.DB.QueryRow(a.Q(`
			SELECT
				t.locked, IFNULL(ln.ID_TOPIC, 0) AS notify, t.isSticky, t.ID_POLL, t.numReplies, IFNULL(mf.ID_MEMBER, 0),
				t.ID_FIRST_MSG, IFNULL(mf.subject, ''), MAX(IFNULL(ml.posterTime, 0), IFNULL(ml.modifiedTime, 0)) AS lastPostTime
			FROM {$db_prefix}topics AS t
				LEFT JOIN {$db_prefix}log_notify AS ln ON (ln.ID_TOPIC = t.ID_TOPIC AND ln.ID_MEMBER = ?)
				LEFT JOIN {$db_prefix}messages AS mf ON (mf.ID_MSG = t.ID_FIRST_MSG)
				LEFT JOIN {$db_prefix}messages AS ml ON (ml.ID_MSG = t.ID_LAST_MSG)
			WHERE t.ID_TOPIC = ?
			LIMIT 1`), c.User.ID, c.Topic).Scan(&locked, &notify, &sticky, &pollID, &page.NumReplies,
			&posterID, &firstMsgID, &firstSubject, &lastPostTime)
		if err != nil {
			c.fatalLangError("472", false)
		}

		// If this topic already has a poll, they sure can't add another.
		if hasPoll && pollID > 0 {
			hasPoll = false
		}

		if !hasMsg {
			if c.User.IsGuest && !c.allowedTo("post_reply_any") {
				c.isNotGuest("")
			}
			if posterID != c.User.ID {
				c.isAllowedTo("post_reply_any")
			} else if !c.allowedTo("post_reply_any") {
				c.isAllowedTo("post_reply_own")
			}
		}

		page.CanLock = c.allowedTo("lock_any") || (c.User.ID == posterID && c.allowedTo("lock_own"))
		page.CanSticky = c.allowedTo("make_sticky") && !a.SettingEmpty("enableStickyTopics")

		page.Notify = notify != 0
		if c.REQUEST.Has("sticky") {
			page.Sticky = !empty(c.REQUEST.Str("sticky"))
		} else {
			page.Sticky = sticky != 0
		}
	} else {
		if !hasPoll || a.Setting("pollMode") != "1" {
			c.isAllowedTo("post_new")
		}

		page.CanLock = c.allowedTo("lock_any", "lock_own")
		page.CanSticky = c.allowedTo("make_sticky") && !a.SettingEmpty("enableStickyTopics")
		page.Sticky = !empty(c.REQUEST.Str("sticky"))
	}

	page.CanNotify = c.allowedTo("mark_any_notify")
	page.CanMove = c.allowedTo("move_any")
	page.CanAnnounce = c.allowedTo("announce_topic")
	page.Locked = locked != 0 || !empty(c.REQUEST.Str("lock"))

	// Don't allow a post if it's locked and you aren't all powerful.
	if locked != 0 && !c.allowedTo("moderate_board") {
		c.fatalLangError("90", false)
	}

	// Check the users permissions - is the user allowed to add or post a
	// poll?
	if hasPoll && a.Setting("pollMode") == "1" {
		// New topic, new poll.
		if c.Topic == 0 {
			c.isAllowedTo("poll_post")
		} else if c.User.ID == posterID && !c.allowedTo("poll_add_any") {
			// This is an old topic - but it is yours!  Can you add to it?
			c.isAllowedTo("poll_add_own")
		} else {
			// If you're not the owner, can you add to any poll?
			c.isAllowedTo("poll_add_any")
		}

		// Set up the poll options.
		page.PollMaxVotes = "1"
		if !empty(c.POST.Str("poll_max_votes")) {
			if v := c.POST.Int("poll_max_votes"); v > 1 {
				page.PollMaxVotes = itoa(v)
			}
		}
		if !empty(c.POST.Str("poll_hide")) {
			page.PollHide = c.POST.Int("poll_hide")
		}
		page.PollExpire = c.POST.Str("poll_expire")
		page.PollChangeVote = c.POST.Has("poll_change_vote")

		// Make all five poll choices empty.
		page.Choices = []PostChoice{
			{0, 1, "", false}, {1, 2, "", false}, {2, 3, "", false},
			{3, 4, "", false}, {4, 5, "", true},
		}
	}


	postError := newPostErrSet()
	for _, e := range c.PostErrors {
		postError.add(e)
	}

	newRepliesError := 0
	oldTopicError := false
	topicSummaryPosts := a.SettingInt("topicSummaryPosts")

	// See if any new replies have come along.
	if !hasMsg && c.Topic != 0 {
		if empty(c.Options["no_new_reply_warning"]) && c.REQUEST.Has("num_replies") {
			newReplies := 0
			if page.NumReplies > c.REQUEST.Int("num_replies") {
				newReplies = page.NumReplies - c.REQUEST.Int("num_replies")
			}
			if newReplies != 0 {
				if newReplies == 1 {
					key := "error_new_reply"
					if c.GET.Has("num_replies") {
						key = "error_new_reply_reading"
					}
					postError.msg["new_reply"] = c.Txt(key)
				} else {
					key := "error_new_replies"
					if c.GET.Has("num_replies") {
						key = "error_new_replies_reading"
					}
					postError.msg["new_replies"] = phpSprintf(c.Txt(key), itoa(newReplies))
				}

				// If they've come from the display page then we treat the
				// error differently....
				if c.GET.Has("num_replies") {
					newRepliesError = newReplies
				} else if newReplies == 1 {
					postError.add("new_reply")
				} else {
					postError.add("new_replies")
				}

				if newReplies > topicSummaryPosts && topicSummaryPosts < 5 {
					topicSummaryPosts = 5
				}
			}
		}
		// Check whether this is a really old post being bumped...
		if !a.SettingEmpty("oldTopicDays") && lastPostTime+int64(a.SettingInt("oldTopicDays"))*86400 < time.Now().Unix() &&
			sticky == 0 && !c.REQUEST.Has("subject") {
			oldTopicError = true
		}
	}

	// Get a response prefix (like 'Re:') in the default forum language.
	page.ResponsePrefix = c.Txt("response_prefix")

	var formSubject, formMessage string

	// Previewing, modifying, or posting?
	if c.REQUEST.Has("message") || !postError.empty() {
		reallyPreviewing := false

		// Validate inputs.
		if postError.empty() {
			if Htmltrim(c.REQUEST.Str("subject")) == "" {
				postError.add("no_subject")
			}
			if Htmltrim(c.REQUEST.Str("message")) == "" {
				postError.add("no_message")
			}
			if !a.SettingEmpty("max_messageLength") && entityLen(c.REQUEST.Str("message")) > a.SettingInt("max_messageLength") {
				postError.add("long_message")
			}

			// Are you... a guest?
			if c.User.IsGuest {
				guestname := strings.TrimSpace(c.REQUEST.Str("guestname"))
				email := strings.TrimSpace(c.REQUEST.Str("email"))
				c.REQUEST.Set("guestname", guestname)
				c.REQUEST.Set("email", email)

				// Validate the name and email.
				if strings.TrimSpace(strings.ReplaceAll(guestname, "_", " ")) == "" {
					postError.add("no_name")
				} else if entityLen(guestname) > 25 {
					postError.add("long_name")
				} else if c.isReservedName(Htmlspecialchars(guestname), 0, true, false) {
					postError.add("bad_name")
				}

				if a.SettingEmpty("guest_post_no_email") {
					if email == "" {
						postError.add("no_email")
					} else if !emailRe.MatchString(email) {
						postError.add("bad_email")
					}
				}
			}

			// This is self explanatory - got any questions?
			if c.REQUEST.Has("question") && strings.TrimSpace(c.REQUEST.Str("question")) == "" {
				postError.add("no_question")
			}

			// This means they didn't click Post and get an error.
			reallyPreviewing = true
		}

		// Set up the inputs for the form.
		formSubject = strings.NewReplacer("\r", "", "\n", "", "\t", "").Replace(Htmlspecialchars(c.REQUEST.Str("subject")))
		formMessage = Htmlspecialchars(c.REQUEST.Str("message"))

		// Make sure the subject isn't too long - taking into account special
		// characters.
		if entityLen(formSubject) > 100 {
			formSubject = entitySubstr(formSubject, 0, 100)
		}

		// Have we inadvertently trimmed off the subject of useful
		// information?
		if Htmltrim(formSubject) == "" {
			postError.add("no_subject")
		}

		// Any errors occurred?
		if !postError.empty() {
			c.loadLanguage("Errors")
			page.ErrorType = "minor"
			for _, e := range postError.keys {
				if m, ok := postError.msg[e]; ok {
					page.ErrorMessages = append(page.ErrorMessages, m)
				} else {
					page.ErrorMessages = append(page.ErrorMessages, c.Txt("error_"+e))
				}
				// If it's not a minor error flag it as such.
				if e != "new_reply" && e != "new_replies" && e != "old_topic" {
					page.ErrorType = "serious"
				}
			}
		}

		if hasPoll {
			page.PollQuestion = Htmlspecialchars(strings.TrimSpace(c.REQUEST.Str("question")))

			page.Choices = nil
			choiceID := 0
			if opts := c.POST.Arr("options"); opts != nil {
				for _, k := range opts.Keys() {
					option, ok := opts.Get(k).(string)
					if !ok || strings.TrimSpace(option) == "" {
						continue
					}
					page.Choices = append(page.Choices, PostChoice{choiceID, choiceID + 1, Htmlspecialchars(option), false})
					choiceID++
				}
			}
			if len(page.Choices) < 2 {
				page.Choices = append(page.Choices,
					PostChoice{choiceID, choiceID + 1, "", false},
					PostChoice{choiceID + 1, choiceID + 2, "", false})
				choiceID += 2
			}
			page.Choices[len(page.Choices)-1].IsLast = true
		}

		// Are you... a guest?
		if c.User.IsGuest {
			page.GuestName = Htmlspecialchars(strings.TrimSpace(c.REQUEST.Str("guestname")))
			page.GuestEmail = Htmlspecialchars(strings.TrimSpace(c.REQUEST.Str("email")))
			page.ShowGuestName = true
			c.User.Name = page.GuestName
		}

		// Only show the preview stuff if they hit Preview.
		if reallyPreviewing || c.REQUEST.Has("xml") {
			// Set up the preview message and subject and censor them...
			page.HasPreview = true
			page.PreviewMessage = c.preparsecode(formMessage, false)
			formMessage = c.preparsecode(formMessage, true)

			// Do all bulletin board code tags, with or without smileys.
			page.PreviewMessage = c.parseBBC(page.PreviewMessage, !c.REQUEST.Has("ns"))

			if formSubject != "" {
				page.PreviewSubject = c.censorText(formSubject)
				page.PreviewMessage = c.censorText(page.PreviewMessage)
			} else {
				page.PreviewSubject = "<i>" + c.Txt("24") + "</i>"
			}

			// Protect any CDATA blocks.
			if c.REQUEST.Has("xml") {
				page.PreviewMessage = strings.ReplaceAll(page.PreviewMessage, "]]>", "]]]]><![CDATA[>")
			}
		}

		// Set up the checkboxes.
		page.Notify = !empty(c.REQUEST.Str("notify"))
		page.UseSmileys = !c.REQUEST.Has("ns")

		if c.REQUEST.Has("icon") {
			page.Icon = iconCleanRe.ReplaceAllString(c.REQUEST.Str("icon"), "")
		} else {
			page.Icon = "xx"
		}

		// Set the destination action for submission.
		page.Destination = "post2;start=" + itoa(c.Start)
		if hasMsg {
			page.Destination += ";msg=" + c.REQUEST.Str("msg") + ";sesc=" + c.Sc
		}
		if hasPoll {
			page.Destination += ";poll"
		}
		if hasMsg {
			page.SubmitLabel = c.Txt("10")
		} else {
			page.SubmitLabel = c.Txt("105")
		}

		// Previewing an edit?
		if hasMsg && c.Topic != 0 {
			c.postEditChecks(msgID, page)
		}

		// No check is needed, since nothing is really posted.
		c.checkSubmitOnce("free")
	} else if hasMsg && c.Topic != 0 {
		// Editing a message...
		c.checkSession("get", "", true)

		var rowMember, smileysEnabled, posterStarter int
		var modifiedTime, posterTime int64
		var body, posterName, posterEmail, subject, icon string
		err := a.DB.QueryRow(a.Q(`
			SELECT
				m.ID_MEMBER, m.modifiedTime, m.smileysEnabled, m.body,
				m.posterName, m.posterEmail, m.subject, m.icon,
				t.ID_MEMBER_STARTED, m.posterTime
			FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
			WHERE m.ID_MSG = ?
				AND m.ID_TOPIC = ?
				AND t.ID_TOPIC = ?`), msgID, c.Topic, c.Topic).Scan(
			&rowMember, &modifiedTime, &smileysEnabled, &body,
			&posterName, &posterEmail, &subject, &icon, &posterStarter, &posterTime)
		// The message they were trying to edit was most likely deleted.
		if err != nil {
			c.fatalLangError("smf232", false)
		}

		if rowMember == c.User.ID && !c.allowedTo("modify_any") {
			// Give an extra five minutes over the disable time threshold, so
			// they can type.
			if !a.SettingEmpty("edit_disable_time") && posterTime+int64(a.SettingInt("edit_disable_time")+5)*60 < time.Now().Unix() {
				c.fatalLangError("modify_post_time_passed", false)
			} else if posterStarter == c.User.ID && !c.allowedTo("modify_own") {
				c.isAllowedTo("modify_replies")
			} else {
				c.isAllowedTo("modify_own")
			}
		} else if posterStarter == c.User.ID && !c.allowedTo("modify_any") {
			c.isAllowedTo("modify_replies")
		} else {
			c.isAllowedTo("modify_any")
		}

		// When was it last modified?
		if modifiedTime != 0 {
			page.LastModified = c.timeformat(modifiedTime)
		}

		// Get the stuff ready for the form.
		formSubject = subject
		formMessage = c.censorText(c.unPreparsecode(body))
		formSubject = c.censorText(formSubject)

		// Check the boxes that should be checked.
		page.UseSmileys = smileysEnabled != 0
		page.Icon = icon

		// Load up 'em attachments!
		if !a.SettingEmpty("attachmentEnable") {
			c.loadCurrentAttachments(msgID, page)
		}

		// Allow moderators to change names....
		if c.allowedTo("moderate_forum") && rowMember == 0 {
			page.GuestName = Htmlspecialchars(posterName)
			page.GuestEmail = Htmlspecialchars(posterEmail)
			page.ShowGuestName = true
		}

		// Set the destinaton.
		page.Destination = "post2;start=" + itoa(c.Start) + ";msg=" + c.REQUEST.Str("msg") + ";sesc=" + c.Sc
		if hasPoll {
			page.Destination += ";poll"
		}
		page.SubmitLabel = c.Txt("10")
	} else {
		// Posting...

		// By default....
		page.UseSmileys = true
		page.Icon = "xx"

		if c.User.IsGuest {
			page.GuestName = ""
			page.GuestEmail = ""
			page.ShowGuestName = true
		}
		page.Destination = "post2;start=" + itoa(c.Start)
		if hasPoll {
			page.Destination += ";poll"
		}

		page.SubmitLabel = c.Txt("105")

		// Posting a quoted reply?
		if c.Topic != 0 && !empty(c.REQUEST.Str("quote")) {
			c.checkSession("get", "", true)

			// Make sure they _can_ quote this post, and if so get it.
			quoteID := c.REQUEST.Int("quote")
			var mname string
			var mdate int64
			err := a.DB.QueryRow(a.Q(`
				SELECT m.subject, IFNULL(mem.realName, m.posterName) AS posterName, m.posterTime, m.body
				FROM {$db_prefix}messages AS m, {$db_prefix}boards AS b
					LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
				WHERE m.ID_MSG = ?
					AND b.ID_BOARD = m.ID_BOARD
					AND `+c.User.QuerySeeBoard+`
				LIMIT 1`), quoteID).Scan(&formSubject, &mname, &mdate, &formMessage)
			if err != nil {
				c.fatalLangError("quoted_post_deleted", false)
			}

			// Add 'Re: ' to the front of the quoted subject.
			prefix := strings.TrimSpace(page.ResponsePrefix)
			if prefix != "" && !strings.HasPrefix(formSubject, prefix) {
				formSubject = page.ResponsePrefix + formSubject
			}

			// Censor the message and subject.
			formMessage = c.censorText(formMessage)
			formSubject = c.censorText(formSubject)

			formMessage = brToNlRe.ReplaceAllString(formMessage, "\n")

			// Remove any nested quotes, if necessary.
			if !a.SettingEmpty("removeNestedQuotes") {
				formMessage = nestedQuoteRe.ReplaceAllString(formMessage, "")
				formMessage = strings.TrimPrefix(formMessage, "\n")
				formMessage = strings.ReplaceAll(formMessage, "[/quote]", "")
			}

			// Add a quote string on the front and end.
			formMessage = "[quote author=" + mname + " link=topic=" + itoa(c.Topic) + ".msg" + itoa(quoteID) + "#msg" + itoa(quoteID) + " date=" + i64toa(mdate) + "]\n" + formMessage + "\n[/quote]"
		} else if c.Topic != 0 {
			// Posting a reply without a quote?

			// Get the first message's subject.
			formSubject = firstSubject

			// Add 'Re: ' to the front of the subject.
			prefix := strings.TrimSpace(page.ResponsePrefix)
			if prefix != "" && formSubject != "" && !strings.HasPrefix(formSubject, prefix) {
				formSubject = page.ResponsePrefix + formSubject
			}

			// Censor the subject.
			formSubject = c.censorText(formSubject)

			formMessage = ""
		} else {
			formSubject = c.GET.Str("subject")
			formMessage = ""
		}
	}

	// Handle the temp attachments / uploads for this form.
	deletedAttachments := c.handlePostFormAttachments(page, hasMsg, msgID)

	// If we are coming here to make a reply, and someone has already
	// replied... make a special warning message.
	if newRepliesError != 0 {
		if newRepliesError == 1 {
			page.ErrorMessages = append(page.ErrorMessages, postError.msg["new_reply"])
		} else {
			page.ErrorMessages = append(page.ErrorMessages, postError.msg["new_replies"])
		}
		page.ErrorType = "minor"
	}

	if oldTopicError {
		page.ErrorMessages = append(page.ErrorMessages, c.Txt("error_old_topic"))
		page.ErrorType = "minor"
	}

	page.PostError = postError.m

	// What are you doing?  Posting a poll, modifying, previewing, new post,
	// or reply...
	if hasPoll {
		c.PageTitle = c.Txt("smf20")
	} else if hasMsg {
		c.PageTitle = c.Txt("66")
	} else if c.REQUEST.Has("subject") && page.PreviewSubject != "" {
		c.PageTitle = c.Txt("507") + " - " + stripTags(page.PreviewSubject)
	} else if c.Topic == 0 {
		c.PageTitle = c.Txt("33")
	} else {
		c.PageTitle = c.Txt("25")
	}

	// Build the link tree.
	if c.Topic == 0 {
		c.LinkTree = append(c.LinkTree, Link{Name: "<i>" + c.Txt("33") + "</i>"})
	} else {
		smallClass := ""
		if !c.Theme.Empty("linktree_inline") {
			smallClass = ` class="smalltext"`
		}
		c.LinkTree = append(c.LinkTree, Link{
			URL:         scripturl + "?topic=" + itoa(c.Topic) + "." + itoa(c.Start),
			Name:        formSubject,
			ExtraBefore: "<span" + smallClass + `><b class="nav">` + c.PageTitle + ` ( </b></span>`,
			ExtraAfter:  "<span" + smallClass + `><b class="nav"> )</b></span>`,
		})
	}

	// If they've unchecked an attachment, they may still want to attach that
	// many more files, but don't allow more than num_allowed_attachments.
	page.NumAllowedAttachments = a.SettingInt("attachmentNumPerPostLimit") - len(page.CurrentAttachments) + deletedAttachments
	if page.NumAllowedAttachments > a.SettingInt("attachmentNumPerPostLimit") {
		page.NumAllowedAttachments = a.SettingInt("attachmentNumPerPostLimit")
	}
	page.CanPostAttachment = !a.SettingEmpty("attachmentEnable") && a.Setting("attachmentEnable") == "1" &&
		c.allowedTo("post_attachment") && page.NumAllowedAttachments > 0

	page.Subject = strings.ReplaceAll(formSubject, `"`, `\"`)
	page.Message = strings.NewReplacer(`"`, "&quot;", "<", "&lt;", ">", "&gt;", "  ", " &nbsp;").Replace(formMessage)
	page.AllowedExtensions = strings.ReplaceAll(a.Setting("attachmentExtensions"), ",", ", ")
	page.MakePoll = hasPoll

	// Message icons - customized icons are off?
	if a.SettingEmpty("messageIcons_enable") {
		type iconDef struct{ value, name string }
		for _, ic := range []iconDef{
			{"xx", c.Txt("281")}, {"thumbup", c.Txt("282")}, {"thumbdown", c.Txt("283")},
			{"exclamation", c.Txt("284")}, {"question", c.Txt("285")}, {"lamp", c.Txt("286")},
			{"smiley", c.Txt("287")}, {"angry", c.Txt("288")}, {"cheesy", c.Txt("289")},
			{"grin", c.Txt("293")}, {"sad", c.Txt("291")}, {"wink", c.Txt("292")},
		} {
			page.Icons = append(page.Icons, PostIcon{
				Value: ic.value, Name: ic.name,
				URL: c.Theme.ImagesURL() + "/post/" + ic.value + ".gif",
			})
		}
		page.IconURL = c.Theme.ImagesURL() + "/post/" + page.Icon + ".gif"
	} else {
		// Otherwise load the icons, and check we give the right image too...
		iconSources := map[string]string{}
		for _, ic := range []string{"xx", "thumbup", "thumbdown", "exclamation", "question", "lamp",
			"smiley", "angry", "cheesy", "grin", "sad", "wink", "moved", "recycled", "wireless"} {
			iconSources[ic] = "images_url"
		}
		rows, err := a.DB.Query(a.Q(`
			SELECT title, filename
			FROM {$db_prefix}message_icons
			WHERE ID_BOARD IN (0, ?)`), c.Board)
		if err == nil {
			for rows.Next() {
				var title, filename string
				rows.Scan(&title, &filename)
				if _, ok := iconSources[filename]; !ok {
					if _, statErr := os.Stat(c.Theme.Get("theme_dir") + "/images/post/" + filename + ".gif"); statErr == nil {
						iconSources[filename] = "images_url"
					} else {
						iconSources[filename] = "default_images_url"
					}
				}
				page.Icons = append(page.Icons, PostIcon{
					Value: filename, Name: title,
					URL: c.Theme.Get(iconSources[filename]) + "/post/" + filename + ".gif",
				})
			}
			rows.Close()
		}
		src := "images_url"
		if s, ok := iconSources[page.Icon]; ok {
			src = s
		}
		page.IconURL = c.Theme.Get(src) + "/post/" + page.Icon + ".gif"
	}

	if len(page.Icons) > 0 {
		page.Icons[len(page.Icons)-1].IsLast = true
	}

	found := false
	for i := range page.Icons {
		page.Icons[i].Selected = page.Icon == page.Icons[i].Value
		if page.Icons[i].Selected {
			found = true
		}
	}
	if !found {
		page.Icons = append([]PostIcon{{
			Value: page.Icon, Name: c.Txt("current_icon"),
			URL: page.IconURL, IsLast: len(page.Icons) == 0, Selected: true,
		}}, page.Icons...)
	}

	if c.Topic != 0 {
		c.getTopicSummary(page, topicSummaryPosts)
	}

	page.BackToTopic = c.REQUEST.Has("goback") || (hasMsg && !c.REQUEST.Has("subject"))
	page.ShowAdditionalOptions = !empty(c.POST.Str("additional_options")) ||
		len(c.tempAttachments()) > 0 || deletedAttachments > 0

	page.IsNewTopic = c.Topic == 0
	page.IsNewPost = !hasMsg
	page.IsFirstPost = page.IsNewTopic || (hasMsg && msgID == firstMsgID)

	// Register this form in the session variables.
	c.checkSubmitOnce("register")

	// Finally, load the template.
	if c.REQUEST.Has("xml") {
		c.SubTemplate = templateXMLPost
	} else {
		c.SubTemplate = templatePostMain
	}
}

// postEditChecks re-runs edit permission checks when previewing an edit
// (Post.php lines 563-641).
func (c *Ctx) postEditChecks(msgID int, page *PostCtx) {
	a := c.App
	var rowMember, posterStarter int
	var posterTime int64
	err := a.DB.QueryRow(a.Q(`
		SELECT m.ID_MEMBER, t.ID_MEMBER_STARTED, m.posterTime
		FROM {$db_prefix}messages AS m, {$db_prefix}topics AS t
		WHERE m.ID_MSG = ?
			AND m.ID_TOPIC = ?
			AND t.ID_TOPIC = ?`), msgID, c.Topic, c.Topic).Scan(&rowMember, &posterStarter, &posterTime)
	if err != nil {
		c.fatalLangError("smf232", false)
	}

	if rowMember == c.User.ID && !c.allowedTo("modify_any") {
		// Give an extra five minutes over the disable time threshold, so
		// they can type.
		if !a.SettingEmpty("edit_disable_time") && posterTime+int64(a.SettingInt("edit_disable_time")+5)*60 < time.Now().Unix() {
			c.fatalLangError("modify_post_time_passed", false)
		} else if posterStarter == c.User.ID && !c.allowedTo("modify_own") {
			c.isAllowedTo("modify_replies")
		} else {
			c.isAllowedTo("modify_own")
		}
	} else if posterStarter == c.User.ID && !c.allowedTo("modify_any") {
		c.isAllowedTo("modify_replies")
	} else {
		c.isAllowedTo("modify_any")
	}

	if !a.SettingEmpty("attachmentEnable") {
		c.loadCurrentAttachments(msgID, page)
	}

	// Allow moderators to change names....
	if c.allowedTo("moderate_forum") && c.Topic != 0 {
		var idMember int
		var posterName, posterEmail string
		err := a.DB.QueryRow(a.Q(`
			SELECT ID_MEMBER, posterName, posterEmail
			FROM {$db_prefix}messages
			WHERE ID_MSG = ?
				AND ID_TOPIC = ?
			LIMIT 1`), msgID, c.Topic).Scan(&idMember, &posterName, &posterEmail)
		if err == nil && idMember == 0 {
			page.GuestName = Htmlspecialchars(posterName)
			page.GuestEmail = Htmlspecialchars(posterEmail)
			page.ShowGuestName = true
		}
	}
}

// loadCurrentAttachments lists a message's attachments for the form.
func (c *Ctx) loadCurrentAttachments(msgID int, page *PostCtx) {
	a := c.App
	rows, err := a.DB.Query(a.Q(`
		SELECT IFNULL(size, -1) AS filesize, filename, ID_ATTACH
		FROM {$db_prefix}attachments
		WHERE ID_MSG = ?
			 AND attachmentType = 0`), msgID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var size, idAttach int
		var filename string
		rows.Scan(&size, &filename, &idAttach)
		if size <= 0 {
			continue
		}
		page.CurrentAttachments = append(page.CurrentAttachments, PostAttachment{
			Name: Htmlspecialchars(filename),
			ID:   itoa(idAttach),
		})
	}
}

// getTopicSummary is getTopic() from Post.php.
func (c *Ctx) getTopicSummary(page *PostCtx, topicSummaryPosts int) {
	a := c.App

	// Calculate the amount of new replies.
	newReplies := 0
	if !empty(c.REQUEST.Str("num_replies")) && page.NumReplies > c.REQUEST.Int("num_replies") {
		newReplies = page.NumReplies - c.REQUEST.Int("num_replies")
	}

	limit := ""
	if c.REQUEST.Has("xml") {
		limit = `
		LIMIT ` + itoa(newReplies)
	} else if topicSummaryPosts != 0 {
		limit = `
		LIMIT ` + itoa(topicSummaryPosts)
	}

	msgCond := ""
	if c.REQUEST.Has("msg") {
		msgCond = `
			AND m.ID_MSG < ` + itoa(c.REQUEST.Int("msg"))
	}

	// If you're modifying, get only those posts before the current one.
	// (otherwise get all.)
	rows, err := a.DB.Query(a.Q(`
		SELECT IFNULL(mem.realName, m.posterName) AS posterName, m.posterTime, m.body, m.smileysEnabled, m.ID_MSG
		FROM {$db_prefix}messages AS m
			LEFT JOIN {$db_prefix}members AS mem ON (mem.ID_MEMBER = m.ID_MEMBER)
		WHERE m.ID_TOPIC = ` + itoa(c.Topic) + msgCond + `
		ORDER BY m.ID_MSG DESC` + limit))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var posterName, body string
		var posterTime int64
		var smileysEnabled, idMsg int
		rows.Scan(&posterName, &posterTime, &body, &smileysEnabled, &idMsg)

		// Censor, BBC, ...
		body = c.censorText(body)
		body = c.parseBBCCached(body, smileysEnabled != 0, itoa(idMsg))

		// ...and store.
		page.PreviousPosts = append(page.PreviousPosts, PostPrevPost{
			Poster:  posterName,
			Message: body,
			Time:    c.timeformat(posterTime),
			ID:      idMsg,
			IsNew:   newReplies != 0,
		})
		if newReplies != 0 {
			newReplies--
		}
	}
}

// tempAttachments returns the session's temp_attachments map (attachID ->
// original name), in insertion order as []tempAttachment.
type tempAttachment struct {
	ID   string
	Name string
}

func (c *Ctx) tempAttachments() []tempAttachment {
	raw, _ := c.Session.Get("temp_attachments").([]any)
	var out []tempAttachment
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			id, _ := m["id"].(string)
			name, _ := m["name"].(string)
			out = append(out, tempAttachment{id, name})
		}
	}
	return out
}

func (c *Ctx) setTempAttachments(list []tempAttachment) {
	arr := make([]any, len(list))
	for i, t := range list {
		arr[i] = map[string]any{"id": t.ID, "name": t.Name}
	}
	c.Session.Set("temp_attachments", arr)
}

// handlePostFormAttachments is the allowedTo('post_attachment') block of
// Post() (Post.php lines 799-935): collect temp attachments and accept new
// uploads into post_tmp_* files. Returns the number of "unchecked" deleted
// attachments.
func (c *Ctx) handlePostFormAttachments(page *PostCtx, hasMsg bool, msgID int) int {
	a := c.App

	if !c.allowedTo("post_attachment") {
		return 0
	}

	temp := c.tempAttachments()

	// If this isn't a new post, check the current attachments.
	quantity := 0
	totalSize := 0
	if hasMsg {
		a.DB.QueryRow(a.Q(`
			SELECT COUNT(*), IFNULL(SUM(size), 0)
			FROM {$db_prefix}attachments
			WHERE ID_MSG = ?
				AND attachmentType = 0`), msgID).Scan(&quantity, &totalSize)
	}

	tempStart := 0
	deletedAttachments := 0
	hadDeleted := false

	attachDel := map[string]bool{}
	attachDelSent := false
	if del := c.POST.Arr("attach_del"); del != nil {
		attachDelSent = del.Len() > 0
		for _, k := range del.Keys() {
			attachDel[del.Str(k)] = true
		}
	}

	myTmpRe := regexp.MustCompile(`^post_tmp_` + itoa(c.User.ID) + `_\d+$`)

	var kept []tempAttachment
	for _, t := range temp {
		tempStart++

		if !myTmpRe.MatchString(t.ID) {
			continue
		}

		if attachDelSent && !attachDel[t.ID] {
			hadDeleted = true
			os.Remove(a.Setting("attachmentUploadDir") + "/" + t.ID)
			continue
		}

		quantity++
		if info, err := os.Stat(a.Setting("attachmentUploadDir") + "/" + t.ID); err == nil {
			totalSize += int(info.Size())
		}

		kept = append(kept, t)
		page.CurrentAttachments = append(page.CurrentAttachments, PostAttachment{
			Name: t.Name,
			ID:   t.ID,
		})
	}
	temp = kept
	c.setTempAttachments(temp)

	if attachDelSent {
		for k := range page.CurrentAttachments {
			if !attachDel[page.CurrentAttachments[k].ID] {
				page.CurrentAttachments[k].Unchecked = true
				deletedAttachments++
				quantity--
			}
		}
	}

	if c.R.MultipartForm != nil {
		for _, fh := range c.R.MultipartForm.File["attachment[]"] {
			if fh.Filename == "" {
				continue
			}

			if !a.SettingEmpty("attachmentSizeLimit") && int(fh.Size) > a.SettingInt("attachmentSizeLimit")*1024 {
				c.fatalLangError("smf122", false, a.Setting("attachmentSizeLimit"))
			}

			quantity++
			if !a.SettingEmpty("attachmentNumPerPostLimit") && quantity > a.SettingInt("attachmentNumPerPostLimit") {
				c.fatalLangError("attachments_limit_per_post", false, a.Setting("attachmentNumPerPostLimit"))
			}

			totalSize += int(fh.Size)
			if !a.SettingEmpty("attachmentPostLimit") && totalSize > a.SettingInt("attachmentPostLimit")*1024 {
				c.fatalLangError("smf122", false, a.Setting("attachmentPostLimit"))
			}

			if !a.SettingEmpty("attachmentCheckExtensions") {
				ext := ""
				if i := strings.LastIndexByte(fh.Filename, '.'); i >= 0 {
					ext = strings.ToLower(fh.Filename[i+1:])
				}
				ok := false
				for _, allowed := range strings.Split(strings.ToLower(a.Setting("attachmentExtensions")), ",") {
					if allowed == ext {
						ok = true
						break
					}
				}
				if !ok {
					c.fatalError(fh.Filename+".<br />"+c.Txt("smf123")+" "+a.Setting("attachmentExtensions")+".", false)
				}
			}

			if !a.SettingEmpty("attachmentDirSizeLimit") {
				// Make sure the directory isn't full.
				dirSize := 0
				entries, err := os.ReadDir(a.Setting("attachmentUploadDir"))
				if err != nil {
					c.fatalLangError("smf115b", true)
				}
				for _, e := range entries {
					if postTmpAnyRe.MatchString(e.Name()) {
						// Temp file is more than 5 hours old!
						if info, err := e.Info(); err == nil && info.ModTime().Unix() < time.Now().Unix()-18000 {
							os.Remove(a.Setting("attachmentUploadDir") + "/" + e.Name())
						}
						continue
					}
					if info, err := e.Info(); err == nil {
						dirSize += int(info.Size())
					}
				}

				// Too big!  Maybe you could zip it or something...
				if int(fh.Size)+dirSize > a.SettingInt("attachmentDirSizeLimit")*1024 {
					c.fatalLangError("smf126", true)
				}
			}

			attachID := "post_tmp_" + itoa(c.User.ID) + "_" + itoa(tempStart)
			tempStart++
			temp = append(temp, tempAttachment{attachID, filepathBase(fh.Filename)})
			page.CurrentAttachments = append(page.CurrentAttachments, PostAttachment{
				Name: filepathBase(fh.Filename),
				ID:   attachID,
			})

			destName := a.Setting("attachmentUploadDir") + "/" + attachID

			src, err := fh.Open()
			if err != nil {
				c.fatalLangError("smf124", true)
			}
			dst, err := os.Create(destName)
			if err != nil {
				src.Close()
				c.fatalLangError("attachments_no_write", true)
			}
			if _, err := io.Copy(dst, src); err != nil {
				dst.Close()
				src.Close()
				c.fatalLangError("smf124", true)
			}
			dst.Close()
			src.Close()
			os.Chmod(destName, 0644)
		}
		c.setTempAttachments(temp)
	}

	if hadDeleted && deletedAttachments == 0 {
		deletedAttachments = 1
	}
	return deletedAttachments
}
