package integrations

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/augustin-wien/augustina-backend/ent"
)

var nextcloudClient = &http.Client{Timeout: 15 * time.Second}

// Nextcloud holds the connection settings of the vendor document folders
type Nextcloud struct {
	URL         string
	PublicURL   string
	Username    string
	AppPassword string
	// VendorRoot is an existing folder of Username that holds one folder per vendor
	VendorRoot string
}

// NextcloudFromSettings reads the connection from the admin settings
func NextcloudFromSettings(s *ent.Settings) Nextcloud {
	return Nextcloud{
		URL:         strings.TrimSuffix(strings.TrimSpace(s.NextcloudURL), "/"),
		PublicURL:   strings.TrimSuffix(strings.TrimSpace(s.NextcloudPublicURL), "/"),
		Username:    strings.TrimSpace(s.NextcloudUsername),
		AppPassword: s.NextcloudAppPassword,
		VendorRoot:  strings.Trim(strings.TrimSpace(s.NextcloudVendorRoot), "/"),
	}
}

// Configured reports whether everything needed to talk to Nextcloud is filled in
func (nc Nextcloud) Configured() bool {
	return nc.URL != "" && nc.Username != "" && nc.AppPassword != "" && nc.VendorRoot != ""
}

// EnsureVendorFolder creates the document folder of the vendor with the given license ID if it
// does not exist yet and returns a link that opens it in the Nextcloud web interface.
func (nc Nextcloud) EnsureVendorFolder(licenseID string) (string, error) {
	if !nc.Configured() {
		return "", errors.New("nextcloud is not configured")
	}
	licenseID = strings.TrimSpace(licenseID)
	if licenseID == "" || strings.ContainsAny(licenseID, `/\`) || licenseID == "." || licenseID == ".." {
		return "", fmt.Errorf("license ID %q can not be used as a folder name", licenseID)
	}

	// Only the vendor's own folder is created. A missing root fails with 409 instead of ending
	// up as a private folder of the service account that nobody else can see.
	segments := append(strings.Split(nc.VendorRoot, "/"), licenseID)
	resp, err := nc.request("MKCOL", segments, nil, nil)
	if err != nil {
		return "", fmt.Errorf("nextcloud: creating %s: %w", strings.Join(segments, "/"), err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusMethodNotAllowed:
	case http.StatusConflict:
		return "", fmt.Errorf("nextcloud: folder %q does not exist", nc.VendorRoot)
	default:
		return "", fmt.Errorf("nextcloud: creating %s: unexpected status %d", strings.Join(segments, "/"), resp.StatusCode)
	}

	props, err := nc.folderProps(segments)
	if err != nil {
		return "", err
	}
	return nc.publicURL() + "/index.php/f/" + url.PathEscape(props.FileID), nil
}

// Test checks the login and that the vendor root exists and new folders can be created in it.
// The message is meant for the backoffice.
func (nc Nextcloud) Test() (ok bool, message string) {
	switch {
	case nc.URL == "":
		return false, "Keine Nextcloud-URL eingetragen"
	case nc.Username == "" || nc.AppPassword == "":
		return false, "Benutzername oder App-Passwort fehlt"
	case nc.VendorRoot == "":
		return false, "Kein Ordner eingetragen"
	}
	props, err := nc.folderProps(strings.Split(nc.VendorRoot, "/"))
	var statusErr *nextcloudStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.Status {
		case http.StatusUnauthorized:
			return false, "Anmeldung fehlgeschlagen: Benutzername oder App-Passwort ist falsch"
		case http.StatusNotFound:
			return false, fmt.Sprintf("Der Ordner „%s“ existiert für %s nicht", nc.VendorRoot, nc.Username)
		}
	}
	if err != nil {
		return false, err.Error()
	}
	// Nextcloud lists what the user may do with the folder; C means creating files and folders
	if !strings.Contains(props.Permissions, "C") {
		return false, fmt.Sprintf("%s darf im Ordner „%s“ keine Ordner anlegen", nc.Username, nc.VendorRoot)
	}
	return true, "Verbindung funktioniert, Ordner können angelegt werden"
}

func (nc Nextcloud) publicURL() string {
	if nc.PublicURL != "" {
		return nc.PublicURL
	}
	return nc.URL
}

func (nc Nextcloud) davURL(segments []string) string {
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	return nc.URL + "/remote.php/dav/files/" + url.PathEscape(nc.Username) + "/" + strings.Join(escaped, "/") + "/"
}

func (nc Nextcloud) request(method string, segments []string, body io.Reader, header map[string]string) (*http.Response, error) {
	base, err := url.Parse(nc.URL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.User != nil {
		return nil, fmt.Errorf("invalid Nextcloud URL %q", nc.URL)
	}
	req, err := http.NewRequest(method, nc.davURL(segments), body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(nc.Username, nc.AppPassword)
	// Without it Nextcloud treats the app password like a browser login and wants a CSRF token.
	req.Header.Set("OCS-APIRequest", "true")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	return nextcloudClient.Do(req)
}

type nextcloudStatusError struct {
	Status int
}

func (e *nextcloudStatusError) Error() string {
	return fmt.Sprintf("nextcloud: unexpected status %d", e.Status)
}

type davProps struct {
	FileID      string `xml:"fileid"`
	Permissions string `xml:"permissions"`
}

type davMultistatus struct {
	Responses []struct {
		Propstats []struct {
			Prop   davProps `xml:"prop"`
			Status string   `xml:"status"`
		} `xml:"propstat"`
	} `xml:"response"`
}

const folderPropfind = `<?xml version="1.0"?>
<d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
  <d:prop><oc:fileid/><oc:permissions/></d:prop>
</d:propfind>`

func (nc Nextcloud) folderProps(segments []string) (davProps, error) {
	resp, err := nc.request("PROPFIND", segments, strings.NewReader(folderPropfind), map[string]string{
		"Depth":        "0",
		"Content-Type": "application/xml",
	})
	if err != nil {
		return davProps{}, fmt.Errorf("nextcloud: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusMultiStatus {
		return davProps{}, &nextcloudStatusError{Status: resp.StatusCode}
	}

	var ms davMultistatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return davProps{}, fmt.Errorf("nextcloud: reading folder: %w", err)
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstats {
			if ps.Prop.FileID != "" && strings.Contains(ps.Status, " 200 ") {
				return ps.Prop, nil
			}
		}
	}
	return davProps{}, errors.New("nextcloud: folder has no file id")
}
