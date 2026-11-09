package cmd

import (
	"testing"
)

func TestNewCommandTreeDoesNotRetainFlags(t *testing.T) {
	a := newCommandTree()
	c, _, err := a.Find([]string{"create", "resource"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Flags().Set("field", "name=secret"); err != nil {
		t.Fatal(err)
	}
	b := newCommandTree()
	d, _, _ := b.Find([]string{"create", "resource"})
	values, _ := d.Flags().GetStringArray("field")
	if len(values) != 0 || d.Flags().Changed("field") {
		t.Fatal("request inherited another request's flags")
	}
	if c == d {
		t.Fatal("commands were reused")
	}
}
