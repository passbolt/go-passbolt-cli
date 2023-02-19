package cmd

import (
	"github.com/passbolt/go-passbolt-cli/folder"
	"github.com/passbolt/go-passbolt-cli/group"
	"github.com/passbolt/go-passbolt-cli/resource"
	"github.com/spf13/cobra"
)

// toclipboardCmd represent the command to copy column entries to the clipboard
var toclipboardCmd = &cobra.Command{
	Use:     "to-clipboard",
	Short:   "Copy entity column entries to clipboard",
	Long:    "All entries of columns defined with -c / --column are copied to the clipoard one by one.",
	Aliases: []string{"clipboard", "copy-clipboard", "copy-clip"},
}

func init() {
	rootCmd.AddCommand(toclipboardCmd)
	toclipboardCmd.PersistentFlags().String("filter", "",
		"Define a CEl expression as filter for any list commands. In the expression, all available columns of subcommand can be used (see -c/--column).\n"+
			"See also CEl specifications under https://github.com/google/cel-spec.\n"+
			"Examples:\n"+
			"\t--filter '(Name == \"SomeName\" || matches(Name, \"RegExpr\")) && URI.startsWith(\"https://auth.\")'\n"+
			"\t--filter 'Username == \"User\" && CreatedTimestamp > timestamp(\"2022-06-10T00:00:00.000-00:00\")'")
	toclipboardCmd.PersistentFlags().IntP("delay", "d", 5, "Seconds of delay to iterating over the column entries.")
	toclipboardCmd.AddCommand(folder.FolderToClipboardCommand)
	toclipboardCmd.AddCommand(group.GroupToClipboardCommand)
	toclipboardCmd.AddCommand(resource.ResourceToClipboardCommand)
}
