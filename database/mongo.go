package database

import (
	"context"
	"fmt"

	"database/sql" // 添加这个导入

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Rename the struct to avoid redeclaration
type MongoDBClient struct {
	Client *mongo.Client
	DB     *mongo.Database // 添加此字段以存储数据库引用
}

// 实现 MongoDB 的 Init 方法
func (m *MongoDBClient) Init(connectionString string) error {
	clientOptions := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	m.Client = client
	m.DB = client.Database("sixin") // 替换为实际的数据库名称
	return nil
}

func (db *MongoDBClient) Close() error {
	return db.Client.Disconnect(context.Background())
}

func (db *MongoDBClient) Query(query string, args ...interface{}) (*sql.Rows, error) {
	// MongoDB 不支持 SQL 查询，这里可以实现相应的 MongoDB 查询逻辑
	return nil, fmt.Errorf("MongoDB does not support SQL queries")
}

func (db *MongoDBClient) QueryRow(query string, args ...interface{}) *sql.Row {
	// MongoDB 不支持 SQL 查询，这里可以实现相应的 MongoDB 查询逻辑
	return nil
}
