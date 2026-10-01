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
	return file.Sync()
}

// WarnIfGroupOrWorldAccessible reports a config file others can read, without touching it.
func WarnIfGroupOrWorldAccessible(path string) {
	detail, fix, exposed := securefile.Exposure(path)
	if !exposed {
		return
	}
	Eprintf("Warning: %s is accessible to other users (%s).\n", Sanitize(path), Sanitize(detail))
	Eprintf("If it is a passbolt config it holds your private key and passphrase in cleartext; run: %s\n", Sanitize(fix))
}
