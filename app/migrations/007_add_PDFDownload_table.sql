-- Write your migrate up statements here
CREATE TABLE PDFDownload (
    ID serial PRIMARY KEY,
    PDF integer REFERENCES PDF(ID),
    LinkID varchar(255) UNIQUE NOT NULL DEFAULT '',
    Timestamp timestamp DEFAULT CURRENT_TIMESTAMP
);
---- create above / drop below ----
DROP TABLE IF EXISTS PDFDownload;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
