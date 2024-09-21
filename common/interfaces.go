package common

import (
	"net/http"

	"github.com/zishang520/socket.io/v2/socket"
)

type WebSocketManager interface {
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	SendMessage(channel string, message []byte)
}

type DatabaseManager interface {
	// 添加需要的数据库方法接口
}

type Base interface {
	SetIO(io *socket.Server)
	SetWebSocketManager(wsManager WebSocketManager)
	SetDatabaseManager(dbManager DatabaseManager)
	GetWebSocketManager() WebSocketManager
	GetIO() *socket.Server
	SendMessageToUsers(message interface{}, userIDs ...uint)
}
