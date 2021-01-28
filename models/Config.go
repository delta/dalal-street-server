package models

import "github.com/sirupsen/logrus"

type Config struct {
	IsDailyChallengeOpen bool   `gorm:"column:isDailyChallengeOpen;default false not null" json:"is_dailychallengeopen"`
	MarketDay            uint32 `gorm:"column:id;marketDay;not null default 0" json:"market_day"`
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

func IsDailyChallengeOpen() (bool, error) {
	l := logger.WithFields(logrus.Fields{
		"method": "IsDailyChallengeOpen",
	})

	l.Debugf("Invoked")

	db := getDB()

	var queryData bool

	if err := db.Table("Config").Select("isDailyChallengeOpen").First(&queryData).Error; err != nil {
		l.Errorf("failed to fetch isDailyChallenge from db %+e", err)
		return queryData, err
	}
	l.Debugf("Done")

	return queryData, nil

}

func setIsDailyChallengeOpen(challengeStatus bool) error {
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

func GetMarketDay() (uint32, error) {
	l := logger.WithFields(logrus.Fields{
		"method": "GetMarketDay",
	})
	l.Debugf("Invoked")

	db := getDB()

	var marketDay uint32

	if err := db.Table("Config").Select("marketDay").First(&marketDay).Error; err != nil {
		l.Errorf("failed to fetch isDailyChallenge from db %+e", err)
		return marketDay, err
	}

	l.Debugf("Done")

	return marketDay, nil

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
