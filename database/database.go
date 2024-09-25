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
	"gorm.io/gorm/clause"
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

func (dm *DatabaseManager) GetUserByID(userId, id uint) (models.User, error) {
	// 根据userId获取用户好友id信息
	var userFriends []models.UserFriend
	err := dm.DB.Model(&models.UserFriend{}).Where("user_id = ?", userId).Find(&userFriends).Error
	if err != nil {
		return models.User{}, err
	}

	// 根据好友id获取用户信息
	var user models.User
	err = dm.DB.Model(&models.User{}).Where("id IN (?)", userFriends).Find(&user).Error

	return user, err
}

func (dm *DatabaseManager) CreateUser(user *models.User) error {
	// 生成密钥
	secretKey, err := generateSecretKey()
	if err != nil {
		return fmt.Errorf("无法生成密钥: %v", err)
	}
	user.SecretKey = secretKey

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("无法哈希密码: %v", err)
	}
	user.Password = string(hashedPassword)

	// 创建用户
	if err := dm.DB.Create(user).Error; err != nil {
		return fmt.Errorf("无法创建用户: %v", err)
	}

	return nil
}

func (dm *DatabaseManager) UpdateUser(userId, id uint, updatedUser models.UserFriend) error {
	// 根据userId获取用户好友id信息,然后修改用户这个好友信息
	var userFriends []models.UserFriend
	err := dm.DB.Model(&models.UserFriend{}).Where("user_id = ?", userId).Find(&userFriends).Error
	if err != nil {
		return err
	}
	// 用 updatedUser 更新 userFriends 中的 friendID 对应的用户信息
	for _, userFriend := range userFriends {
		if userFriend.FriendID == updatedUser.FriendID {
			userFriend.Alias = updatedUser.Alias
			userFriend.IsPrivate = updatedUser.IsPrivate
			err = dm.DB.Model(&models.UserFriend{}).Where("user_id = ? AND friend_id = ?", userId, updatedUser.FriendID).Updates(userFriend).Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dm *DatabaseManager) UpdateUserOwn(userId uint, updatedUser *models.User) error {
	// 先通过userid获取自己的信息
	var existingUser models.User
	if err := dm.DB.First(&existingUser, userId).Error; err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 删除 updatedUser 中的敏感信息
	updatedUser.ID = userId
	updatedUser.Password = ""  // 不允许通过此方法更新密码
	updatedUser.SecretKey = "" // 不允许更新密钥

	// 根据userId修改用户自己的信息updatedUser
	result := dm.DB.Model(&existingUser).Updates(updatedUser)
	if result.Error != nil {
		return fmt.Errorf("更新用户信息失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到ID为%d的用户", userId)
	}
	return nil
}

func (dm *DatabaseManager) DeleteUser(id uint) error {
	return dm.DB.Delete(&models.User{}, id).Error
}

func (dm *DatabaseManager) DeleteUserFriend(userID, friendID uint) error {
	return dm.DB.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&models.UserFriend{}).Error
}

// 户间相关操作
func (dm *DatabaseManager) GetAllRooms() ([]models.Room, error) {
	var rooms []models.Room
	err := dm.DB.Preload("Owner").Preload("Members").Find(&rooms).Error
	return rooms, err
}

func (dm *DatabaseManager) GetRoomByID(userId, id uint) (models.Room, error) {
	// 获取userid
	var userRooms []models.UserRoom
	err := dm.DB.Model(&models.UserRoom{}).Where("user_id = ? AND room_id = ?", userId, id).Find(&userRooms).Error
	if err != nil {
		return models.Room{}, err
	}

	// 获取room信息
	var room models.Room
	err = dm.DB.Model(&models.Room{}).Where("id = ?", id).First(&room).Error
	if err != nil {
		return models.Room{}, err
	}

	return room, nil
}

func (dm *DatabaseManager) CreateRoom(room *models.Room) error {
	return dm.DB.Create(room).Error
}

func (dm *DatabaseManager) UpdateRoom(userId, id uint, updatedRoom models.UserRoom) error {
	// 根据userid获取用户房间id信息,然后修改用户这个房间信息
	var userRooms []models.UserRoom
	err := dm.DB.Model(&models.UserRoom{}).Where("user_id = ?", userId).Find(&userRooms).Error
	if err != nil {
		return err
	}
	// 用 updatedUser 更新 userFriends 中的 friendID 对应的用户信息
	for _, userRoom := range userRooms {
		if userRoom.RoomID == id {
			userRoom.Alias = updatedRoom.Alias
			userRoom.IsPrivate = updatedRoom.IsPrivate
			err = dm.DB.Model(&models.UserRoom{}).Where("user_id = ? AND room_id = ?", userId, id).Updates(userRoom).Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dm *DatabaseManager) UpdateRoomByOwner(userId, id uint, updatedRoom models.Room) error {
	// 根据userid是ownerid是否是这个房间的管理者，如果是就修改用户这个房间信息
	var room models.Room
	err := dm.DB.Model(&models.Room{}).Where("owner_id = ? AND id = ?", userId, id).First(&room).Error
	if err != nil {
		return err
	}

	updatedRoom.OwnerID = userId
	updatedRoom.Members = room.Members

	return dm.DB.Model(&models.Room{}).Where("owner_id = ? AND id = ?", userId, id).Updates(updatedRoom).Error
}

func (dm *DatabaseManager) DeleteRoom(userId, id uint) error {
	// 根据userid获取用户房间id信息,然后删除用户这个房间信息
	var userRooms []models.UserRoom
	err := dm.DB.Model(&models.UserRoom{}).Where("user_id = ?", userId).Find(&userRooms).Error
	if err != nil {
		return err
	}
	for _, userRoom := range userRooms {
		if userRoom.RoomID == id {
			return dm.DB.Delete(&models.UserRoom{}, userRoom.ID).Error
		}
	}
	return nil
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

func (dm *DatabaseManager) GetChats(userID uint) ([]models.Message, error) {
	var messages []models.Message
	err := dm.DB.Model(&models.Message{}).
		Preload("Talker").Preload("Listener").Preload("Room").
		Joins("LEFT JOIN user_rooms ON messages.room_id = user_rooms.room_id AND user_rooms.user_id = ?", userID).
		Where("messages.talker_id = ? OR messages.listener_id = ? OR user_rooms.user_id IS NOT NULL", userID, userID).
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
	err := dm.DB.Model(&models.User{}).
		Joins("JOIN user_rooms ON users.id = user_rooms.user_id").
		Where("user_rooms.id = ?", roomID).
		Find(&members).Error
	return members, err
}

func (dm *DatabaseManager) UpdateRoomAlias(userID, roomID uint, newAlias string) error {
	return dm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "room_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"alias"}),
	}).Create(&models.UserRoom{
		UserID: userID,
		RoomID: roomID,
		Alias:  newAlias,
	}).Error
}

func (dm *DatabaseManager) SetRoomPrivacy(userID, roomID uint, isPrivate bool) error {
	return dm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "room_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_private"}),
	}).Create(&models.UserRoom{
		UserID:    userID,
		RoomID:    roomID,
		IsPrivate: isPrivate,
	}).Error
}

