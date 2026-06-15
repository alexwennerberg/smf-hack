package app

// Port of Sources/LockTopic.php: LockTopic and Sticky.

func init() {
	registerAction("lock", (*Ctx).LockTopic)
	registerAction("sticky", (*Ctx).Sticky)
}

// LockTopic is LockTopic(): locks a topic, toggling between
// locked/unlocked/admin locked.
func (c *Ctx) LockTopic() {
	a := c.App

	// Just quit if there's no topic to lock.
	if c.Topic == 0 {
		c.fatalLangError("472", false)
	}

	c.checkSession("get", "", true)

	// Find out who started the topic - in case User Topic Locking is
	// enabled.
	var starter, locked int
	a.DB.QueryRow(a.Q(`
		SELECT ID_MEMBER_STARTED, locked
		FROM {$db_prefix}topics
		WHERE ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&starter, &locked)

	// Can you lock topics here, mister?
	userLock := !c.allowedTo("lock_any")
	if userLock && starter == c.User.ID {
		c.isAllowedTo("lock_own")
	} else {
		c.isAllowedTo("lock_any")
	}

	if locked == 0 && !userLock {
		// Locking with high privileges.
		locked = 1
	} else if locked == 0 {
		// Locking with low privileges.
		locked = 2
	} else if locked == 2 || (locked == 1 && !userLock) {
		// Unlocking - make sure you don't unlock what you can't.
		locked = 0
	} else {
		// You cannot unlock this!
		c.fatalLangError("smf31", true)
	}

	// Actually lock the topic in the database with the new value.
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}topics
		SET locked = ?
		WHERE ID_TOPIC = ?`), locked, c.Topic)

	// If they are allowed a "moderator" permission, log it in the moderator
	// log.
	if !userLock {
		c.logAction("lock", map[string]any{"topic": c.Topic})
	}
	// Notify people that this topic has been locked?
	if locked == 0 {
		c.sendNotifications(c.Topic, "unlock")
	} else {
		c.sendNotifications(c.Topic, "lock")
	}

	// Back to the topic!
	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}

// Sticky is Sticky(): toggle a topic between sticky and normal.
func (c *Ctx) Sticky() {
	a := c.App

	// Make sure the user can sticky it, and they are stickying *something*.
	c.isAllowedTo("make_sticky")

	// You shouldn't be able to (un)sticky a topic if the setting is
	// disabled.
	if a.SettingEmpty("enableStickyTopics") {
		c.fatalLangError("cannot_make_sticky", false)
	}

	// You can't sticky a board or something!
	if c.Topic == 0 {
		c.fatalLangError("472", false)
	}

	c.checkSession("get", "", true)

	// Is this topic already stickied, or no?
	var isSticky int
	a.DB.QueryRow(a.Q(`
		SELECT isSticky
		FROM {$db_prefix}topics
		WHERE ID_TOPIC = ?
		LIMIT 1`), c.Topic).Scan(&isSticky)

	// Toggle the sticky value.... pretty simple ;).
	newSticky := 1
	if isSticky != 0 {
		newSticky = 0
	}
	a.DB.Exec(a.Q(`
		UPDATE {$db_prefix}topics
		SET isSticky = ?
		WHERE ID_TOPIC = ?`), newSticky, c.Topic)

	// Log this sticky action - always a moderator thing.
	c.logAction("sticky", map[string]any{"topic": c.Topic})
	// Notify people that this topic has been stickied?
	if isSticky == 0 {
		c.sendNotifications(c.Topic, "sticky")
	}

	// Take them back to the now stickied topic.
	c.redirectExit("topic=" + itoa(c.Topic) + "." + c.REQUEST.Str("start"))
}
