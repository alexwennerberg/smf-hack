package app

// Port of Sources/Poll.php: Vote, LockVoting, EditPoll, EditPoll2,
// RemovePoll.

import (
	"math"
	"strings"
	"time"
)

func init() {
	registerAction("vote", (*Ctx).Vote)
	registerAction("lockVoting", (*Ctx).LockVoting)
	registerAction("editpoll", (*Ctx).EditPoll)
	registerAction("editpoll2", (*Ctx).EditPoll2)
	registerAction("removepoll", (*Ctx).RemovePoll)
}

// Vote is Vote(): register a vote in a poll.
func (c *Ctx) Vote() {
	a := c.App

	// Make sure you can vote.
	c.isAllowedTo("poll_vote")

	// Even with poll_vote permission we would never be able to register you.
	if c.User.IsGuest {
		c.fatalLangError("cannot_poll_vote", true)
	}

	c.loadLanguage("Post")

	// Check if they have already voted, or voting is locked.
	var selected, votingLocked, pollID, maxVotes, changeVote int
	var expireTime int64
	err := a.DB.QueryRow(a.Q(`
		SELECT IFNULL(lp.ID_CHOICE, -1) AS selected, p.votingLocked, p.ID_POLL, p.expireTime, p.maxVotes, p.changeVote
		FROM {$db_prefix}polls AS p, {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}log_polls AS lp ON (p.ID_POLL = lp.ID_POLL AND lp.ID_MEMBER = ?)
		WHERE p.ID_POLL = t.ID_POLL
			AND t.ID_TOPIC = ?
		LIMIT 1`), c.User.ID, c.Topic).Scan(&selected, &votingLocked, &pollID, &expireTime, &maxVotes, &changeVote)
	if err != nil {
		c.fatalLangError("smf27", false)
	}

	// Is voting locked or has it expired?
	if votingLocked != 0 || (expireTime != 0 && time.Now().Unix() > expireTime) {
		c.fatalLangError("smf27", false)
	}

	// If they have already voted and aren't allowed to change their vote -
	// hence they are outta here!
	if selected != -1 && changeVote == 0 {
		c.fatalLangError("smf27", false)
	} else if changeVote != 0 {
		// Otherwise if they can change their vote yet they haven't sent any
		// options... remove their vote and redirect.
		c.checkSession("request", "", true)
		var pollOptions []string

		// Find out what they voted for before.
		rows, err := a.DB.Query(a.Q(`
			SELECT ID_CHOICE
			FROM {$db_prefix}log_polls
			WHERE ID_MEMBER = ?
				AND ID_POLL = ?`), c.User.ID, pollID)
		if err == nil {
			for rows.Next() {
				var choice int
				rows.Scan(&choice)
				pollOptions = append(pollOptions, itoa(choice))
			}
			rows.Close()
		}

		// Just skip it if they had voted for nothing before.
		if len(pollOptions) > 0 {
			// Update the poll totals.
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}poll_choices
				SET votes = votes - 1
				WHERE ID_POLL = ` + itoa(pollID) + `
					AND ID_CHOICE IN (` + strings.Join(pollOptions, ", ") + `)
					AND votes > 0`))

			// Delete off the log.
			a.DB.Exec(a.Q(`
				DELETE FROM {$db_prefix}log_polls
				WHERE ID_MEMBER = ?
					AND ID_POLL = ?`), c.User.ID, pollID)
		}

		// Redirect back to the topic so the user can vote again!
		if c.POST.Arr("options") == nil {
			c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
		}
	}

	// Make sure the option(s) are valid.
	opts := c.POST.Arr("options")
	if opts == nil || opts.Len() == 0 {
		c.fatalLangError("smf26", false)
	}

	// Too many options checked!
	if opts.Len() > maxVotes {
		c.fatalLangError("poll_error1", false, itoa(maxVotes))
	}

	var pollOptions []string
	var values []string
	for _, k := range opts.Keys() {
		id := 0
		if s, ok := opts.Get(k).(string); ok {
			id = atoi(s)
		}
		pollOptions = append(pollOptions, itoa(id))
		values = append(values, "("+itoa(pollID)+", "+itoa(c.User.ID)+", "+itoa(id)+")")
	}

	// Add their vote to the tally.
	res, err := a.DB.Exec(a.Q(`
		INSERT OR IGNORE INTO {$db_prefix}log_polls
			(ID_POLL, ID_MEMBER, ID_CHOICE)
		VALUES ` + strings.Join(values, ",")))
	if err == nil {
		if n, _ := res.RowsAffected(); n != 0 {
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}poll_choices
				SET votes = votes + 1
				WHERE ID_POLL = ` + itoa(pollID) + `
					AND ID_CHOICE IN (` + strings.Join(pollOptions, ", ") + `)`))
		}
	}

	// Return to the post...
	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}

