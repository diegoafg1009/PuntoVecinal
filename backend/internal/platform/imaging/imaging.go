// Package imaging cleans photos uploaded by neighbors: it decodes them to
// pixels and encodes them again as JPEG, which drops EXIF (including the GPS
// position of the phone) and any other metadata.
package imaging

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder for image.Decode
)

const (
	// MaxSide is the longest side, in pixels, that is accepted. The app never
	// sends more than 1600 px.
	MaxSide = 2000
	// maxPixels bounds the memory a decode can take (decompression bombs).
	maxPixels = MaxSide * MaxSide
	// jpegQuality is the quality of the stored photos.
	jpegQuality = 85
)

// ErrInvalidImage is returned when the data is not a JPEG or PNG image, is
// corrupt, or is larger than the accepted dimensions.
var ErrInvalidImage = errors.New("invalid image")

// Clean decodes a JPEG or PNG image and returns it re-encoded as a JPEG
// without any metadata. Transparent areas of a PNG become white.
func Clean(data []byte) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	if format != "jpeg" && format != "png" {
		return nil, fmt.Errorf("%w: unsupported format %q", ErrInvalidImage, format)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxSide || cfg.Height > MaxSide || cfg.Width*cfg.Height > maxPixels {
		return nil, fmt.Errorf("%w: dimensions %dx%d", ErrInvalidImage, cfg.Width, cfg.Height)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}

	// Flatten onto white so transparency does not turn black in the JPEG.
	flat := image.NewRGBA(img.Bounds())
	draw.Draw(flat, flat.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, img.Bounds().Min, draw.Over)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, flat, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return out.Bytes(), nil
}
