package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = makelistCmd()

func makelistCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "Lists Passbolt Entitys",
		Long:    `Lists Passbolt Entitys`,
		Aliases: []string{"index", "ls", "filter", "search"},
	}
}

func configureList() {
	rootCmd.AddCommand(listCmd)
	listCmd.PersistentFlags().BoolP("json", "j", false, "Output JSON")
	listCmd.PersistentFlags().String("filter", "",
		"Define a CEl expression as filter for any list commands. In the expression, all available columns of subcommand can be used (see -c/--column).\n"+
			"See also CEl specifications under https://github.com/google/cel-spec.\n"+
			"Examples:\n"+
			"\t--filter '(name == \"SomeName\" || matches(name, \"RegExpr\")) && uri.startsWith(\"https://auth.\")'\n"+
			"\t--filter 'username == \"User\" && created_timestamp > timestamp(\"2022-06-10T00:00:00.000-00:00\")'\n"+
			"Legacy PascalCase variable names (Name, Username, CreatedTimestamp, ...) remain accepted for backwards compatibility.")
	listCmd.AddCommand(resource.ResourceListCmd)
	listCmd.AddCommand(folder.FolderListCmd)
	listCmd.AddCommand(group.GroupListCmd)
	listCmd.AddCommand(user.UserListCmd)
}

func init() { configureList() }
