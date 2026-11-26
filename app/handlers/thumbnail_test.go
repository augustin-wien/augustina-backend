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
)

func TestCreateThumbnail(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	src := image.NewNRGBA(image.Rect(0, 0, 1200, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 1200; x++ {
			src.Set(x, y, color.NRGBA{200, 10, 10, 255})
		}
	}
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, src); err != nil {
		t.Fatal(err)
	}
	if err := createThumbnail(buf.Bytes(), "img/test_0.png"); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, thumbnailDir, "test_0.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	got, format, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" {
		t.Fatalf("opaque png thumbnail stored as %s, want jpeg", format)
	}
	if got.Bounds().Dx() != 400 || got.Bounds().Dy() != 200 {
		t.Fatalf("unexpected size %v", got.Bounds())
	}
	r, _, _, _ := got.At(10, 10).RGBA()
	if d := int(r>>8) - 200; d < -3 || d > 3 {
		t.Fatalf("unexpected colour %d", r>>8)
	}
}

func TestCreateThumbnailKeepsTransparentPNG(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	src := image.NewNRGBA(image.Rect(0, 0, 800, 200))
	for x := 0; x < 400; x++ {
		for y := 0; y < 200; y++ {
			src.Set(x, y, color.NRGBA{10, 10, 200, 255})
		}
	}
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, src); err != nil {
		t.Fatal(err)
	}
	if err := createThumbnail(buf.Bytes(), "img/logo.png"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, thumbnailDir, "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	got, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" {
		t.Fatalf("transparent thumbnail stored as %s, want png", format)
	}
	if got.Bounds().Dx() != 400 || got.Bounds().Dy() != 100 {
		t.Fatalf("unexpected size %v", got.Bounds())
	}
	if _, _, _, a := got.At(300, 50).RGBA(); a != 0 {
		t.Fatalf("transparent area has alpha %d", a)
	}
}
