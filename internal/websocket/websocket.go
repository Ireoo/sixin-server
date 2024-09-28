package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/models"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源,生产环境中应该更严格
	},
}

type WebSocketManager struct {
	connections  map[string][]*websocket.Conn
	mu           sync.RWMutex
	baseInstance *base.Base
}

func NewWebSocketManager(base *base.Base) *WebSocketManager {
	return &WebSocketManager{
		connections:  make(map[string][]*websocket.Conn),
		baseInstance: base,
	}
}

func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到WebSocket连接请求: %s", r.URL)
	// 移除身份验证中间件
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 由于没有身份验证，我们需要一种方式来识别用户
	// 这里使用一个简单的随机ID作为示例
	userID := uint(rand.Uint32())

	// 处理WebSocket连接
	wsm.handleConnection(conn, userID, r)
}

func (wsm *WebSocketManager) handleConnection(conn *websocket.Conn, userID uint, r *http.Request) {
	connType := r.URL.Query().Get("type")
	if connType == "" {
		connType = "web"
	}

	userConnType := fmt.Sprintf("user_%d", userID)
	wsm.addConnection(userConnType, conn)
	defer wsm.removeConnection(userConnType, conn)

	// 发送一个欢迎消息，包含分配的userID
	welcomeMsg := map[string]interface{}{
		"type":   "welcome",
		"userID": userID,
	}
	welcomeMsgJSON, _ := json.Marshal(welcomeMsg)
	conn.WriteMessage(websocket.TextMessage, welcomeMsgJSON)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket读取错误: %v", err)
			break
		}
		wsm.handleMessage(message, userID)
	}
}

func (wsm *WebSocketManager) addConnection(connType string, conn *websocket.Conn) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()
	wsm.connections[connType] = append(wsm.connections[connType], conn)
}

func (wsm *WebSocketManager) removeConnection(connType string, conn *websocket.Conn) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()
	connections := wsm.connections[connType]
	for i, c := range connections {
		if c == conn {
			wsm.connections[connType] = append(connections[:i], connections[i+1:]...)
			break
		}
	}
}

func (wsm *WebSocketManager) handleMessage(msgBytes []byte, userID uint) {
	// 解析通用消息结构
	var genericMessage struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(msgBytes, &genericMessage); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	// 根据消息类型处理不同的操作
	switch genericMessage.Type {
	case "message":
		var message models.Message
		if err := json.Unmarshal(genericMessage.Data, &message); err != nil {
			log.Printf("解析消息数据失败: %v", err)
			return
		}
		message.TalkerID = userID // 使用身份验证获取的用户ID
		wsm.handleChatMessage(&message)
	case "addFriend":
		wsm.handleAddFriend(genericMessage.Data, userID)
	case "removeFriend":
		wsm.handleRemoveFriend(genericMessage.Data, userID)
	case "updateFriendAlias":
		wsm.handleUpdateFriendAlias(genericMessage.Data, userID)
	case "setFriendPrivacy":
		wsm.handleSetFriendPrivacy(genericMessage.Data, userID)
	case "addUserToRoom":
		wsm.handleAddUserToRoom(genericMessage.Data, userID)
	case "removeUserFromRoom":
		wsm.handleRemoveUserFromRoom(genericMessage.Data, userID)
	case "updateRoomAlias":
		wsm.handleUpdateRoomAlias(genericMessage.Data, userID)
	case "setRoomPrivacy":
		wsm.handleSetRoomPrivacy(genericMessage.Data, userID)
	case "getRoomAliasByUsers":
		wsm.handleGetRoomAliasByUsers(genericMessage.Data, userID)
	default:
		log.Printf("未知的消息类型: %s", genericMessage.Type)
	}
}

// 新增函数处理获取房间别名
func (wsm *WebSocketManager) handleGetRoomAliasByUsers(data json.RawMessage, userID uint) {
	var roomID uint
	if err := json.Unmarshal(data, &roomID); err != nil {
		log.Printf("解析房间ID失败: %v", err)
		return
	}

	aliases, err := wsm.baseInstance.DbManager.GetRoomAliasByUsers(userID, roomID)
	if err != nil {
		log.Printf("获取房间别名失败: %v", err)
		return
	}

	response := map[string]interface{}{
		"type":    "getRoomByUsers",
		"aliases": aliases,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("序列化响应失败: %v", err)
		return
	}

	wsm.sendMessageToUsers(responseJSON, userID)
}

// 新增函数处理聊天消息
func (wsm *WebSocketManager) handleChatMessage(message *models.Message) {
	if err := wsm.baseInstance.DbManager.CreateMessage(message); err != nil {
		log.Printf("保存消息失败: %v", err)
		return
	}

	fullMessage, err := wsm.baseInstance.DbManager.GetFullMessage(message.ID)
	if err != nil {
		log.Printf("加载完整消息数据失败: %v", err)
		return
	}

	var recipientID uint
	if message.ListenerID != 0 {
		recipientID = message.ListenerID
	} else {
		recipientID = message.RoomID
	}

	wsm.sendMessageToUsers(fullMessage, message.TalkerID, recipientID)
}

