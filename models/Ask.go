package models

type OrderType uint8

const (
	Limit OrderType = iota
	Market
	Stoploss
)

var orderTypes = [...]string{
	"Limit",
	"Market",
	"Stoploss",
}

func (ot OrderType) String() string {
	return orderTypes[ot-1]
}

type Ask struct {
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT"`
	UserId                 uint32    `gorm:"column:userId;not null"`
	StockId                uint32    `gorm:"column:stockId;not null"`
	OrderType              OrderType `gorm:"column:orderType;not null"`
	Price                  uint32    `gorm:"not null"`
	StockQuantity          uint32    `gorm:"column:stockQuantity;not null"`
	StockQuantityFulFilled uint32    `gorm:"column:stockQuantityFulFilled;not null"`
	IsClosed               bool      `gorm:"column:isClosed;not null"`
	CreatedAt              string    `gorm:"column:createdAt;not null"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null"`
}

func (Ask) TableName() string {
	return "Asks"
}
