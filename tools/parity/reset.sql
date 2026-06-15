-- Reset volatile "most online" + activity state to goldenApp's baseline so the
-- crawl's first board-index view recomputes mostOnlineToday deterministically.
DELETE FROM smf_settings WHERE variable IN ('mostOnlineUpdated');
UPDATE smf_settings SET value='5' WHERE variable='mostOnline';
UPDATE smf_settings SET value='5' WHERE variable='mostOnlineToday';
UPDATE smf_settings SET value='1262304000' WHERE variable='mostDate';
DELETE FROM smf_log_activity;
DELETE FROM smf_log_online;
DELETE FROM smf_log_errors;
-- Reset numViews to goldenApp seed values (views accumulate across crawl runs).
UPDATE smf_topics SET numViews=0  WHERE ID_TOPIC=1;
UPDATE smf_topics SET numViews=ID_TOPIC-100 WHERE ID_TOPIC BETWEEN 101 AND 112;
UPDATE smf_topics SET numViews=7  WHERE ID_TOPIC=200;
UPDATE smf_topics SET numViews=2  WHERE ID_TOPIC=201;
UPDATE smf_topics SET numViews=9  WHERE ID_TOPIC=202;
UPDATE smf_topics SET numViews=4  WHERE ID_TOPIC=203;
-- Clear per-user read/notify state (accumulates across crawl runs); the Go test
-- starts fresh so members have read/notified nothing.
DELETE FROM smf_log_topics;
DELETE FROM smf_log_mark_read;
DELETE FROM smf_log_boards;
DELETE FROM smf_log_notify;
-- Pin admin's now-seeded timestamps + the welcome message to the fixed epoch so
-- date-only columns (memberlist registration) and "Today" rendering don't drift.
UPDATE smf_members SET dateRegistered = 1262304000, lastLogin = 1262304000 WHERE ID_MEMBER = 1;
UPDATE smf_messages SET posterTime = 1262304000 WHERE ID_MSG = 1;
