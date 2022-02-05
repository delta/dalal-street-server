package models

// Short sell bank is an abstraction of inventory, holds
// the number of stocks of stocks available for lending (which is further used for shorting by the user)

type ShortSellBank struct {
	StockId         uint32 `gorm:"column:stockId;primary_key; json:"stockId"`
	AvailableStocks uint32 `gorm:"column:availableStock; default 0; json:"availableStocks"`
}

func (ShortSellBank) TableName() string {
	return "ShortSellBank"
}
