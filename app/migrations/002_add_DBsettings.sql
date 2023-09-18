-- Write your migrate up statements here
CREATE TABLE DBsettings (
    id integer PRIMARY KEY CHECK (ID = 1) DEFAULT 1,
    isInitialized bool NOT NULL DEFAULT FALSE
);

---- create above / drop below ----
DROP TABLE DBsettings;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
