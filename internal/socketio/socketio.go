package socketio

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/models"

	"github.com/pion/webrtc/v3"
	"github.com/zishang520/socket.io/v2/socket"
)

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

func (sim *SocketIOManager) SetupSocketHandlers() *socket.Server {
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
		"self":           sim.handleSelf,
		"receive":        sim.handleReceive,
		"message":        sim.handleMessage,
		"email":          sim.handleEmail,
		"revokemsg":      sim.handleRevokeMsg,
		"getChats":       sim.handleGetChats,
		"getRooms":       sim.handleGetRooms,
		"getUsers":       sim.handleGetUsers,
		"getRoomByUsers": sim.handleGetRoomByUsers,
		"offer":          sim.handleOffer,
		"answer":         sim.handleAnswer,
		"ice-candidate":  sim.handleIceCandidate,
		"createUser":     sim.handleCreateUser,
		"updateUser":     sim.handleUpdateUser,
		"deleteUser":     sim.handleDeleteUser,
		"createRoom":     sim.handleCreateRoom,
		"updateRoom":     sim.handleUpdateRoom,
		"deleteRoom":     sim.handleDeleteRoom,
	}

	for event, handler := range events {
		client.On(event, func(args ...any) {
			handler(client, args...)
		})
	}
}

func (sim *SocketIOManager) cleanupPeerConnection(clientID socket.SocketId) {
	sim.pcMutex.Lock()
	defer sim.pcMutex.Unlock()

	id := string(clientID)
	if peerConnection, exists := sim.peerConnections[id]; exists {
		if err := peerConnection.Close(); err != nil {
			log.Printf("关闭 PeerConnection 失败: %v", err)
		}
		delete(sim.peerConnections, id)
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

func (sim *SocketIOManager) handleRevokeMsg(client *socket.Socket, args ...any) {
	// 实现撤回消息的逻辑
	// TODO: 添加具体实现
}

func (sim *SocketIOManager) handleGetChats(client *socket.Socket, args ...any) {
	messages, err := sim.baseInstance.DbManager.GetChats()
	if err != nil {
		client.Emit("error", err.Error())

		return
	}
	client.Emit("getChats", messages)
}

func (sim *SocketIOManager) handleGetRooms(client *socket.Socket, args ...any) {
	rooms, err := sim.baseInstance.DbManager.GetAllRooms()
	if err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getRooms", rooms)
}

func (sim *SocketIOManager) handleGetUsers(client *socket.Socket, args ...any) {
	users, err := sim.baseInstance.DbManager.GetAllUsers()
	if err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getUsers", users)
}

func (sim *SocketIOManager) handleGetRoomByUsers(client *socket.Socket, args ...any) {
	// 实现获取用户房间的逻辑
	// TODO: 添加具体实现
}

func (sim *SocketIOManager) handleMessage(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少消息内容")
		return
	}

	var msgBytes []byte

	switch arg := args[0].(type) {
	case string:
		msgBytes = []byte(arg)
	case []byte:
		msgBytes = arg
	default:
		client.Emit("error", "消息格式错误")
		return
	}

	fmt.Println("收到消息：", string(msgBytes))

	var message models.Message
	if err := json.Unmarshal(msgBytes, &message); err != nil {
		client.Emit("error", "解析消息失败")
		return
	}

	if err := sim.baseInstance.DbManager.CreateMessage(&message); err != nil {
		client.Emit("error", "保存消息失败")
		return
	}

	fullMessage, err := sim.baseInstance.DbManager.GetFullMessage(message.ID)
	if err != nil {
		client.Emit("error", "加载完整消息数据失败")
		return
	}

	var recipientID uint
	if message.ListenerID != 0 {
		recipientID = message.ListenerID
	} else {
		recipientID = message.RoomID
	}

	sim.SendMessageToUsers(fullMessage, message.TalkerID, recipientID)
}

func (sim *SocketIOManager) handleOffer(client *socket.Socket, sdp ...any) {
	if len(sdp) == 0 {
		client.Emit("error", "缺少 SDP 数据")
		return
	}

	sdpStr, ok := sdp[0].(string)
	if !ok {
		client.Emit("error", "SDP 不是字符串类型")
		return
	}

	fmt.Printf("收到 SDP Offer: %s\n", sdpStr)

	offer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(sdpStr), &offer); err != nil {
		fmt.Printf("解析 Offer SDP 失败: %v\n", err)
		client.Emit("error", "Offer SDP 无效")
		return
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Printf("创建 PeerConnection 失败: %v", err)
		client.Emit("error", "创建 PeerConnection 失败")
		return
	}

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		fmt.Printf("设置远端描述失败: %v\n", err)
		client.Emit("error", "设置远端描述失败")
		return
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		fmt.Printf("创建 SDP Answer 失败: %v\n", err)
		client.Emit("error", "创建 Answer 失败")
		return
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		fmt.Printf("设置本地描述失败: %v\n", err)
		client.Emit("error", "设置本地描述失败")
		return
	}

	answerJSON, err := json.Marshal(answer)
	if err != nil {
		fmt.Printf("序列化 SDP Answer 失败: %v\n", err)
		client.Emit("error", "序列化 Answer 失败")
		return
	}

	sim.pcMutex.Lock()
	sim.peerConnections[string(client.Id())] = peerConnection
	sim.pcMutex.Unlock()

	client.Emit("answer", string(answerJSON))
}

