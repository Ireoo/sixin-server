package database

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var sqliteDB *SQLiteDB

func InitSqliteDB() error {
	// 创建必要的文件夹
	folders := []string{"data", "image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, folder := range folders {
		path := filepath.Join("data", folder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("创建文件夹 %s 失败: %v", path, err)
		}
	}

	// 连接数据库
	var err error
	sqliteDB = &SQLiteDB{}
	err = sqliteDB.Init()
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	fmt.Println("SQLite 数据库连接成功建立。")
	return nil
}

// 添加 GetDB 函数
func GetSqliteDB() *SQLiteDB {
	return sqliteDB
}

func (db *SQLiteDB) Init() error {
	// 创建必要的文件夹
	folders := []string{"data", "image", "avatar", "audio", "video", "attachment", "emoticon", "url", "database"}
	for _, folder := range folders {
		path := filepath.Join("data", folder)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("创建文件夹 %s 失败: %v", path, err)
		}
	}

	// 连接数据库
	var err error
	db.DB, err = gorm.Open(sqlite.Open("./database.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	fmt.Println("SQLite 数据库连接成功建立。")
	return nil
}

func (db *SQLiteDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *SQLiteDB) GetDB() *gorm.DB {
	return db.DB
}
