package models

type OrderFills struct {
	TransactionId uint32 `gorm:"column:transactionId;not null"`
	BidId         uint32 `gorm:"column:bidId;not null"`
	AskId         uint32 `gorm:"column:askId;not null"`
}

func (OrderFills) TableName() string {
	return "OrderFills"
}