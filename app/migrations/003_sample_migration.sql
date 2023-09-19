-- Write your migrate up statements here

ALTER TABLE Settings
ADD COLUMN MoneyLimit integer DEFAULT 50;

---- create above / drop below ----

ALTER TABLE Settings
DROP COLUMN MoneyLimit;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
