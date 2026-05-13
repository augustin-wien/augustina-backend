-- Add unique constraints and missing index on customer table

BEGIN;

-- Partial unique index: only enforce uniqueness for non-empty emails
-- (empty string is the default for customers without an email on record)
CREATE UNIQUE INDEX idx_customer_email_unique ON customer(email) WHERE email != '';

-- Partial unique index: only enforce uniqueness for non-empty keycloak IDs
-- (empty string means the Keycloak account has not been linked yet)
CREATE UNIQUE INDEX idx_customer_keycloakid_unique ON customer(keycloakid) WHERE keycloakid != '';

-- Regular index to speed up email lookups (GetCustomerByEmail queries)
CREATE INDEX idx_customer_email ON customer(email);

COMMIT;
