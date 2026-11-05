ALTER TABLE settings
    ADD COLUMN IF NOT EXISTS accountingreportemail VARCHAR NOT NULL DEFAULT '';

ALTER TABLE db_settings
    ADD COLUMN IF NOT EXISTS accounting_report_sent_month VARCHAR NOT NULL DEFAULT '';
