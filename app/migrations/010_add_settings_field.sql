-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN WebshopIsClosed Boolean DEFAULT FALSE;

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN WebshopIsClosed;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
