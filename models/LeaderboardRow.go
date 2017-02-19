package models

import (
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
