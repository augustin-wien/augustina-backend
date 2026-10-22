package integrations

import (
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeNextcloud is a minimal WebDAV server that keeps the folders in memory
type fakeNextcloud struct {
	mu          sync.Mutex
	folders     map[string]bool
	permissions string
}

func newFakeNextcloud(t *testing.T) (*fakeNextcloud, Nextcloud) {
	t.Helper()
	f := &fakeNextcloud{folders: map[string]bool{"Verkaufende": true}, permissions: "RGDNVCK"}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, Nextcloud{
		URL:         srv.URL,
		PublicURL:   "https://cloud.example.org",
		Username:    "augustin-service",
		AppPassword: "secret",
		VendorRoot:  "Verkaufende",
	}
}

func (f *fakeNextcloud) serve(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "augustin-service" || pass != "secret" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	p := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/remote.php/dav/files/augustin-service/"), "/")

	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method {
	case "MKCOL":
		switch {
		case f.folders[p]:
			w.WriteHeader(http.StatusMethodNotAllowed)
		case !f.folders[path.Dir(p)]:
			w.WriteHeader(http.StatusConflict)
		default:
			f.folders[p] = true
			w.WriteHeader(http.StatusCreated)
		}
	case "PROPFIND":
		if !f.folders[p] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
  <d:response>
    <d:href>/remote.php/dav/files/augustin-service/` + p + `/</d:href>
    <d:propstat><d:prop><oc:fileid>4711</oc:fileid><oc:permissions>` + f.permissions + `</oc:permissions></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
  </d:response>
</d:multistatus>`))
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func TestNextcloudEnsureVendorFolder(t *testing.T) {
	f, nc := newFakeNextcloud(t)

	link, err := nc.EnsureVendorFolder("Vendor 1")
	require.NoError(t, err)
	require.Equal(t, "https://cloud.example.org/index.php/f/4711", link)
	require.True(t, f.folders["Verkaufende/Vendor 1"])

	// Opening it again finds the existing folder
	link, err = nc.EnsureVendorFolder("Vendor 1")
	require.NoError(t, err)
	require.Equal(t, "https://cloud.example.org/index.php/f/4711", link)
	require.Len(t, f.folders, 2)
}

func TestNextcloudEnsureVendorFolderNeedsRoot(t *testing.T) {
	f, nc := newFakeNextcloud(t)
	nc.VendorRoot = "Missing"

	_, err := nc.EnsureVendorFolder("Vendor 1")
	require.ErrorContains(t, err, `"Missing" does not exist`)
	require.Len(t, f.folders, 1)
}

func TestNextcloudEnsureVendorFolderRejectsPaths(t *testing.T) {
	_, nc := newFakeNextcloud(t)

	for _, licenseID := range []string{"", " ", "..", "a/b", `a\b`} {
		_, err := nc.EnsureVendorFolder(licenseID)
		require.Error(t, err, licenseID)
	}
}

func TestNextcloudTest(t *testing.T) {
	f, nc := newFakeNextcloud(t)
	require.NoError(t, nc.Test())

	wrong := nc
	wrong.AppPassword = "wrong"
	require.ErrorContains(t, wrong.Test(), "Anmeldung fehlgeschlagen")

	missing := nc
	missing.VendorRoot = "Missing"
	require.ErrorContains(t, missing.Test(), "existiert")

	f.permissions = "RGDNV"
	require.ErrorContains(t, nc.Test(), "keine Ordner anlegen")

	require.ErrorContains(t, Nextcloud{}.Test(), "URL")
}

func TestNextcloudConfigured(t *testing.T) {
	_, nc := newFakeNextcloud(t)
	require.True(t, nc.Configured())
	nc.AppPassword = ""
	require.False(t, nc.Configured())
}
