package visibility

import (
	"testing"
)

func TestEvaluate_EmptyExpr(t *testing.T) {
	visible, err := Evaluate("", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !visible {
		t.Error("empty expression should be visible")
	}
}

func TestEvaluate_Has(t *testing.T) {
	data := map[string]any{
		"tracking": "123ABC",
		"empty":    "",
	}

	tests := []struct {
		expr    string
		want    bool
		wantErr bool
	}{
		{"has(tracking)", true, false},
		{"has(missing_field)", false, false},
		{"has(empty)", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Evaluate(tt.expr, data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvaluate_Missing(t *testing.T) {
	data := map[string]any{
		"present": "value",
		"empty":   "",
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"missing(present)", false},
		{"missing(absent)", true},
		{"missing(empty)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Evaluate(tt.expr, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvaluate_Empty(t *testing.T) {
	data := map[string]any{
		"full":   "hello",
		"blank":  "",
		"nilval": nil,
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"empty(full)", false},
		{"empty(blank)", true},
		{"empty(nilval)", true},
		{"empty(absent)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Evaluate(tt.expr, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvaluate_NotEmpty(t *testing.T) {
	data := map[string]any{
		"full":  "hello",
		"blank": "",
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"not_empty(full)", true},
		{"not_empty(blank)", false},
		{"not_empty(absent)", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Evaluate(tt.expr, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvaluate_Eq(t *testing.T) {
	data := map[string]any{
		"level":  "EXPRESS",
		"count":  "5",
		"absent": nil,
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"eq(level,EXPRESS)", true},
		{"eq(level,STANDARD)", false},
		{"eq(missing_key,anything)", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Evaluate(tt.expr, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvaluate_Ne(t *testing.T) {
	data := map[string]any{
		"level": "EXPRESS",
	}

	tests := []struct {
		expr string
		want bool
	}{
		{"ne(level,EXPRESS)", false},
		{"ne(level,STANDARD)", true},
		{"ne(absent,anything)", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := Evaluate(tt.expr, data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEvaluate_InvalidExpr(t *testing.T) {
	tests := []string{
		"invalid",
		"has(",
		"has(a,b)", // has only accepts 1 arg
		"unknown(field)",
		"eq(only_one_arg)",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := Evaluate(expr, map[string]any{})
			if err == nil {
				t.Errorf("expected error for expression %q", expr)
			}
		})
	}
}

func TestEvaluate_NilData(t *testing.T) {
	got, err := Evaluate("has(field)", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got {
		t.Error("has() on nil data should be false")
	}
}

func TestEvaluate_WhitespaceExpr(t *testing.T) {
	got, err := Evaluate("   ", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got {
		t.Error("whitespace-only expression should be visible")
	}
}
