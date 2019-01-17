package models

/*
TransactionSummary models entries to the TransactionSummary table
*/
type TransactionSummary struct {
	UserId        uint32  `gorm:"primary_key;column:userId;not null"`
	StockId       uint32  `gorm:"primary_key;column:stockId;not null"`
	StockQuantity int64   `gorm:"column:stockQuantity;not null"`
	Price         float64 `gorm:"column:price;not null"`
}

func (TransactionSummary) TableName() string {
	return "TransactionSummary"
}
