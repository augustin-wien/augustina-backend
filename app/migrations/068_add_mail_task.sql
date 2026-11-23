ALTER TABLE paymentorder
    ADD COLUMN IF NOT EXISTS reconcile_failed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reconcile_error VARCHAR;

CREATE TABLE IF NOT EXISTS mail_task (
    id SERIAL PRIMARY KEY,
    kind VARCHAR NOT NULL,
    recipients TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    attachments BYTEA,
    status VARCHAR NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ,
    order_id INTEGER,
    item_id INTEGER
);

CREATE INDEX IF NOT EXISTS mailtask_status_next_attempt_at ON mail_task (status, next_attempt_at);
CREATE INDEX IF NOT EXISTS mailtask_item_id_kind ON mail_task (item_id, kind);
CREATE INDEX IF NOT EXISTS mailtask_order_id ON mail_task (order_id);
CREATE INDEX IF NOT EXISTS mailtask_created_at ON mail_task (created_at);

ALTER TABLE orderentry
    ADD COLUMN IF NOT EXISTS item_name VARCHAR NOT NULL DEFAULT '';

UPDATE orderentry SET item_name = item.name
FROM item
WHERE orderentry.item = item.id AND orderentry.item_name = '';
