package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseSize converts a size string like "6in", "4in", "50mm", "10cm", "20pt" to mm.
func ParseSize(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	var (
		numStr string
		unit   string
	)

	for i, c := range s {
		if c >= '0' && c <= '9' || c == '.' {
			numStr = s[:i+1]
		} else {
			unit = strings.ToLower(strings.TrimSpace(s[i:]))
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid size string: %q", s)
	}

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value in size %q: %w", s, err)
	}

	switch unit {
	case "in":
		return val * 25.4, nil
	case "cm":
		return val * 10, nil
	case "mm":
		return val, nil
	case "pt":
		return val * 0.352778, nil
	default:
		return 0, fmt.Errorf("unknown unit %q in size %q", unit, s)
	}
}

// ParseFile loads a YAML or JSON template file.
func ParseFile(path string) (*Template, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from validated template directory
	if err != nil {
		return nil, fmt.Errorf("reading file %q: %w", path, err)
	}

	var t Template

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parsing YAML %q: %w", path, err)
		}
	case ".json":
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parsing JSON %q: %w", path, err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &t); err != nil {
			if err2 := json.Unmarshal(data, &t); err2 != nil {
				return nil, fmt.Errorf("parsing file %q: not valid YAML or JSON", path)
			}
		}
	}

	return &t, nil
}

// Validate checks that a template has all required fields.
func Validate(t *Template) error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}

	if t.Size.Width == "" {
		return fmt.Errorf("template size.width is required")
	}

	if t.Size.Height == "" {
		return fmt.Errorf("template size.height is required")
	}

	if _, err := ParseSize(t.Size.Width); err != nil {
		return fmt.Errorf("invalid size.width: %w", err)
	}

	if _, err := ParseSize(t.Size.Height); err != nil {
		return fmt.Errorf("invalid size.height: %w", err)
	}

	validInputTypes := map[string]bool{
		"string": true,
		"number": true,
		"bool":   true,
		"":       true,
	}

	for name, input := range t.Inputs {
		if !validInputTypes[input.Type] {
			return fmt.Errorf("input %q has invalid type %q", name, input.Type)
		}
	}

	validElementTypes := map[string]bool{
		"text":    true,
		"barcode": true,
		"image":   true,
		"line":    true,
		"rect":    true,
		"textbox": true,
	}

	for i, el := range t.Layout {
		if el.ID == "" {
			return fmt.Errorf("layout element at index %d is missing an id", i)
		}

		if !validElementTypes[el.Type] {
			return fmt.Errorf("layout element %q has invalid type %q", el.ID, el.Type)
		}
	}

	return nil
}
