package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     int
	DBType   string
	DBConn   string
	TestMode bool
}

func InitConfig() *Config {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用环境变量和命令行参数")
	}

	// 定义命令行参数
	dbTypeFlag := flag.String("db-type", "sqlite", "数据库类型 (postgres, mongodb, sqlite)")
	dbConnFlag := flag.String("db-uri", "./database.db", "数据库连接地址")
	hostFlag := flag.String("host", "0.0.0.0", "服务器主机名")
	portFlag := flag.Int("port", 80, "服务器端口")
	testFlag := flag.Bool("test", false, "测试模式，启动后立即关闭")
	flag.Parse()

	// 优先使用命令行参数，其次是环境变量，最后是默认值
	config := &Config{
		Host:     getEnv("HOST", *hostFlag),
		Port:     getEnvAsInt("PORT", *portFlag),
		DBType:   getEnv("DB_TYPE", *dbTypeFlag),
		DBConn:   getEnv("DB_URI", *dbConnFlag),
		TestMode: *testFlag,
	}

	return config
}

// getEnv 读取环境变量，如果不存在则返回默认值
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt 读取环境变量并转换为整数，如果不存在或无法转换则返回默认值
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// String 方法用于打印配置信息
func (c *Config) String() string {
	return fmt.Sprintf("Host: %s, Port: %d, DBType: %s, DBConn: %s, TestMode: %v",
		c.Host, c.Port, c.DBType, c.DBConn, c.TestMode)
}
