package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ostretsov/labelsrv/internal/renderer"
	tmpl "github.com/ostretsov/labelsrv/internal/template"
)

func makeLoader(t *testing.T) *tmpl.TemplateLoader {
	t.Helper()

	loader := tmpl.NewTemplateLoader()
	loader.ForTest("shipping", &tmpl.Template{
		Name: "shipping",
		Size: tmpl.Size{Width: "6in", Height: "4in"},
		Inputs: map[string]tmpl.InputField{
			"recipient": {Type: "string", Required: true, MaxLength: 50},
			"tracking":  {Type: "string", Required: false},
		},
		Constants: map[string]tmpl.Constant{
			"service": {Type: "string", Value: "EXPRESS", Locked: true},
		},
		Layout: []tmpl.LayoutElement{
			{ID: "title", Type: "text", Source: "input", Key: "recipient", X: 10, Y: 10, FontSize: 12},
		},
	})

	return loader
}

func makeApp(t *testing.T) *http.ServeMux {
	t.Helper()
	loader := makeLoader(t)

	r, err := renderer.New("")
	if err != nil {
		t.Fatalf("renderer.New() error: %v", err)
	}

	return New(loader, r)
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string, headers map[string]string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec.Result()
}

func TestRenderLabel_Success_JSON(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/shipping", `{"recipient":"Alice"}`, nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, b)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if result["pdf"] == "" {
		t.Error("expected non-empty pdf field in response")
	}
}

func TestRenderLabel_Success_PDF_QueryParam(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/shipping?format=pdf", `{"recipient":"Bob"}`, nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, b)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/pdf") {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}

	pdfBytes, _ := io.ReadAll(resp.Body)
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Error("response body is not a valid PDF")
	}
}

func TestRenderLabel_Success_PDF_AcceptHeader(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/shipping", `{"recipient":"Carol"}`,
		map[string]string{"Accept": "application/pdf"})

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body: %s", resp.StatusCode, b)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/pdf") {
		t.Errorf("Content-Type = %q, want application/pdf", ct)
	}
}

func TestRenderLabel_TemplateNotFound(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/nonexistent", `{}`, nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestRenderLabel_MissingRequiredField(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/shipping", `{}`, nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, b)
	}
}

func TestRenderLabel_ConstantOverride(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/shipping", `{"recipient":"Dave","service":"STANDARD"}`, nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, b)
	}
}

func TestRenderLabel_MaxLengthViolation(t *testing.T) {
	mux := makeApp(t)
	body, _ := json.Marshal(map[string]string{"recipient": strings.Repeat("x", 51)})

	resp := do(t, mux, http.MethodPost, "/labels/shipping", string(body), nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, b)
	}
}

func TestRenderLabel_EmptyBody(t *testing.T) {
	mux := makeApp(t)

	resp := do(t, mux, http.MethodPost, "/labels/shipping", "", nil)

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want 400; body: %s", resp.StatusCode, b)
	}
}

func TestHealth(t *testing.T) {
	mux := makeApp(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestValidateInputs_PatternMatch(t *testing.T) {
	tpl := &tmpl.Template{
		Inputs: map[string]tmpl.InputField{
			"code": {Type: "string", Pattern: `^\d{5}$`},
		},
		Constants: map[string]tmpl.Constant{},
	}

	if err := validateInputs(tpl, map[string]any{"code": "12345"}); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}

	if err := validateInputs(tpl, map[string]any{"code": "abcde"}); err == nil {
		t.Error("expected validation error for pattern mismatch")
	}
}
