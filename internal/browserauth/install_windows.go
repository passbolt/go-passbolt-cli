package browserauth

import (
	"errors"
	"golang.org/x/sys/windows/registry"
)

func registryPaths(target manifestTarget) []string {
	if target.Browser == "firefox" {
		return []string{`Software\Mozilla\NativeMessagingHosts\` + HostName}
	}
	return []string{`Software\Google\Chrome\NativeMessagingHosts\` + HostName, `Software\Chromium\NativeMessagingHosts\` + HostName}
}
func registerHost(target manifestTarget) error {
	for _, path := range registryPaths(target) {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
		if err != nil {
			return err
		}
		err = key.SetStringValue("", target.Path)
		_ = key.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
func unregisterHost(target manifestTarget) error {
	for _, path := range registryPaths(target) {
		if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
	}
	return nil
}
