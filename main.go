package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/Ireoo/sixin-server/pkg/base"
	"github.com/Ireoo/sixin-server/pkg/database"
	"github.com/Ireoo/sixin-server/pkg/socket"
	socketio "github.com/googollee/go-socket.io"
)

func main() {
	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 创建Socket.IO服务器
	server := socketio.NewServer(nil)
	if server == nil {
		log.Fatal("Failed to create Socket.IO server")
	} else {
		// 创建 base.Base 实例
		baseInstance := &base.Base{}
		// 获取数据库连接
		db := database.GetDB()
		// 设置Socket.IO事件处理
		socket.SetupSocketHandlers(server, db, baseInstance)
		log.Println("Socket.IO服务器创建成功")
	}

	// 定义命令行参数
	hostFlag := flag.String("host", "", "服务器主机名")
	portFlag := flag.Int("port", 0, "服务器端口")
	flag.Parse()

	// 优先使用命令行参数，其次是环境变量，最后是默认值
	host := *hostFlag
	if host == "" {
		host = os.Getenv("SERVER_HOST")
		if host == "" {
			host = "localhost"
		}
	}

	port := *portFlag
	if port == 0 {
		portStr := os.Getenv("SERVER_PORT")
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil || port == 0 {
			port = 8000
		}
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	log.Printf("服务器运行在 %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
