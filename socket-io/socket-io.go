package socketIo

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Ireoo/sixin-server/base"
	models "github.com/Ireoo/sixin-server/models"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/zishang520/socket.io/v2/socket"
	"gorm.io/gorm"
)

var db *gorm.DB
var baseInstance *base.Base
var io *socket.Server
var peerConnections = make(map[string]*webrtc.PeerConnection)

func SetupSocketHandlers(database *gorm.DB, baseInst *base.Base) *socket.Server {
	db = database
	baseInstance = baseInst

	io = socket.NewServer(nil, nil)

	io.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		fmt.Println("新连接：", client.Id())

		client.Emit("receive", baseInstance.ReceiveDevice)
		client.Emit("email", baseInstance.EmailNote)
		client.Emit("self", baseInstance.Self)
		client.Emit("qrcode", baseInstance.Qrcode)

		client.On("self", func(args ...any) {
			handleSelf(client)
		})
		client.On("receive", func(args ...any) {
			handleReceive(client)
		})
		client.On("message", func(args ...any) {
			handleMessage(client, args...)
		})
		client.On("email", func(args ...any) {
			handleEmail(client)
		})
		client.On("revokemsg", func(args ...any) {
			handleRevokeMsg(client, args...)
		})
		client.On("getChats", func(args ...any) {
			handleGetChats(client, args...)
		})
		client.On("getRooms", func(args ...any) {
			handleGetRooms(client, args...)
		})
		client.On("getUsers", func(args ...any) {
			handleGetUsers(client, args...)
		})
		client.On("getRoomByUsers", func(args ...any) {
			handleGetRoomByUsers(client, args...)
		})

		client.On("disconnecting", func(reason ...any) {
			fmt.Println("连接断开:", client.Id(), reason)
			clientID := string(client.Id())
			if peerConnection, ok := peerConnections[clientID]; ok {
				peerConnection.Close()            // 关闭 PeerConnection
				delete(peerConnections, clientID) // 从 map 中删除
			}
		})

		// 监听来自客户端的 "offer" 信令
		client.On("offer", func(sdp ...any) {
			if len(sdp) == 0 {
				return
			}

			sdpStr, ok := sdp[0].(string)
			if !ok {
				fmt.Println("SDP 不是字符串类型")
				return
			}

			fmt.Printf("收到 SDP Offer: %s\n", sdpStr)

			// 创建 PeerConnection
			peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				log.Printf("创建 PeerConnection 失败: %v", err)
				return
			}
			peerConnections[string(client.Id())] = peerConnection

			offer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(sdpStr), &offer); err != nil {
				fmt.Printf("解析 Offer SDP 失败: %v", err)
				client.Emit("error", "Offer SDP 无效")
				return
			}

			// 设置远端描述为 Offer
			if err := peerConnection.SetRemoteDescription(offer); err != nil {
				fmt.Printf("设置远端描述失败: %v", err)
				client.Emit("error", "设置远端描述失败")
				return
			}

			// 创建 Answer 并发送给客户端
			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Printf("创建 SDP Answer 失败: %v", err)
				client.Emit("error", "创建 Answer 失败")
				return
			}

			if err := peerConnection.SetLocalDescription(answer); err != nil {
				fmt.Printf("设置本地描述失败: %v", err)
				client.Emit("error", "设置本地描述失败")
				return
			}

			// 将 SDP Answer 发送回客户端
			answerJSON, err := json.Marshal(answer)
			if err != nil {
				fmt.Printf("序列化 SDP Answer 失败: %v", err)
				return
			}
			client.Emit("answer", string(answerJSON))
		})

		// 监听来自客户端的 "answer" 信令
		client.On("answer", func(sdp ...any) {
			if len(sdp) == 0 {
				return
			}

			sdpStr, ok := sdp[0].(string)
			if !ok {
				fmt.Println("SDP 不是字符串类型")
				return
			}

			fmt.Printf("收到 SDP Answer: %s\n", sdpStr)

			// 假设在这里更新 PeerConnection 的远端描述为 Answer
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(sdpStr), &answer); err != nil {
				log.Printf("解析 Answer SDP 失败: %v", err)
				client.Emit("error", "Answer SDP 无效")
				return
			}

			peerConnection, exists := peerConnections[string(client.Id())]
			if !exists {
				log.Printf("PeerConnection 未找到")
				return
			}

			// 更新远端描述为 Answer
			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Printf("设置远端描述失败: %v", err)
				client.Emit("error", "设置远端描述失败")
				return
			}
		})

		// 监听 ICE 候选消息
		client.On("ice-candidate", func(candidate ...any) {
			if len(candidate) == 0 {
				return
			}

			candidateStr, ok := candidate[0].(string)
			if !ok {
				fmt.Println("ICE 候选不是字符串类型")
				return
			}
			fmt.Printf("收到 ICE 候选: %s\n", candidateStr)

			peerConnection, exists := peerConnections[string(client.Id())]
			if !exists {
				log.Printf("PeerConnection 未找到，无法添加 ICE 候选")
				return
			}

			// 解析 ICE 候选并添加到 PeerConnection
			iceCandidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(candidateStr), &iceCandidate); err != nil {
				log.Printf("解析 ICE 候选失败: %v", err)
				return
			}

			if err := peerConnection.AddICECandidate(iceCandidate); err != nil {
				log.Printf("添加 ICE 候选失败: %v", err)
			}
		})
	})

	return io
}

