package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Ireoo/sixin-server/base"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/socket"
	socketio "github.com/googollee/go-socket.io"
)

func main() {
	// 创建必要的文件夹
	folders := []string{"data", "image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, folder := range folders {
		path := filepath.Join("data", folder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			log.Fatalf("创建文件夹 %s 失败: %v", path, err)
		}
	}

	// 初始化数据库
	if err := database.InitSqliteDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 创建Socket.IO服务器
	server := socketio.NewServer(nil)
	if server == nil {
		log.Fatal("Failed to create Socket.IO server")
	} else {
		// 创建 base.Base 实例
		baseInstance := &base.Base{}

		// 用户可以选择数据库类型
		dbType := database.SQLite // 或 database.MongoDB

		err := database.InitDatabase(dbType, "")
		if err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}
		// 获取数据库连接
		db := database.GetSqliteDB()
		// 设置Socket.IO事件处理
		socket.SetupSocketHandlers(server, db.DB, baseInstance)
		log.Println("Socket.IO服务器创建成功")
	}

	// 定义命令行参数
	hostFlag := flag.String("host", "", "服务器主机名")
	portFlag := flag.Int("port", 0, "服务器端口")
	flag.Parse()

	// 优先使用命令行参数，其次是环境变量，最后是默认值
	host := *hostFlag
	if host == "" {
		host = os.Getenv("HOST")
		if host == "" {
			host = "localhost"
		}
	}

	port := *portFlag
	if port == 0 {
		portStr := os.Getenv("PORT")
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
