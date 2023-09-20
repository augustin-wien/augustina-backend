-- Write your migrate up statements here

ALTER TABLE Settings
ADD COLUMN MaxOrderAmount integer 5000;

---- create above / drop below ----

ALTER TABLE Settings
DROP COLUMN MaxOrderAmount;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
