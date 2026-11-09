// Command construction is serialized by the agent dispatcher.
package folder

func ResetCommands() {

	FolderCreateCmd = makeFolderCreateCmd()
	FolderDeleteCmd = makeFolderDeleteCmd()
	FolderGetCmd = makeFolderGetCmd()
	FolderPermissionCmd = makeFolderPermissionCmd()
	FolderListCmd = makeFolderListCmd()
	FolderMoveCmd = makeFolderMoveCmd()
	FolderShareCmd = makeFolderShareCmd()
	FolderUpdateCmd = makeFolderUpdateCmd()
	configureCreate()
	configureGet()
	configureList()
	configureMove()
	configureShare()
	configureUpdate()

}
