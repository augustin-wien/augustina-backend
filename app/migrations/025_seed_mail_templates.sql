-- Seed a basic mail template
INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES ('welcome', 'Welcome to Augustin, {{.Name}}', '<p>Hello {{.Name}},</p><p>Welcome to Augustin.</p>', now(), now())
ON CONFLICT (name) DO NOTHING;
