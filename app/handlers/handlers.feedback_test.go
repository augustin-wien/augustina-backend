package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func putSettingsFields(t *testing.T, fields map[string]string, status int) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		require.NoError(t, writer.WriteField(k, v))
	}
	require.NoError(t, writer.Close())
	utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, writer.FormDataContentType(), status, adminUserToken)
}

func TestUpdateSettingsShopOptions(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	settings, err := database.Db.GetSettings()
	require.NoError(t, err)
	require.True(t, settings.AGBRequired)
	require.True(t, settings.DonationPageEnabled)
	require.False(t, settings.MatomoTrackCart)
	require.False(t, settings.FeedbackEnabled)

	putSettingsFields(t, map[string]string{"AGBRequired": "maybe"}, 400)
	putSettingsFields(t, map[string]string{"FeedbackEmail": "not an address"}, 400)
	putSettingsFields(t, map[string]string{
		"AGBRequired":         "false",
		"DonationPageEnabled": "false",
		"MatomoTrackCart":     "true",
		"FeedbackEnabled":     "true",
		"FeedbackEmail":       " feedback@example.com ",
	}, 200)

	settings, err = database.Db.GetSettings()
	require.NoError(t, err)
	require.False(t, settings.AGBRequired)
	require.False(t, settings.DonationPageEnabled)
	require.True(t, settings.MatomoTrackCart)
	require.True(t, settings.FeedbackEnabled)
	require.Equal(t, "feedback@example.com", settings.FeedbackEmail)

	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	var public map[string]map[string]any
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &public))
	require.Equal(t, false, public["Settings"]["AGBRequired"])
	require.Equal(t, false, public["Settings"]["DonationPageEnabled"])
	require.Equal(t, true, public["Settings"]["MatomoTrackCart"])
	require.Equal(t, true, public["Settings"]["FeedbackEnabled"])
	require.NotContains(t, public["Settings"], "FeedbackEmail")
	require.NotEmpty(t, res.Header().Get("X-Request-Id"))
}

func TestSendFeedback(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	origSend := mailer.Send
	defer func() { mailer.Send = origSend }()
	var sent []*mailer.EmailRequest
	mailer.Send = func(r *mailer.EmailRequest) (bool, error) {
		sent = append(sent, r)
		return true, nil
	}

	feedback := FeedbackInput{
		Message:       "Hat <b>super</b> geklappt",
		TransactionID: "tx-123",
		RequestID:     "req-456",
		VendorID:      "<scanned-id>",
		Page:          "/success",
	}

	utils.TestRequest(t, r, "POST", "/api/feedback/", feedback, 404)
	require.Empty(t, sent)

	putSettingsFields(t, map[string]string{"FeedbackEnabled": "true", "FeedbackEmail": "feedback@example.com"}, 200)

	utils.TestRequest(t, r, "POST", "/api/feedback/", FeedbackInput{Message: "   "}, 400)
	utils.TestRequest(t, r, "POST", "/api/feedback/", feedback, 200)

	require.Len(t, sent, 1)
	body := sent[0].Body()
	require.Contains(t, body, "Hat &lt;b&gt;super&lt;/b&gt; geklappt")
	require.Contains(t, body, "tx-123")
	require.Contains(t, body, "req-456")
	require.Contains(t, body, "/success")
	require.Contains(t, body, "&lt;scanned-id&gt;")
}
