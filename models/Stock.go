package models

import (
	"errors"
	"fmt"
	"math"
	"sync"

	datastreams_pb "github.com/delta/dalal-street-server/proto_build/datastreams"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

const TIMES_RESOLUTION = 60

var (
	InvalidStockError = errors.New("Invalid stock id")
)

type Stock struct {
	Id               uint32  `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	ShortName        string  `gorm:"column:shortName;not null" json:"short_name"`
	FullName         string  `gorm:"column:fullName;not null" json:"full_name"`
	Description      string  `gorm:"not null" json:"description"`
	CurrentPrice     uint64  `gorm:"column:currentPrice;not null"  json:"current_price"`
	DayHigh          uint64  `gorm:"column:dayHigh;not null" json:"day_high"`
	DayLow           uint64  `gorm:"column:dayLow;not null" json:"day_low"`
	AllTimeHigh      uint64  `gorm:"column:allTimeHigh;not null" json:"all_time_high"`
	AllTimeLow       uint64  `gorm:"column:allTimeLow;not null" json:"all_time_low"`
	StocksInExchange uint64  `gorm:"column:stocksInExchange;not null" json:"stocks_in_exchange"`
	StocksInMarket   uint64  `gorm:"column:stocksInMarket;not null" json:"stocks_in_market"`
	PreviousDayClose uint64  `gorm:"column:previousDayClose;not null" json:"previous_day_close"`
	UpOrDown         bool    `gorm:"column:upOrDown;not null" json:"up_or_down"`
	LastTradePrice   uint64  `gorm:"column:lastTradePrice;not null" json:"last_trade_price"`
	RealAvgPrice     float64 `gorm:"column:realAvgPrice;not null" json:"real_avg_price"`
	CreatedAt        string  `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt        string  `gorm:"column:updatedAt;not null" json:"updated_at"`
	GivesDividends   bool    `gorm:"column:givesDividends;not null" json:"gives_dividends"`
	IsBankrupt       bool    `gorm:"column:isBankrupt;not null" json:"is_bankrupt"`

	// HACK: Getting last minute's hl from transactions used by stock history
	open   uint64 // Used to store Open for the last minute
	high   uint64 // Used to store High for the last minute
	low    uint64 // Used to store Low for the last minute
	volume uint64 //Used to store trade volume for the last minute
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
		LastTradePrice:   gStock.LastTradePrice,
		RealAvgPrice:     gStock.RealAvgPrice,
		CreatedAt:        gStock.CreatedAt,
		UpdatedAt:        gStock.UpdatedAt,
		GivesDividends:   gStock.GivesDividends,
		IsBankrupt:       gStock.IsBankrupt,
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
	m map[uint32]float64
}{
	sync.RWMutex{},
	make(map[uint32]float64),
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

func UpdateStockPrice(stockId uint32, price uint64, quantity uint64) error {
	var l = logger.WithFields(logrus.Fields{
		"method":        "UpdateStockPrice",
		"param_stockId": stockId,
		"param_price":   price,
		"param_qty":     quantity,
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

	stock.LastTradePrice = price

	stock.UpdatedAt = utils.GetCurrentTimeISO8601()

	// averageStockCount should not be 0, so the math.Max ensures that it is at least 1.
	// averageStockCount should not be above MAX_AVERAGE_STOCK_COUNT so the math.Min ensures that.
	// The reason for this upper limit is that if the averageStockCount is too high, the price will never change.
	averageStockCount := uint64(math.Min(math.Max(float64(STOCK_AVERAGE_PERCENT*(stock.StocksInExchange+stock.StocksInMarket)/100), 1), MAX_AVERAGE_STOCK_COUNT))
	finalQuantity := uint64(math.Min(float64(quantity), float64(averageStockCount)))

	avgLastPrice.Lock()
	l.Infof("Average stock count = %v, finalQuantity = %v, Previous avgLastPrice = %v", averageStockCount, finalQuantity, avgLastPrice.m[stockId])

	priceDifference := float64(price) - avgLastPrice.m[stockId]
	stockPriceChange := (float64(finalQuantity) * priceDifference) / float64(averageStockCount)

	avgLastPrice.m[stock.Id] += stockPriceChange
	l.Infof("New Current Price = Average of last %v stock trades = +%v", averageStockCount, avgLastPrice.m[stock.Id])
	stock.CurrentPrice = uint64(avgLastPrice.m[stock.Id])
	stock.RealAvgPrice = avgLastPrice.m[stockId]
	avgLastPrice.Unlock()

	if stock.CurrentPrice > stock.DayHigh {
		stock.DayHigh = stock.CurrentPrice
	} else if stock.CurrentPrice < stock.DayLow {
		stock.DayLow = stock.CurrentPrice
	}

	if stock.CurrentPrice > stock.high {
		stock.high = stock.CurrentPrice
	} else if stock.CurrentPrice < stock.low {
		stock.low = stock.CurrentPrice
	}

	if stock.CurrentPrice > stock.AllTimeHigh {
		stock.AllTimeHigh = stock.CurrentPrice
	} else if stock.CurrentPrice < stock.AllTimeLow {
		stock.AllTimeLow = stock.CurrentPrice
	}

	if stock.CurrentPrice > stock.PreviousDayClose {
		stock.UpOrDown = true
	} else {
		stock.UpOrDown = false
	}

	db := getDB()

	if err := db.Save(stock).Error; err != nil {
		*stock = oldStockCopy
		return err
	}

	stockPriceStream := datastreamsManager.GetStockPricesStream()
	stockPriceStream.SendStockPriceUpdate(stockId, stock.CurrentPrice)

	l.Infof("Done")

	return nil
}
func UpdateStockVolume(stockId uint32, volume uint64) {
	allStocks.Lock()
	allStocks.m[stockId].stock.volume += volume
	allStocks.Unlock()
}

func LoadStocks() error {
	var l = logger.WithFields(logrus.Fields{
		"method": "loadStocks",
	})

	l.Infof("Attempting")

	db := getDB()

	var stocks []*Stock
	if err := db.Find(&stocks).Error; err != nil {
		return err
	}

	allStocks.Lock()
	avgLastPrice.Lock()
	allStocks.m = make(map[uint32]*stockAndLock)
	avgLastPrice.m = make(map[uint32]float64)

	for _, stock := range stocks {
		allStocks.m[stock.Id] = &stockAndLock{stock: stock}
		// this is inaccurate but no one would care so much. so it's okay.
		allStocks.m[stock.Id].stock.open = allStocks.m[stock.Id].stock.CurrentPrice
		allStocks.m[stock.Id].stock.high = allStocks.m[stock.Id].stock.CurrentPrice
		allStocks.m[stock.Id].stock.low = allStocks.m[stock.Id].stock.CurrentPrice
		avgLastPrice.m[stock.Id] = stock.RealAvgPrice
	}

	avgLastPrice.Unlock()
	allStocks.Unlock()

	l.Infof("Loaded %+v", &allStocks)

	return nil
}

func GetCompanyDetails(stockId uint32) (*Stock, error) {
	var l = logger.WithFields(logrus.Fields{
		"method":  "GetCompanyDetails",
		"stockId": stockId,
	})

	l.Infof("Attempting to get company profile for stockId : %v", stockId)

	allStocks.m[stockId].RLock()
	defer allStocks.m[stockId].RUnlock()

	stock := *allStocks.m[stockId].stock

	l.Infof("Successfully fetched company profile for stock id : %v", stockId)
	return &stock, nil
}

func AddStocksToExchange(stockId uint32, count uint64) error {
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
	stock.UpdatedAt = utils.GetCurrentTimeISO8601()

	db := getDB()

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

// SetPreviousDayClose will be called when market
// is closing for the day to update the day closing
func SetPreviousDayClose() (err error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "SetPreviousDayClose",
	})

	db := getDB()
	tx := db.Begin()

	l.Info("Attempting to set previous day close")

	l.Info("Locking allStocks in SetPreviousDayClose")
	allStocks.Lock()
	defer func() {
		l.Info("Unlocking allStocks in SetPreviousDayClose")
		allStocks.Unlock()

		// Calling LoadStocks to prevent inconsistency
		// between db and allStocks
		err = LoadStocks()
	}()

	for _, stockNLock := range allStocks.m {
		stockNLock.stock.PreviousDayClose = stockNLock.stock.CurrentPrice
		if err = tx.Save(stockNLock.stock).Error; err != nil {
			l.Errorf("Error occurred : %v", err)
			tx.Rollback()
			return err
		}
	}

	if err = tx.Commit().Error; err != nil {
		l.Errorf("Error while committing transaction : %v", err)
		return err
	}

	return err
}

// SetDayHighAndLow will be called when market
// is opening for the day
func SetDayHighAndLow() (err error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "SetDayHighAndLow",
	})

	db := getDB()
	tx := db.Begin()

	l.Info("Attempting to set previous day close")

	l.Info("Locking allStocks in SetDayHighAndLow")
	allStocks.Lock()
	defer func() {
		l.Info("Unlocking allStocks in SetDayHighAndLow")
		allStocks.Unlock()

		// Calling LoadStocks to prevent inconsistency
		// between db and allStocks
		err = LoadStocks()
	}()

	for _, stockNLock := range allStocks.m {
		stockNLock.stock.DayHigh = stockNLock.stock.CurrentPrice
		stockNLock.stock.DayLow = stockNLock.stock.CurrentPrice

		if err = tx.Save(stockNLock.stock).Error; err != nil {
			l.Errorf("Error occurred : %v", err)
			tx.Rollback()
			return err
		}
	}

	if err = tx.Commit().Error; err != nil {
		l.Errorf("Error while committing transaction : %v", err)
		return err
	}

	return err
}

func SetGivesDividends(stockId uint32, givesDividends bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":               "SetGivesDividends",
		"param_stockId":        stockId,
		"param_givesDividends": givesDividends,
	})

	l.Infof("Attempting to setGivesDividends")

	allStocks.Lock()
	stockNLock, ok := allStocks.m[stockId]
	if !ok {
		return InvalidStockError
	}
	defer allStocks.Unlock()

	stockNLock.Lock()
	defer stockNLock.Unlock()

	stock := stockNLock.stock
	oldStockCopy := *stock

	stock.GivesDividends = givesDividends

	db := getDB()

	if err := db.Save(stock).Error; err != nil {
		*stock = oldStockCopy
		return err
	}

	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: 0,
		Sd: &StockDividendState{
			StockID:       stockId,
			GivesDividend: givesDividends,
		},
		GsType: StockDividendStateUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	return nil
}

