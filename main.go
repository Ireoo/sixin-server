package main

import (
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // 添加这个导入
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
	dbUriFlag := flag.String("db-uri", "", "database uri")
	hostFlag := flag.String("host", "", "服务器主机名")
	portFlag := flag.Int("port", 0, "服务器端口")
	dbTypeFlag := flag.String("db-type", "", "数据库类型 (mongo 或 sqlite)")
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

	// 用户可以选择数据库类型
	dbType := os.Getenv("DB_TYPE")
	if *dbTypeFlag != "" {
		dbType = *dbTypeFlag
	}

	if dbType == "" {
		dbType = "sqlite" // 默认使用 mongo
	}

	// 用户可以指定数据库连接字符串
	dbUri := os.Getenv("DB_URI")
	if *dbUriFlag != "" {
		dbUri = *dbUriFlag
	}

	if dbUri == "" {
		if dbType == "sqlite" {
			dbUri = "./data/database/sixin.db" // SQLite 默认数据库路径
		} else {
			dbUri = "mongodb://localhost:27017/sixin" // MongoDB 默认地址
		}
	}

	// 创建必要的文件夹
	folders := []string{"data", "image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, folder := range folders {
		path := filepath.Join("data", folder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			log.Fatalf("创建文件夹 %s 失败: %v", path, err)
		}
	}

	// 创建Socket.IO服务器
	server := socketio.NewServer(nil)
	if server == nil {
		log.Fatal("Failed to create Socket.IO server")
	} else {
		// 创建 base.Base 实例
		baseInstance := &base.Base{}

		// 初始化数据库
		log.Printf("初始化数据库，类型: %s, 地址: %s", dbType, dbUri)
		err := database.InitDatabase(database.DatabaseType(dbType), dbUri)
		if err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}

		// 获取数据库连接并设置Socket.IO事件处理
		currentDB := database.GetCurrentDB()
		if currentDB == nil {
			log.Fatalf("无法获取数据库实例")
		}
		socket.SetupSocketHandlers(server, currentDB, baseInstance)
		log.Println("Socket.IO服务器创建成功")
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	log.Printf("服务器运行在 %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
