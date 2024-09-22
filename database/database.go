package database

import (
	"fmt"
	"log"

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

// 用户相关操作
func (dm *DatabaseManager) GetAllUsers() ([]models.User, error) {
	var users []models.User
	err := dm.DB.Find(&users).Error
	return users, err
}

func (dm *DatabaseManager) GetUserByID(id uint) (models.User, error) {
	var user models.User
	err := dm.DB.First(&user, id).Error
	return user, err
}

func (dm *DatabaseManager) CreateUser(user *models.User) error {
	return dm.DB.Create(user).Error
}

func (dm *DatabaseManager) UpdateUser(id uint, updatedUser models.User) error {
	return dm.DB.Model(&models.User{}).Where("id = ?", id).Updates(updatedUser).Error
}

func (dm *DatabaseManager) DeleteUser(id uint) error {
	return dm.DB.Delete(&models.User{}, id).Error
}

// 户间相关操作
func (dm *DatabaseManager) GetAllRooms() ([]models.Room, error) {
	var rooms []models.Room
	err := dm.DB.Preload("Owner").Preload("Members").Find(&rooms).Error
	return rooms, err
}

func (dm *DatabaseManager) GetRoomByID(id uint) (models.Room, error) {
	var room models.Room
	err := dm.DB.Preload("Owner").Preload("Members").First(&room, id).Error
	return room, err
}

func (dm *DatabaseManager) CreateRoom(room *models.Room) error {
	return dm.DB.Create(room).Error
}

func (dm *DatabaseManager) UpdateRoom(id uint, updatedRoom models.Room) error {
	return dm.DB.Model(&models.Room{}).Where("id = ?", id).Updates(updatedRoom).Error
}

func (dm *DatabaseManager) DeleteRoom(id uint) error {
	return dm.DB.Delete(&models.Room{}, id).Error
}

// 消息相关操作
func (dm *DatabaseManager) CreateMessage(message *models.Message) error {
	return dm.DB.Create(message).Error
}

func (dm *DatabaseManager) GetFullMessage(id uint) (models.FullMessage, error) {
	var fullMessage models.FullMessage
	err := dm.DB.Model(&models.Message{}).Where("id = ?", id).
		Preload("Talker").Preload("Listener").Preload("Room").
		First(&fullMessage.Message).Error

	return fullMessage, err
}

func (dm *DatabaseManager) GetChats() ([]models.Message, error) {
	var messages []models.Message
	err := dm.DB.Preload("Talker").Preload("Listener").Preload("Room").
		Order("timestamp DESC").Limit(400).Find(&messages).Error
	return messages, err

}

func (dm *DatabaseManager) GetMessageByID(msgID string) (*models.Message, error) {
	var message models.Message
	err := dm.DB.Preload("Talker").Preload("Listener").Preload("Room").
		First(&message, "msg_id = ?", msgID).Error
	if err != nil {

		return nil, err
	}
	return &message, nil
}

// 其他可能需要的数据库操作方法...

// 好友相关操作
func (dm *DatabaseManager) AddFriend(userID, friendID uint, alias string, isPrivate bool) error {
	userFriend := models.UserFriend{
		UserID:    userID,
		FriendID:  friendID,
		Alias:     alias,
		IsPrivate: isPrivate,
	}
	return dm.DB.Create(&userFriend).Error
}

func (dm *DatabaseManager) RemoveFriend(userID, friendID uint) error {
	return dm.DB.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&models.UserFriend{}).Error
}

func (dm *DatabaseManager) GetFriends(userID uint) ([]models.User, error) {
	var friends []models.User
	err := dm.DB.Joins("JOIN user_friends ON users.id = user_friends.friend_id").
		Where("user_friends.user_id = ?", userID).
		Find(&friends).Error
	return friends, err
}

func (dm *DatabaseManager) UpdateFriendAlias(userID, friendID uint, newAlias string) error {
	return dm.DB.Model(&models.UserFriend{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("alias", newAlias).Error
}

func (dm *DatabaseManager) SetFriendPrivacy(userID, friendID uint, isPrivate bool) error {
	return dm.DB.Model(&models.UserFriend{}).
		Where("user_id = ? AND friend_id = ?", userID, friendID).
		Update("is_private", isPrivate).Error
}

// 户间相关操作（新增和修改）
func (dm *DatabaseManager) AddUserToRoom(userID, roomID uint, alias string, isPrivate bool) error {
	userRoom := models.UserRoom{
		UserID:    userID,
		RoomID:    roomID,
		Alias:     alias,
		IsPrivate: isPrivate,
	}
	return dm.DB.Create(&userRoom).Error
}

func (dm *DatabaseManager) RemoveUserFromRoom(userID, roomID uint) error {
	return dm.DB.Where("user_id = ? AND room_id = ?", userID, roomID).Delete(&models.UserRoom{}).Error
}

func (dm *DatabaseManager) GetRoomMembers(roomID uint) ([]models.User, error) {
	var members []models.User
	err := dm.DB.Joins("JOIN user_rooms ON users.id = user_rooms.user_id").
		Where("user_rooms.room_id = ?", roomID).
		Find(&members).Error
	return members, err
}

func (dm *DatabaseManager) UpdateRoomAlias(userID, roomID uint, newAlias string) error {
	return dm.DB.Model(&models.UserRoom{}).
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("alias", newAlias).Error
}

func (dm *DatabaseManager) SetRoomPrivacy(userID, roomID uint, isPrivate bool) error {
	return dm.DB.Model(&models.UserRoom{}).
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("is_private", isPrivate).Error
}

func (dm *DatabaseManager) UpdateRoomMemberAlias(userID, roomID, alias string) error {
	return dm.DB.Model(&models.UserRoom{}).
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("alias", alias).Error
}

// 新增的方法
func (dm *DatabaseManager) SetRoomMemberPrivacy(userID, roomID string, isPrivate bool) error {
	return dm.DB.Model(&models.UserRoom{}).
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("is_private", isPrivate).Error
}
