package socketIo

import (
	"encoding/json"
	"fmt"

	"github.com/Ireoo/sixin-server/base"
	models "github.com/Ireoo/sixin-server/models"
	"github.com/google/uuid"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

var db *gorm.DB
var baseInstance *base.Base
var io *socket.Server

func SetupSocketHandlers(database *gorm.DB, baseInst *base.Base) {
	db = database
	baseInstance = baseInst

	io = socket.NewServer(nil, nil)

	io.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		fmt.Println("新连接：", client.Id())

		client.Emit("receive", baseInstance.ReceiveDevice)
		client.Emit("email", baseInstance.EmailNote)
		client.Emit("self", baseInstance.Self)
		client.Emit("qrcode", baseInstance.Qrcode)

		client.On("self", func(args ...any) {
			handleSelf(client)
		})
		client.On("receive", func(args ...any) {
			handleReceive(client)
		})
		client.On("message", func(args ...any) {
			handleMessage(client, args...)
		})
		client.On("email", func(args ...any) {
			handleEmail(client)
		})
		client.On("revokemsg", func(args ...any) {
			handleRevokeMsg(client, args...)
		})
		client.On("getChats", func(args ...any) {
			handleGetChats(client, args...)
		})
		client.On("getRooms", func(args ...any) {
			handleGetRooms(client, args...)
		})
		client.On("getUsers", func(args ...any) {
			handleGetUsers(client, args...)
		})
		client.On("getRoomByUsers", func(args ...any) {
			handleGetRoomByUsers(client, args...)
		})

		client.On("disconnecting", func(reason ...any) {
			fmt.Println("连接断开:", client.Id(), reason)
		})
	})
}

func handleSelf(client *socket.Socket) {
	client.Emit("self", baseInstance.Self)
}

func handleReceive(client *socket.Socket) {
	baseInstance.ReceiveDevice = !baseInstance.ReceiveDevice
	message := "wechat:receive"
	if !baseInstance.ReceiveDevice {
		message = "wechat:message"
	}
	baseInstance.SendMessage(message, message)
	client.Emit("receive", baseInstance.ReceiveDevice)
}

func handleEmail(client *socket.Socket) {
	baseInstance.EmailNote = !baseInstance.EmailNote
	client.Emit("email", baseInstance.EmailNote)
}

func handleRevokeMsg(client *socket.Socket, args ...any) {
	// 实现撤回消息的逻辑
}

func handleGetChats(client *socket.Socket, args ...any) {
	var messages []models.Message
	result := db.Order("timestamp DESC").Limit(400).Find(&messages)
	if result.Error != nil {
		client.Emit("error", result.Error.Error())
		return
	}
	client.Emit("getChats", messages)
}

func handleGetRooms(client *socket.Socket, args ...any) {
	var rooms []models.Room
	result := db.Order("updated_at DESC").Find(&rooms)
	if result.Error != nil {
		client.Emit("error", result.Error.Error())
		return
	}
	client.Emit("getRooms", rooms)
}

func handleGetUsers(client *socket.Socket, args ...any) {
	var users []models.User
	result := db.Order("updated_at DESC").Find(&users)
	if result.Error != nil {
		client.Emit("error", result.Error.Error())
		return
	}
	client.Emit("getUsers", users)
}

func handleGetRoomByUsers(client *socket.Socket, args ...any) {
	// 实现获取用户房间的逻辑
}

func handleMessage(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		return
	}
	msgStr, ok := args[0].(string)
	if !ok {
		client.Emit("error", "消息格式错误")
		return
	}

	fmt.Println("收到消息：", msgStr)
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

	if err := json.Unmarshal([]byte(msgStr), &data); err != nil {
		client.Emit("error", "消息格式错误: "+err.Error())
		return
	}

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

	if err := recordMessage(message); err != nil {
		client.Emit("error", "保存消息错误: "+err.Error())
	}

	sendData := struct {
		User    models.User    `json:"user"`
		Room    models.Room    `json:"room"`
		Message models.Message `json:"message"`
	}{
		User:    models.User{ID: message.TalkerID},
		Room:    models.Room{ID: message.RoomID},
		Message: message,
	}

	sendMessageToUser(message.TalkerID, sendData)
	sendMessageToUser(message.ListenerID, sendData)
}

func sendMessageToUser(userID uint, message interface{}) {
	io.Sockets().Emit("message", message)
}

func recordMessage(message models.Message) error {
	result := db.Create(&message)
	return result.Error
}

func GetSocketIOServer() *socket.Server {
	return io
}
