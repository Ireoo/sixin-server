package websocket

import (
	"log"
	"net/http"

	"github.com/Ireoo/sixin-server/base"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源,生产环境中应该更严格
	},
}

func HandleWebSocket(baseInstance *base.Base) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade to WebSocket: %v", err)
			return
		}
		defer conn.Close()

		// 获取连接类型 (m5stack, telegram, web)
		connType := r.URL.Query().Get("type")
		if connType == "" {
			connType = "web" // 默认为 web
		}

		// 将连接添加到 base 中的相应组
		baseInstance.AddWebSocketConn(connType, conn)

		// 处理 WebSocket 消息
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				baseInstance.RemoveWebSocketConn(connType, conn)
				break
			}
			// 处理接收到的消息
			baseInstance.HandleWebSocketMessage(connType, messageType, p)
		}
	}
}
