package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"github.com/Ireoo/sixin-server/models"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type DatabaseType string

const (
	SQLite     DatabaseType = "sqlite"
	MySQL      DatabaseType = "mysql"
	Postgres   DatabaseType = "postgres"
	SQLServer  DatabaseType = "sqlserver"
	TiDB       DatabaseType = "tidb"
	ClickHouse DatabaseType = "clickhouse"
)

type Database interface {
	Init(connectionString string) error
	Close() error
	GetDB() *gorm.DB
}

type GormDB struct {
	DB *gorm.DB
}

var currentDB Database

// 初始化数据库并应用迁移
func InitDatabase(dbType DatabaseType, connectionString string) error {
	var db *gorm.DB
	var err error

	switch dbType {
	case SQLite:
		db, err = gorm.Open(sqlite.Open(connectionString), &gorm.Config{})
	case MySQL, TiDB:
		db, err = gorm.Open(mysql.Open(connectionString), &gorm.Config{})
	case Postgres:
		db, err = gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	case SQLServer:
		db, err = gorm.Open(sqlserver.Open(connectionString), &gorm.Config{})
	case ClickHouse:
		db, err = gorm.Open(clickhouse.Open(connectionString), &gorm.Config{})
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	gormDB := &GormDB{DB: db}
	currentDB = gormDB

	// 初始化数据表
	if err := initTables(db); err != nil {
		return fmt.Errorf("初始化数据表失败: %w", err)
	}

	return nil
}

func GetCurrentDB() Database {
	return currentDB
}

// GormDB 方法实现
func (g *GormDB) Init(connectionString string) error {
	// 初始化已经在 InitDatabase 中完成
	return nil
}

func (g *GormDB) Close() error {
	sqlDB, err := g.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (g *GormDB) GetDB() *gorm.DB {
	return g.DB
}

// 使用 models.GetAllModels() 初始化数据表
func initTables(db *gorm.DB) error {
	models := models.GetAllModels()
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}

	log.Println("数据库表初始化成功。")
	return nil
}

// DatabaseManager 结构体及其方法
type DatabaseManager struct {
	DB *gorm.DB
}

func NewDatabaseManager(dbType DatabaseType, connectionString string) (*DatabaseManager, error) {
	err := InitDatabase(dbType, connectionString)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	db := GetCurrentDB()
	return &DatabaseManager{DB: db.GetDB()}, nil
}

// 辅助函数：生成密钥
func generateSecretKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// 用户登录验证方法
func (dm *DatabaseManager) AuthenticateUser(username, password string) (*models.User, error) {
	var user models.User
	if err := dm.DB.Model(&models.User{}).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("密码不正确")
	}

	return &user, nil
}
