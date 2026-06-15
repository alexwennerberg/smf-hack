package app

// SQLite translation of install_1-1.sql. Table and column names are kept
// exactly as in SMF 1.1.21 so queries port verbatim from the PHP sources.
//
// Translation rules (see plan):
//   - auto_increment primary keys -> INTEGER PRIMARY KEY AUTOINCREMENT
//     (AUTOINCREMENT is load-bearing: SMF assumes message/topic IDs are never
//     reused, e.g. for ID_LAST_MSG and "mark read" logic)
//   - other ints/tinyints -> INTEGER, float -> REAL
//   - varchar/tinytext/text -> TEXT (MySQL silently truncated to the column
//     length; the Go code replicates truncation at the insert sites)
//   - date -> TEXT in 'YYYY-MM-DD' form
//   - log_online.logTime was an auto-update TIMESTAMP; here it is INTEGER
//     unix time and the queries bind time.Now().Unix() instead of
//     FROM_UNIXTIME()/NOW()
//   - UNIQUE KEY/KEY -> separate CREATE [UNIQUE] INDEX statements; index
//     names are prefixed with the table name because SQLite index names are
//     database-global; MySQL prefix lengths like memberGroups(48) are dropped
//   - memberName/emailAddress and other identity lookups get COLLATE NOCASE
//     to match MySQL's case-insensitive latin1 collation

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

