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
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
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
		case "/api/login":
			hm.handleLogin(w, r)
		case "/api/register":
			hm.handleRegister(w, r)
		default:
			// 对其他所有路由应用身份验证中间件
			middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
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
			})).ServeHTTP(w, r)
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

	// 添加用户注册路由
	http.HandleFunc("/register", httpManager.handleRegister)
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

func (hm *HTTPManager) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, nil, fmt.Errorf("方法不允许"))
		return
	}

	var userData *models.User

	if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
		sendJSONResponse(w, nil, fmt.Errorf("无效的请求数据"))
		return
	}

	// 验证必填字段
	if userData.Username == "" || userData.Password == "" || userData.Email == "" || userData.WechatID == "" {
		sendJSONResponse(w, nil, fmt.Errorf("用户名、密码、邮箱和微信ID为必填项"))
		return
	}

	// 检查用户名是否已存在
	existingUser, _ := hm.baseInstance.DbManager.GetUserByUsername(userData.Username)
	if existingUser != nil {
		sendJSONResponse(w, nil, fmt.Errorf("用户名已存在"))
		return
	}

	// 检查邮箱是否已存在
	existingUser, _ = hm.baseInstance.DbManager.GetUserByEmail(userData.Email)
	if existingUser != nil {
		sendJSONResponse(w, nil, fmt.Errorf("邮箱已被使用"))
		return
	}

	// 检查微信ID是否已存在
	existingUser, _ = hm.baseInstance.DbManager.GetUserByWechatID(userData.WechatID)
	if existingUser != nil {
		sendJSONResponse(w, nil, fmt.Errorf("微信ID已被使用"))
		return
	}

	// 创建新用户
	newUser := &models.User{
		Username: userData.Username,
		Email:    userData.Email,
		WechatID: userData.WechatID,
		Name:     userData.Name,
		Phone:    userData.Phone,
	}

	// 生成密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSONResponse(w, nil, fmt.Errorf("密码加密失败"))
		return
	}
	newUser.Password = string(hashedPassword)

	// 创建用户
	err = hm.baseInstance.DbManager.CreateUser(newUser)
	if err != nil {
		sendJSONResponse(w, nil, fmt.Errorf("创建用户失败: %v", err))
		return
	}

	// 移除敏感信息
	newUser.Password = ""
	newUser.SecretKey = ""

	sendJSONResponse(w, map[string]interface{}{
		"message": "用户注册成功",
		"user":    newUser,
	}, nil)
}

func (hm *HTTPManager) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, nil, fmt.Errorf("方法不允许"))
		return
	}

	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		sendJSONResponse(w, nil, fmt.Errorf("无效的请求数据"))
		return
	}

	user, err := hm.baseInstance.DbManager.AuthenticateUser(loginData.Username, loginData.Password)
	if err != nil {
		sendJSONResponse(w, nil, err)
		return
	}

	// 创建 JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	// 使用密钥签名 token（这里使用一个示例密钥，实际应用中应该使用更安全的方式存储和管理密钥）
	tokenString, err := token.SignedString([]byte("your_secret_key"))
	if err != nil {
		sendJSONResponse(w, nil, fmt.Errorf("生成 token 失败"))
		return
	}

	sendJSONResponse(w, map[string]string{"token": tokenString}, nil)
}
