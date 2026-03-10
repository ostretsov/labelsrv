package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ostretsov/labelsrv/internal/renderer"
	tmpl "github.com/ostretsov/labelsrv/internal/template"
)

func makeRendererForTest(t *testing.T) *renderer.Renderer {
	t.Helper()

	r, err := renderer.New("")
	if err != nil {
		t.Fatalf("renderer.New() error: %v", err)
	}

	return r
}

func makeLoaderWithTemplate(t *testing.T, name string, tpl *tmpl.Template) *tmpl.TemplateLoader {
	t.Helper()

	loader := tmpl.NewTemplateLoader()
	loader.ForTest(name, tpl)

	return loader
}

func TestGenerateOpenAPI_Structure(t *testing.T) {
	tpl := &tmpl.Template{
		Name: "shipping",
		Size: tmpl.Size{Width: "6in", Height: "4in"},
		Inputs: map[string]tmpl.InputField{
			"recipient": {Type: "string", Required: true, Description: "Recipient name", MaxLength: 100},
			"tracking":  {Type: "string", Required: false, Description: "Tracking number"},
		},
		Constants: map[string]tmpl.Constant{
			"service": {Type: "string", Value: "EXPRESS", Locked: true, Description: "Service level"},
		},
	}

	loader := makeLoaderWithTemplate(t, "shipping", tpl)
	spec := GenerateOpenAPI(loader)

	if spec["openapi"] != "3.0.3" {
		t.Errorf("openapi version = %v, want 3.0.3", spec["openapi"])
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths is not a map")
	}

	path, ok := paths["/labels/shipping"]
	if !ok {
		t.Fatal("path /labels/shipping not found in spec")
	}

	postMap := path.(map[string]any)["post"].(map[string]any)

	if postMap["operationId"] != "renderLabel_shipping" {
		t.Errorf("operationId = %v, want renderLabel_shipping", postMap["operationId"])
	}

	xConsts, ok := postMap["x-constants"].(map[string]any)
	if !ok {
		t.Fatal("x-constants not found or wrong type")
	}

	if _, ok := xConsts["service"]; !ok {
		t.Error("service constant not in x-constants")
	}
}

func TestGenerateOpenAPI_RequiredFields(t *testing.T) {
	tpl := &tmpl.Template{
		Name: "test",
		Size: tmpl.Size{Width: "4in", Height: "6in"},
		Inputs: map[string]tmpl.InputField{
			"required_field": {Type: "string", Required: true},
			"optional_field": {Type: "string", Required: false},
		},
		Constants: map[string]tmpl.Constant{},
	}

	loader := makeLoaderWithTemplate(t, "test", tpl)
	spec := GenerateOpenAPI(loader)

	paths := spec["paths"].(map[string]any)
	post := paths["/labels/test"].(map[string]any)["post"].(map[string]any)
	schema := post["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required field not found or wrong type")
	}

	found := false

	for _, f := range required {
		if f == "required_field" {
			found = true
		}
	}

	if !found {
		t.Error("required_field not in required list")
	}
}

func TestGenerateOpenAPI_EmptyLoader(t *testing.T) {
	loader := tmpl.NewTemplateLoader()
	spec := GenerateOpenAPI(loader)

	paths := spec["paths"].(map[string]any)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestOpenAPIEndpoint(t *testing.T) {
	tpl := &tmpl.Template{
		Name:      "mytemplate",
		Size:      tmpl.Size{Width: "4in", Height: "6in"},
		Inputs:    map[string]tmpl.InputField{},
		Constants: map[string]tmpl.Constant{},
		Layout:    []tmpl.LayoutElement{},
	}
	loader := makeLoaderWithTemplate(t, "mytemplate", tpl)
	mux := New(loader, makeRendererForTest(t))

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var spec map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&spec); err != nil {
		t.Fatalf("decoding openapi spec: %v", err)
	}

	if spec["openapi"] != "3.0.3" {
		t.Errorf("openapi = %v, want 3.0.3", spec["openapi"])
	}
}
