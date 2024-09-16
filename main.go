package main

import (
	"log"

	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/database"
	"github.com/Ireoo/sixin-server/router"
)

func main() {
	// 初始化配置
	cfg := config.InitConfig()

	// 初始化数据库
	if err := database.InitDatabase(database.DatabaseType(cfg.DBType), cfg.DBConn); err != nil {

		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 设置并启动服务器
	router.SetupAndRun(cfg)
}
