-- Write your migrate up statements here
CREATE TABLE PDF (
    ID serial PRIMARY KEY,
    Path varchar(255) UNIQUE NOT NULL DEFAULT '',
    Timestamp timestamp DEFAULT CURRENT_TIMESTAMP
);
---- create above / drop below ----
DROP TABLE IF EXISTS PDF;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
