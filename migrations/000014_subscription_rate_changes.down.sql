--
-- Removes the database changes required to support plan rate and default quota changes.
--

BEGIN;

SET search_path = public, pg_catalog;

-- Drop the plan_rates table.
DROP TABLE IF EXISTS plan_rates;

-- Add the plan_quota_defaults unique key on plan_id and resource_type_id.
CREATE UNIQUE INDEX IF NOT EXISTS plan_quota_defaults_resource_type_plan_index
ON plan_quota_defaults (resource_type_id, plan_id);

COMMIT;
