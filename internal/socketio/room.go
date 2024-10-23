package socketio

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Ireoo/sixin-server/models"
	"github.com/patrickmn/go-cache"
	"github.com/zishang520/socket.io/v2/socket"
)

var roomCache = cache.New(5*time.Minute, 10*time.Minute)
var roomUpdateLimiter = make(chan struct{}, 10) // 最多允许 10 个并发更新
var pool = sync.Pool{
	New: func() interface{} {
		return &models.Room{}
	},
}

func (sim *SocketIOManager) handleGetRooms(client *socket.Socket, args ...any) {
	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}

	cacheKey := fmt.Sprintf("rooms_user_%d", userID)
	if cachedRooms, found := roomCache.Get(cacheKey); found {
		client.Emit("getRooms", cachedRooms)
		return
	}
	go func() {
		rooms, err := sim.baseInstance.DbManager.GetRooms(userID)
		if err != nil {
			emitError(client, "获取房间列表失败", err)
			return
		}

		// 缓存房间信息
		roomCache.Set(cacheKey, rooms, cache.DefaultExpiration)
		client.Emit("getRooms", rooms)
	}()
}

func (sim *SocketIOManager) handleCreateRoom(client *socket.Socket, args ...any) {
	data, err := checkArgsAndType[string](args, 0)
	if err != nil {
		emitError(client, "缺少房间数据或数据类型错误", err)
		return
	}

	room := pool.Get().(*models.Room)
	defer pool.Put(room)

	if err := json.Unmarshal([]byte(data), room); err != nil {
		emitError(client, "无效的房间数据", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}
	room.OwnerID = userID

	// 异步创建房间，避免阻塞
	go func() {
		if err := sim.baseInstance.DbManager.CreateRoom(room); err != nil {
			emitError(client, "创建房间失败", err)
			return
		}

		if err := sim.baseInstance.DbManager.AddUserToRoom(userID, room.ID, "", false); err != nil {
			emitError(client, "将用户添加到房间失败", err)
			return
		}

		client.Emit("roomCreated", room)
	}()
}

func (sim *SocketIOManager) handleUpdateRoom(client *socket.Socket, args ...any) {
	roomUpdateLimiter <- struct{}{}        // 获取并发槽
	defer func() { <-roomUpdateLimiter }() // 请求完成后释放槽

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

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "获取用户ID失败", err)
		return
	}
	go func() {
		if sim.baseInstance.DbManager.CheckUserRoom(userID, updatedRoom.ID) != nil {
			emitError(client, "没有权限更新房间", nil)
			return
		}

		if err := sim.baseInstance.DbManager.UpdateRoomByOwner(userID, updatedRoom.ID, updatedRoom); err != nil {
			emitError(client, "更新房间失败", err)
			return
		}

		client.Emit("roomUpdated", updatedRoom)
	}()
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

	go func() {
		if sim.baseInstance.DbManager.CheckUserRoom(userID, uint(roomIDUint)) != nil {
			emitError(client, "没有权限更新房间", nil)
			return
		}

		if err := sim.baseInstance.DbManager.DeleteRoom(userID, uint(roomIDUint)); err != nil {
			emitError(client, "删除房间失败", err)
			return
		}

		client.Emit("roomDeleted", roomIDStr)
	}()
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

	// 使用 goroutine 池处理添加用户请求
	go func() {
		err = sim.baseInstance.DbManager.AddUserToRoom(userID, roomID, "", false)
		if err != nil {
			emitError(client, "将用户添加到房间失败", err)
			return
		}

		client.Emit("userAddedToRoom", map[string]uint{"userID": userID, "roomID": roomID})
	}()
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

	go func() {
		err = sim.baseInstance.DbManager.RemoveUserFromRoom(userID, roomID)
		if err != nil {
			emitError(client, "将用户从房间移除失败", err)
			return
		}

		client.Emit("userRemovedFromRoom", map[string]uint{"userID": userID, "roomID": roomID})
	}()
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

	go func() {
		err = sim.baseInstance.DbManager.UpdateRoomAlias(userID, roomID, alias)
		if err != nil {
			emitError(client, "更新房间别名失败", err)
			return
		}

		client.Emit("roomAliasUpdated", map[string]interface{}{"userID": userID, "roomID": roomID, "alias": alias})
	}()
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

	go func() {
		err = sim.baseInstance.DbManager.SetRoomPrivacy(roomID, userID, privacy)
		if err != nil {
			emitError(client, "设置房间隐私失败", err)
			return
		}

		client.Emit("roomPrivacySet", map[string]interface{}{"roomID": roomID, "userID": userID, "privacy": privacy})
	}()
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

	// 异步获取房间信息，避免阻塞
	go func() {
		room, err := sim.baseInstance.DbManager.GetRoomAliasByUsers(userID, roomID)
		if err != nil {
			emitError(client, "获取房间信息失败", err)
			return
		}

		client.Emit("getRoomByUsers", room)
	}()
}
