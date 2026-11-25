package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoDirListingFS(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.png"), []byte("png"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	fs := noDirListingFS{http.Dir(dir)}

	f, err := fs.Open("/a.png")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = fs.Open("/sub")
	require.True(t, errors.Is(err, os.ErrNotExist))
	_, err = fs.Open("/missing.png")
	require.Error(t, err)
}
