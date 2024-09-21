package websocket

import (
	"encoding/json"
	"fmt"
	"log"
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

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
	var message models.Message
	if err := json.Unmarshal(msgBytes, &message); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	if err := wsm.baseInstance.DbManager.CreateMessage(&message); err != nil {
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

	// 如果还需要调用原有的 sendMessage 函数，可以保留这行
	// wsm.sendMessage(string(msgBytes), string(msgBytes))
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
