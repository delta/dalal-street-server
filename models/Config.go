package models

import (
	"github.com/sirupsen/logrus"
)

// this files handles config for daily challenges

// Config model
type Config struct {
	Id                   uint32 `gorm:"column:id;primary_key;not null" json:"id"`
	IsDailyChallengeOpen bool   `gorm:"column:isDailyChallengeOpen;default false not null" json:"is_dailychallengeopen"`
	MarketDay            uint32 `gorm:"column:marketDay;not null default 0 unsigned" json:"market_day"`
	IsMarketOpen         bool   `gorm:"column:isMarketOpen;not null default 0 unsigned" json:"is_marketopen"`
}

//ConfigDataInit init initial values for Config table
func ConfigDataInit() {
	l := logger.WithFields(logrus.Fields{
		"method": "ConfigDataInit",
	})

	l.Debugf("Invoked")

	db := getDB()

	//begin transaction
	tx := db.Begin()

	query := "SELECT id FROM `Config` WHERE id = ?"

	row := tx.Raw(query, 1).Row()

	var id uint32

	row.Scan(&id)

	if id == 0 {
		config := &Config{
			Id:                   1,
			IsDailyChallengeOpen: false,
			MarketDay:            0,
			IsMarketOpen:         false,
		}

		if err := tx.Table("Config").Save(&config).Error; err != nil {
			l.Errorf("failed %+e", err)
			tx.Rollback()
		}

		if err := tx.Commit().Error; err != nil {
			l.Errorf("Error %+e", err)
		}

	}

	l.Debugf("Done")

}

//IsDailyChallengeOpen returns dailychallenge status
func IsDailyChallengeOpen() bool {
	l := logger.WithFields(logrus.Fields{
		"method": "IsDailyChallengeOpen",
	})

	l.Debugf("Invoked")

	db := getDB()

	queryData := &Config{}

	if err := db.Table("Config").Select("isDailyChallengeOpen").First(queryData).Error; err != nil {
		l.Error(err)
	}

	l.Debugf("Done")

	return queryData.IsDailyChallengeOpen

}

// SetIsDailyChallengeOpen update IsDailyChallengeOpen in db
func SetIsDailyChallengeOpen(challengeStatus bool) error {
	l := logger.WithFields(logrus.Fields{
		"method": "setIsDailyChallengeOpen",
	})
	l.Debugf("Invoked")

	db := getDB()

	if err := db.Table("Config").Update("isDailyChallengeOpen", challengeStatus).Error; err != nil {
		l.Errorf("failed updating IsDailyChallengeOpen date %+e", err)
		return err
	}

	l.Debugf("Done")

	return nil

}

//GetMarketDay returns current marketday
func GetMarketDay() uint32 {
	l := logger.WithFields(logrus.Fields{
		"method": "GetMarketDay",
	})
	l.Debugf("Invoked")

	db := getDB()

	queryData := &Config{}

	if err := db.Table("Config").Select("marketDay").First(queryData).Error; err != nil {
		l.Error(err)
	}

	l.Debugf("Done")

	return queryData.MarketDay

}

//SetMarketDay updates marketday in db
func SetMarketDay(marketDay uint32) error {
	l := logger.WithFields(logrus.Fields{
		"method":     "SetMarketDay",
		"market_day": marketDay,
	})
	l.Debugf("invoked")

	db := getDB()

	if err := db.Table("Config").Update("marketDay", marketDay).Error; err != nil {
		l.Errorf("failed updating market date %+e", err)
		return err
	}

	l.Debugf("Done")

	return nil

}

// GetDailyChallengeConfig returns DailyChallengeConfig
func GetDailyChallengeConfig() (*Config, uint32, error) {
	l := logger.WithFields(logrus.Fields{
		"method": "GetDailyChallengeConfig",
	})

	l.Debugf("requested")

	totalMarketDays := config.TotalMarketDays

	config := &Config{}

	db := getDB()

	if err := db.Table("Config").First(&config).Error; err != nil {
		l.Errorf("failed fetching DailyChallengeConfig %v", err)
		return config, totalMarketDays, err
	}
	l.Debugf("Done")

	return config, totalMarketDays, nil

}
