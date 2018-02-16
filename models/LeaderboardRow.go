package models

import (
	"time"

	"github.com/Sirupsen/logrus"
	models_pb "github.com/thakkarparth007/dalal-street-server/proto_build/models"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

type LeaderboardRow struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId     uint32 `gorm:"column:userId;not null" json:"user_id"`
	UserName   string `gorm:"column:userName;not null" json:"user_name"`
	Rank       uint32 `gorm:"column:rank;not null" json:"rank"`
	Cash       uint32 `gorm:"column:cash;not null" json:"cash"`
	Debt       uint32 `gorm:"column:debt;not null" json:"debt"`
	StockWorth int32  `gorm:"column:stockWorth;not null" json:"stock_worth"`
	TotalWorth int32  `gorm:"column:totalWorth;not null" json:"total_worth"`
}

func (LeaderboardRow) TableName() string {
	return "Leaderboard"
}

func (l *LeaderboardRow) ToProto() *models_pb.LeaderboardRow {
	return &models_pb.LeaderboardRow{
		Id:         l.Id,
		UserId:     l.UserId,
		UserName:   l.UserName,
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
		count = utils.MinInt(count, LEADERBOARD_COUNT)
	}

	db := getDB()

	db.Model(&User{}).Count(&TotalUserCount)

	//for storing leaderboard details
	var leaderboardDetails []*LeaderboardRow
	//for storing user's position in leaderboard
	currentUserDetails := LeaderboardRow{}

	if err := db.Where("id >= ?", startingId).Order("rank asc").Limit(count).Find(&leaderboardDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	if err := db.Where("userId = ?", userId).First(&currentUserDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	l.Infof("Successfully fetched leaderboard for userId : %v, %+v", userId, leaderboardDetails)

	return leaderboardDetails, &currentUserDetails, TotalUserCount, nil
}

type leaderboardQueryData struct {
	UserId     uint32
	UserName   string
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

	db := getDB()

	db.Raw(`
		SELECT U.id as user_id, U.name as user_name, U.cash as cash,
			ifNull(SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed)),0) AS stock_worth,
			ifnull((U.cash + SUM(cast(S.currentPrice AS signed) * cast(T.stockQuantity AS signed))),U.cash) AS total
		FROM
			Users U LEFT JOIN Transactions T ON U.id = T.userId
					LEFT JOIN Stocks S ON T.stockId = S.id
		GROUP BY U.id
		ORDER BY Total DESC;
	`).Scan(&results)

	var rank = 1
	var counter = 1

	for index, result := range results {
		leaderboardEntries = append(leaderboardEntries, &LeaderboardRow{
			Id:         uint32(index + 1),
			UserId:     result.UserId,
			UserName:   result.UserName,
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

	//begin transaction
	tx := db.Begin()

	db.Exec("TRUNCATE TABLE Leaderboard")

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
		time.Sleep(2 * time.Minute)
	}
}
