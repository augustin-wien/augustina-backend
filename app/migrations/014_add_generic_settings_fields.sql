-- Write your migrate up statements here

ALTER TABLE Settings ADD COLUMN VendorNotFoundHelpUrl Text NOT NULL DEFAULT 'https://example.com/vendor-not-found';
ALTER TABLE Settings ADD COLUMN MaintainanceModeHelpUrl Text NOT NULL DEFAULT 'https://example.com/maintenance';
ALTER TABLE Settings ADD COLUMN VendorEmailPostfix Text NOT NULL DEFAULT '@example.com';
ALTER TABLE Settings ADD COLUMN NewspaperName Text NOT NULL DEFAULT 'Example Newspaper';
ALTER TABLE Settings ADD COLUMN QRCodeUrl Text NOT NULL DEFAULT 'https://example.com/qrcode';
ALTER TABLE Settings ADD COLUMN QRCodeLogoImgUrl Text;
ALTER TABLE Settings ADD COLUMN AGBUrl Text NOT NULL DEFAULT 'https://example.com/agb';
ALTER TABLE Settings ADD COLUMN MapCenterLat double precision NOT NULL DEFAULT 48.2083;
ALTER TABLE Settings ADD COLUMN MapCenterLong double precision NOT NULL DEFAULT 16.3731;

---- create above / drop below ----

ALTER TABLE Settings DROP COLUMN VendorNotFoundHelpUrl;
ALTER TABLE Settings DROP COLUMN MaintainanceModeHelpUrl;
ALTER TABLE Settings DROP COLUMN VendorEmailPostfix;
ALTER TABLE Settings DROP COLUMN NewspaperName;
ALTER TABLE Settings DROP COLUMN QRCodeUrl;
ALTER TABLE Settings DROP COLUMN QRCodeLogoImgUrl;
ALTER TABLE Settings DROP COLUMN AGBUrl;
ALTER TABLE Settings DROP COLUMN MapCenterLat;
ALTER TABLE Settings DROP COLUMN MapCenterLong;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
