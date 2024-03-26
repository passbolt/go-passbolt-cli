package main

import (
	"time"

	"github.com/carlmjohnson/versioninfo"
	"github.com/passbolt/go-passbolt-cli/cmd"
)

func main() {
	cmd.SetVersionInfo(versioninfo.Version, versioninfo.Revision, versioninfo.LastCommit.Format(time.RFC3339))
	cmd.Execute()
}