func (sim *SocketIOManager) handleAnswer(client *socket.Socket, sdp ...any) {
	if len(sdp) == 0 {
		client.Emit("error", "缺少 SDP 数据")
		return
	}

	sdpStr, ok := sdp[0].(string)
	if !ok {
		client.Emit("error", "SDP 不是字符串类型")
		return
	}

	fmt.Printf("收到 SDP Answer: %s\n", sdpStr)

	answer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(sdpStr), &answer); err != nil {
		log.Printf("解析 Answer SDP 失败: %v", err)
		client.Emit("error", "Answer SDP 无效")
		return
	}

	sim.pcMutex.RLock()
	peerConnection, exists := sim.peerConnections[string(client.Id())]
	sim.pcMutex.RUnlock()
	if !exists {
		log.Printf("PeerConnection 未找到")
		client.Emit("error", "PeerConnection 未找到")
		return
	}

	if err := peerConnection.SetRemoteDescription(answer); err != nil {
		log.Printf("设置远端描述失败: %v", err)
		client.Emit("error", "设置远端描述失败")
		return
	}
}

func (sim *SocketIOManager) handleIceCandidate(client *socket.Socket, candidate ...any) {
	if len(candidate) == 0 {
		client.Emit("error", "缺少 ICE 候选数据")
		return
	}

	candidateStr, ok := candidate[0].(string)
	if !ok {
		client.Emit("error", "ICE 候选不是字符串类型")
		return
	}
	fmt.Printf("收到 ICE 候选: %s\n", candidateStr)

	iceCandidate := webrtc.ICECandidateInit{}
	if err := json.Unmarshal([]byte(candidateStr), &iceCandidate); err != nil {
		log.Printf("解析 ICE 候选失败: %v", err)
		client.Emit("error", "ICE 候选解析失败")
		return
	}

	sim.pcMutex.RLock()
	peerConnection, exists := sim.peerConnections[string(client.Id())]
	sim.pcMutex.RUnlock()
	if !exists {
		log.Printf("PeerConnection 未找到，无法添加 ICE 候选")
		client.Emit("error", "PeerConnection 未找到")
		return
	}

	if err := peerConnection.AddICECandidate(iceCandidate); err != nil {
		log.Printf("添加 ICE 候选失败: %v", err)
		client.Emit("error", "添加 ICE 候选失败")
	}
}

