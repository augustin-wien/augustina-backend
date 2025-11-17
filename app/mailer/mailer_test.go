package mailer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseTemplateFromString parses a simple template string
func TestParseTemplateFromString(t *testing.T) {
	r := NewRequest([]string{"test@example.com"}, "subj", "")
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

	r := NewRequest([]string{"foo@bar.test"}, "subj", "")
	if err := r.ParseTemplate("test_mailer_template.html", map[string]string{"Name": "World"}); err != nil {
		t.Fatalf("ParseTemplate returned error: %v", err)
	}
	if r.body != "Hello World" {
		t.Fatalf("unexpected body, want %q got %q", "Hello World", r.body)
	}
}

func TestParseTemplate_EmptyName(t *testing.T) {
	r := NewRequest(nil, "", "")
	if err := r.ParseTemplate("", nil); err == nil {
		t.Fatalf("expected error for empty template name")
	}
}
