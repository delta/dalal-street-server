package models

import (
	"github.com/thakkarparth007/dalal-street-server/proto_build/actions"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
)

type StockHistory struct {
	StockId    uint32 `gorm:"column:stockId;not null" json:"stock_id"`
	StockPrice uint32 `gorm:"column:stockPrice;not null" json:"stock_price"`
	CreatedAt  string `gorm:"column:createdAt;not null" json:"created_at"`
	Interval   uint32 `gorm:"column:interval;not null" json:"interval"`
	Open       uint32 `gorm:"column:open;not null" json:"open"`
	High       uint32 `gorm:"high:close;not null" json:"high"`
	Low        uint32 `gorm:"low:close;not null" json:"low"`
}

func (StockHistory) TableName() string {
	return "StockHistory"
}

type Resolution uint32

//Resolution enum (hack?)
const (
	OneMinute     Resolution = 1  // Range will be 60*Resolution
	FiveMinutes   Resolution = 5  // Range will be 60*Resolution
	TenMinutes    Resolution = 10 // Range will be 60*Resolution
	ThirtyMinutes Resolution = 30 // Range will be 60*Resolution
	SixtyMinutes  Resolution = 60 // Range will be 60*Resolution
	OneDay        Resolution = 0  // Range will be 60*Resolution
)

func ResolutionFromProto(s actions_pb.StockHistoryResolution) Resolution {
	if s == actions_pb.StockHistoryResolution_OneMinute {
		return OneMinute
	} else if s == actions_pb.StockHistoryResolution_FiveMinutes {
		return FiveMinutes
	} else if s == actions_pb.StockHistoryResolution_TenMinutes {
		return TenMinutes
	} else if s == actions_pb.StockHistoryResolution_ThirtyMinutes {
		return ThirtyMinutes
	} else if s == actions_pb.StockHistoryResolution_SixtyMinutes {
		return SixtyMinutes
	} else if s == actions_pb.StockHistoryResolution_OneDay {
		return OneDay
	}
	return OneMinute
}

func (gStockHistory *StockHistory) ToProto() *models_pb.StockHistory {
	return &models_pb.StockHistory{
		StockId:    gStockHistory.StockId,
		StockPrice: gStockHistory.StockPrice,
		CreatedAt:  gStockHistory.CreatedAt,
		Interval:   gStockHistory.Interval,
		Open:       gStockHistory.Open,
		High:       gStockHistory.High,
		Low:        gStockHistory.Low,
	}
}
