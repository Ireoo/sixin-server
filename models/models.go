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
}

type Room struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	RoomID       string `gorm:"uniqueIndex"`
	Name         string
	OwnerID      string
	MemberIDList string `gorm:"type:json"`
	Avatar       string
	AdminIDList  string `gorm:"type:json"`
}

type RoomByUser struct {
	ID     uint `gorm:"primaryKey;autoIncrement"`
	UserID uint
	RoomID uint
	Alias  string
}

type Message struct {
	ID            uint   `gorm:"primaryKey;autoIncrement"`
	MsgID         string `gorm:"uniqueIndex"`
	TalkerID      uint
	ListenerID    uint
	Text          string `gorm:"type:json"`
	Timestamp     int64  // 时间戳
	Type          int
	RoomID        uint
	MentionIDList string `gorm:"type:json"`
}
