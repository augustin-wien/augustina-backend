-- Write your migrate up statements here

ALTER TABLE Vendor ADD COLUMN IsDeleted bool NOT NULL DEFAULT FALSE;
ALTER TABLE Vendor ADD COLUMN AccountProofUrl Text;


---- create above / drop below ----

ALTER TABLE Vendor DROP COLUMN IsDeleted;
ALTER TABLE Vendor DROP COLUMN AccountProofUrl;



-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
