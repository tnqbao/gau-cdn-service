package controller

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/disintegration/imaging"
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

// shouldCompressImage determines if an image should be compressed based on its content type
func shouldCompressImage(contentType string) bool {
	// Don't compress vector/icon formats as they would lose quality or functionality
	skipTypes := []string{
		"image/svg+xml",
		"image/x-icon",
		"image/vnd.microsoft.icon",
	}

	for _, skipType := range skipTypes {
		if strings.Contains(strings.ToLower(contentType), skipType) {
			return false
		}
	}

	// Compress all other image types (JPEG, PNG, WebP) in their original format
	compressibleTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
	}

	for _, compressType := range compressibleTypes {
		if strings.Contains(strings.ToLower(contentType), compressType) {
			return true
		}
	}

	return false
}

// compressImageInOriginalFormat compresses an image in its original format based on maxSize limit
func compressImageInOriginalFormat(input []byte, contentType string, maxSize int64, maxWidth int) ([]byte, error) {
	// Decode the input image
	img, _, err := image.Decode(bytes.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	// Resize the image if its width exceeds maxWidth
	if img.Bounds().Dx() > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	// Compress based on original format
	switch {
	case strings.Contains(strings.ToLower(contentType), "image/jpeg") || strings.Contains(strings.ToLower(contentType), "image/jpg"):
		return compressJPEG(img, maxSize)
	case strings.Contains(strings.ToLower(contentType), "image/png"):
		return compressPNG(img, maxSize)
	case strings.Contains(strings.ToLower(contentType), "image/webp"):
		// For WebP, convert to JPEG as Go's webp package doesn't support encoding
		return compressJPEG(img, maxSize)
	default:
		return nil, fmt.Errorf("unsupported format for compression: %s", contentType)
	}
}

// compressJPEG compresses image to JPEG format under maxSize
func compressJPEG(img image.Image, maxSize int64) ([]byte, error) {
	for quality := 90; quality >= 30; quality -= 10 {
		buf := new(bytes.Buffer)
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("jpeg encode failed: %w", err)
		}
		if int64(buf.Len()) <= maxSize {
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("cannot compress JPEG under %d bytes", maxSize)
}

// compressPNG compresses image to PNG format under maxSize
func compressPNG(img image.Image, maxSize int64) ([]byte, error) {
	// PNG doesn't have quality settings, so we can only resize further if needed
	buf := new(bytes.Buffer)
	encoder := &png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(buf, img); err != nil {
		return nil, fmt.Errorf("png encode failed: %w", err)
	}

	// If still too large, try resizing further
	if int64(buf.Len()) > maxSize {
		for scale := 0.8; scale >= 0.3; scale -= 0.1 {
			newWidth := int(float64(img.Bounds().Dx()) * scale)
			resizedImg := imaging.Resize(img, newWidth, 0, imaging.Lanczos)

			buf = new(bytes.Buffer)
			if err := encoder.Encode(buf, resizedImg); err != nil {
				return nil, fmt.Errorf("png encode failed: %w", err)
			}

			if int64(buf.Len()) <= maxSize {
				return buf.Bytes(), nil
			}
		}
		return nil, fmt.Errorf("cannot compress PNG under %d bytes", maxSize)
	}

	return buf.Bytes(), nil
}
