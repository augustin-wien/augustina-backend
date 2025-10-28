-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN digitalitemsurl varchar(255) NOT NULL DEFAULT 'https://augustina.cc/digital-items';

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN digitalitemsurl;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
