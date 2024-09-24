package socketio

import (
	"encoding/json"
	"fmt"

	"github.com/Ireoo/sixin-server/models"
	"github.com/zishang520/socket.io/v2/socket"
)

func (sim *SocketIOManager) handleGetChats(client *socket.Socket, args ...any) {
	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}
	messages, err := sim.baseInstance.DbManager.GetChats(userID)
	if err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getChats", messages)
}

func (sim *SocketIOManager) handleMessage(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少消息内容")
		return
	}

	var msgBytes []byte
	switch arg := args[0].(type) {
	case string:
		msgBytes = []byte(arg)
	case []byte:
		msgBytes = arg
	default:
		client.Emit("error", "消息格式错误")
		return
	}

	fmt.Println("收到消息：", string(msgBytes))

	var message models.Message
	if err := json.Unmarshal(msgBytes, &message); err != nil {
		client.Emit("error", "解析消息失败")
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		client.Emit("error", "userID 类型转换失败")
		return
	}
	message.TalkerID = userID

	if err := sim.baseInstance.DbManager.CreateMessage(&message); err != nil {
		client.Emit("error", "保存消息失败")
		return
	}

	fullMessage, err := sim.baseInstance.DbManager.GetFullMessage(message.ID)
	if err != nil {
		client.Emit("error", "加载完整消息数据失败")
		return
	}

	var recipientID uint
	if message.ListenerID != 0 {
		recipientID = message.ListenerID
	} else {
		recipientID = message.RoomID
	}

	sim.SendMessageToUsers(fullMessage, message.TalkerID, recipientID)
}
