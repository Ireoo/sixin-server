package database

import (
	"context"
	"database/sql"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/bson" // 注释掉未使用的导入
	// "go.mongodb.org/mongo-driver/mongo/options" // 注释掉未使用的导入
)

type DatabaseType string

const (
	SQLite DatabaseType = "sqlite"
	Mongo  DatabaseType = "mongodb"
)

type Database interface {
	Init(connectionString string) error
	Close() error
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type SQLDatabase struct {
	DB *sql.DB
}

type MongoDatabase struct {
	DB *mongo.Database
}

// 实现 MongoDB 的查询方法
func (m *MongoDatabase) Query(collectionName string, filter interface{}) (*mongo.Cursor, error) {
	collection := m.DB.Collection(collectionName)
	return collection.Find(context.Background(), filter)
}

func (m *MongoDatabase) QueryRow(collectionName string, filter interface{}) *mongo.SingleResult {
	collection := m.DB.Collection(collectionName)
	return collection.FindOne(context.Background(), filter)
}

// 实现 SQLiteDB 的查询方法
func (s *SQLiteDBClinet) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.DB.Query(query, args...)
}

func (s *SQLiteDBClinet) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.DB.QueryRow(query, args...)
}

var CurrentDB Database

func InitDatabase(dbType DatabaseType, connectionString string) error {
	var err error
	switch dbType {
	case SQLite:
		sqliteDB := &SQLiteDBClinet{}
		err = sqliteDB.Init(connectionString)
		if err != nil {
			return err
		}
		CurrentDB = sqliteDB
	case Mongo:
		mongoDB := &MongoDBClient{}
		err = mongoDB.Init(connectionString)
		if err != nil {
			return err
		}
		CurrentDB = mongoDB
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	return err
}

func GetCurrentDB() Database {
	return CurrentDB
}
