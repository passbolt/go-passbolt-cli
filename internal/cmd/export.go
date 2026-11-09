package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/keepass"
	"github.com/spf13/cobra"
)

// exportCmd represents the export command
var exportCmd = makeexportCmd()

func makeexportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Exports Passbolt Data",
		Long:  `Exports Passbolt Data`,
	}
}

func configureExport() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(keepass.KeepassExportCmd)
}

func init() { configureExport() }
