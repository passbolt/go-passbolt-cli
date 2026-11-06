package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/passbolt/go-passbolt-cli/internal/util/securefile"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string

// cfgPath is where configure and verify write, resolved by initConfig.
var cfgPath string

// defaultConfigFile is what configure creates when there is no config yet.
const defaultConfigFile = "go-passbolt-cli.toml"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "passbolt",
	Short:        "A CLI tool to interact with Passbolt.",
	Long:         `A CLI tool to interact with Passbolt.`,
	SilenceUsage: true,
	// Errors are printed by Execute instead of cobra so that server-supplied
	// text can be sanitized first: api.APIError embeds the server's Message
	// verbatim and every command wraps it with %w.
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	// Last, so it covers the error line too. Not deferred: os.Exit skips defers.
	if err != nil {
		util.Eprintln("Error:", util.Sanitize(err.Error()))
		util.Exit(1)
	}
	util.FlushOutputs()
	util.WarnReplacements()
}

func init() {
	pterm.DisableStyling()
	// Backstop for any pterm printer used without an explicit writer; util.RenderTable
	// sets one itself. Progressbars write to os.Stderr and are unaffected.
	pterm.SetDefaultOutput(util.Out)

	// Help, usage and flag-parse errors are cobra's own writes to the terminal.
	rootCmd.SetOut(util.Out)
	rootCmd.SetErr(util.Err)

	cobra.OnInitialize(initConfig)

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config File")

	rootCmd.PersistentFlags().Bool("debug", false, "Enable Debug Logging")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "Assume yes for confirmation prompts, for use in scripts")
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

	rootCmd.PersistentFlags().Uint("workers", 0, "Number of Concurrent Workers for Expensive Operations. 0 (default) uses the number of CPU cores")

	// yes is unbound: configure writes AllSettings, which would persist "yes = true".
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

	viper.BindPFlag("workers", rootCmd.PersistentFlags().Lookup("workers"))
}

func fileToContent(file, contentFlag string) {
	if viper.GetBool("debug") {
		util.Eprintln("Loading file:", file)
	}
	content, err := os.ReadFile(file)
	if err != nil {
		util.Eprintln("Error Loading File: ", util.Sanitize(err.Error()))
		util.Exit(1)
	}
	viper.Set(contentFlag, string(content))
}

// hardenConfigFile repairs the default config's mode before the read, or warns for --config.
func hardenConfigFile() {
	if cfgFile != "" {
		util.WarnIfExposed(cfgPath, util.ConfigExposure)
		return
	}
	// os.MkdirAll leaves an existing directory alone and is masked by the umask.
	if err := securefile.RepairMode(filepath.Dir(cfgPath), 0o700); err != nil {
		util.Eprintln("Warning: failed to set config directory permissions:", util.Sanitize(err.Error()))
	}
	if err := securefile.RepairMode(cfgPath, 0o600); err != nil {
		util.Eprintln("Warning: failed to set config file permissions:", util.Sanitize(err.Error()))
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Where configure writes when the search below finds nothing.
	newConfigPath := ""

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find config directory.
		confDir, err := os.UserConfigDir()
		if err != nil {
			// cobra.CheckErr would print this raw to os.Stderr.
			util.Eprintln("Error:", util.Sanitize(err.Error()))
			util.Exit(1)
		}

		confDir = filepath.Join(confDir, "go-passbolt-cli")
		_ = os.MkdirAll(confDir, 0o700)

		newConfigPath = filepath.Join(confDir, defaultConfigFile)
		viper.AddConfigPath(confDir)
		viper.SetConfigType("toml")
		viper.SetConfigName("go-passbolt-cli")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("debug") {
			util.Eprintln("Using config file:", viper.ConfigFileUsed())
		}
	}

	// Read first: viper records the file it picked even when parsing it failed, which is
	// exactly when the mode still needs repairing.
	cfgPath = viper.ConfigFileUsed()
	if cfgPath == "" {
		cfgPath = newConfigPath
	}

	hardenConfigFile()

	// Read in Private Key from File if userprivatekeyfile is set
	userprivatekeyfile, err := rootCmd.PersistentFlags().GetString("userPrivateKeyFile")
	if err == nil && userprivatekeyfile != "" {
		fileToContent(userprivatekeyfile, "userPrivateKey")
	} else if err != nil && viper.GetBool("debug") {
		util.Eprintln("Getting Private Key File Flag:", err)
	}

	// Read in Client Certificate Private Key from File if tlsClientPrivateKeyFile is set
	tlsclientprivatekeyfile, err := rootCmd.PersistentFlags().GetString("tlsClientPrivateKeyFile")
	if err == nil && tlsclientprivatekeyfile != "" {
		fileToContent(tlsclientprivatekeyfile, "tlsClientPrivateKey")
	} else if err != nil && viper.GetBool("debug") {
		util.Eprintln("Getting Client Certificate Private key File Flag:", err)
	}

	// Read in Client Certificate from File if tlsClientCertFile is set
	tlsclientcertfile, err := rootCmd.PersistentFlags().GetString("tlsClientCertFile")
	if err == nil && tlsclientcertfile != "" {
		fileToContent(tlsclientcertfile, "tlsClientCert")
	} else if err != nil && viper.GetBool("debug") {
		util.Eprintln("Getting Client Certificate File Flag:", err)
	}

	// If Workers is set to 0, autodetect Number of Cores
	workers := viper.GetUint("workers")
	if workers == 0 {
		viper.Set("workers", runtime.NumCPU())
		if viper.GetBool("debug") {
			util.Eprintln("Autodetected Worker Count:", viper.GetUint("workers"))
		}
	}
}

func SetVersionInfo(version, commit, date string, dirty bool) {
	v := fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit)
	if dirty {
		v += " dirty"
	}
	rootCmd.Version = v
}
