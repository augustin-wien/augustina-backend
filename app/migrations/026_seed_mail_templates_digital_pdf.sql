-- Seed digitalLicenceItemTemplate and PDFLicenceItemTemplate
INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES (
  'digitalLicenceItemTemplate.html',
  'A new newspaper has been purchased',
  $$
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>

</head>

<body>
<p>
    Hallo!<br />
    <br />

    Deine neue Zeitung ist da!<br />
    <a href="{{.URL}}">Hier klicken</a>
    um die Zeitung zu lesen.
    <br />
    <br />
    Wenn dies deine erste Zeitung ist, dann solltest du eine Email zum erstellen eines Passworts bekommen haben. 
    Wenn nicht, dann kannst du <a href="{{.URL}}">hier klicken</a> um ein neues Passwort zu erstellen.<br />
    <br />
    <br />
    Viel Spass beim Lesen!<br />

</p>
    
</body>

</html>
$$,
  now(), now()
) ON CONFLICT (name) DO NOTHING;

INSERT INTO mail_templates (name, subject, body, created_at, updated_at)
VALUES (
  'PDFLicenceItemTemplate.html',
  'Deine Zeitung ist bereit zum Download',
  $$
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html>

</head>

<body>
<p>
    Hallo!<br />
    <br />

    Deine neue Zeitung ist da!<br />
    <a href="{{.URL}}">Hier klicken</a>
    um die Zeitung zu lesen.
    <br />
    <br />
    Viel Spass beim Lesen!<br />

</p>
    
</body>

</html>
$$,
  now(), now()
) ON CONFLICT (name) DO NOTHING;
