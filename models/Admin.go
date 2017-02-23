package models

import (
	"github.com/Sirupsen/logrus"
)

func IsAdmin(username, password string) (bool, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "IsAdmin",
	})

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return false, err
	}
	defer db.Close()
	db.LogMode(true)

	sql := "Select * from Admins where username = ? and password = MD5(?)"
	db = db.Exec(sql, username, password)
	if err := db.Error; err != nil {
		return false, err
	}
	return true, err
}

func AdminLog(username, msg string) {
	var l = logger.WithFields(logrus.Fields{
		"method":         "AdminLog",
		"param_username": username,
		"param_msg":      msg,
	})

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return
	}
	defer db.Close()
	db.LogMode(true)

	sql := "Insert into AdminLogs (username, msg) Values(?, ?)"
	if err := db.Exec(sql, username, msg).Error; err != nil {
		l.Error(err)
		return
	}

	l.Infof("Executed")
}
