-- Migration: 020_working_times_from_locations.sql
-- Create `working_times` table and backfill from `locations.working_time`.
BEGIN;

CREATE TABLE IF NOT EXISTS working_times (
    id SERIAL PRIMARY KEY,
    day TEXT DEFAULT 'monday',
    open_time TEXT,
    close_time TEXT,
    closed BOOLEAN DEFAULT false,
    location_working_times INTEGER REFERENCES locations(id)
);

-- Backfill: copy existing `working_time` text into a single working_times row per location.
INSERT INTO working_times (day, open_time, close_time, closed, location_working_times)
SELECT 'monday', working_time, NULL, false, id
FROM locations
WHERE working_time IS NOT NULL AND working_time <> '';

-- Note: we keep the original `locations.working_time` column for compatibility.
COMMIT;
