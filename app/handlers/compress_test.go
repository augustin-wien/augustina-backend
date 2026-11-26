package handlers

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONResponsesAreCompressed(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/settings/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.Equal(t, "gzip", res.Header().Get("Content-Encoding"))
	zr, err := gzip.NewReader(res.Body)
	require.NoError(t, err)
	var settings map[string]any
	require.NoError(t, json.NewDecoder(zr).Decode(&settings))
	require.Contains(t, settings, "Settings")

	req = httptest.NewRequest(http.MethodGet, "/api/settings/", nil)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	require.Empty(t, res.Header().Get("Content-Encoding"), "clients without gzip get plain JSON")
}
