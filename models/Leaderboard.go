package models

type Leaderboard struct {
	Id         uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	UserId     uint32 `gorm:"column:userId;not null"`
	Cash       uint32 `gorm:"column:cash;not null"`
	Debt       uint32 `gorm:"column:debt;not null"`
	StockWorth int32  `gorm:"column:stockWorth;not null"`
	TotalWorth int32  `gorm:"column:totalWorth;not null"`
	UpdatedAt  string `gorm:"column:updatedAt;not null"`
}

func (Leaderboard) TableName() string {
	return "Leaderboard"
}
