package socketio

import (
	"encoding/json"

	"github.com/Ireoo/sixin-server/models"
	"github.com/zishang520/socket.io/v2/socket"
)

func (sim *SocketIOManager) handleGetUsers(client *socket.Socket, args ...any) {
	go func() {
		users, err := sim.baseInstance.DbManager.GetAllUsers()
		if err != nil {
			emitError(client, "获取用户列表失败", err)
			return
		}
		client.Emit("getUsers", users)
	}()
}

func (sim *SocketIOManager) handleUpdateUser(client *socket.Socket, args ...any) {
	data, err := checkArgsAndType[string](args, 0)
	if err != nil {
		emitError(client, "缺少更新数据或数据类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}

	var updatedUser models.User
	if err := json.Unmarshal([]byte(data), &updatedUser); err != nil {
		emitError(client, "无效的用户数据", err)
		return
	}
	go func() {
		if err := sim.baseInstance.DbManager.UpdateUserOwn(userID, &updatedUser); err != nil {
			emitError(client, "更新用户失败", err)
			return
		}

		client.Emit("userUpdated", updatedUser)
	}()
}

func (sim *SocketIOManager) handleDeleteUser(client *socket.Socket, args ...any) {
	_, err := checkArgsAndType[string](args, 0)
	if err != nil {
		emitError(client, "缺少用户ID或ID类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}
	go func() {
		if err := sim.baseInstance.DbManager.DeleteUser(userID); err != nil {
			emitError(client, "删除用户失败", err)
			return
		}

		client.Emit("userDeleted", userID)
	}()
}

func (sim *SocketIOManager) handleAddFriend(client *socket.Socket, args ...any) {
	friendID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少好友ID或ID类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}
	go func() {
		err = sim.baseInstance.DbManager.AddFriend(userID, friendID, "", false)
		if err != nil {
			emitError(client, "添加好友失败", err)
			return
		}

		client.Emit("friendAdded", map[string]uint{"userID": userID, "friendID": friendID})
	}()
}

func (sim *SocketIOManager) handleRemoveFriend(client *socket.Socket, args ...any) {
	friendID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少好友ID或ID类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}
	go func() {
		err = sim.baseInstance.DbManager.RemoveFriend(userID, friendID)
		if err != nil {
			emitError(client, "删除好友失败", err)
			return
		}

		client.Emit("friendRemoved", map[string]uint{"userID": userID, "friendID": friendID})
	}()
}

func (sim *SocketIOManager) handleUpdateFriendAlias(client *socket.Socket, args ...any) {
	friendID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少好友ID或ID类型错误", err)
		return
	}

	alias, err := checkArgsAndType[string](args, 1)
	if err != nil {
		emitError(client, "缺少别名或别名类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}
	go func() {
		err = sim.baseInstance.DbManager.UpdateFriendAlias(userID, friendID, alias)
		if err != nil {
			emitError(client, "更新好友别名失败", err)
			return
		}

		client.Emit("friendAliasUpdated", map[string]interface{}{"userID": userID, "friendID": friendID, "alias": alias})
	}()
}

func (sim *SocketIOManager) handleSetFriendPrivacy(client *socket.Socket, args ...any) {
	friendID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少好友ID或ID类型错误", err)
		return
	}

	privacy, err := checkArgsAndType[bool](args, 1)
	if err != nil {
		emitError(client, "缺少隐私设置或设置类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}
	go func() {
		err = sim.baseInstance.DbManager.SetFriendPrivacy(userID, friendID, privacy)
		if err != nil {
			emitError(client, "设置好友隐私失败", err)
			return
		}

		client.Emit("friendPrivacySet", map[string]interface{}{"userID": userID, "friendID": friendID, "privacy": privacy})
	}()
}
