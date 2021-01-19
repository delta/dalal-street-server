package models

import (
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

func GetDailyLeaderboard(userId, startingId, count uint32) ([]*LeaderboardRow, *LeaderboardRow, uint32, error) {
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

	table := db.Table("DailyLeaderboard")

	//for storing leaderboard details
	var leaderboardDetails []*LeaderboardRow
	//for storing user's position in leaderboard
	currentUserDetails := LeaderboardRow{}

	if err := table.Where("id >= ?", startingId).Order("rank asc").Limit(count).Find(&leaderboardDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	if err := table.Where("userId = ?", userId).First(&currentUserDetails).Error; err != nil {
		return nil, nil, TotalUserCount, err
	}

	l.Infof("Successfully fetched DailyLeaderboard for userId : %v, %+v", userId, leaderboardDetails)

	return leaderboardDetails, &currentUserDetails, TotalUserCount, nil
}
