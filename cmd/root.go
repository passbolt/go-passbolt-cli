package cmd

import (
	"fmt"
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
	rootCmd.PersistentFlags().MarkDeprecated("totpToken", "use --mfaTotpToken instead")
	rootCmd.PersistentFlags().String("mfaTotpToken", "", "Token to generate TOTP's, only used in nointeractive-totp mode")

	rootCmd.PersistentFlags().Duration("totpOffset", time.Duration(0), "TOTP Generation offset only used in noninteractive-totp mode")
	rootCmd.PersistentFlags().MarkDeprecated("totpOffset", "use --mfaTotpOffset instead")
	rootCmd.PersistentFlags().Duration("mfaTotpOffset", time.Duration(0), "TOTP Generation offset only used in noninteractive-totp mode")

	rootCmd.PersistentFlags().Uint("mfaRetrys", 3, "How often to retry TOTP Auth, only used in nointeractive modes")
	rootCmd.PersistentFlags().Duration("mfaDelay", time.Second*10, "Delay between MFA Attempts, only used in noninteractive modes")

	rootCmd.PersistentFlags().Bool("tlsSkipVerify", false, "Allow servers with self-signed certificates")
	rootCmd.PersistentFlags().String("tlsClientPrivateKeyFile", "", "Client private key path for mtls")
	rootCmd.PersistentFlags().String("tlsClientCertFile", "", "Client certificate path for mtls")
	rootCmd.PersistentFlags().String("tlsClientPrivateKey", "", "Client private key for mtls")
	rootCmd.PersistentFlags().String("tlsClientCert", "", "Client certificate for mtls")

	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("serverAddress", rootCmd.PersistentFlags().Lookup("serverAddress"))
	viper.BindPFlag("userPrivateKey", rootCmd.PersistentFlags().Lookup("userPrivateKey"))
	viper.BindPFlag("userPassword", rootCmd.PersistentFlags().Lookup("userPassword"))
	viper.BindPFlag("mfaMode", rootCmd.PersistentFlags().Lookup("mfaMode"))
	viper.BindPFlag("totpToken", rootCmd.PersistentFlags().Lookup("totpToken"))
	viper.BindPFlag("mfaTotpToken", rootCmd.PersistentFlags().Lookup("mfaTotpToken"))
	viper.BindPFlag("totpOffset", rootCmd.PersistentFlags().Lookup("totpOffset"))
	viper.BindPFlag("mfaTotpOffset", rootCmd.PersistentFlags().Lookup("mfaTotpOffset"))
	viper.BindPFlag("mfaRetrys", rootCmd.PersistentFlags().Lookup("mfaRetrys"))
	viper.BindPFlag("mfaDelay", rootCmd.PersistentFlags().Lookup("mfaDelay"))

	viper.BindPFlag("tlsSkipVerify", rootCmd.PersistentFlags().Lookup("tlsSkipVerify"))
	viper.BindPFlag("tlsClientCert", rootCmd.PersistentFlags().Lookup("tlsClientCert"))
	viper.BindPFlag("tlsClientPrivateKey", rootCmd.PersistentFlags().Lookup("tlsClientPrivateKey"))
}

func fileToContent(file, contentFlag string) {
	if viper.GetBool("debug") {
		fmt.Fprintln(os.Stderr, "Loading file:", file)
	}
	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error Loading File: ", err)
		os.Exit(1)
	}
	viper.Set(contentFlag, string(content))
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
		fileToContent(userprivatekeyfile, "userPrivateKey")
	} else if err != nil && viper.GetBool("debug") {
		fmt.Fprintln(os.Stderr, "Getting Private Key File Flag:", err)
	}

	// Read in Client Certificate Private Key from File if tlsClientPrivateKeyFile is set
	tlsclientprivatekeyfile, err := rootCmd.PersistentFlags().GetString("tlsClientPrivateKeyFile")
	if err == nil && tlsclientprivatekeyfile != "" {
		fileToContent(tlsclientprivatekeyfile, "tlsClientPrivateKey")
	} else if err != nil && viper.GetBool("debug") {
		fmt.Fprintln(os.Stderr, "Getting Client Certificate Private key File Flag:", err)
	}

	// Read in Client Certificate from File if tlsClientCertFile is set
	tlsclientcertfile, err := rootCmd.PersistentFlags().GetString("tlsClientCertFile")
	if err == nil && tlsclientcertfile != "" {
		fileToContent(tlsclientcertfile, "tlsClientCert")
	} else if err != nil && viper.GetBool("debug") {
		fmt.Fprintln(os.Stderr, "Getting Client Certificate File Flag:", err)
	}
}

func SetVersionInfo(version, commit, date string, dirty bool) {
	v := fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit)
	if dirty {
		v = v + " dirty"
	}
	rootCmd.Version = v
}
