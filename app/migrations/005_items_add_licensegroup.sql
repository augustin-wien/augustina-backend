-- Write your migrate up statements here

ALTER TABLE Item ADD COLUMN LicenseGroup varchar(255);


---- create above / drop below ----

ALTER TABLE Item DROP COLUMN LicenseGroup;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
