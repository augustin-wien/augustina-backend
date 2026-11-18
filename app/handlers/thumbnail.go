package handlers

import (
	"bytes"
	"crypto/rand"
	"errors"
	"github.com/go-chi/chi/v5"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// thumbnailWidth is the maximum width of generated item image thumbnails
const thumbnailWidth = 400

// thumbnailDir is the folder below img/ that holds the thumbnails. A thumbnail has
// the same file name as the original, so img/x_0.png has img/thumbs/x_0.png.
const thumbnailDir = "img/thumbs"

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
	if ext == ".png" {
		err = png.Encode(buf, thumb)
	} else {
		err = jpeg.Encode(buf, thumb, &jpeg.Options{Quality: 85})
	}
	if err != nil {
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

	thumbName := filepath.Base(thumbnailDir) + "/" + name
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
	if isRegularFile(root, thumbName) && serveRootFile(w, r, root, thumbName) {
		return
	}
	if !serveRootFile(w, r, root, name) {
		http.NotFound(w, r)
	}
}

// serveRootFile serves a file from root and reports whether it could be opened
func serveRootFile(w http.ResponseWriter, r *http.Request, root *os.Root, name string) bool {
	f, err := root.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
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
