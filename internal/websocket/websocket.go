package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/internal/middleware"
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
	middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			// 处理错误
			return
		}
		defer conn.Close()

		// 获取用户ID
		userID := r.Header.Get("UserID")

		// 处理WebSocket连接
		wsm.handleConnection(conn, userID, r)
	})).ServeHTTP(w, r)
}

func (wsm *WebSocketManager) handleConnection(conn *websocket.Conn, userID string, r *http.Request) {
	connType := r.URL.Query().Get("type")
	if connType == "" {
		connType = "web"
	}

	wsm.addConnection(connType, conn)
	defer wsm.removeConnection(connType, conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		wsm.handleMessage(message)
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

func (wsm *WebSocketManager) handleMessage(msgBytes []byte) {
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
		wsm.handleChatMessage(&message)
	case "addFriend":
		wsm.handleAddFriend(genericMessage.Data)
	case "removeFriend":
		wsm.handleRemoveFriend(genericMessage.Data)
	case "updateFriendAlias":
		wsm.handleUpdateFriendAlias(genericMessage.Data)
	case "setFriendPrivacy":
		wsm.handleSetFriendPrivacy(genericMessage.Data)
	case "addUserToRoom":
		wsm.handleAddUserToRoom(genericMessage.Data)
	case "removeUserFromRoom":
		wsm.handleRemoveUserFromRoom(genericMessage.Data)
	case "updateRoomAlias":
		wsm.handleUpdateRoomAlias(genericMessage.Data)
	case "setRoomPrivacy":
		wsm.handleSetRoomPrivacy(genericMessage.Data)
	default:
		log.Printf("未知的消息类型: %s", genericMessage.Type)
	}
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

func (wsm *WebSocketManager) handleAddFriend(data json.RawMessage) {
	var friendRequest struct {
		UserID    uint   `json:"user_id"`
		FriendID  uint   `json:"friend_id"`
		Alias     string `json:"alias"`
		IsPrivate bool   `json:"is_private"`
	}
	if err := json.Unmarshal(data, &friendRequest); err != nil {
		log.Printf("解析添加好友请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.AddFriend(friendRequest.UserID, friendRequest.FriendID, friendRequest.Alias, friendRequest.IsPrivate)
	if err != nil {
		log.Printf("添加好友失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(friendRequest.UserID, "好友添加成功")
	wsm.sendNotification(friendRequest.FriendID, "您有新的好友请求")
}

func (wsm *WebSocketManager) handleRemoveFriend(data []byte) {
	var friendRequest struct {
		UserID   uint `json:"user_id"`
		FriendID uint `json:"friend_id"`
	}
	if err := json.Unmarshal(data, &friendRequest); err != nil {
		log.Printf("解析删除好友请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.RemoveFriend(friendRequest.UserID, friendRequest.FriendID)
	if err != nil {
		log.Printf("删除好友失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(friendRequest.UserID, "好友删除成功")
	wsm.sendNotification(friendRequest.FriendID, "您已被移除好友列表")
}

func (wsm *WebSocketManager) handleUpdateFriendAlias(data []byte) {
	var aliasRequest struct {
		UserID   uint   `json:"user_id"`
		FriendID uint   `json:"friend_id"`
		Alias    string `json:"alias"`
	}
	if err := json.Unmarshal(data, &aliasRequest); err != nil {
		log.Printf("解析更新好友别名请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.UpdateFriendAlias(aliasRequest.UserID, aliasRequest.FriendID, aliasRequest.Alias)
	if err != nil {
		log.Printf("更新好友别名失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(aliasRequest.UserID, "好友别名更新成功")
}

func (wsm *WebSocketManager) handleSetFriendPrivacy(data []byte) {
	var privacyRequest struct {
		UserID    uint `json:"user_id"`
		FriendID  uint `json:"friend_id"`
		IsPrivate bool `json:"is_private"`
	}
	if err := json.Unmarshal(data, &privacyRequest); err != nil {
		log.Printf("解析设置好友隐私请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.SetFriendPrivacy(privacyRequest.UserID, privacyRequest.FriendID, privacyRequest.IsPrivate)
	if err != nil {
		log.Printf("设置好友隐私失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(privacyRequest.UserID, "好友隐私设置更新成功")
}

func (wsm *WebSocketManager) handleAddUserToRoom(data []byte) {
	var roomRequest struct {
		UserID    uint   `json:"user_id"`
		RoomID    uint   `json:"room_id"`
		Alias     string `json:"alias"`
		IsPrivate bool   `json:"is_private"`
	}
	if err := json.Unmarshal(data, &roomRequest); err != nil {
		log.Printf("解析添加用户到房间请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.AddUserToRoom(roomRequest.UserID, roomRequest.RoomID, roomRequest.Alias, roomRequest.IsPrivate)
	if err != nil {
		log.Printf("添加用户到房间失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(roomRequest.UserID, "您已被添加到新的房间")
	// 可以考虑通知房间内的其他成员
}

func (wsm *WebSocketManager) handleRemoveUserFromRoom(data []byte) {
	var roomRequest struct {
		UserID uint `json:"user_id"`
		RoomID uint `json:"room_id"`
	}
	if err := json.Unmarshal(data, &roomRequest); err != nil {
		log.Printf("解析从房间移除用户请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.RemoveUserFromRoom(roomRequest.UserID, roomRequest.RoomID)
	if err != nil {
		log.Printf("从房间移除用户失败: %v", err)
		return
	}

	// 发送通知给相关用户
	wsm.sendNotification(roomRequest.UserID, "您已被移出房间")
	// 可以考虑通知房间内的其他成员
}

func (wsm *WebSocketManager) handleUpdateRoomAlias(data []byte) {
	var aliasRequest struct {
		UserID uint   `json:"user_id"`
		RoomID uint   `json:"room_id"`
		Alias  string `json:"alias"`
	}
	if err := json.Unmarshal(data, &aliasRequest); err != nil {
		log.Printf("解析更新房间别名请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.UpdateRoomAlias(aliasRequest.UserID, aliasRequest.RoomID, aliasRequest.Alias)
	if err != nil {
		log.Printf("更新房间别名失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(aliasRequest.UserID, "房间别名更新成功")
}

func (wsm *WebSocketManager) handleSetRoomPrivacy(data []byte) {
	var privacyRequest struct {
		UserID    uint `json:"user_id"`
		RoomID    uint `json:"room_id"`
		IsPrivate bool `json:"is_private"`
	}
	if err := json.Unmarshal(data, &privacyRequest); err != nil {
		log.Printf("解析设置房间隐私请求数据失败: %v", err)
		return
	}

	err := wsm.baseInstance.DbManager.SetRoomPrivacy(privacyRequest.UserID, privacyRequest.RoomID, privacyRequest.IsPrivate)
	if err != nil {
		log.Printf("设置房间隐私失败: %v", err)
		return
	}

	// 发送通知给用户
	wsm.sendNotification(privacyRequest.UserID, "房间隐私设置更新成功")
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
