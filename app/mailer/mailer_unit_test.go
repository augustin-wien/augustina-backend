package mailer

import (
	"net/smtp"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoginAuthSendsUsernameThenPassword(t *testing.T) {
	a := LoginAuth("user", "secret")

	mech, initial, err := a.Start(&smtp.ServerInfo{Name: "localhost", TLS: true})
	require.NoError(t, err)
	require.Equal(t, "LOGIN", mech)
	require.Nil(t, initial)

	resp, err := a.Next([]byte("Username:"), true)
	require.NoError(t, err)
	require.Equal(t, []byte("user"), resp)

	resp, err = a.Next([]byte("Password:"), true)
	require.NoError(t, err)
	require.Equal(t, []byte("secret"), resp)

	resp, err = a.Next([]byte("?"), true)
	require.NoError(t, err)
	require.Nil(t, resp)

	resp, err = a.Next(nil, false)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestNewRequestSanitizesRecipients(t *testing.T) {
	r, err := NewRequest([]string{" a@example.com ", "", "b@example.com"}, "Betreff", "<p>x</p>")
	require.NoError(t, err)
	require.Equal(t, []string{"a@example.com", "b@example.com"}, r.To())
	require.Equal(t, "Betreff", r.Subject())
	require.Equal(t, "<p>x</p>", r.Body())

	r.SetSubject("Neu")
	require.Equal(t, "Neu", r.Subject())

	_, err = NewRequest([]string{"a@example.com\r\nBcc: evil@example.com"}, "s", "")
	require.Error(t, err)

	_, err = NewRequest([]string{"a@example.com"}, "s\r\nBcc: evil@example.com", "")
	require.Error(t, err)
}

func TestAddAttachmentRejectsHeaderInjection(t *testing.T) {
	r, err := NewRequest([]string{"a@example.com"}, "s", "")
	require.NoError(t, err)

	require.NoError(t, r.AddAttachment("gutschein.pdf", "application/pdf", []byte("%PDF")))
	require.Len(t, r.Attachments(), 1)

	require.Error(t, r.AddAttachment("evil\".pdf", "application/pdf", nil))
	require.Error(t, r.AddAttachment("evil.pdf", "application/pdf\r\nX-Evil: 1", nil))
	require.Len(t, r.Attachments(), 1)
}

func TestEncodeRFC2047(t *testing.T) {
	require.Equal(t, "Plain subject", encodeRFC2047("Plain subject"))
	require.Equal(t, "=?UTF-8?B?R3LDvMOfZQ==?=", encodeRFC2047("Grüße"))
	require.Equal(t, "=?UTF-8?B?YQpi?=", encodeRFC2047("a\nb"))
}

func TestNewRequestFromTemplate(t *testing.T) {
	dir := "./templates"
	require.NoError(t, os.MkdirAll(dir, 0o755))
	tmpl := filepath.Join(dir, "test_from_template.html")
	require.NoError(t, os.WriteFile(tmpl, []byte("Hallo {{.Name}}"), 0o644))
	t.Cleanup(func() {
		_ = os.Remove(tmpl)
		_ = os.Remove(dir)
	})

	r, err := NewRequestFromTemplate([]string{"a@example.com"}, "s", "test_from_template.html", map[string]string{"Name": "Augustin"})
	require.NoError(t, err)
	require.Equal(t, "Hallo Augustin", r.Body())

	_, err = NewRequestFromTemplate([]string{"a@example.com"}, "s", "does_not_exist.html", nil)
	require.Error(t, err)

	_, err = NewRequestFromTemplate([]string{"a@example.com"}, "s\n", "test_from_template.html", nil)
	require.Error(t, err)
}

func TestParseTemplateFromStringErrors(t *testing.T) {
	r, err := NewRequest([]string{"a@example.com"}, "s", "")
	require.NoError(t, err)
	require.Error(t, r.ParseTemplateFromString("", nil))
	require.Error(t, r.ParseTemplateFromString("{{.Unclosed", nil))
}
