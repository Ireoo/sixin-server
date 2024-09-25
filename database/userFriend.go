package database

import "github.com/Ireoo/sixin-server/models"

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

func (dm *DatabaseManager) DeleteUserFriend(userID, friendID uint) error {
	return dm.DB.Where("user_id = ? AND friend_id = ?", userID, friendID).Delete(&models.UserFriend{}).Error
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
