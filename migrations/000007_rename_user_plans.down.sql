BEGIN;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS subscriptions RENAME TO user_plans;

COMMIT;