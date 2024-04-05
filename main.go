package main

import (
	"github.com/carlmjohnson/versioninfo"
	"github.com/passbolt/go-passbolt-cli/cmd"
	"time"
)

// this variable is one of the goreleaser's default ldflag
// which considers the latest git tag as version (the v prefix is stripped)
// More info: https://goreleaser.com/cookbooks/using-main.version/
var version string = "(devel)"

func main() {
	cmd.SetVersionInfo(version, versioninfo.Revision, versioninfo.LastCommit.Format(time.RFC3339))
	cmd.Execute()
}
