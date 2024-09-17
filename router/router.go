package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/handlers"
	"github.com/Ireoo/sixin-server/middleware"
	"github.com/Ireoo/sixin-server/socketio"
	stunServer "github.com/Ireoo/sixin-server/stun"
	"github.com/zishang520/socket.io/v2/socket"
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery

		next.ServeHTTP(sw, r)

		latency := time.Since(start)
		clientIP := r.RemoteAddr
		method := r.Method
		statusCode := sw.statusCode

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

func chainMiddlewares(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middlewares {
		handler = m(handler)
	}
	return handler
}

func SetupAndRun(cfg *config.Config) {
	// 创建 base.Base 实例
	baseInstance := &base.Base{}

	// 获取数据库实例
	err := database.InitDatabase(database.DatabaseType(cfg.DBType), cfg.DBConn)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db := database.GetCurrentDB()

	// 设置 Socket.IO 事件处理
	io := socketio.SetupSocketHandlers(db.GetDB(), baseInstance)
	http.Handle("/socket.io/", io.ServeHandler(nil))

	// 设置中间件和路由
	handler := chainMiddlewares(
		handleRoutes(),
		loggerMiddleware,
		middleware.CORS,
	)
	http.HandleFunc("/", handler)

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 创建上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 设置 STUN 服务器
	go func() {
		stunAddress := fmt.Sprintf("%s:%d", cfg.Host, cfg.StunPort)
		if err := stunServer.StartSTUNServer(ctx, stunAddress); err != nil {
			log.Printf("Failed to start STUN server: %v", err)
		}
	}()

	// 创建 http.Server 实例
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器
	startServer(server, io)

	// 在退出时取消上下文
	cancel()
}

func handleRoutes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
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
		case "/api/message":
			switch r.Method {
			case http.MethodPost:
				handlers.CreateMessage(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			if strings.HasPrefix(r.URL.Path, "/api/users/") {
				id := r.URL.Path[len("/api/users/"):]
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

func startServer(server *http.Server, io *socket.Server) {
	log.Printf("Server running on %s\n", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	exit := make(chan struct{})
	SignalC := make(chan os.Signal, 1) // 添加缓冲区

	signal.Notify(SignalC, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for s := range SignalC {
			switch s {
			case os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				close(exit)
				return
			}
		}
	}()

	<-exit
	io.Close(nil)
	server.Shutdown(context.Background())
	os.Exit(0)
}
