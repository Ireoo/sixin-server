package database

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDB struct {
	DB *gorm.DB
}

var sqliteDB *SQLiteDB

func InitSqliteDB() error {
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
