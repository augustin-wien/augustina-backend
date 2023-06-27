CREATE TABLE Settings (
    ID integer UNIQUE PRIMARY KEY CHECK (ID = 1) DEFAULT 1,
    Color varchar(255),
    Logo varchar(255)
);

CREATE TABLE Items (
    ID serial PRIMARY KEY,
    Name varchar(255) UNIQUE NOT NULL,
    Price real NOT NULL
);

---- create above / drop below ----

DROP TABLE Settings, Items;
