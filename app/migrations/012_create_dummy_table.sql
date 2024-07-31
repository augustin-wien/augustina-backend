-- Write your migrate up statements here

CREATE TABLE Dummy2 (
    ID SERIAL PRIMARY KEY
)

---- create above / drop below ----

DROP TABLE Dummy2

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
