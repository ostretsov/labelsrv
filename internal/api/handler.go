package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/ostretsov/labelsrv/internal/renderer"
	tmpl "github.com/ostretsov/labelsrv/internal/template"
)

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// RenderLabel returns an http.HandlerFunc for POST /labels/{template}.
func RenderLabel(loader *tmpl.TemplateLoader, r *renderer.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		templateName := req.PathValue("template")

		t, ok := loader.Get(templateName)
		if !ok {
			jsonError(w, fmt.Sprintf("template %q not found", templateName), http.StatusNotFound)

			return
		}

		var data map[string]any

		if req.Body != nil && req.ContentLength != 0 {
			if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
				jsonError(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)

				return
			}
		}

		if data == nil {
			data = make(map[string]any)
		}

		if err := validateInputs(t, data); err != nil {
			jsonError(w, err.Error(), http.StatusBadRequest)

			return
		}

		pdfBytes, err := r.Render(t, data)
		if err != nil {
			jsonError(w, fmt.Sprintf("rendering label: %v", err), http.StatusInternalServerError)

			return
		}

		accept := req.Header.Get("Accept")
		format := req.URL.Query().Get("format")

		if format == "pdf" || strings.Contains(accept, "application/pdf") {
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, templateName))

			_, _ = w.Write(pdfBytes)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(map[string]string{
			"pdf": base64.StdEncoding.EncodeToString(pdfBytes),
		})
	}
}

// validateInputs validates the request data against the template.
func validateInputs(t *tmpl.Template, data map[string]any) error {
	for constName := range t.Constants {
		if _, ok := data[constName]; ok {
			return fmt.Errorf("field %q is a constant and cannot be overridden", constName)
		}
	}

	for name, field := range t.Inputs {
		if !field.Required {
			continue
		}

		val, ok := data[name]
		if !ok || val == nil {
			return fmt.Errorf("required field %q is missing", name)
		}

		if s, ok := val.(string); ok && s == "" {
			return fmt.Errorf("required field %q is empty", name)
		}
	}

	for name, field := range t.Inputs {
		val, ok := data[name]
		if !ok {
			continue
		}

		s, ok := val.(string)
		if !ok {
			continue
		}

		if field.MaxLength > 0 && len(s) > field.MaxLength {
			return fmt.Errorf("field %q exceeds max_length of %d", name, field.MaxLength)
		}

		if field.Pattern != "" {
			re, err := regexp.Compile(field.Pattern)
			if err != nil {
				return fmt.Errorf("field %q has invalid pattern: %w", name, err)
			}

			if !re.MatchString(s) {
				return fmt.Errorf("field %q does not match required pattern", name)
			}
		}
	}

	return nil
}
