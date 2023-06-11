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

---- create above / drop below ----

DROP TABLE Payments, PaymentTypes, Accounts;
