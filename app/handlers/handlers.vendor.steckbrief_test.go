package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func uploadSteckbriefImage(t *testing.T, url string, data []byte, status int) database.VendorSteckbrief {
	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	f, err := w.CreateFormFile("Image", "photo.html")
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	res := utils.TestRequestMultiPartWithAuth(t, r, "POST", url, body, w.FormDataContentType(), status, adminUserToken)
	var s database.VendorSteckbrief
	if status == 200 {
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &s))
	}
	return s
}

// TestVendorSteckbrief checks that the backoffice can fill in the Steckbrief and upload
// its photo once the module is enabled, and that a vendor update keeps the Steckbrief
func TestVendorSteckbrief(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorID, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Steck",
		LastName:  "Brief",
		Email:     "steckbrief@vendor.com",
		LicenseID: null.StringFrom("SB-1"),
	})
	require.NoError(t, err)
	base := "/api/vendors/" + strconv.Itoa(vendorID) + "/steckbrief/"

	utils.TestRequestWithAuth(t, r, "GET", base, nil, 403, adminUserToken)
	putSettingsFields(t, map[string]string{"SteckbriefEnabled": "true"}, 200)
	res := utils.TestRequest(t, r, "GET", "/api/settings/", nil, 200)
	require.Contains(t, res.Body.String(), `"SteckbriefEnabled":true`)

	utils.TestRequest(t, r, "GET", base, nil, 401)
	utils.TestRequestWithAuth(t, r, "GET", "/api/vendors/999999/steckbrief/", nil, 404, adminUserToken)

	res = utils.TestRequestWithAuth(t, r, "PUT", base, map[string]string{
		"Description": "  Ich verkaufe seit 2019.  ",
		"Family":      "Zwei Kinder",
		"Residence":   "Wien",
		"Education":   "Pflichtschule",
		"CareerGoal":  "Koch",
		"Support":     "Kleidung",
		"FirstName":   "Ignored",
		"Image":       "img/other.png",
	}, 200, adminUserToken)
	var s database.VendorSteckbrief
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &s))
	require.Equal(t, "Ich verkaufe seit 2019.", s.Description)
	require.Equal(t, "Koch", s.CareerGoal)
	require.Equal(t, "Steck", s.FirstName)
	require.Equal(t, "SB-1", s.LicenseID)
	require.Empty(t, s.Image)

	utils.TestRequestWithAuth(t, r, "PUT", base, map[string]string{
		"Description": strings.Repeat("a", maxSteckbriefTextLen+1),
	}, 400, adminUserToken)

	// Only real images are accepted, whatever the file name says
	uploadSteckbriefImage(t, base+"image/", []byte("<script>alert(1)</script>"), 400)

	var photo bytes.Buffer
	require.NoError(t, png.Encode(&photo, image.NewNRGBA(image.Rect(0, 0, 600, 800))))
	first := uploadSteckbriefImage(t, base+"image/", photo.Bytes(), 200)
	require.Regexp(t, `^img/steckbrief_[a-z0-9]+_0\.png$`, first.Image)
	require.FileExists(t, first.Image)
	require.FileExists(t, filepath.Join(thumbnailDir, filepath.Base(first.Image)))

	second := uploadSteckbriefImage(t, base+"image/", photo.Bytes(), 200)
	require.NotEqual(t, first.Image, second.Image)
	require.NoFileExists(t, first.Image, "the replaced photo is removed")
	require.NoFileExists(t, filepath.Join(thumbnailDir, filepath.Base(first.Image)))

	// Updating the vendor itself, e.g. from the profile form, leaves the Steckbrief alone
	vendor, err := database.Db.GetVendor(vendorID)
	require.NoError(t, err)
	vendor.Telephone = "0123"
	require.NoError(t, database.Db.UpdateVendor(vendorID, vendor))
	s, err = database.Db.GetVendorSteckbrief(vendorID)
	require.NoError(t, err)
	require.Equal(t, second.Image, s.Image)
	require.Equal(t, "Wien", s.Residence)

	res = utils.TestRequestWithAuth(t, r, "DELETE", base+"image/", nil, 200, adminUserToken)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &s))
	require.Empty(t, s.Image)
	require.NoFileExists(t, second.Image)
	_, err = os.Stat(filepath.Join(thumbnailDir, filepath.Base(second.Image)))
	require.ErrorIs(t, err, os.ErrNotExist)
}

// TestPublicVendorSteckbrief checks what the shop gets to see of the Steckbrief
func TestPublicVendorSteckbrief(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	vendorID, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Public",
		LastName:  "Sheet",
		Email:     "public@vendor.com",
		LicenseID: null.StringFrom("PS-1"),
	})
	require.NoError(t, err)
	check := func() checkLicenseIDResponse {
		res := utils.TestRequest(t, r, "GET", "/api/vendors/check/PS-1/", nil, 200)
		var c checkLicenseIDResponse
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &c))
		return c
	}
	const url = "/api/vendors/check/PS-1/steckbrief/"

	_, err = database.Db.UpdateVendorSteckbrief(vendorID, database.VendorSteckbrief{Residence: "Linz"})
	require.NoError(t, err)
	require.False(t, check().HasSteckbrief, "module off")
	utils.TestRequest(t, r, "GET", url, nil, 404)

	putSettingsFields(t, map[string]string{"SteckbriefEnabled": "true"}, 200)
	require.True(t, check().HasSteckbrief)
	res := utils.TestRequest(t, r, "GET", url, nil, 200)
	var s PublicVendorSteckbrief
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &s))
	require.Equal(t, PublicVendorSteckbrief{FirstName: "Public", LastName: "Sheet", LicenseID: "PS-1", Residence: "Linz"}, s)
	require.NotContains(t, res.Body.String(), "public@vendor.com")

	putSettingsFields(t, map[string]string{"UseVendorLicenseIdInShop": "true"}, 200)
	res = utils.TestRequest(t, r, "GET", url, nil, 200)
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &s))
	require.Empty(t, s.FirstName)
	require.Empty(t, s.LastName)

	vendor, err := database.Db.GetVendor(vendorID)
	require.NoError(t, err)
	vendor.IsBlocked = true
	require.NoError(t, database.Db.UpdateVendor(vendorID, vendor))
	utils.TestRequest(t, r, "GET", url, nil, 404)
	vendor.IsBlocked = false
	require.NoError(t, database.Db.UpdateVendor(vendorID, vendor))

	_, err = database.Db.UpdateVendorSteckbrief(vendorID, database.VendorSteckbrief{})
	require.NoError(t, err)
	require.False(t, check().HasSteckbrief, "empty Steckbrief")
	utils.TestRequest(t, r, "GET", url, nil, 404)
	utils.TestRequest(t, r, "GET", "/api/vendors/check/unknown/steckbrief/", nil, 404)
}
