package barcode

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/code39"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/qr"
)

// GenerateCode128 generates a Code 128 barcode as PNG bytes.
func GenerateCode128(content string, width, height int) ([]byte, error) {
	bc, err := code128.Encode(content)
	if err != nil {
		return nil, fmt.Errorf("encoding Code128 barcode: %w", err)
	}

	return scaledPNG(bc, width, height)
}

// GenerateCode39 generates a Code 39 barcode as PNG bytes.
func GenerateCode39(content string, width, height int) ([]byte, error) {
	bc, err := code39.Encode(content, true, true)
	if err != nil {
		return nil, fmt.Errorf("encoding Code39 barcode: %w", err)
	}

	return scaledPNG(bc, width, height)
}

// GenerateEAN13 generates an EAN-13 barcode as PNG bytes.
func GenerateEAN13(content string, width, height int) ([]byte, error) {
	bc, err := ean.Encode(content)
	if err != nil {
		return nil, fmt.Errorf("encoding EAN13 barcode: %w", err)
	}

	return scaledPNG(bc, width, height)
}

// GenerateQR generates a QR code barcode as PNG bytes.
func GenerateQR(content string, size int) ([]byte, error) {
	bc, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("encoding QR barcode: %w", err)
	}

	return scaledPNG(bc, size, size)
}

// Generate generates a barcode of the given type as PNG bytes.
func Generate(barcodeType string, content string, width, height int) ([]byte, error) {
	switch barcodeType {
	case "code128", "Code128", "CODE128":
		return GenerateCode128(content, width, height)
	case "code39", "Code39", "CODE39":
		return GenerateCode39(content, width, height)
	case "ean13", "EAN13", "ean-13":
		return GenerateEAN13(content, width, height)
	case "qr", "QR", "qrcode":
		return GenerateQR(content, width)
	default:
		return nil, fmt.Errorf("unsupported barcode type: %q", barcodeType)
	}
}

// scaledPNG scales a barcode to the given dimensions and encodes it as PNG.
func scaledPNG(bc barcode.Barcode, width, height int) ([]byte, error) {
	if width <= 0 {
		width = 200
	}

	if height <= 0 {
		height = 100
	}

	scaled, err := barcode.Scale(bc, width, height)
	if err != nil {
		return nil, fmt.Errorf("scaling barcode: %w", err)
	}

	// Convert to NRGBA (8-bit) since gofpdf does not support 16-bit PNGs.
	src := scaled.(image.Image)
	bounds := src.Bounds()
	nrgba := image.NewNRGBA(bounds)
	draw.Draw(nrgba, bounds, src, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, nrgba); err != nil {
		return nil, fmt.Errorf("encoding barcode as PNG: %w", err)
	}

	return buf.Bytes(), nil
}