func (sim *SocketIOManager) SendMessageToUsers(message interface{}, userIDs ...uint) {
	for _, userID := range userIDs {
		socketID := socket.SocketId(fmt.Sprintf("%d", userID))
		clients := sim.Io.Sockets().Sockets()
		clients.Range(func(id socket.SocketId, client *socket.Socket) bool {
			if client.Id() == socketID {
				err := client.Emit("message", message)
				if err != nil {
					log.Printf("发送消息给用户 %d 失败: %v", userID, err)
				}
				return false
			}
			return true
		})
	}
}

// func (sim *SocketIOManager) GetSocketIOServer() *socket.Server {
// 	return sim.io
// }

func (sim *SocketIOManager) handleCreateUser(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少用户数据")
		return
	}

	var user models.User
	if err := json.Unmarshal([]byte(args[0].(string)), &user); err != nil {
		client.Emit("error", "无效的用户数据")
		return
	}

	if err := sim.baseInstance.DbManager.CreateUser(&user); err != nil {
		client.Emit("error", "创建用户失败")
		return
	}

	client.Emit("userCreated", user)
}

func (sim *SocketIOManager) handleUpdateUser(client *socket.Socket, args ...any) {
	if len(args) < 2 {
		client.Emit("error", "缺少用户ID或更新数据")
		return
	}

	userID := args[0].(string)
	var updatedUser models.User
	if err := json.Unmarshal([]byte(args[1].(string)), &updatedUser); err != nil {
		client.Emit("error", "无效的用户数据")
		return
	}

	userIDUint, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		client.Emit("error", "无效的用户ID")
		return
	}

	if err := sim.baseInstance.DbManager.UpdateUser(uint(userIDUint), updatedUser); err != nil {
		client.Emit("error", "更新用户失败")
		return
	}

	client.Emit("userUpdated", updatedUser)
}

func (sim *SocketIOManager) handleDeleteUser(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少用户ID")
		return
	}
	userID, err := strconv.ParseUint(args[0].(string), 10, 64)
	if err != nil {
		client.Emit("error", "无效的用户ID")
		return
	}
	if err := sim.baseInstance.DbManager.DeleteUser(uint(userID)); err != nil {
		client.Emit("error", "删除用户失败")
		return
	}

	client.Emit("userDeleted", userID)
}

func (sim *SocketIOManager) handleCreateRoom(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少房间数据")
		return
	}

	var room models.Room
	if err := json.Unmarshal([]byte(args[0].(string)), &room); err != nil {
		client.Emit("error", "无效的房间数据")
		return
	}

	if err := sim.baseInstance.DbManager.CreateRoom(&room); err != nil {
		client.Emit("error", "创建房间失败")
		return
	}

	client.Emit("roomCreated", room)
}

func (sim *SocketIOManager) handleUpdateRoom(client *socket.Socket, args ...any) {
	if len(args) < 2 {
		client.Emit("error", "缺少房间ID或更新数据")
		return
	}

	roomID := args[0].(string)
	var updatedRoom models.Room
	if err := json.Unmarshal([]byte(args[1].(string)), &updatedRoom); err != nil {
		client.Emit("error", "无效的房间数据")
		return
	}

	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		client.Emit("error", "无效的房间ID")
		return
	}

	if err := sim.baseInstance.DbManager.UpdateRoom(uint(roomIDUint), updatedRoom); err != nil {
		client.Emit("error", "更新房间失败")
		return
	}

	client.Emit("roomUpdated", updatedRoom)
}

func (sim *SocketIOManager) handleDeleteRoom(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少房间ID")
		return
	}

	roomID := args[0].(string)
	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		client.Emit("error", "房间ID无效")
		return
	}
	if err := sim.baseInstance.DbManager.DeleteRoom(uint(roomIDUint)); err != nil {
		client.Emit("error", "删除房间失败")
		return
	}

	client.Emit("roomDeleted", roomID)
}
