package socketio

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/models"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

var (
	db           *gorm.DB
	baseInstance *base.Base
	io           *socket.Server
	// 用于存储客户端的 PeerConnection
	peerConnections = make(map[string]*webrtc.PeerConnection)
	pcMutex         sync.RWMutex
)

// SetupSocketHandlers 初始化 Socket.IO 服务器并设置事件处理器
func SetupSocketHandlers(database *gorm.DB, baseInst *base.Base) *socket.Server {
	db = database
	baseInstance = baseInst

	io = socket.NewServer(nil, nil)

	io.On("connection", handleConnection)

	return io
}

// handleConnection 处理新的客户端连接
func handleConnection(clients ...any) {
	client := clients[0].(*socket.Socket)
	fmt.Println("新连接：", client.Id())

	// 发送初始状态
	emitInitialState(client)

	// 注册事件处理器
	registerClientHandlers(client)

	client.On("disconnecting", func(reason ...any) {
		fmt.Println("连接断开:", client.Id(), reason)
		cleanupPeerConnection(client.Id())
	})
}

// emitInitialState 发送初始状态
func emitInitialState(client *socket.Socket) {
	client.Emit("receive", baseInstance.ReceiveDevice)
	client.Emit("email", baseInstance.EmailNote)
	client.Emit("self", baseInstance.Self)
	client.Emit("qrcode", baseInstance.Qrcode)
}

// registerClientHandlers 注册各种事件的处理器
func registerClientHandlers(client *socket.Socket) {
	events := map[string]func(*socket.Socket, ...any){
		"self":           handleSelf,
		"receive":        handleReceive,
		"message":        handleMessage,
		"email":          handleEmail,
		"revokemsg":      handleRevokeMsg,
		"getChats":       handleGetChats,
		"getRooms":       handleGetRooms,
		"getUsers":       handleGetUsers,
		"getRoomByUsers": handleGetRoomByUsers,
		"offer":          handleOffer,
		"answer":         handleAnswer,
		"ice-candidate":  handleIceCandidate,
	}

	for event, handler := range events {
		client.On(event, func(args ...any) {
			handler(client, args...)
		})
	}
}

// cleanupPeerConnection 关闭并移除 PeerConnection
func cleanupPeerConnection(clientID socket.SocketId) {
	pcMutex.Lock()
	defer pcMutex.Unlock()

	id := string(clientID)
	if peerConnection, exists := peerConnections[id]; exists {
		if err := peerConnection.Close(); err != nil {
			log.Printf("关闭 PeerConnection 失败: %v", err)
		}
		delete(peerConnections, id)
	}
}

// handleSelf 处理 "self" 事件
func handleSelf(client *socket.Socket, args ...any) {
	client.Emit("self", baseInstance.Self)
}

// handleReceive 处理 "receive" 事件
func handleReceive(client *socket.Socket, args ...any) {
	baseInstance.ReceiveDevice = !baseInstance.ReceiveDevice
	message := "wechat:receive"
	if !baseInstance.ReceiveDevice {
		message = "wechat:message"
	}
	baseInstance.SendMessage(message, message)
	client.Emit("receive", baseInstance.ReceiveDevice)
}

// handleEmail 处理 "email" 事件
func handleEmail(client *socket.Socket, args ...any) {
	baseInstance.EmailNote = !baseInstance.EmailNote
	client.Emit("email", baseInstance.EmailNote)
}

// handleRevokeMsg 处理 "revokemsg" 事件
func handleRevokeMsg(client *socket.Socket, args ...any) {
	// 实现撤回消息的逻辑
	// TODO: 添加具体实现
}

// handleGetChats 处理 "getChats" 事件
func handleGetChats(client *socket.Socket, args ...any) {
	var messages []models.Message
	if err := db.Preload("Talker").Preload("Listener").Preload("Room").
		Order("timestamp DESC").Limit(400).Find(&messages).Error; err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getChats", messages)
}

// handleGetRooms 处理 "getRooms" 事件
func handleGetRooms(client *socket.Socket, args ...any) {
	var rooms []models.Room
	if err := db.Preload("Owner").Preload("Members").
		Order("created_at DESC").Find(&rooms).Error; err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getRooms", rooms)
}

