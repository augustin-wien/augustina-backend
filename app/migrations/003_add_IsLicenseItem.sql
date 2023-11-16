-- Write your migrate up statements here

ALTER TABLE Item ADD COLUMN IsLicenseItem boolean DEFAULT false;


---- create above / drop below ----

ALTER TABLE Item DROP COLUMN IsLicenseItem;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
