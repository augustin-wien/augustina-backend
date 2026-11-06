package handlers

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

// TestSanitizeUploadFilename verifies that client-supplied upload filenames are stripped of
// any directory components and that traversal attempts are rejected, so a write can never
// escape the intended img/ or pdf/ directory.
func TestSanitizeUploadFilename(t *testing.T) {
	okCases := []struct {
		filename string
		wantBase string
		wantExt  string
	}{
		{"photo.png", "photo", "png"},
		{"../../etc/passwd.png", "passwd", "png"},
		{"/var/www/logo.jpg", "logo", "jpg"},
	}
	for _, tc := range okCases {
		base, ext, err := sanitizeUploadFilename(tc.filename)
		if err != nil {
			t.Errorf("sanitizeUploadFilename(%q) returned unexpected error: %v", tc.filename, err)
			continue
		}
		if base != tc.wantBase || ext != tc.wantExt {
			t.Errorf("sanitizeUploadFilename(%q) = (%q, %q), want (%q, %q)", tc.filename, base, ext, tc.wantBase, tc.wantExt)
		}
		if strings.Contains(base, "..") || strings.ContainsAny(base, `/\`) {
			t.Errorf("sanitizeUploadFilename(%q) produced unsafe base %q", tc.filename, base)
		}
	}

	errCases := []string{
		"",
		"noextension",
		".hidden",                 // empty base
		"trailingdot.",            // empty ext
		"..",                      // no extension after base
		`..\..\windows\evil.pdf`,  // backslashes are not path separators on Linux; reject defensively
	}
	for _, filename := range errCases {
		if _, _, err := sanitizeUploadFilename(filename); err == nil {
			t.Errorf("sanitizeUploadFilename(%q) expected error, got nil", filename)
		}
	}
}

// Uploads are served with a content type taken from their extension, item images from /img/ and
// PDFs through downloadPDF, so a file that a browser would run as a page must be refused
func TestItemUploadsRejectActiveContent(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	require.NoError(t, database.Db.InitEmptyTestDb())

	patterns := []string{"img/uploadxss_*", "pdf/*_uploadxss.*"}
	removeUploads := func() {
		for _, pattern := range patterns {
			matches, _ := filepath.Glob(pattern)
			for _, m := range matches {
				_ = os.Remove(m)
			}
		}
	}
	removeUploads()
	t.Cleanup(removeUploads)

	upload := func(field, filename string) {
		t.Helper()
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		require.NoError(t, writer.WriteField("Name", "Upload "+filename))
		require.NoError(t, writer.WriteField("Description", "Upload test"))
		require.NoError(t, writer.WriteField("Price", "100"))
		file, err := writer.CreateFormFile(field, filename)
		require.NoError(t, err)
		_, err = file.Write([]byte(`<script>alert(document.domain)</script>`))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		utils.TestRequestMultiPartWithAuth(t, r, "POST", "/api/items/", body, writer.FormDataContentType(), 400, adminUserToken)
	}

	upload("Image", "uploadxss.html")
	upload("Image", "uploadxss.svg")
	upload("PDF", "uploadxss.html")

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		require.Empty(t, matches, "nothing may be written for a rejected upload")
	}

	items, err := database.Db.ListItemsWithDisabled(false, false)
	require.NoError(t, err)
	for _, item := range items {
		require.False(t, strings.HasPrefix(item.Name, "Upload "), "no item may be created for a rejected upload")
	}
}
