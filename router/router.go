package router

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/handlers"
	"github.com/Ireoo/sixin-server/middleware"
	"github.com/Ireoo/sixin-server/socket"
	"github.com/Ireoo/sixin-server/webrtc"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		log.Printf("| %3d | %13v | %15s | %s  %s\n%s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
			raw,
		)
	}
}

func SetupAndRun(cfg *config.Config) {
	// 创建Gin引擎
	r := gin.Default()

	// 创建 base.Base 实例
	baseInstance := &base.Base{}

	// 设置Socket.IO事件处理
	socketServer := initSocketServer(baseInstance)

	// 初始化WebRTC服务器
	webrtcServer := initWebRTCServer()

	// 设置中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(loggerMiddleware())

	// Socket.IO路由
	r.GET("/socket.io/*any", gin.WrapH(socketServer))
	r.POST("/socket.io/*any", gin.WrapH(socketServer))

	// WebRTC路由
	r.GET("/webrtc", func(c *gin.Context) {
		webrtcServer.HandleWebRTC(c.Writer, c.Request)
	})

	// API路由组
	api := r.Group("/api")
	{
		api.GET("/ping", handlers.Ping)

		// 用户相关路由
		users := api.Group("/users")
		{
			users.GET("", handlers.GetUsers)
			users.POST("", handlers.CreateUser)
			users.GET("/:id", handlers.GetUser)
			users.PUT("/:id", handlers.UpdateUser)
			users.DELETE("/:id", handlers.DeleteUser)
		}

		// 可以添加更多API路由...
	}

	// 静态文件服务
	r.Static("/static", "./static")

	// 404处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Not Found"})
	})

	// 启动服务器
	startServer(r, cfg)
}

func initSocketServer(baseInstance *base.Base) *socketio.Server {
	socketServer := socketio.NewServer(nil)
	if socketServer == nil {
		log.Fatal("创建Socket.IO服务器失败")
	}
	db := database.GetCurrentDB()
	socket.SetupSocketHandlers(socketServer, db.GetDB(), baseInstance)
	log.Println("Socket.IO服务器创建成功")
	return socketServer
}

func initWebRTCServer() *webrtc.WebRTCServer {
	webrtcServer, err := webrtc.NewWebRTCServer()
	if err != nil {
		log.Fatalf("初始化WebRTC服务器失败: %v", err)
	}
	return webrtcServer
}

func startServer(r *gin.Engine, cfg *config.Config) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("服务器运行在 %s...\n", addr)
	log.Fatal(r.Run(addr))
}
