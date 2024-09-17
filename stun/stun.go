package stunServer

import (
	"context"
	"log"
	"net"

	"github.com/pion/stun"
)

const (
	MaxBufferSize = 2048
	MaxGoroutines = 1000
)

// Custom Transaction ID Setter
type transactionIDSetter [stun.TransactionIDSize]byte

func (t transactionIDSetter) AddTo(m *stun.Message) error {
	copy(m.TransactionID[:], t[:])
	return nil
}

func handleSTUNRequest(conn *net.UDPConn, addr *net.UDPAddr, message *stun.Message) error {
	// Create a custom Transaction ID Setter using the incoming message's Transaction ID
	tidSetter := transactionIDSetter(message.TransactionID)

	// Build the response message
	response, err := stun.Build(
		stun.NewType(stun.MethodBinding, stun.ClassSuccessResponse),
		tidSetter, // Use the custom Transaction ID Setter
		&stun.XORMappedAddress{
			IP:   addr.IP,
			Port: addr.Port,
		},
		stun.Fingerprint,
	)
	if err != nil {
		return err
	}

	_, err = conn.WriteToUDP(response.Raw, addr)
	return err
}

func StartSTUNServer(ctx context.Context, address string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("STUN server started at %s", address)

	tokens := make(chan struct{}, MaxGoroutines)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down STUN server...")
			return nil
		default:
			buffer := make([]byte, MaxBufferSize)
			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				log.Printf("Error reading from UDP: %v", err)
				continue
			}

			data := make([]byte, n)
			copy(data, buffer[:n])

			if !stun.IsMessage(data) {
				log.Println("Received non-STUN message")
				continue
			}

			tokens <- struct{}{}
			go func(remoteAddr *net.UDPAddr, data []byte) {
				defer func() { <-tokens }()

				message := &stun.Message{Raw: data}
				if err := message.Decode(); err != nil {
					log.Printf("Error decoding STUN message from %s: %v", remoteAddr, err)
					return
				}

				if err := handleSTUNRequest(conn, remoteAddr, message); err != nil {
					log.Printf("Error handling STUN request from %s: %v", remoteAddr, err)
				}
			}(remoteAddr, data)
		}
	}
}
