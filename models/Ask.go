package models

import (
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

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
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                 uint32    `gorm:"column:userId;not null" json:"user_id"`
	StockId                uint32    `gorm:"column:stockId;not null" json:"stock_id"`
	OrderType              OrderType `gorm:"column:orderType;not null" json:"order_type"`
	Price                  uint32    `gorm:"not null" json:"price"`
	StockQuantity          uint32    `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	StockQuantityFulfilled uint32    `gorm:"column:stockQuantityFulFilled;not null" json:"stock_quantity_fulfilled"`
	IsClosed               bool      `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt              string    `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Ask) TableName() string {
	return "Asks"
}

func (gAsk *Ask) ToProto() *models_proto.Ask {
	m := make(map[OrderType]models_proto.OrderType)
	m[Limit] = models_proto.OrderType_LIMIT
	m[Market] = models_proto.OrderType_MARKET
	m[Stoploss] = models_proto.OrderType_STOPLOSS

	pAsk := &models_proto.Ask{
		Id:                     gAsk.Id,
		UserId:                 gAsk.UserId,
		StockId:                gAsk.StockId,
		Price:                  gAsk.Price,
		OrderType:              m[gAsk.OrderType],
		StockQuantity:          gAsk.StockQuantity,
		StockQuantityFulfilled: gAsk.StockQuantityFulfilled,
		IsClosed:               gAsk.IsClosed,
		CreatedAt:              gAsk.CreatedAt,
		UpdatedAt:              gAsk.UpdatedAt,
	}
	// if gAsk.OrderType == Limit {
	// 	pAsk.OrderType = models_proto.OrderType_LIMIT
	// } else if gAsk.OrderType == Market {
	// 	pAsk.OrderType = models_proto.OrderType_MARKET
	// } else if gAsk.OrderType == Stoploss {
	// 	pAsk.OrderType = models_proto.OrderType_STOPLOSS
	// }

	return pAsk
}
