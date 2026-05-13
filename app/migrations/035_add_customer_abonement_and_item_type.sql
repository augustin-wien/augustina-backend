-- Add Customer and Abonement tables, plus type field to Item

BEGIN;

-- 1) Add item_type column to item table
ALTER TABLE item ADD COLUMN item_type VARCHAR(50) DEFAULT 'normal_item';

-- 2) Create customer table
CREATE TABLE IF NOT EXISTS customer (
    id BIGSERIAL PRIMARY KEY,
    keycloakid VARCHAR(255) NOT NULL,
    email VARCHAR(255) DEFAULT '',
    firstname VARCHAR(255) DEFAULT '',
    lastname VARCHAR(255) DEFAULT '',
    licensegroups TEXT DEFAULT '',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

-- 3) Create abonement table
CREATE TABLE IF NOT EXISTS abonement (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL,
    abonement_item BIGINT,
    from_date TIMESTAMPTZ NOT NULL,
    to_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customer(id) ON DELETE CASCADE,
    CONSTRAINT fk_item FOREIGN KEY (abonement_item) REFERENCES item(id) ON DELETE CASCADE
);

-- 4) Create indexes for better query performance
CREATE INDEX idx_customer_keycloakid ON customer(keycloakid);
CREATE INDEX idx_abonement_customer_id ON abonement(customer_id);
CREATE INDEX idx_abonement_item_id ON abonement(abonement_item);
CREATE INDEX idx_abonement_dates ON abonement(from_date, to_date);

COMMIT;
