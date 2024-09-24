-- Write your migrate up statements here

ALTER TABLE Item ADD COLUMN ItemOrder INT NOT NULL DEFAULT 0;
ALTER TABLE Item ADD COLUMN ItemColor Text DEFAULT NULL;


---- create above / drop below ----

ALTER TABLE Item DROP COLUMN ItemOrder;
ALTER TABLE Item DROP COLUMN ItemColor;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
