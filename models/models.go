package models

type User struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	WechatID  string `gorm:"uniqueIndex"`
	Name      string
	Phone     string `gorm:"type:json"`
	Province  string
	Signature string
	Type      int
	Weixin    string
	Alias     string
	Avatar    string
	City      string
	Friend    bool
	Gender    string
	UpdatedAt int64
	CreatedAt int64
}

type Room struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	RoomID       string `gorm:"uniqueIndex"`
	Topic        string
	OwnerID      string
	MemberIDList string `gorm:"type:json"`
	Avatar       string
	AdminIDList  string `gorm:"type:json"`
	UpdatedAt    int64
	CreatedAt    int64
}

type RoomByUser struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	Name      string
	Alias     string
	Topic     string
	UpdatedAt int64
	CreatedAt int64
}

type Message struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	MsgID         string `gorm:"uniqueIndex"`
	TalkerID      string
	ListenerID    string
	Text          string `gorm:"type:text"`
	Timestamp     int64
	Type          int
	RoomID        string
	MentionIDList string `gorm:"type:json"`
	UpdatedAt     int64
	CreatedAt     int64
}
