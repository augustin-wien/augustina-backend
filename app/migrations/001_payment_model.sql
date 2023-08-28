-- First migration file for augustin backend, corresponds to v0.0.1

CREATE TABLE Vendor (
    ID serial PRIMARY KEY,
    KeycloakID varchar(255) NOT NULL DEFAULT '',
    UrlID varchar(255) NOT NULL DEFAULT '',
    LicenseID varchar(255) NOT NULL DEFAULT '',
    FirstName varchar(255) NOT NULL DEFAULT '',
    LastName varchar(255) NOT NULL DEFAULT '',
    Email varchar(255) NOT NULL DEFAULT '',
    LastPayout timestamp
);

CREATE TYPE AccountType AS ENUM ('', 'user_auth', 'user_anon', 'vendor', 'orga', 'cash');  -- user_anon and cash should only exist once

CREATE TABLE Account (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL DEFAULT '',
    Balance real NOT NULL DEFAULT 0,
    Type AccountType NOT NULL DEFAULT '',
    UserID UUID,  -- Keycloak UUID if type is user_auth
    Vendor integer REFERENCES Vendor ON DELETE SET NULL
);

CREATE TABLE Item (
    ID serial PRIMARY KEY,
    Name varchar(255) UNIQUE NOT NULL,
    Description varchar(255),
    Price integer NOT NULL DEFAULT 0,  -- Price in cents
    Image varchar(255) NOT NULL DEFAULT '',
    LicenseItem integer REFERENCES Item,  -- Required to be bought first
    Archived bool NOT NULL DEFAULT FALSE
);

CREATE TABLE PaymentOrder (
    ID bigserial PRIMARY KEY,
    TransactionID integer NOT NULL DEFAULT 0,
    Verified bool NOT NULL DEFAULT FALSE,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UserID UUID,  -- Keycloak UUID if user is authenticated
    Vendor integer REFERENCES Vendor
);

CREATE TABLE OrderEntry (
    ID bigserial PRIMARY KEY,
    Item integer REFERENCES Item,
    Price integer NOT NULL DEFAULT 0,  -- Price at time of purchase in cents
    Quantity integer NOT NULL DEFAULT 0,
    PaymentOrder integer REFERENCES PaymentOrder,
    Sender integer NOT NULL REFERENCES Account,
    Receiver integer NOT NULL REFERENCES Account
);

CREATE TABLE Payment (
    ID bigserial PRIMARY KEY,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Sender integer NOT NULL REFERENCES Account,
    Receiver integer NOT NULL REFERENCES Account,
    Amount integer NOT NULL,  -- Price in cents
    AuthorizedBy varchar(255) NOT NULL DEFAULT '',
    PaymentOrder integer REFERENCES PaymentOrder,
    OrderEntry integer REFERENCES OrderEntry
);

CREATE TABLE Settings (
    ID integer UNIQUE PRIMARY KEY CHECK (ID = 1) DEFAULT 1,
    Color varchar(255) NOT NULL DEFAULT '',
    Logo varchar(255) NOT NULL DEFAULT '',
    MainItem integer REFERENCES Item,
    RefundFees bool NOT NULL DEFAULT FALSE
);


CREATE OR REPLACE FUNCTION truncate_tables(username IN VARCHAR) RETURNS void AS $$
DECLARE
    statements CURSOR FOR
        SELECT tablename FROM pg_tables
        WHERE tableowner = username AND schemaname = 'public' AND tablename != 'schema_version';
BEGIN
    FOR stmt IN statements LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(stmt.tablename) || ' CASCADE;';
    END LOOP;
END;
$$ LANGUAGE plpgsql;

---- create above / drop below ----


DROP TABLE Vendor, Account, Item, PaymentOrder, OrderItem, Payment, Settings;
DROP TYPE AccountType;
