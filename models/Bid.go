package models

import (
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type Bid struct {
	Id                     uint32    `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId                 uint32    `gorm:"column:userId;not null" json:"user_id"`
	StockId                uint32    `gorm:"column:stockId;not null" json:"stock_id"`
	OrderType              OrderType `gorm:"column:orderType;not null" json:"order_type"`
	Price                  uint32    `gorm:"not null" json:"price"`
	StockQuantity          uint32    `gorm:"column:stockQuantity;not null" json:"stock_quantity"`
	StockQuantityFulfilled uint32    `gorm:"column:stockQuantityFulFilled;not null"json:"stock_quantity_fulfilled"`
	IsClosed               bool      `gorm:"column:isClosed;not null" json:"is_closed"`
	CreatedAt              string    `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt              string    `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Bid) TableName() string {
	return "Bids"
}

func (gBid *Bid) ToProto() *models_proto.Bid {
	pBid := &models_proto.Bid{
		Id:      gBid.Id,
		UserId:  gBid.UserId,
		StockId: gBid.StockId,
		Price:   gBid.Price,
		//	OrderType              OrderType `protobuf:"varint,5,opt,name=order_type,json=orderType,enum=dalalstreet.socketapi.models.OrderType" json:"order_type,omitempty"`
		StockQuantity:          gBid.StockQuantity,
		StockQuantityFulfilled: gBid.StockQuantityFulfilled,
		IsClosed:               gBid.IsClosed,
		CreatedAt:              gBid.CreatedAt,
		UpdatedAt:              gBid.UpdatedAt,
	}
	if gBid.OrderType == Limit {
		pBid.OrderType = models_proto.OrderType_LIMIT
	} else if gBid.OrderType == Market {
		pBid.OrderType = models_proto.OrderType_MARKET
	} else if gBid.OrderType == Stoploss {
		pBid.OrderType = models_proto.OrderType_STOPLOSS
	}
	return pBid
}
