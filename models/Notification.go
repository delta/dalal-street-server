package models

import (
	"github.com/Sirupsen/logrus"
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

func GetNotifications(lastId, count uint32) (bool, map[uint32]*Notification, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetNotifications",
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get notifications")

	db, err := DbOpen()
	if err != nil {
		return true, nil, err
	}
	defer db.Close()

	var notifications []*Notification

	//set default value of count if it is zero
	if count == 0 {
		count = GET_NOTIFICATION_COUNT
	} else {
		count = min(count, GET_NOTIFICATION_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Order("desc id").Limit(count).Find(&notifications).Error; err != nil {
		return true, nil, err
	}

	notificationsMap := make(map[uint32]*Notification)

	for _, notification := range notifications {
		notificationsMap[notification.Id] = notification
	}

	var moreExists = len(notifications) < int(count)
	l.Infof("Successfully fetched notifications")
	return moreExists, notificationsMap, nil
}
