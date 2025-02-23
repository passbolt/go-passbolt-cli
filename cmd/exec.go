package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/passbolt/go-passbolt-cli/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const PassboltPrefix = "passbolt://"

// execCmd represents the exec command
var execCmd = &cobra.Command{
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

func init() {
	rootCmd.AddCommand(execCmd)
}

func execAction(_ *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("timeout"))
	defer cancel()

	client, err := util.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("Creating client: %w", err)
	}

	envVars, err := resolveEnvironmentSecrets(ctx, client)
	if err != nil {
		return fmt.Errorf("Resolving secrets: %w", err)
	}

	if err = client.Logout(ctx); err != nil {
		return fmt.Errorf("Logging out client: %w", err)
	}

	subCmd := exec.Command(args[0], args[1:]...)
	subCmd.Stdin = os.Stdin
	subCmd.Stdout = os.Stdout
	subCmd.Stderr = os.Stderr
	subCmd.Env = envVars

	if err = subCmd.Run(); err != nil {
		return fmt.Errorf("Running command: %w", err)
	}

	return nil
}

func resolveEnvironmentSecrets(ctx context.Context, client *api.Client) ([]string, error) {
	envVars := os.Environ()

	for i, envVar := range envVars {
		splitIndex := strings.Index(envVar, "=")
		if splitIndex == -1 {
			continue
		}

		key := envVar[:splitIndex]
		value := envVar[splitIndex+1:]

		if !strings.HasPrefix(value, PassboltPrefix) {
			continue
		}

		resourceId := strings.TrimPrefix(value, PassboltPrefix)
		_, _, _, _, secret, _, err := helper.GetResource(ctx, client, resourceId)
		if err != nil {
			return nil, fmt.Errorf("Getting resource: %w", err)
		}

		envVars[i] = key + "=" + secret

		if viper.GetBool("debug") {
			fmt.Fprintf(os.Stdout, "%v env var populated with resource id %v\n", key, resourceId)
		}
	}

	return envVars, nil
}
