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
	"github.com/Ireoo/sixin-server/socket-io"
	"github.com/Ireoo/sixin-server/webrtc"
	"github.com/zishang520/socket.io/v2/socket"
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
	// 创建路由器
	mux := http.NewServeMux()

	// 创建 base.Base 实例
	baseInstance := &base.Base{}

	// 获取数据库实例
	db := database.GetCurrentDB()

	// 设置Socket.IO事件处理
	socketIo.SetupSocketHandlers(db.GetDB(), baseInstance)
	socketServer := socket.NewServer(mux, nil)

	// 初始化WebRTC服务器
	webrtcServer := initWebRTCServer()

	// 设置中间件
	handler := handleRoutes(webrtcServer)
	handler = loggerMiddleware(handler)
	handler = middleware.Logger(handler)
	handler = middleware.CORS(handler)
	mux.HandleFunc("/", handler)

	// 静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 修改路由处理
	mux.Handle("/socket.io/", socketServer.ServeHandler(nil))

	// 启动服务器
	startServer(mux, cfg)
}

func handleRoutes(webrtcServer *webrtc.WebRTCServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
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

func initWebRTCServer() *webrtc.WebRTCServer {
	webrtcServer, err := webrtc.NewWebRTCServer()
	if err != nil {
		log.Fatalf("初始化WebRTC服务器失败: %v", err)
	}
	return webrtcServer
}
