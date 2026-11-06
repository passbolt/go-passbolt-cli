package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/passbolt/go-passbolt-cli/internal/util"
	"github.com/spf13/viper"
)

// The config file holds credentials in cleartext, so its mode is part of its contract.
// Like the rest of this package these tests share global viper and never run in parallel.

func skipWithoutPOSIXModes(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not POSIX on windows")
	}
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

// captureStderr redirects util.Err for the test and returns the buffer behind it.
func captureStderr(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := util.Err
	util.Err = util.NewSanitizingWriter(&buf)
	t.Cleanup(func() { util.Err = orig })
	return &buf
}

// runConfigure runs configure the way cobra would, after initConfig resolved the path.
func runConfigure(t *testing.T) error {
	t.Helper()
	return configureCmd.RunE(configureCmd, nil)
}

// useCustomConfig points the next initConfig at path, as --config does.
func useCustomConfig(t *testing.T, path string) {
	t.Helper()
	viper.Reset()
	cfgFile = path
	t.Cleanup(func() { cfgFile = "" })
}

// useDefaultConfig sends the next initConfig to a throwaway user config directory and
// returns it. Both env vars: os.UserConfigDir reads XDG_CONFIG_HOME on unix, HOME on darwin.
func useDefaultConfig(t *testing.T) string {
	t.Helper()
	viper.Reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	cfgFile = ""

	confDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	return filepath.Join(confDir, "go-passbolt-cli")
}

// TestConfigure_CustomPathIsOwnerOnly is the reported bug: --config wrote 0644.
func TestConfigure_CustomPathIsOwnerOnly(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "pb.toml")
	useCustomConfig(t, path)

	initConfig()
	if err := runConfigure(t); err != nil {
		t.Fatalf("configure: %v", err)
	}

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

func TestConfigure_DefaultPathIsOwnerOnly(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := useDefaultConfig(t)

	initConfig()
	if err := runConfigure(t); err != nil {
		t.Fatalf("configure: %v", err)
	}

	if got := modeOf(t, filepath.Join(dir, "go-passbolt-cli.toml")); got != 0600 {
		t.Errorf("file mode = %04o, want 0600", got)
	}
	if got := modeOf(t, dir); got != 0700 {
		t.Errorf("dir mode = %04o, want 0700", got)
	}
}

// TestInitConfig_RepairsUnparsableDefaultFile: the old chmod sat behind a successful parse.
func TestInitConfig_RepairsUnparsableDefaultFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := useDefaultConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "go-passbolt-cli.toml")
	if err := os.WriteFile(path, []byte("not = = valid toml\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	// No configure: the repair must happen at startup, so any command fixes it.
	initConfig()

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

// TestInitConfig_RepairsLooseDefaultDirectory guards the directory os.MkdirAll leaves alone.
func TestInitConfig_RepairsLooseDefaultDirectory(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := useDefaultConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}

	initConfig()

	if got := modeOf(t, dir); got != 0700 {
		t.Errorf("dir mode = %04o, want 0700", got)
	}
}

// TestConfigure_TightensPlantedFile: the file exists first, so the umask cannot help.
func TestConfigure_TightensPlantedFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "pb.toml")
	if err := os.WriteFile(path, []byte("not = = valid toml\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0666); err != nil {
		t.Fatal(err)
	}
	useCustomConfig(t, path)

	initConfig()
	if err := runConfigure(t); err != nil {
		t.Fatalf("configure: %v", err)
	}

	if got := modeOf(t, path); got != 0600 {
		t.Errorf("mode = %04o, want 0600", got)
	}
}

