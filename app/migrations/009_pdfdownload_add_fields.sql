-- Write your migrate up statements here

ALTER TABLE PDFDownload ADD COLUMN EmailSent Boolean DEFAULT FALSE;
ALTER TABLE PDFDownload ADD COLUMN OrderID integer REFERENCES paymentorder(ID) ON DELETE SET NULL;
ALTER TABLE PDFDownload ADD COLUMN LastDownload timestamp DEFAULT NULL;
ALTER TABLE PDFDownload ADD COLUMN DownloadCount integer DEFAULT 0;
ALTER TABLE PDFDownload ADD COLUMN ItemID integer REFERENCES item(ID) ON DELETE SET NULL;

---- create above / drop below ----

ALTER TABLE PDFDownload DROP COLUMN EmailSent;
ALTER TABLE PDFDownload DROP COLUMN OrderID;
ALTER TABLE PDFDownload DROP COLUMN LastDownload;
ALTER TABLE PDFDownload DROP COLUMN DownloadCount;
ALTER TABLE PDFDownload DROP COLUMN ItemID;


-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
