package models

import "github.com/sirupsen/logrus"

type EndOfDayValue struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId     uint32 `gorm:"column:userId;not null" json:"user_id"`
	Cash       uint64 `gorm:"column:cash;not null" json:"cash"`
	Debt       uint64 `gorm:"column:debt;not null" json:"debt"`
	StockWorth int64  `gorm:"column:stockWorth;not null" json:"stock_worth"`
	TotalWorth int64  `gorm:"column:totalWorth;not null" json:"total_worth"`
}

func (EndOfDayValue) TableName() string {
	return "EndOfDayValues"
}

// Updates EndOfDayValues Table which is used to Calculate DailyLeaderboard
func UpdateEndOfDayValues() error {
	var l = logger.WithFields(logrus.Fields{
		"method": "UpdateEndOfDayValues",
	})

	l.Infof("Attempting to Update EndOfDayValues")

	leaderboard, err := GetEntireLeaderboard()

	if err != nil {
		return err
	}

	var endOfDayValues []*EndOfDayValue

	for _, leaderboardRow := range leaderboard {
		endOfDayValues = append(endOfDayValues, &EndOfDayValue{
			UserId:     leaderboardRow.UserId,
			Cash:       leaderboardRow.Cash,
			Debt:       leaderboardRow.Debt,
			StockWorth: leaderboardRow.StockWorth,
			TotalWorth: leaderboardRow.TotalWorth,
		})
	}

	db := getDB()
	// Begin Transaction
	tx := db.Begin()

	tx.Exec("TRUNCATE TABLE EndOfDayValues")

	for _, value := range endOfDayValues {
		if err := tx.Save(value).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Commit Transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	l.Infof("Successfully Updated EndOfDayValues")
	return nil
}
