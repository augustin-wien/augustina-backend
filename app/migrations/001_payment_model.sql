-- First migration file for augustin backend, corresponds to v0.0.1
-- User cannot be used as a table name, so we use UserAccount instead

CREATE TABLE Account (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL DEFAULT '',
    Balance real NOT NULL DEFAULT 0
);

CREATE TABLE Vendor (
    ID serial PRIMARY KEY,
    KeycloakID varchar(255) NOT NULL DEFAULT '',
    UrlID varchar(255) NOT NULL DEFAULT '',
    LicenseID varchar(255) NOT NULL DEFAULT '',
    FirstName varchar(255) NOT NULL DEFAULT '',
    LastName varchar(255) NOT NULL DEFAULT '',
    Email varchar(255) NOT NULL DEFAULT '',
    LastPayout timestamp,
    Account integer UNIQUE REFERENCES Account NOT NULL
);

CREATE TABLE Item (
    ID serial PRIMARY KEY,
    Name varchar(255) UNIQUE NOT NULL,
    Description varchar(255),
    Price real NOT NULL DEFAULT 0,
    Image varchar(255)
);

CREATE TABLE Order (
    ID bigserial PRIMARY KEY,
    TransactionID varchar(255) NOT NULL DEFAULT '',
    TransactionVerified bool NOT NULL DEFAULT FALSE,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Vendor integer REFERENCES Vendor,
)

CREATE TABLE OrderItem (
    ID bigserial PRIMARY KEY,
    Item integer REFERENCES Item,
    Quantity real NOT NULL DEFAULT 0
    Order integer REFERENCES Order
);

CREATE TABLE Payment (
    ID bigserial PRIMARY KEY,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Sender integer NOT NULL REFERENCES Account,
    Receiver integer NOT NULL REFERENCES Account,
    Amount real NOT NULL,
    AuthorizedBy varchar(255) NOT NULL DEFAULT '',
    Item integer REFERENCES Item,
    Order integer REFERENCES Order
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


DROP TABLE Vendor, Account, Item, Order, OrderItem, Payment, Settings;
DROP TYPE AccountType;
