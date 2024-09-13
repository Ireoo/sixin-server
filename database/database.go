package database

import (
	"fmt"

	"gorm.io/gorm"
)

type DatabaseType string

const (
	SQLite DatabaseType = "sqlite"
)

type Database interface {
	Init() error
	Close() error
	GetDB() *gorm.DB
}

type SQLiteDB struct {
	DB *gorm.DB
}

var CurrentDB Database

func InitDatabase(dbType DatabaseType, connectionString string) error {
	var err error
	switch dbType {
	case SQLite:
		sqliteDB := &SQLiteDB{}
		err = sqliteDB.Init()
		CurrentDB = sqliteDB
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
	return err
}

func GetCurrentDB() Database {
	return CurrentDB
}
