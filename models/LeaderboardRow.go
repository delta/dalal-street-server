package models

import (
	"time"
	"github.com/Sirupsen/logrus"
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type LeaderboardRow struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId     uint32 `gorm:"column:userId;not null" json:"user_id"`
	Rank       uint32 `gorm:"column:rank;not null" json:"rank"`
	Cash       uint32 `gorm:"column:cash;not null" json:"cash"`
	Debt       uint32 `gorm:"column:debt;not null" json:"debt"`
	StockWorth int32  `gorm:"column:stockWorth;not null" json:"stock_worth"`
	TotalWorth int32  `gorm:"column:totalWorth;not null" json:"total_worth"`
}

func (LeaderboardRow) TableName() string {
	return "Leaderboard"
}

func (l *LeaderboardRow) ToProto() *models_proto.LeaderboardRow {
	return &models_proto.LeaderboardRow{
		Id:         l.Id,
		UserId:     l.UserId,
		Cash:       l.Cash,
		Rank:       l.Rank,
		Debt:       l.Debt,
		StockWorth: l.StockWorth,
		TotalWorth: l.TotalWorth,
	}
}

func GetLeaderboard(userId, startingId, count uint32) ([]*LeaderboardRow, *LeaderboardRow, uint32, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":     "GetLeaderboard",
		"userId":     userId,
		"startingId": startingId,
		"count":      count,
	})

	l.Infof("Attempting to fetch leaderboard for userId : %v", userId)

	if startingId == 0 {
		startingId = 1
	}
	if count == 0 {
		count = LEADERBOARD_COUNT
	} else {
		count = min(count, LEADERBOARD_COUNT)
	}

	db, err := DbOpen()
	if err != nil {
		return nil, nil, TotalUserCount, err
	}
	defer db.Close()

	//for storing leaderboard details
	var leaderboardDetails []*LeaderboardRow
	//for storing user's position in leaderboard
	var currentUserDetails *LeaderboardRow

	if err := db.Where("id >= ?", startingId).Order("asc rank").Limit(count).Find(&leaderboardDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	if err := db.Where("userId = ?", userId).First(&currentUserDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	l.Infof("Successfully fetched leaderboard for userId : %v", userId)

	return leaderboardDetails, currentUserDetails, TotalUserCount, nil
}

type leaderboardQueryData struct {
	UserId     uint32
	Cash       uint32
	StockWorth int32
	Total      int32
}

//function to update leaderboard. Must be called periodically
func UpdateLeaderboard() {
	var l = logger.WithFields(logrus.Fields{
		"method": "UpdateLeaderboard",
	})

	l.Infof("Attempting to update leaderboard")

	var results []leaderboardQueryData
	var leaderboardEntries []*LeaderboardRow

	db, err := DbOpen()
	if err != nil {
		l.Errorf("Error opening database. %+v", err)
		return
	}
	defer db.Close()

	db.Raw("SELECT U.id as user_id, U.cash as cash, SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed)) AS stock_worth, (U.cash + SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed))) AS total from Users U, Transactions T, Stocks S WHERE U.id = T.userId and T.stockId = S.id GROUP BY U.id ORDER BY Total DESC").Scan(&results)

	var rank = 1
	var counter = 1

	for index, result := range results {
		leaderboardEntries = append(leaderboardEntries, &LeaderboardRow{
			Id:         uint32(index + 1),
			UserId:     result.UserId,
			Cash:       result.Cash,
			Rank:       uint32(rank),
			Debt:       0,
			StockWorth: result.StockWorth,
			TotalWorth: result.Total,
		})

		counter += 1
		if index+1 < len(results) && results[index+1].Total < result.Total {
			rank = counter
		}
	}

	db.Exec("LOCK TABLES Leaderboard WRITE")
	defer db.Exec("UNLOCK TABLES")

	db.Exec("TRUNCATE TABLE Leaderboard")

	//begin transaction
	tx := db.Begin()

	for _, leaderboardEntry := range leaderboardEntries {
		if err := db.Save(leaderboardEntry).Error; err != nil {
			l.Errorf("Error updating leaderboard. Failing. %+v", err)
			tx.Rollback()
			return
		}
	}

	//commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing leaderboardUpdate transaction. Failing. %+v", err)
		return
	}

	l.Infof("Successfully updated leaderboard")
}

//helper function to update leaderboard every two minutes
func UpdateLeaderboardTicker() {
	for {
		UpdateLeaderboard()
		time.Sleep(2*time.Minute)
	}
}
