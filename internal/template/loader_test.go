package template

import (
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `
name: test-shipping
size:
  width: 6in
  height: 4in
inputs:
  recipient:
    type: string
    required: true
layout:
  - id: name
    type: text
    source: input
    key: recipient
    x: 10
    y: 10
    font_size: 12
`

func TestLoader_LoadAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shipping.yaml"), []byte(validYAML), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewTemplateLoader()
	if err := loader.LoadAll(dir); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	tmpl, ok := loader.Get("test-shipping")
	if !ok {
		t.Fatal("template not found after LoadAll")
	}

	if tmpl.Name != "test-shipping" {
		t.Errorf("Name = %q, want test-shipping", tmpl.Name)
	}
}

func TestLoader_LoadAll_SkipsInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	// Write one valid and one invalid template
	if err := os.WriteFile(filepath.Join(dir, "valid.yaml"), []byte(validYAML), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte("name: [bad yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewTemplateLoader()
	// Should not error even if some files are invalid
	if err := loader.LoadAll(dir); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	names := loader.List()
	if len(names) != 1 {
		t.Errorf("expected 1 template loaded, got %d", len(names))
	}
}

func TestLoader_LoadAll_SkipsNonTemplateFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("docs"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b"), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewTemplateLoader()
	if err := loader.LoadAll(dir); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	if len(loader.List()) != 0 {
		t.Error("expected no templates from non-template files")
	}
}

func TestLoader_LoadAll_NonexistentDir(t *testing.T) {
	loader := NewTemplateLoader()

	err := loader.LoadAll("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestLoader_Get_NotFound(t *testing.T) {
	loader := NewTemplateLoader()

	_, ok := loader.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent template")
	}
}

func TestLoader_List(t *testing.T) {
	dir := t.TempDir()

	tmpl1 := `name: label-one
size:
  width: 4in
  height: 6in
layout:
  - id: el1
    type: text
    value: "test"
    x: 10
    y: 10
`
	tmpl2 := `name: label-two
size:
  width: 4in
  height: 6in
layout:
  - id: el1
    type: text
    value: "test"
    x: 10
    y: 10
`

	if err := os.WriteFile(filepath.Join(dir, "one.yaml"), []byte(tmpl1), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "two.yaml"), []byte(tmpl2), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewTemplateLoader()
	if err := loader.LoadAll(dir); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	names := loader.List()
	if len(names) != 2 {
		t.Errorf("expected 2 templates, got %d: %v", len(names), names)
	}
}

func TestLoader_All(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shipping.yaml"), []byte(validYAML), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewTemplateLoader()
	if err := loader.LoadAll(dir); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	all := loader.All()
	if len(all) != 1 {
		t.Errorf("All() returned %d templates, want 1", len(all))
	}

	if _, ok := all["test-shipping"]; !ok {
		t.Error("All() missing test-shipping template")
	}
}

func TestLoader_LoadAll_JSON(t *testing.T) {
	dir := t.TempDir()

	content := `{
  "name": "json-label",
  "size": {"width": "100mm", "height": "50mm"},
  "layout": [
    {"id": "el1", "type": "text", "value": "test", "x": 5, "y": 5}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "label.json"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	loader := NewTemplateLoader()
	if err := loader.LoadAll(dir); err != nil {
		t.Fatalf("LoadAll() error: %v", err)
	}

	_, ok := loader.Get("json-label")
	if !ok {
		t.Error("JSON template not loaded")
	}
}
