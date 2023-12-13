-- Write your migrate up statements here

ALTER TABLE OrderEntry ADD COLUMN CustomerEmail varchar(255);


---- create above / drop below ----

ALTER TABLE OrderEntry DROP COLUMN CustomerEmail;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
