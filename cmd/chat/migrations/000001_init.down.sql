DROP INDEX IF EXISTS chat.idx_messages_channel;

DROP INDEX IF EXISTS chat.idx_pm_receiver;

DROP INDEX IF EXISTS chat.idx_pm_sender;

DROP TABLE IF EXISTS chat.private_messages CASCADE;

DROP TABLE IF EXISTS chat.messages CASCADE;

DROP SCHEMA IF EXISTS chat CASCADE;
