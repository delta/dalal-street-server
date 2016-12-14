package models

type StockHistory struct {
	StockId    uint32 `gorm:"column:stockId;not null"`
	StockPrice uint32 `gorm:"column:stockPrice;not null"`
	CreatedAt  string `gorm:"column:createdAt;not null"`
}

func (StockHistory) TableName() string {
	return "StockHistory"
}