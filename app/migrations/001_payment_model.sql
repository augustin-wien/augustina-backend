CREATE TABLE Accounts (
    id integer PRIMARY KEY
);

CREATE TABLE PaymentTypes (
    id serial PRIMARY KEY,
    name varchar(255) NOT NULL
);

CREATE TABLE Payments (
    id serial PRIMARY KEY,
    timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sender integer REFERENCES Accounts,
    receiver integer REFERENCES Accounts,
    type integer REFERENCES PaymentTypes,
    amount real NOT NULL
);

---- create above / drop below ----

DROP TABLE Payments, PaymentTypes, Accounts;
