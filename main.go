package main

import (
	"fmt"

	"github.com/Ireoo/sixin-server/config"
	"github.com/Ireoo/sixin-server/router"
)

func main() {
	// 初始化配置
	cfg := config.InitConfig()

	// 打印配置
	fmt.Printf("配置: %+v\n", cfg)

	// 设置并启动服务器
	router.SetupAndRun(cfg)
}
