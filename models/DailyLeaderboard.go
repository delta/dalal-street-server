package models

import (
	"fmt"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

type DailyLeaderboardRow struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId     uint32 `gorm:"column:userId;not null" json:"user_id"`
	UserName   string `gorm:"column:userName;not null" json:"user_name"`
	Rank       uint32 `gorm:"column:rank;not null" json:"rank"`
	Cash       int64  `gorm:"column:cash;not null" json:"cash"`
	Debt       uint64 `gorm:"column:debt;not null" json:"debt"`
	StockWorth int64  `gorm:"column:stockWorth;not null" json:"stock_worth"`
	TotalWorth int64  `gorm:"column:totalWorth;not null" json:"total_worth"`
	IsBlocked  bool   `gorm:"column:isBlocked;not null" json:"is_blocked"`
}

func (DailyLeaderboardRow) TableName() string {
	return "DailyLeaderboard"
}

func (l *DailyLeaderboardRow) ToProto() *models_pb.DailyLeaderboardRow {
	return &models_pb.DailyLeaderboardRow{
		Id:         l.Id,
		UserId:     l.UserId,
		UserName:   l.UserName,
		Cash:       l.Cash,
		Rank:       l.Rank,
		Debt:       l.Debt,
		StockWorth: l.StockWorth,
		TotalWorth: l.TotalWorth,
		IsBlocked:  l.IsBlocked,
	}
}

func GetDailyLeaderboard(userId, startingId, count uint32) ([]*DailyLeaderboardRow, *DailyLeaderboardRow, uint32, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":     "GetDailyLeaderboard",
		"userId":     userId,
		"startingId": startingId,
		"count":      count,
	})

	l.Infof("Attempting to fetch DailyLeaderboard for userId : %v", userId)

	if startingId == 0 {
		startingId = 1
	}
	if count == 0 {
		count = LEADERBOARD_COUNT
	} else {
		count = utils.MinInt32(count, LEADERBOARD_COUNT)
	}

	db := getDB()

	db.Model(&User{}).Count(&TotalUserCount)

	//for storing leaderboard details
	var leaderboardDetails []*DailyLeaderboardRow
	//for storing user's position in leaderboard
	currentUserDetails := DailyLeaderboardRow{}

	if err := db.Where("id >= ?", startingId).Order("`rank` asc").Limit(count).Find(&leaderboardDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	if err := db.Where("userId = ?", userId).First(&currentUserDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	l.Infof("Successfully fetched DailyLeaderboard for userId : %v, %+v", userId, leaderboardDetails)

	return leaderboardDetails, &currentUserDetails, TotalUserCount, nil
}

type dailyLeaderboardQueryData struct {
	UserId     uint32
	UserName   string
	Cash       int64
	StockWorth int64
	Total      int64
	IsBlocked  bool
}

//function to update DailyLeaderboard. Must be called periodically
func UpdateDailyLeaderboard() {
	var l = logger.WithFields(logrus.Fields{
		"method": "UpdateDailyLeaderboard",
	})

	l.Infof("Attempting to update DailyLeaderboard")

	var results []dailyLeaderboardQueryData
	var dailyLeaderboardEntries []*DailyLeaderboardRow

	db := getDB()
	tx := db.Begin()

	query := fmt.Sprintf(`
	SELECT L.userId as user_id, L.userName as user_name, L.isBlocked as is_blocked,
	       ifnull(L.cash,0) - ifnull(E.cash,%d) as cash,
	       ifnull(L.stockWorth,0) - ifnull(E.stockWorth,0) as stock_worth,
	       ifnull(L.totalWorth,0) - ifnull(E.totalWorth,%d) as total
	FROM Leaderboard L
	LEFT JOIN EndOfDayValues E ON L.userId = E.userId
	ORDER BY Total DESC;
	`, STARTING_CASH, STARTING_CASH)

	tx.Raw(query).Scan(&results)

	var rank = 1
	var counter = 1

	for index, result := range results {
		dailyLeaderboardEntries = append(dailyLeaderboardEntries, &DailyLeaderboardRow{
			Id:         uint32(index + 1),
			UserId:     result.UserId,
			UserName:   result.UserName,
			Cash:       result.Cash,
			Rank:       uint32(rank),
			Debt:       0,
			StockWorth: result.StockWorth,
			TotalWorth: result.Total,
			IsBlocked:  result.IsBlocked,
		})

		counter += 1
		if index+1 < len(results) && results[index+1].Total < result.Total {
			rank = counter
		}
	}

	tx.Exec("TRUNCATE TABLE DailyLeaderboard")

	for _, dailyLeaderboardEntry := range dailyLeaderboardEntries {
		if err := tx.Save(dailyLeaderboardEntry).Error; err != nil {
			l.Errorf("Error updating DailyLeaderboard. Failing. %+v", err)
			tx.Rollback()
			return
		}
	}

	//commit transaction
	if err := tx.Commit().Error; err != nil {
		l.Errorf("Error committing DailyLeaderboard transaction. Failing. %+v", err)
		tx.Rollback()
		return
	}

	l.Infof("Successfully updated DailyLeaderboard")
}
