package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/internal/handlers"
	"github.com/Ireoo/sixin-server/internal/middleware"
	"github.com/Ireoo/sixin-server/models"
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
	userHandler    *base.UserHandler
	roomHandler    *base.RoomHandler
	messageHandler *base.MessageHandler // Added messageHandler variable
)

// SetHandlers 设置处理器
func SetHandlers(uh *base.UserHandler, rh *base.RoomHandler, mh *base.MessageHandler) {
	userHandler = uh
	roomHandler = rh
	messageHandler = mh
}

func HandleRoutes(b *base.Base) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			handlers.Ping(w, r)
		case "/api/users":
			switch r.Method {
			case http.MethodGet:
				users, err := userHandler.GetUsers()
				if err != nil {
					http.Error(w, "获取用户列表失败", http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(users)
			case http.MethodPost:
				var user models.User

				// 从请求体解析用户数据到user
				err := userHandler.CreateUser(&user)
				if err != nil {
					http.Error(w, "创建用户失败", http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(user)
				// 将创建的用户转换为JSON并写入响应

			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "/api/rooms":
			switch r.Method {
			case http.MethodGet:
				rooms, err := roomHandler.GetRooms()
				if err != nil {
					http.Error(w, "获取房间列表失败", http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(rooms)
			case http.MethodPost:
				var room models.Room
				if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
					http.Error(w, "无效的请求数据", http.StatusBadRequest)
					return
				}
				if err := roomHandler.CreateRoom(&room); err != nil {
					http.Error(w, "创建房间失败", http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(room)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "/api/message":
			switch r.Method {
			case http.MethodPost:
				var messageData []byte
				if err := json.NewDecoder(r.Body).Decode(&messageData); err != nil {
					http.Error(w, "无效的请求数据", http.StatusBadRequest)
					return
				}
				message, err := messageHandler.HandleMessage(messageData)
				if err != nil {
					http.Error(w, "处理消息失败: "+err.Error(), http.StatusInternalServerError)
					return
				}

				var recipientID uint
				if message.ListenerID != 0 {
					recipientID = message.ListenerID
				} else {
					recipientID = message.RoomID
				}

				sendData := struct {
					Talker   *models.User    `json:"talker"`
					Listener *models.User    `json:"listener,omitempty"`
					Room     *models.Room    `json:"room,omitempty"`
					Message  *models.Message `json:"message"`
				}{
					Talker:   message.Talker,
					Listener: message.Listener,
					Room:     message.Room,
					Message:  message,
				}

				b.SendMessageToUsers(sendData, message.TalkerID, recipientID)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "success",
					"message": "消息已创建并发送",
					"data":    sendData,
				})
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			if strings.HasPrefix(r.URL.Path, "/api/users/") {
				id := r.URL.Path[len("/api/users/"):]
				switch r.Method {
				case http.MethodGet:
					user, err := userHandler.GetUser(id)
					if err != nil {
						http.Error(w, "获取用户失败", http.StatusNotFound)
						return
					}
					// 将user转换为JSON并写入响应
					json.NewEncoder(w).Encode(user)
				case http.MethodPut:
					var updatedUser models.User
					// 从请求体解析更新的用户数据到updatedUser
					err := userHandler.UpdateUser(id, &updatedUser)
					if err != nil {
						http.Error(w, "更新用户失败", http.StatusInternalServerError)
						return
					}
					// 将更新后的用户转换为JSON并写入响应
				case http.MethodDelete:
					err := userHandler.DeleteUser(id)
					if err != nil {
						http.Error(w, "删除用户失败", http.StatusInternalServerError)
						return
					}
					// 返回成功删除的响应
				default:
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

				}
			} else if strings.HasPrefix(r.URL.Path, "/api/rooms/") {
				id := r.URL.Path[len("/api/rooms/"):]
				switch r.Method {
				case http.MethodGet:
					room, err := roomHandler.GetRoom(id)
					if err != nil {
						http.Error(w, "获取房间失败", http.StatusNotFound)
						return
					}
					json.NewEncoder(w).Encode(room)
				case http.MethodPut:
					var updatedRoom models.Room
					if err := json.NewDecoder(r.Body).Decode(&updatedRoom); err != nil {
						http.Error(w, "无效的请求数据", http.StatusBadRequest)
						return
					}
					if err := roomHandler.UpdateRoom(id, &updatedRoom); err != nil {
						http.Error(w, "更新房间失败", http.StatusInternalServerError)
						return
					}
					json.NewEncoder(w).Encode(updatedRoom)
				case http.MethodDelete:
					if err := roomHandler.DeleteRoom(id); err != nil {
						http.Error(w, "删除房间失败", http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				default:
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

				}
			} else {
				http.NotFound(w, r)
			}
		}
	}
}

func SetupHTTPHandlers(b *base.Base) {
	// SetHandlers(userHandler, roomHandler, messageHandler)

	// 设置中间件和路由

	handler := ChainMiddlewares(
		HandleRoutes(b),
		LoggerMiddleware,
		middleware.CORS,
	)
	http.HandleFunc("/", handler)

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}
