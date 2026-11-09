package util

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestMachineClientKeepsAuthenticationAndCleanupOffStdout(t *testing.T) {
	oldOut, oldErr, oldData := Out, Err, DataOut
	oldProvider, oldRelease := ClientProvider, ClientRelease
	oldPrompt := PasswordPrompt
	defer func() {
		Out, Err, DataOut = oldOut, oldErr, oldData
		ClientProvider, ClientRelease = oldProvider, oldRelease
		PasswordPrompt = oldPrompt
		viper.Reset()
	}()
	var stdout, stderr bytes.Buffer
	Out, Err, DataOut = &stdout, &stderr, &stdout
	viper.Reset()
	viper.Set("timeout", "1s")
	ClientProvider = func(context.Context) (*api.Client, error) {
		Println("authentication progress")
		return &api.Client{}, nil
	}
	ClientRelease = func(context.Context, *api.Client) {
		Println("cleanup warning")
		SessionKeyWarning(errors.New("private-error-token"))
	}
	err := WithMachineClient(&cobra.Command{}, func(context.Context, *api.Client) error { _, err := io.WriteString(DataOut, "raw-token"); return err })
	if strings.Contains(stderr.String(), "private-error-token") {
		t.Fatal("cleanup leaked a server error")
	}
	if err != nil || stdout.String() != "raw-token" || !strings.Contains(stderr.String(), "authentication progress") || !strings.Contains(stderr.String(), "cleanup warning") {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout.String(), stderr.String(), err)
	}
}
