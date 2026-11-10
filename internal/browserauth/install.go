package browserauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const ChromiumExtensionID = "didegimhafipceonhjepacocaffmoppf"
const FirefoxExtensionID = "passbolt@passbolt.com"

type manifestTarget struct {
	Path    string
	Data    []byte
	Browser string
}

func manifestTargets(platform, home, config, executable string, extraChromiumIDs ...string) ([]manifestTarget, error) {
	type target struct{ browser, path string }
	var targets []target
	switch platform {
	case "linux":
		targets = []target{{"chromium", filepath.Join(config, "google-chrome/NativeMessagingHosts")}, {"chromium", filepath.Join(config, "chromium/NativeMessagingHosts")}, {"firefox", filepath.Join(home, ".mozilla/native-messaging-hosts")}}
	case "darwin":
		targets = []target{{"chromium", filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts")}, {"chromium", filepath.Join(home, "Library/Application Support/Chromium/NativeMessagingHosts")}, {"firefox", filepath.Join(home, "Library/Application Support/Mozilla/NativeMessagingHosts")}}
	case "windows":
		targets = []target{{"chromium", filepath.Join(config, "Passbolt/NativeMessagingHosts/chromium")}, {"firefox", filepath.Join(config, "Passbolt/NativeMessagingHosts/firefox")}}
	default:
		return nil, fmt.Errorf("unsupported native messaging platform")
	}
	result := make([]manifestTarget, 0, len(targets))
	for _, target := range targets {
		manifest := map[string]any{"name": HostName, "description": "Passbolt CLI browser authorization", "path": executable, "type": "stdio"}
		if target.browser == "firefox" {
			manifest["allowed_extensions"] = []string{FirefoxExtensionID}
		} else {
			origins := []string{"chrome-extension://" + ChromiumExtensionID + "/"}
			for _, id := range extraChromiumIDs {
				if id != "" {
					origins = append(origins, "chrome-extension://"+id+"/")
				}
			}
			manifest["allowed_origins"] = origins
		}
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return nil, err
		}
		result = append(result, manifestTarget{filepath.Join(target.path, HostName+".json"), append(data, '\n'), target.browser})
	}
	return result, nil
}
func localTargets(executable string, extraChromiumIDs ...string) ([]manifestTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return manifestTargets(runtime.GOOS, home, config, executable, extraChromiumIDs...)
}

// Install registers the separately installed native host for the current user.
func Install(executable string, extraChromiumIDs ...string) error {
	if !filepath.IsAbs(executable) {
		return fmt.Errorf("native host path must be absolute")
	}
	info, err := os.Stat(executable)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("native host must be a regular executable")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("native host is not executable")
	}
	targets, err := localTargets(executable, extraChromiumIDs...)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err = os.MkdirAll(filepath.Dir(target.Path), 0700); err != nil {
			return err
		}
		file, err := os.CreateTemp(filepath.Dir(target.Path), ".passbolt-host-*")
		if err != nil {
			return err
		}
		temp := file.Name()
		_, writeErr := file.Write(target.Data)
		closeErr := file.Close()
		if writeErr != nil {
			_ = os.Remove(temp)
			return writeErr
		}
		if closeErr != nil {
			_ = os.Remove(temp)
			return closeErr
		}
		if err = os.Rename(temp, target.Path); err != nil {
			_ = os.Remove(temp)
			return err
		}
		if err = registerHost(target); err != nil {
			return err
		}
	}
	return nil
}
func Uninstall() error {
	targets, err := localTargets("")
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err = unregisterHost(target); err != nil {
			return err
		}
		if err = os.Remove(target.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
