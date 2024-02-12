--
-- Makes the database changes required to support multi-year subscriptions.
--

BEGIN;

SET search_path = public, pg_catalog;

ALTER TABLE resource_types ADD IF NOT EXISTS expendable boolean DEFAULT FALSE;

UPDATE resource_types SET expendable = TRUE WHERE "name" = 'cpu.hours';
UPDATE resource_types SET expendable = FALSE WHERE "name" = 'data.size';

COMMIT;
