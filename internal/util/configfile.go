package util

import (
	"bytes"
	"path/filepath"
	"strings"

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

	file, err := CreateSecretFile(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(settings.Bytes()); err != nil {
		return err
	}
	return file.Sync()
}
