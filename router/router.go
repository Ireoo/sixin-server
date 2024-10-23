package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	httpHandler "github.com/Ireoo/sixin-server/internal/http"
	"github.com/Ireoo/sixin-server/internal/socketio"
	"github.com/Ireoo/sixin-server/logger"
	"github.com/gorilla/mux"
)

func SetupAndRun(cfg *config.Config) {
	baseInstance := base.NewBase(cfg)
	if baseInstance == nil {
		logger.Error("创建 base 实例失败")
		return
	}

	r := mux.NewRouter()

	// 设置 Socket.IO 路由
	ioManager := socketio.NewSocketIOManager(baseInstance)
	baseInstance.IoManager = ioManager.Io
	r.Handle("/socket.io/", baseInstance.IoManager.ServeHandler(nil))

	// 设置 HTTP 处理程序
	httpManager := httpHandler.NewHTTPManager(baseInstance)
	httpManager.SetupRoutes(r)

	// 创建 http.Server 实例
	serverInstance := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	logger.Info(fmt.Sprintf("服务器正在监听 %s", serverInstance.Addr))
	if err := serverInstance.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(fmt.Sprintf("服务器启动失败: %v", err))
	}
}