// TestConfigure_WritesThroughOwnedSymlink: a dotfiles config is a link to our own file.
func TestConfigure_WritesThroughOwnedSymlink(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles.toml")
	if err := os.WriteFile(target, []byte("serverAddress = \"old\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "pb.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	useCustomConfig(t, link)

	captureStderr(t) // --config only warns; the mode is not this test's subject
	initConfig()
	viper.Set("serverAddress", "https://new.example.com")

	if err := runConfigure(t); err != nil {
		t.Fatalf("configure: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "new.example.com") {
		t.Errorf("target content = %q, want the new setting written through the link", got)
	}
	if got := modeOf(t, target); got != 0600 {
		t.Errorf("target mode = %04o, want 0600", got)
	}
}

// TestInitConfig_SymlinkedDefaultConfigIsQuiet: a dotfiles link used to warn every run.
func TestInitConfig_SymlinkedDefaultConfigIsQuiet(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := useDefaultConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "dotfiles.toml")
	if err := os.WriteFile(target, []byte("serverAddress = \"https://dotfiles.example.com\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "go-passbolt-cli.toml")); err != nil {
		t.Fatal(err)
	}

	warnings := captureStderr(t)
	initConfig()
	util.FlushOutputs()

	if warnings.Len() != 0 {
		t.Errorf("warnings = %q, want none for a symlink to our own file", warnings)
	}
	if got := modeOf(t, target); got != 0600 {
		t.Errorf("target mode = %04o, want it repaired to 0600", got)
	}
	if got := viper.GetString("serverAddress"); got != "https://dotfiles.example.com" {
		t.Errorf("serverAddress = %q, want the linked config to have been read", got)
	}
}

// TestConfigure_LeavesUnsupportedFileIntact: the write used to truncate before viper
// decided it could not produce that format, so "--config notes.txt" emptied notes.txt.
func TestConfigure_LeavesUnsupportedFileIntact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	const notes = "shopping list\nmilk\n"
	if err := os.WriteFile(path, []byte(notes), 0600); err != nil {
		t.Fatal(err)
	}
	useCustomConfig(t, path)

	initConfig()
	if err := runConfigure(t); err == nil {
		t.Fatal("configure into a .txt file succeeded, want an unsupported-format error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != notes {
		t.Errorf("content = %q, want it untouched at %q", got, notes)
	}
}

// TestInitConfig_HardensTheConfigViperActuallyUses: viper searches json before toml, so
// the file to repair and to write back is not always the .toml we would create.
func TestInitConfig_HardensTheConfigViperActuallyUses(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := useDefaultConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(dir, "go-passbolt-cli.json")
	if err := os.WriteFile(jsonPath, []byte(`{"serverAddress":"https://json.example.com"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(jsonPath, 0644); err != nil {
		t.Fatal(err)
	}

	initConfig()

	if got := modeOf(t, jsonPath); got != 0600 {
		t.Errorf("mode = %04o, want the json config repaired to 0600", got)
	}

	viper.Set("serverAddress", "https://written.example.com")
	if err := runConfigure(t); err != nil {
		t.Fatalf("configure: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "go-passbolt-cli.toml")); err == nil {
		t.Error("configure wrote a .toml beside the .json that viper picks up first")
	}
	written, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "written.example.com") {
		t.Errorf("json config = %q, want configure to have rewritten it", written)
	}
}

// TestInitConfig_DoesNotChmodCustomPath: even "help" used to chmod whatever --config named.
func TestInitConfig_DoesNotChmodCustomPath(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "shared-data.toml")
	if err := os.WriteFile(path, []byte("key = \"value\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	useCustomConfig(t, path)

	warnings := captureStderr(t)

	initConfig()
	util.FlushOutputs()

	if got := modeOf(t, path); got != 0644 {
		t.Errorf("mode = %04o, want it left at 0644", got)
	}
	if !strings.Contains(warnings.String(), "chmod 600") {
		t.Errorf("warning = %q, want it to tell the user how to fix the file", warnings.String())
	}
}

// TestConfigure_DoesNotPersistYesFlag: bound to viper it would persist and disable the prompt.
func TestConfigure_DoesNotPersistYesFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pb.toml")
	useCustomConfig(t, path)

	if err := rootCmd.PersistentFlags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = rootCmd.PersistentFlags().Set("yes", "false")
		rootCmd.PersistentFlags().Lookup("yes").Changed = false
	})

	initConfig()
	if err := runConfigure(t); err != nil {
		t.Fatalf("configure: %v", err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), "yes") {
		t.Errorf("config file records the --yes flag, want it absent:\n%s", written)
	}
}
