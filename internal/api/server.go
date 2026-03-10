package api

import (
	"encoding/json"
	"net/http"

	"github.com/ostretsov/labelsrv/internal/renderer"
	tmpl "github.com/ostretsov/labelsrv/internal/template"
)

const redocHTML = `<!DOCTYPE html>
<html>
  <head>
    <title>labelsrv API</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>body { margin: 0; padding: 0; }</style>
  </head>
  <body>
    <redoc spec-url="/openapi.json"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
  </body>
</html>`

func serveRedoc(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, _ = w.Write([]byte(redocHTML))
}

// New creates and configures an http.ServeMux.
func New(loader *tmpl.TemplateLoader, r *renderer.Renderer) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(GenerateOpenAPI(loader))
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"templates": loader.List(),
		})
	})

	mux.HandleFunc("GET /docs", serveRedoc)

	mux.HandleFunc("POST /labels/{template}", RenderLabel(loader, r))

	return mux
}
