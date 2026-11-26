package handlers

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
)

func TestServeThumbnail(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	src := image.NewNRGBA(image.Rect(0, 0, 1000, 500))
	for y := 0; y < 500; y++ {
		for x := 0; x < 1000; x++ {
			src.Set(x, y, color.NRGBA{R: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, src))

	const name = "zz_thumbnail_serve_test.png"
	const broken = "zz_thumbnail_serve_broken.png"
	require.NoError(t, os.WriteFile(filepath.Join("img", name), buf.Bytes(), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join("img", broken), []byte("not an image"), 0o644))
	t.Cleanup(func() {
		for _, p := range []string{
			filepath.Join("img", name), filepath.Join(thumbnailDir, name),
			filepath.Join("img", broken), filepath.Join(thumbnailDir, broken),
		} {
			_ = os.Remove(p)
		}
	})

	// The thumbnail is created on the first request and served from then on. The opaque
	// png becomes a jpeg, which the content type has to say despite the .png name.
	res := utils.TestRequest(t, r, "GET", "/img/thumbs/"+name, nil, 200)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(res.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Equal(t, "image/jpeg", res.Header().Get("Content-Type"))
	require.Equal(t, thumbnailWidth, cfg.Width)
	require.Equal(t, thumbnailWidth/2, cfg.Height)
	require.FileExists(t, filepath.Join(thumbnailDir, name))
	require.Equal(t, cacheRevalidate, res.Header().Get("Cache-Control"), "the name can be overwritten")

	res = utils.TestRequest(t, r, "GET", "/img/thumbs/"+name+"?v=abc", nil, 200)
	cfg, _, err = image.DecodeConfig(bytes.NewReader(res.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, thumbnailWidth, cfg.Width)
	require.Equal(t, cacheImmutable, res.Header().Get("Cache-Control"), "a versioned URL never changes")

	// If no thumbnail can be made, the original is served, but not cached for long
	res = utils.TestRequest(t, r, "GET", "/img/thumbs/"+broken+"?v=abc", nil, 200)
	require.Equal(t, "not an image", res.Body.String())
	require.Equal(t, cacheRevalidate, res.Header().Get("Cache-Control"))

	utils.TestRequest(t, r, "GET", "/img/thumbs/zz_does_not_exist.png", nil, 404)
	utils.TestRequest(t, r, "GET", "/img/thumbs/.hidden.png", nil, 404)
	utils.TestRequest(t, r, "GET", "/img/thumbs/script.sh", nil, 404)
}

func TestReadFileLimited(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.png"), []byte("12345"), 0o644))
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer root.Close()

	data, err := readFileLimited(root, "a.png", 10)
	require.NoError(t, err)
	require.Equal(t, "12345", string(data))

	_, err = readFileLimited(root, "a.png", 4)
	require.Error(t, err)
	_, err = readFileLimited(root, "missing.png", 10)
	require.Error(t, err)

	require.True(t, isRegularFile(root, "a.png"))
	require.False(t, isRegularFile(root, "missing.png"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub.png"), 0o755))
	require.False(t, isRegularFile(root, "sub.png"))
}

func TestServeImagesCacheControl(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()

	const upload = "zz_cache_test_3.png"
	const fixed = "zz_cache_test.png"
	for _, n := range []string{upload, fixed} {
		require.NoError(t, os.WriteFile(filepath.Join("img", n), []byte("x"), 0o644))
	}
	t.Cleanup(func() {
		_ = os.Remove(filepath.Join("img", upload))
		_ = os.Remove(filepath.Join("img", fixed))
	})

	res := utils.TestRequest(t, r, "GET", "/img/"+upload, nil, 200)
	require.Equal(t, cacheImmutable, res.Header().Get("Cache-Control"), "item uploads get a new name on every upload")
	res = utils.TestRequest(t, r, "GET", "/img/"+fixed, nil, 200)
	require.Equal(t, cacheRevalidate, res.Header().Get("Cache-Control"), "settings images are overwritten in place")
	res = utils.TestRequest(t, r, "GET", "/img/"+fixed+"?v=1a2b3c4d", nil, 200)
	require.Equal(t, cacheImmutable, res.Header().Get("Cache-Control"))
}
