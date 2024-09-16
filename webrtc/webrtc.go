package webrtcServer

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

// WebRTCServer 定义了 WebRTC 服务器结构
type WebRTCServer struct {
	peerConnections map[string]*webrtc.PeerConnection
	mu              sync.Mutex
}

// NewWebRTCServer 创建一个新的 WebRTC 服务器实例
func NewWebRTCServer() *WebRTCServer {
	return &WebRTCServer{
		peerConnections: make(map[string]*webrtc.PeerConnection),
	}
}

// HandleWebRTC 处理 WebRTC 信令
func (s *WebRTCServer) HandleWebRTC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	var offer webrtc.SessionDescription
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	peerConnection, err := s.createPeerConnection()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 添加视频轨道
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "pion")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = peerConnection.AddTrack(videoTrack)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 启动一个 goroutine 来发送视频样本
	go func() {
		// 实现从视频源获取样本的逻辑
		for {
			// 模拟发送空白视频样本
			err := videoTrack.WriteSample(media.Sample{Data: []byte{}, Duration: 0})
			if err != nil {
				log.Println("写入视频样本失败:", err)
				return
			}
			// 控制发送频率
			// time.Sleep(time.Millisecond * 30)
		}
	}()

	// 创建数据通道
	dataChannel, err := peerConnection.CreateDataChannel("chat", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dataChannel.OnOpen(func() {
		log.Println("数据通道已打开")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("接收到数据通道消息: %s\n", string(msg.Data))
		// 回应消息
		err := dataChannel.SendText("消息已收到: " + string(msg.Data))
		if err != nil {
			log.Println("发送消息失败:", err)
		}
	})

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = peerConnection.SetLocalDescription(answer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.peerConnections[offer.SDP] = peerConnection
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(answer)

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		if connectionState == webrtc.ICEConnectionStateDisconnected || connectionState == webrtc.ICEConnectionStateFailed {
			peerConnection.Close()
			s.mu.Lock()
			delete(s.peerConnections, offer.SDP)
			s.mu.Unlock()
		}
	})

	// 处理接收到的远端视频轨道
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("接收到新的轨道: %s\n", track.Kind().String())
		for {
			_, _, err := track.ReadRTP()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Println("读取 RTP 包失败:", err)
				break
			}
			// 处理 RTP 包
		}
	})

	// 处理接收到的远端数据通道
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("接收到新的数据通道: %s\n", d.Label())
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("接收到消息: %s\n", string(msg.Data))
			// 回应消息
			err := d.SendText("消息已收到: " + string(msg.Data))
			if err != nil {
				log.Println("发送消息失败:", err)
			}
		})
	})
}

// createPeerConnection 创建一个新的 PeerConnection 并配置 ICE 服务器
func (s *WebRTCServer) createPeerConnection() (*webrtc.PeerConnection, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{"stun:your-stun-server.com:3478"},
				Username:   "your-username",
				Credential: "your-password",
			},
			{
				URLs:       []string{"turn:your-turn-server.com:3478"},
				Username:   "your-username",
				Credential: "your-password",
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// 监听远端数据通道
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("接收到新的数据通道: %s\n", d.Label())
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("接收到消息: %s\n", string(msg.Data))
			// 处理消息
		})
	})

	return peerConnection, nil
}