func SetBankruptcy(stockId uint32, isBankrupt bool) error {
	var l = logger.WithFields(logrus.Fields{
		"method":           "SetBankruptcy",
		"param_stockId":    stockId,
		"param_isBankrupt": isBankrupt,
	})

	l.Infof("Attempting to setBankruptcy")

	allStocks.Lock()
	stockNLock, ok := allStocks.m[stockId]
	if !ok {
		return InvalidStockError
	}
	defer allStocks.Unlock()

	stockNLock.Lock()
	defer stockNLock.Unlock()

	stock := stockNLock.stock
	oldStockCopy := *stock

	stock.IsBankrupt = isBankrupt
	stock.CurrentPrice = 0

	db := getDB()

	if err := db.Save(stock).Error; err != nil {
		*stock = oldStockCopy
		return err
	}

	stockPriceStream := datastreamsManager.GetStockPricesStream()
	stockPriceStream.SendStockPriceUpdate(stockId, stock.CurrentPrice)

	gameStateStream := datastreamsManager.GetGameStateStream()
	g := &GameState{
		UserID: 0,
		Sb: &StockBankruptState{
			StockId:    stockId,
			IsBankrupt: isBankrupt,
		},
		GsType: StockBankruptStateUpdate,
	}
	gameStateStream.SendGameStateUpdate(g.ToProto())

	SendPushNotification(0, PushNotification{
		Title:   "Message from Dalal Street! A company just went bankrupt.",
		Message: fmt.Sprintf("%v has declared bankruptcy !! Click here to know more.", stock.FullName),
		LogoUrl: fmt.Sprintf("%v/public/src/images/dalalfavicon.png", config.FrontEndUrl),
	})

	return nil
}

func IsStockBankrupt(stockId uint32) bool {
	allStocks.m[stockId].RLock()
	isBankrupt := allStocks.m[stockId].stock.IsBankrupt
	allStocks.m[stockId].RUnlock()
	return isBankrupt
}
