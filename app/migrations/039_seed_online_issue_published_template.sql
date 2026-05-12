-- Seed online issue published mail template
INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES (
    'onlineIssuePublished',
    'New Online Issue Available: {{.IssueName}}',
    '<p>Hello,</p>
<p>A new online issue is now available: <strong>{{.IssueName}}</strong>.</p>
<p><img src="{{.ImageURL}}" alt="{{.IssueName}}" style="max-width:100%;height:auto;" /></p>
<p>Best regards,<br/>The Augustin Team</p>',
    now(),
    now()
)
ON CONFLICT (name) DO UPDATE
SET body = EXCLUDED.body, subject = EXCLUDED.subject, updated_at = now();
