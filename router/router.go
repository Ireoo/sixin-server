package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	httpHandler "github.com/Ireoo/sixin-server/internal/http"
	"github.com/Ireoo/sixin-server/internal/server"
	"github.com/Ireoo/sixin-server/internal/socketio"
	stunServer "github.com/Ireoo/sixin-server/internal/stun"
	"github.com/Ireoo/sixin-server/internal/websocket"
	"github.com/Ireoo/sixin-server/logger"
)

func SetupAndRun(cfg *config.Config) {
	// 创建 base.Base 实例
	baseInstance := base.NewBase()

	// 获取数据库实例
	err := database.InitDatabase(database.DatabaseType(cfg.DBType), cfg.DBConn)
	if err != nil {
		logger.Error("Failed to initialize database:", err)
		return
	}
	db := database.GetCurrentDB()

	// 将数据库实例保存到 base 中
	baseInstance.SetDB(db.GetDB())

	// 设置 Socket.IO 事件处理
	io := socketio.SetupSocketHandlers(db.GetDB(), baseInstance)

	// 将 Socket.IO 实例保存到 base 中
	baseInstance.SetIO(io)

	http.Handle("/socket.io/", io.ServeHandler(nil))

	// 设置 HTTP 处理程序
	httpHandler.SetupHTTPHandlers(baseInstance)

	// 设置 STUN 服务器
	go func() {
		stunAddress := fmt.Sprintf("%s:%d", cfg.Host, cfg.StunPort)
		ctx := context.Background()
		if err := stunServer.StartSTUNServer(ctx, stunAddress); err != nil {
			logger.Error("Failed to start STUN server:", err)
		}
	}()

	// 创建 WebSocketManager
	wsManager := websocket.NewWebSocketManager()

	// 设置 WebSocket 路由
	http.HandleFunc("/ws", wsManager.HandleWebSocket)

	// 将 WebSocketManager 保存到 baseInstance
	baseInstance.SetWebSocketManager(wsManager)

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
