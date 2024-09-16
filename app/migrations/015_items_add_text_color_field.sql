-- Write your migrate up statements here

ALTER TABLE Item ADD COLUMN ItemTextColor Text DEFAULT NULL;


---- create above / drop below ----

ALTER TABLE Item DROP COLUMN ItemTextColor;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
