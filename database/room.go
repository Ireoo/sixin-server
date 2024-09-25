package database

import (
	"github.com/Ireoo/sixin-server/models"
)

func (dm *DatabaseManager) GetRoomMembers(roomID uint) ([]models.User, error) {
	var members []models.User
	err := dm.DB.Model(&models.User{}).
		Joins("JOIN user_rooms ON users.id = user_rooms.user_id").
		Where("user_rooms.id = ?", roomID).
		Find(&members).Error
	return members, err
}

func (dm *DatabaseManager) GetRooms(userID uint) ([]models.Room, error) {
	var rooms []models.Room
	err := dm.DB.Model(&models.Room{}).Where("user_id = ?", userID).Find(&rooms).Error
	return rooms, err
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

func (dm *DatabaseManager) CreateRoom(room *models.Room) error {
	return dm.DB.Model(&models.Room{}).Create(room).Error
}

func (dm *DatabaseManager) GetAllRooms() ([]models.Room, error) {
	var rooms []models.Room
	err := dm.DB.Preload("Owner").Preload("Members").Find(&rooms).Error
	return rooms, err
}
