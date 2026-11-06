package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// initConfig and the flag/viper bindings use the global viper singleton, so
// these tests reset it and never run in parallel.

func TestInitConfig_LoadsTOMLFile(t *testing.T) {
	viper.Reset()
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "go-passbolt-cli.toml")
	content := "serverAddress = \"https://file.example.com\"\nmfaMode = \"none\"\ntimeout = \"45s\"\n"
	if err := os.WriteFile(tomlPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfgFile = tomlPath
	defer func() { cfgFile = "" }()

	initConfig()

	if got := viper.GetString("serverAddress"); got != "https://file.example.com" {
		t.Errorf("serverAddress = %q, want from file", got)
	}
	if got := viper.GetString("mfaMode"); got != "none" {
		t.Errorf("mfaMode = %q, want none", got)
	}
	if got := viper.GetDuration("timeout"); got != 45*time.Second {
		t.Errorf("timeout = %v, want 45s", got)
	}
	// workers is absent from the file (0) and must autodetect to the CPU count.
	if got := viper.GetUint("workers"); got != uint(runtime.NumCPU()) {
		t.Errorf("workers = %d, want NumCPU=%d", got, runtime.NumCPU())
	}
}

func TestInitConfig_ReadsEnvVars(t *testing.T) {
	useDefaultConfig(t)
	// No SetEnvPrefix / key replacer is configured, so viper maps the key
	// "serverAddress" to the env var SERVERADDRESS (uppercased, no separators).
	t.Setenv("SERVERADDRESS", "https://env.example.com")

	initConfig()

	if got := viper.GetString("serverAddress"); got != "https://env.example.com" {
		t.Errorf("serverAddress = %q, want from env", got)
	}
}

func TestInitConfig_LoadsPrivateKeyFromFile(t *testing.T) {
	useDefaultConfig(t)

	keyPath := filepath.Join(t.TempDir(), "key.asc")
	if err := os.WriteFile(keyPath, []byte("PRIVATE-KEY-CONTENT"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rootCmd.PersistentFlags().Set("userPrivateKeyFile", keyPath); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootCmd.PersistentFlags().Set("userPrivateKeyFile", "") }()

	initConfig()

	if got := viper.GetString("userPrivateKey"); got != "PRIVATE-KEY-CONTENT" {
		t.Errorf("userPrivateKey = %q, want file content", got)
	}
}

func TestInitConfig_LoadsTLSFilesFromFile(t *testing.T) {
	useDefaultConfig(t)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "client.crt")
	keyPath := filepath.Join(dir, "client.key")
	if err := os.WriteFile(certPath, []byte("CERT-CONTENT"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("KEY-CONTENT"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rootCmd.PersistentFlags().Set("tlsClientCertFile", certPath); err != nil {
		t.Fatal(err)
	}
	if err := rootCmd.PersistentFlags().Set("tlsClientPrivateKeyFile", keyPath); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = rootCmd.PersistentFlags().Set("tlsClientCertFile", "")
		_ = rootCmd.PersistentFlags().Set("tlsClientPrivateKeyFile", "")
	}()

	initConfig()

	if got := viper.GetString("tlsClientCert"); got != "CERT-CONTENT" {
		t.Errorf("tlsClientCert = %q, want file content", got)
	}
	if got := viper.GetString("tlsClientPrivateKey"); got != "KEY-CONTENT" {
		t.Errorf("tlsClientPrivateKey = %q, want file content", got)
	}
}

// Precedence: flag > env > config file, for a representative bound key.
func TestConfigPrecedence_FlagOverridesEnvOverridesFile(t *testing.T) {
	viper.Reset()
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "go-passbolt-cli.toml")
	if err := os.WriteFile(tomlPath, []byte("serverAddress = \"fileval\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfgFile = tomlPath
	defer func() { cfgFile = "" }()
	t.Setenv("SERVERADDRESS", "envval")

	initConfig()

	// env wins over file
	if got := viper.GetString("serverAddress"); got != "envval" {
		t.Errorf("env should override file: got %q, want envval", got)
	}

	// flag wins over env. Bind an isolated flag set so we don't mutate the shared
	// rootCmd flag's Changed state for other tests.
	fs := pflag.NewFlagSet("precedence", pflag.ContinueOnError)
	fs.String("serverAddress", "", "")
	if err := viper.BindPFlag("serverAddress", fs.Lookup("serverAddress")); err != nil {
		t.Fatal(err)
	}
	if err := fs.Set("serverAddress", "flagval"); err != nil {
		t.Fatal(err)
	}
	if got := viper.GetString("serverAddress"); got != "flagval" {
		t.Errorf("flag should override env: got %q, want flagval", got)
	}
}

// Default values are part of the documented config contract; assert them via the
// flag definitions (independent of global viper state).
func TestConfigDefaults(t *testing.T) {
	cases := []struct{ name, want string }{
		{"timeout", "1m0s"},
		{"mfaMode", "interactive-totp"},
		{"mfaRetrys", "3"},
		{"mfaDelay", "10s"},
		{"tlsSkipVerify", "false"},
		{"workers", "0"},
	}
	for _, c := range cases {
		f := rootCmd.PersistentFlags().Lookup(c.name)
		if f == nil {
			t.Errorf("flag %q not defined", c.name)
			continue
		}
		if f.DefValue != c.want {
			t.Errorf("%s default = %q, want %q", c.name, f.DefValue, c.want)
		}
	}
}

func TestMFADeprecatedAliases(t *testing.T) {
	for _, name := range []string{"totpToken", "totpOffset"} {
		f := rootCmd.PersistentFlags().Lookup(name)
		if f == nil {
			t.Fatalf("flag %q not defined", name)
		}
		if f.Deprecated == "" {
			t.Errorf("flag %q should be marked deprecated", name)
		}
	}
}
