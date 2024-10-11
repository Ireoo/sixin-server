package database

import (
	"fmt"

	"github.com/Ireoo/sixin-server/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

func (dm *DatabaseManager) GetUsers(userID uint) ([]models.User, error) {
	var users []models.User
	err := dm.DB.Model(&models.User{}).Where("id = ?", userID).Find(&users).Error
	return users, err
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
	result := dm.DB.Model(existingUser).Updates(updatedUser)
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

func (dm *DatabaseManager) GetUserInfo(userId uint) (models.User, error) {
	// 根据好友id获取用户信息
	var user models.User
	err := dm.DB.Model(&models.User{}).Where("id = ?", userId).Find(&user).Error

	return user, err
}
