package models

import (
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/sirupsen/logrus"
)

var (
	MarketDay       uint32 = 1
	IsChallengeOpen bool   = false
)

type DailyChallenge struct {
	Id            uint32 `gorm:"column:id;primary_key;AUTO_INCREMENT" json:"id"`
	MarketDay     uint32 `gorm:"column:marketDay;not null" json:"market_day"`
	ChallengeType string `gorm:"column:challengeType;not null" json:"challenge_type"`
	Value         uint64 `gorm:"column:value;not null" json:"value"`
	StockId       uint32 `gorm:"column:stockId; default null" json:"stock_id"`
}

func (DailyChallenge) TableName() string {
	return "DailyChallenge"
}

func (d *DailyChallenge) ToProto() *models_pb.DailyChallenge {
	pDailyChallenge := &models_pb.DailyChallenge{
		ChallengeId:   d.Id,
		MarketDay:     d.MarketDay,
		ChallengeType: d.ChallengeType,
		Value:         d.Value,
		StockId:       d.StockId,
	}
	return pDailyChallenge
}

func GetDailyChallenges() ([]*DailyChallenge, error) {

	l := logger.WithFields(logrus.Fields{
		"method":    "GetDailyChallenges",
		"marketDay": MarketDay,
	})

	var dailyChallenges []*DailyChallenge

	l.Infof("Attempting to get dailyChallenges")

	db := getDB()

	if err := db.Table("DailyChallenge").Where("marketDay = ?", MarketDay).Find(&dailyChallenges).Error; err != nil {
		l.Errorf("error while querying from db")
		return nil, err
	}

	l.Infof("Successfully fetched dailyChallenges")
	return dailyChallenges, nil
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
		StockId:       stockId,
	}

	if stockId == 0 {
		if err := db.Table("DailyChallenge").Omit("StockId").Save(dailyChallenge).Error; err != nil {
			l.Error(err)
			return err
		}
	} else {
		if err := db.Table("DailyChallenge").Save(dailyChallenge).Error; err != nil {
			l.Error(err)
			return err
		}
	}

	l.Infof("successfully added daily challenge")

	return nil
}

func OpenDailyChallenge() error {
	//TODO: open dailyChallenge
	//and save user state for later computation
	IsChallengeOpen = true
	return nil
}

func CloseDailyChallenge() error {
	//TODO: close DailyChallenge
	// and compute dailychallenges
	IsChallengeOpen = false

	return nil
}
