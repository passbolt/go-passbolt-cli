package cmd

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"

	dockercredentials "github.com/docker/docker-credential-helpers/credentials"
	"github.com/passbolt/go-passbolt-cli/internal/credential"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newGitCredentialCommand() *cobra.Command {
	return &cobra.Command{
		Use: "git-credential <get|store|erase>", Short: "Read Git credentials by matching resource service URLs",
		Long: "Match the requested HTTP(S) credential context against Passbolt resource URI fields. Ambiguous lookups require an interactive terminal and explicit selection. Store and erase are no-ops.",
		Args: func(_ *cobra.Command, args []string) error {
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
func newDockerCredentialCommand() *cobra.Command {
	return &cobra.Command{Use: "docker-credential <get|list|store|erase>", Short: "Read Docker registry credentials from Passbolt", Args: func(_ *cobra.Command, args []string) error {
		if len(args) != 1 {
			return credential.ErrRequest
		}
		switch args[0] {
		case "get", "list", "store", "erase":
			return nil
		}
		return credential.ErrRequest
	}, RunE: dockerCredential}
}
func newCredentialHelperCommand() *cobra.Command {
	root := &cobra.Command{Use: "credential-helper", Short: "Read credentials using standard Git or Docker protocols"}
	git := newGitCredentialCommand()
	git.Use = "git <get|store|erase>"
	docker := newDockerCredentialCommand()
	docker.Use = "docker <get|list|store|erase>"
	root.AddCommand(git, docker, newCloudCredentialCommand("aws"), newCloudCredentialCommand("k8s"))
	return root
}
func init() {
	rootCmd.AddCommand(newGitCredentialCommand(), newDockerCredentialCommand(), newCredentialHelperCommand())
}

func isCredentialCommand(cmd *cobra.Command) bool {
	if cmd.Name() == "git-credential" || cmd.Name() == "docker-credential" {
		return true
	}
	return cmd.Parent() != nil && cmd.Parent().Name() == "credential-helper"
}
func gitCredential(cmd *cobra.Command, args []string) error {
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
		cmd.SetIn(strings.NewReader(req.Encode()))
		if err := callAgent(cmd, "run"); err != nil {
			return credential.ErrLookup
		}
		return nil
	}
	err = withCredentialClient(cmd, func(ctx context.Context, resolver credential.Resolver) error {
		username, password, err := resolver.Git(ctx, req)
		if errors.Is(err, credential.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return credential.WriteResponse(util.DataOut, username, password)
	})
	if err != nil {
		return credential.ErrLookup
	}
	return nil
}
func dockerCredential(cmd *cobra.Command, args []string) error {
	action := args[0]
	if action == "store" || action == "erase" {
		if err := credential.HandleDocker(&credential.DockerBackend{}, action, cmd.InOrStdin(), io.Discard); err != nil {
			return credential.ErrReadOnly
		}
		return nil
	}
	if viper.GetBool("debug") {
		return credential.ErrLookup
	}
	var original string
	var err error
	if action == "get" {
		original, _, err = credential.ReadDockerServer(cmd.InOrStdin())
		if err != nil {
			return err
		}
	}
	if !inAgent && viper.GetBool("agent") {
		if action == "get" {
			cmd.SetIn(strings.NewReader(original + "\n"))
		}
		if err := callAgent(cmd, "run"); err != nil {
			return credential.ErrLookup
		}
		return nil
	}
	err = withCredentialClient(cmd, func(ctx context.Context, resolver credential.Resolver) error {
		input := io.Reader(strings.NewReader(""))
		if action == "get" {
			input = strings.NewReader(original + "\n")
		}
		return credential.HandleDocker(&credential.DockerBackend{Context: ctx, Resolver: resolver}, action, input, util.DataOut)
	})
	if dockercredentials.IsErrCredentialsNotFound(err) {
		// This fixed message is Docker protocol data, not a diagnostic: upstream
		// client.Get distinguishes a missing entry from failure using helper stdout.
		if _, writeErr := io.WriteString(util.DataOut, dockercredentials.NewErrCredentialsNotFound().Error()+"\n"); writeErr != nil {
			return writeErr
		}
		return credential.ErrNotFound
	}
	if err != nil {
		return credential.ErrLookup
	}
	return nil
}
func withCredentialClient(cmd *cobra.Command, fn func(context.Context, credential.Resolver) error) error {
	return withCredentialAPI(cmd, func(ctx context.Context, client *api.Client) error {
		dialog := util.CurrentDialog()
		resolver := credential.Resolver{Store: credential.ClientStore{Client: client, Approve: approveCredentialResource}, Select: func(ctx context.Context, matches []credential.Match) (int, error) {
			labels := make([]string, len(matches))
			for i, match := range matches {
				labels[i] = credential.MatchLabel(match)
			}
			return dialog.Select(ctx, "Passbolt credentials", "Select the matching service URL and account", labels)
		}}
		return fn(ctx, resolver)
	})
}

func withCredentialAPI(cmd *cobra.Command, fn func(context.Context, *api.Client) error) error {
	oldOut, oldErr := util.Out, util.Err
	util.Out, util.Err = io.Discard, io.Discard
	defer func() { util.Out, util.Err = oldOut, oldErr }()
	return util.WithMachineClient(cmd, fn)
}

// HelperArguments lets release binaries or symlinks use conventional helper
// names while sharing the same implementation and authenticated agent.
func HelperArguments(program string, args []string) []string {
	name := strings.TrimSuffix(strings.ToLower(filepath.Base(program)), ".exe")
	var protocol string
	switch name {
	case "git-credential-passbolt":
		protocol = "git"
	case "docker-credential-passbolt":
		protocol = "docker"
	case "passbolt-browser-host":
		return append([]string{"browser", "host"}, args...)
	default:
		return args
	}
	result := []string{"credential-helper", protocol}
	return append(result, args...)
}

func approveCredentialResource(ctx context.Context, record credential.Record) error {
	return util.ApproveResource(ctx, record.ID, record.Name)
}
