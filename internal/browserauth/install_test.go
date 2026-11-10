package browserauth

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeManifestsRestrictOfficialExtensionIdentities(t *testing.T) {
	for _, platform := range []string{"linux", "darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			targets, err := manifestTargets(platform, "/home/test", "/config", "/bin/passbolt-browser-host")
			if err != nil {
				t.Fatal(err)
			}
			if len(targets) < 2 {
				t.Fatal("missing browser family")
			}
			for _, target := range targets {
				var manifest map[string]any
				if err = json.Unmarshal(target.Data, &manifest); err != nil {
					t.Fatal(err)
				}
				if manifest["name"] != HostName || manifest["path"] != "/bin/passbolt-browser-host" || manifest["type"] != "stdio" {
					t.Fatalf("invalid host manifest %s", target.Data)
				}
				if strings.Contains(string(target.Data), "*") {
					t.Fatal("wildcard extension authorization")
				}
				if !filepath.IsAbs(target.Path) {
					t.Fatal("relative manifest path")
				}
			}
		})
	}
}
