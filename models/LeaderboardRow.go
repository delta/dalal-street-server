package models

import (
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