func (wsm *WebSocketManager) handleAddFriend(data json.RawMessage, userID uint) {
	var friendRequest struct {
		FriendID  uint   `json:"friend_id"`
		Alias     string `json:"alias"`
		IsPrivate bool   `json:"is_private"`
	}
	if err := json.Unmarshal(data, &friendRequest); err != nil {
		log.Printf("解析添加好友请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.AddFriend(userID, friendRequest.FriendID, friendRequest.Alias, friendRequest.IsPrivate)
	if err != nil {
		log.Printf("添加好友失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(userID, "好友添加成功")
	wsm.sendNotification(friendRequest.FriendID, "您有新的好友请求")
}

func (wsm *WebSocketManager) handleRemoveFriend(data []byte, userID uint) {
	var friendRequest struct {
		FriendID uint `json:"friend_id"`
	}
	if err := json.Unmarshal(data, &friendRequest); err != nil {
		log.Printf("解析删除好友请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.RemoveFriend(userID, friendRequest.FriendID)
	if err != nil {
		log.Printf("删除好友失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(userID, "好友删除成功")
	wsm.sendNotification(friendRequest.FriendID, "您已被移除好友列表")
}

func (wsm *WebSocketManager) handleUpdateFriendAlias(data []byte, userID uint) {
	var aliasRequest struct {
		FriendID uint   `json:"friend_id"`
		Alias    string `json:"alias"`
	}
	if err := json.Unmarshal(data, &aliasRequest); err != nil {
		log.Printf("解析更新好友别名请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.UpdateFriendAlias(userID, aliasRequest.FriendID, aliasRequest.Alias)
	if err != nil {
		log.Printf("更新好友别名失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(userID, "好友别名更新成功")
}

func (wsm *WebSocketManager) handleSetFriendPrivacy(data []byte, userID uint) {
	var privacyRequest struct {
		FriendID  uint `json:"friend_id"`
		IsPrivate bool `json:"is_private"`
	}
	if err := json.Unmarshal(data, &privacyRequest); err != nil {
		log.Printf("解析设置好友隐私请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.SetFriendPrivacy(userID, privacyRequest.FriendID, privacyRequest.IsPrivate)
	if err != nil {
		log.Printf("设置好友隐私失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(userID, "好友隐私设置更新成功")
}

func (wsm *WebSocketManager) handleAddUserToRoom(data []byte, userID uint) {
	var roomRequest struct {
		RoomID    uint   `json:"room_id"`
		Alias     string `json:"alias"`
		IsPrivate bool   `json:"is_private"`
	}
	if err := json.Unmarshal(data, &roomRequest); err != nil {
		log.Printf("解析添加用户到房间请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.AddUserToRoom(userID, roomRequest.RoomID, roomRequest.Alias, roomRequest.IsPrivate)
	if err != nil {
		log.Printf("添加用户到房间失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(userID, "您已被添加到新的房间")
	// 可以考虑通知房间内的其他成员
}

func (wsm *WebSocketManager) handleRemoveUserFromRoom(data []byte, userID uint) {
	var roomRequest struct {
		RoomID uint `json:"room_id"`
	}
	if err := json.Unmarshal(data, &roomRequest); err != nil {
		log.Printf("解析从房间移除用户请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.RemoveUserFromRoom(userID, roomRequest.RoomID)
	if err != nil {
		log.Printf("从房间移除用户失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(userID, "您已被移出房间")
	// 可以考虑通知房间内的其他成员
}

func (wsm *WebSocketManager) handleUpdateRoomAlias(data []byte, userID uint) {
	var aliasRequest struct {
		RoomID uint   `json:"room_id"`
		Alias  string `json:"alias"`
	}
	if err := json.Unmarshal(data, &aliasRequest); err != nil {
		log.Printf("解析更新房间别名请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.UpdateRoomAlias(userID, aliasRequest.RoomID, aliasRequest.Alias)
	if err != nil {
		log.Printf("更新房间别名失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(userID, "房间别名更新成功")
}

func (wsm *WebSocketManager) handleSetRoomPrivacy(data []byte, userID uint) {
	var privacyRequest struct {
		RoomID    uint `json:"room_id"`
		IsPrivate bool `json:"is_private"`
	}
	if err := json.Unmarshal(data, &privacyRequest); err != nil {
		log.Printf("解析设置房间隐私请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.SetRoomPrivacy(userID, privacyRequest.RoomID, privacyRequest.IsPrivate)
	if err != nil {
		log.Printf("设置房间隐私失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(userID, "房间隐私设置更新成功")
}

func (wsm *WebSocketManager) sendNotification(userID uint, message string) {
	notification := map[string]interface{}{
		"type":    "notification",
		"message": message,
	}
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		log.Printf("序列化通知失败: %v", err)
		return
	}
	wsm.sendMessageToUsers(notificationJSON, userID)
}

func (wsm *WebSocketManager) sendMessageToUsers(message interface{}, userIDs ...uint) {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return
	}

	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	for _, userID := range userIDs {
		userConnType := fmt.Sprintf("user_%d", userID)
		for _, conn := range wsm.connections[userConnType] {
			if err := conn.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
				log.Printf("发送消息给用户 %d 失败: %v", userID, err)
			}
		}
	}
}

func (wsm *WebSocketManager) SendMessage(connType string, message []byte) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()
	for _, conn := range wsm.connections[connType] {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Error sending message to %s: %v", connType, err)
		}
	}
}
