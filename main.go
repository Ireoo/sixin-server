package main

import (
	"fmt"
	"log"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
)

var server *socketio.Server

func main() {
	var err error
	// 创建Socket.IO服务器
	server, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	// 处理Socket.IO连接
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)
	})

	// 启动HTTP服务器
	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./public")))

	// HTTP处理程序
	http.HandleFunc("/message", handleMessage)

	log.Println("Serving at localhost:8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}

// HTTP处理程序
func handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		msg := r.FormValue("msg")
		fmt.Println("HTTP message:", msg)
		// 向所有连接的Socket.IO客户端发送消息
		server.BroadcastToNamespace("/", "reply", "HTTP message: "+msg)
		w.Write([]byte("Message received: " + msg))
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
