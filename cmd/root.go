package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "passbolt",
	Short:        "A CLI tool to interact with Passbolt.",
	Long:         `A CLI tool to interact with Passbolt.`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	pterm.DisableStyling()

	cobra.OnInitialize(initConfig)

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config File")

	rootCmd.PersistentFlags().Bool("debug", false, "Enable Debug Logging")
	rootCmd.PersistentFlags().Duration("timeout", time.Minute, "Timeout for the Context")
	rootCmd.PersistentFlags().String("serverAddress", "", "Passbolt Server Address (https://passbolt.example.com)")
	rootCmd.PersistentFlags().String("userPrivateKey", "", "Passbolt User Private Key")
	rootCmd.PersistentFlags().String("userPrivateKeyFile", "", "Passbolt User Private Key File, if set then the userPrivateKey will be Overwritten with the File Content")
	rootCmd.PersistentFlags().String("userPassword", "", "Passbolt User Password")
	rootCmd.PersistentFlags().String("mfaMode", "interactive-totp", "How to Handle MFA, the following Modes exist: none, interactive-totp and noninteractive-totp")
	rootCmd.PersistentFlags().String("totpToken", "", "Token to generate TOTP's, only used in nointeractive-totp mode")
	rootCmd.PersistentFlags().Duration("totpOffset", time.Duration(0), "TOTP Generation offset only used in noninteractive-totp mode")
	rootCmd.PersistentFlags().Uint("mfaRetrys", 3, "How often to retry TOTP Auth, only used in nointeractive modes")
	rootCmd.PersistentFlags().Duration("mfaDelay", time.Second*10, "Delay between MFA Attempts, only used in noninteractive modes")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("serverAddress", rootCmd.PersistentFlags().Lookup("serverAddress"))
	viper.BindPFlag("userPrivateKey", rootCmd.PersistentFlags().Lookup("userPrivateKey"))
	viper.BindPFlag("userPassword", rootCmd.PersistentFlags().Lookup("userPassword"))
	viper.BindPFlag("mfaMode", rootCmd.PersistentFlags().Lookup("mfaMode"))
	viper.BindPFlag("totpToken", rootCmd.PersistentFlags().Lookup("totpToken"))
	viper.BindPFlag("totpOffset", rootCmd.PersistentFlags().Lookup("totpOffset"))
	viper.BindPFlag("mfaRetrys", rootCmd.PersistentFlags().Lookup("mfaRetrys"))
	viper.BindPFlag("mfaDelay", rootCmd.PersistentFlags().Lookup("mfaDelay"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find config directory.
		confDir, err := os.UserConfigDir()
		cobra.CheckErr(err)

		confDir = filepath.Join(confDir, "go-passbolt-cli")
		_ = os.MkdirAll(confDir, 0700)

		viper.SetConfigPermissions(os.FileMode(0600))
		viper.AddConfigPath(confDir)
		viper.SetConfigType("toml")
		viper.SetConfigName("go-passbolt-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("debug") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
		// update Config file Permissions
		os.Chmod(viper.ConfigFileUsed(), 0600)
	}

	// Read in Private Key from File if userprivatekeyfile is set
	userprivatekeyfile, err := rootCmd.PersistentFlags().GetString("userPrivateKeyFile")
	if err == nil && userprivatekeyfile != "" {
		if viper.GetBool("debug") {
			fmt.Fprintln(os.Stderr, "Loading Private Key from File:", userprivatekeyfile)
		}
		content, err := ioutil.ReadFile(userprivatekeyfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error Loading Private Key from File: ", err)
			os.Exit(1)
		}
		viper.Set("userprivatekey", string(content))
	} else if err != nil && viper.GetBool("debug") {
		fmt.Fprintln(os.Stderr, "Getting Private Key File Flag:", err)
	}
}
