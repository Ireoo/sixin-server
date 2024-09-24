package socketio

import (
	"encoding/json"

	"github.com/Ireoo/sixin-server/models"
	"github.com/zishang520/socket.io/v2/socket"
)

func (sim *SocketIOManager) handleGetUsers(client *socket.Socket, args ...any) {
	users, err := sim.baseInstance.DbManager.GetAllUsers()
	if err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getUsers", users)
}

func (sim *SocketIOManager) handleUpdateUser(client *socket.Socket, args ...any) {
	if len(args) < 1 {
		client.Emit("error", "缺少更新数据")
		return
	}

	userIDUint, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}

	var updatedUser models.User
	if err := json.Unmarshal([]byte(args[0].(string)), &updatedUser); err != nil {
		client.Emit("error", "无效的用户数据")
		return
	}

	if err := sim.baseInstance.DbManager.UpdateUser(userIDUint, updatedUser); err != nil {
		client.Emit("error", "更新用户失败: "+err.Error())
		return
	}

	client.Emit("userUpdated", updatedUser)
}

func (sim *SocketIOManager) handleDeleteUser(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少用户ID")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}
	if err := sim.baseInstance.DbManager.DeleteUser(userID); err != nil {
		client.Emit("error", "删除用户失败")
		return
	}

	client.Emit("userDeleted", userID)
}

func (sim *SocketIOManager) handleAddFriend(client *socket.Socket, args ...any) {
	if len(args) < 1 {
		client.Emit("error", "缺少好友ID")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}
	friendID, ok := args[0].(uint)
	if !ok {
		client.Emit("error", "userID 类型转换失败")
		return
	}

	err = sim.baseInstance.DbManager.AddFriend(userID, friendID, "", false)
	if err != nil {
		client.Emit("error", "添加好友失败: "+err.Error())
		return
	}

	client.Emit("friendAdded", map[string]uint{"userID": userID, "friendID": friendID})
}

func (sim *SocketIOManager) handleRemoveFriend(client *socket.Socket, args ...any) {
	if len(args) < 1 {
		client.Emit("error", "缺少好友ID")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}

	friendID, ok := args[0].(uint)

	if !ok {
		client.Emit("error", "无效的好友ID")
		return
	}

	err = sim.baseInstance.DbManager.RemoveFriend(userID, friendID)
	if err != nil {
		client.Emit("error", "删除好友失败: "+err.Error())
		return
	}

	client.Emit("friendRemoved", map[string]uint{"userID": userID, "friendID": friendID})
}

func (sim *SocketIOManager) handleUpdateFriendAlias(client *socket.Socket, args ...any) {
	if len(args) < 2 {
		client.Emit("error", "缺少好友ID或别名")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}
	friendID, ok := args[0].(uint)
	if !ok {
		client.Emit("error", "无效的好友ID")
		return
	}
	alias, ok := args[1].(string)
	if !ok {
		client.Emit("error", "无效的别名")
		return
	}

	err = sim.baseInstance.DbManager.UpdateFriendAlias(userID, friendID, alias)
	if err != nil {
		client.Emit("error", "更新好友别名失败: "+err.Error())
		return
	}

	client.Emit("friendAliasUpdated", map[string]interface{}{"userID": userID, "friendID": friendID, "alias": alias})
}

func (sim *SocketIOManager) handleSetFriendPrivacy(client *socket.Socket, args ...any) {
	if len(args) < 3 {
		client.Emit("error", "缺少用户ID、好友ID或隐私设置")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}
	friendID, ok := args[0].(uint)
	if !ok {
		client.Emit("error", "无效的好友ID")
		return
	}
	privacy, ok := args[1].(bool)
	if !ok {
		client.Emit("error", "无效的隐私设置")
		return
	}

	err = sim.baseInstance.DbManager.SetFriendPrivacy(userID, friendID, privacy)
	if err != nil {
		client.Emit("error", "设置好友隐私失败: "+err.Error())
		return
	}

	client.Emit("friendPrivacySet", map[string]interface{}{"userID": userID, "friendID": friendID, "privacy": privacy})
}
