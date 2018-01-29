package models

import (
	"github.com/Sirupsen/logrus"
)

func IsAdmin(username, password string) (bool, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "IsAdmin",
	})

	db := getDB()

	sql := "Select * from Admins where username = ? and password = MD5(?)"
	db = db.Exec(sql, username, password)
	if err := db.Error; err != nil {
		l.Errorf("Error checking if user is admin: %+v", err)
		return false, err
	}
	return true, nil
}

func AdminLog(username, msg string) {
	var l = logger.WithFields(logrus.Fields{
		"method":         "AdminLog",
		"param_username": username,
		"param_msg":      msg,
	})

	db := getDB()

	sql := "Insert into AdminLogs (username, msg) Values(?, ?)"
	if err := db.Exec(sql, username, msg).Error; err != nil {
		l.Error(err)
		return
	}

	l.Infof("Executed")
}
