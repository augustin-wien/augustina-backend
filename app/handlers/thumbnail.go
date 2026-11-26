package handlers

import (
	"bytes"
	"crypto/rand"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

// thumbnailWidth is the maximum width of generated item image thumbnails
const thumbnailWidth = 400

// thumbnailDir holds the thumbnails, which are served at /img/thumbs/<name>. A thumbnail
// has the same file name as the original, so img/x_0.png has <thumbnailDir>/x_0.png.
// The version folder changes whenever the thumbnail format changes, so thumbnails in the
// old format are regenerated on their next request.
const thumbnailDir = "img/thumbs/v2"

// Cache-Control values for images. Item uploads get a new, numbered file name on every
// upload (see updateItemImage), and the settings images carry a ?v=<content hash> in their
// stored URL, so both URLs never change their content. Any other image may be overwritten.
// Several papers can share one API host name, behind which the proxy picks the backend by
// referer, so the same URL can be a different image per paper: shared caches must not store it.
const (
	cacheImmutable   = "private, max-age=31536000, immutable"
	cacheRevalidate  = "no-cache"
	thumbnailQuality = 82
)

// uniqueUploadName matches the file names that updateItemImage gives to item uploads
var uniqueUploadName = regexp.MustCompile(`_[0-9]+\.[A-Za-z]+$`)

// imageCacheControl returns the Cache-Control header for the image file name served for r
func imageCacheControl(r *http.Request, name string) string {
	if r.URL.Query().Get("v") != "" || uniqueUploadName.MatchString(name) {
		return cacheImmutable
	}
	return cacheRevalidate
}

// serveImages serves the files in dir with the Cache-Control from imageCacheControl
func serveImages(dir http.FileSystem) http.Handler {
	fs := http.FileServer(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", imageCacheControl(r, filepath.Base(r.URL.Path)))
		fs.ServeHTTP(w, r)
	})
}

// isOpaque reports whether the image has no transparent pixels
func isOpaque(img image.Image) bool {
	o, ok := img.(interface{ Opaque() bool })
	return ok && o.Opaque()
}

// encodeThumbnail encodes a thumbnail. Opaque images are stored as JPEG, also when the
// original is a PNG, because photos and scans saved as PNG are many times larger. Only
// images that use transparency stay PNG.
func encodeThumbnail(w io.Writer, thumb image.Image) error {
	if isOpaque(thumb) {
		return jpeg.Encode(w, thumb, &jpeg.Options{Quality: thumbnailQuality})
	}
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(w, thumb)
}

// Limits for source images, so a crafted image cannot exhaust memory when decoded
const (
	maxThumbnailSourceSide   = 12000
	maxThumbnailSourcePixels = 50_000_000
	maxThumbnailSourceBytes  = 64 << 20
)

// thumbnailSlots limits concurrent thumbnail generation on the public route
var thumbnailSlots = make(chan struct{}, 2)

// scaleDown shrinks src to the given width with area averaging, keeping the aspect ratio.
// Images that are not wider than width are returned unchanged.
func scaleDown(src image.Image, width int) image.Image {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= width || sw == 0 || sh == 0 {
		return src
	}
	height := sh * width / sw
	if height < 1 {
		height = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		y0, y1 := y*sh/height, (y+1)*sh/height
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < width; x++ {
			x0, x1 := x*sw/width, (x+1)*sw/width
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, bl, a, n uint64
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					pr, pg, pb, pa := src.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
					r += uint64(pr)
					g += uint64(pg)
					bl += uint64(pb)
					a += uint64(pa)
					n++
				}
			}
			i := dst.PixOffset(x, y)
			if a == 0 {
				continue
			}
			// colour sums are alpha-premultiplied, convert back to non-premultiplied 8 bit
			dst.Pix[i+0] = uint8((r * 0xffff / a) >> 8)
			dst.Pix[i+1] = uint8((g * 0xffff / a) >> 8)
			dst.Pix[i+2] = uint8((bl * 0xffff / a) >> 8)
			dst.Pix[i+3] = uint8((a / n) >> 8)
		}
	}
	return dst
}

// createThumbnail writes a thumbnail of the image data to img/thumbs/<name of imagePath>.
// Only png and jpeg images get a thumbnail; other formats are skipped without error.
func createThumbnail(data []byte, imagePath string) error {
	name := filepath.Base(imagePath)
	if !validImageName(name) {
		return errors.New("invalid image name")
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxThumbnailSourceSide || cfg.Height > maxThumbnailSourceSide ||
		cfg.Width*cfg.Height > maxThumbnailSourcePixels {
		return errors.New("image too large for thumbnail")
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}
	thumb := scaleDown(src, thumbnailWidth)

	buf := bytes.NewBuffer(nil)
	if err = encodeThumbnail(buf, thumb); err != nil {
		return err
	}
	if err = os.MkdirAll(thumbnailDir, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(thumbnailDir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	tmpName := ".tmp-" + rand.Text()
	tmp, err := root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(tmpName) }()
	if _, err = tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return root.Rename(tmpName, name)
}

// validImageName reports whether name is a plain image file name without any path
// components, hidden-file prefix or non-image extension.
func validImageName(name string) bool {
	return name != "" && name == filepath.Base(name) && !strings.HasPrefix(name, ".") &&
		!strings.ContainsAny(name, `/\`) &&
		allowedImageExtensions[strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")]
}

// serveThumbnail serves img/thumbs/<name>. A missing thumbnail is created from the
// original image on first request; if that is not possible the original is served.
// All file access goes through an os.Root on img/, so no path can leave that folder.
func serveThumbnail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "*")
	if !validImageName(name) {
		http.NotFound(w, r)
		return
	}
	root, err := os.OpenRoot("img")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = root.Close() }()

	thumbName := strings.TrimPrefix(thumbnailDir, "img/") + "/" + name
	if !isRegularFile(root, name) {
		http.NotFound(w, r)
		return
	}
	if !isRegularFile(root, thumbName) {
		select {
		case thumbnailSlots <- struct{}{}:
			if data, err := readFileLimited(root, name, maxThumbnailSourceBytes); err == nil {
				if terr := createThumbnail(data, name); terr != nil {
					log.Error("serveThumbnail: ", terr)
				}
			}
			<-thumbnailSlots
		default:
		}
	}
	if isRegularFile(root, thumbName) && serveRootFile(w, r, root, thumbName, imageCacheControl(r, name)) {
		return
	}
	// The original stands in for a thumbnail that could not be made yet, so it must not be
	// cached for long under the thumbnail URL
	if !serveRootFile(w, r, root, name, cacheRevalidate) {
		http.NotFound(w, r)
	}
}

// serveRootFile serves a file from root and reports whether it could be opened. The
// content type is taken from the file's content, since a thumbnail of a PNG can be a JPEG.
func serveRootFile(w http.ResponseWriter, r *http.Request, root *os.Root, name, cacheControl string) bool {
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return false
	}
	if contentType := http.DetectContentType(head[:n]); strings.HasPrefix(contentType, "image/") {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, filepath.Base(name), fi.ModTime(), f)
	return true
}

// isRegularFile reports whether name is an existing regular file in root (no directory or symlink)
func isRegularFile(root *os.Root, name string) bool {
	fi, err := root.Lstat(name)
	return err == nil && fi.Mode().IsRegular()
}

// readFileLimited reads a file from root that must not be larger than limit bytes
func readFileLimited(root *os.Root, name string, limit int64) ([]byte, error) {
	fi, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if fi.Size() > limit {
		return nil, errors.New("file too large for thumbnail")
	}
	return root.ReadFile(name)
}
