package util

import (
	"context"
	"fmt"
	"strings"

	"cel.dev/cel-go/cel"
)

// ColumnAlias declares a canonical column name together with the legacy input
// forms that should resolve to it. Used by list commands to accept
// snake_case (canonical), PascalCase, and lowercased-concat forms uniformly
// in --column and --filter flags.
type ColumnAlias struct {
	Name    string
	Aliases []string
}

// ColumnAliasResolver normalizes user-supplied column names to their
// canonical form via case-insensitive lookup against name+aliases.
type ColumnAliasResolver struct {
	canonical []string
	lookup    map[string]string
}

func NewColumnAliasResolver(specs []ColumnAlias) *ColumnAliasResolver {
	r := &ColumnAliasResolver{
		canonical: make([]string, 0, len(specs)),
		lookup:    make(map[string]string),
	}
	for _, s := range specs {
		r.canonical = append(r.canonical, s.Name)
		r.lookup[strings.ToLower(s.Name)] = s.Name
		for _, a := range s.Aliases {
			r.lookup[strings.ToLower(a)] = s.Name
		}
	}
	return r
}

// Normalize maps an input column name to its canonical form. Returns an
// error listing the valid canonical names if the input matches nothing.
func (r *ColumnAliasResolver) Normalize(input string) (string, error) {
	if canon, ok := r.lookup[strings.ToLower(input)]; ok {
		return canon, nil
	}
	return "", fmt.Errorf("unknown column: %q (valid: %s)", input, strings.Join(r.canonical, ", "))
}

// NormalizeAll normalizes a slice of input column names, returning the
// canonical-name slice or the first error encountered.
func (r *ColumnAliasResolver) NormalizeAll(inputs []string) ([]string, error) {
	out := make([]string, len(inputs))
	for i, in := range inputs {
		canon, err := r.Normalize(in)
		if err != nil {
			return nil, err
		}
		out[i] = canon
	}
	return out, nil
}

// Canonical returns the canonical column names in declaration order. Useful
// for building --column flag help text and default value slices.
func (r *ColumnAliasResolver) Canonical() []string {
	out := make([]string, len(r.canonical))
	copy(out, r.canonical)
	return out
}

// ColumnSpec describes one --column / --filter attribute on an entity of type
// T. The canonical Name is snake_case (matching JSON tags and Passbolt API
// conventions); Aliases keep legacy PascalCase and lowercased-concat forms
// working for backwards compatibility, both as CEL variables and as eval-map
// keys.
type ColumnSpec[T any] struct {
	Name            string
	Aliases         []string
	DefaultTable    bool
	RequiresSecrets bool // value depends on the expensive secret-decryption path
	CelType         *cel.Type
	CelValue        func(item T) any
	TableValue      func(item T) string
}

// ColumnRegistry holds the column specs for one entity type and precomputes the
// lookups and CEL options derived from them. It owns the shared CEL filter
// loop, so each entity package only declares its columns once.
type ColumnRegistry[T any] struct {
	columns       []ColumnSpec[T]
	byName        map[string]ColumnSpec[T]
	resolver      *ColumnAliasResolver
	defaultCols   []string
	celEnvOptions []cel.EnvOption
	secretNames   []string
	noun          string // plural entity word for the "no such <noun> found" error
}

// NewColumnRegistry builds a registry from the given column specs. noun is the
// plural entity word used in the "no such <noun> found with filter" error
// returned by Filter when nothing matches. Each canonical name and every alias
// is registered as a CEL variable so legacy and canonical filter expressions
// both compile.
func NewColumnRegistry[T any](noun string, columns []ColumnSpec[T]) *ColumnRegistry[T] {
	r := &ColumnRegistry[T]{
		columns: columns,
		byName:  make(map[string]ColumnSpec[T], len(columns)),
		noun:    noun,
	}
	aliases := make([]ColumnAlias, len(columns))
	for i, c := range columns {
		r.byName[c.Name] = c
		aliases[i] = ColumnAlias{Name: c.Name, Aliases: c.Aliases}
		if c.DefaultTable {
			r.defaultCols = append(r.defaultCols, c.Name)
		}
		r.celEnvOptions = append(r.celEnvOptions, cel.Variable(c.Name, c.CelType))
		for _, a := range c.Aliases {
			r.celEnvOptions = append(r.celEnvOptions, cel.Variable(a, c.CelType))
		}
		if c.RequiresSecrets {
			r.secretNames = append(r.secretNames, c.Name)
			r.secretNames = append(r.secretNames, c.Aliases...)
		}
	}
	r.resolver = NewColumnAliasResolver(aliases)
	return r
}

// Resolver returns the alias resolver for normalizing --column / --filter
// input to canonical names and for building flag help text.
func (r *ColumnRegistry[T]) Resolver() *ColumnAliasResolver { return r.resolver }

// DefaultTableColumns returns the canonical names of columns shown in the
// default table output, in declaration order.
func (r *ColumnRegistry[T]) DefaultTableColumns() []string {
	out := make([]string, len(r.defaultCols))
	copy(out, r.defaultCols)
	return out
}

// CelEnvOptions returns the CEL environment options registering every canonical
// name and alias as a variable. Pass to util.CELExpressionReferencesFields.
func (r *ColumnRegistry[T]) CelEnvOptions() []cel.EnvOption {
	out := make([]cel.EnvOption, len(r.celEnvOptions))
	copy(out, r.celEnvOptions)
	return out
}

// SecretCelNames returns all CEL variable names (canonical and alias) whose
// value comes from secret decryption — used to detect whether a --filter
// expression forces secret fetching.
func (r *ColumnRegistry[T]) SecretCelNames() []string {
	out := make([]string, len(r.secretNames))
	copy(out, r.secretNames)
	return out
}

// RequiresSecrets reports whether any of the (already-normalized, canonical)
// columns demands secret decryption.
func (r *ColumnRegistry[T]) RequiresSecrets(columns []string) bool {
	for _, col := range columns {
		if spec, ok := r.byName[col]; ok && spec.RequiresSecrets {
			return true
		}
	}
	return false
}

// TableValue resolves a canonical column name to the item's rendered table
// cell. Suitable as the valueFn for PrintTable; returns false for unknown
// columns.
func (r *ColumnRegistry[T]) TableValue(item T, col string) (string, bool) {
	spec, ok := r.byName[col]
	if !ok {
		return "", false
	}
	return spec.TableValue(item), true
}

// evalMap builds the variable→value map passed to ContextEval, populating both
// canonical and alias keys with the same value.
func (r *ColumnRegistry[T]) evalMap(item T) map[string]any {
	m := make(map[string]any, len(r.columns)*2)
	for _, c := range r.columns {
		v := c.CelValue(item)
		m[c.Name] = v
		for _, a := range c.Aliases {
			m[a] = v
		}
	}
	return m
}

// Filter evaluates celCmd against each item and returns the matching subset.
// An empty celCmd returns items unchanged; a valid expression matching nothing
// returns an error naming the entity.
func (r *ColumnRegistry[T]) Filter(ctx context.Context, items []T, celCmd string) ([]T, error) {
	if celCmd == "" {
		return items, nil
	}

	program, err := InitCELProgram(celCmd, r.celEnvOptions...)
	if err != nil {
		return nil, err
	}

	filtered := []T{}
	for _, item := range items {
		val, _, err := (*program).ContextEval(ctx, r.evalMap(item))
		if err != nil {
			return nil, err
		}
		if val.Value() == true {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no such %s found with filter %v", r.noun, celCmd)
	}
	return filtered, nil
}
