package socketio

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"log"
	"sync"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/internal/middleware"
	"github.com/Ireoo/sixin-server/logger"
	"github.com/pion/webrtc/v3"
	"github.com/zishang520/socket.io/v2/socket"
)

type socketData struct {
	data sync.Map // 使用 sync.Map 来提高并发性能并减少锁竞争
}

func newSocketData() *socketData {
	return &socketData{}
}

type userSocketMap struct {
	data sync.Map // 使用 sync.Map 来降低锁的开销并提升并发性能
}

func newUserSocketMap() *userSocketMap {
	return &userSocketMap{}
}

type SocketIOManager struct {
	Io              *socket.Server
	baseInstance    *base.Base
	peerConnections map[string]*webrtc.PeerConnection
	pcMutex         sync.RWMutex

	userSocketMap *userSocketMap
	socketData    *socketData
	cache         *cache.Cache
}

func NewSocketIOManager(baseInst *base.Base) *SocketIOManager {
	return &SocketIOManager{
		Io:              socket.NewServer(nil, nil),
		baseInstance:    baseInst,
		peerConnections: make(map[string]*webrtc.PeerConnection),
		userSocketMap:   newUserSocketMap(),
		socketData:      newSocketData(),
		cache:           cache.New(5*time.Minute, 10*time.Minute), // 初始化缓存，设置默认过期时间为 5 分钟
	}
}

func (sim *SocketIOManager) authMiddleware(next func(*socket.Socket, ...any)) func(*socket.Socket, ...any) {
	return func(s *socket.Socket, args ...any) {
		token, _ := s.Request().Query().Get("token")
		if token == "" {
			emitError(s, "未提供身份验证令牌", nil)
			s.Disconnect(true)
			return
		}

		userID, err := middleware.ValidateToken(token)
		if err != nil {
			emitError(s, "无效的身份验证令牌", err)
			s.Disconnect(true)
			return
		}

		sim.socketData.data.Store(s, map[string]interface{}{"userID": userID})
		sim.userSocketMap.data.Store(userID, s)

		next(s, args...)
	}
}

func (sim *SocketIOManager) SetupSocketHandlers() *socket.Server {
	sim.Io.Use(func(s *socket.Socket, next func(*socket.ExtendedError)) {
		sim.authMiddleware(func(s *socket.Socket, _ ...any) {
			next(nil)
		})(s)
	})
	sim.Io.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		sim.handleConnection(client)
	})
	sim.Io.On("connection_timeout", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		log.Printf("连接超时: %s", client.Id())
		client.Disconnect(false)
	})
	return sim.Io
}

func (sim *SocketIOManager) handleConnection(clients ...any) {
	client := clients[0].(*socket.Socket)
	go func(client *socket.Socket) { // 使用 goroutine 处理新连接，避免单线程性能瓶颈
		logger.Info(fmt.Sprintf("新连接：%s", client.Id()))

		sim.emitInitialState(client)
		sim.registerClientHandlers(client)

		client.On("disconnecting", func(reason ...any) {
			logger.Info(fmt.Sprintf("连接断开: %s, 原因: %v", client.Id(), reason))
			sim.cleanupPeerConnection(client.Id())
		})

		// 添加连接超时检测
		time.AfterFunc(30*time.Minute, func() {
			if client.Connected() {
				log.Printf("连接超时未活动，断开连接: %s", client.Id())
				client.Disconnect(true)
			}
		})
	}(client)
}

func (sim *SocketIOManager) emitInitialState(client *socket.Socket) {
	client.Emit("receive", sim.baseInstance.ReceiveDevice)
	client.Emit("email", sim.baseInstance.EmailNote)
	client.Emit("self", sim.baseInstance.Self)
	client.Emit("qrcode", sim.baseInstance.Qrcode)
}

