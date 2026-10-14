package mailer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

// TestParseTemplateFromString parses a simple template string
func TestParseTemplateFromString(t *testing.T) {
	r, _ := NewRequest([]string{"test@example.com"}, "subj", "")
	tmpl := "Hello {{.Name}}, welcome!"
	data := map[string]string{"Name": "Alice"}
	err := r.ParseTemplateFromString(tmpl, data)
	require.NoError(t, err)
	require.Contains(t, r.body, "Alice")
}

func TestParseTemplate_Success(t *testing.T) {
	dir := "./templates"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}
	tmpl := filepath.Join(dir, "test_mailer_template.html")
	content := "Hello {{.Name}}"
	if err := os.WriteFile(tmpl, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(tmpl)
		_ = os.Remove(dir)
	})

	r, _ := NewRequest([]string{"foo@bar.test"}, "subj", "")
	if err := r.ParseTemplate("test_mailer_template.html", map[string]string{"Name": "World"}); err != nil {
		t.Fatalf("ParseTemplate returned error: %v", err)
	}
	if r.body != "Hello World" {
		t.Fatalf("unexpected body, want %q got %q", "Hello World", r.body)
	}
}

func TestParseTemplate_EmptyName(t *testing.T) {
	r, _ := NewRequest(nil, "", "")
	if err := r.ParseTemplate("", nil); err == nil {
		t.Fatalf("expected error for empty template name")
	}
}

// TestHtmlToPlain_KeepsLinkURLs guards the plain-text alternative: links must keep
// their URL, otherwise text-only clients show "Hier klicken" with nothing to click.
func TestHtmlToPlain_KeepsLinkURLs(t *testing.T) {
	body := `<p>Neu!<br /><a href="https://zeitung.example.test">Hier klicken</a> um zu lesen.
<a href="https://a.test"><b>https://a.test</b></a> <a href="">leer</a></p>`
	require.Equal(t, "Neu!Hier klicken (https://zeitung.example.test) um zu lesen.\nhttps://a.test leer", htmlToPlain(body))
}

// TestBuildMessage_SenderName checks that the sender name from the settings
// wins over SMTP_SENDER_NAME, which stays the fallback.
func TestBuildMessage_SenderName(t *testing.T) {
	oldAddr, oldEnvName, oldName := config.Config.SMTPSenderAddress, config.Config.SMTPSenderName, config.Config.MailSenderName
	t.Cleanup(func() {
		config.Config.SMTPSenderAddress, config.Config.SMTPSenderName, config.Config.MailSenderName = oldAddr, oldEnvName, oldName
	})
	config.Config.SMTPSenderAddress = "sender@example.com"
	config.Config.SMTPSenderName = "Env Name"

	fromHeader := func() string {
		r, err := NewRequest([]string{"to@example.com"}, "Hi", "<p>Hi</p>")
		require.NoError(t, err)
		msg, _, err := r.BuildMessage()
		require.NoError(t, err)
		return strings.SplitN(string(msg), "\r\n", 2)[0]
	}

	config.Config.MailSenderName = ""
	require.Equal(t, "From: Env Name <sender@example.com>", fromHeader())

	config.Config.MailSenderName = "Augustin Zeitung"
	require.Equal(t, "From: Augustin Zeitung <sender@example.com>", fromHeader())

	config.Config.MailSenderName = "Straßenzeitung"
	require.Equal(t, "From: "+encodeRFC2047("Straßenzeitung")+" <sender@example.com>", fromHeader())
}
