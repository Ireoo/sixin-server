package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/internal/handlers"
	"github.com/Ireoo/sixin-server/internal/middleware"
	"github.com/Ireoo/sixin-server/models"
)

type HTTPManager struct {
	dbManager    *database.DatabaseManager
	baseInstance *base.Base
}

func NewHTTPManager(baseInst *base.Base) *HTTPManager {
	return &HTTPManager{
		baseInstance: baseInst,
	}
}

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

func (hm *HTTPManager) HandleRoutes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			handlers.Ping(w, r)
		case "/api/users":
			hm.handleUsers(w, r)
		case "/api/rooms":
			hm.handleRooms(w, r)
		case "/api/message":
			hm.handleMessage(w, r)
		default:
			if strings.HasPrefix(r.URL.Path, "/api/users/") {
				hm.handleUserByID(w, r)
			} else if strings.HasPrefix(r.URL.Path, "/api/rooms/") {
				hm.handleRoomByID(w, r)
			} else {
				http.NotFound(w, r)
			}
		}
	}
}

func (hm *HTTPManager) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := hm.dbManager.GetAllUsers()
		if err != nil {
			http.Error(w, "获取用户列表失败", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(users)
	case http.MethodPost:
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "无效的请求数据", http.StatusBadRequest)
			return
		}
		if err := hm.dbManager.CreateUser(&user); err != nil {
			http.Error(w, "创建用户失败", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (hm *HTTPManager) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rooms, err := hm.dbManager.GetAllRooms()
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
		if err := hm.dbManager.CreateRoom(&room); err != nil {
			http.Error(w, "创建房间失败", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(room)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (hm *HTTPManager) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var message models.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	if err := hm.dbManager.CreateMessage(&message); err != nil {
		http.Error(w, "保存消息失败", http.StatusInternalServerError)
		return
	}

	fullMessage, err := hm.dbManager.GetFullMessage(message.ID)
	if err != nil {
		http.Error(w, "加载完整消息数据失败", http.StatusInternalServerError)
		return
	}

	var recipientID uint
	if message.ListenerID != 0 {
		recipientID = message.ListenerID
	} else {
		recipientID = message.RoomID
	}

	hm.baseInstance.SendMessageToUsers(fullMessage, message.TalkerID, recipientID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "消息已创建并发送",
		"data":    fullMessage,
	})
}

func (hm *HTTPManager) handleUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/users/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "无效的用户ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := hm.dbManager.GetUserByID(uint(id))
		if err != nil {
			http.Error(w, "获取用户失败", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(user)
	case http.MethodPut:
		var updatedUser models.User
		if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
			http.Error(w, "无效的请求数据", http.StatusBadRequest)
			return
		}
		if err := hm.dbManager.UpdateUser(uint(id), updatedUser); err != nil {
			http.Error(w, "更新用户失败", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(updatedUser)
	case http.MethodDelete:
		if err := hm.dbManager.DeleteUser(uint(id)); err != nil {
			http.Error(w, "删除用户失败", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (hm *HTTPManager) handleRoomByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/rooms/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "无效的房间ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		room, err := hm.dbManager.GetRoomByID(uint(id))
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
		if err := hm.dbManager.UpdateRoom(uint(id), updatedRoom); err != nil {
			http.Error(w, "更新房间失败", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(updatedRoom)
	case http.MethodDelete:
		if err := hm.dbManager.DeleteRoom(uint(id)); err != nil {
			http.Error(w, "删除房间失败", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func SetupHTTPHandlers(b *base.Base) {
	httpManager := NewHTTPManager(b)

	handler := ChainMiddlewares(
		httpManager.HandleRoutes(),
		LoggerMiddleware,
		middleware.CORS,
	)
	http.HandleFunc("/", handler)

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}
