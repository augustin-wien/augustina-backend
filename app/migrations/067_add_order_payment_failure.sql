ALTER TABLE paymentorder
    ADD COLUMN IF NOT EXISTS payment_event_id INTEGER,
    ADD COLUMN IF NOT EXISTS payment_failed_at TIMESTAMPTZ;
