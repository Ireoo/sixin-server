package database

import (
	"github.com/Ireoo/sixin-server/models"
)

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

func (dm *DatabaseManager) GetMessageByID(msgID string) (*models.Message, error) {
	var message models.Message
	err := dm.DB.Preload("Talker").Preload("Listener").Preload("Room").
		First(&message, "msg_id = ?", msgID).Error
	if err != nil {

		return nil, err
	}
	return &message, nil
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
