package handlers

import (
	"bytes"
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
	ext := strings.ToLower(filepath.Ext(imagePath))
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
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	name := filepath.Base(imagePath)
	if name == "." || name == string(filepath.Separator) {
		return errors.New("invalid image path")
	}
	if err = os.MkdirAll(filepath.Join(dir, thumbnailDir), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Join(dir, thumbnailDir), ".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err = tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmp.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, thumbnailDir, name))
}

// serveThumbnail serves img/thumbs/<name>. A missing thumbnail is created from the
// original image on first request; if that is not possible the original is served.
func serveThumbnail(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "*")
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") ||
		!allowedImageExtensions[strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")] {
		http.NotFound(w, r)
		return
	}
	thumbPath := filepath.Join(thumbnailDir, name)
	origPath := filepath.Join("img", name)
	if !isRegularFile(origPath) {
		http.NotFound(w, r)
		return
	}
	if !isRegularFile(thumbPath) {
		select {
		case thumbnailSlots <- struct{}{}:
			if data, err := readFileLimited(origPath, maxThumbnailSourceBytes); err == nil {
				if terr := createThumbnail(data, origPath); terr != nil {
					log.Error("serveThumbnail: ", terr)
				}
			}
			<-thumbnailSlots
		default:
		}
	}
	if isRegularFile(thumbPath) {
		http.ServeFile(w, r, thumbPath)
		return
	}
	http.ServeFile(w, r, origPath)
}

// isRegularFile reports whether path is an existing regular file (no directory or symlink)
func isRegularFile(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode().IsRegular()
}

// readFileLimited reads a file that must not be larger than limit bytes
func readFileLimited(path string, limit int64) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Size() > limit {
		return nil, errors.New("file too large for thumbnail")
	}
	return os.ReadFile(path)
}
