package models

import (
	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
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

	db, err := DbOpen()
	if err != nil {
		return true, nil, err
	}
	defer db.Close()
	db.LogMode(true)

	var notifications []*Notification

	//set default value of count if it is zero
	if count == 0 {
		count = GET_NOTIFICATION_COUNT
	} else {
		count = utils.MinInt(count, GET_NOTIFICATION_COUNT)
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

	db, err := DbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	n := &Notification{
		UserId:      userId,
		Text:        text,
		IsBroadcast: isBroadcast,
	}

	if err := db.Save(n).Error; err != nil {
		l.Error(err)
		return err
	}

	datastreams.SendNotification(n.ToProto())

	l.Infof("Done")

	return nil
}
