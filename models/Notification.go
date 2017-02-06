package models

import (
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type Notification struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId    uint32 `gorm:"column:userId;not null" json:"user_id"`
	Text      string `gorm:"column:text" json:"text"`
	CreatedAt string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (Notification) TableName() string {
	return "Notifications"
}

func (gNotification *Notification) ToProto() *models_proto.Notification {
	return &models_proto.Notification{
		Id:        gNotification.Id,
		UserId:    gNotification.UserId,
		Text:      gNotification.Text,
		CreatedAt: gNotification.CreatedAt,
	}
}
