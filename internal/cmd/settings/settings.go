// Package settings implements plugin-specific organization settings commands.
package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt/api"
	"github.com/spf13/cobra"
)

type clientRunner func(*cobra.Command, func(context.Context, *api.Client) error) error

// NewGetCommand reads organization settings using the configured connection.
func NewGetCommand() *cobra.Command { return newSettingsCommand("get", util.WithClient) }

// NewUpdateCommand submits a plugin-specific JSON settings object.
func NewUpdateCommand() *cobra.Command { return newSettingsCommand("update", util.WithClient) }

// NewDeleteCommand deletes or disables settings through supported plugin routes.
func NewDeleteCommand() *cobra.Command { return newSettingsCommand("delete", util.WithClient) }

func newSettingsCommand(operation string, runClient clientRunner) *cobra.Command {
	names := pluginNames(operation, "")
	cmd := &cobra.Command{
		Use:   "organization-settings",
		Short: map[string]string{"get": "Gets organization settings as JSON", "update": "Updates organization settings from a JSON object", "delete": "Deletes or disables organization settings"}[operation],
		Long:  "Manage plugin-specific organization settings. Available operations depend on the server edition, enabled plugins and your permissions.\nSupported plugins for this command: " + strings.Join(names, ", "),
		Args:  cobra.NoArgs,
	}
	cmd.Flags().String("plugin", "", "Organization settings plugin")
	cmd.MarkFlagRequired("plugin")
	if operation == "update" {
		cmd.Flags().String("input-file", "", "Plugin-specific JSON object file, or - to read stdin")
		cmd.MarkFlagRequired("input-file")
	}
	if operation == "delete" {
		cmd.Flags().String("id", "", "Settings record UUID, required only for plugins with record-specific deletion")
	} else {
		cmd.Flags().BoolP("json", "j", false, "Print the JSON response (get always prints JSON)")
	}
	_ = cmd.RegisterFlagCompletionFunc("plugin", func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		return pluginNames(operation, prefix), cobra.ShellCompDirectiveNoFileComp
	})
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		request, err := parseSettingsRequest(cmd, operation)
		if err != nil {
			return err
		}
		return runClient(cmd, func(ctx context.Context, client *api.Client) error {
			return runSettingsOperation(ctx, client, operation, request, cmd.OutOrStdout())
		})
	}
	return cmd
}

func supportsOperation(p api.OrganizationSettingsPlugin, operation string) bool {
	switch operation {
	case "get":
		return p.Readable
	case "update":
		return p.Writable
	case "delete":
		return p.Deletable
	}
	return false
}

func pluginNames(operation, prefix string) []string {
	var names []string
	for _, p := range api.OrganizationSettingsPlugins() {
		if supportsOperation(p, operation) && strings.HasPrefix(p.Name, prefix) {
			names = append(names, p.Name)
		}
	}
	return names
}

type settingsRequest struct {
	plugin, id string
	input      json.RawMessage
	jsonOutput bool
}

var settingsIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func parseSettingsRequest(cmd *cobra.Command, operation string) (settingsRequest, error) {
	var request settingsRequest
	plugin, err := cmd.Flags().GetString("plugin")
	if err != nil {
		return request, err
	}
	var capability *api.OrganizationSettingsPlugin
	for _, p := range api.OrganizationSettingsPlugins() {
		if p.Name == plugin {
			capability = &p
			break
		}
	}
	if capability == nil {
		return request, fmt.Errorf("unknown organization settings plugin %q", plugin)
	}
	if !supportsOperation(*capability, operation) {
		return request, fmt.Errorf("organization settings plugin %q does not support %s", plugin, operation)
	}
	request.plugin = plugin
	if operation == "delete" {
		request.id, err = cmd.Flags().GetString("id")
		if err != nil {
			return request, err
		}
		if capability.DeleteRequiresID {
			if !settingsIDPattern.MatchString(request.id) {
				return request, fmt.Errorf("--id must be a settings record UUID for plugin %q", plugin)
			}
		} else if request.id != "" {
			return request, fmt.Errorf("plugin %q does not accept --id", plugin)
		}
		return request, nil
	}
	request.jsonOutput, err = cmd.Flags().GetBool("json")
	if err != nil {
		return request, err
	}
	if operation == "update" {
		file, err := cmd.Flags().GetString("input-file")
		if err != nil {
			return request, err
		}
		if file == "" {
			return request, fmt.Errorf("--input-file is required")
		}
		var input []byte
		if file == "-" {
			input, err = io.ReadAll(cmd.InOrStdin())
		} else {
			input, err = os.ReadFile(file)
		}
		if err != nil {
			return request, fmt.Errorf("reading settings input: %w", err)
		}
		input = bytes.TrimSpace(input)
		if !json.Valid(input) || len(input) == 0 || input[0] != '{' {
			return request, fmt.Errorf("settings input must be a JSON object")
		}
		request.input = input
	}
	return request, nil
}

func runSettingsOperation(ctx context.Context, client *api.Client, operation string, request settingsRequest, out io.Writer) error {
	var response json.RawMessage
	var err error
	switch operation {
	case "get":
		response, err = client.GetOrganizationSettings(ctx, request.plugin)
	case "update":
		response, err = client.SetOrganizationSettings(ctx, request.plugin, request.input)
	case "delete":
		err = client.DeleteOrganizationSettings(ctx, request.plugin, request.id)
	}
	if err != nil {
		return fmt.Errorf("%s organization settings for %s: %w", operation, request.plugin, err)
	}
	if operation == "get" || request.jsonOutput {
		return util.FprintJSON(out, response)
	}
	return nil
}
