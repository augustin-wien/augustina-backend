-- Migrate working_time from text codes ('v','n','g') to structured JSON (jsonb)

BEGIN;

-- 1) Add temporary jsonb column
ALTER TABLE locations ADD COLUMN working_time_tmp jsonb;

-- 2) Populate tmp column based on old text codes
UPDATE locations SET working_time_tmp =
  CASE
    WHEN working_time = 'v' THEN
      (
        jsonb_build_object('mode', 'everyday', 'everyday', jsonb_build_array(jsonb_build_object('from','08:00','to','12:00')))
      )
    WHEN working_time = 'n' THEN
      (
        jsonb_build_object('mode', 'everyday', 'everyday', jsonb_build_array(jsonb_build_object('from','13:00','to','17:00')))
      )
    WHEN working_time = 'g' THEN
      (
        jsonb_build_object('mode', 'everyday', 'everyday', jsonb_build_array(jsonb_build_object('full_day', true)))
      )
    WHEN working_time IS NULL OR trim(working_time) = '' THEN NULL
    ELSE
      (
        -- unknown codes: keep them in the mode for inspection
        jsonb_build_object('mode', working_time)
      )
  END
  WHERE working_time IS NOT NULL;

-- 3) Drop old column and rename tmp
ALTER TABLE locations DROP COLUMN working_time;
ALTER TABLE locations RENAME COLUMN working_time_tmp TO working_time;

COMMIT;

-- Down migration: convert json back to text codes when possible
-- Note: irreversible for unknown/custom values; we attempt best-effort mapping
-- To roll back manually, inspect the JSON values and choose appropriate text codes.
