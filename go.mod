module github.com/speatzle/go-passbolt-cli

go 1.16

require (
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/pterm/pterm v0.12.30
	github.com/speatzle/go-passbolt v0.2.1
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	golang.org/x/sys v0.0.0-20210902050250-f475640dd07b // indirect
)

replace github.com/speatzle/go-passbolt => ../go-passbolt
