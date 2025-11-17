-- Migration: create mail_templates table
-- Created by assistant

CREATE TABLE IF NOT EXISTS mail_templates (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- End of migration
