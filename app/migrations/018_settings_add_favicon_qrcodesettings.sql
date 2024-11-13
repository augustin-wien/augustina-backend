-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN Favicon text NOT NULL DEFAULT '/img/favicon.png';
ALTER TABLE Settings ADD COLUMN QrCodeSettings text NOT NULL DEFAULT '';

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN Favicon;
ALTER TABLE Settings DROP COLUMN QrCodeSettings;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
