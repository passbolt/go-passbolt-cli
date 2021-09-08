package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-passbolt-cli",
	Short: "A CLI tool to interact with Passbolt.",
	Long:  `A CLI tool to interact with Passbolt.`,
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
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config File")

	rootCmd.PersistentFlags().Bool("debug", false, "Enable Debug Logging")
	rootCmd.PersistentFlags().Duration("timeout", time.Minute, "Timeout for the Context")
	rootCmd.PersistentFlags().String("serverAddress", "", "Passbolt Server Address (https://passbolt.example.com)")
	rootCmd.PersistentFlags().String("userPrivateKey", "", "Passbolt User Private Key")
	rootCmd.PersistentFlags().String("userPassword", "", "Passbolt User Password")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("serverAddress", rootCmd.PersistentFlags().Lookup("serverAddress"))
	viper.BindPFlag("userPrivateKey", rootCmd.PersistentFlags().Lookup("userPrivateKey"))
	viper.BindPFlag("userPassword", rootCmd.PersistentFlags().Lookup("userPassword"))
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
		_ = os.MkdirAll(confDir, 0755)

		viper.AddConfigPath(confDir)
		viper.SetConfigType("toml")
		viper.SetConfigName("go-passbolt-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil && viper.GetBool("debug") {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
