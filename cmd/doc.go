package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// configureCmd represents the configure command
var genDocCmd = &cobra.Command{
	Use:    "gendoc",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		docType, err := cmd.Flags().GetString("type")
		if err != nil {
			return err
		}
		rootCmd.DisableAutoGenTag = true
		switch docType {
		case "markdown":
			return doc.GenMarkdownTree(rootCmd, "doc")
		case "man":
			return doc.GenManTree(rootCmd, nil, "man")
		default:
			return fmt.Errorf("Unknown type: %v", docType)
		}
	},
}

func init() {
	rootCmd.AddCommand(genDocCmd)
	genDocCmd.Flags().StringP("type", "t", "markdown", "what to generate, markdown or man")
}
