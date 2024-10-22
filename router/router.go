package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	httpHandler "github.com/Ireoo/sixin-server/internal/http"
	"github.com/Ireoo/sixin-server/internal/server"
	"github.com/Ireoo/sixin-server/internal/socketio"
	"github.com/Ireoo/sixin-server/internal/websocket"
	"github.com/Ireoo/sixin-server/logger"
)

func SetupAndRun(cfg *config.Config) {
	// 创建 base.Base 实例
	baseInstance := base.NewBase(cfg)

	if baseInstance == nil {
		logger.Error("创建 base 实例失败")
		return
	}

	// 设置 Socket.IO 路由
	ioManager := socketio.NewSocketIOManager(baseInstance)
	baseInstance.IoManager = ioManager.Io
	http.Handle("/socket.io/", baseInstance.IoManager.ServeHandler(nil))

	// 设置 HTTP 处理程序
	httpHandler.SetupHTTPHandlers(baseInstance)

	// 设置 WebSocket 路由
	WebSocketManager := websocket.NewWebSocketManager(baseInstance)
	http.HandleFunc("/ws", WebSocketManager.HandleWebSocket)

	// 创建 http.Server 实例
	serverInstance := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	server.StartServer(serverInstance, baseInstance)
}