// LockVoting is LockVoting(): lock or unlock voting on a poll.
func (c *Ctx) LockVoting() {
	a := c.App

	c.checkSession("get", "", true)

	// Get the poll starter, ID, and whether or not it is locked.
	var memberID, pollID, votingLocked int
	a.DB.QueryRow(a.Q(`
		SELECT t.ID_MEMBER_STARTED, t.ID_POLL, p.votingLocked
		FROM {$db_prefix}topics AS t, {$db_prefix}polls AS p
		WHERE t.ID_TOPIC = ?
			AND p.ID_POLL = t.ID_POLL
		LIMIT 1`), c.Topic).Scan(&memberID, &pollID, &votingLocked)

	// If the user _can_ modify the poll....
	if !c.allowedTo("poll_lock_any") {
		if c.User.ID == memberID {
			c.isAllowedTo("poll_lock_own")
		} else {
			c.isAllowedTo("poll_lock_any")
		}
	}

	if votingLocked == 1 {
		// It's been locked by a non-moderator.
		votingLocked = 0
	} else if votingLocked == 2 && c.allowedTo("moderate_board") {
		// Locked by a moderator, and this is a moderator.
		votingLocked = 0
	} else if votingLocked == 2 {
		// Sorry, a moderator locked it.
		c.fatalLangError("smf31", true)
	} else if votingLocked == 0 && c.allowedTo("moderate_board") {
		// A moderator *is* locking it.
		votingLocked = 2
	} else {
		// Well, it's gonna be locked one way or another otherwise...
		votingLocked = 1
	}

	// Lock!  *Poof* - no one can vote.
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}polls
		SET votingLocked = ?
		WHERE ID_POLL = ?`), votingLocked, pollID)

	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}

// EditPollChoice is one choice row on the poll edit form.
type EditPollChoice struct {
	ID     int
	Number int
	Votes  int
	Label  string
	IsLast bool
}

// EditPollCtx is the page context for the Poll template.
type EditPollCtx struct {
	CanModeratePoll bool
	Start           int
	IsEdit          bool
	PollID          int
	Question        string
	HideResults     int
	MaxVotes        string
	ChangeVote      bool
	Expiration      string
	Choices         []EditPollChoice
	PollError       map[string]bool
	ErrorMessages   []string
}

// EditPoll is EditPoll(): display the screen for editing or adding a poll.
func (c *Ctx) EditPoll() {
	a := c.App

	if c.Topic == 0 {
		c.fatalLangError("1", false)
	}

	c.loadLanguage("Post")

	page, ok := c.Page.(*EditPollCtx)
	if !ok {
		page = &EditPollCtx{PollError: map[string]bool{}}
		c.Page = page
	}

	page.CanModeratePoll = c.allowedTo("moderate_board")
	page.Start = c.REQUEST.Int("start")
	page.IsEdit = !c.REQUEST.Has("add")

	// Check if a poll currently exists on this topic, and get the id,
	// question and starter.
	var topicStarter, pollID, hideResults, maxVotes, changeVote, pollStarter int
	var question string
	var expireTime int64
	err := a.DB.QueryRow(a.Q(`
		SELECT
			t.ID_MEMBER_STARTED, IFNULL(p.ID_POLL, 0), IFNULL(p.question, ''), IFNULL(p.hideResults, 0),
			IFNULL(p.expireTime, 0), IFNULL(p.maxVotes, 0), IFNULL(p.changeVote, 0), IFNULL(p.ID_MEMBER, 0)
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}polls AS p ON (p.ID_POLL = t.ID_POLL)
		WHERE t.ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&topicStarter, &pollID, &question, &hideResults,
		&expireTime, &maxVotes, &changeVote, &pollStarter)

	// Assume the the topic exists, right?
	if err != nil {
		c.fatalLangError("smf232", true)
	}

	// If we are adding a new poll - make sure that there isn't already a
	// poll there.
	if !page.IsEdit && pollID != 0 {
		c.fatalLangError("poll_already_exists", true)
	} else if page.IsEdit && pollID == 0 {
		// Otherwise, if we're editing it, it does exist I assume?
		c.fatalLangError("poll_not_found", true)
	}

	// Can you do this?
	if page.IsEdit && !c.allowedTo("poll_edit_any") {
		if c.User.ID == topicStarter || (pollStarter != 0 && c.User.ID == pollStarter) {
			c.isAllowedTo("poll_edit_own")
		} else {
			c.isAllowedTo("poll_edit_any")
		}
	} else if !page.IsEdit && !c.allowedTo("poll_add_any") {
		if c.User.ID == topicStarter {
			c.isAllowedTo("poll_add_own")
		} else {
			c.isAllowedTo("poll_add_any")
		}
	}

	// Want to make sure before you actually submit?  Must be a lot of
	// options, or something.
	if c.POST.Has("preview") {
		page.PollID = pollID
		page.Question = Htmlspecialchars(c.POST.Str("question"))
		page.HideResults = c.POST.Int("poll_hide")
		page.ChangeVote = c.POST.Has("poll_change_vote")
		page.MaxVotes = "1"
		if !empty(c.POST.Str("poll_max_votes")) && c.POST.Int("poll_max_votes") > 1 {
			page.MaxVotes = itoa(c.POST.Int("poll_max_votes"))
		}

		// Start at number one with no last id to speak of.
		number := 1
		lastID := 0

		postOpts := c.POST.Arr("options")

		page.Choices = nil
		choiceIdx := map[int]int{} // ID_CHOICE -> index in page.Choices

		// Get all the choices - if this is an edit.
		if page.IsEdit {
			rows, err := a.DB.Query(a.Q(`
				SELECT label, votes, ID_CHOICE
				FROM {$db_prefix}poll_choices
				WHERE ID_POLL = ?`), pollID)
			if err == nil {
				for rows.Next() {
					var label string
					var votes, idChoice int
					rows.Scan(&label, &votes, &idChoice)

					// Get the highest id so we can add more without reusing.
					if idChoice >= lastID {
						lastID = idChoice + 1
					}

					// They cleared this by either omitting it or emptying it.
					if postOpts == nil || !postOpts.Has(itoa(idChoice)) || postOpts.Str(itoa(idChoice)) == "" {
						continue
					}

					label = c.censorText(label)

					// Add the choice!
					choiceIdx[idChoice] = len(page.Choices)
					page.Choices = append(page.Choices, EditPollChoice{
						ID: idChoice, Number: number, Votes: votes, Label: label,
					})
					number++
				}
				rows.Close()
			}
		}

		// Work out how many options we have, so we get the 'is_last' field
		// right...
		totalPostOptions := 0
		if postOpts != nil {
			for _, k := range postOpts.Keys() {
				if postOpts.Str(k) != "" {
					totalPostOptions++
				}
			}
		}

		count := 1
		// If an option exists, update it.  If it is new, add it - but don't
		// reuse ids!
		if postOpts != nil {
			for _, k := range postOpts.Keys() {
				label := c.censorText(Htmlspecialchars(postOpts.Str(k)))

				if idx, ok := choiceIdx[atoi(k)]; ok {
					page.Choices[idx].Label = label
				} else if label != "" {
					page.Choices = append(page.Choices, EditPollChoice{
						ID: lastID, Number: number, Label: label, Votes: -1,
						IsLast: count == totalPostOptions && totalPostOptions > 1,
					})
					lastID++
					number++
					count++
				}
			}
		}

		var pollErrors []string

		// Make sure we have two choices for sure!
		if totalPostOptions < 2 {
			// Need two?
			if totalPostOptions == 0 {
				page.Choices = append(page.Choices, EditPollChoice{
					ID: lastID, Number: number, Label: "", Votes: -1,
				})
				lastID++
				number++
			}
			pollErrors = append(pollErrors, "poll_few")
		}

		// Always show one extra box...
		page.Choices = append(page.Choices, EditPollChoice{
			ID: lastID, Number: number, Label: "", Votes: -1, IsLast: true,
		})
		lastID++
		number++

		if c.allowedTo("moderate_board") {
			page.Expiration = c.POST.Str("poll_expire")
		}

		// Check the question/option count for errors.
		if strings.TrimSpace(c.POST.Str("question")) == "" && len(page.PollError) == 0 {
			pollErrors = append(pollErrors, "no_question")
		}

		// No check is needed, since nothing is really posted.
		c.checkSubmitOnce("free")

		// Take a check for any errors... assuming we haven't already done
		// so!
		if len(pollErrors) > 0 && len(page.PollError) == 0 {
			c.loadLanguage("Errors")
			for _, e := range pollErrors {
				page.PollError[e] = true
				page.ErrorMessages = append(page.ErrorMessages, c.Txt("error_"+e))
			}
		}
	} else {
		// Basic theme info...
		page.PollID = pollID
		page.Question = question
		page.HideResults = hideResults
		page.MaxVotes = itoa(maxVotes)
		page.ChangeVote = changeVote != 0

		// Poll expiration time?
		if expireTime != 0 && c.allowedTo("moderate_board") {
			days := -1.0
			if expireTime > time.Now().Unix() {
				days = float64(expireTime-time.Now().Unix()) / (3600 * 24)
			}
			page.Expiration = itoa(int(math.Ceil(days)))
		}

		// Get all the choices - if this is an edit.
		if page.IsEdit {
			rows, err := a.DB.Query(a.Q(`
				SELECT label, votes, ID_CHOICE
				FROM {$db_prefix}poll_choices
				WHERE ID_POLL = ?`), pollID)
			number := 1
			lastID := 0
			if err == nil {
				for rows.Next() {
					var label string
					var votes, idChoice int
					rows.Scan(&label, &votes, &idChoice)
					label = c.censorText(label)
					page.Choices = append(page.Choices, EditPollChoice{
						ID: idChoice, Number: number, Votes: votes, Label: label,
					})
					number++
					if idChoice >= lastID {
						lastID = idChoice + 1
					}
				}
				rows.Close()
			}

			// Add an extra choice...
			page.Choices = append(page.Choices, EditPollChoice{
				ID: lastID, Number: number, Votes: -1, Label: "", IsLast: true,
			})
		} else {
			// New poll? Setup the default poll options.
			page.PollID = 0
			page.Question = ""
			page.HideResults = 0
			page.MaxVotes = "1"
			page.ChangeVote = false
			page.Expiration = ""

			// Make all five poll choices empty.
			page.Choices = []EditPollChoice{
				{0, 1, -1, "", false}, {1, 2, -1, "", false}, {2, 3, -1, "", false},
				{3, 4, -1, "", false}, {4, 5, -1, "", true},
			}
		}
	}
	if page.IsEdit {
		c.PageTitle = c.Txt("smf39")
	} else {
		c.PageTitle = c.Txt("add_poll")
	}

	// Build the link tree.
	c.LinkTree = append(c.LinkTree, Link{Name: c.PageTitle})

	// Register this form in the session variables.
	c.checkSubmitOnce("register")

	c.SubTemplate = templatePollMain
}

