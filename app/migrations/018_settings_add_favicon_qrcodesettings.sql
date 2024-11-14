-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN Favicon text NOT NULL DEFAULT '/img/favicon.png';
ALTER TABLE Settings ADD COLUMN QRCodeSettings text NOT NULL DEFAULT '';
ALTER TABLE Settings ADD COLUMN QRCodeEnableLogo boolean NOT NULL DEFAULT false;
ALTER TABLE Settings ALTER COLUMN QRCodeLogoImgUrl SET NOT NULL;
ALTER TABLE Settings ALTER COLUMN QRCodeLogoImgUrl SET DEFAULT '';

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN Favicon;
ALTER TABLE Settings DROP COLUMN QRCodeSettings;
ALTER TABLE Settings DROP COLUMN QRCodeEnableLogo;
ALTER TABLE Settings ALTER COLUMN QRCodeLogoImgUrl DROP NOT NULL;
ALTER TABLE Settings ALTER COLUMN QRCodeLogoImgUrl SET DEFAULT NULL;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
