package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/credential"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const PassboltPrefix = "passbolt://"

// execCmd represents the exec command
var execCmd = makeexecCmd()

func makeexecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec -- command [args...]",
		Short: "Run a command with secrets injected into the environment.",
		Long: `The command allows you to execute another command with environment variables that reference secrets stored in Passbolt.
Any environment variables containing passbolt:// references are automatically resolved to their corresponding secret values
	before the specified command is executed. This ensures that secrets are securely injected into the child process's environment
	without exposing them to the parent shell.

	For example:
	export GITHUB_TOKEN=passbolt://<PASSBOLT_RESOURCE_ID_HERE>
	passbolt exec -- gh auth login

	This would resolve the passbolt:// reference in GITHUB_TOKEN to its actual secret value and pass it to the gh process.
`,
		Args: cobra.MinimumNArgs(1),
		RunE: execAction,
	}
}

func configureExec() {
	rootCmd.AddCommand(execCmd)
}

func execAction(_ *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(util.InvocationContext, viper.GetDuration("timeout"))
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	defer util.SaveSessionKeysAndLogout(ctx, client)

	envVars, err := resolveEnvironmentSecrets(ctx, client)
	if err != nil {
		return fmt.Errorf("resolving secrets: %w", err)
	}

	if inAgent {
		agentExecEnvironment = envVars
		agentExecArgs = append([]string(nil), args...)
		return nil
	}

	subCmd := exec.Command(args[0], args[1:]...)
	subCmd.Stdin = os.Stdin
	// The child owns these streams; an io.Writer wrapper would cost it its tty.
	subCmd.Stdout = os.Stdout //nolint:forbidigo // raw fd passthrough
	subCmd.Stderr = os.Stderr //nolint:forbidigo // raw fd passthrough
	subCmd.Env = envVars

	if err = subCmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}

func resolveEnvironmentSecrets(ctx context.Context, client *api.Client) ([]string, error) {
	envVars := os.Environ()
	if inAgent {
		envVars = append([]string(nil), invocationEnvironment...)
	}

	for i, envVar := range envVars {
		before, after, ok := strings.Cut(envVar, "=")
		if !ok {
			continue
		}

		key := before
		value := after

		if !strings.HasPrefix(value, PassboltPrefix) {
			continue
		}

		resourceID := strings.TrimPrefix(value, PassboltPrefix)
		data, err := (credential.ClientStore{Client: client, Approve: approveCredentialResource}).Load(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("getting resource: %w", err)
		}

		secret := helper.GetStringField(data.Secret, "password")
		envVars[i] = key + "=" + secret

		if viper.GetBool("debug") {
			util.Printf("%v env var populated with resource id %v\n", key, resourceID)
		}
	}

	return envVars, nil
}

func init() { configureExec() }
