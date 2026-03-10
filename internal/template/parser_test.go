package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		wantMM  float64
		wantErr bool
	}{
		{"6in", 152.4, false},
		{"4in", 101.6, false},
		{"1in", 25.4, false},
		{"100mm", 100, false},
		{"50mm", 50, false},
		{"10cm", 100, false},
		{"2.5cm", 25, false},
		{"72pt", 72 * 0.352778, false},
		{"0mm", 0, false},
		{"", 0, true},
		{"abc", 0, true},
		{"6xyz", 0, true},
		{"in", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSize(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}

			if !tt.wantErr {
				const epsilon = 0.001

				diff := got - tt.wantMM
				if diff < -epsilon || diff > epsilon {
					t.Errorf("ParseSize(%q) = %v, want %v", tt.input, got, tt.wantMM)
				}
			}
		})
	}
}

func TestParseFile_YAML(t *testing.T) {
	content := `
name: test-label
size:
  width: 4in
  height: 6in
inputs:
  recipient:
    type: string
    required: true
    description: Recipient name
constants:
  company:
    type: string
    value: "ACME"
    locked: true
layout:
  - id: title
    type: text
    value: "Hello"
    x: 10
    y: 10
    font_size: 12
`
	path := writeTempFile(t, "test.yaml", content)

	tmpl, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() unexpected error: %v", err)
	}

	if tmpl.Name != "test-label" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "test-label")
	}

	if tmpl.Size.Width != "4in" {
		t.Errorf("Size.Width = %q, want %q", tmpl.Size.Width, "4in")
	}

	if len(tmpl.Inputs) != 1 {
		t.Errorf("len(Inputs) = %d, want 1", len(tmpl.Inputs))
	}

	if !tmpl.Inputs["recipient"].Required {
		t.Error("recipient input should be required")
	}

	if len(tmpl.Constants) != 1 {
		t.Errorf("len(Constants) = %d, want 1", len(tmpl.Constants))
	}

	if tmpl.Constants["company"].Value != "ACME" {
		t.Errorf("company constant value = %q, want ACME", tmpl.Constants["company"].Value)
	}

	if len(tmpl.Layout) != 1 {
		t.Errorf("len(Layout) = %d, want 1", len(tmpl.Layout))
	}
}

func TestParseFile_JSON(t *testing.T) {
	content := `{
  "name": "json-label",
  "size": {"width": "100mm", "height": "50mm"},
  "inputs": {
    "name": {"type": "string", "required": true}
  },
  "layout": [
    {"id": "elem1", "type": "text", "value": "test", "x": 5, "y": 5, "font_size": 10}
  ]
}`
	path := writeTempFile(t, "test.json", content)

	tmpl, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() unexpected error: %v", err)
	}

	if tmpl.Name != "json-label" {
		t.Errorf("Name = %q, want json-label", tmpl.Name)
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/template.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseFile_InvalidYAML(t *testing.T) {
	path := writeTempFile(t, "bad.yaml", "name: [invalid yaml {}")

	_, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	validTemplate := &Template{
		Name: "test",
		Size: Size{Width: "4in", Height: "6in"},
		Layout: []LayoutElement{
			{ID: "el1", Type: "text"},
		},
	}

	if err := Validate(validTemplate); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	tmpl := &Template{
		Size: Size{Width: "4in", Height: "6in"},
	}

	err := Validate(tmpl)
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestValidate_MissingWidth(t *testing.T) {
	tmpl := &Template{
		Name: "test",
		Size: Size{Height: "6in"},
	}

	err := Validate(tmpl)
	if err == nil {
		t.Error("expected error for missing width")
	}
}

func TestValidate_InvalidSizeUnit(t *testing.T) {
	tmpl := &Template{
		Name: "test",
		Size: Size{Width: "6xyz", Height: "4in"},
	}

	err := Validate(tmpl)
	if err == nil {
		t.Error("expected error for invalid size unit")
	}
}

func TestValidate_InvalidInputType(t *testing.T) {
	tmpl := &Template{
		Name: "test",
		Size: Size{Width: "4in", Height: "6in"},
		Inputs: map[string]InputField{
			"field1": {Type: "unsupported"},
		},
	}

	err := Validate(tmpl)
	if err == nil {
		t.Error("expected error for invalid input type")
	}
}

func TestValidate_MissingElementID(t *testing.T) {
	tmpl := &Template{
		Name: "test",
		Size: Size{Width: "4in", Height: "6in"},
		Layout: []LayoutElement{
			{Type: "text"}, // missing ID
		},
	}

	err := Validate(tmpl)
	if err == nil {
		t.Error("expected error for missing element ID")
	}
}

func TestValidate_InvalidElementType(t *testing.T) {
	tmpl := &Template{
		Name: "test",
		Size: Size{Width: "4in", Height: "6in"},
		Layout: []LayoutElement{
			{ID: "el1", Type: "unknown"},
		},
	}

	err := Validate(tmpl)
	if err == nil {
		t.Error("expected error for invalid element type")
	}
}

// writeTempFile creates a temporary file with the given content and returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeTempFile: %v", err)
	}

	return path
}
