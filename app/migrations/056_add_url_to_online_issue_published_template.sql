-- Add a link to the 'onlineIssuePublished' mail template. The notification now
-- supplies a URL placeholder: a personal PDF download link when the online issue
-- is a PDF, otherwise the online paper URL.
-- Only touches the row if it still holds the body seeded by migration 039, so
-- hand-edited templates are left alone.
-- Note: {{ "{{" }} and {{ "}}" }} escape the Go template delimiters, because tern
-- processes migration files as Go templates.
UPDATE mail_templates
SET body = $$<p>Hello,</p>
<p>A new online issue is now available: <strong>{{ "{{" }}.IssueName{{ "}}" }}</strong>.</p>
<p><img src="{{ "{{" }}.ImageURL{{ "}}" }}" alt="{{ "{{" }}.IssueName{{ "}}" }}" style="max-width:100%;height:auto;" /></p>
<p><a href="{{ "{{" }}.URL{{ "}}" }}">Click here to read the issue</a>.</p>
<p>Best regards,<br/>The Augustin Team</p>$$,
    updated_at = now()
WHERE name = 'onlineIssuePublished'
  AND body = $$<p>Hello,</p>
<p>A new online issue is now available: <strong>{{ "{{" }}.IssueName{{ "}}" }}</strong>.</p>
<p><img src="{{ "{{" }}.ImageURL{{ "}}" }}" alt="{{ "{{" }}.IssueName{{ "}}" }}" style="max-width:100%;height:auto;" /></p>
<p>Best regards,<br/>The Augustin Team</p>$$;
