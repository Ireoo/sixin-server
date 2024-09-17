package models

import (
	"gorm.io/gorm"
)

func GetAllModels() []interface{} {
	return []interface{}{
		&User{},
		&Room{},
		&RoomByUser{},
		&Message{},
		// 在这里添加新模型
	}
}

type User struct {
	gorm.Model
	WechatID  string `gorm:"uniqueIndex;not null"`
	Name      string
	Phone     map[string]string `gorm:"type:json"`
	Province  string
	Signature string
	Type      int
	Weixin    string
	Alias     string
	Avatar    string
	City      string
	Friend    bool
	Gender    string
	// 定义与 Room 的多对多关系
	Rooms []*Room `gorm:"many2many:user_rooms;"`
	// 定义与 Message 的一对多关系
	Messages []Message `gorm:"foreignKey:TalkerID"`
}

type Room struct {
	gorm.Model
	RoomID  string `gorm:"uniqueIndex;not null"`
	Name    string
	OwnerID uint
	Owner   *User `gorm:"foreignKey:OwnerID"`
	// 定义与 User 的多对多关系
	Members []*User `gorm:"many2many:user_rooms;"`
	Avatar  string
	// 定义管理员与用户的多对多关系
	Admins []*User `gorm:"many2many:room_admins;"`
	// 定义与 Message 的一对多关系
	Messages []Message `gorm:"foreignKey:RoomID"`
}

type RoomByUser struct {
	gorm.Model
	UserID uint `gorm:"not null"`
	RoomID uint `gorm:"not null"`
	Alias  string
	User   *User `gorm:"foreignKey:UserID"`
	Room   *Room `gorm:"foreignKey:RoomID"`
}

type Message struct {
	gorm.Model
	MsgID         string                 `gorm:"uniqueIndex;not null"`
	TalkerID      uint                   `gorm:"not null"` // 发送者 ID
	Talker        *User                  `gorm:"foreignKey:TalkerID"`
	ListenerID    uint                   // 接收者 ID
	Listener      *User                  `gorm:"foreignKey:ListenerID"`
	Text          map[string]interface{} `gorm:"type:json"`
	Timestamp     int64                  // 时间戳
	Type          int
	RoomID        uint   // 如果消息在群里
	Room          *Room  `gorm:"foreignKey:RoomID"`
	MentionIDList []uint `gorm:"type:json"`
}
