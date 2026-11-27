ALTER TABLE db_settings
    ADD COLUMN IF NOT EXISTS accounting_report_last_to VARCHAR NOT NULL DEFAULT '';
