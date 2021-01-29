package models

import (
	"github.com/sirupsen/logrus"
)

type Config struct {
	IsDailyChallengeOpen bool   `gorm:"column:isDailyChallengeOpen;default false not null" json:"is_dailychallengeopen"`
	MarketDay            uint32 `gorm:"column:marketDay;not null default 0 unsigned" json:"market_day"`
	IsMarketOpen         bool   `gorm:"column:isMarketOpen;not null default 0 unsigned" json:"is_marketopen"`
}

func ConfigDataInit() {
	l := logger.WithFields(logrus.Fields{
		"method": "ConfigDataInit",
	})

	l.Debugf("Invoked")

	db := getDB()

	//begin transaction
	tx := db.Begin()

	tx.Exec("TRUNCATE TABLE Config")

	config := &Config{
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

	l.Debugf("Done")

}

func IsDailyChallengeOpen() bool {
	l := logger.WithFields(logrus.Fields{
		"method": "IsDailyChallengeOpen",
	})

	l.Debugf("Invoked")

	db := getDB()

	var challengeStatus bool

	query := "SELECT DISTINCT isDailyChallengeOpen FROM `Config`"

	row := db.Raw(query).Row()

	row.Scan(&challengeStatus)

	l.Debugf("Done")

	return challengeStatus

}

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

func GetMarketDay() uint32 {
	l := logger.WithFields(logrus.Fields{
		"method": "GetMarketDay",
	})
	l.Debugf("Invoked")

	db := getDB()

	var marketDay uint32

	query := "SELECT DISTINCT marketDay FROM `Config`"

	row := db.Raw(query).Row()

	row.Scan(&marketDay)

	l.Debugf("Done")

	return marketDay

}

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

func GetDailyChallengeConfig() (*Config, error) {
	l := logger.WithFields(logrus.Fields{
		"method": "GetDailyChallengeConfig",
	})

	l.Debugf("requested")

	var config *Config

	db := getDB()

	if err := db.Table("Config").First(&config).Error; err != nil {
		l.Errorf("failed fetching DailyChallengeConfig %+e", err)
		return config, err
	}

	return config, nil

	l.Debugf("Done")

}
