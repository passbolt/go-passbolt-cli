// Command construction is serialized by the agent dispatcher.
package cmd

import (
	"github.com/passbolt/go-passbolt-cli/internal/cmd/folder"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/group"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/keepass"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/resource"
	"github.com/passbolt/go-passbolt-cli/internal/cmd/user"
	"github.com/spf13/cobra"
)

func newCommandTree() *cobra.Command {
	folder.ResetCommands()
	group.ResetCommands()
	resource.ResetCommands()
	user.ResetCommands()
	keepass.ResetCommands()
	completionCmd = makecompletionCmd()
	configureCmd = makeconfigureCmd()
	createCmd = makecreateCmd()
	deleteCmd = makedeleteCmd()
	genDocCmd = makegenDocCmd()
	execCmd = makeexecCmd()
	exportCmd = makeexportCmd()
	getCmd = makegetCmd()
	listCmd = makelistCmd()
	moveCmd = makemoveCmd()
	rootCmd = makerootCmd()
	shareCmd = makeshareCmd()
	updateCmd = makeupdateCmd()
	verifyCMD = makeverifyCMD()
	configureCompletion()
	configureConfigure()
	configureCreate()
	configureDelete()
	configureDoc()
	configureExec()
	configureExport()
	configureGet()
	configureList()
	configureMove()
	configureRoot()
	rootCmd.AddCommand(newGitCredentialCommand(), newDockerCredentialCommand(), newCredentialHelperCommand())
	configureShare()
	configureUpdate()
	configureVerify()
	return rootCmd
}
