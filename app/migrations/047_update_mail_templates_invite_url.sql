-- Add one-time login link to digital licence and abonement confirmation emails.
-- {{.InviteURL}} is injected by the backend when a WordPress invite is configured.
-- Dollar-quoting is used so tern does not interpret Go template delimiters.

UPDATE mail_templates
SET body = $$<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>
<body>
<p>
    Hallo!<br />
    <br />
    Deine neue Zeitung ist da!<br />
    <a href="{{.URL}}">Hier klicken</a>
    um die Zeitung zu lesen.
    <br />
    <br />
    {{if .InviteURL}}
    Du kannst dich direkt einloggen &ndash; kein Passwort n&ouml;tig:<br />
    <a href="{{.InviteURL}}">Jetzt einloggen</a><br />
    <br />
    {{else}}
    Wenn dies deine erste Zeitung ist, dann solltest du eine Email zum erstellen eines Passworts bekommen haben.
    Wenn nicht, dann kannst du <a href="{{.URL}}">hier klicken</a> um ein neues Passwort zu erstellen.<br />
    <br />
    {{end}}
    Viel Spass beim Lesen!<br />
</p>
</body>
</html>$$,
    updated_at = now()
WHERE name = 'digitalLicenceItemTemplate.html';

UPDATE mail_templates
SET body = $$<p>Hello {{.CustomerName}},</p>
<p>Your subscription to <strong>{{.ItemName}}</strong> has been activated.</p>
<p><strong>Subscription Details:</strong></p>
<ul>
  <li><strong>Item:</strong> {{.ItemName}}</li>
  <li><strong>Valid from:</strong> {{.FromDate}}</li>
  <li><strong>Valid until:</strong> {{.ToDate}}</li>
  <li><strong>Status:</strong> {{.Status}}</li>
</ul>
{{if .InviteURL}}
<p>You can log in directly &ndash; no password needed:<br/>
<a href="{{.InviteURL}}">Log in now</a></p>
{{end}}
<p>You can now access all benefits of your subscription. If you have any questions, please contact our support team.</p>
<p>Best regards,<br/>The Augustin Team</p>$$,
    updated_at = now()
WHERE name = 'abonementConfirmation';
