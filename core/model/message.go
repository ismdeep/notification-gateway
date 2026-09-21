package model

import "time"

type Message struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SenderName      string    `gorm:"column:sender_name;type:varchar(255);not null" json:"sender_name"`
	ClientMessageID string    `gorm:"column:client_message_id;type:varchar(255);not null" json:"client_message_id"`
	Title           string    `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Content         string    `gorm:"column:content;type:text;not null" json:"content"`
	SendStatus      int       `gorm:"column:send_status;default:0" json:"send_status"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

const (
	MessageSendSuccess = 1
	MessageSendFail    = 2
)

func (receiver *Message) TableName() string {
	return "messages"
}
