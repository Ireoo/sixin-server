package router

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/handlers"
	"github.com/Ireoo/sixin-server/middleware"
	"github.com/Ireoo/sixin-server/socket-io"
	"github.com/Ireoo/sixin-server/stun"
	"github.com/Ireoo/sixin-server/webrtc"
)

func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		raw := r.URL.RawQuery

		next.ServeHTTP(w, r)

		latency := time.Since(start)
		clientIP := r.RemoteAddr
		method := r.Method
		statusCode := http.StatusOK // 注意：这里无法获取真实的状态码，需要自定义 ResponseWriter

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

func SetupAndRun(cfg *config.Config) {
	// 创建 http.Server 实例而不是 http.ServeMux
	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}

	// 创建路由器
	mux := http.NewServeMux()

	// 创建 base.Base 实例
	baseInstance := &base.Base{}

	// 获取数据库实例
	db := database.GetCurrentDB()

	// 设置Socket.IO事件处理
	io := socketIo.SetupSocketHandlers(db.GetDB(), baseInstance)

	http.Handle("/socket.io/", io.ServeHandler(nil))

	// 初始化WebRTC服务器
	_webrtcServer := webrtcServer.NewWebRTCServer()

	http.HandleFunc("/webrtc", _webrtcServer.HandleWebRTC)

	// // 初始化SFU
	// sfuConfig := sfu.Config{}
	// sfuInstance := sfu.NewSFU(sfuConfig)

	// // 设置SFU处理程序
	// mux.HandleFunc("/sfu", handleSFU(sfuInstance))

	// 设置中间件
	handler := handleRoutes()
	handler = loggerMiddleware(handler)
	handler = middleware.Logger(handler)
	handler = middleware.CORS(handler)
	mux.HandleFunc("/", handler)

	// 静态文件服务
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 设置STUN服务器
	go func() {
		stunAddress := fmt.Sprintf("%s:%d", cfg.Host, cfg.StunPort) // 假设配置中有StunPort
		if err := stunServer.StartSTUNServer(stunAddress); err != nil {
			log.Printf("STUN服务器启动失败: %v", err)
		}
	}()

	// 设置服务器的处理器
	server.Handler = mux

	// 启动服务器
	startServer(server, cfg)
}

func handleRoutes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ping":
			handlers.Ping(w, r)
		case "/api/users":
			switch r.Method {
			case http.MethodGet:
				handlers.GetUsers(w, r)
			case http.MethodPost:
				handlers.CreateUser(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			if r.URL.Path[:10] == "/api/users/" {
				id := r.URL.Path[10:]
				switch r.Method {
				case http.MethodGet:
					handlers.GetUser(w, r, id)
				case http.MethodPut:
					handlers.UpdateUser(w, r, id)
				case http.MethodDelete:
					handlers.DeleteUser(w, r, id)
				default:
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			} else {
				http.NotFound(w, r)
			}
		}
	}
}

// func handleSFU(sfuInstance *sfu.SFU) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// 升级HTTP连接为WebSocket
// 		upgrader := websocket.Upgrader{
// 			CheckOrigin: func(r *http.Request) bool {
// 				return true
// 			},
// 		}
// 		conn, err := upgrader.Upgrade(w, r, nil)
// 		if err != nil {
// 			log.Printf("升级到WebSocket失败: %v", err)
// 			return
// 		}
// 		defer conn.Close()

// 		// 创建新的 WebRTC PeerConnection
// 		peerConnectionConfig := webrtc.Configuration{}
// 		peerConnection, err := webrtc.NewPeerConnection(peerConnectionConfig)
// 		if err != nil {
// 			log.Printf("创建WebRTC PeerConnection失败: %v", err)
// 			return
// 		}
// 		defer peerConnection.Close()

// 		// 创建包装器
// 		pcWrapper := &peerConnectionWrapper{
// 			pc:  peerConnection,
// 			sfu: sfuInstance,
// 		}

// 		// 使用包装器创建 peer
// 		peer := sfu.NewPeer(pcWrapper)

// 		// 监听信令消息（SDP 和 ICE 候选）
// 		for {
// 			_, message, err := conn.ReadMessage()
// 			if err != nil {
// 				log.Printf("读取WebSocket消息失败: %v", err)
// 				break
// 			}

// 			// 假设接收到的消息是 SDP，可以进一步解析处理
// 			log.Printf("收到消息: %s", message)

// 			// 根据具体信令处理消息，通常包括 SDP 和 ICE 候选的交换
// 			offer := webrtc.SessionDescription{}
// 			err = json.Unmarshal(message, &offer)
// 			if err != nil {
// 				log.Printf("解析SDP失败: %v", err)
// 				continue
// 			}
// 			// peer.OnOffer 等方法来处理信令
// 			peer.OnOffer(&offer)
// 		}
// 	}
// }

// // 创建一个包装器结构体
// type peerConnectionWrapper struct {
// 	pc  *webrtc.PeerConnection
// 	sfu *sfu.SFU
// }

// // 实现 GetSession 方法
// func (pcw *peerConnectionWrapper) GetSession(sid string) (sfu.Session, sfu.WebRTCTransportConfig) {
// 	session, config := pcw.sfu.GetSession(sid)
// 	return session, config
// }

func startServer(server *http.Server, cfg *config.Config) {
	log.Printf("服务器运行在 %s...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}
