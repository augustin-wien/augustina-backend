package handlers

import (
	"bytes"
	"image"
	"image/color"
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
	f, err := os.Open(filepath.Join(dir, "img/thumbs/test_0.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 400 || got.Bounds().Dy() != 200 {
		t.Fatalf("unexpected size %v", got.Bounds())
	}
	r, _, _, _ := got.At(10, 10).RGBA()
	if r>>8 != 200 {
		t.Fatalf("unexpected colour %d", r>>8)
	}
}
