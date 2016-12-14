package models

type Bids struct {
	Id                     uint32       `gorm:"primary_key;AUTO_INCREMENT"`
	UserId                 uint32       `gorm:"column:userId;not null"`
	StockId                uint32       `gorm:"column:stockId;not null"`
	OrderType              OrderType    `gorm:"column:orderType;not null"`
	Price                  uint32       `gorm:"not null"`
	StockQuantity          uint32       `gorm:"column:stockQuantity;not null"`
	StockQuantityFulFilled uint32       `gorm:"column:stockQuantityFulFilled;not null"`
	IsClosed               uint8        `gorm:"column:isClosed;not null"`
	CreatedAt              string       `gorm:"column:createdAt;not null"`
	UpdatedAt              string       `gorm:"column:updatedAt;not null"`
	OrderFills             []OrderFills `gorm:"ForeignKey:bidId"`
}

func (Bids) TableName() string {
	return "Bids"
}