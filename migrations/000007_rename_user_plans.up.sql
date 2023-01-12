BEGIN;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS user_plans RENAME TO subscriptions;
ALTER TRIGGER user_plans_last_modified_at_trigger ON TABLE subscriptions RENAME TO subscriptions_last_modified_at_trigger;
ALTER TRIGGER user_plans_last_modified_by_trigger ON TABLE subscriptions RENAME TO subscriptions_last_modified_by_trigger;
ALTER TRIGGER user_plans_last_modified_by_insert_trigger ON TABLE subscriptions RENAME TO subscriptions_last_modified_by_insert_trigger;
ALTER TRIGGER user_plans_created_by_trigger ON TABLE subscriptions RENAME TO subscriptions_created_by_trigger;

ALTER TABLE IF EXISTS quotas RENAME COLUMN user_plan_id TO subscription_id;
ALTER INDEX IF EXISTS quotas_resource_type_user_plan_index RENAME TO quotas_resource_type_subscription_index;

ALTER TABLE IF EXISTS usages RENAME COLUMN user_plan_id to subscription_id;
ALTER INDEX IF EXISTS usages_resource_type_user_plan_index RENAME TO usages_resource_type_subscription_index;

COMMIT;