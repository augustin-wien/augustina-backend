CREATE TABLE Accounts (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL
);

CREATE TABLE PaymentTypes (
    ID serial PRIMARY KEY,
    Name varchar(255) NOT NULL
);

-- TODO: All NOT NULL
CREATE TABLE Payments (
    ID bigserial PRIMARY KEY,
    Timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    Sender integer REFERENCES Accounts,
    Receiver integer REFERENCES Accounts,
    Type integer REFERENCES PaymentTypes,
    Amount real NOT NULL
);

---- create above / drop below ----

DROP TABLE Payments, PaymentTypes, Accounts;
