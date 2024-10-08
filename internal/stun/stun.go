package stunServer

import (
	"context"
	"log"
	"net"

	"github.com/pion/stun"
)

type transactionIDSetter [stun.TransactionIDSize]byte

func (t transactionIDSetter) AddTo(m *stun.Message) error {
	copy(m.TransactionID[:], t[:])
	return nil
}

// 处理 STUN 请求
func handleSTUNRequest(conn *net.UDPConn, addr *net.UDPAddr, msg *stun.Message) error {
	// Create a custom Transaction ID Setter using the incoming message's Transaction ID
	tidSetter := transactionIDSetter(msg.TransactionID)

	// 创建 STUN 响应消息
	response, err := stun.Build(
		stun.NewType(stun.MethodBinding, stun.ClassSuccessResponse), // 生成 STUN 成功响应类型
		tidSetter, // 直接使用 stun.TransactionID
		&stun.XORMappedAddress{
			IP:   addr.IP,
			Port: addr.Port,
		},
		stun.Fingerprint, // 添加指纹
	)
	if err != nil {
		return err
	}

	// 发送 STUN 响应
	_, err = conn.WriteToUDP(response.Raw, addr)
	return err
}

// 启动 STUN 服务器
func StartSTUNServer(ctx context.Context, address string) error {
	// 解析 UDP 地址
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		return err
	}

	// 开始监听 UDP 连接
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 打印STUN服务器启动信息
	log.Printf("STUN server started at %s", address)

	// 创建一个用于接收取消信号的通道
	go func() {
		<-ctx.Done()
		conn.Close()
		log.Println("STUN server shutting down")
	}()

	for {
		// 读取 UDP 数据
		buffer := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("Error reading from UDP: %v", err)
				continue
			}
		}

		// 检查是否是 STUN 消息
		if !stun.IsMessage(buffer[:n]) {
			log.Println("Received non-STUN message")
			continue
		}

		// 解码 STUN 消息
		message := &stun.Message{Raw: buffer[:n]}
		if err := message.Decode(); err != nil {
			log.Printf("Error decoding STUN message: %v", err)
			continue
		}

		// 并发处理 STUN 请求
		go func(remoteAddr *net.UDPAddr, message *stun.Message) {
			if err := handleSTUNRequest(conn, remoteAddr, message); err != nil {
				log.Printf("Error handling STUN request from %s: %v", remoteAddr, err)
			}
		}(remoteAddr, message)
	}
}
