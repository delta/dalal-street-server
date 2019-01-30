package models

/*
TransactionSummary models entries to the TransactionSummary table
*/
type TransactionSummary struct {
	Id            uint32  `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId        uint32  `gorm:"column:userId;not null"`
	StockId       uint32  `gorm:"column:stockId;not null"`
	StockQuantity int64   `gorm:"column:stockQuantity;not null"`
	Price         float64 `gorm:"column:price;not null"`
}

func (TransactionSummary) TableName() string {
	return "TransactionSummary"
}
