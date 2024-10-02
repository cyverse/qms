--
-- Makes the database changes required to support plan rate and default quota changes.
--

BEGIN;

SET search_path = public, pg_catalog;

-- Add the new plan_rates table.
CREATE TABLE IF NOT EXISTS plan_rates (
    id uuid NOT NULL DEFAULT uuid_generate_v1(),
    plan_id uuid NOT NULL,
    effective_date timestamp WITH time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rate money NOT NULL,
    FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE,
    PRIMARY KEY (id)
);

-- Drop the plan_quota_defaults unique key on plan_id and resource_type_id.
DROP INDEX IF EXISTS plan_quota_defaults_resource_type_plan_index;

COMMIT;