func handleSelf(client *socket.Socket) {
	client.Emit("self", baseInstance.Self)
}

func handleReceive(client *socket.Socket) {
	baseInstance.ReceiveDevice = !baseInstance.ReceiveDevice
	message := "wechat:receive"
	if !baseInstance.ReceiveDevice {
		message = "wechat:message"
	}
	baseInstance.SendMessage(message, message)
	client.Emit("receive", baseInstance.ReceiveDevice)
}

func handleEmail(client *socket.Socket) {
	baseInstance.EmailNote = !baseInstance.EmailNote
	client.Emit("email", baseInstance.EmailNote)
}

func handleRevokeMsg(client *socket.Socket, args ...any) {
	// 实现撤回消息的逻辑
}

func handleGetChats(client *socket.Socket, args ...any) {
	var messages []models.Message
	result := db.Order("timestamp DESC").Limit(400).Find(&messages)
	if result.Error != nil {
		client.Emit("error", result.Error.Error())
		return
	}
	client.Emit("getChats", messages)
}

func handleGetRooms(client *socket.Socket, args ...any) {
	var rooms []models.Room
	result := db.Order("timestamp DESC").Find(&rooms)
	if result.Error != nil {
		client.Emit("error", result.Error.Error())
		return
	}
	client.Emit("getRooms", rooms)
}

func handleGetUsers(client *socket.Socket, args ...any) {
	var users []models.User
	result := db.Order("timestamp DESC").Find(&users)
	if result.Error != nil {
		client.Emit("error", result.Error.Error())
		return
	}
	client.Emit("getUsers", users)
}

func handleGetRoomByUsers(client *socket.Socket, args ...any) {
	// 实现获取用户房间的逻辑
}

func handleMessage(client *socket.Socket, args ...any) {
	if len(args) == 0 {
		return
	}
	msgStr, ok := args[0].(string)
	if !ok {
		client.Emit("error", "消息格式错误")
		return
	}

	fmt.Println("收到消息：", msgStr)
	var data struct {
		Message struct {
			MsgID      string `json:"msgId"`
			TalkerID   uint   `json:"talkerId"`
			ListenerID uint   `json:"listenerId"`
			RoomID     uint   `json:"roomId"`
			Text       struct {
				Message string `json:"message"`
				Image   string `json:"image"`
			} `json:"text"`
			Timestamp     int64  `json:"timestamp"`
			Type          int    `json:"type"`
			MentionIDList string `json:"mentionIdList"`
		} `json:"message"`
	}

	if err := json.Unmarshal([]byte(msgStr), &data); err != nil {
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
		Text:          data.Message.Text.Message,
		Timestamp:     data.Message.Timestamp,
		Type:          data.Message.Type,
		MentionIDList: data.Message.MentionIDList,
		RoomID:        data.Message.RoomID,
	}

	if err := recordMessage(message); err != nil {
		client.Emit("error", "保存消息错误: "+err.Error())
	}

	sendData := struct {
		User    models.User    `json:"user"`
		Room    models.Room    `json:"room"`
		Message models.Message `json:"message"`
	}{
		User:    models.User{ID: message.TalkerID},
		Room:    models.Room{ID: message.RoomID},
		Message: message,
	}

	sendMessageToUser(message.TalkerID, sendData)
	sendMessageToUser(message.ListenerID, sendData)
}

func sendMessageToUser(userID uint, message interface{}) {
	sockets := io.Sockets().Sockets()
	sockets.Range(func(key socket.SocketId, value *socket.Socket) bool {
		client := value
		if client.Id() == socket.SocketId(fmt.Sprintf("%d", userID)) {
			client.Emit("message", message)
			return false
		}
		return true
	})
}

func recordMessage(message models.Message) error {
	result := db.Create(&message)
	return result.Error
}

func GetSocketIOServer() *socket.Server {
	return io
}
