DROP INDEX IF EXISTS notification.idx_notifications_player;

DROP INDEX IF EXISTS notification.idx_notifications_unread;

DROP TABLE IF EXISTS notification.notifications CASCADE;

DROP SCHEMA IF EXISTS notification CASCADE;
