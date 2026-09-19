ALTER TABLE paymentorder
    ADD COLUMN IF NOT EXISTS odoo_synced_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS odoo_sync_error TEXT;
