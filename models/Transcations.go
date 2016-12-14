package models

type Transactions struct {
	Id            uint32          `gorm:"primary_key;AUTO_INCREMENT"`
	UserId        uint32          `gorm:"column:userId;not null"`
	StockId       uint32          `gorm:"column:stockId;not null"`
	Type          TransactionType `gorm:"column:type;not null"`
	StockQuantity uint32          `gorm:"column:stockQuantity;not null"`
	Price         uint32          `gorm:"not null"`
	CreatedAt     string          `gorm:"column:createdAt;not null"`
	OrderFills    []OrderFills    `gorm:"ForeignKey:transactionId"`
}

func (Transactions) TableName() string {
	return "Transactions"
}