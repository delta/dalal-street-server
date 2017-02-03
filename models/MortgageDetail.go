package models

type MortgageDetail struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT"`
	UserId       uint32 `gorm:"column:userId;not null"`
	StockId      uint32 `gorm:"column:stockId;not null"`
	StocksInBank uint32 `gorm:"column:stocksInBank;not null"`
}

func (MortgageDetail) TableName() string {
	return "MortgageDetails"
}
