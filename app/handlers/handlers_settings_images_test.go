package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// TestUpdateSettingsRejectsInvalidImages checks that an invalid image upload
// rejects the whole form with one error response
func TestUpdateSettingsRejectsInvalidImages(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	before, err := database.Db.GetSettings()
	require.NoError(t, err)

	for field, filename := range map[string]string{
		"Logo":             "logo.jpg",
		"Favicon":          "favicon",
		"QRCodeLogoImgUrl": "qr.gif",
	} {
		t.Run(field, func(t *testing.T) {
			body := new(bytes.Buffer)
			w := multipart.NewWriter(body)
			require.NoError(t, w.WriteField("MaxOrderAmount", "4321"))
			f, err := w.CreateFormFile(field, filename)
			require.NoError(t, err)
			_, err = f.Write([]byte("image"))
			require.NoError(t, err)
			require.NoError(t, w.Close())

			res := utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, w.FormDataContentType(), 400, adminUserToken)
			var answer map[string]any
			require.NoError(t, json.Unmarshal(res.Body.Bytes(), &answer), "one JSON response: %s", res.Body.String())

			after, err := database.Db.GetSettings()
			require.NoError(t, err)
			require.Equal(t, before.MaxOrderAmount, after.MaxOrderAmount, "nothing is saved")
		})
	}
}
