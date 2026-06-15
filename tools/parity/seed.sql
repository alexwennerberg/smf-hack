-- Mirror of goldenApp (go/internal/app/golden_test.go) applied on top of a
-- fresh SMF 1.1.21 install. Raw inserts, exactly like the Go seed, so derived
-- aggregate settings (totalMembers/totalMessages/...) stay at install values on
-- both sides. epoch = 1262304000 (2010-01-01 UTC). next() = epoch + 3600*N.

-- Frozen modSettings (goldenApp UpdateSettings).
INSERT INTO smf_settings (variable, value) VALUES
  ('securityDisable','1'),
  ('mostOnline','5'),
  ('mostOnlineToday','5'),
  ('mostDate','1262304000'),
  ('settings_updated','1262304000'),
  ('defaultMaxTopics','10'),
  ('defaultMaxMessages','10'),
  ('defaultMaxMembers','10')
ON DUPLICATE KEY UPDATE value = VALUES(value);

-- Members. admin is id 1 (from install). member=2, mod=3, user01..user14=4..17.
INSERT INTO smf_members
  (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
  VALUES ('member','Member One', SHA1('membermemberpass'),'m@b.com',0,1,1262304000,1262304000,1,4);
INSERT INTO smf_members
  (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP)
  VALUES ('mod','Mod One', SHA1('modmodpass'),'mod@b.com',0,2,1262304000,1262304000,1,4);

INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user01','User 1','x','user01@b.com',0,3,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user02','User 2','x','user02@b.com',0,6,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user03','User 3','x','user03@b.com',0,9,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user04','User 4','x','user04@b.com',0,12,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user05','User 5','x','user05@b.com',0,15,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user06','User 6','x','user06@b.com',0,18,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user07','User 7','x','user07@b.com',0,21,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user08','User 8','x','user08@b.com',0,24,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user09','User 9','x','user09@b.com',0,27,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user10','User 10','x','user10@b.com',0,30,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user11','User 11','x','user11@b.com',0,33,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user12','User 12','x','user12@b.com',0,36,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user13','User 13','x','user13@b.com',0,39,1262304000,1262304000,1,4);
INSERT INTO smf_members (memberName, realName, passwd, emailAddress, ID_GROUP, posts, dateRegistered, lastLogin, is_activated, ID_POST_GROUP) VALUES ('user14','User 14','x','user14@b.com',0,42,1262304000,1262304000,1,4);

-- mod moderates board 1.
INSERT INTO smf_moderators (ID_BOARD, ID_MEMBER) VALUES (1, 3);

-- Child board under board 1, and a second top-level board in the cat.
INSERT INTO smf_boards (ID_BOARD, ID_CAT, ID_PARENT, childLevel, name, description, memberGroups, boardOrder) VALUES (2, 1, 1, 1, 'Child Board', 'A nested board.', '-1,0,2', 1);
INSERT INTO smf_boards (ID_BOARD, ID_CAT, name, description, memberGroups, boardOrder) VALUES (3, 1, 'Second Board', 'Another board.', '-1,0,2', 2);

-- 12 plain topics on board 1 (ids 101..112). Messages get ids 2..13.
-- addTopic: insert message first, then topic. posterTime = epoch + 3600*i.
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (101,1,1,'admin','a@b.com',1262304000+3600*1,'Topic number 01','Body of topic 01.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (101,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,1,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (102,1,1,'admin','a@b.com',1262304000+3600*2,'Topic number 02','Body of topic 02.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (102,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,2,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (103,1,1,'admin','a@b.com',1262304000+3600*3,'Topic number 03','Body of topic 03.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (103,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,3,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (104,1,1,'admin','a@b.com',1262304000+3600*4,'Topic number 04','Body of topic 04.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (104,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,4,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (105,1,1,'admin','a@b.com',1262304000+3600*5,'Topic number 05','Body of topic 05.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (105,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,5,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (106,1,1,'admin','a@b.com',1262304000+3600*6,'Topic number 06','Body of topic 06.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (106,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,6,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (107,1,1,'admin','a@b.com',1262304000+3600*7,'Topic number 07','Body of topic 07.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (107,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,7,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (108,1,1,'admin','a@b.com',1262304000+3600*8,'Topic number 08','Body of topic 08.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (108,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,8,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (109,1,1,'admin','a@b.com',1262304000+3600*9,'Topic number 09','Body of topic 09.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (109,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,9,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (110,1,1,'admin','a@b.com',1262304000+3600*10,'Topic number 10','Body of topic 10.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (110,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,10,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (111,1,1,'admin','a@b.com',1262304000+3600*11,'Topic number 11','Body of topic 11.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (111,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,11,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (112,1,1,'admin','a@b.com',1262304000+3600*12,'Topic number 12','Body of topic 12.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (112,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,12,0,0);

-- BBC-heavy topic (id 200), message id 14. posterTime epoch+3600*13.
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (200,1,1,'admin','a@b.com',1262304000+3600*13,'BBC showcase','[quote author=admin]nested [b]quote[/b][/quote]\n[code]if (x) { return; }[/code]\n[list][li]one[/li][li]two[/li][/list]\n[url=http://example.com]link[/url] [color=red]red[/color]\n[table][tr][td]a[/td][td]b[/td][/tr][/table]','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (200,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,7,0,0);

-- Locked + sticky topic (id 201), message id 15. posterTime epoch+3600*14.
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (201,1,1,'admin','a@b.com',1262304000+3600*14,'Announcement','Pinned and locked.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (201,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,2,1,1);

-- Long topic (id 202), first message id 16 (posterTime epoch+3600*15), then 12 replies (ids 17..28, posterTime epoch+3600*16..27).
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'admin','a@b.com',1262304000+3600*15,'Long thread','First post.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (202,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,0,9,0,0);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*16,'Re: Long thread','Reply 01.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*17,'Re: Long thread','Reply 02.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*18,'Re: Long thread','Reply 03.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*19,'Re: Long thread','Reply 04.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*20,'Re: Long thread','Reply 05.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*21,'Re: Long thread','Reply 06.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*22,'Re: Long thread','Reply 07.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*23,'Re: Long thread','Reply 08.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*24,'Re: Long thread','Reply 09.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*25,'Re: Long thread','Reply 10.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*26,'Re: Long thread','Reply 11.','xx');
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (202,1,1,'member','m@b.com',1262304000+3600*27,'Re: Long thread','Reply 12.','xx');
UPDATE smf_topics SET ID_LAST_MSG = (SELECT MAX(ID_MSG) FROM smf_messages WHERE ID_TOPIC=202), ID_MEMBER_UPDATED = 0, numReplies = 12 WHERE ID_TOPIC = 202;

-- Poll (id 1) + choices, and poll topic (id 203), message id 29. posterTime epoch+3600*28.
INSERT INTO smf_polls (ID_POLL, question, votingLocked, maxVotes, expireTime, hideResults, changeVote, ID_MEMBER, posterName) VALUES (1, 'Favorite color?', 0, 1, 0, 0, 0, 1, 'admin');
INSERT INTO smf_poll_choices (ID_POLL, ID_CHOICE, label, votes) VALUES (1, 0, 'Red', 2), (1, 1, 'Green', 5), (1, 2, 'Blue', 3);
INSERT INTO smf_messages (ID_TOPIC, ID_BOARD, ID_MEMBER, posterName, posterEmail, posterTime, subject, body, icon) VALUES (203,1,1,'admin','a@b.com',1262304000+3600*28,'Vote here','Cast your vote.','xx');
INSERT INTO smf_topics (ID_TOPIC, ID_BOARD, ID_FIRST_MSG, ID_LAST_MSG, ID_MEMBER_STARTED, ID_MEMBER_UPDATED, ID_POLL, numViews, locked, isSticky) VALUES (203,1,LAST_INSERT_ID(),LAST_INSERT_ID(),1,1,1,4,0,0);

-- PM (id 1) from admin to member. msgtime epoch+3600*29.
INSERT INTO smf_personal_messages (ID_PM, ID_MEMBER_FROM, deletedBySender, fromName, msgtime, subject, body) VALUES (1, 1, 0, 'admin', 1262304000+3600*29, 'Hello there', 'A private message body.');
INSERT INTO smf_pm_recipients (ID_PM, ID_MEMBER, labels, bcc, is_read, deleted) VALUES (1, 2, '-1', 0, 0, 0);

-- Moderation-log entry.
INSERT INTO smf_log_actions (logTime, ID_MEMBER, ip, action, extra) VALUES (1262304000, 1, '127.0.0.1', 'remove', 'a:5:{s:5:"board";i:1;s:10:"board_name";s:18:"General Discussion";s:6:"member";s:5:"admin";s:7:"subject";s:15:"Topic number 01";s:5:"topic";i:101;}');

-- Fix-ups: topic first/last/reply counts, then board totals (matches goldenApp).
UPDATE smf_topics t SET
  ID_FIRST_MSG = (SELECT MIN(ID_MSG) FROM smf_messages m WHERE m.ID_TOPIC = t.ID_TOPIC),
  ID_LAST_MSG  = (SELECT MAX(ID_MSG) FROM smf_messages m WHERE m.ID_TOPIC = t.ID_TOPIC),
  numReplies   = (SELECT COUNT(*) - 1 FROM smf_messages m WHERE m.ID_TOPIC = t.ID_TOPIC);

UPDATE smf_boards b SET
  numTopics = (SELECT COUNT(*) FROM smf_topics t WHERE t.ID_BOARD = b.ID_BOARD),
  numPosts  = (SELECT COUNT(*) FROM smf_messages m WHERE m.ID_BOARD = b.ID_BOARD),
  ID_LAST_MSG    = COALESCE((SELECT MAX(ID_MSG) FROM smf_messages m WHERE m.ID_BOARD = b.ID_BOARD), 0),
  ID_MSG_UPDATED = COALESCE((SELECT MAX(ID_MSG) FROM smf_messages m WHERE m.ID_BOARD = b.ID_BOARD), 0);
