CREATE TABLE IF NOT EXISTS external_payments (
    id SERIAL PRIMARY KEY,
    transaction_id TEXT NOT NULL UNIQUE,
    order_code TEXT NOT NULL DEFAULT '',
    amount INTEGER NOT NULL DEFAULT 0,
    fee INTEGER NOT NULL DEFAULT 0,
    status_id TEXT NOT NULL DEFAULT '',
    source_code TEXT NOT NULL DEFAULT '',
    source_name TEXT NOT NULL DEFAULT '',
    terminal_id INTEGER NOT NULL DEFAULT 0,
    merchant_trns TEXT NOT NULL DEFAULT '',
    paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS external_payments_paid_at ON external_payments (paid_at);

ALTER TABLE settings ADD COLUMN IF NOT EXISTS donationtrackingenabled BOOLEAN NOT NULL DEFAULT false;
