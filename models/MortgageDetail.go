package models

import (
	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type MortgageDetail struct {
	Id           uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserId       uint32 `gorm:"column:userId;not null" json:"user_id"`
	StockId      uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	StocksInBank uint32 `gorm:"column:stocksInBank;not null" json:"stocks_in_bank"`
}

func (MortgageDetail) TableName() string {
	return "MortgageDetails"
}

func (md *MortgageDetail) ToProto() *models_proto.MortgageDetail {
	return &models_proto.MortgageDetail{
		Id:           md.Id,
		UserId:       md.UserId,
		StockId:      md.StockId,
		StocksInBank: md.StocksInBank,
	}
}
