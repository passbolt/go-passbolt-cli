package util

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/passbolt/go-passbolt-cli/internal/util/securefile"
	"github.com/spf13/viper"
)

// WriteViperConfig writes viper's settings to path with mode 0600.
func WriteViperConfig(path string) error {
	// WriteConfigTo prefers an explicit type over the extension, unlike writeConfig.
	if ext := filepath.Ext(path); ext != "" && ext != filepath.Base(path) {
		viper.SetConfigType(strings.TrimPrefix(ext, "."))
	}

	// Marshal first: a format WriteConfigTo cannot write leaves the existing file alone.
	var settings bytes.Buffer
	if err := viper.WriteConfigTo(&settings); err != nil {
		return err
	}

	file, err := securefile.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(settings.Bytes()); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	WarnIfExposed(path, ConfigExposure)
	return nil
}

// ConfigExposure is what WarnIfExposed says a config file holds.
const ConfigExposure = "If it is a passbolt config it holds your private key and passphrase in cleartext"

// warnedExposed holds the paths WarnIfExposed has already reported, so a command that reads
// and then writes a file (configure, verify) warns about it once.
var warnedExposed = map[string]bool{}

// WarnIfExposed prints a warning to stderr if other users can read path or it is not yours.
// holds describes what the file contains. Each path is warned about once.
func WarnIfExposed(path, holds string) {
	if warnedExposed[path] {
		return
	}
	detail, fix, exposed := securefile.Exposure(path)
	if !exposed {
		return
	}
	warnedExposed[path] = true
	if fix == "" { // someone else owns it: moving it is the only fix
		Eprintf("Warning: %s is %s, not you, so others may read or replace it.\n", Sanitize(path), Sanitize(detail))
		Eprintf("%s; keep it somewhere you own.\n", holds)
		return
	}
	Eprintf("Warning: %s is accessible to other users (%s).\n", Sanitize(path), Sanitize(detail))
	Eprintf("%s; run: %s\n", holds, Sanitize(fix))
}
