package models

import (
	"fmt"

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
		CreatedAt:      gIpoStock.CreatedAt,
		UpdatedAt:      gIpoStock.UpdatedAt,
		GivesDividends: gIpoStock.GivesDividends,
		SlotPrice:      gIpoStock.SlotPrice,
		StockPrice:     gIpoStock.StockPrice,
		SlotQuantity:   gIpoStock.SlotQuantity,
	}
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

	totalslots := IpoStock.SlotQuantity

	var openIpoBids []*IpoBid

	//Load open ipoBid orders from database
	if err := db.Where("IpoStockId = ? AND isClosed = ?", IpoStockId, 0).Find(&openIpoBids).Error; err != nil {
		l.Error(err)
	}

	l.Infof("Done")

	totalbids := len(openIpoBids)
	var subscriptionRatio = float64(totalbids) / float64(totalslots)
	var IpoStocksInMarket uint64

	if subscriptionRatio <= 1.00 {
		for _, ipoBid := range openIpoBids {
			// ToDo: allot 1 slot worth of stocks to userid  -- check out transactions table

			ipoBid.IsFulfilled = true
			ipoBid.IsClosed = true
			ipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()
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

			// ToDo: allot 1 slot worth of stocks to userid
			AllotedIpoBid.IsFulfilled = true
			AllotedIpoBid.IsClosed = true
			AllotedIpoBid.UpdatedAt = utils.GetCurrentTimeISO8601()

			if err := db.Save(AllotedIpoBid).Error; err != nil {
				l.Error(err)
				return err
			}
		}

		var UnfulIpoBids []*IpoBid

		//Load open unfulfilled ipoBid orders from database
		if err := db.Where("ipoStockId = ? AND isClosed = ? AND isFulfilled = ?", IpoStockId, 0, 0).Find(&UnfulIpoBids).Error; err != nil {
			panic("Error loading unfulfilled ipoBid orders in matching engine when alloting: " + err.Error())
		}
		for _, UnfulipoBid := range UnfulIpoBids {
			//  ToDo: Refund slotprice amount to userid
			UnfulipoBid.IsClosed = true
			UnfulipoBid.UpdatedAt = utils.GetCurrentTimeISO8601()
			if err := db.Save(UnfulipoBid).Error; err != nil {
				l.Error(err)
				return err
			}
		}

		IpoStocksInMarket = uint64(totalslots * IpoStock.StocksPerSlot)
	}

	var ListingPrice uint64

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

	SendPushNotification(0, PushNotification{
		Title:    fmt.Sprintf("IPO Allotment has just happened for %v", IpoStock.FullName),
		Message:  fmt.Sprintf("IPO Allotment has just happened for  %v. Click here to know more.", IpoStock.FullName),
		LogoUrl:  "",
		ImageUrl: "",
	})

	LoadStocks() // does it overwrite the existing stock values (the regular ones, not IPO stocks)?

	//  ToDo: Streams stuff - user portfolio is updated and regarding parts where stock price is updated(LoadStocks()??)

	return nil
}
