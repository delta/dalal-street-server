package models

import (
	"github.com/sirupsen/logrus"
	"encoding/json"
)

type UserSubscription struct {
	ID uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserID uint32 `gorm:"column:userId; not null" json:"user_id"`
	EndPoint string `gorm:"column:endpoint; not null" json:"end_point"`
	P256dh string `gorm:"column:p256dh; not null" json:"p256dh"`
	Auth string `gorm:"column:auth; not null" json:"auth"`
}


func (UserSubscription) TableName() string {
	return "UserSubscription"
}

// Adds the subscription keys for sending push notifications
func AddUserSubscription(email string ,data string) error {

	var l = logger.WithFields(logrus.Fields{
		"method":      "Add User Subscription",
		"param_email": email,
		"param_data":  data,
	})

	l.Infof("Add User subscription details requsted")

	db := getDB()

	user := User{
		Email : email,
	}

	err := db.Table("Users").Where("email = ?",email).First(&user).Error; 

	if err != nil {
	   l.Errorf("User not found in Database")
	   return UserNotFoundError
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(data),&result)

	keys := result["keys"].(map[string]interface{})

	userSubscription := &UserSubscription{
		UserID :   user.Id,
		EndPoint : result["endpoint"].(string),
		P256dh : keys["p256dh"].(string),
		Auth : keys["auth"].(string),
	}   

	if err := db.Table("UserSubscription").Save(userSubscription).Error; err != nil {
		l.Errorf("Error saving User Subscription. Failing. %+v",err)
		return InternalServerError
	}

	return nil
}
