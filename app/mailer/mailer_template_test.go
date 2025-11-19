package mailer

import (
	"testing"

	"github.com/augustin-wien/augustina-backend/config"
	"github.com/stretchr/testify/require"
)

func TestParseTemplateFromString_DefaultsURLAndEmail(t *testing.T) {
	// set a known frontend URL
	config.Config.OnlinePaperUrl = "https://example.test/online"

	r := NewRequest([]string{"buyer@example.test"}, "subj", "")
	// pass nil data -> ParseTemplateFromString should fill URL and EMAIL
	tmpl := "Hello {{.EMAIL}} - open at {{.URL}}"
	err := r.ParseTemplateFromString(tmpl, nil)
	require.NoError(t, err)
	require.Contains(t, r.body, "buyer@example.test")
	require.Contains(t, r.body, "https://example.test/online")
}
