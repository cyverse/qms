BEGIN;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS subscriptions RENAME TO user_plans;
ALTER TRIGGER IF EXISTS subscriptions_last_modified_at_trigger ON TABLE user_plans RENAME TO user_plans_last_modified_at_trigger;
ALTER TRIGGER IF EXISTS subscriptions_last_modified_by_trigger ON TABLE user_plans RENAME TO user_plans_last_modified_by_trigger;
ALTER TRIGGER IF EXISTS subscriptions_last_modified_by_insert_trigger ON TABLE user_plans RENAME TO user_plans_last_modified_by_insert_trigger;
ALTER TRIGGER IF EXISTS subscriptions_created_by_trigger ON TABLE user_plans RENAME TO user_plans_created_by_trigger;

COMMIT;