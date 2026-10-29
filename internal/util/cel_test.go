package util

import (
	"testing"

	"cel.dev/cel-go/cel"
)

func TestInitCELProgram_InvalidExpression(t *testing.T) {
	if _, err := InitCELProgram("1 +"); err == nil {
		t.Fatal("expected error compiling invalid CEL expression")
	}
}

func TestInitCELProgram_Valid(t *testing.T) {
	p, err := InitCELProgram("1 == 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected a non-nil program")
	}
}

func TestCELExpressionReferencesFields(t *testing.T) {
	opts := []cel.EnvOption{
		cel.Variable("name", cel.StringType),
		cel.Variable("age", cel.IntType),
	}

	cases := []struct {
		name    string
		expr    string
		fields  []string
		want    bool
		wantErr bool
	}{
		{"empty expression", "", []string{"name"}, false, false},
		{"references listed field", `name == "x"`, []string{"name"}, true, false},
		{"references unlisted field", `age == 1`, []string{"name"}, false, false},
		{"invalid expression", `name ==`, []string{"name"}, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CELExpressionReferencesFields(c.expr, c.fields, opts...)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("CELExpressionReferencesFields(%q) = %v, want %v", c.expr, got, c.want)
			}
		})
	}
}