// EditPoll2 is EditPoll2(): update the settings for a poll, or add a new
// one.
func (c *Ctx) EditPoll2() {
	a := c.App

	var pollErrors []string
	if c.checkSession("post", "", false) != "" {
		pollErrors = append(pollErrors, "session_timeout")
	}

	if c.POST.Has("preview") {
		c.EditPoll()
		return
	}

	// HACKERS (!!) can't edit :P.
	if c.Topic == 0 {
		c.fatalLangError("1", false)
	}

	// Is this a new poll, or editing an existing?
	isEdit := !c.REQUEST.Has("add")

	// Get the starter and the poll's ID - if it's an edit.
	var topicStarter, pollID, pollStarter int
	err := a.DB.QueryRow(a.Q(`
		SELECT t.ID_MEMBER_STARTED, IFNULL(t.ID_POLL, 0), IFNULL(p.ID_MEMBER, 0)
		FROM {$db_prefix}topics AS t
			LEFT JOIN {$db_prefix}polls AS p ON (p.ID_POLL = t.ID_POLL)
		WHERE t.ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&topicStarter, &pollID, &pollStarter)
	if err != nil {
		c.fatalLangError("smf232", true)
	}

	// Check their adding/editing is valid.
	if !isEdit && pollID != 0 {
		c.fatalLangError("poll_already_exists", true)
	} else if isEdit && pollID == 0 {
		// Are we editing a poll which doesn't exist?
		c.fatalLangError("poll_not_found", true)
	}

	// Check if they have the power to add or edit the poll.
	if isEdit && !c.allowedTo("poll_edit_any") {
		if c.User.ID == topicStarter || (pollStarter != 0 && c.User.ID == pollStarter) {
			c.isAllowedTo("poll_edit_own")
		} else {
			c.isAllowedTo("poll_edit_any")
		}
	} else if !isEdit && !c.allowedTo("poll_add_any") {
		if c.User.ID == topicStarter {
			c.isAllowedTo("poll_add_own")
		} else {
			c.isAllowedTo("poll_add_any")
		}
	}

	optionCount := 0
	// Ensure the user is leaving a valid amount of options - there must be
	// at least two.
	opts := c.POST.Arr("options")
	if opts != nil {
		for _, k := range opts.Keys() {
			if strings.TrimSpace(opts.Str(k)) != "" {
				optionCount++
			}
		}
	}
	if optionCount < 2 {
		pollErrors = append(pollErrors, "poll_few")
	}

	// Also - ensure they are not removing the question.
	if strings.TrimSpace(c.POST.Str("question")) == "" {
		pollErrors = append(pollErrors, "no_question")
	}

	// Got any errors to report?
	if len(pollErrors) > 0 {
		c.loadLanguage("Errors")
		// Previewing.
		c.POST.Set("preview", "1")

		page := &EditPollCtx{PollError: map[string]bool{}}
		for _, e := range pollErrors {
			page.PollError[e] = true
			page.ErrorMessages = append(page.ErrorMessages, c.Txt("error_"+e))
		}
		c.Page = page

		c.EditPoll()
		return
	}

	// Prevent double submission of this form.
	c.checkSubmitOnce("check")

	// Now we've done all our error checking, let's get the core poll
	// information cleaned... question first.
	question := Htmlspecialchars(c.POST.Str("question"))

	pollHide := c.POST.Int("poll_hide")
	pollChangeVote := 0
	if c.POST.Has("poll_change_vote") {
		pollChangeVote = 1
	}
	pollExpire := int64(0)
	pollMaxVotes := 1

	// Ensure that the number options allowed makes sense, and the expiration
	// date is valid.
	if !isEdit || c.allowedTo("moderate_board") {
		if empty(c.POST.Str("poll_expire")) && pollHide == 2 {
			pollHide = 1
		} else if !empty(c.POST.Str("poll_expire")) {
			pollExpire = time.Now().Unix() + int64(c.POST.Int("poll_expire"))*3600*24
		}

		if !empty(c.POST.Str("poll_max_votes")) && c.POST.Int("poll_max_votes") > 0 {
			pollMaxVotes = c.POST.Int("poll_max_votes")
		}
	}

	// If we're editing, let's commit the changes.
	if isEdit {
		if c.allowedTo("moderate_board") {
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}polls
				SET question = ?, changeVote = ?,
					hideResults = ?, expireTime = ?, maxVotes = ?
				WHERE ID_POLL = ?`), question, pollChangeVote, pollHide, pollExpire, pollMaxVotes, pollID)
		} else {
			a.DB.Exec(a.Q(`
				UPDATE {$db_prefix}polls
				SET question = ?, changeVote = ?,
					hideResults = IIF(expireTime = 0 AND ? = 2, 1, ?)
				WHERE ID_POLL = ?`), question, pollChangeVote, pollHide, pollHide, pollID)
		}
	} else {
		// Otherwise, let's get our poll going! Create the poll.
		res, err := a.DB.Exec(a.Q(`
			INSERT INTO {$db_prefix}polls
				(question, hideResults, maxVotes, expireTime, ID_MEMBER, posterName, changeVote)
			VALUES (SUBSTR(?, 1, 255), ?, ?, ?, ?, SUBSTR(?, 1, 255), ?)`),
			question, pollHide, pollMaxVotes, pollExpire, c.User.ID, c.User.Username, pollChangeVote)
		if err == nil {
			id, _ := res.LastInsertId()
			pollID = int(id)
		}

		// Link the poll to the topic
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}topics
			SET ID_POLL = ?
			WHERE ID_TOPIC = ?`), pollID, c.Topic)
	}

	// Get all the choices.  (no better way to remove all emptied and add
	// previously non-existent ones.)
	choices := map[int]bool{}
	rows, err := a.DB.Query(a.Q(`
		SELECT ID_CHOICE
		FROM {$db_prefix}poll_choices
		WHERE ID_POLL = ?`), pollID)
	if err == nil {
		for rows.Next() {
			var idChoice int
			rows.Scan(&idChoice)
			choices[idChoice] = true
		}
		rows.Close()
	}

	var deleteOptions []string
	if opts != nil {
		for _, key := range opts.Keys() {
			// Make sure the key is numeric for sanity's sake.
			k := atoi(key)
			option := opts.Str(key)

			// They've cleared the box.  Either they want it deleted, or it
			// never existed.
			if strings.TrimSpace(option) == "" {
				// They want it deleted.  Bye.
				if choices[k] {
					deleteOptions = append(deleteOptions, itoa(k))
				}
				continue
			}

			// Dress the option up for its big date with the database.
			option = Htmlspecialchars(option)

			// If it's already there, update it.  If it's not... add it.
			if choices[k] {
				a.DB.Exec(a.Q(`
					UPDATE {$db_prefix}poll_choices
					SET label = ?
					WHERE ID_POLL = ?
						AND ID_CHOICE = ?`), option, pollID, k)
			} else {
				a.DB.Exec(a.Q(`
					INSERT INTO {$db_prefix}poll_choices
						(ID_POLL, ID_CHOICE, label, votes)
					VALUES (?, ?, SUBSTR(?, 1, 255), 0)`), pollID, k, option)
			}
		}
	}

	// I'm sorry, but... well, no one was choosing you.  Poor options, I'll
	// put you out of your misery.
	if len(deleteOptions) > 0 {
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}log_polls
			WHERE ID_POLL = ` + itoa(pollID) + `
				AND ID_CHOICE IN (` + strings.Join(deleteOptions, ", ") + `)`))
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}poll_choices
			WHERE ID_POLL = ` + itoa(pollID) + `
				AND ID_CHOICE IN (` + strings.Join(deleteOptions, ", ") + `)`))
	}

	// Shall I reset the vote count, sir?
	if c.POST.Has("resetVoteCount") {
		a.DB.Exec(a.Q(`
			UPDATE {$db_prefix}poll_choices
			SET votes = 0
			WHERE ID_POLL = ?`), pollID)
		a.DB.Exec(a.Q(`
			DELETE FROM {$db_prefix}log_polls
			WHERE ID_POLL = ?`), pollID)
	}

	// Off we go.
	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}

// RemovePoll is RemovePoll(): remove a poll from a topic without removing
// the topic.
func (c *Ctx) RemovePoll() {
	a := c.App

	// Make sure the topic is not empty.
	if c.Topic == 0 {
		c.fatalLangError("1", false)
	}

	c.checkSession("get", "", true)

	// Check permissions.
	if !c.allowedTo("poll_remove_any") {
		var topicStarter, pollStarter int
		err := a.DB.QueryRow(a.Q(`
			SELECT t.ID_MEMBER_STARTED, p.ID_MEMBER AS pollStarter
			FROM {$db_prefix}topics AS t, {$db_prefix}polls AS p
			WHERE t.ID_TOPIC = ?
				AND p.ID_POLL = t.ID_POLL
			LIMIT 1`), c.Topic).Scan(&topicStarter, &pollStarter)
		if err != nil {
			c.fatalLangError("1", true)
		}

		if topicStarter == c.User.ID || (pollStarter != 0 && c.User.ID == pollStarter) {
			c.isAllowedTo("poll_remove_own")
		} else {
			c.isAllowedTo("poll_remove_any")
		}
	}

	// Retrieve the poll ID.
	var pollID int
	a.DB.QueryRow(a.Q(`
		SELECT ID_POLL
		FROM {$db_prefix}topics
		WHERE ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&pollID)

	// Remove all user logs for this poll.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}log_polls WHERE ID_POLL = ?`), pollID)
	// Remove all poll choices.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}poll_choices WHERE ID_POLL = ?`), pollID)
	// Remove the poll itself.
	a.DB.Exec(a.Q(`DELETE FROM {$db_prefix}polls WHERE ID_POLL = ?`), pollID)
	// Finally set the topic poll ID back to 0!
	a.DB.Exec(a.Q(`UPDATE {$db_prefix}topics SET ID_POLL = 0 WHERE ID_TOPIC = ?`), c.Topic)

	// Take the moderator back to the topic.
	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}
