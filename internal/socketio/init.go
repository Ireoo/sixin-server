package socketio

import (
	"fmt"
	"log"
	"sync"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/internal/middleware"

	"github.com/pion/webrtc/v3"
	"github.com/zishang520/socket.io/v2/socket"
)

var socketData = struct {
	sync.RWMutex
	data map[*socket.Socket]map[string]interface{}
}{data: make(map[*socket.Socket]map[string]interface{})}

var userSocketMap = struct {
	sync.RWMutex
	data map[uint]*socket.Socket
}{data: make(map[uint]*socket.Socket)}

type SocketIOManager struct {
	Io              *socket.Server
	baseInstance    *base.Base
	peerConnections map[string]*webrtc.PeerConnection
	pcMutex         sync.RWMutex
}

func NewSocketIOManager(baseInst *base.Base) *SocketIOManager {
	return &SocketIOManager{
		Io:              socket.NewServer(nil, nil),
		baseInstance:    baseInst,
		peerConnections: make(map[string]*webrtc.PeerConnection),
	}
}

func (sim *SocketIOManager) authMiddleware(next func(*socket.Socket, ...any)) func(*socket.Socket, ...any) {
	return func(s *socket.Socket, args ...any) {
		token, _ := s.Request().Query().Get("token")
		if token == "" {
			s.Emit("error", "未提供身份验证令牌")
			s.Disconnect(true)
			return
		}

		userID, err := middleware.ValidateToken(token)
		if err != nil {
			s.Emit("error", "无效的身份验证令牌")
			s.Disconnect(true)
			return
		}

		// // 初始化自定义属性
		socketData.Lock()
		socketData.data[s] = make(map[string]interface{})
		socketData.data[s]["userID"] = userID
		socketData.Unlock()

		userSocketMap.Lock()
		userSocketMap.data[userID] = s
		userSocketMap.Unlock()
		next(s, args...)
	}
}

func (sim *SocketIOManager) SetupSocketHandlers() *socket.Server {
	sim.Io.Use(func(s *socket.Socket, next func(*socket.ExtendedError)) {
		sim.authMiddleware(func(s *socket.Socket, _ ...any) {
			next(nil)
		})(s)
	})
	sim.Io.On("connection", sim.handleConnection)
	return sim.Io
}

func (sim *SocketIOManager) handleConnection(clients ...any) {
	client := clients[0].(*socket.Socket)
	fmt.Println("新连接：", client.Id())

	sim.emitInitialState(client)
	sim.registerClientHandlers(client)

	client.On("disconnecting", func(reason ...any) {
		fmt.Println("连接断开:", client.Id(), reason)
		sim.cleanupPeerConnection(client.Id())
	})
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
		client.On(event, func(args ...any) {
			handler(client, args...)
		})
	}
}

func (sim *SocketIOManager) handleSelf(client *socket.Socket, args ...any) {
	client.Emit("self", sim.baseInstance.Self)
}

func (sim *SocketIOManager) handleReceive(client *socket.Socket, args ...any) {
	sim.baseInstance.ReceiveDevice = !sim.baseInstance.ReceiveDevice
	message := "wechat:receive"
	if !sim.baseInstance.ReceiveDevice {
		message = "wechat:message"
	}
	sim.baseInstance.SendMessage(message, message)
	client.Emit("receive", sim.baseInstance.ReceiveDevice)
}

func (sim *SocketIOManager) handleEmail(client *socket.Socket, args ...any) {
	sim.baseInstance.EmailNote = !sim.baseInstance.EmailNote
	client.Emit("email", sim.baseInstance.EmailNote)
}

func (sim *SocketIOManager) SendMessageToUsers(message interface{}, userIDs ...uint) {
	for _, userID := range userIDs {
		userSocketMap.RLock()
		client, exists := userSocketMap.data[userID]
		userSocketMap.RUnlock()

		if !exists {
			log.Printf("用户 %d 未找到对应的客户端", userID)
			continue
		}

		err := client.Emit("message", message)
		if err != nil {
			log.Printf("发送消息给用户 %d 失败: %v", userID, err)
		}
	}
}

// 提取通用的从 socketData 获取 userID 的逻辑
func (sim *SocketIOManager) getUserIDFromSocket(client *socket.Socket) (uint, error) {
	socketData.RLock()
	defer socketData.RUnlock()

	userID, ok := socketData.data[client]["userID"].(uint)
	if !ok {
		return 0, fmt.Errorf("userID 类型转换失败")
	}
	return userID, nil
}