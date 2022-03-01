package models

import (
	"fmt"
	"sync"

	"github.com/delta/dalal-street-server/proto_build/models"
	models_pb "github.com/delta/dalal-street-server/proto_build/models"
	"github.com/delta/dalal-street-server/utils"
	"github.com/sirupsen/logrus"
)

type IpoStock struct {
	Id             uint32 `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	ShortName      string `gorm:"column:shortName;not null" json:"short_name"`
	FullName       string `gorm:"column:fullName;not null" json:"full_name"`
	Description    string `gorm:"not null" json:"description"`
	CreatedAt      string `gorm:"column:createdAt;not null" json:"created_at"`
	UpdatedAt      string `gorm:"column:updatedAt;not null" json:"updated_at"`
	IsBiddable     bool   `gorm:"column:isBiddable;not null" json:"is_biddable"`
	GivesDividends bool   `gorm:"column:givesDividends;not null" json:"gives_dividends"`
	SlotPrice      uint64 `gorm:"column:slotPrice;not null"  json:"slot_price"`
	StockPrice     uint64 `gorm:"column:stockPrice;not null"  json:"stock_price"`
	SlotQuantity   uint32 `gorm:"column:slotQuantity;not null"  json:"slot_quantity"`
	StocksPerSlot  uint32 `gorm:"column:stocksPerSlot;not null"  json:"stock_per_slot"`
}

func (IpoStock) TableName() string {
	return "IpoStocks"
}

func (gIpoStock *IpoStock) ToProto() *models_pb.IpoStock {
	return &models_pb.IpoStock{
		Id:             gIpoStock.Id,
		ShortName:      gIpoStock.ShortName,
		FullName:       gIpoStock.FullName,
		Description:    gIpoStock.Description,
		SlotPrice:      gIpoStock.SlotPrice,
		StockPrice:     gIpoStock.StockPrice,
		SlotQuantity:   gIpoStock.SlotQuantity,
		StocksPerSlot:  gIpoStock.StocksPerSlot,
		CreatedAt:      gIpoStock.CreatedAt,
		UpdatedAt:      gIpoStock.UpdatedAt,
		IsBiddable:     gIpoStock.IsBiddable,
		GivesDividends: gIpoStock.GivesDividends,
	}
}

// Refer models/stock.go
var allIpoStocks = struct {
	sync.RWMutex
	m map[uint32]*ipoStockAndLock
}{
	sync.RWMutex{},
	make(map[uint32]*ipoStockAndLock),
}

type ipoStockAndLock struct {
	sync.RWMutex
	ipostock *IpoStock
}

func GetAllIpoStocks() map[uint32]*IpoStock {

	var allIpoStocksCopy = make(map[uint32]*IpoStock)
	for ipoStockId, ipoStockNLock := range allIpoStocks.m {
		ipoStockNLock.RLock()
		allIpoStocksCopy[ipoStockId] = &IpoStock{}
		*allIpoStocksCopy[ipoStockId] = *ipoStockNLock.ipostock
		ipoStockNLock.RUnlock()
	}

	return allIpoStocksCopy
}

func AllowIpoBidding(IpoStockId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":           "OpenIpoBidding",
		"param_IpoStockId": IpoStockId,
	})

	l.Debugf("Attempting")

	db := getDB()

	IpoStock := &IpoStock{}
	if err := db.First(IpoStock, IpoStockId).Error; err != nil {
		l.Error(err)
		return err
	}

	if IpoStock == nil {
		return InvalidOrderIDError{}
	}

	if IpoStock.IsBiddable == true {
		return AlreadyClosedError{IpoStockId}
	}

	IpoStock.IsBiddable = true
	IpoStock.UpdatedAt = utils.GetCurrentTimeISO8601()

	if err := db.Save(IpoStock).Error; err != nil {
		l.Error(err)
		return err
	}

	return nil
}

func AllotSlots(IpoStockId uint32) error {
	var l = logger.WithFields(logrus.Fields{
		"method":           "AllotSlots",
		"param_IpoStockId": IpoStockId,
	})

	l.Debugf("Attempting")

	db := getDB()

	IpoStock := &IpoStock{}
	if err := db.First(IpoStock, IpoStockId).Error; err != nil {
		l.Error(err)
		return err
	}

	if IpoStock == nil {
		return InvalidOrderIDError{}
	}

	if IpoStock.IsBiddable == false {
		return AlreadyClosedError{IpoStockId}
	}

	IpoStock.IsBiddable = false
	IpoStock.UpdatedAt = utils.GetCurrentTimeISO8601()

	if err := db.Save(IpoStock).Error; err != nil {
		l.Error(err)
		return err
	}
	totalslots := IpoStock.SlotQuantity

	var openIpoBids []*IpoBid

	//Load open ipoBid orders from database
	if err := db.Where("IpoStockId = ? AND isClosed = ?", IpoStockId, 0).Find(&openIpoBids).Error; err != nil {
		l.Error(err)
	}

	if openIpoBids == nil {
		return AlreadyClosedError{}
	}

	l.Infof("Done")

	totalbids := len(openIpoBids)
	var subscriptionRatio = float64(totalbids) / float64(totalslots)
	var IpoStocksInMarket uint64
	var ListingPrice uint64

	newStock := &models.Stock{
		ShortName:        IpoStock.ShortName,
		FullName:         IpoStock.FullName,
		Description:      IpoStock.Description,
		CurrentPrice:     ListingPrice,
		DayHigh:          ListingPrice,
		DayLow:           ListingPrice,
		AllTimeHigh:      ListingPrice,
		AllTimeLow:       ListingPrice,
		StocksInExchange: 0,
		StocksInMarket:   IpoStocksInMarket,
		UpOrDown:         true,
		PreviousDayClose: ListingPrice,
		LastTradePrice:   ListingPrice,
		RealAvgPrice:     float64(ListingPrice),
		CreatedAt:        utils.GetCurrentTimeISO8601(),
		GivesDividends:   IpoStock.GivesDividends,
		IsBankrupt:       false,
	}
	newStock.UpdatedAt = newStock.CreatedAt

	if err := db.Create(newStock).Error; err != nil {
		l.Error(err)
		return err
	}

	cost := int64(uint64(IpoStock.SlotQuantity) * IpoStock.SlotPrice)
	// NewStockId = newStock.Id // Will this be defined or do i have to check db to get it?

	if subscriptionRatio <= 1.00 {
		for _, ipoBid := range openIpoBids {
			// allot 1 slot worth of stocks to userid
			AllotIpoTransaction := GetTransactionRef(ipoBid.UserId, newStock.Id, IpoAllotmentTransaction, 0, int64(IpoStock.SlotQuantity*IpoStock.StocksPerSlot), 0, -cost, 0)

			ipoBid.IsFulfilled = true
			ipoBid.IsClosed = true
			ipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()

			l.Infof("Saving AllotIpoTransaction, IpoStockId : %d, SlotQuantity : %d, UserId : %d, Cost: %d", ipoBid.IpoStockId, ipoBid.SlotQuantity, ipoBid.UserId, cost)

			if err := db.Save(AllotIpoTransaction).Error; err != nil {
				l.Error(err)
				return err
			}

			if err := db.Save(ipoBid).Error; err != nil {
				l.Error(err)
				return err
			}

			IpoStocksInMarket = uint64(len(openIpoBids))

		}
	} else {

		var AllotedIpoBids []*IpoBid

		// select 'totalslots' number of bids randomly
		if err := db.Raw("SELECT * FROM IpoBids WHERE ipoStockId = ? AND isClosed = ? ORDER BY RAND() LIMIT ?", IpoStockId, 0, totalslots).Scan(&AllotedIpoBids).Error; err != nil {
			l.Error(err)
			return err
		}

		for _, AllotedIpoBid := range AllotedIpoBids {

			//F allot 1 slot worth of stocks to userid
			AllotIpoTransaction := GetTransactionRef(AllotedIpoBid.UserId, newStock.Id, IpoAllotmentTransaction, 0, int64(IpoStock.SlotQuantity*IpoStock.StocksPerSlot), 0, -cost, 0)

			AllotedIpoBid.IsFulfilled = true
			AllotedIpoBid.IsClosed = true
			AllotedIpoBid.UpdatedAt = utils.GetCurrentTimeISO8601()

			l.Infof("Saving AllotIpoTransaction, IpoStockId : %d, SlotQuantity : %d, UserId : %d, Cost: %d", AllotedIpoBid.IpoStockId, AllotedIpoBid.SlotQuantity, AllotedIpoBid.UserId, cost)

			if err := db.Save(AllotIpoTransaction).Error; err != nil {
				l.Error(err)
				return err
			}

			if err := db.Save(AllotedIpoBid).Error; err != nil {
				l.Error(err)
				return err
			}
		}

		var UnfulIpoBids []*IpoBid

		//Load open unfulfilled ipoBid orders from database
		if err := db.Where("ipoStockId = ? AND isClosed = ? AND isFulfilled = ?", IpoStockId, 0, 0).Find(&UnfulIpoBids).Error; err != nil {
			l.Error(err)
			return err
		}

		for _, UnfulipoBid := range UnfulIpoBids {
			//  Refund slotprice amount to userid
			IpoRefundTransaction := GetTransactionRef(UnfulipoBid.UserId, newStock.Id, IpoAllotmentTransaction, 0, 0, 0, cost, -cost)

			UnfulipoBid.IsClosed = true
			UnfulipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()

			l.Infof("Saving IpoRefundTransaction, IpoStockId : %d, SlotQuantity : %d, UserId : %d, Cost: %d", UnfulipoBid.IpoStockId, UnfulipoBid.SlotQuantity, UnfulipoBid.UserId, cost)

			if err := db.Save(IpoRefundTransaction).Error; err != nil {
				l.Error(err)
				return err
			}

			if err := db.Save(UnfulipoBid).Error; err != nil {
				l.Error(err)
				return err
			}
		}

		IpoStocksInMarket = uint64(totalslots * IpoStock.StocksPerSlot)
	}

	if subscriptionRatio <= 1.05 && subscriptionRatio >= 0.95 {
		ListingPrice = IpoStock.StockPrice
	} else if subscriptionRatio <= 1.20 && subscriptionRatio >= 1.05 {
		ListingPrice = IpoStock.StockPrice * 105 / 100
	} else if subscriptionRatio <= 0.95 && subscriptionRatio >= 0.80 {
		ListingPrice = IpoStock.StockPrice * 95 / 100
	} else if subscriptionRatio >= 1.20 {
		ListingPrice = IpoStock.StockPrice * 110 / 100
	} else { // if subscriptionRatio <= 0.80
		ListingPrice = IpoStock.StockPrice * 90 / 100
	}

	// Update stock values
	newStock.CurrentPrice = ListingPrice
	newStock.DayHigh = ListingPrice
	newStock.DayLow = ListingPrice
	newStock.AllTimeHigh = ListingPrice
	newStock.AllTimeLow = ListingPrice
	newStock.StocksInMarket = IpoStocksInMarket
	newStock.PreviousDayClose = ListingPrice
	newStock.LastTradePrice = ListingPrice
	newStock.RealAvgPrice = float64(ListingPrice)
	newStock.UpdatedAt = utils.GetCurrentTimeISO8601()

	SendPushNotification(0, PushNotification{
		Title:    fmt.Sprintf("IPO Allotment has just happened for %v", IpoStock.FullName),
		Message:  fmt.Sprintf("IPO Allotment has just happened for  %v. Click here to know more.", IpoStock.FullName),
		LogoUrl:  "",
		ImageUrl: "",
	})

	LoadStocks() // does it overwrite the existing stock values (the regular ones, not IPO stocks)?
	// called when market day opens

	//  ToDo: Streams stuff - user portfolio is updated and regarding parts where stock price is updated(LoadStocks()??)

	return nil
}
