package message

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Ireoo/sixin-server/models"
	"github.com/google/uuid"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

type MessageHandler struct {
	DB *gorm.DB
	IO *socket.Server
}

func NewMessageHandler(db *gorm.DB, io *socket.Server) *MessageHandler {
	return &MessageHandler{DB: db, IO: io}
}

func (mh *MessageHandler) HandleMessage(messageData []byte) error {
	var data struct {
		Message struct {
			MsgID         string                 `json:"msgId"`
			TalkerID      uint                   `json:"talkerId"`
			ListenerID    uint                   `json:"listenerId"`
			RoomID        uint                   `json:"roomId"`
			Text          map[string]interface{} `json:"text"`
			Timestamp     int64                  `json:"timestamp"`
			Type          int                    `json:"type"`
			MentionIDList []uint                 `json:"mentionIdList"`
		} `json:"message"`
	}

	if err := json.Unmarshal(messageData, &data); err != nil {
		return fmt.Errorf("消息格式错误: %v", err)
	}

	// 检查并转换 MentionIDList 字段
	if mentionIDList, ok := data.Message.Text["mentionIdList"].([]interface{}); ok {
		var ids []uint
		for _, id := range mentionIDList {
			if idFloat, ok := id.(float64); ok {
				ids = append(ids, uint(idFloat))
			}
		}
		data.Message.MentionIDList = ids
	} else {
		data.Message.MentionIDList = []uint{}
	}

	if data.Message.MsgID == "" {
		data.Message.MsgID = uuid.New().String()
	}

	message := models.Message{
		MsgID:         data.Message.MsgID,
		TalkerID:      data.Message.TalkerID,
		ListenerID:    data.Message.ListenerID,
		Text:          data.Message.Text,
		Timestamp:     data.Message.Timestamp,
		Type:          data.Message.Type,
		MentionIDList: data.Message.MentionIDList,
		RoomID:        data.Message.RoomID,
	}

	if err := mh.RecordMessage(&message); err != nil {
		return fmt.Errorf("保存消息错误: %v", err)
	}

	// 加载关联的用户和房间信息
	if err := mh.DB.Preload("Talker").Preload("Listener").Preload("Room").
		First(&message, "msg_id = ?", message.MsgID).Error; err != nil {
		return fmt.Errorf("加载消息关联信息错误: %v", err)
	}

	return nil
}

func (mh *MessageHandler) RecordMessage(message *models.Message) error {
	return mh.DB.Create(message).Error
}

func (mh *MessageHandler) GetChats() ([]models.Message, error) {
	var messages []models.Message
	if err := mh.DB.Preload("Talker").Preload("Listener").Preload("Room").
		Order("timestamp DESC").Limit(400).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// 其他消息相关的方法...

// 添加 GetMessageByID 方法
func (mh *MessageHandler) GetMessageByID(msgID string) (*models.Message, error) {
	var message models.Message
	err := mh.DB.Preload("Talker").Preload("Listener").Preload("Room").
		First(&message, "msg_id = ?", msgID).Error
	if err != nil {
		return nil, fmt.Errorf("获取消息失败: %v", err)
	}
	return &message, nil
}

// ... 保留其他现有的方法 ...

// 在 message/message.go 文件中添加以下方法

func (mh *MessageHandler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var messageData []byte
	if err := json.NewDecoder(r.Body).Decode(&messageData); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	if err := mh.HandleMessage(messageData); err != nil {
		http.Error(w, "处理消息失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var msgData struct {
		Message struct {
			MsgID string `json:"msgId"`
		} `json:"message"`
	}
	if err := json.Unmarshal(messageData, &msgData); err != nil {
		http.Error(w, "解析消息ID失败", http.StatusInternalServerError)
		return
	}

	message, err := mh.GetMessageByID(msgData.Message.MsgID)
	if err != nil {
		http.Error(w, "获取消息失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var recipientID uint
	if message.ListenerID != 0 {
		recipientID = message.ListenerID
	} else {
		recipientID = message.RoomID
	}

	sendData := struct {
		Talker   *models.User    `json:"talker"`
		Listener *models.User    `json:"listener,omitempty"`
		Room     *models.Room    `json:"room,omitempty"`
		Message  *models.Message `json:"message"`
	}{
		Talker:   message.Talker,
		Listener: message.Listener,
		Room:     message.Room,
		Message:  message,
	}

	// 直接在这里发送消息给用户
	mh.sendMessageToUsers(sendData, message.TalkerID, recipientID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "消息已发送",
		"data":    message,
	})
}

// sendMessageToUsers 发送消息给多个用户
func (mh *MessageHandler) sendMessageToUsers(message interface{}, userIDs ...uint) {
	for _, userID := range userIDs {
		socketID := socket.SocketId(fmt.Sprintf("%d", userID))
		clients := mh.IO.Sockets().Sockets()
		clients.Range(func(id socket.SocketId, client *socket.Socket) bool {
			if client.Id() == socketID {
				err := client.Emit("message", message)
				if err != nil {
					fmt.Printf("发送消息给用户 %d 失败: %v\n", userID, err)
				}
				return false
			}
			return true
		})
	}
}