func (sim *SocketIOManager) registerClientHandlers(client *socket.Socket) {
	events := map[string]func(*socket.Socket, ...any){
		"self":               sim.handleSelf,
		"receive":            sim.handleReceive,
		"message":            sim.handleMessage,
		"email":              sim.handleEmail,
		"getChats":           sim.handleGetChats,
		"getRooms":           sim.handleGetRooms,
		"getUsers":           sim.handleGetUsers,
		"getRoomByUsers":     sim.handleGetRoomByUsers,
		"offer":              sim.handleOffer,
		"answer":             sim.handleAnswer,
		"ice-candidate":      sim.handleIceCandidate,
		"updateUser":         sim.handleUpdateUser,
		"deleteUser":         sim.handleDeleteUser,
		"createRoom":         sim.handleCreateRoom,
		"updateRoom":         sim.handleUpdateRoom,
		"deleteRoom":         sim.handleDeleteRoom,
		"addFriend":          sim.handleAddFriend,
		"removeFriend":       sim.handleRemoveFriend,
		"updateFriendAlias":  sim.handleUpdateFriendAlias,
		"setFriendPrivacy":   sim.handleSetFriendPrivacy,
		"addUserToRoom":      sim.handleAddUserToRoom,
		"removeUserFromRoom": sim.handleRemoveUserFromRoom,
		"updateRoomAlias":    sim.handleUpdateRoomAlias,
		"setRoomPrivacy":     sim.handleSetRoomPrivacy,
	}

	for event, handler := range events {
		h := handler
		client.On(event, func(args ...any) {
			h(client, args...)
		})
	}
}

func (sim *SocketIOManager) handleSelf(client *socket.Socket, args ...any) {
	// 获取当前用户ID
	userID, err := sim.getUserIDFromSocket(client)
	if err != nil {
		emitError(client, "无法获取用户信息", err)
		return
	}

	// 尝试从缓存中获取用户信息
	if cachedUserInfo, found := sim.cache.Get(fmt.Sprintf("userInfo_%d", userID)); found {
		client.Emit("self", cachedUserInfo)
		return
	}

	// 从数据库或缓存中获取用户详细信息
	userInfo, err := sim.baseInstance.DbManager.GetUserInfo(userID)
	if err != nil {
		emitError(client, "获取用户信息失败", err)
		return
	}

	// 缓存用户信息以减少数据库查询
	sim.cache.Set(fmt.Sprintf("userInfo_%d", userID), userInfo, cache.DefaultExpiration)

	// 返回用户信息
	client.Emit("self", userInfo)
}

func (sim *SocketIOManager) handleReceive(client *socket.Socket, args ...any) {
	sim.baseInstance.ReceiveDevice = !sim.baseInstance.ReceiveDevice
	message := "wechat:receive"
	if !sim.baseInstance.ReceiveDevice {
		message = "wechat:message"
	}
	sim.baseInstance.SendMessage(message)
	client.Emit("receive", sim.baseInstance.ReceiveDevice)
}

func (sim *SocketIOManager) handleEmail(client *socket.Socket, args ...any) {
	sim.baseInstance.EmailNote = !sim.baseInstance.EmailNote
	client.Emit("email", sim.baseInstance.EmailNote)
}

func (sim *SocketIOManager) SendMessageToUsers(message interface{}, userIDs ...uint) {
	var wg sync.WaitGroup
	messageQueue := make(chan uint, len(userIDs))

	// 将所有用户 ID 添加到消息队列中
	for _, userID := range userIDs {
		messageQueue <- userID
	}
	close(messageQueue)

	// 使用多个 goroutine 并发处理消息发送
	workerCount := 10 // 可以根据实际情况调整并发数量
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range messageQueue {
				clientInterface, exists := sim.userSocketMap.data.Load(userID)
				if !exists {
					log.Printf("用户 %d 未找到对应的客户端", userID)
					continue
				}

				client := clientInterface.(*socket.Socket)
				err := client.Emit("message", message)
				if err != nil {
					log.Printf("发送消息给用户 %d 失败: %v", userID, err)
				}
			}
		}()
	}

	// 等待所有 goroutine 完成
	wg.Wait()
}

// 提取通用的从 socketData 获取 userID 的逻辑
func (sim *SocketIOManager) getUserIDFromSocket(client *socket.Socket) (uint, error) {
	value, ok := sim.socketData.data.Load(client)
	if !ok {
		return 0, fmt.Errorf("未找到用户数据")
	}

	userID, ok := value.(map[string]interface{})["userID"].(uint)
	if !ok {
		return 0, fmt.Errorf("userID 类型转换失败")
	}
	return userID, nil
}
