package http

import (
	"encoding/json"
	"fmt"
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
		case "/api/room-members":
			hm.handleRoomMembers(w, r)
		case "/api/room-privacy":
			hm.handleSetRoomPrivacy(w, r)
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

// 新增一个辅助函数来发送统一格式的 JSON 响应
func sendJSONResponse(w http.ResponseWriter, data interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"success": err == nil,
		"data":    data,
	}

	if err != nil {
		response["error"] = err.Error()
	}

	json.NewEncoder(w).Encode(response)
}

func (hm *HTTPManager) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := hm.dbManager.GetAllUsers()
		sendJSONResponse(w, users, err)
	case http.MethodPost:
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
			return
		}
		err := hm.dbManager.CreateUser(&user)
		sendJSONResponse(w, user, err)
	default:
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
	}
}

func (hm *HTTPManager) handleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rooms, err := hm.dbManager.GetAllRooms()
		sendJSONResponse(w, rooms, err)
	case http.MethodPost:
		var room models.Room
		if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
			sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
			return
		}
		err := hm.dbManager.CreateRoom(&room)
		sendJSONResponse(w, room, err)
	default:
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
	}
}

func (hm *HTTPManager) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
		return
	}

	var message models.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
		return
	}

	err := hm.dbManager.CreateMessage(&message)
	if err != nil {
		sendJSONResponse(w, map[string]string{"message": "保存消息失败"}, err)
		return
	}

	fullMessage, err := hm.dbManager.GetFullMessage(message.ID)
	if err != nil {
		sendJSONResponse(w, map[string]string{"message": "加载完整消息数据失败"}, err)
		return
	}

	var recipientID uint
	if message.ListenerID != 0 {
		recipientID = message.ListenerID
	} else {
		recipientID = message.RoomID
	}

	hm.baseInstance.SendMessageToUsers(fullMessage, message.TalkerID, recipientID)

	sendJSONResponse(w, fullMessage, nil)
}

func (hm *HTTPManager) handleUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/users/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的用户ID"}, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := hm.dbManager.GetUserByID(uint(id))
		sendJSONResponse(w, user, err)
	case http.MethodPut:
		var updatedUser models.User
		if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
			sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
			return
		}
		err := hm.dbManager.UpdateUser(uint(id), updatedUser)
		sendJSONResponse(w, updatedUser, err)
	case http.MethodDelete:
		err := hm.dbManager.DeleteUser(uint(id))
		sendJSONResponse(w, map[string]string{"message": "用户删除成功"}, err)
	default:
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
	}
}

func (hm *HTTPManager) handleRoomByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/rooms/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的房间ID"}, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		room, err := hm.dbManager.GetRoomByID(uint(id))
		sendJSONResponse(w, room, err)
	case http.MethodPut:
		var updatedRoom models.Room
		if err := json.NewDecoder(r.Body).Decode(&updatedRoom); err != nil {
			sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
			return
		}
		err := hm.dbManager.UpdateRoom(uint(id), updatedRoom)
		sendJSONResponse(w, updatedRoom, err)
	case http.MethodDelete:
		err := hm.dbManager.DeleteRoom(uint(id))
		sendJSONResponse(w, map[string]string{"message": "房间删除成功"}, err)
	default:
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
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

func (hm *HTTPManager) handleRoomMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		hm.handleAddUserToRoom(w, r)
	case http.MethodDelete:
		hm.handleRemoveUserFromRoom(w, r)
	case http.MethodPut:
		hm.handleUpdateRoomMember(w, r)
	default:
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
	}
}

func (hm *HTTPManager) handleAddUserToRoom(w http.ResponseWriter, r *http.Request) {
	var roomRequest struct {
		UserID    uint   `json:"user_id"`
		RoomID    uint   `json:"room_id"`
		Alias     string `json:"alias"`
		IsPrivate bool   `json:"is_private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&roomRequest); err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
		return
	}

	err := hm.dbManager.AddUserToRoom(roomRequest.UserID, roomRequest.RoomID, roomRequest.Alias, roomRequest.IsPrivate)
	sendJSONResponse(w, map[string]string{"message": "用户成功添加到房间"}, err)
}

func (hm *HTTPManager) handleRemoveUserFromRoom(w http.ResponseWriter, r *http.Request) {
	var roomRequest struct {
		UserID uint `json:"user_id"`
		RoomID uint `json:"room_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&roomRequest); err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
		return
	}

	err := hm.dbManager.RemoveUserFromRoom(roomRequest.UserID, roomRequest.RoomID)
	sendJSONResponse(w, map[string]string{"message": "用户从房间中删除成功"}, err)
}

func (hm *HTTPManager) handleUpdateRoomMember(w http.ResponseWriter, r *http.Request) {
	var roomRequest struct {
		UserID    uint   `json:"user_id"`
		RoomID    uint   `json:"room_id"`
		Alias     string `json:"alias"`
		IsPrivate bool   `json:"is_private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&roomRequest); err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
		return
	}

	err := hm.dbManager.UpdateRoomMemberAlias(fmt.Sprint(roomRequest.UserID), fmt.Sprint(roomRequest.RoomID), roomRequest.Alias)
	if err != nil {
		sendJSONResponse(w, map[string]string{"message": "更新房间成员别名失败"}, err)
		return
	}

	err = hm.dbManager.SetRoomMemberPrivacy(fmt.Sprint(roomRequest.UserID), fmt.Sprint(roomRequest.RoomID), roomRequest.IsPrivate)
	sendJSONResponse(w, map[string]string{"message": "房间成员信息更新成功"}, err)
}

func (hm *HTTPManager) handleSetRoomPrivacy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendJSONResponse(w, map[string]string{"message": "方法不允许"}, fmt.Errorf("方法不允许"))
		return
	}

	var roomRequest struct {
		UserID    uint `json:"user_id"`
		RoomID    uint `json:"room_id"`
		IsPrivate bool `json:"is_private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&roomRequest); err != nil {
		sendJSONResponse(w, map[string]string{"message": "无效的请求数据"}, err)
		return
	}

	err := hm.dbManager.SetRoomPrivacy(roomRequest.UserID, roomRequest.RoomID, roomRequest.IsPrivate)
	sendJSONResponse(w, map[string]string{"message": "房间隐私设置更新成功"}, err)
}
