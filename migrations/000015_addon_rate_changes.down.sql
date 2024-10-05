--
-- Removes the database changes required to support addon rate and default quota changes.
--

BEGIN;

SET search_path = public, pg_catalog;

-- Drop the addon_rates table.
DROP TABLE IF EXISTS addon_rates;

COMMIT;
