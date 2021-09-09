package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure saves the provided global flags to the Config File",
	Long: `Configure saves the provided global flags to the Config File.
this makes using the cli easier as they don't have to be specified all the time.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if viper.ConfigFileUsed() == "" {
			err := viper.SafeWriteConfig()
			if err != nil {
				return fmt.Errorf("Writing Config: %w", err)
			}
		} else {
			err := viper.WriteConfig()
			if err != nil {
				return fmt.Errorf("Writing Config: %w", err)
			}
		}
		if viper.GetBool("debug") {
			fmt.Printf("Saved: %+v\n", viper.AllSettings())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
