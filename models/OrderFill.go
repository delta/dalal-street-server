package models

import (
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type OrderFill struct {
	TransactionId uint32 `gorm:"column:transactionId;not null" json:"transaction_id"`
	BidId         uint32 `gorm:"column:bidId;not null" json:"bid_id"`
	AskId         uint32 `gorm:"column:askId;not null" json:"ask_id"`
}

func (OrderFill) TableName() string {
	return "OrderFills"
}

func (gOrderFill *OrderFill) ToProto() *models_proto.OrderFill {
	return &models_proto.OrderFill{
		TransactionId: gOrderFill.TransactionId,
		BidId:         gOrderFill.BidId,
		AskId:         gOrderFill.AskId,
	}
}
