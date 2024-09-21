package http

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Ireoo/sixin-server/handlers"
	"github.com/Ireoo/sixin-server/message" // Added import for message package
	"github.com/Ireoo/sixin-server/middleware"
	"github.com/Ireoo/sixin-server/room"
	"github.com/Ireoo/sixin-server/user"
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func LoggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
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

func ChainMiddlewares(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middlewares {
		handler = m(handler)
	}
	return handler
}

var (
	userHandler    *user.UserHandler
	roomHandler    *room.RoomHandler
	messageHandler *message.MessageHandler // Added messageHandler variable
)

// SetHandlers 设置处理器
func SetHandlers(uh *user.UserHandler, rh *room.RoomHandler) {
	userHandler = uh
	roomHandler = rh
}

func HandleRoutes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			handlers.Ping(w, r)
		case "/api/users":
			switch r.Method {
			case http.MethodGet:
				userHandler.GetUsers(w, r)
			case http.MethodPost:
				userHandler.CreateUser(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "/api/message":
			switch r.Method {
			case http.MethodPost:
				messageHandler.CreateMessage(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			if strings.HasPrefix(r.URL.Path, "/api/users/") {
				id := r.URL.Path[len("/api/users/"):]
				switch r.Method {
				case http.MethodGet:
					userHandler.GetUser(w, r, id)
				case http.MethodPut:
					userHandler.UpdateUser(w, r, id)
				case http.MethodDelete:
					userHandler.DeleteUser(w, r, id)
				default:
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			} else if strings.HasPrefix(r.URL.Path, "/api/rooms/") {
				id := r.URL.Path[len("/api/rooms/"):]
				switch r.Method {
				case http.MethodGet:
					roomHandler.GetRoom(w, r, id)
				case http.MethodPut:
					roomHandler.UpdateRoom(w, r, id)
				case http.MethodDelete:
					roomHandler.DeleteRoom(w, r, id)
				default:
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			} else {
				http.NotFound(w, r)
			}
		}
	}
}

func SetupHTTPHandlers(uh *user.UserHandler, rh *room.RoomHandler, mh *message.MessageHandler) {
	SetHandlers(uh, rh)
	messageHandler = mh

	// 设置中间件和路由
	handler := ChainMiddlewares(
		HandleRoutes(),
		LoggerMiddleware,
		middleware.CORS,
	)
	http.HandleFunc("/", handler)

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}
