package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // 使用纯 Go 实现的 SQLite 驱动
)

var sqliteDB *sql.DB

func InitSqliteDB() error {
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
	sqliteDB, err = sql.Open("sqlite", "./database.db")
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 测试连接
	if err = sqliteDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("数据库连接成功建立。")
	return nil
}

// 添加 GetDB 函数
func GetSqliteDB() *sql.DB {
	return sqliteDB
}

func (db *SQLiteDB) Init() error {
	// 连接数据库
	var err error
	db.DB, err = sql.Open("sqlite", "./database.db")
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 测试连接
	if err = db.DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	fmt.Println("SQLite 数据库连接成功建立。")
	return nil
}

func (db *SQLiteDB) Close() error {
	return db.DB.Close()
}
