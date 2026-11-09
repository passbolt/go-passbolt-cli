package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/agent"
	"github.com/passbolt/go-passbolt-cli/internal/credential"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newGitCredentialCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "git-credential <get|store|erase>",
		Short: "Read credentials using a resource UUID in the credential URL path",
		Long:  "Read a Git credential request from stdin and return the selected resource's username and password. The URL path must be a Passbolt resource UUID; protocol and host identify the consumer, not the Passbolt server. Store and erase are no-ops.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return credential.ErrRequest
			}
			switch args[0] {
			case "get", "store", "erase":
				return nil
			}
			return credential.ErrRequest
		},
		RunE: gitCredential,
	}
}
func init() { rootCmd.AddCommand(newGitCredentialCommand()) }

func gitCredential(cmd *cobra.Command, args []string) error {
	// Never persist, alter or delete Passbolt resources in response to Git hooks.
	if args[0] != "get" {
		return nil
	}
	if viper.GetBool("debug") {
		return credential.ErrLookup
	}
	req, err := credential.ReadRequest(cmd.InOrStdin())
	if err != nil {
		return err
	}
	if !inAgent && viper.GetBool("agent") {
		// Validation happens before starting the agent or asking for approval. Replay
		// the consumed protocol fields through its demand-driven stdin channel.
		cmd.SetIn(strings.NewReader(req.Encode()))
		if err := callAgent(cmd, "run"); err != nil {
			return credential.ErrLookup
		}
		return nil
	}
	ctx, cancel := util.GetContext()
	defer cancel()
	oldOut, oldErr := util.Out, util.Err
	oldPrompt := util.PasswordPrompt
	// Auth progress and API error text must never enter protocol output or expose
	// values embedded in server errors. The caller gets a generic lookup error.
	util.Out, util.Err = io.Discard, io.Discard
	defer func() { util.Out, util.Err = oldOut, oldErr; util.PasswordPrompt = oldPrompt }()
	if oldPrompt == nil {
		dialog := agent.Dialog{Backend: viper.GetString("agent-dialog"), Pinentry: viper.GetString("agent-pinentry"), Terminal: agent.CurrentTerminal(), PID: os.Getpid(), Environment: os.Environ()}
		util.PasswordPrompt = func(prompt string) (string, error) {
			return dialog.Ask(ctx, "Passbolt credential authentication", prompt, true)
		}
	}
	client, err := util.GetClient(ctx)
	if err != nil {
		return credential.ErrLookup
	}
	defer util.SaveSessionKeysAndLogout(ctx, client)
	_, _, username, _, password, _, err := helper.GetResource(ctx, client, req.ResourceID)
	if err != nil {
		return credential.ErrLookup
	}
	if req.Username != "" && username != "" && req.Username != username {
		return credential.ErrLookup
	}
	if username == "" {
		username = req.Username
	}
	if err := credential.WriteResponse(util.DataOut, username, password); err != nil {
		return credential.ErrLookup
	}
	return nil
}
