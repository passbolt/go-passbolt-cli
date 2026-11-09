package agent

import (
	"context"
	"errors"
	"testing"
)

func TestNoninteractiveSelectionCancelsEvenWithDesktop(t *testing.T) {
	dialog := Dialog{Backend: "auto", Environment: []string{"DISPLAY=:0"}}
	if _, err := dialog.Select(context.Background(), "title", "prompt", []string{"one", "two"}); !errors.Is(err, ErrDenied) {
		t.Fatal("unattended selection did not cancel", err)
	}
}
