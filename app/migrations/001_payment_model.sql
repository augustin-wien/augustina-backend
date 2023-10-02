-- First migration file for augustin backend, corresponds to v0.0.1

CREATE TABLE Vendor (
    ID serial PRIMARY KEY,
    KeycloakID varchar(255) NOT NULL DEFAULT '',
    UrlID varchar(255) NOT NULL DEFAULT '',
    LicenseID varchar(255) UNIQUE,
    FirstName varchar(255) NOT NULL DEFAULT '',
    LastName varchar(255) NOT NULL DEFAULT '',
    Email varchar(255) NOT NULL DEFAULT '',
    LastPayout timestamp,
    IsDisabled bool NOT NULL DEFAULT FALSE,
    Longitude double precision NOT NULL DEFAULT 0,
    Latitude double precision NOT NULL DEFAULT 0,
    Address varchar(255) NOT NULL DEFAULT '',
    PLZ varchar(255) NOT NULL DEFAULT '',
    Location varchar(255) NOT NULL DEFAULT '',
    WorkingTime varchar(1) NOT NULL DEFAULT '',
    Lang varchar(255) NOT NULL DEFAULT '',
    Comment text NOT NULL DEFAULT '',
    Telephone varchar(255) NOT NULL DEFAULT '',
    RegistrationDate varchar(255) NOT NULL DEFAULT '',
    VendorSince varchar(255) NOT NULL DEFAULT '',
    OnlineMap bool NOT NULL DEFAULT FALSE,
    HasSmartphone bool NOT NULL DEFAULT FALSE
);

CREATE TYPE AccountType AS ENUM ('', 'UserAuth', 'UserAnon', 'Vendor', 'Orga', 'Cash', 'Paypal', 'VivaWallet');  -- UserAnon, Orga, and Cash, VivaWallet, Paypal should only exist once

CREATE TABLE Account (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL DEFAULT '',
    Balance real NOT NULL DEFAULT 0,
    Type AccountType NOT NULL DEFAULT '',
    UserID UUID UNIQUE,  -- Keycloak UUID if type is user_auth
    Vendor integer UNIQUE REFERENCES Vendor ON DELETE SET NULL
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
    OrderCode varchar(255) UNIQUE NOT NULL,  -- Generated by the backend
    TransactionID varchar(255) NOT NULL DEFAULT '',  -- Sent from the frontend
    Verified bool NOT NULL DEFAULT FALSE,
    TransactionTypeID integer NOT NULL DEFAULT -1,  -- value -1 not in list and hopefully never will be, param list here: https://developer.vivawallet.com/integration-reference/response-codes/#transactiontypeid-parameter
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
    Receiver integer NOT NULL REFERENCES Account,
    IsSale bool NOT NULL DEFAULT FALSE
);

CREATE TABLE Payment (
    ID bigserial PRIMARY KEY,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Sender integer NOT NULL REFERENCES Account,
    Receiver integer NOT NULL REFERENCES Account,
    Amount integer NOT NULL,  -- Price in cents
    AuthorizedBy varchar(255) NOT NULL DEFAULT '',
    PaymentOrder integer REFERENCES PaymentOrder,
    OrderEntry integer REFERENCES OrderEntry,
    IsSale bool NOT NULL DEFAULT FALSE
);

CREATE TABLE Settings (
    ID integer UNIQUE PRIMARY KEY CHECK (ID = 1) DEFAULT 1,
    Color varchar(255) NOT NULL DEFAULT '#008000',
    FontColor varchar(255) NOT NULL DEFAULT '#FFFFFF',
    Logo varchar(255) NOT NULL DEFAULT 'img/logo.png',
    MainItem integer REFERENCES Item,
    MaxOrderAmount integer NOT NULL DEFAULT 10000,  -- Default value is 10000, which equals 100€
    OrgaCoversTransactionCosts bool NOT NULL DEFAULT TRUE
);

CREATE TABLE DBsettings (
    id integer PRIMARY KEY CHECK (ID = 1) DEFAULT 1,
    isInitialized bool NOT NULL DEFAULT FALSE
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


DROP TABLE Vendor, Account, Item, PaymentOrder, OrderItem, Payment, Settings, DBsettings;
DROP TYPE AccountType;
