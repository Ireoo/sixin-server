package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	WechatID string `gorm:"uniqueIndex"`
	Name     string
}

type Message struct {
	gorm.Model
	Text       string
	TalkerID   string
	ListenerID string
	RoomID     string
	Timestamp  int64
}

type Room struct {
	gorm.Model
	RoomID string `gorm:"uniqueIndex"`
	Topic  string
}
