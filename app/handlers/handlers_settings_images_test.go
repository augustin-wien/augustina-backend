package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
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

// TestUpdateSettingsLogoThumbnail checks that an uploaded logo gets a versioned URL and a
// thumbnail, and that a logo without a thumbnail does not leave the previous one behind
func TestUpdateSettingsLogoThumbnail(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	upload := func(data []byte) string {
		body := new(bytes.Buffer)
		w := multipart.NewWriter(body)
		f, err := w.CreateFormFile("Logo", "logo.png")
		require.NoError(t, err)
		_, err = f.Write(data)
		require.NoError(t, err)
		require.NoError(t, w.Close())
		utils.TestRequestMultiPartWithAuth(t, r, "PUT", "/api/settings/", body, w.FormDataContentType(), 200, adminUserToken)
		settings, err := database.Db.GetSettings()
		require.NoError(t, err)
		return settings.Logo
	}

	src := image.NewNRGBA(image.Rect(0, 0, 1600, 400))
	var logo bytes.Buffer
	require.NoError(t, png.Encode(&logo, src))
	sum := sha256.Sum256(logo.Bytes())
	thumb := filepath.Join(thumbnailDir, "logo.png")
	t.Cleanup(func() { _ = os.Remove(thumb) })

	require.Equal(t, "/img/logo.png?v="+hex.EncodeToString(sum[:4]), upload(logo.Bytes()))
	cfg, err := png.DecodeConfig(mustOpen(t, thumb))
	require.NoError(t, err)
	require.Equal(t, thumbnailWidth, cfg.Width)

	path := upload([]byte("not an image"))
	require.NotContains(t, path, hex.EncodeToString(sum[:4]))
	require.NoFileExists(t, thumb)
}

func mustOpen(t *testing.T, name string) *os.File {
	f, err := os.Open(name)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}
