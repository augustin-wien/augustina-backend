-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN UseVendorLicenseIdInShop bool NOT NULL DEFAULT FALSE;

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN UseVendorLicenseIdInShop;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
