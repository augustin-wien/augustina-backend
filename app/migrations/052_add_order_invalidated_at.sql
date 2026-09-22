ALTER TABLE paymentorder
    ADD COLUMN IF NOT EXISTS invalidated_at TIMESTAMPTZ;
