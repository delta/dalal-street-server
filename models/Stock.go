package models

import (
	"fmt"
	"sync"

	"github.com/thakkarparth007/dalal-street-server/proto_build/datastreams"

	"github.com/Sirupsen/logrus"
	"github.com/thakkarparth007/dalal-street-server/proto_build/models"
)

const TIMES_RESOLUTION = 60

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
	AvgLastPrice     uint32 `gorm:"column:avgLastPrice;not null" json:"avg_last_price"`
	CreatedAt        string `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt        string `gorm:"column:updatedAt;not null" json:"updated_at"`

	// HACK: Getting last minute's hl from transactions used by stock history
	open uint32 // Used to store Open for the last minute
	high uint32 // Used to store High for the last minute
	low  uint32 // Used to store Low for the last minute
}

func (Stock) TableName() string {
	return "Stocks"
}

func (gStock *Stock) ToProto() *models_pb.Stock {
	return &models_pb.Stock{
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
		AvgLastPrice:     gStock.AvgLastPrice,
		CreatedAt:        gStock.CreatedAt,
		UpdatedAt:        gStock.UpdatedAt,
	}
}

type stockAndLock struct {
	sync.RWMutex
	stock *Stock
}

var allStocks = struct {
	sync.RWMutex
	m map[uint32]*stockAndLock
}{
	sync.RWMutex{},
	make(map[uint32]*stockAndLock),
}

var avgLastPrice = struct {
	sync.RWMutex
	m map[uint32]uint32
}{
	sync.RWMutex{},
	make(map[uint32]uint32),
}

func GetStockCopy(stockId uint32) (Stock, error) {
	allStocks.RLock()
	defer allStocks.RUnlock()

	stockNLock, ok := allStocks.m[stockId]

	if !ok {
		return Stock{}, fmt.Errorf("Invalid stock id %d", stockId)
	}

	stockNLock.RLock()
	stockCopy := *stockNLock.stock
	stockNLock.RUnlock()

	return stockCopy, nil
}

func GetAllStocks() map[uint32]*Stock {
	allStocks.RLock()
	defer allStocks.RUnlock()

	var allStocksCopy = make(map[uint32]*Stock)
	for stockId, stockNLock := range allStocks.m {
		stockNLock.RLock()
		allStocksCopy[stockId] = &Stock{}
		*allStocksCopy[stockId] = *stockNLock.stock
		stockNLock.RUnlock()
	}

	return allStocksCopy
}

func UpdateStockPrice(stockId, price uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "loadStocks",
	})

	l.Infof("Attempting")

	allStocks.Lock()
	stockNLock, ok := allStocks.m[stockId]
	if !ok {
		return fmt.Errorf("Not found stock for id %d", stockId)
	}
	allStocks.Unlock()

	stockNLock.Lock()
	defer stockNLock.Unlock()

	stock := stockNLock.stock
	oldStockCopy := *stock

	stock.CurrentPrice = price
	if price > stock.DayHigh {
		stock.DayHigh = price
	} else if price < stock.DayLow {
		stock.DayLow = price
	}

	if price > stock.high {
		stock.high = price
	} else if price < stock.low {
		stock.low = price
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

	avgLastPrice.Lock()
	avgLastPrice.m[stock.Id] = avgLastPrice.m[stock.Id] - uint32((avgLastPrice.m[stock.Id]+stock.CurrentPrice)/20)
	stock.AvgLastPrice = avgLastPrice.m[stock.Id]
	avgLastPrice.Unlock()

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()

	if err := db.Save(stock).Error; err != nil {
		*stock = oldStockCopy
		return err
	}

	stockPriceStream := datastreamsManager.GetStockPricesStream()
	stockPriceStream.SendStockPriceUpdate(stockId, price)

	l.Infof("Done")

	return nil
}

func LoadStocks() error {
	var l = logger.WithFields(logrus.Fields{
		"method": "loadStocks",
	})

	l.Infof("Attempting")

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()

	var stocks []*Stock
	if err := db.Find(&stocks).Error; err != nil {
		return err
	}

	allStocks.Lock()
	avgLastPrice.Lock()
	for _, stock := range stocks {
		allStocks.m[stock.Id] = &stockAndLock{stock: stock}
		avgLastPrice.m[stock.Id] = stock.CurrentPrice
	}
	avgLastPrice.Unlock()
	allStocks.Unlock()

	l.Infof("Loaded %+v", allStocks)

	return nil
}

func GetStockHistory(stockId uint32, interval Resolution) ([]*StockHistory, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":   "GetStockHistory",
		"stockId":  stockId,
		"interval": interval,
	})

	l.Infof("Attempting to get stock History for stockId : %v", stockId)

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	allStocks.m[stockId].RLock()
	defer allStocks.m[stockId].RUnlock()

	var histories []*StockHistory

	if interval != 0 {
		if err := db.Where("interval_record =", interval).Order("id desc").Limit(TIMES_RESOLUTION).Find(histories).Error; err != nil {
			return nil, err
		}
	}

	return histories, nil
}

func GetCompanyDetails(stockId uint32) (*Stock, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "GetCompanyDetails",
		"stockId": stockId,
	})

	l.Infof("Attempting to get company profile for stockId : %v", stockId)

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return nil, err
	}
	defer db.Close()

	allStocks.m[stockId].RLock()
	defer allStocks.m[stockId].RUnlock()

	stock := *allStocks.m[stockId].stock

	l.Infof("Successfully fetched company profile for stock id : %v", stockId)
	return &stock, nil
}

func AddStocksToExchange(stockId, count uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "AddStocksToExchange",
		"param_stockId": stockId,
		"param_count":   count,
	})

	l.Infof("Attempting")

	allStocks.Lock()
	stockNLock, ok := allStocks.m[stockId]
	if !ok {
		return fmt.Errorf("Not found stock for id %d", stockId)
	}
	allStocks.Unlock()

	stockNLock.Lock()
	defer stockNLock.Unlock()
	stock := stockNLock.stock

	stock.StocksInExchange += count

	db, err := DbOpen()
	if err != nil {
		l.Error(err)
		return err
	}
	defer db.Close()

	if err := db.Save(stock).Error; err != nil {
		stock.StocksInExchange -= count
		return err
	}

	stockExchangeStream := datastreamsManager.GetStockExchangeStream()
	stockExchangeStream.SendStockExchangeUpdate(stockId, &datastreams_pb.StockExchangeDataPoint{
		Price:            stock.CurrentPrice,
		StocksInExchange: stock.StocksInExchange,
		StocksInMarket:   stock.StocksInMarket,
	})

	l.Infof("Done")

	return nil
}
