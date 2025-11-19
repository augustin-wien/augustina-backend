package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/augustin-wien/augustina-backend/config"
	dbpkg "github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
)

func TestSendMailTemplateHandler_FillsURLAndEmailAndSends(t *testing.T) {
	// prepare request body: only `to` so Data will be nil and handler should default it
	body := map[string]interface{}{
		"to": []string{"buyer@example.test"},
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/mail-templates/mytemplate/send/", bytes.NewReader(b))
	// set chi route param `name`
	rc := chi.NewRouteContext()
	rc.URLParams.Add("name", "mytemplate")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()

	// stub DB function and mailer.Send
	origBuild := dbpkg.BuildEmailRequestFromTemplate
	origSend := mailer.Send
	defer func() {
		dbpkg.BuildEmailRequestFromTemplate = origBuild
		mailer.Send = origSend
	}()

	var sentMail *mailer.EmailRequest
	dbpkg.BuildEmailRequestFromTemplate = func(name string, to []string, data interface{}) (*mailer.EmailRequest, error) {
		// ensure handler supplied name and recipient
		if name != "mytemplate" {
			t.Fatalf("unexpected template name: %s", name)
		}
		if len(to) != 1 || to[0] != "buyer@example.test" {
			t.Fatalf("unexpected recipients: %#v", to)
		}
		// Render a template that uses both casings: EMAIL uppercase and url lowercase
		tmpl := "email: {{.EMAIL}} url: {{.url}}"
		r := mailer.NewRequest(to, "subject", "")
		if err := r.ParseTemplateFromString(tmpl, data); err != nil {
			t.Fatalf("ParseTemplateFromString failed: %v", err)
		}
		return r, nil
	}

	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		sentMail = r
		return true, nil
	}

	// Set a known OnlinePaperUrl so handler's defaults are deterministic
	config.Config.OnlinePaperUrl = "https://example.test/online"

	// call handler
	SendMailTemplateTest(rr, req)

	resp := rr.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// ensure mail was sent and body contained URL and email
	require.NotNil(t, sentMail)
	// The handler defaults URL to http://example.com when no data provided
	require.Contains(t, sentMail.Body(), "http://example.com")
	require.Contains(t, sentMail.Body(), "buyer@example.test")
}
