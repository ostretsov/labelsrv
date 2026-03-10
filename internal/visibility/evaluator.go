package visibility

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reFuncCall = regexp.MustCompile(`^(\w+)\(([^)]*)\)$`)
)

// Evaluate evaluates a visible_if expression against the given data.
// Returns true if the element should be visible, false otherwise.
// If the expression is empty, the element is always visible.
func Evaluate(expr string, data map[string]any) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}

	m := reFuncCall.FindStringSubmatch(expr)
	if m == nil {
		return false, fmt.Errorf("invalid visible_if expression: %q", expr)
	}

	fn := m[1]

	args := strings.Split(m[2], ",")
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}

	switch fn {
	case "has":
		if len(args) != 1 {
			return false, fmt.Errorf("has() requires exactly 1 argument")
		}

		return hasField(data, args[0]), nil

	case "missing":
		if len(args) != 1 {
			return false, fmt.Errorf("missing() requires exactly 1 argument")
		}

		return !hasField(data, args[0]), nil

	case "empty":
		if len(args) != 1 {
			return false, fmt.Errorf("empty() requires exactly 1 argument")
		}

		return isEmptyField(data, args[0]), nil

	case "not_empty":
		if len(args) != 1 {
			return false, fmt.Errorf("not_empty() requires exactly 1 argument")
		}

		return !isEmptyField(data, args[0]), nil

	case "eq":
		if len(args) != 2 {
			return false, fmt.Errorf("eq() requires exactly 2 arguments")
		}

		return fieldEquals(data, args[0], args[1]), nil

	case "ne":
		if len(args) != 2 {
			return false, fmt.Errorf("ne() requires exactly 2 arguments")
		}

		return !fieldEquals(data, args[0], args[1]), nil

	default:
		return false, fmt.Errorf("unknown function %q in visible_if expression", fn)
	}
}

// hasField returns true if the field exists in data and is non-empty.
func hasField(data map[string]any, field string) bool {
	val, ok := data[field]
	if !ok {
		return false
	}

	if val == nil {
		return false
	}

	if s, ok := val.(string); ok {
		return s != ""
	}

	return true
}

// isEmptyField returns true if the field is absent or is an empty string.
func isEmptyField(data map[string]any, field string) bool {
	val, ok := data[field]
	if !ok {
		return true
	}

	if val == nil {
		return true
	}

	if s, ok := val.(string); ok {
		return s == ""
	}

	return false
}

// fieldEquals returns true if the field value equals the given string value.
func fieldEquals(data map[string]any, field, value string) bool {
	val, ok := data[field]
	if !ok {
		return false
	}

	if val == nil {
		return value == ""
	}

	s := fmt.Sprintf("%v", val)

	return s == value
}
