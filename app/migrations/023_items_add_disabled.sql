-- Add a per-item disabled flag to the Item table
-- Up: add column `Disabled` (boolean) default false
ALTER TABLE Item ADD COLUMN Disabled boolean DEFAULT false;


---- create above / drop below ----

-- Down: remove the `Disabled` column
ALTER TABLE Item DROP COLUMN Disabled;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
