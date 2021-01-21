package models

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	MarketDay       = 1
	isChallengeOpen = false
)

type DailyChallenge struct {
	Id            uint32 `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	MarketDay     uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	ChallengeType string `gorm:"column:challengeType;not null" json:"challenge_type"`
	Value         uint64 `gorm:"column:value;not null" json:"value"`
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

// add daily challenge to db
func AddDailyChallenge(value uint64, marketDay uint32, stockId uint32, challengeType string) error {
	l := logger.WithFields(logrus.Fields{
		"method":              "AddDailyChallenge",
		"param_day":           MarketDay,
		"param_value":         value,
		"param_stockid":       stockId,
		"param_challengeType": challengeType,
	})

	db := getDB()

	l.Infof("Attempting to save daily challenge")

	dailyChallenge := &DailyChallenge{
		MarketDay:     marketDay,
		ChallengeType: challengeType,
		Value:         value,
	}

	if stockId != 0 {
		dailyChallenge.StockId = stockId
	}

	if err := db.Save(dailyChallenge).Error; err != nil {
		l.Error(err)
		return err
	}

	l.Infof("successfully added daily challenge")

	return nil
}
