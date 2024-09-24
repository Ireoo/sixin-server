package socketio

import (
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/zishang520/socket.io/v2/socket"
)

func (sim *SocketIOManager) cleanupPeerConnection(clientID socket.SocketId) {
	sim.pcMutex.Lock()
	defer sim.pcMutex.Unlock()

	id := string(clientID)
	if peerConnection, exists := sim.peerConnections[id]; exists {
		if err := peerConnection.Close(); err != nil {
			fmt.Printf("关闭 PeerConnection 失败: %v", err)
		}
		delete(sim.peerConnections, id)
	}
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
		fmt.Printf("创建 PeerConnection 失败: %v\n", err)
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
		fmt.Printf("解析 Answer SDP 失败: %v\n", err)
		client.Emit("error", "Answer SDP 无效")
		return
	}

	sim.pcMutex.RLock()
	peerConnection, exists := sim.peerConnections[string(client.Id())]
	sim.pcMutex.RUnlock()
	if !exists {
		fmt.Printf("PeerConnection 未找到\n")
		client.Emit("error", "PeerConnection 未找到")
		return
	}

	if err := peerConnection.SetRemoteDescription(answer); err != nil {
		fmt.Printf("设置远端描述失败: %v\n", err)
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
		fmt.Printf("解析 ICE 候选失败: %v\n", err)
		client.Emit("error", "ICE 候选解析失败")
		return
	}

	sim.pcMutex.RLock()
	peerConnection, exists := sim.peerConnections[string(client.Id())]
	sim.pcMutex.RUnlock()
	if !exists {
		fmt.Printf("PeerConnection 未找到，无法添加 ICE 候选\n")
		client.Emit("error", "PeerConnection 未找到")
		return
	}

	if err := peerConnection.AddICECandidate(iceCandidate); err != nil {
		fmt.Printf("添加 ICE 候选失败: %v\n", err)
		client.Emit("error", "添加 ICE 候选失败")
	}
}
