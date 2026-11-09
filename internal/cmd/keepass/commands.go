// Command construction is serialized by the agent dispatcher.
package keepass

func ResetCommands() {

	KeepassExportCmd = makeKeepassExportCmd()
	configureExport()

}
