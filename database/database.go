package database

import (
	"database/sql"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

type DatabaseType string

const (
	SQLite      DatabaseType = "sqlite"
	MongoDBType DatabaseType = "mongodb"
)

type Database interface {
	Init() error
	Close() error
}

type SQLiteDB struct {
	DB *sql.DB
}

type MongoDB struct {
	Client *mongo.Client
}

var CurrentDB Database

func InitDatabase(dbType DatabaseType, connectionString string) error {
	var err error
	switch dbType {
	case SQLite:
		sqliteDB := &SQLiteDB{}
		err = sqliteDB.Init()
		CurrentDB = sqliteDB
	case MongoDBType:
		mongoDB := &MongoDB{}
		err = mongoDB.Init()
		CurrentDB = mongoDB
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	return err
}

func GetCurrentDB() Database {
	return CurrentDB
}
