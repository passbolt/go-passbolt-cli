// Command construction is serialized by the agent dispatcher.
package user

func ResetCommands() {

	UserCreateCmd = makeUserCreateCmd()
	UserDeleteCmd = makeUserDeleteCmd()
	UserGetCmd = makeUserGetCmd()
	UserListCmd = makeUserListCmd()
	UserUpdateCmd = makeUserUpdateCmd()
	configureCreate()
	configureGet()
	configureList()
	configureUpdate()

}
