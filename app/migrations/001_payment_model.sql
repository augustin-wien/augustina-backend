CREATE TABLE Person (
    ID serial PRIMARY KEY,
    KeycloakID varchar(255) UNIQUE,
    UrlID varchar(255) UNIQUE,
    LicenseID varchar(255) UNIQUE,
    FirstName varchar(255) NOT NULL,
    LastName varchar(255) NOT NULL DEFAULT '',
    IsVendor boolean NOT NULL DEFAULT FALSE,
    IsAdmin boolean NOT NULL DEFAULT FALSE
);

CREATE TABLE Account (
    ID serial PRIMARY KEY,
    Name varchar(255),
    Person integer REFERENCES Person
);

CREATE TABLE Item (
    ID serial PRIMARY KEY,
    Name varchar(255) UNIQUE NOT NULL,
    Description varchar(255),
    Price real NOT NULL DEFAULT 0,
    Image varchar(255)
);

CREATE TABLE PaymentType (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL
);

CREATE TABLE PaymentBatch (
    ID bigserial PRIMARY KEY
);

CREATE TABLE Payment (
    ID bigserial PRIMARY KEY,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Sender integer NOT NULL REFERENCES Account,
    Receiver integer NOT NULL REFERENCES Account,
    Type integer NOT NULL REFERENCES PaymentType,
    Amount real NOT NULL,
    AuthorizedBy integer REFERENCES Person,
    Item integer REFERENCES Item,
    PaymentBatch integer REFERENCES PaymentBatch
);

CREATE TABLE Settings (
    ID integer UNIQUE PRIMARY KEY CHECK (ID = 1) DEFAULT 1,
    Color varchar(255),
    Logo varchar(255),
    Newspaper integer REFERENCES Item,
    RefundFees bool NOT NULL DEFAULT FALSE
);

CREATE OR REPLACE FUNCTION truncate_tables(username IN VARCHAR) RETURNS void AS $$
DECLARE
    statements CURSOR FOR
        SELECT tablename FROM pg_tables
        WHERE tableowner = username AND schemaname = 'public';
BEGIN
    FOR stmt IN statements LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(stmt.tablename) || ' CASCADE;';
    END LOOP;
END;
$$ LANGUAGE plpgsql;

---- create above / drop below ----


DROP TABLE Person, Account, Item, PaymentType, PaymentBatch, Payment, Settings;
