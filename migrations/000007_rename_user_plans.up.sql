BEGIN;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS user_plans RENAME TO subscriptions;
ALTER TRIGGER IF EXISTS user_plans_last_modified_at_trigger ON TABLE subscriptions RENAME TO subscriptions_last_modified_at_trigger;
ALTER TRIGGER IF EXISTS user_plans_last_modified_by_trigger ON TABLE subscriptions RENAME TO subscriptions_last_modified_by_trigger;
ALTER TRIGGER IF EXISTS user_plans_last_modified_by_insert_trigger ON TABLE subscriptions RENAME TO subscriptions_last_modified_by_insert_trigger;
ALTER TRIGGER IF EXISTS user_plans_created_by_trigger ON TABLE subscriptions RENAME TO subscriptions_created_by_trigger;

COMMIT;