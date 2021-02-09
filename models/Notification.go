package models

import (
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

type Notification struct {
	Id          uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId      uint32 `gorm:"column:userId;not null" json:"user_id"`
	Text        string `gorm:"column:text" json:"text"`
	IsBroadcast bool   `gorm:"column:isBroadcast" json:"is_broadcast"`
	CreatedAt   string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (Notification) TableName() string {
	return "Notifications"
}

func (gNotification *Notification) ToProto() *models_pb.Notification {
	return &models_pb.Notification{
		Id:          gNotification.Id,
		UserId:      gNotification.UserId,
		Text:        gNotification.Text,
		IsBroadcast: gNotification.IsBroadcast,
		CreatedAt:   gNotification.CreatedAt,
	}
}

func GetNotifications(userId, lastId, count uint32) (bool, []*Notification, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "GetNotifications",
		"lastId": lastId,
		"count":  count,
	})

	l.Infof("Attempting to get notifications")

	db := getDB()

	var notifications []*Notification

	//set default value of count if it is zero
	if count == 0 {
		count = GET_NOTIFICATION_COUNT
	} else {
		count = utils.MinInt32(count, GET_NOTIFICATION_COUNT)
	}

	//get latest events if lastId is zero
	if lastId != 0 {
		// 0 means broadcast!
		db = db.Where("id <= ?", lastId)
	}
	if err := db.Where("isBroadcast = true or userId = ?", userId).Order("id desc").Limit(count).Find(&notifications).Error; err != nil {
		return true, nil, err
	}

	var moreExists = len(notifications) >= int(count)
	l.Infof("Successfully fetched notifications")
	return moreExists, notifications, nil
}

func SendNotification(userId uint32, text string, isBroadcast bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":            "SendNotification",
		"param_userId":      userId,
		"param_text":        text,
		"param_isBroadcast": isBroadcast,
	})

	l.Infof("Sending notification")

	db := getDB()

	n := &Notification{
		UserId:      userId,
		Text:        text,
		IsBroadcast: isBroadcast,
		CreatedAt:   utils.GetCurrentTimeISO8601(),
	}

	if err := db.Save(n).Error; err != nil {
		l.Error(err)
		return err
	}

	notificationsStream := datastreamsManager.GetNotificationsStream()
	notificationsStream.SendNotification(n.ToProto())

	err := SendPushNotification(userId, PushNotification{Title: "Message from Dalal Street", Message: text})

	if err != nil {
		l.Errorf("Couldn't send push notification, ", err)
	}

	l.Infof("Done")

	return nil
}
