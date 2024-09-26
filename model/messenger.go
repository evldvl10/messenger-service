package model

import "gorm.io/gorm"

type MessengerDialog struct {
	gorm.Model
	OwnerID   int
	UserID    int
	MessageID int
	Owner     User             `gorm:"not null; foreignKey:OwnerID" json:"owner"`
	User      User             `gorm:"not null; foreignKey:UserID" json:"user"`
	Message   MessengerMessage `gorm:"not null; foreignKey:MessageID" json:"message"`
}

type MessengerMessage struct {
	gorm.Model
	FromID   int
	ToID     int
	From     User   `gorm:"not null; foreignKey:FromID" json:"from"`
	To       User   `gorm:"not null; foreignKey:ToID" json:"to"`
	Type     string `gorm:"not null" json:"type"`
	Metadata string `gorm:"not null" json:"metadata"`
	Data     string `gorm:"not null" json:"data"`
	DialogID uint   `gorm:"not null; default:0" json:"dialog_id"`
	Read     bool   `gorm:"not null" json:"read"`
}

type MessengerImage struct {
	gorm.Model
	Data string `gorm:"not null" json:"data"`
}