// handleGetUsers 处理 "getUsers" 事件
func handleGetUsers(client *socket.Socket, args ...any) {
	var users []models.User
	if err := db.Preload("Rooms").Order("created_at DESC").Find(&users).Error; err != nil {
		client.Emit("error", err.Error())
		return
	}
	client.Emit("getUsers", users)
}

// handleGetRoomByUsers 处理 "getRoomByUsers" 事件
func handleGetRoomByUsers(client *socket.Socket, args ...any) {
	// 实现获取用户房间的逻辑
	// TODO: 添加具体实现
}

// handleMessage 处理 "message" 事件
func handleMessage(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		client.Emit("error", "缺少消息内容")
		return
	}

	var data struct {
		Message struct {
			MsgID         string                 `json:"msgId"`
			TalkerID      uint                   `json:"talkerId"`
			ListenerID    uint                   `json:"listenerId"`
			RoomID        uint                   `json:"roomId"`
			Text          map[string]interface{} `json:"text"`
			Timestamp     int64                  `json:"timestamp"`
			Type          int                    `json:"type"`
			MentionIDList []uint                 `json:"mentionIdList"`
		} `json:"message"`
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

	if err := json.Unmarshal(msgBytes, &data); err != nil {
		client.Emit("error", "消息格式错误: "+err.Error())
		return
	}

	if data.Message.MsgID == "" {
		data.Message.MsgID = uuid.New().String()
	}

	message := models.Message{
		MsgID:         data.Message.MsgID,
		TalkerID:      data.Message.TalkerID,
		ListenerID:    data.Message.ListenerID,
		Text:          data.Message.Text,
		Timestamp:     data.Message.Timestamp,
		Type:          data.Message.Type,
		MentionIDList: data.Message.MentionIDList,
		RoomID:        data.Message.RoomID,
	}

	if err := recordMessage(&message); err != nil {
		client.Emit("error", "保存消息错误: "+err.Error())
		return
	}

	// 加载关联的用户和房间信息
	if err := db.Preload("Talker").Preload("Listener").Preload("Room").
		First(&message, "msg_id = ?", message.MsgID).Error; err != nil {
		client.Emit("error", "加载消息关联信息错误: "+err.Error())
		return
	}

	sendData := struct {
		User    *models.User    `json:"user"`
		Room    *models.Room    `json:"room"`
		Message *models.Message `json:"message"`
	}{
		User:    message.Talker,
		Room:    message.Room,
		Message: &message,
	}

	sendMessageToUsers(sendData, message.TalkerID, message.ListenerID)
}

// handleOffer 处理 "offer" 信令
func handleOffer(client *socket.Socket, sdp ...any) {
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

	// 保存 PeerConnection
	pcMutex.Lock()
	peerConnections[string(client.Id())] = peerConnection
	pcMutex.Unlock()

	client.Emit("answer", string(answerJSON))
}

// handleAnswer 处理 "answer" 信令
func handleAnswer(client *socket.Socket, sdp ...any) {
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

	pcMutex.RLock()
	peerConnection, exists := peerConnections[string(client.Id())]
	pcMutex.RUnlock()
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

// handleIceCandidate 处理 ICE 候选
func handleIceCandidate(client *socket.Socket, candidate ...any) {
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

	pcMutex.RLock()
	peerConnection, exists := peerConnections[string(client.Id())]
	pcMutex.RUnlock()
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

// sendMessageToUsers 发送消息给多个用户
func sendMessageToUsers(message interface{}, userIDs ...uint) {
	for _, userID := range userIDs {
		socketID := socket.SocketId(fmt.Sprintf("%d", userID)) // 将 socketID 转换为 socket.SocketId 类型
		clients := io.Sockets().Sockets()
		clients.Range(func(id socket.SocketId, client *socket.Socket) bool {
			if client.Id() == socketID {
				err := client.Emit("message", message)
				if err != nil {
					log.Printf("发送消息给用户 %d 失败: %v", userID, err)
				}
				return false // 停止遍历
			}
			return true // 继续遍历
		})
	}
}

// recordMessage 保存消息到数据库
func recordMessage(message *models.Message) error {
	return db.Create(message).Error
}

// GetSocketIOServer 获取 Socket.IO 服务器实例
func GetSocketIOServer() *socket.Server {
	return io
}
