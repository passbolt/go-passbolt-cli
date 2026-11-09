package agent

import (
	"context"
	"strings"
	"testing"
)

func TestPinentryDenialAndEscaping(t *testing.T) {
	if got := assuanEscape("a\nb%\rc"); got != "a%0Ab%25%0Dc" {
		t.Fatal(got)
	}
	for _, reply := range []string{"ERR 83886179 canceled\n", "BYE\n", ""} {
		_, err := readAssuan(strings.NewReader(reply))
		if err == nil {
			t.Fatal("accepted failed pinentry response")
		}
	}
	value, err := readAssuan(strings.NewReader("D p%25ss%0Aword\nOK\n"))
	if err != nil || value != "p%ss\nword" {
		t.Fatalf("%q %v", value, err)
	}
}
func TestDialogBackendValidation(t *testing.T) {
	_, err := (Dialog{Backend: "unknown"}).Ask(context.Background(), "title", "prompt", false)
	if err == nil {
		t.Fatal("accepted unknown backend")
	}
}
