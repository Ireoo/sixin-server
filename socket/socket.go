package socket

import (
	"database/sql"
	"fmt"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/models"

	socketio "github.com/googollee/go-socket.io"
)

var db *sql.DB
var baseInstance *base.Base

func SetupSocketHandlers(server *socketio.Server, database *sql.DB, baseInst *base.Base) {
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
	rows, err := db.Query("SELECT * FROM messages ORDER BY timestamp DESC LIMIT 400")
	if err != nil {
		s.Emit("error", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var msg models.Message
		err := rows.Scan(&msg.ID, &msg.Text, &msg.Timestamp) // 根据实际字段调整
		if err != nil {
			s.Emit("error", err.Error())
			return
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		s.Emit("error", err.Error())
		return
	}
	s.Emit("getChats", messages)
}

func handleGetRooms(s socketio.Conn) {
	var rooms []models.Room
	rows, err := db.Query("SELECT * FROM rooms ORDER BY updated_at DESC")
	if err != nil {
		s.Emit("error", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var room models.Room
		err := rows.Scan(&room.ID, &room.Topic, &room.UpdatedAt) // 根据实际字段调整
		if err != nil {
			s.Emit("error", err.Error())
			return
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		s.Emit("error", err.Error())
		return
	}
	s.Emit("getRooms", rooms)
}

func handleGetUsers(s socketio.Conn) {
	var users []models.User
	rows, err := db.Query("SELECT * FROM users ORDER BY updated_at DESC")
	if err != nil {
		s.Emit("error", err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Name, &user.UpdatedAt) // 根据实际字段调整
		if err != nil {
			s.Emit("error", err.Error())
			return
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		s.Emit("error", err.Error())
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
