package models

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

type DailyChallenge struct {
	Id            uint32 `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	MarketDay     uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	ChallengeType string `gorm:"column:challengeType;not null" json:"challenge_type"`
	Value         uint32 `gorm:"column:value;not null" json:"value"`
	StockId       uint32 `gorm:"column:stockIdnot null" json:"stock_id"`
}

func (DailyChallenge) TableName() string {
	return "DailyChallenge"
}

func GetDailyChallenge(day uint32) ([]DailyChallenge, error) {

	l := logger.WithFields(logrus.Fields{
		"method":    "GetDailyChallenge",
		"marketDay": day,
	})

	var dailyChallenge []DailyChallenge

	if day == 0 {
		return nil, errors.New("incorrect day")
	}

	l.Infof("Attempting to get dailyChallenges")

	db := getDB()

	if err := db.Where("marketDay = ?", day).Find(&dailyChallenge).Error; err != nil {
		fmt.Println(err)
		return nil, err
	}

	l.Infof("Successfully fetched dailyChallenges")
	return dailyChallenge, nil
}

//saves registered users state when market opens
func SaveUsersState() error {
	l := logger.WithFields(logrus.Fields{
		"method": "SaveUsersState",
	})
	l.Infof("Attempting to save user state")
	//TODO: save user state

	return nil

}
