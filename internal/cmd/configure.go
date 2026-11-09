package cmd

import (
	"errors"
	"fmt"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configureCmd represents the configure command
var configureCmd = makeconfigureCmd()

func makeconfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "Configure saves the provided global flags to the Config File",
		Long: `Configure saves the provided global flags to the Config File.
this makes using the cli easier as they don't have to be specified all the time.

Secrets are stored in cleartext, so the file is kept at mode 0600;
a --config file with looser permissions is reported, not changed.`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := util.WriteViperConfig(cfgPath); err != nil {
				return fmt.Errorf("writing Config: %w", err)
			}
			if viper.GetBool("debug") {
				assumeYes, _ := cmd.Flags().GetBool("yes")
				dumpSettings(assumeYes)
			}
			return nil
		},
	}
}

// dumpSettings prints every saved setting, secrets included, once confirmed.
func dumpSettings(assumeYes bool) {
	if !assumeYes {
		util.Eprintln("Warning: --debug prints your private key, passphrase and MFA seed in cleartext.")
		confirmed, err := util.Confirm("Print them? [y/N]: ")
		if errors.Is(err, util.ErrNotInteractive) {
			util.Eprintln("Skipped: not a terminal. Re-run with --yes to print anyway.")
			return
		}
		if err != nil || !confirmed {
			return
		}
	}
	// stderr keeps secrets out of a --json pipeline and lets 2>/dev/null suppress them.
	util.Eprintf("Saved: %v\n", util.Sanitize(fmt.Sprintf("%+v", viper.AllSettings())))
}

func configureConfigure() {
	rootCmd.AddCommand(configureCmd)
}

func init() { configureConfigure() }
