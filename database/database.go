package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // 使用纯 Go 实现的 SQLite 驱动
)

var DB *sql.DB

type User struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	WechatID  string `gorm:"uniqueIndex"`
	Name      string
	Phone     string `gorm:"type:json"`
	Province  string
	Signature string
	Type      int
	Weixin    string
	Alias     string
	Avatar    string
	City      string
	Friend    bool
	Gender    string
}

type Room struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	RoomID       string `gorm:"uniqueIndex"`
	Topic        string
	OwnerID      string
	MemberIDList string `gorm:"type:json"`
	Avatar       string
	AdminIDList  string `gorm:"type:json"`
}

type RoomByUser struct {
	ID    uint `gorm:"primaryKey;autoIncrement"`
	Name  string
	Alias string
	Topic string
}

type Message struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	MsgID         string `gorm:"uniqueIndex"`
	TalkerID      string
	ListenerID    string
	Text          string `gorm:"type:text"`
	Timestamp     int64
	Type          int
	RoomID        string
	MentionIDList string `gorm:"type:json"`
}

func InitDB() error {
	// 创建必要的文件夹
	folders := []string{"data", "image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, folder := range folders {
		path := filepath.Join("data", folder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create folder %s: %v", path, err)
		}
	}

	// 连接数据库
	var err error
	DB, err = sql.Open("sqlite", "./database.db")
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 测试连接
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("数据库连接成功建立。")
	return nil
}

// 添加 GetDB 函数
func GetDB() *sql.DB {
	return DB
}
