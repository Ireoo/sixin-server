package socketio

import (
	"encoding/json"
	"strconv"

	"github.com/Ireoo/sixin-server/models"
	"github.com/zishang520/socket.io/v2/socket"
)

func (sim *SocketIOManager) handleGetRooms(client *socket.Socket, args ...any) {
	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}
	rooms, err := sim.baseInstance.DbManager.GetRooms(userID)
	if err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getRooms", rooms)
}

func (sim *SocketIOManager) handleGetRoomByUsers(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少房间ID")
		return
	}

	roomID, ok := args[0].(uint)
	if !ok {
		client.Emit("error", "房间ID格式错误")
		return
	}
	// 从 auth 中获取 userID
	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}

	aliases, err := sim.baseInstance.DbManager.GetRoomAliasByUsers(userID, roomID)
	if err != nil {
		client.Emit("error", "获取房间别名失败: "+err.Error())
		return
	}

	client.Emit("getRoomByUsers", aliases)
}

func (sim *SocketIOManager) handleCreateRoom(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少房间数据")
		return
	}

	var room models.Room
	if err := json.Unmarshal([]byte(args[0].(string)), &room); err != nil {
		client.Emit("error", "无效的房间数据")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}
	room.OwnerID = userID
	if err := sim.baseInstance.DbManager.CreateRoom(&room); err != nil {
		client.Emit("error", "创建房间失败")
		return
	}

	sim.baseInstance.DbManager.AddUserToRoom(userID, room.ID, "", false)

	client.Emit("roomCreated", room)
}

func (sim *SocketIOManager) handleUpdateRoom(client *socket.Socket, args ...any) {
	if len(args) < 1 {
		client.Emit("error", "缺少房间更新数据")
		return
	}

	var updatedRoom models.Room
	if err := json.Unmarshal([]byte(args[0].(string)), &updatedRoom); err != nil {
		client.Emit("error", "无效的房间数据")
		return
	}

	if updatedRoom.ID == 0 {
		client.Emit("error", "房间ID无效")
		return
	}

	if updatedRoom.OwnerID == 0 {
		client.Emit("error", "房间所有者ID无效")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}

	if sim.baseInstance.DbManager.CheckUserRoom(userID, updatedRoom.ID) != nil {
		client.Emit("error", "没有权限更新房间")
		return
	}

	if err := sim.baseInstance.DbManager.UpdateRoom(updatedRoom.ID, updatedRoom); err != nil {
		client.Emit("error", "更新房间失败")
		return
	}

	client.Emit("roomUpdated", updatedRoom)
}

func (sim *SocketIOManager) handleDeleteRoom(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少房间ID")
		return
	}

	roomID := args[0].(string)
	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		client.Emit("error", "无效的房间ID")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}

	if sim.baseInstance.DbManager.CheckUserRoom(userID, uint(roomIDUint)) != nil {
		client.Emit("error", "没有权限更新房间")
		return
	}

	if err := sim.baseInstance.DbManager.DeleteRoom(uint(roomIDUint)); err != nil {
		client.Emit("error", "删除房间失败")
		return
	}

	client.Emit("roomDeleted", roomID)
}

func (sim *SocketIOManager) handleAddUserToRoom(client *socket.Socket, args ...any) {
	if len(args) < 1 {
		client.Emit("error", "缺少房间ID")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}
	roomID, ok := args[0].(uint)

	if !ok {
		client.Emit("error", "无效的房间ID")
		return
	}

	err = sim.baseInstance.DbManager.AddUserToRoom(userID, roomID, "", false)
	if err != nil {
		client.Emit("error", "将用户添加到房间失败: "+err.Error())
		return
	}

	client.Emit("userAddedToRoom", map[string]uint{"userID": userID, "roomID": roomID})
}

func (sim *SocketIOManager) handleRemoveUserFromRoom(client *socket.Socket, args ...any) {
	if len(args) < 1 {
		client.Emit("error", "缺少房间ID")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}
	roomID, ok := args[0].(uint)

	if !ok {
		client.Emit("error", "无效的房间ID")
		return
	}

	err = sim.baseInstance.DbManager.RemoveUserFromRoom(userID, roomID)
	if err != nil {
		client.Emit("error", "将用户从房间移除失败: "+err.Error())
		return
	}

	client.Emit("userRemovedFromRoom", map[string]uint{"userID": userID, "roomID": roomID})
}

func (sim *SocketIOManager) handleUpdateRoomAlias(client *socket.Socket, args ...any) {
	if len(args) < 2 {
		client.Emit("error", "缺少房间ID或别名")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}
	roomID, ok := args[0].(uint)
	if !ok {
		client.Emit("error", "无效的房间ID")
		return
	}
	alias, ok := args[1].(string)

	if !ok {
		client.Emit("error", "无效的参数")
		return
	}

	err = sim.baseInstance.DbManager.UpdateRoomAlias(userID, roomID, alias)
	if err != nil {
		client.Emit("error", "更新房间别名失败: "+err.Error())
		return
	}

	client.Emit("roomAliasUpdated", map[string]interface{}{"userID": userID, "roomID": roomID, "alias": alias})
}

func (sim *SocketIOManager) handleSetRoomPrivacy(client *socket.Socket, args ...any) {
	if len(args) < 3 {
		client.Emit("error", "缺少房间ID、用户ID或隐私设置")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "获取用户ID失败: "+err.Error())
		return
	}
	roomID, ok := args[0].(uint)
	if !ok {
		client.Emit("error", "无效的房间ID")
		return
	}
	privacy, ok := args[2].(bool)
	if !ok {
		client.Emit("error", "无效的参数")
		return
	}

	err = sim.baseInstance.DbManager.SetRoomPrivacy(roomID, userID, privacy)
	if err != nil {
		client.Emit("error", "设置房间隐私失败: "+err.Error())
		return
	}

	client.Emit("roomPrivacySet", map[string]interface{}{"roomID": roomID, "userID": userID, "privacy": privacy})
}
