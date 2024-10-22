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
		emitError(client, "获取用户ID失败", err)
		return
	}
	rooms, err := sim.baseInstance.DbManager.GetRooms(userID)
	if err != nil {
		emitError(client, "获取房间列表失败", err)
		return
	}
	client.Emit("getRooms", rooms)
}

func (sim *SocketIOManager) handleGetRoomByUsers(client *socket.Socket, args ...any) {
	roomID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少房间ID或ID类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	aliases, err := sim.baseInstance.DbManager.GetRoomAliasByUsers(userID, roomID)
	if err != nil {
		emitError(client, "获取房间别名失败", err)
		return
	}

	client.Emit("getRoomByUsers", aliases)
}

func (sim *SocketIOManager) handleCreateRoom(client *socket.Socket, args ...any) {
	data, err := checkArgsAndType[string](args, 0)
	if err != nil {
		emitError(client, "缺少房间数据或数据类型错误", err)
		return
	}

	var room models.Room
	if err := json.Unmarshal([]byte(data), &room); err != nil {
		emitError(client, "无效的房间数据", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}
	room.OwnerID = userID
	if err := sim.baseInstance.DbManager.CreateRoom(&room); err != nil {
		emitError(client, "创建房间失败", err)
		return
	}

	sim.baseInstance.DbManager.AddUserToRoom(userID, room.ID, "", false)

	client.Emit("roomCreated", room)
}

func (sim *SocketIOManager) handleUpdateRoom(client *socket.Socket, args ...any) {
	data, err := checkArgsAndType[string](args, 0)
	if err != nil {
		emitError(client, "缺少房间更新数据或数据类型错误", err)
		return
	}

	var updatedRoom models.Room
	if err := json.Unmarshal([]byte(data), &updatedRoom); err != nil {
		emitError(client, "无效的房间数据", err)
		return
	}

	if updatedRoom.ID == 0 {
		emitError(client, "房间ID无效", nil)
		return
	}

	if updatedRoom.OwnerID == 0 {
		emitError(client, "房间所有者ID无效", nil)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	if sim.baseInstance.DbManager.CheckUserRoom(userID, updatedRoom.ID) != nil {
		emitError(client, "没有权限更新房间", nil)
		return
	}

	if err := sim.baseInstance.DbManager.UpdateRoomByOwner(userID, updatedRoom.ID, updatedRoom); err != nil {
		emitError(client, "更新房间失败", err)
		return
	}

	client.Emit("roomUpdated", updatedRoom)
}

func (sim *SocketIOManager) handleDeleteRoom(client *socket.Socket, args ...any) {
	roomIDStr, err := checkArgsAndType[string](args, 0)
	if err != nil {
		emitError(client, "缺少房间ID或ID类型错误", err)
		return
	}

	roomIDUint, err := strconv.ParseUint(roomIDStr, 10, 64)
	if err != nil {
		emitError(client, "无效的房间ID", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	if sim.baseInstance.DbManager.CheckUserRoom(userID, uint(roomIDUint)) != nil {
		emitError(client, "没有权限更新房间", nil)
		return
	}

	if err := sim.baseInstance.DbManager.DeleteRoom(userID, uint(roomIDUint)); err != nil {
		emitError(client, "删除房间失败", err)
		return
	}

	client.Emit("roomDeleted", roomIDStr)
}

func (sim *SocketIOManager) handleAddUserToRoom(client *socket.Socket, args ...any) {
	roomID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少房间ID或ID类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	err = sim.baseInstance.DbManager.AddUserToRoom(userID, roomID, "", false)
	if err != nil {
		emitError(client, "将用户添加到房间失败", err)
		return
	}

	client.Emit("userAddedToRoom", map[string]uint{"userID": userID, "roomID": roomID})
}

func (sim *SocketIOManager) handleRemoveUserFromRoom(client *socket.Socket, args ...any) {
	roomID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少房间ID或ID类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	err = sim.baseInstance.DbManager.RemoveUserFromRoom(userID, roomID)
	if err != nil {
		emitError(client, "将用户从房间移除失败", err)
		return
	}

	client.Emit("userRemovedFromRoom", map[string]uint{"userID": userID, "roomID": roomID})
}

func (sim *SocketIOManager) handleUpdateRoomAlias(client *socket.Socket, args ...any) {
	roomID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少房间ID或ID类型错误", err)
		return
	}

	alias, err := checkArgsAndType[string](args, 1)
	if err != nil {
		emitError(client, "缺少别名或别名类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	err = sim.baseInstance.DbManager.UpdateRoomAlias(userID, roomID, alias)
	if err != nil {
		emitError(client, "更新房间别名失败", err)
		return
	}

	client.Emit("roomAliasUpdated", map[string]interface{}{"userID": userID, "roomID": roomID, "alias": alias})
}

func (sim *SocketIOManager) handleSetRoomPrivacy(client *socket.Socket, args ...any) {
	roomID, err := checkArgsAndType[uint](args, 0)
	if err != nil {
		emitError(client, "缺少房间ID或ID类型错误", err)
		return
	}

	privacy, err := checkArgsAndType[bool](args, 1)
	if err != nil {
		emitError(client, "缺少隐私设置或设置类型错误", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	err = sim.baseInstance.DbManager.SetRoomPrivacy(roomID, userID, privacy)
	if err != nil {
		emitError(client, "设置房间隐私失败", err)
		return
	}

	client.Emit("roomPrivacySet", map[string]interface{}{"roomID": roomID, "userID": userID, "privacy": privacy})
}
