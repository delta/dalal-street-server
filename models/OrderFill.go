package models

type OrderFill struct {
	TransactionId uint32 `gorm:"column:transactionId;not null"`
	BidId         uint32 `gorm:"column:bidId;not null"`
	AskId         uint32 `gorm:"column:askId;not null"`
}

func (OrderFill) TableName() string {
	return "OrderFills"
}
