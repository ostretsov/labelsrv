package barcode

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGenerateCode128(t *testing.T) {
	data, err := GenerateCode128("HELLO123", 200, 80)
	if err != nil {
		t.Fatalf("GenerateCode128() error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerateCode128_Empty(t *testing.T) {
	// Empty content should fail barcode encoding
	_, err := GenerateCode128("", 200, 80)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestGenerateCode39(t *testing.T) {
	data, err := GenerateCode39("TEST123", 200, 80)
	if err != nil {
		t.Fatalf("GenerateCode39() error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerateEAN13(t *testing.T) {
	// EAN13 requires exactly 12 digits (13th is checksum)
	data, err := GenerateEAN13("590123412345", 200, 80)
	if err != nil {
		t.Fatalf("GenerateEAN13() error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerateEAN13_Invalid(t *testing.T) {
	_, err := GenerateEAN13("not-a-number", 200, 80)
	if err == nil {
		t.Error("expected error for non-numeric EAN13 content")
	}
}

func TestGenerateQR(t *testing.T) {
	data, err := GenerateQR("https://example.com", 200)
	if err != nil {
		t.Fatalf("GenerateQR() error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerate_Code128(t *testing.T) {
	data, err := Generate("code128", "ABC123", 200, 80)
	if err != nil {
		t.Fatalf("Generate(code128) error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerate_Code128Uppercase(t *testing.T) {
	data, err := Generate("CODE128", "ABC", 200, 80)
	if err != nil {
		t.Fatalf("Generate(CODE128) error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerate_Code39(t *testing.T) {
	data, err := Generate("code39", "ABC", 200, 80)
	if err != nil {
		t.Fatalf("Generate(code39) error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerate_QR(t *testing.T) {
	data, err := Generate("qr", "hello", 100, 100)
	if err != nil {
		t.Fatalf("Generate(qr) error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerate_QRCode(t *testing.T) {
	data, err := Generate("qrcode", "hello", 100, 100)
	if err != nil {
		t.Fatalf("Generate(qrcode) error: %v", err)
	}

	assertValidPNG(t, data)
}

func TestGenerate_Unsupported(t *testing.T) {
	_, err := Generate("pdf417", "data", 200, 80)
	if err == nil {
		t.Error("expected error for unsupported barcode type")
	}
}

func TestGenerate_UnknownType(t *testing.T) {
	_, err := Generate("unknown", "data", 200, 80)
	if err == nil {
		t.Error("expected error for unknown barcode type")
	}
}

func TestGenerateCode128_DefaultDimensions(t *testing.T) {
	// Test with zero width/height - should use defaults
	data, err := GenerateCode128("TEST", 0, 0)
	if err != nil {
		t.Fatalf("GenerateCode128() with zero dims error: %v", err)
	}

	assertValidPNG(t, data)
}

func assertValidPNG(t *testing.T, data []byte) {
	t.Helper()

	if len(data) == 0 {
		t.Fatal("PNG data is empty")
	}

	_, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a valid PNG: %v", err)
	}
}
