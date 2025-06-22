package controller

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/jpeg"
)

// coverts an image to JPEG format and compresses it to be under 100KB
func compressToJPEGUnder100KB(input []byte, maxWidth int) ([]byte, error) {
	// Decode the input image
	img, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	// Resize the image if its width exceeds maxWidth
	if img.Bounds().Dx() > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	// Try compressing the image to JPEG format with decreasing quality
	for quality := 80; quality >= 30; quality -= 5 {
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("jpeg encode failed: %w", err)
		}
		if buf.Len() <= 100*1024 {
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("cannot compress under 100KB")
}
