package socket

import (
	"fmt"
	"github.com/Ireoo/sixin-server/base"
	models "github.com/Ireoo/sixin-server/models"
	socketio "github.com/googollee/go-socket.io"
	"gorm.io/gorm"
)

var db *gorm.DB
var baseInstance *base.Base

func SetupSocketHandlers(server *socketio.Server, database *gorm.DB, baseInst *base.Base) {
	db = database
	baseInstance = baseInst

	server.OnConnect("/", handleConnect)
	server.OnEvent("/", "self", handleSelf)
	server.OnEvent("/", "receive", handleReceive)
	server.OnEvent("/", "message", handleMessage)
	server.OnEvent("/", "email", handleEmail)
	server.OnEvent("/", "revokemsg", handleRevokeMsg)
	server.OnEvent("/", "getChats", handleGetChats)
	server.OnEvent("/", "getRooms", handleGetRooms)
	server.OnEvent("/", "getUsers", handleGetUsers)
	server.OnEvent("/", "getRoomByUsers", handleGetRoomByUsers)
	server.OnDisconnect("/", handleDisconnect)
}

func handleConnect(s socketio.Conn) error {
	s.SetContext("")
	fmt.Println("connected:", s.ID())
	s.Emit("receive", baseInstance.ReceiveDevice)
	s.Emit("email", baseInstance.EmailNote)
	s.Emit("self", baseInstance.Self)
	s.Emit("qrcode", baseInstance.Qrcode)
	return nil
}

func handleSelf(s socketio.Conn, msg string) {
	s.Emit("self", baseInstance.Self)
}

func handleReceive(s socketio.Conn, msg string) {
	baseInstance.ReceiveDevice = !baseInstance.ReceiveDevice
	message := "wechat:receive"
	if !baseInstance.ReceiveDevice {
		message = "wechat:message"
	}
	baseInstance.SendMessage(message, message)
	s.Emit("receive", baseInstance.ReceiveDevice)
}

func handleEmail(s socketio.Conn, msg string) {
	baseInstance.EmailNote = !baseInstance.EmailNote
	s.Emit("email", baseInstance.EmailNote)
}

func handleRevokeMsg(s socketio.Conn, id string) {
	// 实现撤回消息的逻辑
}
func handleGetChats(s socketio.Conn) {
	var messages []models.Message
	result := db.Order("timestamp DESC").Limit(400).Find(&messages)
	if result.Error != nil {
		s.Emit("error", result.Error.Error())
		return
	}
	s.Emit("getChats", messages)
}

func handleGetRooms(s socketio.Conn) {
	var rooms []models.Room
	result := db.Order("updated_at DESC").Find(&rooms)
	if result.Error != nil {
		s.Emit("error", result.Error.Error())
		return
	}
	s.Emit("getRooms", rooms)
}

func handleGetUsers(s socketio.Conn) {
	var users []models.User
	result := db.Order("updated_at DESC").Find(&users)
	if result.Error != nil {
		s.Emit("error", result.Error.Error())
		return
	}
	s.Emit("getUsers", users)
}

func handleGetRoomByUsers(s socketio.Conn) {
	// 实现获取用户房间的逻辑
}

func handleDisconnect(s socketio.Conn, reason string) {
	fmt.Println("closed", reason)
}

func handleMessage(s socketio.Conn, msg string) {
	// 实现消息处理逻辑
}
