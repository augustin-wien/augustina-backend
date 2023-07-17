CREATE TABLE Accounts (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL
);

CREATE TABLE PaymentTypes (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL
);

CREATE TABLE Payments (
    ID bigserial PRIMARY KEY,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Sender integer NOT NULL REFERENCES Accounts,
    Receiver integer NOT NULL REFERENCES Accounts,
    Type integer NOT NULL REFERENCES PaymentTypes,
    Amount real NOT NULL
);

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
