package api

import (
	tmpl "github.com/ostretsov/labelsrv/internal/template"
)

// GenerateOpenAPI generates an OpenAPI 3.0 spec as a map.
func GenerateOpenAPI(loader *tmpl.TemplateLoader) map[string]any {
	paths := make(map[string]any)

	for name, t := range loader.All() {
		properties := make(map[string]any)
		required := []string{}

		for fieldName, field := range t.Inputs {
			prop := map[string]any{
				"type": "string",
			}

			if field.Description != "" {
				prop["description"] = field.Description
			}

			if field.MaxLength > 0 {
				prop["maxLength"] = field.MaxLength
			}

			if field.Pattern != "" {
				prop["pattern"] = field.Pattern
			}

			if field.Type != "" && field.Type != "string" {
				prop["type"] = field.Type
			}

			properties[fieldName] = prop

			if field.Required {
				required = append(required, fieldName)
			}
		}

		requestSchema := map[string]any{
			"type":       "object",
			"properties": properties,
		}

		if len(required) > 0 {
			requestSchema["required"] = required
		}

		xConstants := make(map[string]any)

		for constName, c := range t.Constants {
			xConstants[constName] = map[string]any{
				"type":        c.Type,
				"value":       c.Value,
				"locked":      c.Locked,
				"description": c.Description,
			}
		}

		operation := map[string]any{
			"summary":     "Render label: " + name,
			"description": "Renders the " + name + " label template to PDF",
			"operationId": "renderLabel_" + name,
			"tags":        []string{"labels"},
			"requestBody": map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": requestSchema,
					},
				},
			},
			"parameters": []map[string]any{
				{
					"name":        "format",
					"in":          "query",
					"description": "Response format: 'pdf' for raw PDF, or omit for base64 JSON",
					"schema": map[string]any{
						"type": "string",
						"enum": []string{"pdf"},
					},
				},
			},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Successfully rendered label",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"pdf": map[string]any{
										"type":        "string",
										"format":      "byte",
										"description": "Base64-encoded PDF bytes",
									},
								},
							},
						},
						"application/pdf": map[string]any{
							"schema": map[string]any{
								"type":   "string",
								"format": "binary",
							},
						},
					},
				},
				"400": map[string]any{
					"description": "Invalid request data",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"$ref": "#/components/schemas/Error",
							},
						},
					},
				},
				"404": map[string]any{
					"description": "Template not found",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"$ref": "#/components/schemas/Error",
							},
						},
					},
				},
				"500": map[string]any{
					"description": "Internal server error",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"$ref": "#/components/schemas/Error",
							},
						},
					},
				},
			},
			"x-constants": xConstants,
		}

		paths["/labels/"+name] = map[string]any{
			"post": operation,
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "labelsrv API",
			"description": "Configuration-driven label rendering server",
			"version":     "1.0.0",
		},
		"paths": paths,
		"components": map[string]any{
			"schemas": map[string]any{
				"Error": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"error": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}
}
