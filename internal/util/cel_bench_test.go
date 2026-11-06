package util

// Benchmarks for the CEL/column filtering path. ColumnRegistry.Filter runs a
// compiled CEL program against every item in a list, rebuilding an eval map per
// item — an O(n) cost paid on every `list --filter` invocation. The size-tiered
// sub-benchmarks expose how that cost scales with list length.

import (
	"context"
	"fmt"
	"testing"

	"cel.dev/cel-go/cel"
)

// benchFilterSizes spans the realistic range: a small vault (100), a large team
// vault (1000), and an enterprise-scale list (10000) where the per-item eval
// cost dominates.
var benchFilterSizes = []int{100, 1000, 10000}

// benchRow is a minimal filterable entity for the registry benchmarks.
type benchRow struct {
	name string
	age  int
}

func benchRegistry() *ColumnRegistry[benchRow] {
	return NewColumnRegistry("rows", []ColumnSpec[benchRow]{
		{
			Name:       "name",
			CelType:    cel.StringType,
			CelValue:   func(r benchRow) any { return r.name },
			TableValue: func(r benchRow) string { return r.name },
		},
		{
			Name:       "age",
			CelType:    cel.IntType,
			CelValue:   func(r benchRow) any { return r.age },
			TableValue: func(r benchRow) string { return fmt.Sprintf("%d", r.age) },
		},
	})
}

func benchRows(n int) []benchRow {
	rows := make([]benchRow, n)
	for i := range rows {
		rows[i] = benchRow{name: fmt.Sprintf("row-%d", i), age: i % 100}
	}
	return rows
}

// boolSink defeats dead-code elimination for the filtered result.
var benchFilterSink int

// BenchmarkInitCELProgram measures one-time CEL compilation cost: env build +
// compile + program assembly. This is paid once per Filter call regardless of
// list size.
func BenchmarkInitCELProgram(b *testing.B) {
	opts := []cel.EnvOption{
		cel.Variable("name", cel.StringType),
		cel.Variable("age", cel.IntType),
	}
	const expr = `name.startsWith("row-") && age >= 50`

	b.ReportAllocs()
	for b.Loop() {
		p, err := InitCELProgram(expr, opts...)
		if err != nil {
			b.Fatalf("InitCELProgram: %v", err)
		}
		if p == nil {
			b.Fatal("nil program")
		}
	}
}

// BenchmarkColumnRegistry_Filter measures the full per-list filter: compile once
// then eval over every item. The expression matches roughly half the rows so
// Filter never hits its zero-match error path.
func BenchmarkColumnRegistry_Filter(b *testing.B) {
	reg := benchRegistry()
	ctx := context.Background()
	const expr = `age >= 50`

	for _, n := range benchFilterSizes {
		rows := benchRows(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				out, err := reg.Filter(ctx, rows, expr)
				if err != nil {
					b.Fatalf("Filter: %v", err)
				}
				benchFilterSink = len(out)
			}
		})
	}
}
