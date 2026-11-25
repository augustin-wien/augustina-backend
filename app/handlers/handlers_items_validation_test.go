package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"strconv"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

// itemForm builds a multipart item form with the given fields and files (field name to file name)
func itemForm(t *testing.T, fields map[string]string, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	for field, name := range files {
		f, err := w.CreateFormFile(field, name)
		require.NoError(t, err)
		_, err = f.Write([]byte("content"))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return body, w.FormDataContentType()
}

func sendItemForm(t *testing.T, method, path string, fields, files map[string]string, want int) []byte {
	t.Helper()
	body, ct := itemForm(t, fields, files)
	return utils.TestRequestMultiPartWithAuth(t, r, method, path, body, ct, want, adminUserToken).Body.Bytes()
}

// removeItemFiles deletes the image and PDF an item upload wrote to disk
func removeItemFiles(t *testing.T, id int) {
	t.Helper()
	item, err := database.Db.GetItemIncludingArchived(id)
	require.NoError(t, err)
	if item.Image != "" {
		_ = os.Remove(item.Image)
		_ = os.Remove(thumbnailDir + "/" + item.Image[len("img/"):])
	}
	if item.PDF.Valid {
		pdf, err := database.Db.GetPDFByID(item.PDF.Int64)
		require.NoError(t, err)
		_ = os.Remove(pdf.Path)
	}
}

func TestCreateItemValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	item := func(name, kind string) map[string]string {
		return map[string]string{"Name": name, "Description": "Test item", "Price": "300", "Type": kind}
	}
	with := func(m map[string]string, k, v string) map[string]string {
		out := map[string]string{k: v}
		for key, val := range m {
			if key != k {
				out[key] = val
			}
		}
		return out
	}

	// Digital items need a license cost and a license group
	sendItemForm(t, "POST", "/api/items/", with(item("Issue A", "online_issue"), "LicenseGroup", "val_a"), nil, 400)
	sendItemForm(t, "POST", "/api/items/", with(item("Issue B", "online_issue"), "license_cost", "100"), nil, 400)
	sendItemForm(t, "POST", "/api/items/", with(with(item("Issue C", "online_issue"), "license_cost", "100"), "LicenseGroup", "val_c"), nil, 200)
	sendItemForm(t, "POST", "/api/items/", with(with(item("Issue C", "online_issue"), "license_cost", "100"), "LicenseGroup", "val_c"), nil, 400)
	res := sendItemForm(t, "POST", "/api/items/", with(with(item("Abo A", "abonement"), "license_cost", "100"), "LicenseGroup", "val_abo"), nil, 200)
	var aboID int
	require.NoError(t, json.Unmarshal(res, &aboID))
	abo, err := database.Db.GetItemIncludingDisabled(aboID)
	require.NoError(t, err)
	require.True(t, abo.LicenseItem.Valid, "an abonement gets its license item")

	// Names are unique and fields are typed
	sendItemForm(t, "POST", "/api/items/", item("Normal A", "normal_item"), nil, 200)
	sendItemForm(t, "POST", "/api/items/", item("Normal A", "normal_item"), nil, 400)
	sendItemForm(t, "POST", "/api/items/", with(item("Normal B", "normal_item"), "Price", "cheap"), nil, 400)

	// Uploads are checked by their file name
	sendItemForm(t, "POST", "/api/items/", item("Image A", "normal_item"), map[string]string{"Image": "noextension"}, 400)
	sendItemForm(t, "POST", "/api/items/", item("Image B", "normal_item"), map[string]string{"Image": "page.html"}, 400)
	sendItemForm(t, "POST", "/api/items/", item("PDF A", "issue"), map[string]string{"PDF": "noextension"}, 400)
	sendItemForm(t, "POST", "/api/items/", item("PDF B", "issue"), map[string]string{"PDF": "notes.txt"}, 400)

	res = sendItemForm(t, "POST", "/api/items/", item("Upload A", "issue"), map[string]string{"Image": "zz-upload-test.png", "PDF": "zz-upload-test.pdf"}, 200)
	var uploadID int
	require.NoError(t, json.Unmarshal(res, &uploadID))
	t.Cleanup(func() { removeItemFiles(t, uploadID) })
	uploaded, err := database.Db.GetItemIncludingDisabled(uploadID)
	require.NoError(t, err)
	require.Contains(t, uploaded.Image, "zz-upload-test")
	require.True(t, uploaded.PDF.Valid)
}

func TestUpdateItemValidation(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	a, err := database.Db.CreateItem(database.Item{Name: "Update A", Description: "Test item", Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	b, err := database.Db.CreateItem(database.Item{Name: "Update B", Description: "Test item", Price: 300, Type: "normal_item"})
	require.NoError(t, err)
	license, err := database.Db.CreateItem(database.Item{Name: "Update License", Description: "Test item", Price: 100, Type: "license_item", IsLicenseItem: true})
	require.NoError(t, err)
	_, err = database.Db.CreateItem(database.Item{Name: "License Owner", Description: "Test item", Price: 300, Type: "issue", LicenseItem: null.IntFrom(int64(license))})
	require.NoError(t, err)
	path := func(id int) string { return "/api/items/" + strconv.Itoa(id) + "/" }

	sendItemForm(t, "PUT", "/api/items/999999/", map[string]string{"Name": "x"}, nil, 400)
	sendItemForm(t, "PUT", path(a), map[string]string{"Price": "cheap"}, nil, 400)
	sendItemForm(t, "PUT", path(a), map[string]string{"Name": "Update B"}, nil, 409)
	sendItemForm(t, "PUT", path(a), map[string]string{"LicenseItem": strconv.Itoa(license)}, nil, 400)
	sendItemForm(t, "PUT", path(a), map[string]string{"Type": "online_issue"}, nil, 400)
	sendItemForm(t, "PUT", path(a), nil, map[string]string{"Image": "page.svg"}, 400)
	sendItemForm(t, "PUT", path(a), nil, map[string]string{"PDF": "notes.doc"}, 400)

	// Omitted fields keep their values
	sendItemForm(t, "PUT", path(b), map[string]string{"Price": "450"}, nil, 200)
	updated, err := database.Db.GetItemIncludingDisabled(b)
	require.NoError(t, err)
	require.Equal(t, 450, updated.Price)
	require.Equal(t, "Update B", updated.Name)
	require.Equal(t, "Test item", updated.Description)

	sendItemForm(t, "PUT", path(b), nil, map[string]string{"PDF": fmt.Sprintf("zz-update-%d.pdf", b)}, 200)
	t.Cleanup(func() { removeItemFiles(t, b) })
}
