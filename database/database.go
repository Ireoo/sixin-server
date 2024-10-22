package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/Ireoo/sixin-server/models"
	"golang.org/x/crypto/bcrypt"
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

// 初始化数据库并应用迁移
func InitDatabase(dbType DatabaseType, connectionString string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch dbType {
	case SQLite:
		dialector = sqlite.Open(connectionString)
	case MySQL, TiDB:
		dialector = mysql.Open(connectionString)
	case Postgres:
		dialector = postgres.Open(connectionString)
	case SQLServer:
		dialector = sqlserver.Open(connectionString)
	case ClickHouse:
		dialector = clickhouse.Open(connectionString)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := initTables(db); err != nil {
		return nil, fmt.Errorf("初始化数据表失败: %w", err)
	}

	return db, nil
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
	db, err := InitDatabase(dbType, connectionString)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	dbManager := &DatabaseManager{DB: db}
	return dbManager, nil
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
