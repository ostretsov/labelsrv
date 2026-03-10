package renderer

import (
	"bytes"
	"testing"

	tmpl "github.com/ostretsov/labelsrv/internal/template"
)

func mustNew(t *testing.T) *Renderer {
	t.Helper()

	r, err := New("")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	return r
}

func makeTestTemplate(name string, layout []tmpl.LayoutElement) *tmpl.Template {
	return &tmpl.Template{
		Name: name,
		Size: tmpl.Size{Width: "4in", Height: "6in"},
		Inputs: map[string]tmpl.InputField{
			"recipient": {Type: "string", Required: true},
			"tracking":  {Type: "string", Required: false},
		},
		Constants: map[string]tmpl.Constant{
			"company": {Type: "string", Value: "ACME Corp", Locked: true},
		},
		Layout: layout,
	}
}

func TestRender_BasicText(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "title", Type: "text", Value: "Hello World", X: 10, Y: 10, FontSize: 14},
	})

	data := map[string]any{"recipient": "Alice"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_InputText(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "name", Type: "text", Source: "input", Key: "recipient", X: 10, Y: 10, FontSize: 12},
	})

	data := map[string]any{"recipient": "John Doe"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_ConstantText(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "company", Type: "text", Source: "constant", Key: "company", X: 10, Y: 10, FontSize: 12},
	})

	data := map[string]any{"recipient": "Bob"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_MissingConstant(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "missing", Type: "text", Source: "constant", Key: "nonexistent", X: 10, Y: 10},
	})

	_, err := r.Render(tpl, map[string]any{})
	if err == nil {
		t.Error("expected error for missing constant reference")
	}
}

func TestRender_Barcode(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "bc", Type: "barcode", Source: "input", Key: "tracking",
			BarcodeType: "code128", X: 10, Y: 30, Width: 80, Height: 20},
	})

	data := map[string]any{
		"recipient": "Alice",
		"tracking":  "1Z9999999",
	}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() barcode error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_BarcodeEmptySkipped(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "bc", Type: "barcode", Source: "input", Key: "tracking",
			BarcodeType: "code128", X: 10, Y: 30, Width: 80, Height: 20},
	})

	// tracking is absent - barcode should be skipped (empty value)
	data := map[string]any{"recipient": "Alice"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_VisibleIf_Has_Present(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "conditional", Type: "text", Value: "Shown", X: 10, Y: 10,
			VisibleIf: "has(tracking)"},
	})

	data := map[string]any{
		"recipient": "Alice",
		"tracking":  "ABC123",
	}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_VisibleIf_Has_Absent(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "conditional", Type: "text", Value: "Hidden", X: 10, Y: 10,
			VisibleIf: "has(tracking)"},
	})

	data := map[string]any{"recipient": "Alice"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_VisibleIf_Missing(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "fallback", Type: "text", Value: "No tracking", X: 10, Y: 10,
			VisibleIf: "missing(tracking)"},
	})

	data := map[string]any{"recipient": "Alice"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_MultipleElements(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "title", Type: "text", Value: "LABEL", X: 10, Y: 5, FontSize: 16},
		{ID: "name", Type: "text", Source: "input", Key: "recipient", X: 10, Y: 20, FontSize: 12},
		{ID: "company", Type: "text", Source: "constant", Key: "company", X: 10, Y: 35, FontSize: 10},
		{ID: "bc", Type: "barcode", Source: "input", Key: "tracking",
			BarcodeType: "code128", X: 10, Y: 50, Width: 80, Height: 20,
			VisibleIf: "has(tracking)"},
	})

	data := map[string]any{
		"recipient": "Jane Smith",
		"tracking":  "TRACK001",
	}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() multiple elements error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_InvalidVisibleIf(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "el", Type: "text", Value: "test", X: 10, Y: 10,
			VisibleIf: "badexpr"},
	})

	_, err := r.Render(tpl, map[string]any{})
	if err == nil {
		t.Error("expected error for invalid visible_if expression")
	}
}

func TestRender_MultiCellText(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{ID: "addr", Type: "text", Source: "input", Key: "recipient",
			X: 10, Y: 10, FontSize: 10, MaxWidth: 60},
	})

	data := map[string]any{"recipient": "123 Long Street Name, Springfield, IL 62701"}

	pdfBytes, err := r.Render(tpl, data)
	if err != nil {
		t.Fatalf("Render() MultiCell error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_TextBox_WithBorder(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{
			ID: "box", Type: "textbox",
			Value: "Line one\nLine two\nLine three",
			X:     4, Y: 20, Width: 80, Height: 30,
			BorderColor: "#333333",
			FillColor:   "#FFFFEE",
			TextColor:   "#111111",
			Padding:     3,
			FontSize:    9,
		},
	})

	pdfBytes, err := r.Render(tpl, map[string]any{})
	if err != nil {
		t.Fatalf("Render() textbox error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_TextBox_NoBorder(t *testing.T) {
	r := mustNew(t)
	tpl := makeTestTemplate("test", []tmpl.LayoutElement{
		{
			ID: "box", Type: "textbox",
			Value: "Debug invisible box with wrapped text that is quite long",
			X:     4, Y: 20, Width: 60, Height: 25,
			FontSize: 8,
		},
	})

	pdfBytes, err := r.Render(tpl, map[string]any{})
	if err != nil {
		t.Fatalf("Render() textbox no-border error: %v", err)
	}

	assertValidPDF(t, pdfBytes)
}

func TestRender_InvalidSize(t *testing.T) {
	r := mustNew(t)
	tpl := &tmpl.Template{
		Name:   "bad",
		Size:   tmpl.Size{Width: "bad", Height: "6in"},
		Layout: []tmpl.LayoutElement{},
	}

	_, err := r.Render(tpl, map[string]any{})
	if err == nil {
		t.Error("expected error for invalid template size")
	}
}

func assertValidPDF(t *testing.T, data []byte) {
	t.Helper()

	if len(data) < 4 {
		t.Fatal("PDF output is too short")
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatalf("output does not start with %%PDF, got: %q", string(data[:min(10, len(data))]))
	}
}
