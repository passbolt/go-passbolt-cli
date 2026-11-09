package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/credential"
	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCloudCredentialCommand(protocol string) *cobra.Command {
	cmd := &cobra.Command{Use: protocol + " get --id UUID", Short: "Read " + protocol + " process credentials from one Passbolt resource", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 || args[0] != "get" {
			return credential.ErrRequest
		}
		id, err := cmd.Flags().GetString("id")
		if err != nil || !credential.ValidResourceID(id) {
			return credential.ErrRequest
		}
		return nil
	}, RunE: func(cmd *cobra.Command, _ []string) error { return cloudCredential(cmd, protocol) }}
	cmd.Flags().String("id", "", "Immutable Passbolt resource UUID")
	cmd.MarkFlagRequired("id")
	cmd.Flags().String("expiration-field", "expiration", "Optional RFC3339 expiration field in the resource; empty disables it")
	if protocol == "aws" {
		cmd.Flags().String("session-token-field", "session_token", "Optional session token field in the resource; empty disables it")
	}
	return cmd
}
func invocationValue(key string) string {
	if inAgent {
		for _, entry := range invocationEnvironment {
			if name, value, ok := strings.Cut(entry, "="); ok && name == key {
				return value
			}
		}
		return ""
	}
	return os.Getenv(key)
}
func cloudCredential(cmd *cobra.Command, protocol string) error {
	if viper.GetBool("debug") {
		return credential.ErrLookup
	}
	interactive := true
	if protocol == "k8s" {
		allowed, err := credential.KubernetesInteractive(invocationValue("KUBERNETES_EXEC_INFO"))
		if err != nil {
			return fmt.Errorf("invalid Kubernetes exec request")
		}
		interactive = allowed
		// Agent use always requires explicit UI consent, which an exec host may
		// prohibit through the official request's spec.interactive flag.
		if !interactive && viper.GetBool("agent") {
			return fmt.Errorf("kubernetes exec host disallows interactive agent approval")
		}
	}
	if !inAgent && viper.GetBool("agent") {
		if err := callAgent(cmd, "run"); err != nil {
			return credential.ErrLookup
		}
		return nil
	}
	previousPrompt := util.PasswordPrompt
	if !interactive {
		util.PasswordPrompt = func(string) (string, error) { return "", util.ErrNotInteractive }
		defer func() { util.PasswordPrompt = previousPrompt }()
	}
	id, _ := cmd.Flags().GetString("id")
	expiry, _ := cmd.Flags().GetString("expiration-field")
	// The machine client routes progress to stderr. Keep failure messages generic
	// so SDK response bodies cannot expose returned cloud credentials.
	err := withCredentialAPI(cmd, func(ctx context.Context, client *api.Client) error {
		data, err := (credential.ClientStore{Client: client, Approve: approveCredentialResource}).Load(ctx, strings.ToLower(id))
		if err != nil {
			return credential.ErrLookup
		}
		if cmd.Flags().Changed("expiration-field") && expiry != "" {
			value, err := data.Field(expiry)
			if err != nil || value == nil {
				return credential.ErrResponse
			}
		}
		if protocol == "aws" {
			field, _ := cmd.Flags().GetString("session-token-field")
			if cmd.Flags().Changed("session-token-field") && field != "" {
				value, err := data.Field(field)
				if err != nil || value == nil {
					return credential.ErrResponse
				}
			}
			return credential.WriteAWS(util.DataOut, data, field, expiry)
		}
		return credential.WriteKubernetes(util.DataOut, data, expiry)
	})
	if err != nil {
		return fmt.Errorf("%s credential output failed: %w", protocol, credential.ErrLookup)
	}
	return nil
}
