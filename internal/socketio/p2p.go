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
	sdpStr, err := checkArgsAndType[string](sdp, 0)
	if err != nil {
		emitError(client, "SDP 不是字符串类型或缺少 SDP 数据", err)
		return
	}

	fmt.Printf("收到 SDP Offer: %s\n", sdpStr)

	offer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(sdpStr), &offer); err != nil {
		emitError(client, "解析 Offer SDP 失败", err)
		return
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		emitError(client, "创建 PeerConnection 失败", err)
		return
	}

	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		emitError(client, "设置远端描述失败", err)
		return
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		emitError(client, "创建 SDP Answer 失败", err)
		return
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		emitError(client, "设置本地描述失败", err)
		return
	}

	answerJSON, err := json.Marshal(answer)
	if err != nil {
		emitError(client, "序列化 SDP Answer 失败", err)
		return
	}

	sim.pcMutex.Lock()
	sim.peerConnections[string(client.Id())] = peerConnection
	sim.pcMutex.Unlock()

	client.Emit("answer", string(answerJSON))
}

func (sim *SocketIOManager) handleAnswer(client *socket.Socket, sdp ...any) {
	sdpStr, err := checkArgsAndType[string](sdp, 0)
	if err != nil {
		emitError(client, "SDP 不是字符串类型或缺少 SDP 数据", err)
		return
	}

	fmt.Printf("收到 SDP Answer: %s\n", sdpStr)

	answer := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(sdpStr), &answer); err != nil {
		emitError(client, "解析 Answer SDP 失败", err)
		return
	}

	sim.pcMutex.RLock()
	peerConnection, exists := sim.peerConnections[string(client.Id())]
	sim.pcMutex.RUnlock()
	if !exists {
		emitError(client, "PeerConnection 未找到", nil)
		return
	}

	if err := peerConnection.SetRemoteDescription(answer); err != nil {
		emitError(client, "设置远端描述失败", err)
		return
	}
}

func (sim *SocketIOManager) handleIceCandidate(client *socket.Socket, candidate ...any) {
	candidateStr, err := checkArgsAndType[string](candidate, 0)
	if err != nil {
		emitError(client, "ICE 候选不是字符串类型或缺少 ICE 候选数据", err)
		return
	}
	fmt.Printf("收到 ICE 候选: %s\n", candidateStr)

	iceCandidate := webrtc.ICECandidateInit{}
	if err := json.Unmarshal([]byte(candidateStr), &iceCandidate); err != nil {
		emitError(client, "解析 ICE 候选失败", err)
		return
	}

	sim.pcMutex.RLock()
	peerConnection, exists := sim.peerConnections[string(client.Id())]
	sim.pcMutex.RUnlock()
	if !exists {
		emitError(client, "PeerConnection 未找到，无法添加 ICE 候选", nil)
		return
	}

	if err := peerConnection.AddICECandidate(iceCandidate); err != nil {
		emitError(client, "添加 ICE 候选失败", err)
	}
}
