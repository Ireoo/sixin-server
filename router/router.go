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
	socketio "github.com/googollee/go-socket.io"
)

func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery

		next.ServeHTTP(w, r)

		latency := time.Since(start)
		clientIP := r.RemoteAddr
		method := r.Method
		statusCode := http.StatusOK // 注意：这里无法获取真实的状态码，需要自定义 ResponseWriter

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
	// 创建 base.Base 实例
	baseInstance := &base.Base{}

	// 设置Socket.IO事件处理
	socketServer := initSocketServer(baseInstance)

	go socketServer.Serve()
	defer socketServer.Close()

	// 初始化WebRTC服务器
	webrtcServer := initWebRTCServer()

	// 创建路由器
	mux := http.NewServeMux()
	// 设置中间件
	handler := handleRoutes(socketServer, webrtcServer)
	handler = loggerMiddleware(handler)
	handler = middleware.Logger(handler)
	handler = middleware.CORS(handler)
	mux.HandleFunc("/", handler)

	// 静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 启动服务器
	startServer(mux, cfg)
}

func handleRoutes(socketServer *socketio.Server, webrtcServer *webrtc.WebRTCServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/socket.io/":
			socketServer.ServeHTTP(w, r)
		case "/webrtc":
			webrtcServer.HandleWebRTC(w, r)
		case "/api/ping":
			handlers.Ping(w, r)
		case "/api/users":
			switch r.Method {
			case http.MethodGet:
				handlers.GetUsers(w, r)
			case http.MethodPost:
				handlers.CreateUser(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			if r.URL.Path[:10] == "/api/users/" {
				id := r.URL.Path[10:]
				switch r.Method {
				case http.MethodGet:
					handlers.GetUser(w, r, id)
				case http.MethodPut:
					handlers.UpdateUser(w, r, id)
				case http.MethodDelete:
					handlers.DeleteUser(w, r, id)
				default:
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			} else {
				http.NotFound(w, r)
			}
		}
	}
}

// initSocketServer 和 initWebRTCServer 函数保持不变

func startServer(mux *http.ServeMux, cfg *config.Config) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("服务器运行在 %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
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
