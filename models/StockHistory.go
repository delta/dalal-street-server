package models

import (
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
)

type StockHistory struct {
	StockId    uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	StockPrice uint32 `gorm:"column:stockPrice;not null" json:"stock_price"`
	CreatedAt  string `gorm:"column:createdAt;not null" json:"created_at"`
}

func (StockHistory) TableName() string {
	return "StockHistory"
}

func (gStockHistory *StockHistory) ToProto() *models_pb.StockHistory {
	return &models_pb.StockHistory{
		StockId:    gStockHistory.StockId,
		StockPrice: gStockHistory.StockPrice,
		CreatedAt:  gStockHistory.CreatedAt,
	}
}
