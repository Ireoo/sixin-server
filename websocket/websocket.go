package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/Ireoo/sixin-server/message"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源,生产环境中应该更严格
	},
}

type WebSocketManager struct {
	connections    map[string][]*websocket.Conn
	mu             sync.RWMutex
	sendMessage    func(text, msg string)
	messageHandler *message.MessageHandler
}

func NewWebSocketManager(sendMessageFunc func(text, msg string), messageHandler *message.MessageHandler) *WebSocketManager {
	return &WebSocketManager{
		connections:    make(map[string][]*websocket.Conn),
		sendMessage:    sendMessageFunc,
		messageHandler: messageHandler,
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
		wsm.handleMessage(connType, message)
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

func (wsm *WebSocketManager) handleMessage(connType string, message []byte) {
	// 处理从 WebSocket 接收到的消息
	log.Printf("Received WebSocket message from %s: %s", connType, string(message))

	// 使用 MessageHandler 处理消息
	if err := wsm.messageHandler.HandleMessage(message); err != nil {
		log.Printf("Error handling message: %v", err)
		return
	}

	// 调用 sendMessage 函数处理消息
	wsm.sendMessage(string(message), string(message))
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
