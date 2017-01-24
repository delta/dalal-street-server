package models

import (
	"sync"
	"fmt"
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

// User models the User object.
type User struct {
	Id        uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	Email     string `gorm:"unique;not null"`
	Name      string `gorm:"not null"`
	Cash      int32  `gorm:"not null"`
	Total     int32  `gorm:"not null"`
	CreatedAt string `gorm:"column:createdAt;not null"`
}

// User.TableName() is for letting Gorm know the correct table name.
func (User) TableName() string {
	return "Users"
}

var userLogger *logrus.Entry

func InitUsers() {
	userLogger = utils.Logger.WithFields(logrus.Fields{
		"module": "models/User",
	})
}

// CreateUser() creates a user given his email and name.
func CreateUser(email, name string) (*User, error) {
	var l = userLogger.WithFields(logrus.Fields{
		"method": "CreateUser",
		"email" : email,
		"name"  : name,
	})

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	u := &User{
		Email: email,
		Name : name,
		Cash : STARTING_CASH,
		Total: STARTING_CASH,
	}
	l.Debugf("Creating user")

	err = db.Create(u).Error

	if err != nil {
		l.Error(err)
		return nil, err
	}

	l.Infof("Created user. UserId: %d", u.Id)
	return u, nil
}
