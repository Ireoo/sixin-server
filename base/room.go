package base

import (
	"github.com/Ireoo/sixin-server/models"
)

type RoomHandler struct {
	Base *Base
}

func NewRoomHandler(base *Base) *RoomHandler {
	return &RoomHandler{Base: base}
}

func (rh *RoomHandler) GetRooms() ([]models.Room, error) {
	var rooms []models.Room
	if err := rh.Base.DB.Preload("Owner").Preload("Members").Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

func (rh *RoomHandler) CreateRoom(room *models.Room) error {
	return rh.Base.DB.Create(room).Error
}

func (rh *RoomHandler) GetRoom(id string) (*models.Room, error) {
	var room models.Room
	if err := rh.Base.DB.Preload("Owner").Preload("Members").First(&room, id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (rh *RoomHandler) UpdateRoom(id string, updatedRoom *models.Room) error {
	var room models.Room
	if err := rh.Base.DB.First(&room, id).Error; err != nil {
		return err
	}
	return rh.Base.DB.Model(&room).Updates(updatedRoom).Error
}

func (rh *RoomHandler) DeleteRoom(id string) error {
	return rh.Base.DB.Delete(&models.Room{}, id).Error
}

func (rh *RoomHandler) GetRoomByUsers(userIDs []uint) (*models.Room, error) {
	// 实现获取用户房间的逻辑
	// 这里需要根据您的具体需求来实现
	return nil, nil
}
