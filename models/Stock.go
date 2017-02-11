package models

import (
	"sync"
	"fmt"

	models_proto "github.com/thakkarparth007/dalal-street-server/socketapi/proto_build/models"
)

type Stock struct {
	Id               uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	ShortName        string `gorm:"column:shortName;not null" json:"short_name"`
	FullName         string `gorm:"column:fullName;not null" json:"full_name"`
	Description      string `gorm:"not null" json:"description"`
	CurrentPrice     uint32 `gorm:"column:currentPrice;not null"  json:"current_price"`
	DayHigh          uint32 `gorm:"column:dayHigh;not null" json:"day_high"`
	DayLow           uint32 `gorm:"column:dayLow;not null" json:"day_low"`
	AllTimeHigh      uint32 `gorm:"column:allTimeHigh;not null" json:"all_time_high"`
	AllTimeLow       uint32 `gorm:"column:allTimeLow;not null" json:"all_time_low"`
	StocksInExchange uint32 `gorm:"column:stocksInExchange;not null" json:"stocks_in_exchange"`
	StocksInMarket   uint32 `gorm:"column:stocksInMarket;not null" json:"stocks_in_market"`
	PreviousDayClose uint32 `gorm:"column:previousDayClose;not null" json:"previous_day_close"`
	UpOrDown         bool   `gorm:"column:upOrDown;not null" json:"up_or_down"`
	CreatedAt        string `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt        string `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (Stock) TableName() string {
	return "Stocks"
}

func (gStock *Stock) ToProto() *models_proto.Stock {
	return &models_proto.Stock{
		Id:               gStock.Id,
		ShortName:        gStock.ShortName,
		FullName:         gStock.FullName,
		Description:      gStock.Description,
		CurrentPrice:     gStock.CurrentPrice,
		DayHigh:          gStock.DayHigh,
		DayLow:           gStock.DayLow,
		AllTimeHigh:      gStock.AllTimeHigh,
		AllTimeLow:       gStock.AllTimeLow,
		StocksInExchange: gStock.StocksInExchange,
		StocksInMarket:   gStock.StocksInMarket,
		UpOrDown:         gStock.UpOrDown,
		PreviousDayClose: gStock.PreviousDayClose,
		CreatedAt:        gStock.CreatedAt,
		UpdatedAt:        gStock.UpdatedAt,
	}
}

type stockAndLock struct {
	sync.RWMutex
	stock *Stock
}

var allStocks = struct{
	sync.RWMutex
	m map[uint32]*stockAndLock
}{
	sync.RWMutex{},
	make(map[uint32]*stockAndLock),
}

func GetAllStocks() (map[uint32]Stock) {
	allStocks.RLock()
	defer allStocks.RUnlock()

	var allStocksCopy = make(map[uint32]Stock)
	for stockId, stockNLock := range allStocks.m {
		stockNLock.RLock()
		allStocksCopy[stockId] = *stockNLock.stock
		stockNLock.RUnlock()
	}

	return allStocksCopy
}

func updateStockPrice(stockId, price uint32) error {
	allStocks.Lock()
	defer allStocks.Unlock()

	stockNLock, ok := allStocks.m[stockId]
	if !ok {
		return fmt.Errorf("Not found stock for id %d", stockId)
	}

	stock := stockNLock.stock
	stock.CurrentPrice = price
	if price > stock.DayHigh {
		stock.DayHigh = price
	} else if price < stock.DayLow {
		stock.DayLow = price
	}

	if price > stock.AllTimeHigh {
		stock.AllTimeHigh = price
	} else if price < stock.AllTimeLow {
		stock.AllTimeLow = price
	}

	if price > stock.PreviousDayClose {
		stock.UpOrDown = true
	} else {
		stock.UpOrDown = false
	}

	return nil
}
