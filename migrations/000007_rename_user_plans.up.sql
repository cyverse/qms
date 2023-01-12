BEGIN;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS user_plans RENAME TO subscriptions;

COMMIT;