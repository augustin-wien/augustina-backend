-- Seed abonement confirmation mail template
-- Note: {{ "{{" }} and {{ "}}" }} are used to escape Go template delimiters,
-- because tern processes migration files as Go templates.
INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES (
    'abonementConfirmation',
    'Your {{ "{{" }}.ItemName{{ "}}" }} Subscription Confirmation',
    '<p>Hello {{ "{{" }}.CustomerName{{ "}}" }},</p>
<p>Your subscription to <strong>{{ "{{" }}.ItemName{{ "}}" }}</strong> has been activated.</p>
<p><strong>Subscription Details:</strong></p>
<ul>
  <li><strong>Item:</strong> {{ "{{" }}.ItemName{{ "}}" }}</li>
  <li><strong>Valid from:</strong> {{ "{{" }}.FromDate{{ "}}" }}</li>
  <li><strong>Valid until:</strong> {{ "{{" }}.ToDate{{ "}}" }}</li>
  <li><strong>Status:</strong> {{ "{{" }}.Status{{ "}}" }}</li>
</ul>
<p>You can now access all benefits of your subscription. If you have any questions, please contact our support team.</p>
<p>Best regards,<br/>The Augustin Team</p>',
    now(),
    now()
)
ON CONFLICT (name) DO UPDATE
SET body = EXCLUDED.body, subject = EXCLUDED.subject, updated_at = now();
