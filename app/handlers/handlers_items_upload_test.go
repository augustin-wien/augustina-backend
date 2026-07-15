package handlers

import (
	"strings"
	"testing"
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
