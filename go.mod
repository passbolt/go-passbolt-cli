module github.com/passbolt/go-passbolt-cli

go 1.16

require (
	github.com/alessio/shellescape v1.4.1
	github.com/gookit/color v1.5.0 // indirect
	github.com/passbolt/go-passbolt v0.5.5
	github.com/pterm/pterm v0.12.37
	github.com/spf13/afero v1.8.1 // indirect
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	github.com/tobischo/gokeepasslib/v3 v3.2.4
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292 // indirect
	golang.org/x/sys v0.0.0-20220222160653-b146bcec3beb // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/ini.v1 v1.66.4 // indirect
)

// replace github.com/passbolt/go-passbolt => ../go-passbolt
