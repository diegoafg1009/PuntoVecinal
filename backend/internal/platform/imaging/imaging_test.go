package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// withEXIF inserts an APP1 segment carrying GPS-looking data right after the
// JPEG start-of-image marker.
func withEXIF(t *testing.T, jpg []byte) []byte {
	t.Helper()
	payload := append([]byte("Exif\x00\x00"), []byte("GPSLatitude=-11.9;GPSLongitude=-77.0")...)
	n := len(payload) + 2
	seg := append([]byte{0xFF, 0xE1, byte(n >> 8), byte(n)}, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func TestCleanDropsEXIF(t *testing.T) {
	in := withEXIF(t, encodeJPEG(t, solid(40, 30, color.RGBA{R: 200, A: 255})))
	if !bytes.Contains(in, []byte("GPSLatitude")) {
		t.Fatal("test setup: input must carry EXIF")
	}

	out, err := Clean(in)
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if bytes.Contains(out, []byte("GPSLatitude")) || bytes.Contains(out, []byte("Exif")) {
		t.Error("output still carries EXIF data")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || format != "jpeg" {
		t.Fatalf("output is not a readable jpeg: %v %q", err, format)
	}
	if cfg.Width != 40 || cfg.Height != 30 {
		t.Errorf("size = %dx%d, want 40x30", cfg.Width, cfg.Height)
	}
}

func TestCleanConvertsPNGToJPEG(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, solid(10, 10, color.NRGBA{A: 0})); err != nil {
		t.Fatal(err)
	}
	out, err := Clean(b.Bytes())
	if err != nil {
		t.Fatalf("Clean: %v", err)
	}
	img, format, err := image.Decode(bytes.NewReader(out))
	if err != nil || format != "jpeg" {
		t.Fatalf("output is not a jpeg: %v %q", err, format)
	}
	r, g, bl, _ := img.At(5, 5).RGBA()
	if r>>8 < 250 || g>>8 < 250 || bl>>8 < 250 {
		t.Errorf("transparent pixel = %d,%d,%d, want white", r>>8, g>>8, bl>>8)
	}
}

func TestCleanRejects(t *testing.T) {
	tests := map[string][]byte{
		"not an image":  []byte("hello, I am a text file"),
		"empty":         nil,
		"too wide":      encodeJPEG(t, solid(MaxSide+1, 1, color.White)),
		"too tall":      encodeJPEG(t, solid(1, MaxSide+1, color.White)),
		"truncated jpg": encodeJPEG(t, solid(50, 50, color.White))[:100],
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Clean(data); !errors.Is(err, ErrInvalidImage) {
				t.Errorf("err = %v, want ErrInvalidImage", err)
			}
		})
	}
}
