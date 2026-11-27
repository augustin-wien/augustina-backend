ALTER TABLE settings
    ADD COLUMN IF NOT EXISTS salesanonymizationenabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS vendoranonymizationenabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS anonymizationretentiondays INTEGER NOT NULL DEFAULT 90;

---- create above / drop below ----

ALTER TABLE settings
    DROP COLUMN IF EXISTS salesanonymizationenabled,
    DROP COLUMN IF EXISTS vendoranonymizationenabled,
    DROP COLUMN IF EXISTS anonymizationretentiondays;