func (dm *DatabaseManager) UpdateRoomMemberAlias(userID, roomID uint, alias string) error {
	return dm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "room_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"alias"}),
	}).Create(&models.UserRoom{
		UserID: userID,
		RoomID: roomID,
		Alias:  alias,
	}).Error
}

// 新增的方法
func (dm *DatabaseManager) SetRoomMemberPrivacy(userID, roomID uint, isPrivate bool) error {
	return dm.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "room_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_private"}),
	}).Create(&models.UserRoom{
		UserID:    userID,
		RoomID:    roomID,
		IsPrivate: isPrivate,
	}).Error
}

// 辅助函数：生成密钥
func generateSecretKey() (string, error) {
	key := make([]byte, 32) // 256位密钥
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

// 用户资料更新功能
func (dm *DatabaseManager) UpdateUserProfile(userID uint, updates map[string]interface{}) error {
	return dm.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (dm *DatabaseManager) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := dm.DB.Model(&models.User{}).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (dm *DatabaseManager) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := dm.DB.Model(&models.User{}).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (dm *DatabaseManager) GetUserByWechatID(wechatID string) (*models.User, error) {
	var user models.User
	err := dm.DB.Model(&models.User{}).Where("wechat_id = ?", wechatID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// 新增的方法
func (dm *DatabaseManager) GetRoomAliasByUsers(userID, roomID uint) (map[uint]string, error) {
	var userRooms []models.UserRoom
	aliases := make(map[uint]string)

	// 修正查询条件并添加 roomID 过滤
	if err := dm.DB.Model(&models.UserRoom{}).
		Where("user_id = ? AND id = ?", userID, roomID).
		Find(&userRooms).Error; err != nil {
		return nil, err
	}

	for _, userRoom := range userRooms {
		aliases[userRoom.UserID] = userRoom.Alias
	}

	return aliases, nil
}

func (dm *DatabaseManager) CheckUserRoom(userID, roomID uint) error {
	var userRoom models.UserRoom
	if err := dm.DB.Model(&models.UserRoom{}).
		Where("user_id = ? AND id = ?", userID, roomID).
		First(&userRoom).Error; err != nil {
		return err
	}
	return nil
}

func (dm *DatabaseManager) GetRooms(userID uint) ([]models.Room, error) {
	var rooms []models.Room
	err := dm.DB.Model(&models.Room{}).Where("user_id = ?", userID).Find(&rooms).Error
	return rooms, err
}

func (dm *DatabaseManager) GetUsers(userID uint) ([]models.User, error) {
	var users []models.User
	err := dm.DB.Model(&models.User{}).Where("id = ?", userID).Find(&users).Error
	return users, err
}
