package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/handlers"
	"github.com/Ireoo/sixin-server/middleware"
	"github.com/Ireoo/sixin-server/socketio"
	"github.com/Ireoo/sixin-server/stun"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bodySize   int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.bodySize += size
	return size, err
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery
		clientIP := r.RemoteAddr
		method := r.Method

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		latency := time.Since(start)
		statusCode := lrw.statusCode

		log.Printf("| %3d | %13v | %15s | %s  %s\n%s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
			raw,
		)
	})
}

func SetupAndRun(cfg *config.Config) {
	// 创建 base.Base 实例
	baseInstance := &base.Base{}

	// 初始化数据库
	err := database.InitDatabase(database.DatabaseType(cfg.DBType), cfg.DBConn)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db := database.GetCurrentDB()
	defer db.Close()

	// 设置 Socket.IO 事件处理
	io := socketio.SetupSocketHandlers(db.GetDB(), baseInstance)

	// 创建路由器
	router := mux.NewRouter()

	// 注册路由和处理器
	router.HandleFunc("/api/ping", handlers.Ping).Methods("GET")
	router.HandleFunc("/api/users", handlers.GetUsers).Methods("GET")
	router.HandleFunc("/api/users", handlers.CreateUser).Methods("POST")
	router.HandleFunc("/api/users/{id}", handlers.GetUser).Methods("GET")
	router.HandleFunc("/api/users/{id}", handlers.UpdateUser).Methods("PUT")
	router.HandleFunc("/api/users/{id}", handlers.DeleteUser).Methods("DELETE")

	// 静态文件服务
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 设置中间件
	router.Use(middleware.Logger)
	router.Use(middleware.CORS)
	router.Use(loggerMiddleware)

	// 设置 Socket.IO
	router.Handle("/socket.io/", io.ServeHandler(nil))

	// 创建 http.Server 实例
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: router,
	}

	// 启动服务器，将 cfg 传递进去
	startServer(server, cfg)
}

func startServer(server *http.Server, cfg *config.Config) {
	// 创建一个可取消的上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 确保在退出时取消上下文

	// 设置 STUN 服务器
	go func() {
		stunAddress := fmt.Sprintf("%s:%d", cfg.Host, cfg.StunPort)
		if err := stunServer.StartSTUNServer(ctx, stunAddress); err != nil {
			log.Printf("Failed to start STUN server: %v", err)
		}
	}()

	// 在协程中启动 HTTP 服务器
	go func() {
		log.Printf("Server running on %s...\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 取消上下文，通知 STUN 服务器关闭
	cancel()

	// 创建一个带超时的上下文用于关闭 HTTP 服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}
