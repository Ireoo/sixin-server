package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Ireoo/sixin-server/models"
	"github.com/Ireoo/sixin-server/socketio"
)

// 响应结构体
type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// sendJSON 辅助函数，用于发送 JSON 响应
func sendJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// Ping 处理 ping 请求
func Ping(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, Response{Message: "pong"})
}

// GetUsers 获取所有用户
func GetUsers(w http.ResponseWriter, r *http.Request) {
	// 这里应该实现获取用户列表的逻辑
	sendJSON(w, http.StatusOK, Response{Message: "获取所有用户"})
}

// CreateUser 创建新用户
func CreateUser(w http.ResponseWriter, r *http.Request) {
	// 这里应该实现创建用户的逻辑
	sendJSON(w, http.StatusCreated, Response{Message: "创建新用户"})
}

// GetUser 获取特定用户
func GetUser(w http.ResponseWriter, r *http.Request, id string) {
	// 这里应该实现获取特定用户的逻辑
	sendJSON(w, http.StatusOK, Response{
		Message: "获取用户",
		Data:    map[string]string{"id": id},
	})
}

// UpdateUser 更新用户信息
func UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	// 这里应该实现更新用户信息的逻辑
	sendJSON(w, http.StatusOK, Response{
		Message: "更新用户信息",
		Data:    map[string]string{"id": id},
	})
}

// DeleteUser 删除用户
func DeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	// 这里应该实现删除用户的逻辑
	sendJSON(w, http.StatusOK, Response{
		Message: "删除用户",
		Data:    map[string]string{"id": id},
	})
}

// CreateMessage 创建并发送消息
func CreateMessage(w http.ResponseWriter, r *http.Request) {
	var message models.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSON(w, http.StatusBadRequest, Response{Message: "无效的请求数据", Data: err.Error()})
		return
	}

	// 保存消息到数据库
	if err := socketio.RecordMessage(&message); err != nil {
		sendJSON(w, http.StatusInternalServerError, Response{Message: "保存消息失败", Data: err.Error()})
		return
	}

	// 发送消息给发送者和接收者
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
		Message:  &message,
	}

	socketio.SendMessageToUsers(sendData, message.TalkerID, recipientID)

	sendJSON(w, http.StatusCreated, Response{Message: "消息已发送", Data: message})
}
