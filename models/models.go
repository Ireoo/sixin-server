package models

import (
	"time"

	"gorm.io/gorm"
)

func GetAllModels() []interface{} {
	return []interface{}{
		&User{},
		&Room{},
		&RoomByUser{},
		&Message{},
		&UserFriend{}, // 新增
		&UserRoom{},   // 新增
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
	Messages  []Message `gorm:"foreignKey:TalkerID"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Room struct {
	gorm.Model
	RoomID  string `gorm:"uniqueIndex;not null"`
	Name    string
	OwnerID uint
	Owner   User   `gorm:"foreignKey:OwnerID"`
	Members []User `gorm:"many2many:room_members;"`
	Avatar  string
	// 定义管理员与用户的多对多关系
	Admins []*User `gorm:"many2many:room_admins;"`
	// 定义与 Message 的一对多关系
	Messages  []Message `gorm:"foreignKey:RoomID"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RoomByUser struct {
	gorm.Model
	UserID    uint `gorm:"not null"`
	RoomID    uint `gorm:"not null"`
	Alias     string
	User      *User     `gorm:"foreignKey:UserID"`
	Room      *Room     `gorm:"foreignKey:RoomID"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Message struct {
	gorm.Model
	ID            uint                   `gorm:"primaryKey" json:"id"`
	MsgID         string                 `gorm:"uniqueIndex" json:"msgId"`
	TalkerID      uint                   `json:"talkerId"`
	ListenerID    uint                   `json:"listenerId"`
	RoomID        uint                   `json:"roomId"`
	Text          map[string]interface{} `gorm:"type:json" json:"text"`
	Timestamp     int64                  `json:"timestamp"`
	Type          int                    `json:"type"`
	MentionIDList []uint                 `gorm:"type:json" json:"mentionIdList"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// 新增 UserFriend 结构体
type UserFriend struct {
	gorm.Model
	UserID    uint  `gorm:"not null"`
	FriendID  uint  `gorm:"not null"`
	User      *User `gorm:"foreignKey:UserID"`
	Friend    *User `gorm:"foreignKey:FriendID"`
	Alias     string
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// 新增 UserRoom 结构体
type UserRoom struct {
	gorm.Model
	UserID    uint  `gorm:"not null"`
	RoomID    uint  `gorm:"not null"`
	User      *User `gorm:"foreignKey:UserID"`
	Room      *Room `gorm:"foreignKey:RoomID"`
	Alias     string
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FullMessage struct {
	Message
	Talker   *User `json:"talker,omitempty"`
	Listener *User `json:"listener,omitempty"`
	Room     *Room `json:"room,omitempty"`
}
