-- Fix two bugs in mail_templates rows written by earlier migrations:
--
-- 1. 'welcome': its subject/body referenced a Name placeholder, but the Go
--    code that builds the welcome mail (VerifyOrderAndCreatePayments) only
--    ever supplies URL and EMAIL. Go's text/template hard-errors on an
--    undefined struct field, so building the welcome mail has always failed:
--    "can't evaluate field Name in type struct { URL string; EMAIL string }".
--
-- 2. 'digitalLicenceItemTemplate.html' and 'abonementConfirmation': migration
--    047 wrote its URL / InviteURL / CustomerName / etc placeholders WITHOUT
--    escaping the Go template delimiters. tern renders every migration .sql
--    file as a Go template (against an essentially empty data map) before
--    running it against Postgres, so those placeholders were resolved at
--    migration-apply time to the literal text "<no value>" and baked into
--    the stored rows -- and the InviteURL conditional branch was permanently
--    collapsed to its else-branch. Verified by re-running the migrations
--    against a scratch database and inspecting the resulting rows.
--
-- 3. 'PDFLicenceItemTemplate.html' has the same bug, introduced earlier by
--    migration 026, which also wrote its URL placeholder unescaped.
--
-- As in migrations 025/037/039, the delimiter-escaping trick below (quoting
-- the literal brace characters as template actions) keeps tern from touching
-- the real placeholders, so the app's own template engine can fill them in
-- at send time (see BuildEmailRequestFromTemplate). Updates are guarded to
-- only touch rows that still contain the known-broken content, in case an
-- admin already hand-edited a template.

UPDATE mail_templates
SET subject = 'Welcome to Augustin!',
    body = $$<p>Hello,</p>
<p>Welcome to Augustin! Your account has been created for {{ "{{" }}.EMAIL{{ "}}" }}.</p>
<p><a href="{{ "{{" }}.URL{{ "}}" }}">Click here to read your newspaper</a>.</p>$$,
    updated_at = now()
WHERE name = 'welcome'
  AND (subject LIKE '%{{ "{{" }}.Name{{ "}}" }}%' OR body LIKE '%{{ "{{" }}.Name{{ "}}" }}%');

UPDATE mail_templates
SET body = $$<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<body>
<p>
    Hallo!<br />
    <br />
    Deine neue Zeitung ist da!<br />
    <a href="{{ "{{" }}.URL{{ "}}" }}">Hier klicken</a>
    um die Zeitung zu lesen.
    <br />
    <br />
    {{ "{{" }}if .InviteURL{{ "}}" }}
    Du kannst dich direkt einloggen &ndash; kein Passwort n&ouml;tig:<br />
    <a href="{{ "{{" }}.InviteURL{{ "}}" }}">Jetzt einloggen</a><br />
    <br />
    {{ "{{" }}else{{ "}}" }}
    Wenn dies deine erste Zeitung ist, dann solltest du eine Email zum erstellen eines Passworts bekommen haben.
    Wenn nicht, dann kannst du <a href="{{ "{{" }}.URL{{ "}}" }}">hier klicken</a> um ein neues Passwort zu erstellen.<br />
    <br />
    {{ "{{" }}end{{ "}}" }}
    Viel Spass beim Lesen!<br />
</p>
</body>
</html>$$,
    updated_at = now()
WHERE name = 'digitalLicenceItemTemplate.html'
  AND body LIKE '%<no value>%';

UPDATE mail_templates
SET body = $$<p>Hello {{ "{{" }}.CustomerName{{ "}}" }},</p>
<p>Your subscription to <strong>{{ "{{" }}.ItemName{{ "}}" }}</strong> has been activated.</p>
<p><strong>Subscription Details:</strong></p>
<ul>
  <li><strong>Item:</strong> {{ "{{" }}.ItemName{{ "}}" }}</li>
  <li><strong>Valid from:</strong> {{ "{{" }}.FromDate{{ "}}" }}</li>
  <li><strong>Valid until:</strong> {{ "{{" }}.ToDate{{ "}}" }}</li>
  <li><strong>Status:</strong> {{ "{{" }}.Status{{ "}}" }}</li>
</ul>
{{ "{{" }}if .InviteURL{{ "}}" }}
<p>You can log in directly &ndash; no password needed:<br/>
<a href="{{ "{{" }}.InviteURL{{ "}}" }}">Log in now</a></p>
{{ "{{" }}end{{ "}}" }}
<p>You can now access all benefits of your subscription. If you have any questions, please contact our support team.</p>
<p>Best regards,<br/>The Augustin Team</p>$$,
    updated_at = now()
WHERE name = 'abonementConfirmation'
  AND body LIKE '%<no value>%';

UPDATE mail_templates
SET body = $$<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<body>
<p>
    Hallo!<br />
    <br />
    Deine neue Zeitung ist da!<br />
    <a href="{{ "{{" }}.URL{{ "}}" }}">Hier klicken</a>
    um die Zeitung zu lesen.
    <br />
    <br />
    Viel Spass beim Lesen!<br />
</p>
</body>
</html>$$,
    updated_at = now()
WHERE name = 'PDFLicenceItemTemplate.html'
  AND body LIKE '%<no value>%';
