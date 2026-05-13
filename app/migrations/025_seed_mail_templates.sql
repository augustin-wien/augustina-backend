-- Seed a basic mail template
-- Note: {{ "{{" }} and {{ "}}" }} are used to escape Go template delimiters,
-- because tern processes migration files as Go templates.
INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES ('welcome', 'Welcome to Augustin, {{ "{{" }}.Name{{ "}}" }}', '<p>Hello {{ "{{" }}.Name{{ "}}" }},</p><p>Welcome to Augustin.</p>', now(), now())
ON CONFLICT (name) DO NOTHING;
