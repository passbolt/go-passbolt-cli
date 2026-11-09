package util

import (
	"context"
	"errors"
	"os"

	"github.com/passbolt/go-passbolt-cli/internal/agent"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InvocationDialog is scoped to a serialized agent request so stdin can remain a
// machine protocol while authentication and selection use the caller's terminal.
var InvocationDialog *agent.Dialog

var RedactServerErrors bool

func CurrentDialog() agent.Dialog {
	if InvocationDialog != nil {
		return *InvocationDialog
	}
	return agent.Dialog{Backend: viper.GetString("agent-dialog"), Pinentry: viper.GetString("agent-pinentry"), Terminal: agent.CurrentTerminal(), PID: os.Getpid(), Environment: os.Environ()}
}

// WithMachineClient reserves stdout for data, including during login and cleanup.
// Authentication never consumes the command's stdin stream.
func WithMachineClient(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
	if viper.GetBool("debug") {
		return errors.New("debug logging is unavailable with machine-readable secret output")
	}
	previousOut, previousPrompt := Out, PasswordPrompt
	previousRedaction := RedactServerErrors
	RedactServerErrors = true
	Out = Err
	ctx, cancel := GetContext()
	defer cancel()
	if PasswordPrompt == nil {
		dialog := CurrentDialog()
		PasswordPrompt = func(prompt string) (string, error) { return dialog.Ask(ctx, "Passbolt authentication", prompt, true) }
	}
	defer func() { Out = previousOut; PasswordPrompt = previousPrompt; RedactServerErrors = previousRedaction }()
	return WithClient(cmd, fn)
}

// SessionKeyWarning avoids echoing server-controlled response bodies while a
// machine credential command is running, including its deferred cleanup.
func SessionKeyWarning(err error) {
	if RedactServerErrors {
		Eprintln("Warning: failed to save pending session keys")
		return
	}
	Eprintf("Warning: failed to save session keys: %v\n", err)
}
