package database

import (
	"github.com/Ireoo/sixin-server/models"
	"gorm.io/gorm/clause"
)

func (dm *DatabaseManager) JoinRoom(userID uint, roomID uint, alias string) error {
	// 新增 conflicts 处理，解决重复插入数据问题
	return dm.DB.Model(&models.UserRoom{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "room_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"alias"}),
	}).Create(&models.UserRoom{
		UserID: userID,
		RoomID: roomID,
		Alias:  alias,
	}).Error
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

func (dm *DatabaseManager) SetRoomMemberPrivacy(userID, roomID uint, isPrivate bool) error {
	return dm.DB.Model(&models.UserRoom{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "room_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_private"}),
	}).Create(&models.UserRoom{
		UserID:    userID,
		RoomID:    roomID,
		IsPrivate: isPrivate,
	}).Error
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
	return dm.DB.Model(&models.UserRoom{}).Where("user_id = ? AND room_id = ?", userID, roomID).Delete(&models.UserRoom{}).Error
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
