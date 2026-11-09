// Command construction is serialized by the agent dispatcher.
package resource

func ResetCommands() {

	ResourceCreateCmd = makeResourceCreateCmd()
	ResourceDeleteCmd = makeResourceDeleteCmd()
	ResourceGetCmd = makeResourceGetCmd()
	ResourcePermissionCmd = makeResourcePermissionCmd()
	ResourceListCmd = makeResourceListCmd()
	ResourceMoveCmd = makeResourceMoveCmd()
	ResourceShareCmd = makeResourceShareCmd()
	ResourceUpdateCmd = makeResourceUpdateCmd()
	configureCreate()
	configureGet()
	configureList()
	configureMove()
	configureShare()
	configureUpdate()

}
