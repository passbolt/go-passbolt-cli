// Command construction is serialized by the agent dispatcher.
package group

func ResetCommands() {

	GroupCreateCmd = makeGroupCreateCmd()
	GroupDeleteCmd = makeGroupDeleteCmd()
	GroupGetCmd = makeGroupGetCmd()
	GroupListCmd = makeGroupListCmd()
	GroupUpdateCmd = makeGroupUpdateCmd()
	configureCreate()
	configureGet()
	configureList()
	configureUpdate()

}