var schemaStatements = []string{
	// attachments
	`CREATE TABLE {$db_prefix}attachments (
  ID_ATTACH INTEGER PRIMARY KEY AUTOINCREMENT,
  ID_THUMB INTEGER NOT NULL DEFAULT 0,
  ID_MSG INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  attachmentType INTEGER NOT NULL DEFAULT 0,
  filename TEXT NOT NULL DEFAULT '',
  file_hash TEXT NOT NULL DEFAULT '',
  size INTEGER NOT NULL DEFAULT 0,
  downloads INTEGER NOT NULL DEFAULT 0,
  width INTEGER NOT NULL DEFAULT 0,
  height INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE UNIQUE INDEX {$db_prefix}attachments_ID_MEMBER ON {$db_prefix}attachments (ID_MEMBER, ID_ATTACH)`,
	`CREATE INDEX {$db_prefix}attachments_ID_MSG ON {$db_prefix}attachments (ID_MSG)`,

	// ban_groups
	`CREATE TABLE {$db_prefix}ban_groups (
  ID_BAN_GROUP INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL DEFAULT '',
  ban_time INTEGER NOT NULL DEFAULT 0,
  expire_time INTEGER,
  cannot_access INTEGER NOT NULL DEFAULT 0,
  cannot_register INTEGER NOT NULL DEFAULT 0,
  cannot_post INTEGER NOT NULL DEFAULT 0,
  cannot_login INTEGER NOT NULL DEFAULT 0,
  reason TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT ''
)`,

	// ban_items
	`CREATE TABLE {$db_prefix}ban_items (
  ID_BAN INTEGER PRIMARY KEY AUTOINCREMENT,
  ID_BAN_GROUP INTEGER NOT NULL DEFAULT 0,
  ip_low1 INTEGER NOT NULL DEFAULT 0,
  ip_high1 INTEGER NOT NULL DEFAULT 0,
  ip_low2 INTEGER NOT NULL DEFAULT 0,
  ip_high2 INTEGER NOT NULL DEFAULT 0,
  ip_low3 INTEGER NOT NULL DEFAULT 0,
  ip_high3 INTEGER NOT NULL DEFAULT 0,
  ip_low4 INTEGER NOT NULL DEFAULT 0,
  ip_high4 INTEGER NOT NULL DEFAULT 0,
  hostname TEXT NOT NULL DEFAULT '',
  email_address TEXT NOT NULL DEFAULT '',
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  hits INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE INDEX {$db_prefix}ban_items_ID_BAN_GROUP ON {$db_prefix}ban_items (ID_BAN_GROUP)`,

	// board_permissions
	`CREATE TABLE {$db_prefix}board_permissions (
  ID_GROUP INTEGER NOT NULL DEFAULT 0,
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  permission TEXT NOT NULL DEFAULT '',
  addDeny INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (ID_GROUP, ID_BOARD, permission)
)`,

	// boards
	`CREATE TABLE {$db_prefix}boards (
  ID_BOARD INTEGER PRIMARY KEY AUTOINCREMENT,
  ID_CAT INTEGER NOT NULL DEFAULT 0,
  childLevel INTEGER NOT NULL DEFAULT 0,
  ID_PARENT INTEGER NOT NULL DEFAULT 0,
  boardOrder INTEGER NOT NULL DEFAULT 0,
  ID_LAST_MSG INTEGER NOT NULL DEFAULT 0,
  ID_MSG_UPDATED INTEGER NOT NULL DEFAULT 0,
  memberGroups TEXT NOT NULL DEFAULT '-1,0',
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  numTopics INTEGER NOT NULL DEFAULT 0,
  numPosts INTEGER NOT NULL DEFAULT 0,
  countPosts INTEGER NOT NULL DEFAULT 0,
  ID_THEME INTEGER NOT NULL DEFAULT 0,
  permission_mode INTEGER NOT NULL DEFAULT 0,
  override_theme INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE UNIQUE INDEX {$db_prefix}boards_categories ON {$db_prefix}boards (ID_CAT, ID_BOARD)`,
	`CREATE INDEX {$db_prefix}boards_ID_PARENT ON {$db_prefix}boards (ID_PARENT)`,
	`CREATE INDEX {$db_prefix}boards_ID_MSG_UPDATED ON {$db_prefix}boards (ID_MSG_UPDATED)`,
	`CREATE INDEX {$db_prefix}boards_memberGroups ON {$db_prefix}boards (memberGroups)`,

	// calendar
	`CREATE TABLE {$db_prefix}calendar (
  ID_EVENT INTEGER PRIMARY KEY AUTOINCREMENT,
  startDate TEXT NOT NULL DEFAULT '0001-01-01',
  endDate TEXT NOT NULL DEFAULT '0001-01-01',
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  title TEXT NOT NULL DEFAULT '',
  ID_MEMBER INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE INDEX {$db_prefix}calendar_startDate ON {$db_prefix}calendar (startDate)`,
	`CREATE INDEX {$db_prefix}calendar_endDate ON {$db_prefix}calendar (endDate)`,
	`CREATE INDEX {$db_prefix}calendar_topic ON {$db_prefix}calendar (ID_TOPIC, ID_MEMBER)`,

	// calendar_holidays
	`CREATE TABLE {$db_prefix}calendar_holidays (
  ID_HOLIDAY INTEGER PRIMARY KEY AUTOINCREMENT,
  eventDate TEXT NOT NULL DEFAULT '0001-01-01',
  title TEXT NOT NULL DEFAULT ''
)`,
	`CREATE INDEX {$db_prefix}calendar_holidays_eventDate ON {$db_prefix}calendar_holidays (eventDate)`,

	// categories
	`CREATE TABLE {$db_prefix}categories (
  ID_CAT INTEGER PRIMARY KEY AUTOINCREMENT,
  catOrder INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL DEFAULT '',
  canCollapse INTEGER NOT NULL DEFAULT 1
)`,

	// collapsed_categories
	`CREATE TABLE {$db_prefix}collapsed_categories (
  ID_CAT INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_CAT, ID_MEMBER)
)`,

	// log_actions (extra was PHP-serialized; here it is JSON)
	`CREATE TABLE {$db_prefix}log_actions (
  ID_ACTION INTEGER PRIMARY KEY AUTOINCREMENT,
  logTime INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ip TEXT NOT NULL DEFAULT '                ',
  action TEXT NOT NULL DEFAULT '',
  extra TEXT NOT NULL DEFAULT ''
)`,
	`CREATE INDEX {$db_prefix}log_actions_logTime ON {$db_prefix}log_actions (logTime)`,
	`CREATE INDEX {$db_prefix}log_actions_ID_MEMBER ON {$db_prefix}log_actions (ID_MEMBER)`,

	// log_activity
	`CREATE TABLE {$db_prefix}log_activity (
  date TEXT NOT NULL DEFAULT '0001-01-01',
  hits INTEGER NOT NULL DEFAULT 0,
  topics INTEGER NOT NULL DEFAULT 0,
  posts INTEGER NOT NULL DEFAULT 0,
  registers INTEGER NOT NULL DEFAULT 0,
  mostOn INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (date)
)`,
	`CREATE INDEX {$db_prefix}log_activity_hits ON {$db_prefix}log_activity (hits)`,
	`CREATE INDEX {$db_prefix}log_activity_mostOn ON {$db_prefix}log_activity (mostOn)`,

	// log_banned
	`CREATE TABLE {$db_prefix}log_banned (
  ID_BAN_LOG INTEGER PRIMARY KEY AUTOINCREMENT,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ip TEXT NOT NULL DEFAULT '                ',
  email TEXT NOT NULL DEFAULT '',
  logTime INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE INDEX {$db_prefix}log_banned_logTime ON {$db_prefix}log_banned (logTime)`,

	// log_boards
	`CREATE TABLE {$db_prefix}log_boards (
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  ID_MSG INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_MEMBER, ID_BOARD)
)`,

	// log_errors
	`CREATE TABLE {$db_prefix}log_errors (
  ID_ERROR INTEGER PRIMARY KEY AUTOINCREMENT,
  logTime INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ip TEXT NOT NULL DEFAULT '                ',
  url TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  session TEXT NOT NULL DEFAULT '                                '
)`,
	`CREATE INDEX {$db_prefix}log_errors_logTime ON {$db_prefix}log_errors (logTime)`,
	`CREATE INDEX {$db_prefix}log_errors_ID_MEMBER ON {$db_prefix}log_errors (ID_MEMBER)`,
	`CREATE INDEX {$db_prefix}log_errors_ip ON {$db_prefix}log_errors (ip)`,

	// log_floodcontrol
	`CREATE TABLE {$db_prefix}log_floodcontrol (
  ip TEXT NOT NULL DEFAULT '                ',
  logTime INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ip)
)`,

	// log_mark_read
	`CREATE TABLE {$db_prefix}log_mark_read (
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  ID_MSG INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_MEMBER, ID_BOARD)
)`,

	// log_notify
	`CREATE TABLE {$db_prefix}log_notify (
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  sent INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_MEMBER, ID_TOPIC, ID_BOARD)
)`,

	// log_online (logTime: INTEGER unix time, was auto-update TIMESTAMP)
	`CREATE TABLE {$db_prefix}log_online (
  session TEXT NOT NULL DEFAULT '',
  logTime INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ip INTEGER NOT NULL DEFAULT 0,
  url TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (session)
)`,
	`CREATE INDEX {$db_prefix}log_online_logTime ON {$db_prefix}log_online (logTime)`,
	`CREATE INDEX {$db_prefix}log_online_ID_MEMBER ON {$db_prefix}log_online (ID_MEMBER)`,

	// log_polls
	`CREATE TABLE {$db_prefix}log_polls (
  ID_POLL INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_CHOICE INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_POLL, ID_MEMBER, ID_CHOICE)
)`,

	// log_search_messages
	`CREATE TABLE {$db_prefix}log_search_messages (
  ID_SEARCH INTEGER NOT NULL DEFAULT 0,
  ID_MSG INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_SEARCH, ID_MSG)
)`,

	// log_search_results
	`CREATE TABLE {$db_prefix}log_search_results (
  ID_SEARCH INTEGER NOT NULL DEFAULT 0,
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  ID_MSG INTEGER NOT NULL DEFAULT 0,
  relevance INTEGER NOT NULL DEFAULT 0,
  num_matches INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_SEARCH, ID_TOPIC)
)`,

	// log_search_subjects
	`CREATE TABLE {$db_prefix}log_search_subjects (
  word TEXT NOT NULL DEFAULT '',
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (word, ID_TOPIC)
)`,
	`CREATE INDEX {$db_prefix}log_search_subjects_ID_TOPIC ON {$db_prefix}log_search_subjects (ID_TOPIC)`,

	// log_search_topics
	`CREATE TABLE {$db_prefix}log_search_topics (
  ID_SEARCH INTEGER NOT NULL DEFAULT 0,
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_SEARCH, ID_TOPIC)
)`,

	// log_topics
	`CREATE TABLE {$db_prefix}log_topics (
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  ID_MSG INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_MEMBER, ID_TOPIC)
)`,
	`CREATE INDEX {$db_prefix}log_topics_ID_TOPIC ON {$db_prefix}log_topics (ID_TOPIC)`,

	// membergroups
	`CREATE TABLE {$db_prefix}membergroups (
  ID_GROUP INTEGER PRIMARY KEY AUTOINCREMENT,
  groupName TEXT NOT NULL DEFAULT '',
  onlineColor TEXT NOT NULL DEFAULT '',
  minPosts INTEGER NOT NULL DEFAULT -1,
  maxMessages INTEGER NOT NULL DEFAULT 0,
  stars TEXT NOT NULL DEFAULT ''
)`,
	`CREATE INDEX {$db_prefix}membergroups_minPosts ON {$db_prefix}membergroups (minPosts)`,

	// members
	`CREATE TABLE {$db_prefix}members (
  ID_MEMBER INTEGER PRIMARY KEY AUTOINCREMENT,
  memberName TEXT NOT NULL DEFAULT '' COLLATE NOCASE,
  dateRegistered INTEGER NOT NULL DEFAULT 0,
  posts INTEGER NOT NULL DEFAULT 0,
  ID_GROUP INTEGER NOT NULL DEFAULT 0,
  lngfile TEXT NOT NULL DEFAULT '',
  lastLogin INTEGER NOT NULL DEFAULT 0,
  realName TEXT NOT NULL DEFAULT '' COLLATE NOCASE,
  instantMessages INTEGER NOT NULL DEFAULT 0,
  unreadMessages INTEGER NOT NULL DEFAULT 0,
  buddy_list TEXT NOT NULL DEFAULT '',
  pm_ignore_list TEXT NOT NULL DEFAULT '',
  messageLabels TEXT NOT NULL DEFAULT '',
  passwd TEXT NOT NULL DEFAULT '',
  emailAddress TEXT NOT NULL DEFAULT '' COLLATE NOCASE,
  personalText TEXT NOT NULL DEFAULT '',
  gender INTEGER NOT NULL DEFAULT 0,
  birthdate TEXT NOT NULL DEFAULT '0001-01-01',
  websiteTitle TEXT NOT NULL DEFAULT '',
  websiteUrl TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  hideEmail INTEGER NOT NULL DEFAULT 0,
  showOnline INTEGER NOT NULL DEFAULT 1,
  timeFormat TEXT NOT NULL DEFAULT '',
  signature TEXT NOT NULL DEFAULT '',
  timeOffset REAL NOT NULL DEFAULT 0,
  avatar TEXT NOT NULL DEFAULT '',
  pm_email_notify INTEGER NOT NULL DEFAULT 0,
  usertitle TEXT NOT NULL DEFAULT '',
  notifyAnnouncements INTEGER NOT NULL DEFAULT 1,
  notifyOnce INTEGER NOT NULL DEFAULT 1,
  notifySendBody INTEGER NOT NULL DEFAULT 0,
  notifyTypes INTEGER NOT NULL DEFAULT 2,
  memberIP TEXT NOT NULL DEFAULT '',
  memberIP2 TEXT NOT NULL DEFAULT '',
  secretQuestion TEXT NOT NULL DEFAULT '',
  secretAnswer TEXT NOT NULL DEFAULT '',
  ID_THEME INTEGER NOT NULL DEFAULT 0,
  is_activated INTEGER NOT NULL DEFAULT 1,
  validation_code TEXT NOT NULL DEFAULT '',
  ID_MSG_LAST_VISIT INTEGER NOT NULL DEFAULT 0,
  additionalGroups TEXT NOT NULL DEFAULT '',
  smileySet TEXT NOT NULL DEFAULT '',
  ID_POST_GROUP INTEGER NOT NULL DEFAULT 0,
  totalTimeLoggedIn INTEGER NOT NULL DEFAULT 0,
  passwordSalt TEXT NOT NULL DEFAULT ''
)`,
	`CREATE INDEX {$db_prefix}members_memberName ON {$db_prefix}members (memberName)`,
	`CREATE INDEX {$db_prefix}members_dateRegistered ON {$db_prefix}members (dateRegistered)`,
	`CREATE INDEX {$db_prefix}members_ID_GROUP ON {$db_prefix}members (ID_GROUP)`,
	`CREATE INDEX {$db_prefix}members_birthdate ON {$db_prefix}members (birthdate)`,
	`CREATE INDEX {$db_prefix}members_posts ON {$db_prefix}members (posts)`,
	`CREATE INDEX {$db_prefix}members_lastLogin ON {$db_prefix}members (lastLogin)`,
	`CREATE INDEX {$db_prefix}members_lngfile ON {$db_prefix}members (lngfile)`,
	`CREATE INDEX {$db_prefix}members_ID_POST_GROUP ON {$db_prefix}members (ID_POST_GROUP)`,

	// message_icons
	`CREATE TABLE {$db_prefix}message_icons (
  ID_ICON INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL DEFAULT '',
  filename TEXT NOT NULL DEFAULT '',
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  iconOrder INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE INDEX {$db_prefix}message_icons_ID_BOARD ON {$db_prefix}message_icons (ID_BOARD)`,

	// messages
	`CREATE TABLE {$db_prefix}messages (
  ID_MSG INTEGER PRIMARY KEY AUTOINCREMENT,
  ID_TOPIC INTEGER NOT NULL DEFAULT 0,
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  posterTime INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_MSG_MODIFIED INTEGER NOT NULL DEFAULT 0,
  subject TEXT NOT NULL DEFAULT '',
  posterName TEXT NOT NULL DEFAULT '',
  posterEmail TEXT NOT NULL DEFAULT '',
  posterIP TEXT NOT NULL DEFAULT '',
  smileysEnabled INTEGER NOT NULL DEFAULT 1,
  modifiedTime INTEGER NOT NULL DEFAULT 0,
  modifiedName TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT 'xx'
)`,
	`CREATE UNIQUE INDEX {$db_prefix}messages_topic ON {$db_prefix}messages (ID_TOPIC, ID_MSG)`,
	`CREATE UNIQUE INDEX {$db_prefix}messages_ID_BOARD ON {$db_prefix}messages (ID_BOARD, ID_MSG)`,
	`CREATE UNIQUE INDEX {$db_prefix}messages_ID_MEMBER ON {$db_prefix}messages (ID_MEMBER, ID_MSG)`,
	`CREATE INDEX {$db_prefix}messages_ipIndex ON {$db_prefix}messages (posterIP, ID_TOPIC)`,
	`CREATE INDEX {$db_prefix}messages_participation ON {$db_prefix}messages (ID_MEMBER, ID_TOPIC)`,
	`CREATE INDEX {$db_prefix}messages_showPosts ON {$db_prefix}messages (ID_MEMBER, ID_BOARD)`,
	`CREATE INDEX {$db_prefix}messages_ID_TOPIC ON {$db_prefix}messages (ID_TOPIC)`,

	// moderators
	`CREATE TABLE {$db_prefix}moderators (
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_BOARD, ID_MEMBER)
)`,

	// package_servers (kept for schema fidelity; package manager is dropped)
	`CREATE TABLE {$db_prefix}package_servers (
  ID_SERVER INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT ''
)`,

	// permissions
	`CREATE TABLE {$db_prefix}permissions (
  ID_GROUP INTEGER NOT NULL DEFAULT 0,
  permission TEXT NOT NULL DEFAULT '',
  addDeny INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (ID_GROUP, permission)
)`,

	// personal_messages
	`CREATE TABLE {$db_prefix}personal_messages (
  ID_PM INTEGER PRIMARY KEY AUTOINCREMENT,
  ID_MEMBER_FROM INTEGER NOT NULL DEFAULT 0,
  deletedBySender INTEGER NOT NULL DEFAULT 0,
  fromName TEXT NOT NULL DEFAULT '',
  msgtime INTEGER NOT NULL DEFAULT 0,
  subject TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT ''
)`,
	`CREATE INDEX {$db_prefix}personal_messages_ID_MEMBER ON {$db_prefix}personal_messages (ID_MEMBER_FROM, deletedBySender)`,
	`CREATE INDEX {$db_prefix}personal_messages_msgtime ON {$db_prefix}personal_messages (msgtime)`,

	// pm_recipients
	`CREATE TABLE {$db_prefix}pm_recipients (
  ID_PM INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  labels TEXT NOT NULL DEFAULT '-1',
  bcc INTEGER NOT NULL DEFAULT 0,
  is_read INTEGER NOT NULL DEFAULT 0,
  deleted INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_PM, ID_MEMBER)
)`,
	`CREATE UNIQUE INDEX {$db_prefix}pm_recipients_ID_MEMBER ON {$db_prefix}pm_recipients (ID_MEMBER, deleted, ID_PM)`,

	// polls
	`CREATE TABLE {$db_prefix}polls (
  ID_POLL INTEGER PRIMARY KEY AUTOINCREMENT,
  question TEXT NOT NULL DEFAULT '',
  votingLocked INTEGER NOT NULL DEFAULT 0,
  maxVotes INTEGER NOT NULL DEFAULT 1,
  expireTime INTEGER NOT NULL DEFAULT 0,
  hideResults INTEGER NOT NULL DEFAULT 0,
  changeVote INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  posterName TEXT NOT NULL DEFAULT ''
)`,

	// poll_choices
	`CREATE TABLE {$db_prefix}poll_choices (
  ID_POLL INTEGER NOT NULL DEFAULT 0,
  ID_CHOICE INTEGER NOT NULL DEFAULT 0,
  label TEXT NOT NULL DEFAULT '',
  votes INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (ID_POLL, ID_CHOICE)
)`,

	// settings
	`CREATE TABLE {$db_prefix}settings (
  variable TEXT NOT NULL DEFAULT '',
  value TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (variable)
)`,

	// sessions (data was PHP session encoding; here it is JSON)
	`CREATE TABLE {$db_prefix}sessions (
  session_id TEXT NOT NULL,
  last_update INTEGER NOT NULL,
  data TEXT NOT NULL,
  PRIMARY KEY (session_id)
)`,

	// smileys
	`CREATE TABLE {$db_prefix}smileys (
  ID_SMILEY INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL DEFAULT '',
  filename TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  smileyRow INTEGER NOT NULL DEFAULT 0,
  smileyOrder INTEGER NOT NULL DEFAULT 0,
  hidden INTEGER NOT NULL DEFAULT 0
)`,

	// themes
	`CREATE TABLE {$db_prefix}themes (
  ID_MEMBER INTEGER NOT NULL DEFAULT 0,
  ID_THEME INTEGER NOT NULL DEFAULT 1,
  variable TEXT NOT NULL DEFAULT '',
  value TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (ID_THEME, ID_MEMBER, variable)
)`,
	`CREATE INDEX {$db_prefix}themes_ID_MEMBER ON {$db_prefix}themes (ID_MEMBER)`,

	// topics
	`CREATE TABLE {$db_prefix}topics (
  ID_TOPIC INTEGER PRIMARY KEY AUTOINCREMENT,
  isSticky INTEGER NOT NULL DEFAULT 0,
  ID_BOARD INTEGER NOT NULL DEFAULT 0,
  ID_FIRST_MSG INTEGER NOT NULL DEFAULT 0,
  ID_LAST_MSG INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER_STARTED INTEGER NOT NULL DEFAULT 0,
  ID_MEMBER_UPDATED INTEGER NOT NULL DEFAULT 0,
  ID_POLL INTEGER NOT NULL DEFAULT 0,
  numReplies INTEGER NOT NULL DEFAULT 0,
  numViews INTEGER NOT NULL DEFAULT 0,
  locked INTEGER NOT NULL DEFAULT 0
)`,
	`CREATE UNIQUE INDEX {$db_prefix}topics_lastMessage ON {$db_prefix}topics (ID_LAST_MSG, ID_BOARD)`,
	`CREATE UNIQUE INDEX {$db_prefix}topics_firstMessage ON {$db_prefix}topics (ID_FIRST_MSG, ID_BOARD)`,
	`CREATE UNIQUE INDEX {$db_prefix}topics_poll ON {$db_prefix}topics (ID_POLL, ID_TOPIC)`,
	`CREATE INDEX {$db_prefix}topics_isSticky ON {$db_prefix}topics (isSticky)`,
	`CREATE INDEX {$db_prefix}topics_ID_BOARD ON {$db_prefix}topics (ID_BOARD)`,
}

