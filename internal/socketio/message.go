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
		emitError(client, "userID 类型转换失败", err)
		return
	}
	messages, err := sim.baseInstance.DbManager.GetChats(userID)
	if err != nil {
		emitError(client, "获取聊天记录失败", err)
		return
	}
	client.Emit("getChats", messages)
}

func (sim *SocketIOManager) handleMessage(client *socket.Socket, args ...any) {
	msgBytes, err := checkArgsAndType[[]byte](args, 0)
	if err != nil {
		emitError(client, "缺少消息内容或消息格式错误", err)
		return
	}

	fmt.Println("收到消息：", string(msgBytes))

	var message models.Message
	if err := json.Unmarshal(msgBytes, &message); err != nil {
		emitError(client, "解析消息失败", err)
		return
	}

	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "userID 类型转换失败", err)
		return
	}
	message.TalkerID = userID

	if err := sim.baseInstance.DbManager.CreateMessage(&message); err != nil {
		emitError(client, "保存消息失败", err)
		return
	}

	fullMessage, err := sim.baseInstance.DbManager.GetFullMessage(message.ID)
	if err != nil {
		emitError(client, "加载完整消息数据失败", err)
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
