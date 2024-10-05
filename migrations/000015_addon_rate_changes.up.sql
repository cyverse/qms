--
-- Makes the database changes required to support addon rate and default quota changes.
--

BEGIN;

SET search_path = public, pg_catalog;

-- Create the addon_rates table.
CREATE TABLE IF NOT EXISTS addon_rates (
    id uuid NOT NULL default uuid_generate_v1(),
    addon_id uuid NOT NULL,
    effective_date timestamp WITH time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rate money NOT NULL,
    FOREIGN KEY (addon_id) REFERENCES addons(id) ON DELETE CASCADE,
    PRIMARY KEY (id)
);

-- Populate the addon_rates table. This could cause a problem if this isn't the exact set of addons, but the chances of
-- that happening in any DE deployment at this time are small because the subscription admin pages don't currently have
-- a feature to add or remove addons.
INSERT INTO addon_rates (id, addon_id, effective_date, rate) VALUES
('d612d958-82ad-11ef-b4b2-5a8d7f4f1112', 'c21dd61f-aa41-40ad-8005-859679ceed9c', '2022-01-01', 125.00)
ON CONFLICT (id) DO NOTHING;

COMMIT;
