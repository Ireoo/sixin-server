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
	// 定义命令行参数
	dbTypeFlag := flag.String("db-type", "sqlite", "数据库类型 (postgres, mongodb, sqlite)")
	dbConnFlag := flag.String("db-uri", "./database.db", "数据库连接地址")
	hostFlag := flag.String("host", "0.0.0.0", "服务器主机名")
	portFlag := flag.Int("port", 8000, "服务器端口")
	testFlag := flag.Bool("test", false, "测试模式，启动后立即关闭")
	flag.Parse()

	// 创建必要的文件夹
	folders := []string{"data", "image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, folder := range folders {
		path := filepath.Join("data", folder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			log.Fatalf("创建文件夹 %s 失败: %v", path, err)
		}
	}

	// 优先使用命令行参数，其次是环境变量，最后是默认值
	dbType := database.DatabaseType(*dbTypeFlag)
	if dbType == "" {
		dbTypeStr := os.Getenv("DB_TYPE")
		if dbTypeStr == "" {
			dbTypeStr = "sqlite" // 默认数据库类型
		}
	}

	dbConn := *dbConnFlag
	if dbConn == "" {
		dbConn = os.Getenv("DB_URI")
		if dbConn == "" {
			dbConn = "./database.db" // 默认数据库连接地址
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
		err := database.InitDatabase(dbType, dbConn)
		if err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}

		// 获取当前数据库连接
		db := database.GetCurrentDB()
		// 设置Socket.IO事件处理
		socket.SetupSocketHandlers(server, db.GetDB(), baseInstance)
		log.Println("Socket.IO服务器创建成功")
	}

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

	// 检查是否为测试模式
	if *testFlag {
		fmt.Println("测试模式启动，现在退出...")
		os.Exit(0)
	}
}
