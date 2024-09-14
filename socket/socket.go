package socket

import (
	"encoding/json"
	"fmt"
	"github.com/Ireoo/sixin-server/base"
	models "github.com/Ireoo/sixin-server/models"
	"github.com/google/uuid"
	socketio "github.com/googollee/go-socket.io"
	"gorm.io/gorm"
)

var db *gorm.DB
var baseInstance *base.Base
var activeSockets map[string]socketio.Conn = make(map[string]socketio.Conn)

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
	// 添加连接到集合
	activeSockets[s.ID()] = s
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
	// 从集合中移除连接
	delete(activeSockets, s.ID())
	fmt.Println("closed", reason)
}

func handleMessage(s socketio.Conn, msg string) {
	var data struct {
		Message struct {
			MsgID      string `json:"msgId"`
			TalkerID   uint   `json:"talkerId"`
			ListenerID uint   `json:"listenerId"`
			RoomID     uint   `json:"roomId"`
			Text       struct {
				Message string `json:"message"`
				Image   string `json:"image"`
			} `json:"text"`
			Timestamp     int64  `json:"timestamp"`
			Type          int    `json:"type"`
			MentionIDList string `json:"mentionIdList"`
		} `json:"message"`
	}

	if err := json.Unmarshal([]byte(msg), &data); err != nil {
		s.Emit("error", "消息格式错误: "+err.Error())
		return
	}

	// 将消息记录到数据库，并发送给对应的用户
	if data.Message.MsgID == "" {
		data.Message.MsgID = uuid.New().String()
	}

	message := models.Message{
		MsgID:         data.Message.MsgID,
		TalkerID:      data.Message.TalkerID,
		ListenerID:    data.Message.ListenerID,
		Text:          data.Message.Text.Message,
		Timestamp:     data.Message.Timestamp,
		Type:          data.Message.Type,
		MentionIDList: data.Message.MentionIDList,
		RoomID:        data.Message.RoomID,
	}
	// 记录消息到数据库
	if err := recordMessage(message); err != nil {
		s.Emit("error", "保存消息错误: "+err.Error())
	}
	// 定义一个发送信息 sendData 是一个 user或者room和message结构数据
	sendData := struct {
		User    models.User    `json:"user"`
		Room    models.Room    `json:"room"`
		Message models.Message `json:"message"`
	}{
		User:    models.User{ID: message.TalkerID},
		Room:    models.Room{ID: message.RoomID},
		Message: message,
	}
	// 发送消息给对应的用户
	sendMessageToUser(message.TalkerID, sendData)
	sendMessageToUser(message.ListenerID, sendData)
}

// sendMessageToUser 发送消息给指定的用户
func sendMessageToUser(userID uint, message interface{}) {
	// 遍历activeSockets数组，发送消息
	for _, socket := range activeSockets {
		if userContext, ok := socket.Context().(map[string]interface{}); ok {
			if id, ok := userContext["user_id"].(uint); ok && id == userID {
				socket.Emit("message", message)
				fmt.Printf("Message sent to user %d: %+v\n", userID, message)
			}
		}
	}
}

// recordMessage 将消息记录到数据库
func recordMessage(message models.Message) error {
	// 假设使用 GORM 进行数据库操作
	result := db.Create(&message)
	return result.Error
}
