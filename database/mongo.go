package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	Client *mongo.Client
}

func (db *MongoDBClient) Init() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("无法连接到 MongoDB: %v", err)
	}

	// 检查连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("无法 ping MongoDB: %v", err)
	}

	db.Client = client
	log.Println("成功连接到 MongoDB")
	return nil
}

func (db *MongoDBClient) Close() error {
	return db.Client.Disconnect(context.Background())
}

func (db *MongoDBClient) GetCollection(database, collection string) *mongo.Collection {
	return db.Client.Database(database).Collection(collection)
}

func (m *MongoDB) Close() error {
	return m.Client.Disconnect(context.Background())
}

// 添加 Init 方法
func (m *MongoDB) Init() error {
	// 在这里初始化 MongoDB 连接
	fmt.Println("Initializing MongoDB connection")
	// ... 初始化代码 ...
	return nil
}

func NewDatabase() (Database, error) {
	mongoDB := &MongoDB{}
	err := mongoDB.Init() // 现在可以调用 Init 方法了
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB: %w", err)
	}
	return mongoDB, nil
}
