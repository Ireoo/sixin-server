package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Host              string
	StunPort          int
	Port              int
	DBType            string
	DBConn            string
	TestMode          bool
	EnableSomeFeature bool
}

// InitConfig initializes and returns the application configuration
func InitConfig() *Config {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("警告: 未找到 .env 文件，将使用环境变量和命令行参数")
	}

	// Define command-line flags
	pflag.String("db-type", "", "数据库类型 (postgres, mongodb, sqlite)")
	pflag.String("db-uri", "", "数据库连接地址")
	pflag.String("host", "", "服务器主机名")
	pflag.Int("port", 0, "服务器端口")
	pflag.Int("stun-port", 0, "STUN 服务器端口")
	pflag.Bool("test", false, "测试模式，启动后立即关闭")
	pflag.Bool("enable-feature", false, "是否启用某个功能")
	pflag.Parse()

	// Bind command-line flags to viper
	err = viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("错误: 绑定命令行参数失败: %v", err)
	}

	// Read environment variables
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 80)
	viper.SetDefault("stun-port", 3478)
	viper.SetDefault("db-type", "sqlite")
	viper.SetDefault("db-uri", "./database.db")
	viper.SetDefault("enable-feature", false)

	// Create Config instance
	config := &Config{
		Host:              getStringConfig("host"),
		Port:              getIntConfig("port"),
		StunPort:          getIntConfig("stun-port"),
		DBType:            getStringConfig("db-type"),
		DBConn:            getStringConfig("db-uri"),
		TestMode:          viper.GetBool("test"),
		EnableSomeFeature: viper.GetBool("enable-feature"),
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	return config
}

// getStringConfig retrieves a string configuration value
func getStringConfig(key string) string {
	value := viper.GetString(key)
	if value == "" {
		log.Fatalf("配置错误: 必须设置 '%s'", key)
	}
	return value
}

// getIntConfig retrieves an integer configuration value
func getIntConfig(key string) int {
	if !viper.IsSet(key) {
		log.Fatalf("配置错误: 必须设置 '%s'", key)
	}
	return viper.GetInt(key)
}

// Validate checks the Config for required fields and valid values
func (c *Config) Validate() error {
	if c.DBConn == "" {
		return fmt.Errorf("数据库连接字符串不能为空")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("无效的服务器端口号: %d", c.Port)
	}
	if c.StunPort <= 0 || c.StunPort > 65535 {
		return fmt.Errorf("无效的 STUN 服务器端口号: %d", c.StunPort)
	}
	// 添加其他验证逻辑
	return nil
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	return fmt.Sprintf("Host: %s, STUN Port: %d, Port: %d, DB Type: %s, DB Conn: %s, Test Mode: %v, Enable Feature: %v",
		c.Host, c.StunPort, c.Port, c.DBType, c.DBConn, c.TestMode, c.EnableSomeFeature)
}