// CreateSchema creates all tables and indexes on an empty database.
func CreateSchema(db *sql.DB, prefix string) error {
	for _, stmt := range schemaStatements {
		if _, err := db.Exec(strings.ReplaceAll(stmt, "{$db_prefix}", prefix)); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

// InsertDefaultData populates a fresh database with the rows install.php
// inserted, with the English install-language strings substituted (from
// Themes/default/languages/Install.english.php).
func InsertDefaultData(db *sql.DB, cfg Config) error {
	now := time.Now().Unix()
	p := func(q string) string { return strings.ReplaceAll(q, "{$db_prefix}", cfg.DBPrefix) }

	exec := func(q string, args ...any) error {
		_, err := db.Exec(p(q), args...)
		return err
	}

	// board_permissions
	type bp struct {
		group int
		perm  string
	}
	boardPerms := []bp{{-1, "poll_view"}}
	memberPerms := []string{"remove_own", "lock_own", "mark_any_notify", "mark_notify", "modify_own",
		"poll_add_own", "poll_edit_own", "poll_lock_own", "poll_post", "poll_view", "poll_vote",
		"post_attachment", "post_new", "post_reply_any", "post_reply_own", "delete_own",
		"report_any", "send_topic", "view_attachments"}
	for _, perm := range memberPerms {
		boardPerms = append(boardPerms, bp{0, perm})
	}
	// Groups 2 (global moderator) and 3 (moderator) share the same set.
	modPerms := []string{"moderate_board", "post_new", "post_reply_own", "post_reply_any",
		"poll_post", "poll_remove_any", "poll_view", "poll_vote", "report_any",
		"lock_own", "send_topic", "mark_any_notify", "mark_notify", "delete_own",
		"modify_own", "make_sticky", "lock_any", "remove_any", "move_any",
		"merge_any", "split_any", "delete_any", "modify_any"}
	for _, perm := range append([]string{"poll_add_any", "poll_edit_any"}, modPerms...) {
		boardPerms = append(boardPerms, bp{2, perm})
	}
	for _, perm := range append([]string{"poll_add_own"}, modPerms...) {
		boardPerms = append(boardPerms, bp{3, perm})
	}
	for _, e := range boardPerms {
		if err := exec(`INSERT INTO {$db_prefix}board_permissions (ID_GROUP, ID_BOARD, permission) VALUES (?, 0, ?)`, e.group, e.perm); err != nil {
			return err
		}
	}

	// boards
	if err := exec(`INSERT INTO {$db_prefix}boards
		(ID_BOARD, ID_CAT, boardOrder, ID_LAST_MSG, ID_MSG_UPDATED, name, description, numTopics, numPosts, memberGroups)
		VALUES (1, 1, 1, 1, 1, ?, ?, 1, 1, '-1,0')`,
		"General Discussion", "Feel free to talk about anything and everything in this board."); err != nil {
		return err
	}

	// calendar_holidays
	for _, h := range defaultHolidays {
		if err := exec(`INSERT INTO {$db_prefix}calendar_holidays (title, eventDate) VALUES (?, ?)`, h.title, h.date); err != nil {
			return err
		}
	}

	// categories
	if err := exec(`INSERT INTO {$db_prefix}categories (ID_CAT, catOrder, name, canCollapse) VALUES (1, 0, ?, 1)`, "General Category"); err != nil {
		return err
	}

	// membergroups
	type mg struct {
		id       int
		name     string
		color    string
		minPosts int
		stars    string
	}
	for _, g := range []mg{
		{1, "Administrator", "#FF0000", -1, "5#staradmin.gif"},
		{2, "Global Moderator", "#0000FF", -1, "5#stargmod.gif"},
		{3, "Moderator", "", -1, "5#starmod.gif"},
		{4, "Newbie", "", 0, "1#star.gif"},
		{5, "Jr. Member", "", 50, "2#star.gif"},
		{6, "Full Member", "", 100, "3#star.gif"},
		{7, "Sr. Member", "", 250, "4#star.gif"},
		{8, "Hero Member", "", 500, "5#star.gif"},
	} {
		if err := exec(`INSERT INTO {$db_prefix}membergroups (ID_GROUP, groupName, onlineColor, minPosts, stars) VALUES (?, ?, ?, ?, ?)`,
			g.id, g.name, g.color, g.minPosts, g.stars); err != nil {
			return err
		}
	}

	// message_icons
	for i, ic := range []struct{ filename, title string }{
		{"xx", "Standard"}, {"thumbup", "Thumb Up"}, {"thumbdown", "Thumb Down"},
		{"exclamation", "Exclamation point"}, {"question", "Question mark"}, {"lamp", "Lamp"},
		{"smiley", "Smiley"}, {"angry", "Angry"}, {"cheesy", "Cheesy"}, {"grin", "Grin"},
		{"sad", "Sad"}, {"wink", "Wink"},
	} {
		if err := exec(`INSERT INTO {$db_prefix}message_icons (filename, title, iconOrder) VALUES (?, ?, ?)`, ic.filename, ic.title, i); err != nil {
			return err
		}
	}

	// messages
	if err := exec(`INSERT INTO {$db_prefix}messages
		(ID_MSG, ID_MSG_MODIFIED, ID_TOPIC, ID_BOARD, posterTime, subject, posterName, posterEmail, posterIP, modifiedName, body, icon)
		VALUES (1, 1, 1, 1, ?, ?, 'Simple Machines', 'info@simplemachines.org', '127.0.0.1', '', ?, 'xx')`,
		now, "Welcome to SMF!",
		"Welcome to Simple Machines Forum!<br /><br />We hope you enjoy using your forum.&nbsp; If you have any problems, please feel free to [url=http://www.simplemachines.org/community/index.php]ask us for assistance[/url].<br /><br />Thanks!<br />Simple Machines"); err != nil {
		return err
	}

	// package_servers (table kept populated for fidelity even though the
	// package manager is not ported)
	if err := exec(`INSERT INTO {$db_prefix}package_servers (name, url) VALUES ('Simple Machines Third-party Mod Site', 'http://mods.simplemachines.org')`); err != nil {
		return err
	}

	// permissions
	guestPerms := []string{"search_posts", "calendar_view", "view_stats", "profile_view_any"}
	memberGeneralPerms := []string{"view_mlist", "search_posts", "profile_view_own", "profile_view_any",
		"pm_read", "pm_send", "calendar_view", "view_stats", "who_view", "profile_identity_own",
		"profile_extra_own", "profile_remove_own", "profile_server_avatar", "profile_upload_avatar",
		"profile_remote_avatar"}
	gmodGeneralPerms := append(append([]string{}, memberGeneralPerms[:15]...),
		"profile_title_own", "calendar_post", "calendar_edit_any")
	for _, perm := range guestPerms {
		if err := exec(`INSERT INTO {$db_prefix}permissions (ID_GROUP, permission) VALUES (-1, ?)`, perm); err != nil {
			return err
		}
	}
	for _, perm := range memberGeneralPerms {
		if err := exec(`INSERT INTO {$db_prefix}permissions (ID_GROUP, permission) VALUES (0, ?)`, perm); err != nil {
			return err
		}
	}
	for _, perm := range gmodGeneralPerms {
		if err := exec(`INSERT INTO {$db_prefix}permissions (ID_GROUP, permission) VALUES (2, ?)`, perm); err != nil {
			return err
		}
	}

	// settings
	settings := [][2]string{
		{"smfVersion", "1.1.21"},
		{"news", "SMF - Just Installed!"},
		{"compactTopicPagesContiguous", "5"},
		{"compactTopicPagesEnable", "1"},
		{"enableStickyTopics", "1"},
		{"todayMod", "1"},
		{"enablePreviousNext", "1"},
		{"pollMode", "1"},
		{"enableVBStyleLogin", "1"},
		{"enableCompressedOutput", "1"},
		{"attachmentSizeLimit", "128"},
		{"attachmentPostLimit", "192"},
		{"attachmentNumPerPostLimit", "4"},
		{"attachmentDirSizeLimit", "10240"},
		{"attachmentUploadDir", cfg.AssetDir + "/attachments"},
		{"attachmentExtensions", "doc,gif,jpg,mpg,pdf,png,txt,zip"},
		{"attachmentCheckExtensions", "0"},
		{"attachmentShowImages", "1"},
		{"attachmentEnable", "1"},
		{"attachmentEncryptFilenames", "1"},
		{"attachmentThumbnails", "1"},
		{"attachmentThumbWidth", "150"},
		{"attachmentThumbHeight", "150"},
		{"censorIgnoreCase", "1"},
		{"mostOnline", "1"},
		{"mostOnlineToday", "1"},
		{"mostDate", fmt.Sprint(now)},
		{"allow_disableAnnounce", "1"},
		{"trackStats", "1"},
		{"userLanguage", "1"},
		{"titlesEnable", "1"},
		{"topicSummaryPosts", "15"},
		{"enableErrorLogging", "1"},
		{"max_image_width", "0"},
		{"max_image_height", "0"},
		{"onlineEnable", "0"},
		{"cal_holidaycolor", "000080"},
		{"cal_bdaycolor", "920AC4"},
		{"cal_eventcolor", "078907"},
		{"cal_enabled", "0"},
		{"cal_maxyear", "2010"},
		{"cal_minyear", "2004"},
		{"cal_daysaslink", "0"},
		{"cal_defaultboard", ""},
		{"cal_showeventsonindex", "0"},
		{"cal_showbdaysonindex", "0"},
		{"cal_showholidaysonindex", "0"},
		{"cal_showeventsoncalendar", "1"},
		{"cal_showbdaysoncalendar", "1"},
		{"cal_showholidaysoncalendar", "1"},
		{"cal_showweeknum", "0"},
		{"cal_maxspan", "7"},
		{"smtp_host", ""},
		{"smtp_port", "25"},
		{"smtp_username", ""},
		{"smtp_password", ""},
		{"mail_type", "0"},
		{"timeLoadPageEnable", "0"},
		{"totalTopics", "1"},
		{"totalMessages", "1"},
		{"simpleSearch", "0"},
		{"censor_vulgar", ""},
		{"censor_proper", ""},
		{"enablePostHTML", "0"},
		{"theme_allow", "1"},
		{"theme_default", "1"},
		{"theme_guests", "1"},
		{"enableEmbeddedFlash", "0"},
		{"xmlnews_enable", "1"},
		{"xmlnews_maxlen", "255"},
		{"hotTopicPosts", "15"},
		{"hotTopicVeryPosts", "25"},
		{"registration_method", "0"},
		{"send_validation_onChange", "0"},
		{"send_welcomeEmail", "1"},
		{"allow_editDisplayName", "1"},
		{"allow_hideOnline", "1"},
		{"allow_hideEmail", "1"},
		{"guest_hideContacts", "0"},
		{"spamWaitTime", "5"},
		{"pm_spam_settings", "10,5,20"},
		{"reserveWord", "0"},
		{"reserveCase", "1"},
		{"reserveUser", "1"},
		{"reserveName", "1"},
		{"reserveNames", "Admin\nWebmaster\nGuest\nroot"},
		{"autoLinkUrls", "1"},
		{"banLastUpdated", "0"},
		{"smileys_dir", cfg.AssetDir + "/Smileys"},
		{"smileys_url", cfg.BoardURL + "/Smileys"},
		{"avatar_directory", cfg.AssetDir + "/avatars"},
		{"avatar_url", cfg.BoardURL + "/avatars"},
		{"avatar_max_height_external", "65"},
		{"avatar_max_width_external", "65"},
		{"avatar_action_too_large", "option_html_resize"},
		{"avatar_max_height_upload", "65"},
		{"avatar_max_width_upload", "65"},
		{"avatar_resize_upload", "1"},
		{"avatar_download_png", "1"},
		{"failed_login_threshold", "3"},
		{"oldTopicDays", "120"},
		{"edit_wait_time", "90"},
		{"edit_disable_time", "0"},
		{"autoFixDatabase", "1"},
		{"allow_guestAccess", "1"},
		{"time_format", "%B %d, %Y, %I:%M:%S %p"},
		{"number_format", "1234.00"},
		{"enableBBC", "1"},
		{"max_messageLength", "20000"},
		{"max_signatureLength", "300"},
		{"autoOptDatabase", "7"},
		{"autoOptMaxOnline", "0"},
		{"autoOptLastOpt", "0"},
		{"defaultMaxMessages", "15"},
		{"defaultMaxTopics", "20"},
		{"defaultMaxMembers", "30"},
		{"enableParticipation", "1"},
		{"recycle_enable", "0"},
		{"recycle_board", "0"},
		{"maxMsgID", "1"},
		{"enableAllMessages", "0"},
		{"fixLongWords", "0"},
		{"knownThemes", "1"},
		{"who_enabled", "1"},
		{"time_offset", "0"},
		{"cookieTime", "60"},
		{"lastActive", "15"},
		{"smiley_sets_known", "default,classic"},
		{"smiley_sets_names", "Default\nClassic"},
		{"smiley_sets_default", "default"},
		{"cal_days_for_index", "7"},
		{"requireAgreement", "1"},
		{"unapprovedMembers", "0"},
		{"default_personalText", ""},
		{"package_make_backups", "1"},
		{"databaseSession_enable", "1"},
		{"databaseSession_loose", "1"},
		{"databaseSession_lifetime", "2880"},
		{"search_cache_size", "50"},
		{"search_results_per_page", "30"},
		{"search_weight_frequency", "30"},
		{"search_weight_age", "25"},
		{"search_weight_length", "20"},
		{"search_weight_subject", "15"},
		{"search_weight_first_message", "10"},
		{"search_max_results", "1200"},
		{"permission_enable_deny", "0"},
		{"permission_enable_postgroups", "0"},
		{"permission_enable_by_board", "0"},
	}
	for _, s := range settings {
		if err := exec(`INSERT INTO {$db_prefix}settings (variable, value) VALUES (?, ?)`, s[0], s[1]); err != nil {
			return err
		}
	}

	// smileys
	for i, s := range []struct {
		code, filename, description string
		hidden                      int
	}{
		{":)", "smiley.gif", "Smiley", 0},
		{";)", "wink.gif", "Wink", 0},
		{":D", "cheesy.gif", "Cheesy", 0},
		{";D", "grin.gif", "Grin", 0},
		{">:(", "angry.gif", "Angry", 0},
		{":(", "sad.gif", "Sad", 0},
		{":o", "shocked.gif", "Shocked", 0},
		{"8)", "cool.gif", "Cool", 0},
		{"???", "huh.gif", "Huh?", 0},
		{"::)", "rolleyes.gif", "Roll Eyes", 0},
		{":P", "tongue.gif", "Tongue", 0},
		{":-[", "embarrassed.gif", "Embarrassed", 0},
		{":-X", "lipsrsealed.gif", "Lips Sealed", 0},
		{":-\\", "undecided.gif", "Undecided", 0},
		{":-*", "kiss.gif", "Kiss", 0},
		{":'(", "cry.gif", "Cry", 0},
		{">:D", "evil.gif", "Evil", 1},
		{"^-^", "azn.gif", "Azn", 1},
		{"O0", "afro.gif", "Afro", 1},
	} {
		if err := exec(`INSERT INTO {$db_prefix}smileys (code, filename, description, smileyOrder, hidden) VALUES (?, ?, ?, ?, ?)`,
			s.code, s.filename, s.description, i, s.hidden); err != nil {
			return err
		}
	}

	// themes (only the default theme is ported)
	themeRows := [][3]string{
		{"1", "name", "SMF Default Theme - Core"},
		{"1", "theme_url", cfg.BoardURL + "/Themes/default"},
		{"1", "images_url", cfg.BoardURL + "/Themes/default/images"},
		{"1", "theme_dir", cfg.AssetDir + "/Themes/default"},
		{"1", "show_bbc", "1"},
		{"1", "show_latest_member", "1"},
		{"1", "show_modify", "1"},
		{"1", "show_user_images", "1"},
		{"1", "show_blurb", "1"},
		{"1", "show_gender", "0"},
		{"1", "show_newsfader", "0"},
		{"1", "number_recent_posts", "0"},
		{"1", "show_member_bar", "1"},
		{"1", "linktree_link", "1"},
		{"1", "show_profile_buttons", "1"},
		{"1", "show_mark_read", "1"},
		{"1", "show_sp1_info", "1"},
		{"1", "linktree_inline", "0"},
		{"1", "show_board_desc", "1"},
		{"1", "newsfader_time", "5000"},
		{"1", "allow_no_censored", "0"},
		{"1", "additional_options_collapsable", "1"},
		{"1", "use_image_buttons", "1"},
		{"1", "enable_news", "1"},
	}
	for _, r := range themeRows {
		if err := exec(`INSERT INTO {$db_prefix}themes (ID_THEME, variable, value) VALUES (?, ?, ?)`, r[0], r[1], r[2]); err != nil {
			return err
		}
	}

	// topics
	if err := exec(`INSERT INTO {$db_prefix}topics
		(ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED)
		VALUES (1, 1, 1, 1, 0, 0)`); err != nil {
		return err
	}

	return nil
}
