-- Write your migrate up statements here

-- Default value is 10000, which equals 100€
ALTER TABLE Settings
ADD COLUMN MaxOrderAmount integer DEFAULT 10000;

---- create above / drop below ----

ALTER TABLE Settings
DROP COLUMN MaxOrderAmount;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
